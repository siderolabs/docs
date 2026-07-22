package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- YAML structure ---

type Config struct {
	Navigation Navigation `yaml:"navigation"`
}

type Navigation struct {
	Version string `yaml:"version"`
	Tabs    []Tab  `yaml:"tabs"`
}

type Tab struct {
	Groups []Group `yaml:"groups"`
}

// Page can be either a plain string or a nested group.
// We use a custom type to handle both cases.
type Page struct {
	Path   string  // set if this is a plain page
	Group  string  // set if this is a nested group
	Folder string  // optional base folder override for the group
	Pages  []Page  // sub-pages if this is a nested group
}

func (p *Page) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		p.Path = value.Value
		return nil
	}
	// It's a mapping node (nested group).
	var m struct {
		Group  string `yaml:"group"`
		Folder string `yaml:"folder"`
		Pages  []Page `yaml:"pages"`
	}
	if err := value.Decode(&m); err != nil {
		return err
	}
	p.Group = m.Group
	p.Folder = m.Folder
	p.Pages = m.Pages
	return nil
}

type Group struct {
	Group  string `yaml:"group"`
	Folder string `yaml:"folder"`
	Pages  []Page `yaml:"pages"`
}

// --- Main logic ---

func main() {
	workspace := flag.String("workspace", ".", "Path to the workspace root (repo root)")
	fix := flag.Bool("fix", false, "Insert pages that exist on disk but are missing from the yaml nav, then exit 0 (best-effort, non-blocking)")
	flag.Parse()

	if err := os.Chdir(*workspace); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing to workspace %s: %v\n", *workspace, err)
		os.Exit(1)
	}

	// Find all yaml files in the root (excluding common.yaml which has no nav pages).
	allFiles, err := filepath.Glob("*.yaml")
	if err != nil || len(allFiles) == 0 {
		fmt.Fprintln(os.Stderr, "No yaml files found")
		os.Exit(1)
	}

	var yamlFiles []string
	for _, f := range allFiles {
		if f == "common.yaml" || f == "changelog.yaml" {
			continue
		}
		yamlFiles = append(yamlFiles, f)
	}
	sort.Strings(yamlFiles)

	// --fix is a best-effort sync: it inserts what it safely can and reports the
	// rest as warnings, always exiting 0 so it never blocks an upgrade run. The
	// plain (blocking) validation still runs afterwards as the correctness gate.
	if *fix {
		total := 0
		for _, yamlFile := range yamlFiles {
			inserted, err := fixVersion(yamlFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  ⚠ nav sync: error fixing %s: %v\n", yamlFile, err)
				continue
			}
			total += inserted
		}
		fmt.Printf("\nnav sync: inserted %d page(s)\n", total)
		return
	}

	totalIssues := 0

	for _, yamlFile := range yamlFiles {
		issues, err := validateVersion(yamlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating %s: %v\n", yamlFile, err)
			totalIssues++
			continue
		}

		version := strings.TrimSuffix(yamlFile, ".yaml")

		if len(issues) == 0 {
			fmt.Printf("%-12s OK\n", version)
		} else {
			fmt.Printf("%-12s %d issue(s)\n", version, len(issues))
			for _, issue := range issues {
				fmt.Printf("  %s\n", issue)
			}
			totalIssues += len(issues)
		}
	}

	fmt.Println()
	if totalIssues > 0 {
		fmt.Fprintf(os.Stderr, "Found %d issue(s) across all versions\n", totalIssues)
		os.Exit(1)
	}

	fmt.Println("All versions OK")
}

