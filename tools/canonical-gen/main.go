// Command canonical-gen makes sure every Talos documentation page declares a
// `canonical:` link in its YAML frontmatter, pointing at the page that is
// currently authoritative for that content.
//
// # Why this is not just a path rewrite
//
// Each Talos version lives in its own directory and every directory is
// permanent, so when the docs are restructured nothing is ever "moved": v1.11
// keeps `networking/vip.mdx` forever while v1.12 onwards has
// `networking/advanced/vip.mdx`. The two paths are a correspondence between
// diverged trees, not a rename, so neither the filesystem nor git history
// records the relationship. It has to be inferred.
//
// # How a canonical target is chosen
//
// For a page at public/talos/<own>/<relPath>.mdx, in order:
//
//  0. <relPath> is a per-release document (release notes and the like, detected
//     from a title that both varies across versions and names a release) ->
//     point at the page's own version. "What's New in Talos 1.6.0" is not an
//     outdated rendering of the 1.14 page; they are different documents.
//  1. <relPath> exists in the current version -> point there. (Covers the page
//     already being in the current version, which points at itself.)
//  2. Otherwise find the newest version that still has <relPath>, and look at
//     the pages born in the version right after it -- the ones that appeared
//     exactly when this page disappeared. Pick a successor from that pool by
//     the first rule that yields exactly one candidate:
//     a. same file name
//     b. <relPath> became a directory holding a single page
//     c. file-name word overlap (wireguard-network -> wireguard)
//     d. body similarity >= contentSimMin
//     Rules a-c ignore page content, so they survive a full rewrite; rule d
//     catches renames that preserved the text. A winner is then resolved
//     recursively, so a page that moves twice still lands on the final page.
//  3. If no rule wins -- the page was split into several pages, or simply
//     dropped -- point at the newest version that still has <relPath>. That is
//     the newest surviving copy of this exact content, it always exists, and it
//     consolidates every older copy onto one URL instead of leaving each to
//     claim authority over the same content.
//
// Nothing is guessed: a rule must produce a single unambiguous candidate or it
// is skipped. Pages that fall through to step 3 are reported so a genuine
// deletion is distinguishable from a restructure the rules could not follow.
//
// Usage:
//
//	canonical-gen [flags] [path ...]
//
// Each path is a file or a directory to walk for .mdx files; with no path the
// tool processes public/talos. Files outside a public/talos/<version>/ tree are
// skipped. With --check nothing is written and the tool exits non-zero if any
// file would change.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const baseURL = "https://docs.siderolabs.com"

// Thresholds for the two scored successor rules. Both were calibrated against
// the real docs tree: known-good pairs score 0.70-0.92 on body similarity and
// unrelated pages 0.05-0.34, so contentSimMin sits in the empty band between.
const (
	tokenOverlapMin = 0.34
	contentSimMin   = 0.55
)

// versionRe extracts `export const version = 'v1.14'` from custom-variables.mdx.
// It anchors on `version` immediately followed by whitespace/`=` so it never
// matches the sibling `version_v1_6`, `version_v1_7`, ... constants.
var versionRe = regexp.MustCompile(`(?m)^\s*export\s+const\s+version\s*=\s*['"]([^'"]+)['"]`)

// talosPathRe pulls the version directory and the page path out of a file path,
// tolerating any prefix (absolute, ./, ../../) before "public/talos/".
var talosPathRe = regexp.MustCompile(`public/talos/([^/]+)/(.+)\.mdx$`)

// canonicalRe matches an existing `canonical:` frontmatter line.
var canonicalRe = regexp.MustCompile(`^\s*canonical\s*:`)

// versionDirRe matches a version directory name such as "v1.14".
var versionDirRe = regexp.MustCompile(`^v(\d+)\.(\d+)$`)

// titleRe pulls the value out of a `title:` frontmatter line, with or without
// surrounding quotes.
var titleRe = regexp.MustCompile(`^\s*title\s*:\s*["']?(.*?)["']?\s*$`)

// titleVersionRe matches a release number inside a page title, e.g. the
// "1.14.0" in "What's New in Talos 1.14.0".
var titleVersionRe = regexp.MustCompile(`\d+\.\d+`)

