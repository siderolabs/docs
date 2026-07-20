package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Release is the subset of the GitHub releases API response we need.
type Release struct {
	TagName    string `json:"tag_name"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
}

// VersionInfo holds all the version data extracted for the new Talos release.
type VersionInfo struct {
	// e.g. "v1.14"
	Version string
	// e.g. "v1.14.0" or "v1.14.0-beta.0"
	Release string
	// e.g. "release-1.14"
	ReleaseBranch string
	// e.g. "1.37.0"
	K8sRelease string
	// e.g. "1.36.1" (the previous k8s_release value, becomes k8s_prev_release)
	K8sPrevRelease string
	// e.g. "v1.20.0"
	NvidiaCTKRelease string
	// e.g. "590.100.00"
	NvidiaDriverRelease string
}

var (
	k8sRe   = regexp.MustCompile(`(?m)^Kubernetes:\s*(\S+)`)
	nvCTKRe = regexp.MustCompile(`(?m)^nvidia-container-toolkit:\s*(\S+)`)
	nvLTSRe = regexp.MustCompile(`(?m)^NVIDIA LTS:\s*(\S+)`)

	// tagRe is the accepted target tag shape: vMAJOR.MINOR.PATCH with an optional
	// -alpha.N / -beta.N / -rc.N pre-release suffix.
	tagRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?$`)
)

func main() {
	workspace := flag.String("workspace", ".", "Path to the workspace root (repo root)")
	tag := flag.String("tag", "", "Target talosctl image tag, e.g. v1.14.0-beta.0 or v1.14.0")
	flag.Parse()

	*tag = strings.TrimSpace(*tag)
	if *tag == "" {
		fmt.Fprintln(os.Stderr, "Error: --tag is required (e.g. --tag v1.14.0-beta.0)")
		os.Exit(1)
	}
	if !tagRe.MatchString(*tag) {
		fmt.Fprintf(os.Stderr, "Error: invalid tag %q\n", *tag)
		fmt.Fprintln(os.Stderr, "Expected vMAJOR.MINOR.PATCH with an optional -alpha.N / -beta.N / -rc.N suffix.")
		fmt.Fprintln(os.Stderr, "Examples: v1.14.0, v1.14.0-alpha.0, v1.14.0-beta.2, v1.14.0-rc.1")
		os.Exit(1)
	}

	if err := os.Chdir(*workspace); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to workspace %s: %v\n", *workspace, err)
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")

	varsFile := "public/snippets/custom-variables.mdx"
	bannerFile := "public/snippets/version-warning-banner.jsx"

	fmt.Fprintln(os.Stderr, "Reading current version from Makefile...")
	currentTag, err := readCurrentImageTag("Makefile")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Makefile: %v\n", err)
		os.Exit(1)
	}

	currentMinor, err := minorOf(currentTag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing current tag %q: %v\n", currentTag, err)
		os.Exit(1)
	}
	targetMinor, err := minorOf(*tag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing target tag %q: %v\n", *tag, err)
		os.Exit(1)
	}

	stable := isStable(*tag)
	sameMinor := targetMinor == currentMinor

	stage := "stable"
	if !stable {
		stage = strings.TrimPrefix(prereleaseStage(*tag), "-")
		if stage == "" {
			stage = "prerelease"
		}
	}
	scope := "new minor"
	if sameMinor {
		scope = "same minor"
	}

	fmt.Fprintf(os.Stderr, "  Current image: %s  (folder %s)\n", currentTag, currentMinor)
	fmt.Fprintf(os.Stderr, "  Target:        %s  ->  stage: %s, folder: %s (%s)\n", *tag, stage, targetMinor, scope)

	if stable {
		runStable(token, currentTag, *tag, currentMinor, targetMinor, varsFile, bannerFile)
	} else {
		runPrerelease(token, currentTag, *tag, currentMinor, targetMinor, varsFile)
	}

	// Write the new folder version to a temp file so the Makefile can read it after.
	if err := os.WriteFile(".upgrade-version-tmp", []byte(targetMinor), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temp version file: %v\n", err)
		os.Exit(1)
	}
}