// validateVersion checks one yaml config against its content directories.
func validateVersion(yamlFile string) ([]string, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	// Collect all pages listed in the yaml and all explicit folder paths.
	listedPages := make(map[string]bool)
	walkDirs := make(map[string]bool)

	for _, tab := range config.Navigation.Tabs {
		for _, group := range tab.Groups {
			collectPages(group.Folder, group.Pages, listedPages)
			collectWalkDirs(group.Folder, group.Pages, walkDirs)
		}
	}

	if len(listedPages) == 0 {
		return nil, nil
	}

	// Collect all .mdx files that actually exist in the relevant directories.
	existingFiles := make(map[string]bool)
	for dir := range walkDirs {
		fullDir := filepath.Join("public", dir)
		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			continue
		}
		err = filepath.WalkDir(fullDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
				rel := strings.TrimPrefix(path, "public/")
				rel = strings.TrimSuffix(rel, ".mdx")
				existingFiles[rel] = true
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", fullDir, err)
		}
	}

	var issues []string

	// Listed in yaml but file doesn't exist.
	for page := range listedPages {
		if !existingFiles[page] {
			issues = append(issues, fmt.Sprintf("in yaml but file missing:  %s.mdx", page))
		}
	}

	// File exists but not listed in yaml.
	for file := range existingFiles {
		if !listedPages[file] {
			issues = append(issues, fmt.Sprintf("file exists but not in yaml: %s.mdx", file))
		}
	}

	sort.Strings(issues)
	return issues, nil
}

// collectWalkDirs gathers all non-root folder paths to use for directory walking.
func collectWalkDirs(folder string, pages []Page, out map[string]bool) {
	clean := strings.Trim(folder, "/")
	if clean != "" {
		out[clean] = true
	}
	for _, page := range pages {
		if page.Group != "" {
			subFolder := folder
			if page.Folder != "" {
				subFolder = page.Folder
			}
			collectWalkDirs(subFolder, page.Pages, out)
		}
	}
}

// collectPages recursively resolves all page paths relative to their folder.
func collectPages(folder string, pages []Page, out map[string]bool) {
	for _, page := range pages {
		if page.Path != "" {
			// Plain page — resolve against current folder.
			resolved := resolvePage(folder, page.Path)
			out[resolved] = true
		} else {
			// Nested group — use its own folder if set, otherwise inherit parent.
			subFolder := folder
			if page.Folder != "" {
				subFolder = page.Folder
			}
			collectPages(subFolder, page.Pages, out)
		}
	}
}

// resolvePage combines a folder and page path into a normalized key.
func resolvePage(folder, page string) string {
	page = strings.TrimSuffix(page, ".mdx")

	folderRel := strings.Trim(strings.TrimPrefix(folder, "public/"), "/")

	if folderRel == "" {
		return page
	}

	// Don't double up if page already has the folder prefix.
	if strings.HasPrefix(page, folderRel+"/") {
		return page
	}

	return folderRel + "/" + page
}

// --- Auto-insert (--fix) ---

// navEntry is one plain page listed under a group: its resolved key (folder-joined,
// no .mdx — the same key space validate uses) and the raw string as written in the
// yaml (with .mdx), used to locate it in the source text.
type navEntry struct {
	resolved string
	stored   string
}

// navGroup is a group that directly contains page strings, with its display title.
// It is the unit we match a new file's folder against and insert into.
type navGroup struct {
	title   string
	entries []navEntry
}

// acronyms maps a lowercase folder name to the display title used when a brand-new
// folder needs a freshly created section. It only matters the first time a new
// folder appears; from then on the section is matched by the folder its pages live
// in (not by title), so any human correction to the title sticks permanently.
var acronyms = map[string]string{
	"cri":  "CRI",
	"cli":  "CLI",
	"api":  "API",
	"cni":  "CNI",
	"tls":  "TLS",
	"dns":  "DNS",
	"kms":  "KMS",
	"lvm":  "LVM",
	"raid": "RAID",
	"bgp":  "BGP",
	"vrf":  "VRF",
	"vlan": "VLAN",
	"vip":  "VIP",
	"dhcp": "DHCP",
}

// labelFor derives a section title from a folder name, using the acronym overrides
// above and falling back to simple title-casing.
func labelFor(folder string) string {
	if folder == "" {
		return ""
	}
	if v, ok := acronyms[strings.ToLower(folder)]; ok {
		return v
	}
	return strings.ToUpper(folder[:1]) + folder[1:]
}

