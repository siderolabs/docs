package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// page builds a minimal .mdx file with frontmatter and the given body lines.
// Body lines must be longer than three characters to survive body(), which is
// what the real pages look like.
func page(body ...string) string {
	return "---\ntitle: \"A page\"\n---\n\n" + strings.Join(body, "\n") + "\n"
}

// titled builds a page with an explicit title.
func titled(title string, body ...string) string {
	return "---\ntitle: \"" + title + "\"\n---\n\n" + strings.Join(body, "\n") + "\n"
}

// writeTree materialises a public/talos tree from "version/page path" keys and
// returns the path to the public/talos directory.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "public", "talos")
	for p, content := range files {
		full := filepath.Join(root, filepath.FromSlash(p)+".mdx")
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func resolveIn(t *testing.T, root, relPath, own, cur string) target {
	t.Helper()
	ix, err := newIndex(root)
	if err != nil {
		t.Fatal(err)
	}
	return ix.resolve(relPath, own, cur, map[string]bool{})
}

func wantTarget(t *testing.T, got target, version, relPath string) {
	t.Helper()
	if got.version != version || got.relPath != relPath {
		t.Errorf("got %s/%s, want %s/%s", got.version, got.relPath, version, relPath)
	}
}

func TestReadVersion(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "vars.mdx")
	body := "export const release = 'v1.14.0'\n" +
		"export const version = 'v1.14'\n" +
		"export const version_v1_6 = 'v1.6'\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readVersion(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.14" {
		t.Errorf("got %q, want %q (must not match version_v1_6)", got, "v1.14")
	}
}

func TestReadVersionMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "vars.mdx")
	if err := os.WriteFile(p, []byte("export const release = 'v1.14.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readVersion(p); err == nil {
		t.Fatal("expected an error when `export const version` is absent")
	}
}

func TestParseTalosPage(t *testing.T) {
	for _, tc := range []struct {
		in                       string
		ok                       bool
		prefix, version, relPath string
	}{
		{"public/talos/v1.14/security/rbac.mdx", true, "", "v1.14", "security/rbac"},
		{"../../public/talos/v1.9/networking/vip.mdx", true, "../../", "v1.9", "networking/vip"},
		{"/abs/public/talos/v1.6/a/b/c.mdx", true, "/abs/", "v1.6", "a/b/c"},
		{"public/omni/overview.mdx", false, "", "", ""},
		{"public/talos/v1.14/README.md", false, "", "", ""},
	} {
		got, ok := parseTalosPage(tc.in)
		if ok != tc.ok {
			t.Fatalf("%s: ok=%v, want %v", tc.in, ok, tc.ok)
		}
		if !ok {
			continue
		}
		if got.prefix != tc.prefix || got.version != tc.version || got.relPath != tc.relPath {
			t.Errorf("%s: got %+v", tc.in, got)
		}
	}
}

func TestCanonicalFor(t *testing.T) {
	want := "https://docs.siderolabs.com/talos/v1.14/networking/advanced/vip"
	if got := canonicalFor("v1.14", "networking/advanced/vip"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLessVersionOrdersNumerically(t *testing.T) {
	if !lessVersion("v1.9", "v1.10") {
		t.Error("v1.9 must sort before v1.10, not lexically after it")
	}
	if lessVersion("v1.14", "v1.13") {
		t.Error("v1.14 must not sort before v1.13")
	}
}

// A page in the current version is its own canonical.
func TestResolveCurrentVersionPointsAtItself(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/security/rbac": page("Older copy."),
		"v1.14/security/rbac": page("Current copy."),
	})
	wantTarget(t, resolveIn(t, root, "security/rbac", "v1.13", "v1.14"), "v1.14", "security/rbac")
}

// The common case: the same path still exists in the current version.
func TestResolveSamePathMapsForward(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/security/rbac": page("Older copy."),
		"v1.14/security/rbac": page("Current copy."),
	})
	wantTarget(t, resolveIn(t, root, "security/rbac", "v1.13", "v1.14"), "v1.14", "security/rbac")
}

// Rule (a): the file name survived, the folder changed.
func TestResolveRuleSameFileName(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/networking/vip":          page("VIP docs."),
		"v1.13/networking/kubespan":     page("Kubespan docs."),
		"v1.14/networking/advanced/vip": page("VIP docs, rewritten entirely."),
		"v1.14/networking/kubespan":     page("Kubespan docs."),
	})
	wantTarget(t, resolveIn(t, root, "networking/vip", "v1.13", "v1.14"), "v1.14", "networking/advanced/vip")
}

