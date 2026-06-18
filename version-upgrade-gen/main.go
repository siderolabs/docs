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
)

func main() {
	workspace := flag.String("workspace", ".", "Path to the workspace root (repo root)")
	beta := flag.Bool("beta", false, "Beta release mode")
	flag.Parse()

	if err := os.Chdir(*workspace); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to workspace %s: %v\n", *workspace, err)
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")

	fmt.Fprintln(os.Stderr, "Reading current version from Makefile...")
	currentVersion, currentRelease, err := readCurrentVersionFromMakefile("Makefile")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading Makefile: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Current version: %s (%s)\n", currentVersion, currentRelease)

	newVersion := bumpMinorVersion(currentVersion)
	newReleaseBranch := "release-" + strings.TrimPrefix(newVersion, "v")
	fmt.Fprintf(os.Stderr, "  New version: %s\n", newVersion)

	varsFile := "public/snippets/custom-variables.mdx"

	// Ask the user interactively whether this is a beta or stable release.
	isBeta := *beta
	if !isBeta {
		isBeta = promptIsBeta(newVersion)
	}

	if isBeta {
		runBeta(token, newVersion, newReleaseBranch, varsFile)
	} else {
		runStable(token, currentVersion, currentRelease, newVersion, newReleaseBranch, varsFile)
	}

	// Write the new version to a temp file so the Makefile can read it after.
	if err := os.WriteFile(".upgrade-version-tmp", []byte(newVersion), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing temp version file: %v\n", err)
		os.Exit(1)
	}
}

// promptIsBeta interactively asks the user whether this is a beta or stable release.
func promptIsBeta(newVersion string) bool {
	betaExample := newVersion + ".0-beta.0"
	stableExample := newVersion + ".0"

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "What type of release is this?")
	fmt.Fprintf(os.Stderr, "  [1] Beta   - e.g %s\n", betaExample)
	fmt.Fprintf(os.Stderr, "  [2] Stable - e.g %s\n", stableExample)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprint(os.Stderr, "Enter your choice (1 or 2): ")

	var input string
	for {
		fmt.Fscan(os.Stdin, &input)
		input = strings.TrimSpace(input)
		switch input {
		case "1":
			fmt.Fprintln(os.Stderr, "")
			return true
		case "2":
			fmt.Fprintln(os.Stderr, "")
			return false
		default:
			fmt.Fprint(os.Stderr, "Please enter 1 or 2: ")
		}
	}
}

// runBeta handles beta release mode.
func runBeta(token, newVersion, newReleaseBranch, varsFile string) {
	fmt.Fprintln(os.Stderr, "Mode: beta")

	betaTag := newVersion + ".0-beta.0"
	fmt.Fprintf(os.Stderr, "  Beta tag: %s\n", betaTag)

	fmt.Fprintln(os.Stderr, "Updating versioned block in custom-variables.mdx...")
	if err := updateBetaVariables(varsFile, newVersion, betaTag); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating custom-variables.mdx: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Updating Makefile (adding new version at bottom)...")
	if err := updateMakefileBeta("Makefile", newVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating Makefile: %v\n", err)
		os.Exit(1)
	}
}