// fixVersion inserts any pages that exist on disk but are missing from one version's
// yaml nav, placing each in the group that owns its folder (or creating a section for
// a brand-new folder). It returns the number of pages inserted. Ambiguous or
// unplaceable files are reported as warnings and skipped — never an error.
func fixVersion(yamlFile string) (int, error) {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return 0, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return 0, fmt.Errorf("parsing yaml: %w", err)
	}

	var groups []navGroup
	listedPages := make(map[string]bool)
	walkDirs := make(map[string]bool)
	for _, tab := range config.Navigation.Tabs {
		for _, group := range tab.Groups {
			collectGroups(group.Folder, group.Group, group.Pages, &groups)
			collectPages(group.Folder, group.Pages, listedPages)
			collectWalkDirs(group.Folder, group.Pages, walkDirs)
		}
	}
	if len(listedPages) == 0 {
		return 0, nil
	}

	existingFiles, err := gatherExisting(walkDirs)
	if err != nil {
		return 0, err
	}

	// Missing = on disk but not listed in the yaml.
	var missing []string
	for file := range existingFiles {
		if !listedPages[file] {
			missing = append(missing, file)
		}
	}
	if len(missing) == 0 {
		return 0, nil
	}
	sort.Strings(missing)

	// Map each folder (directory of a resolved key) to the groups that own it.
	dirOwners := make(map[string][]int)
	for gi, g := range groups {
		seen := make(map[string]bool)
		for _, e := range g.entries {
			d := path.Dir(e.resolved)
			if !seen[d] {
				dirOwners[d] = append(dirOwners[d], gi)
				seen[d] = true
			}
		}
	}

	// created records folders we made a fresh section for in this run, so a second
	// missing file in the same brand-new folder is inserted into that section
	// instead of creating a duplicate.
	type createdSection struct{ prefix, entryIndent string }
	created := make(map[string]createdSection)

	lines := strings.Split(string(data), "\n")
	inserted := 0
	for _, r := range missing {
		d := path.Dir(r)
		if c, ok := created[d]; ok {
			if insertEntry(&lines, c.prefix, path.Base(r), c.entryIndent, "\"") {
				inserted++
			}
			continue
		}
		owners := dirOwners[d]
		switch len(owners) {
		case 1:
			if insertIntoGroup(&lines, groups[owners[0]], d, r) {
				inserted++
			} else {
				fmt.Fprintf(os.Stderr, "  ⚠ nav sync: could not place %s.mdx (no anchor in %q); add by hand\n", r, groups[owners[0]].title)
			}
		case 0:
			ok, prefix, entryIndent := createGroupAndInsert(&lines, groups, d, r)
			if ok {
				inserted++
				created[d] = createdSection{prefix: prefix, entryIndent: entryIndent}
			} else {
				fmt.Fprintf(os.Stderr, "  ⚠ nav sync: no section for new folder %q; add %s.mdx by hand\n", path.Base(d), r)
			}
		default:
			var names []string
			for _, oi := range owners {
				names = append(names, groups[oi].title)
			}
			sort.Strings(names)
			fmt.Fprintf(os.Stderr, "  ⚠ nav sync: %s.mdx is ambiguous (folder %q used by %s); add by hand\n", r, path.Base(d), strings.Join(names, ", "))
		}
	}

	if inserted > 0 {
		if err := os.WriteFile(yamlFile, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return inserted, err
		}
		fmt.Printf("%-12s inserted %d page(s)\n", strings.TrimSuffix(yamlFile, ".yaml"), inserted)
	}
	return inserted, nil
}

// collectGroups walks the nav tree and records every group that directly contains
// page strings, along with its title. A group with both pages and subgroups is
// recorded for its own pages and recursed into for the subgroups.
func collectGroups(folder, title string, pages []Page, out *[]navGroup) {
	var ents []navEntry
	for _, page := range pages {
		if page.Path != "" {
			ents = append(ents, navEntry{resolved: resolvePage(folder, page.Path), stored: page.Path})
		} else {
			sub := folder
			if page.Folder != "" {
				sub = page.Folder
			}
			collectGroups(sub, page.Group, page.Pages, out)
		}
	}
	if len(ents) > 0 {
		*out = append(*out, navGroup{title: title, entries: ents})
	}
}