// runPrerelease handles alpha/beta (and any pre-release) targets. It advances the
// image pin and the versioned variables block, but never touches the "latest
// stable" pointer (banner, canonical URLs, latest-stable block).
func runPrerelease(token, currentTag, tag, currentMinor, targetMinor, varsFile string) {
	fmt.Fprintln(os.Stderr, "Updating versioned block in custom-variables.mdx...")
	if err := updatePrereleaseVariables(varsFile, targetMinor, tag, token); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating custom-variables.mdx: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Updating Makefile pins (TALOSCTL_IMAGE -> %s, TALOS_VERSION -> %s)...\n", tag, targetMinor)
	if err := setMakefilePin("Makefile", currentTag, tag, currentMinor, targetMinor); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating Makefile pins: %v\n", err)
		os.Exit(1)
	}

	// Alpha builds are generated but kept out of the published navigation. The
	// version only enters the docs.json nav (at the bottom of the list) once it
	// reaches beta. addNavBottom is idempotent, so calling it for every beta
	// correctly adds the entry an earlier alpha deliberately skipped.
	if prereleaseStage(tag) == "-alpha" {
		fmt.Fprintln(os.Stderr, "Alpha release: leaving docs.json navigation unchanged.")
		return
	}

	fmt.Fprintln(os.Stderr, "Ensuring version is in navigation (bottom of list)...")
	if err := addNavBottom("Makefile", targetMinor); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating Makefile navigation: %v\n", err)
		os.Exit(1)
	}
}

// runStable handles stable targets. It advances the image pin, refreshes the
// k8s/nvidia values from the release notes, and — when the latest stable minor
// actually changes — promotes the version in the banner, canonical URLs and nav.
func runStable(token, currentTag, tag, currentMinor, targetMinor string, varsFile, bannerFile string) {
	release := tag
	releaseBranch := "release-" + strings.TrimPrefix(targetMinor, "v")

	fmt.Fprintln(os.Stderr, "Fetching Talos release from GitHub...")
	talosRelease, err := fetchRelease("siderolabs/talos", release, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching Talos release %s: %v\n", release, err)
		os.Exit(1)
	}

	k8sVersion := parseField(talosRelease.Body, k8sRe)
	if k8sVersion == "" {
		fmt.Fprintf(os.Stderr, "Error: could not find Kubernetes version in Talos release body\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Kubernetes: %s\n", k8sVersion)

	// nvidia values: try the extensions release for this tag, and if they are not
	// published there yet, fall back to the unversioned latest-stable values so a
	// stable upgrade degrades gracefully instead of aborting.
	fmt.Fprintln(os.Stderr, "Resolving nvidia versions...")
	varsData, err := os.ReadFile(varsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", varsFile, err)
		os.Exit(1)
	}
	nvCTK, nvDriver := resolveNvidia(release, token, string(varsData))

	fmt.Fprintln(os.Stderr, "Reading current k8s versions from custom-variables.mdx...")
	currentK8s, err := readExportVar(varsFile, "k8s_release")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading k8s_release: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Previous k8s: %s -> New k8s: %s\n", currentK8s, k8sVersion)

	info := VersionInfo{
		Version:             targetMinor,
		Release:             release,
		ReleaseBranch:       releaseBranch,
		K8sRelease:          k8sVersion,
		K8sPrevRelease:      currentK8s,
		NvidiaCTKRelease:    nvCTK,
		NvidiaDriverRelease: nvDriver,
	}

	fmt.Fprintln(os.Stderr, "Updating public/snippets/custom-variables.mdx...")
	if err := updateCustomVariables(varsFile, info); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating custom-variables.mdx: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Updating Makefile pins (TALOSCTL_IMAGE -> %s, TALOS_VERSION -> %s)...\n", tag, targetMinor)
	if err := setMakefilePin("Makefile", currentTag, tag, currentMinor, targetMinor); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating Makefile pins: %v\n", err)
		os.Exit(1)
	}

	// The "latest stable" pointer is tracked by the banner, not by TALOS_VERSION
	// (which may already sit on this minor from an earlier alpha/beta). Only move
	// it when the newly stable minor differs from the current latest stable.
	prevStable, err := readBannerLatest(bannerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current latest version from banner: %v\n", err)
		os.Exit(1)
	}

	if prevStable != targetMinor {
		fmt.Fprintf(os.Stderr, "Promoting latest stable %s -> %s...\n", prevStable, targetMinor)

		fmt.Fprintln(os.Stderr, "Updating public/snippets/version-warning-banner.jsx...")
		if err := updateVersionBanner(bannerFile, prevStable, targetMinor); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating version-warning-banner.jsx: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "Updating canonical URLs across all Talos versions...")
		if err := updateCanonicalURLs("public/talos", prevStable, targetMinor); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating canonical URLs: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(os.Stderr, "Moving new version to the top of the navigation...")
		if err := moveNavTop("Makefile", targetMinor, prevStable); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating Makefile navigation: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintf(os.Stderr, "  %s is already the latest stable; banner and nav unchanged.\n", targetMinor)
	}
}