func main() {
	varsPath := flag.String("variables", "public/snippets/custom-variables.mdx",
		"path to the custom-variables.mdx file holding `export const version`")
	version := flag.String("version", "",
		"current Talos version for the canonical URL (default: read from --variables)")
	check := flag.Bool("check", false,
		"report files that need changes and exit non-zero without writing any")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: canonical-gen [flags] [path ...]")
		fmt.Fprintln(os.Stderr, "  Ensures every Talos .mdx page has a correct `canonical:` frontmatter link.")
		fmt.Fprintln(os.Stderr, "  Each path is a file or directory; with none, processes public/talos.")
		flag.PrintDefaults()
	}
	flag.Parse()

	ver := *version
	if ver == "" {
		v, err := readVersion(*varsPath)
		if err != nil {
			fatalf("%v", err)
		}
		ver = v
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"public/talos"}
	}

	files, err := collectMDX(paths)
	if err != nil {
		fatalf("%v", err)
	}

	var (
		ix         *index
		changed    []string
		fellBack   = map[string]target{}
		perRelease = map[string]struct{}{}
		processed  int
	)

	for _, f := range files {
		page, ok := parseTalosPage(f)
		if !ok {
			// Not a versioned Talos page -- nothing to do.
			continue
		}

		// The index needs every version, not just the files we were asked to
		// process, so build it once from the tree the first page points into.
		if ix == nil {
			ix, err = newIndex(page.prefix + "public/talos")
			if err != nil {
				fatalf("%v", err)
			}
			if !ix.hasVersion(ver) {
				fatalf("current version %s has no directory under %s", ver, ix.root)
			}
		}

		t := ix.resolve(page.relPath, page.version, ver, map[string]bool{})
		if t.version == "" {
			fmt.Fprintf(os.Stderr, "canonical-gen: %s: could not determine a canonical target; skipping\n", f)
			continue
		}
		switch {
		case ix.isPerRelease(page.relPath):
			perRelease[page.relPath] = struct{}{}
		case t.version != ver:
			fellBack[page.relPath] = t
		}

		did, err := process(f, canonicalFor(t.version, t.relPath), *check)
		if err != nil {
			fatalf("%v", err)
		}
		processed++
		if did {
			changed = append(changed, f)
		}
	}

	fmt.Fprint(os.Stderr, perReleaseReport(perRelease))
	fmt.Fprint(os.Stderr, fallbackReport(fellBack))

	if *check {
		if len(changed) > 0 {
			fmt.Fprintf(os.Stderr, "canonical-gen: %d file(s) need a canonical link:\n", len(changed))
			for _, f := range changed {
				fmt.Fprintf(os.Stderr, "  %s\n", f)
			}
			os.Exit(1)
		}
		fmt.Printf("canonical-gen: all %d file(s) have a correct canonical link.\n", processed)
		return
	}

	for _, f := range changed {
		fmt.Printf("updated %s\n", f)
	}
	fmt.Printf("canonical-gen: updated %d of %d file(s) for Talos %s.\n", len(changed), processed, ver)
}

// fallbackReport describes the pages that could not be resolved to the current
// version, grouped by page path so a page present in six old versions is
// mentioned once. It returns "" when there is nothing to report: an empty
// report is the steady state, which is what makes a non-empty one worth
// reading. Returning the text rather than printing it keeps this testable --
// this report is the only thing separating a genuine deletion from a
// restructure the rules could not follow.
func fallbackReport(fellBack map[string]target) string {
	if len(fellBack) == 0 {
		return ""
	}
	rels := sortedKeys(fellBack)

	var b strings.Builder
	fmt.Fprintf(&b, "\ncanonical-gen: %d page(s) have no equivalent in the current version;\n", len(rels))
	b.WriteString("  older copies now point at the newest version that still has them:\n")
	for _, r := range rels {
		fmt.Fprintf(&b, "    %s -> %s\n", r, fellBack[r].version)
	}
	return b.String()
}

// perReleaseReport names the page paths treated as per-release documents, so
// the exemption is visible rather than silent. It returns "" when there is
// nothing to report.
func perReleaseReport(perRelease map[string]struct{}) string {
	if len(perRelease) == 0 {
		return ""
	}
	rels := sortedKeys(perRelease)

	var b strings.Builder
	fmt.Fprintf(&b, "\ncanonical-gen: %d page path(s) are per-release documents;\n", len(rels))
	b.WriteString("  every version's copy is authoritative for its own release:\n")
	for _, r := range rels {
		fmt.Fprintf(&b, "    %s\n", r)
	}
	return b.String()
}