// runStable handles stable release mode.
func runStable(token, currentVersion, currentRelease, newVersion, newReleaseBranch, varsFile string) {
	fmt.Fprintln(os.Stderr, "Mode: stable")

	newRelease := newVersion + ".0"

	fmt.Fprintln(os.Stderr, "Fetching Talos release from GitHub...")
	talosRelease, err := fetchRelease("siderolabs/talos", newRelease, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching Talos release %s: %v\n", newRelease, err)
		os.Exit(1)
	}

	k8sVersion := parseField(talosRelease.Body, k8sRe)
	if k8sVersion == "" {
		fmt.Fprintf(os.Stderr, "Error: could not find Kubernetes version in Talos release body\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Kubernetes: %s\n", k8sVersion)

	fmt.Fprintln(os.Stderr, "Fetching Extensions release from GitHub...")
	extRelease, err := fetchRelease("siderolabs/extensions", newRelease, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching Extensions release %s: %v\n", newRelease, err)
		os.Exit(1)
	}

	nvCTK := parseField(extRelease.Body, nvCTKRe)
	if nvCTK == "" {
		fmt.Fprintf(os.Stderr, "Error: could not find nvidia-container-toolkit version in Extensions release body\n")
		os.Exit(1)
	}
	nvDriver := parseField(extRelease.Body, nvLTSRe)
	if nvDriver == "" {
		fmt.Fprintf(os.Stderr, "Error: could not find NVIDIA LTS driver version in Extensions release body\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  nvidia-container-toolkit: %s\n", nvCTK)
	fmt.Fprintf(os.Stderr, "  NVIDIA LTS driver: %s\n", nvDriver)

	fmt.Fprintln(os.Stderr, "Reading current k8s versions from custom-variables.mdx...")
	currentK8s, err := readExportVar(varsFile, "k8s_release")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading k8s_release: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "  Previous k8s: %s -> New k8s: %s\n", currentK8s, k8sVersion)

	info := VersionInfo{
		Version:             newVersion,
		Release:             newRelease,
		ReleaseBranch:       newReleaseBranch,
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

	fmt.Fprintln(os.Stderr, "Updating public/snippets/version-warning-banner.jsx...")
	if err := updateVersionBanner("public/snippets/version-warning-banner.jsx", currentVersion, newVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating version-warning-banner.jsx: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Updating canonical URLs across all Talos versions...")
	if err := updateCanonicalURLs("public/talos", currentVersion, newVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating canonical URLs: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Updating Makefile...")
	if err := updateMakefileStable("Makefile", currentVersion, currentRelease, newVersion, newRelease); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating Makefile: %v\n", err)
		os.Exit(1)
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

// readCurrentVersionFromMakefile reads TALOS_VERSION and TALOSCTL_IMAGE from the Makefile.
func readCurrentVersionFromMakefile(path string) (version, release string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	versionRe := regexp.MustCompile(`(?m)^TALOS_VERSION := (v[\d.]+)`)
	releaseRe := regexp.MustCompile(`(?m)^TALOSCTL_IMAGE := ghcr\.io/siderolabs/talosctl:(v[\d.]+)`)

	vm := versionRe.FindSubmatch(data)
	if vm == nil {
		return "", "", fmt.Errorf("TALOS_VERSION not found in Makefile")
	}
	rm := releaseRe.FindSubmatch(data)
	if rm == nil {
		return "", "", fmt.Errorf("TALOSCTL_IMAGE not found in Makefile")
	}

	return string(vm[1]), string(rm[1]), nil
}

// bumpMinorVersion increments the minor part of a version string like "v1.13" -> "v1.14".
func bumpMinorVersion(version string) string {
	trimmed := strings.TrimPrefix(version, "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) < 2 {
		return version
	}

	minor := 0
	fmt.Sscanf(parts[1], "%d", &minor)
	parts[1] = fmt.Sprintf("%d", minor+1)

	return "v" + strings.Join(parts, ".")
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

// updateBetaVariables only updates the release tag in the versioned block (or appends it).
func updateBetaVariables(path, newVersion, betaTag string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	varSuffix := "_v" + strings.ReplaceAll(strings.TrimPrefix(newVersion, "v"), ".", "_")

	if strings.Contains(content, "export const release"+varSuffix) {
		// Block exists — only update the release tag line.
		content = replaceExportLine(content, "release"+varSuffix, betaTag, '\'')
	} else {
		// Block doesn't exist yet — append a full new block with beta tag.
		newReleaseBranch := "release-" + strings.TrimPrefix(newVersion, "v")
		block := fmt.Sprintf(`
{/* %s talos release */}
export const release%s = '%s'
export const release_branch%s = '%s'
export const version%s = '%s'
export const nvidia_container_toolkit_release%s = ""
export const nvidia_driver_release%s = ""
`,
			strings.TrimPrefix(newVersion, "v"),
			varSuffix, betaTag,
			varSuffix, newReleaseBranch,
			varSuffix, newVersion,
			varSuffix,
			varSuffix,
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

// updateMakefileBeta adds the new yaml at the BOTTOM of the list (before omni.yaml) in all four targets.
func updateMakefileBeta(path, newVersion string) error {
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

// updateMakefileStable updates TALOSCTL_IMAGE, TALOS_VERSION, and moves/inserts the new yaml at the TOP.
func updateMakefileStable(path, oldVersion, oldRelease, newVersion, newRelease string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	newYAML := "talos-" + newVersion + ".yaml"
	oldYAML := "talos-" + oldVersion + ".yaml"

	// Update TALOSCTL_IMAGE.
	content = strings.ReplaceAll(
		content,
		"TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:"+oldRelease,
		"TALOSCTL_IMAGE := ghcr.io/siderolabs/talosctl:"+newRelease,
	)

	// Update TALOS_VERSION.
	content = strings.ReplaceAll(
		content,
		"TALOS_VERSION := "+oldVersion,
		"TALOS_VERSION := "+newVersion,
	)

	if strings.Contains(content, newYAML) {
		// Was added at bottom during beta — remove from bottom and insert at top.
		content = strings.ReplaceAll(content, "\t\t"+newYAML+" \\\n\t\tomni.yaml", "\t\tomni.yaml")
		content = strings.ReplaceAll(content, "\t\t../"+newYAML+" \\\n\t\t../omni.yaml", "\t\t../omni.yaml")
		content = strings.ReplaceAll(content, "\t\t"+oldYAML, "\t\t"+newYAML+" \\\n\t\t"+oldYAML)
		content = strings.ReplaceAll(content, "\t\t../"+oldYAML, "\t\t../"+newYAML+" \\\n\t\t../"+oldYAML)
	} else {
		// Not in list yet — insert at top.
		content = strings.ReplaceAll(content, "\t\t"+oldYAML, "\t\t"+newYAML+" \\\n\t\t"+oldYAML)
		content = strings.ReplaceAll(content, "\t\t../"+oldYAML, "\t\t../"+newYAML+" \\\n\t\t../"+oldYAML)
	}

	return os.WriteFile(path, []byte(content), 0o644)
}