// gatherExisting collects every .mdx file (as a resolved key) under the given walk
// directories — the same set validate compares against.
func gatherExisting(walkDirs map[string]bool) (map[string]bool, error) {
	existingFiles := make(map[string]bool)
	for dir := range walkDirs {
		fullDir := filepath.Join("public", dir)
		if _, err := os.Stat(fullDir); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(fullDir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(p, ".mdx") {
				rel := strings.TrimPrefix(p, "public/")
				rel = strings.TrimSuffix(rel, ".mdx")
				existingFiles[rel] = true
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", fullDir, err)
		}
	}
	return existingFiles, nil
}

// insertIntoGroup adds a page for `resolved` into an existing group, alphabetically
// among the group's siblings that share the same folder `dir`. It derives the path
// prefix, indentation and quote style from an existing sibling so the diff is a
// single new line, then delegates placement to insertEntry.
func insertIntoGroup(lines *[]string, g navGroup, dir, resolved string) bool {
	var anchorStored string
	for _, e := range g.entries {
		if path.Dir(e.resolved) == dir {
			anchorStored = e.stored
			break
		}
	}
	if anchorStored == "" {
		return false
	}

	anchorIdx := findEntryLine(*lines, anchorStored)
	if anchorIdx < 0 {
		return false
	}
	anchorLine := (*lines)[anchorIdx]
	indent := leadingWS(anchorLine)
	quote := "\""
	if strings.Contains(anchorLine, "'"+anchorStored+"'") {
		quote = "'"
	}

	return insertEntry(lines, path.Dir(anchorStored), path.Base(resolved), indent, quote)
}

// insertEntry adds a `- "<prefix>/<base>.mdx"` line in alphabetical order among the
// entries already in that same prefix folder. It scans the current lines each call,
// so entries inserted earlier in the same run are taken into account (keeping a
// batch of new siblings correctly ordered). Returns false if there is nothing in the
// prefix to anchor against.
func insertEntry(lines *[]string, prefixDir, baseName, entryIndent, quote string) bool {
	newStored := prefixDir + "/" + baseName + ".mdx"

	type cur struct {
		stored string
		idx    int
	}
	var curs []cur
	for i, ln := range *lines {
		p := extractQuoted(ln)
		if p != "" && path.Dir(p) == prefixDir {
			curs = append(curs, cur{p, i})
		}
	}
	if len(curs) == 0 {
		return false
	}
	for _, c := range curs {
		if c.stored == newStored {
			return true // already present
		}
	}
	sort.Slice(curs, func(i, j int) bool { return curs[i].stored < curs[j].stored })

	newLine := entryIndent + "- " + quote + newStored + quote

	insertAt := -1
	for _, c := range curs {
		if c.stored > newStored {
			insertAt = c.idx
			break
		}
	}
	if insertAt < 0 {
		last := curs[0].idx
		for _, c := range curs {
			if c.idx > last {
				last = c.idx
			}
		}
		insertAt = last + 1
	}

	*lines = insertLines(*lines, insertAt, []string{newLine})
	return true
}

// extractQuoted returns the first single- or double-quoted value on a line, or "".
func extractQuoted(line string) string {
	for _, q := range []byte{'"', '\''} {
		i := strings.IndexByte(line, q)
		if i < 0 {
			continue
		}
		j := strings.IndexByte(line[i+1:], q)
		if j < 0 {
			continue
		}
		return line[i+1 : i+1+j]
	}
	return ""
}

// createGroupAndInsert creates a new section for a brand-new folder and inserts the
// page, placing the section alphabetically among the sibling sections under the same
// parent folder. On success it returns the entry path prefix and the page-entry
// indentation, so any further files in the same new folder can be added to this
// section rather than duplicating it. Best-effort: returns false if it can't anchor.
func createGroupAndInsert(lines *[]string, groups []navGroup, dir, resolved string) (bool, string, string) {
	parent := path.Dir(dir)

	type sibGroup struct {
		title  string
		indent string
		prefix string
		hdr    int
	}
	var sibs []sibGroup
	for _, g := range groups {
		// A true sibling section holds exactly one folder, and that folder is a
		// direct child of `parent`. This excludes container groups (e.g. the
		// "Reference" group, which spans several folders at a shallower nesting)
		// and multi-folder groups (e.g. "Misc"), so the new section is placed at
		// the correct level among the per-folder sections.
		dirSet := make(map[string]bool)
		for _, e := range g.entries {
			dirSet[path.Dir(e.resolved)] = true
		}
		if len(dirSet) != 1 {
			continue
		}
		var groupDir string
		for d := range dirSet {
			groupDir = d
		}
		if path.Dir(groupDir) != parent {
			continue
		}
		anchorStored := g.entries[0].stored
		li := findEntryLine(*lines, anchorStored)
		if li < 0 {
			continue
		}
		hdr := nearestGroupHeaderIdx(*lines, li)
		if hdr < 0 {
			continue
		}
		sibs = append(sibs, sibGroup{
			title:  g.title,
			indent: leadingWS((*lines)[hdr]),
			prefix: path.Dir(path.Dir(anchorStored)),
			hdr:    hdr,
		})
	}
	if len(sibs) == 0 {
		return false, "", ""
	}
	sort.Slice(sibs, func(i, j int) bool { return sibs[i].title < sibs[j].title })

	newTitle := labelFor(path.Base(dir))
	indent := sibs[0].indent
	entryIndent := indent + "    "
	prefix := sibs[0].prefix + "/" + path.Base(dir)
	newStored := prefix + "/" + path.Base(resolved) + ".mdx"
	block := []string{
		indent + "- group: \"" + newTitle + "\"",
		indent + "  pages:",
		entryIndent + "- \"" + newStored + "\"",
	}

	insertAt := -1
	for _, s := range sibs {
		if s.title > newTitle {
			insertAt = s.hdr
			break
		}
	}
	if insertAt < 0 {
		last := sibs[len(sibs)-1]
		insertAt = blockEnd(*lines, last.hdr, last.indent)
	}

	*lines = insertLines(*lines, insertAt, block)
	return true, prefix, entryIndent
}

// findEntryLine returns the index of the line that lists the given stored page path
// (quoted), or -1 if not found.
func findEntryLine(lines []string, stored string) int {
	for i, ln := range lines {
		if strings.Contains(ln, "\""+stored+"\"") || strings.Contains(ln, "'"+stored+"'") {
			return i
		}
	}
	return -1
}

// nearestGroupHeaderIdx scans upward from idx for the nearest `- group:` line.
func nearestGroupHeaderIdx(lines []string, idx int) int {
	for i := idx; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimLeft(lines[i], " \t"), "- group:") {
			return i
		}
	}
	return -1
}

// blockEnd returns the index of the first line that is no longer part of the group
// whose header is at hdrIdx (i.e. the next line indented no deeper than the header,
// skipping blank lines).
func blockEnd(lines []string, hdrIdx int, groupIndent string) int {
	gi := len(groupIndent)
	end := hdrIdx + 1
	for end < len(lines) {
		if strings.TrimSpace(lines[end]) == "" {
			end++
			continue
		}
		if len(leadingWS(lines[end])) > gi {
			end++
			continue
		}
		break
	}
	return end
}

// leadingWS returns the leading whitespace (spaces/tabs) of a line.
func leadingWS(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

// insertLines returns lines with `add` spliced in at index `at`.
func insertLines(lines []string, at int, add []string) []string {
	if at < 0 {
		at = 0
	}
	if at > len(lines) {
		at = len(lines)
	}
	out := make([]string, 0, len(lines)+len(add))
	out = append(out, lines[:at]...)
	out = append(out, add...)
	out = append(out, lines[at:]...)
	return out
}