// sortedKeys returns a map's keys in a stable order, so report output is
// deterministic rather than following Go's randomised map iteration.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// readVersion returns the value of `export const version` from the given file.
func readVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading version from %s: %w", path, err)
	}
	m := versionRe.FindSubmatch(data)
	if m == nil {
		return "", fmt.Errorf("no `export const version` found in %s", path)
	}
	return string(m[1]), nil
}

// collectMDX expands the given files/directories into a list of .mdx files.
// Directories are walked recursively.
func collectMDX(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			out = append(out, p)
			continue
		}
		err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// talosPage holds the pieces of a public/talos/<version>/<rest>.mdx path:
// everything before "public/talos/", the page's own version directory, and
// its path below that directory (without the .mdx suffix).
type talosPage struct {
	prefix  string
	version string
	relPath string
}

// parseTalosPage extracts the pieces of a Talos page path, and false if the
// path is not under a public/talos/<version>/ tree.
func parseTalosPage(p string) (talosPage, bool) {
	slashPath := filepath.ToSlash(p)
	loc := talosPathRe.FindStringSubmatchIndex(slashPath)
	if loc == nil {
		return talosPage{}, false
	}
	return talosPage{
		prefix:  slashPath[:loc[0]],
		version: slashPath[loc[2]:loc[3]],
		relPath: slashPath[loc[4]:loc[5]],
	}, true
}

// canonicalFor returns the canonical URL for a page path under a version.
func canonicalFor(version, relPath string) string {
	return fmt.Sprintf("%s/talos/%s/%s", baseURL, version, relPath)
}

// target names the page a canonical should point at.
type target struct {
	version string
	relPath string
}

// index holds, for every version directory found under public/talos, the set of
// page paths it contains, plus a cache of page bodies for similarity scoring.
type index struct {
	root     string
	versions []string // oldest first
	pos      map[string]int
	pages    map[string]map[string]struct{}
	bodies   map[string]map[string]struct{}
	titles   map[string]string
	perRel   map[string]bool
}

// newIndex scans every version directory below root.
func newIndex(root string) (*index, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", root, err)
	}

	ix := &index{
		root:   root,
		pos:    map[string]int{},
		pages:  map[string]map[string]struct{}{},
		bodies: map[string]map[string]struct{}{},
		titles: map[string]string{},
		perRel: map[string]bool{},
	}

	for _, e := range entries {
		if !e.IsDir() || !versionDirRe.MatchString(e.Name()) {
			continue
		}
		v := e.Name()
		base := filepath.Join(root, v)
		set := map[string]struct{}{}
		err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".mdx") {
				return nil
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return err
			}
			set[strings.TrimSuffix(filepath.ToSlash(rel), ".mdx")] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, err
		}
		ix.pages[v] = set
		ix.versions = append(ix.versions, v)
	}

	if len(ix.versions) == 0 {
		return nil, fmt.Errorf("no version directories found under %s", root)
	}

	sort.Slice(ix.versions, func(i, j int) bool {
		return lessVersion(ix.versions[i], ix.versions[j])
	})
	for i, v := range ix.versions {
		ix.pos[v] = i
	}
	return ix, nil
}

// lessVersion orders "v1.9" before "v1.10" by comparing the numbers rather than
// the strings.
func lessVersion(a, b string) bool {
	am := versionDirRe.FindStringSubmatch(a)
	bm := versionDirRe.FindStringSubmatch(b)
	if am == nil || bm == nil {
		return a < b
	}
	amaj, _ := strconv.Atoi(am[1])
	bmaj, _ := strconv.Atoi(bm[1])
	if amaj != bmaj {
		return amaj < bmaj
	}
	amin, _ := strconv.Atoi(am[2])
	bmin, _ := strconv.Atoi(bm[2])
	return amin < bmin
}

func (ix *index) hasVersion(v string) bool {
	_, ok := ix.pages[v]
	return ok
}

func (ix *index) has(v, relPath string) bool {
	_, ok := ix.pages[v][relPath]
	return ok
}

// newestWith returns the newest version containing relPath.
func (ix *index) newestWith(relPath string) (string, bool) {
	for i := len(ix.versions) - 1; i >= 0; i-- {
		if ix.has(ix.versions[i], relPath) {
			return ix.versions[i], true
		}
	}
	return "", false
}