// Rule (b): the page became a directory holding exactly one page.
func TestResolveRuleBecameDirectoryWithOnePage(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/storage/disks":          page("Disk docs."),
		"v1.14/storage/disks/overview": page("Disk docs, reorganised."),
	})
	wantTarget(t, resolveIn(t, root, "storage/disks", "v1.13", "v1.14"), "v1.14", "storage/disks/overview")
}

// Rule (c): file-name word overlap, when content was rewritten so similarity
// cannot help. wireguard-network -> wireguard.
func TestResolveRuleFileNameWordOverlap(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/networking/wireguard-network": page("Old wireguard prose, totally different."),
		"v1.14/networking/advanced/wireguard": page("Brand new wireguard prose.",
			"Nothing in common with the old page at all."),
		"v1.14/networking/configuration/dynamic": page("Unrelated dynamic addressing page."),
	})
	wantTarget(t, resolveIn(t, root, "networking/wireguard-network", "v1.13", "v1.14"),
		"v1.14", "networking/advanced/wireguard")
}

// Rule (d): renamed with the body preserved, and a file name too different for
// rule (c) to fire. verifying-images -> source-talos-images.
func TestResolveRuleContentSimilarity(t *testing.T) {
	shared := []string{
		"Sidero Labs signs the container images with cosign.",
		"The installer image is signed.",
		"The talosctl image is signed.",
		"The imager image is signed.",
	}
	root := writeTree(t, map[string]string{
		"v1.13/security/verifying-images":    page(shared...),
		"v1.14/security/source-talos-images": page(shared...),
		"v1.14/security/selinux":             page("Something else entirely, about SELinux."),
	})
	wantTarget(t, resolveIn(t, root, "security/verifying-images", "v1.13", "v1.14"),
		"v1.14", "security/source-talos-images")
}

// A split has no single successor, so the newest surviving copy wins.
func TestResolveSplitFallsBackToNewestSurvivingCopy(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.12/networking/advanced-networking": page("Bonding, bridges and VLANs."),
		"v1.13/networking/advanced-networking": page("Bonding, bridges and VLANs."),
		"v1.14/networking/logical/bond":        page("Bonding only."),
		"v1.14/networking/logical/bridge":      page("Bridges only."),
		"v1.14/networking/logical/vlan":        page("VLANs only."),
	})
	// Both older copies consolidate onto the newest one that still exists.
	wantTarget(t, resolveIn(t, root, "networking/advanced-networking", "v1.12", "v1.14"),
		"v1.13", "networking/advanced-networking")
}

// A page that became a directory of several pages is a split, not a move.
func TestResolveDirectoryWithSeveralPagesIsASplit(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/storage/disks":          page("Disk docs."),
		"v1.14/storage/disks/overview": page("Overview."),
		"v1.14/storage/disks/user":     page("User volumes."),
		"v1.14/storage/disks/system":   page("System volumes."),
	})
	wantTarget(t, resolveIn(t, root, "storage/disks", "v1.13", "v1.14"), "v1.13", "storage/disks")
}

// A genuinely deleted page is the authority for its own content.
func TestResolveDroppedPagePointsAtNewestSurvivingCopy(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.6/platforms/digital-rebar": page("Digital Rebar instructions."),
		"v1.7/platforms/digital-rebar": page("Digital Rebar instructions."),
		"v1.8/platforms/matchbox":      page("Matchbox instructions, unrelated."),
		"v1.14/platforms/matchbox":     page("Matchbox instructions, unrelated."),
	})
	// v1.6 defers to v1.7; v1.7 is the last copy, so it points at itself.
	wantTarget(t, resolveIn(t, root, "platforms/digital-rebar", "v1.6", "v1.14"), "v1.7", "platforms/digital-rebar")
}

// Two equally good candidates must not be guessed between.
func TestResolveAmbiguousCandidatesFallBack(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/networking/vip":          page("Old VIP prose."),
		"v1.14/networking/advanced/vip": page("One new page about something."),
		"v1.14/networking/logical/vip":  page("Another new page about something else."),
	})
	wantTarget(t, resolveIn(t, root, "networking/vip", "v1.13", "v1.14"), "v1.13", "networking/vip")
}

// A page that moves in one release and again in the next resolves through the
// chain to its final home.
func TestResolveFollowsSuccessorChain(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.12/a/thing":     page("Thing docs."),
		"v1.12/stable/page": page("Something stable."),
		"v1.13/b/thing":     page("Thing docs, moved once."),
		"v1.13/stable/page": page("Something stable."),
		"v1.14/c/thing":     page("Thing docs, moved twice."),
		"v1.14/stable/page": page("Something stable."),
	})
	wantTarget(t, resolveIn(t, root, "a/thing", "v1.12", "v1.14"), "v1.14", "c/thing")
}

func TestProcessInsertsMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mdx")
	os.WriteFile(p, []byte("---\ntitle: \"A\"\n---\n\nBody.\n"), 0o644)

	changed, err := process(p, "https://x/talos/v1.14/a", false)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got, _ := os.ReadFile(p)
	want := "---\ntitle: \"A\"\ncanonical: https://x/talos/v1.14/a\n---\n\nBody.\n"
	if string(got) != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestProcessRewritesExistingInPlace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mdx")
	os.WriteFile(p, []byte("---\ntitle: \"A\"\ncanonical: https://x/talos/v1.13/a\naliases:\n  - /old\n---\n\nBody.\n"), 0o644)

	changed, err := process(p, "https://x/talos/v1.14/a", false)
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	got, _ := os.ReadFile(p)
	want := "---\ntitle: \"A\"\ncanonical: https://x/talos/v1.14/a\naliases:\n  - /old\n---\n\nBody.\n"
	if string(got) != want {
		t.Errorf("field order and other frontmatter must be preserved; got:\n%q", got)
	}
}

func TestProcessNoopWhenCorrect(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mdx")
	body := "---\ntitle: \"A\"\ncanonical: https://x/talos/v1.14/a\n---\n\nBody.\n"
	os.WriteFile(p, []byte(body), 0o644)

	changed, err := process(p, "https://x/talos/v1.14/a", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("an already-correct file must not be reported as changed")
	}
	got, _ := os.ReadFile(p)
	if string(got) != body {
		t.Error("file must be untouched")
	}
}

func TestProcessCheckDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mdx")
	body := "---\ntitle: \"A\"\n---\n\nBody.\n"
	os.WriteFile(p, []byte(body), 0o644)

	changed, err := process(p, "https://x/talos/v1.14/a", true)
	if err != nil || !changed {
		t.Fatalf("check mode must still report the needed change; changed=%v err=%v", changed, err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != body {
		t.Error("check mode must not write")
	}
}

func TestProcessSkipsFileWithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.mdx")
	body := "No frontmatter here.\n"
	os.WriteFile(p, []byte(body), 0o644)

	changed, err := process(p, "https://x/talos/v1.14/a", false)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("a file without frontmatter must be skipped, not rewritten")
	}
	got, _ := os.ReadFile(p)
	if string(got) != body {
		t.Error("file must be untouched")
	}
}

// The title parser has to cope with every shape the docs use: the tree has
// 1138 quoted titles and 435 unquoted ones.
func TestTitleReFormats(t *testing.T) {
	for in, want := range map[string]string{
		`title: "What's New in Talos 1.14.0"`: `What's New in Talos 1.14.0`,
		`title: 'Single quoted'`:              `Single quoted`,
		`title: Unquoted Title`:               `Unquoted Title`,
		"  title:   \"Padded\"  ":             `Padded`,
		`title: "Colon: inside"`:              `Colon: inside`,
		`title: ""`:                           ``,
	} {
		m := titleRe.FindStringSubmatch(in)
		if m == nil {
			t.Errorf("%q did not match", in)
			continue
		}
		if m[1] != want {
			t.Errorf("%q -> %q, want %q", in, m[1], want)
		}
	}
}

// Release-notes pages are separate documents per release, not versions of one
// page, so each must be authoritative for its own release.
func TestResolvePerReleaseDocumentPointsAtItsOwnVersion(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.12/getting-started/whats-new": titled("What's New in Talos 1.12.0", "The 1.12 changes."),
		"v1.13/getting-started/whats-new": titled("What's New in Talos 1.13.0", "The 1.13 changes."),
		"v1.14/getting-started/whats-new": titled("What's New in Talos 1.14.0", "The 1.14 changes."),
	})
	// Every copy keeps its own release, including the oldest.
	wantTarget(t, resolveIn(t, root, "getting-started/whats-new", "v1.12", "v1.14"),
		"v1.12", "getting-started/whats-new")
	wantTarget(t, resolveIn(t, root, "getting-started/whats-new", "v1.13", "v1.14"),
		"v1.13", "getting-started/whats-new")
	wantTarget(t, resolveIn(t, root, "getting-started/whats-new", "v1.14", "v1.14"),
		"v1.14", "getting-started/whats-new")
}