// fetchRelease fetches a specific release from the GitHub API by tag.
func fetchRelease(repo, tag, token string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "version-upgrade-gen")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d for %s@%s: %s", resp.StatusCode, repo, tag, body)
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	return &release, nil
}

// parseField extracts the first capture group from a regex match against text.
func parseField(text string, re *regexp.Regexp) string {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// readCurrentImageTag reads the full talosctl image tag (including any
// pre-release suffix) from TALOSCTL_IMAGE in the Makefile.
func readCurrentImageTag(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`(?m)^TALOSCTL_IMAGE := ghcr\.io/siderolabs/talosctl:(\S+)`)
	m := re.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("TALOSCTL_IMAGE not found in Makefile")
	}

	return string(m[1]), nil
}

// minorOf returns the "vMAJOR.MINOR" folder label for a tag like
// "v1.14.0-beta.0" -> "v1.14".
func minorOf(tag string) (string, error) {
	t := strings.TrimPrefix(tag, "v")
	if i := strings.IndexByte(t, '-'); i >= 0 {
		t = t[:i]
	}
	parts := strings.Split(t, ".")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot parse major.minor from %q", tag)
	}
	return "v" + parts[0] + "." + parts[1], nil
}

// prereleaseStage returns the pre-release suffix marker of a tag, e.g.
// "-beta" for "v1.14.0-beta.0", or "" for a stable tag.
func prereleaseStage(tag string) string {
	switch {
	case strings.Contains(tag, "-alpha"):
		return "-alpha"
	case strings.Contains(tag, "-beta"):
		return "-beta"
	case strings.Contains(tag, "-"):
		return "-prerelease"
	default:
		return ""
	}
}

// isStable reports whether a tag is a final stable release (no pre-release suffix).
func isStable(tag string) bool {
	return !strings.Contains(tag, "-")
}

// readBannerLatest reads the current latestVersion constant from the banner JSX file.
func readBannerLatest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`const latestVersion = "(v[\d.]+)"`)
	m := re.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("latestVersion not found in %s", path)
	}

	return string(m[1]), nil
}

// readExportVar reads a single `export const <name>` value from an MDX file.
func readExportVar(path, name string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`(?m)^export const ` + name + ` = ['"]([^'"]+)['"]`)
	m := re.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("export const %s not found in %s", name, path)
	}

	return string(m[1]), nil
}

// exportValue returns the value of an `export const <name> = '...'` line from a
// content string, or "" if the line is absent or its value is empty.
func exportValue(content, name string) string {
	re := regexp.MustCompile(`(?m)^export const ` + name + ` = ['"]([^'"]*)['"]`)
	m := re.FindStringSubmatch(content)
	if m == nil {
		return ""
	}
	return m[1]
}

// resolveNvidia determines the nvidia-container-toolkit and NVIDIA driver values
// for a tag. It first tries the siderolabs/extensions release notes for that exact
// tag; if those are unavailable (a pre-release with no extensions release, a fetch
// error, or missing fields) it falls back to the unversioned latest-stable values
// already present in the variables content. It never fails.
func resolveNvidia(tag, token, content string) (nvCTK, nvDriver string) {
	if ext, err := fetchRelease("siderolabs/extensions", tag, token); err == nil {
		ctk := parseField(ext.Body, nvCTKRe)
		drv := parseField(ext.Body, nvLTSRe)
		if ctk != "" && drv != "" {
			fmt.Fprintf(os.Stderr, "  nvidia from extensions %s: container-toolkit=%s driver=%s\n", tag, ctk, drv)
			return ctk, drv
		}
		fmt.Fprintf(os.Stderr, "  extensions %s missing nvidia fields; using latest-stable values\n", tag)
	} else {
		fmt.Fprintf(os.Stderr, "  extensions release %s unavailable (%v); using latest-stable values\n", tag, err)
	}

	nvCTK = exportValue(content, "nvidia_container_toolkit_release")
	nvDriver = exportValue(content, "nvidia_driver_release")
	fmt.Fprintf(os.Stderr, "  nvidia fallback (latest stable): container-toolkit=%s driver=%s\n", nvCTK, nvDriver)
	return nvCTK, nvDriver
}