// next returns the version immediately after v.
func (ix *index) next(v string) (string, bool) {
	i, ok := ix.pos[v]
	if !ok || i+1 >= len(ix.versions) {
		return "", false
	}
	return ix.versions[i+1], true
}

// born returns the pages present in version `in` but not in `prev` -- the pages
// that appeared exactly when `prev`'s vanished pages disappeared.
func (ix *index) born(in, prev string) []string {
	var out []string
	for p := range ix.pages[in] {
		if !ix.has(prev, p) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// title returns a page's frontmatter title.
func (ix *index) title(version, relPath string) (string, bool) {
	key := version + "/" + relPath
	if t, ok := ix.titles[key]; ok {
		return t, t != ""
	}
	data, err := os.ReadFile(filepath.Join(ix.root, version, filepath.FromSlash(relPath)+".mdx"))
	if err != nil {
		ix.titles[key] = ""
		return "", false
	}
	lines := strings.Split(string(data), "\n")
	end, ok := frontmatterBounds(lines)
	if !ok {
		ix.titles[key] = ""
		return "", false
	}
	for i := 1; i < end; i++ {
		if m := titleRe.FindStringSubmatch(lines[i]); m != nil {
			ix.titles[key] = m[1]
			return m[1], m[1] != ""
		}
	}
	ix.titles[key] = ""
	return "", false
}

// isPerRelease reports whether relPath is a per-release document -- release
// notes and the like -- rather than one page maintained across versions.
//
// Such a page must canonicalize to itself. "What's New in Talos 1.6.0" is not
// an outdated rendering of "What's New in Talos 1.14.0"; they are different
// documents, and pointing the older at the newer would deindex content the
// newer page does not contain, which is worse than declaring no canonical at
// all.
//
// The signal is a title that both varies between versions and names a release:
// a page maintained across releases keeps one title ("Virtual (shared) IP"),
// and a plain wording change ("Deploy First Workload") names no release. When
// the two coincide, the version is part of the page's identity.
//
// A false positive here costs a missed consolidation; a false negative
// destroys a page's indexing. The rule is deliberately biased towards the
// former.
func (ix *index) isPerRelease(relPath string) bool {
	if v, ok := ix.perRel[relPath]; ok {
		return v
	}
	distinct := map[string]struct{}{}
	named := false
	for _, v := range ix.versions {
		if !ix.has(v, relPath) {
			continue
		}
		t, ok := ix.title(v, relPath)
		if !ok {
			continue
		}
		distinct[t] = struct{}{}
		if titleVersionRe.MatchString(t) {
			named = true
		}
	}
	out := named && len(distinct) > 1
	ix.perRel[relPath] = out
	return out
}

// resolve returns the page a canonical for relPath should point at, given the
// page's own version and the current version cur. See the package comment for
// the rules.
func (ix *index) resolve(relPath, own, cur string, seen map[string]bool) target {
	// A per-release document is authoritative for its own release.
	if ix.isPerRelease(relPath) {
		return target{own, relPath}
	}

	if ix.has(cur, relPath) {
		return target{cur, relPath}
	}

	last, ok := ix.newestWith(relPath)
	if !ok {
		return target{}
	}
	// Guard against a successor chain that loops back on itself.
	key := last + "/" + relPath
	if seen[key] {
		return target{last, relPath}
	}
	seen[key] = true

	died, ok := ix.next(last)
	if !ok {
		return target{last, relPath}
	}

	if succ, ok := ix.pickSuccessor(relPath, died, last); ok {
		if t := ix.resolve(succ, died, cur, seen); t.version != "" {
			return t
		}
	}

	// Split or dropped: the newest surviving copy of this exact content.
	return target{last, relPath}
}

// pickSuccessor chooses the page in version `died` that supersedes relPath from
// version `prev`, or false when no rule produces a single unambiguous
// candidate.
func (ix *index) pickSuccessor(relPath, died, prev string) (string, bool) {
	pool := ix.born(died, prev)
	if len(pool) == 0 {
		return "", false
	}
	ownBase := path.Base(relPath)

	// (a) Same file name in a different folder. Several matches fall through to
	// the later rules rather than giving up, because body similarity can still
	// separate them; rule (b) is the opposite, because several children mean a
	// split and no single page can stand in for it.
	var same []string
	for _, c := range pool {
		if path.Base(c) == ownBase {
			same = append(same, c)
		}
	}
	if len(same) == 1 {
		return same[0], true
	}

	// (b) The page became a directory. Only a directory holding exactly one
	// page is a move; several pages is a split, which falls back instead.
	var children []string
	for _, c := range pool {
		if strings.HasPrefix(c, relPath+"/") {
			children = append(children, c)
		}
	}
	if len(children) == 1 {
		return children[0], true
	}
	if len(children) > 1 {
		return "", false
	}

	// (c) File-name word overlap.
	if c, ok := bestUnique(pool, func(c string) float64 {
		return jaccard(words(ownBase), words(path.Base(c)))
	}, tokenOverlapMin); ok {
		return c, true
	}

	// (d) Body similarity.
	if src, ok := ix.body(prev, relPath); ok {
		if c, ok := bestUnique(pool, func(c string) float64 {
			other, ok := ix.body(died, c)
			if !ok {
				return 0
			}
			return dice(src, other)
		}, contentSimMin); ok {
			return c, true
		}
	}

	return "", false
}

// bestUnique returns the single highest-scoring candidate, if its score clears
// min and nothing else ties it.
func bestUnique(pool []string, score func(string) float64, min float64) (string, bool) {
	var (
		best float64
		win  string
		ties int
	)
	for _, c := range pool {
		s := score(c)
		switch {
		case s > best:
			best, win, ties = s, c, 1
		case s == best && s > 0:
			ties++
		}
	}
	if best >= min && ties == 1 {
		return win, true
	}
	return "", false
}

// words splits a file name into its lowercase parts.
func words(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == '-' || r == '_' || r == '.'
	}) {
		out[w] = struct{}{}
	}
	return out
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// dice is the Sorensen-Dice coefficient over two sets of body lines.
func dice(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	return 2 * float64(inter) / float64(len(a)+len(b))
}

// body returns the set of meaningful body lines for a page, excluding
// frontmatter and import statements, which are near-identical across all pages
// and would otherwise inflate every score.
func (ix *index) body(version, relPath string) (map[string]struct{}, bool) {
	key := version + "/" + relPath
	if b, ok := ix.bodies[key]; ok {
		return b, len(b) > 0
	}
	data, err := os.ReadFile(filepath.Join(ix.root, version, filepath.FromSlash(relPath)+".mdx"))
	if err != nil {
		ix.bodies[key] = nil
		return nil, false
	}
	lines := strings.Split(string(data), "\n")
	if end, ok := frontmatterBounds(lines); ok {
		lines = lines[end+1:]
	}
	set := map[string]struct{}{}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) <= 3 || strings.HasPrefix(l, "import ") {
			continue
		}
		set[l] = struct{}{}
	}
	ix.bodies[key] = set
	return set, len(set) > 0
}