// A page whose title merely changed wording is still one page across versions,
// so it must keep mapping forward.
func TestResolveTitleRewordingIsNotPerRelease(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/getting-started/deploy": titled("Deploy your first workload", "How to deploy."),
		"v1.14/getting-started/deploy": titled("Deploy First Workload", "How to deploy."),
	})
	wantTarget(t, resolveIn(t, root, "getting-started/deploy", "v1.13", "v1.14"),
		"v1.14", "getting-started/deploy")
}

// A version number in a title that never changes is not a per-release marker.
func TestResolveStableVersionedTitleIsNotPerRelease(t *testing.T) {
	root := writeTree(t, map[string]string{
		"v1.13/guides/k8s": titled("Kubernetes 1.36 notes", "Same page each release."),
		"v1.14/guides/k8s": titled("Kubernetes 1.36 notes", "Same page each release."),
	})
	wantTarget(t, resolveIn(t, root, "guides/k8s", "v1.13", "v1.14"), "v1.14", "guides/k8s")
}

// collectMDX expands directories and single files, and sorts for determinism.
func TestCollectMDX(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"b/two.mdx", "a/one.mdx", "a/skip.md"} {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := collectMDX([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "a", "one.mdx"), filepath.Join(root, "b", "two.mdx")}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (.md must be skipped)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d: got %s, want %s (output must be sorted)", i, got[i], want[i])
		}
	}

	// A single file is passed through as given.
	one := filepath.Join(root, "a", "one.mdx")
	got, err = collectMDX([]string{one})
	if err != nil || len(got) != 1 || got[0] != one {
		t.Errorf("single file: got %v err=%v", got, err)
	}

	if _, err := collectMDX([]string{filepath.Join(root, "nope")}); err == nil {
		t.Error("a missing path must be an error, not silently ignored")
	}
}

// An empty report must produce no output at all: the steady state is silence,
// which is what makes a non-empty report worth reading.
func TestReportsAreSilentWhenEmpty(t *testing.T) {
	if got := fallbackReport(map[string]target{}); got != "" {
		t.Errorf("fallbackReport on empty input returned %q, want \"\"", got)
	}
	if got := perReleaseReport(map[string]struct{}{}); got != "" {
		t.Errorf("perReleaseReport on empty input returned %q, want \"\"", got)
	}
}

// The fallback report is the only thing distinguishing a genuine deletion from
// a restructure the rules could not follow, so it must name every page and the
// version each fell back to.
func TestFallbackReportNamesEveryPageAndItsVersion(t *testing.T) {
	got := fallbackReport(map[string]target{
		"networking/device-selector": {version: "v1.11", relPath: "networking/device-selector"},
		"platforms/digital-rebar":    {version: "v1.7", relPath: "platforms/digital-rebar"},
		"storage/disk-management":    {version: "v1.10", relPath: "storage/disk-management"},
	})

	for _, want := range []string{
		"3 page(s) have no equivalent",
		"networking/device-selector -> v1.11",
		"platforms/digital-rebar -> v1.7",
		"storage/disk-management -> v1.10",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report is missing %q; got:\n%s", want, got)
		}
	}
}

// Map iteration is randomised, so the report must sort or its output would
// churn between runs and make diffs unreadable.
func TestFallbackReportIsOrdered(t *testing.T) {
	in := map[string]target{
		"zebra/page": {version: "v1.9", relPath: "zebra/page"},
		"alpha/page": {version: "v1.8", relPath: "alpha/page"},
		"middle/one": {version: "v1.7", relPath: "middle/one"},
	}
	first := fallbackReport(in)

	if strings.Index(first, "alpha/page") > strings.Index(first, "middle/one") ||
		strings.Index(first, "middle/one") > strings.Index(first, "zebra/page") {
		t.Errorf("entries are not sorted; got:\n%s", first)
	}
	// Same input must give byte-identical output every time.
	for i := 0; i < 20; i++ {
		if again := fallbackReport(in); again != first {
			t.Fatalf("output is not stable across runs:\n%s\nvs\n%s", first, again)
		}
	}
}

func TestPerReleaseReportNamesEveryPath(t *testing.T) {
	got := perReleaseReport(map[string]struct{}{
		"getting-started/what's-new-in-talos": {},
		"getting-started/release-notes":       {},
	})

	for _, want := range []string{
		"2 page path(s) are per-release documents",
		"authoritative for its own release",
		"getting-started/what's-new-in-talos",
		"getting-started/release-notes",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report is missing %q; got:\n%s", want, got)
		}
	}
	if strings.Index(got, "release-notes") > strings.Index(got, "what's-new") {
		t.Errorf("entries are not sorted; got:\n%s", got)
	}
}