// updatePrereleaseVariables updates (or appends) the versioned block for a
// pre-release tag. It always advances the release tag line. nvidia values are
// resolved via resolveNvidia (extensions release notes, else the unversioned
// latest-stable values), so a new block is never written empty and an existing
// block with empty nvidia values is backfilled. Existing non-empty nvidia values
// are left untouched.
func updatePrereleaseVariables(path, newVersion, prereleaseTag, token string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	varSuffix := "_v" + strings.ReplaceAll(strings.TrimPrefix(newVersion, "v"), ".", "_")

	if strings.Contains(content, "export const release"+varSuffix) {
		// Block exists — always advance the release tag line.
		content = replaceExportLine(content, "release"+varSuffix, prereleaseTag, '\'')

		// Backfill nvidia only when currently empty, so good values from an
		// earlier stable run are never clobbered.
		ctkEmpty := exportValue(content, "nvidia_container_toolkit_release"+varSuffix) == ""
		drvEmpty := exportValue(content, "nvidia_driver_release"+varSuffix) == ""
		if ctkEmpty || drvEmpty {
			nvCTK, nvDriver := resolveNvidia(prereleaseTag, token, content)
			if ctkEmpty {
				content = replaceExportLine(content, "nvidia_container_toolkit_release"+varSuffix, nvCTK, '"')
			}
			if drvEmpty {
				content = replaceExportLine(content, "nvidia_driver_release"+varSuffix, nvDriver, '"')
			}
		}
	} else {
		// Block doesn't exist yet — append a full new block, seeding nvidia values.
		nvCTK, nvDriver := resolveNvidia(prereleaseTag, token, content)
		newReleaseBranch := "release-" + strings.TrimPrefix(newVersion, "v")
		block := fmt.Sprintf(`
{/* %s talos release */}
export const release%s = '%s'
export const release_branch%s = '%s'
export const version%s = '%s'
export const nvidia_container_toolkit_release%s = "%s"
export const nvidia_driver_release%s = "%s"
`,
			strings.TrimPrefix(newVersion, "v"),
			varSuffix, prereleaseTag,
			varSuffix, newReleaseBranch,
			varSuffix, newVersion,
			varSuffix, nvCTK,
			varSuffix, nvDriver,
		)
		content = strings.TrimRight(content, "\n") + "\n" + block
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// updateCustomVariables replaces the entire latest stable block and updates/appends the versioned block.
func updateCustomVariables(path string, info VersionInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	// Replace the k8s lines (unversioned).
	content = replaceExportLine(content, "k8s_prev_release", info.K8sPrevRelease, '\'')
	content = replaceExportLine(content, "k8s_release", info.K8sRelease, '\'')

	// Replace the entire "latest stable Talos release version" block.
	latestBlock := fmt.Sprintf(`{/* latest stable Talos release version */}
export const release = '%s'
export const release_branch = '%s'
export const version = '%s'
export const nvidia_container_toolkit_release = '%s'
export const nvidia_driver_release = '%s'`,
		info.Release,
		info.ReleaseBranch,
		info.Version,
		info.NvidiaCTKRelease,
		info.NvidiaDriverRelease,
	)

	latestRe := regexp.MustCompile(`(?s)\{/\* latest stable Talos release version \*/\}[^\n]*\n(?:export const [^\n]+\n){1,10}`)
	if latestRe.MatchString(content) {
		content = latestRe.ReplaceAllString(content, latestBlock+"\n")
	} else {
		return fmt.Errorf("could not find '{/* latest stable Talos release version */}' block in %s", path)
	}

	// Update or append the versioned snapshot block.
	varSuffix := "_v" + strings.ReplaceAll(strings.TrimPrefix(info.Version, "v"), ".", "_")
	if strings.Contains(content, "export const release"+varSuffix) {
		// Block already exists (e.g. added during alpha/beta) — update all lines.
		content = replaceExportLine(content, "release"+varSuffix, info.Release, '\'')
		content = replaceExportLine(content, "release_branch"+varSuffix, info.ReleaseBranch, '\'')
		content = replaceExportLine(content, "version"+varSuffix, info.Version, '\'')
		content = replaceExportLine(content, "nvidia_container_toolkit_release"+varSuffix, info.NvidiaCTKRelease, '"')
		content = replaceExportLine(content, "nvidia_driver_release"+varSuffix, info.NvidiaDriverRelease, '"')
	} else {
		// Block doesn't exist yet — append it.
		block := fmt.Sprintf(`
{/* %s talos release */}
export const release%s = '%s'
export const release_branch%s = '%s'
export const version%s = '%s'
export const nvidia_container_toolkit_release%s = "%s"
export const nvidia_driver_release%s = "%s"
`,
			strings.TrimPrefix(info.Version, "v"),
			varSuffix, info.Release,
			varSuffix, info.ReleaseBranch,
			varSuffix, info.Version,
			varSuffix, info.NvidiaCTKRelease,
			varSuffix, info.NvidiaDriverRelease,
		)
		content = strings.TrimRight(content, "\n") + "\n" + block
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// replaceExportLine replaces a single `export const <name> = '...'` line.
func replaceExportLine(content, name, value string, quote byte) string {
	re := regexp.MustCompile(`(?m)^export const ` + name + ` = ['"][^'"]*['"]`)
	replacement := fmt.Sprintf("export const %s = %c%s%c", name, quote, value, quote)
	return re.ReplaceAllString(content, replacement)
}

// updateVersionBanner replaces the latestVersion constant in the banner JSX file.
func updateVersionBanner(path, oldVersion, newVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	old := fmt.Sprintf(`const latestVersion = "%s"`, oldVersion)
	updated := strings.ReplaceAll(string(data), old, fmt.Sprintf(`const latestVersion = "%s"`, newVersion))
	if updated == string(data) {
		return fmt.Errorf("latestVersion %q not found in %s", oldVersion, path)
	}

	return os.WriteFile(path, []byte(updated), 0o644)
}

// updateCanonicalURLs walks all MDX files and updates canonical URLs from old to new version.
func updateCanonicalURLs(dir, oldVersion, newVersion string) error {
	oldCanonical := "https://docs.siderolabs.com/talos/" + oldVersion + "/"
	newCanonical := "https://docs.siderolabs.com/talos/" + newVersion + "/"

	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".mdx") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		if !strings.Contains(content, oldCanonical) {
			return nil
		}

		return os.WriteFile(path, []byte(strings.ReplaceAll(content, oldCanonical, newCanonical)), 0o644)
	})
}

// setMakefilePin rewrites TALOSCTL_IMAGE to the exact new tag and TALOS_VERSION to
// the new folder minor. It matches the full current tag (suffix included) so a
// pre-release pin like v1.14.0-alpha.2 is replaced cleanly rather than by prefix.
func setMakefilePin(path, currentTag, newTag, currentMinor, newMinor string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	oldImageLine := "TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:" + currentTag
	newImageLine := "TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:" + newTag
	if !strings.Contains(content, oldImageLine) {
		return fmt.Errorf("could not find %q in Makefile", oldImageLine)
	}
	content = strings.ReplaceAll(content, oldImageLine, newImageLine)

	// TALOS_VERSION only changes when the folder minor changes.
	if currentMinor != newMinor {
		content = strings.ReplaceAll(content, "TALOS_VERSION := "+currentMinor, "TALOS_VERSION := "+newMinor)
	}

	return os.WriteFile(path, []byte(content), 0o644)
}

// addNavBottom adds the new yaml at the BOTTOM of the list (before omni.yaml) in all four targets.
func addNavBottom(path, newVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	newYAML := "talos-" + newVersion + ".yaml"

	if strings.Contains(content, newYAML) {
		fmt.Fprintf(os.Stderr, "  %s already in Makefile, skipping\n", newYAML)
		return nil
	}

	// Insert before omni.yaml in both containerised and local forms.
	content = strings.ReplaceAll(content, "\t\tomni.yaml", "\t\t"+newYAML+" \\\n\t\tomni.yaml")
	content = strings.ReplaceAll(content, "\t\t../omni.yaml", "\t\t../"+newYAML+" \\\n\t\t../omni.yaml")

	return os.WriteFile(path, []byte(content), 0o644)
}

// moveNavTop moves (or inserts) the new yaml at the TOP of the list, before the
// previous top entry (prevVersion), in all four targets.
func moveNavTop(path, newVersion, prevVersion string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	newYAML := "talos-" + newVersion + ".yaml"
	prevYAML := "talos-" + prevVersion + ".yaml"

	if strings.Contains(content, newYAML) {
		// Was added at bottom during a pre-release — remove from bottom first.
		content = strings.ReplaceAll(content, "\t\t"+newYAML+" \\\n\t\tomni.yaml", "\t\tomni.yaml")
		content = strings.ReplaceAll(content, "\t\t../"+newYAML+" \\\n\t\t../omni.yaml", "\t\t../omni.yaml")
	}

	// Guard against duplicate insertion if it is already at the top.
	if !strings.Contains(content, "\t\t"+newYAML) {
		content = strings.ReplaceAll(content, "\t\t"+prevYAML, "\t\t"+newYAML+" \\\n\t\t"+prevYAML)
		content = strings.ReplaceAll(content, "\t\t../"+prevYAML, "\t\t../"+newYAML+" \\\n\t\t../"+prevYAML)
	}

	return os.WriteFile(path, []byte(content), 0o644)
}