// process inserts or corrects the canonical link in one file. It returns whether
// the file needed a change. With check=true it never writes.
func process(path, want string, check bool) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	lines := strings.Split(string(data), "\n")

	end, ok := frontmatterBounds(lines)
	if !ok {
		fmt.Fprintf(os.Stderr, "canonical-gen: %s has no frontmatter block; skipping\n", path)
		return false, nil
	}

	line := "canonical: " + want
	changed := false
	if idx := findCanonical(lines, end); idx >= 0 {
		if lines[idx] != line {
			lines[idx] = line
			changed = true
		}
	} else {
		// Insert as the last frontmatter field, just before the closing "---".
		lines = append(lines[:end], append([]string{line}, lines[end:]...)...)
		changed = true
	}

	if !changed || check {
		return changed, nil
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// frontmatterBounds returns the index of the closing fence of a leading
// frontmatter block, and whether one was found. The opening fence is always
// line 0. Trailing whitespace on a fence line is tolerated ("--- ").
func frontmatterBounds(lines []string) (end int, ok bool) {
	if len(lines) == 0 || !isFence(lines[0]) {
		return 0, false
	}
	for i := 1; i < len(lines); i++ {
		if isFence(lines[i]) {
			return i, true
		}
	}
	return 0, false
}

// findCanonical returns the index of an existing `canonical:` line inside the
// frontmatter block, exclusive of the fences at line 0 and line end, or -1.
func findCanonical(lines []string, end int) int {
	for i := 1; i < end; i++ {
		if canonicalRe.MatchString(lines[i]) {
			return i
		}
	}
	return -1
}

func isFence(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "canonical-gen: "+format+"\n", args...)
	os.Exit(1)
}
