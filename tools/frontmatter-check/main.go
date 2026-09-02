// Command frontmatter-check verifies that .mdx pages under public/omni,
// public/talos, and public/kubernetes-guides carry the frontmatter fields
// their section requires:
//
//   - Talos pages              : title, description, canonical
//   - Omni / Kubernetes guides : title, description
//
// With no file arguments, it scans every .mdx file under public/. Given file
// arguments (e.g. from `git diff`), it checks only those files, so it can be
// used as a fast pre-commit / changed-files gate.
//
// Every .mdx file must be accounted for: it either belongs to a section and
// is checked, or it is listed in `exempt` as deliberately not page content.
// A file matching neither is an error, so a new content tree cannot pass
// unchecked simply because nothing knows about it yet.
//
// It reports every file missing a required field and exits non-zero if any
// are found.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// contentRoot is the directory a full scan walks.
const contentRoot = "public"

// section describes one content directory and the frontmatter fields it requires.
type section struct {
	dir    string
	fields []string
}

var sections = []section{
	{dir: "public/talos", fields: []string{"title", "description", "canonical"}},
	{dir: "public/omni", fields: []string{"title", "description"}},
	{dir: "public/kubernetes-guides", fields: []string{"title", "description"}},
}

// exempt lists paths that are deliberately not page content and so carry no
// frontmatter requirements: the generated changelog, and the snippets holding
// shared `export const` variables. An entry may be a single file or a
// directory. Anything under public/ matching neither `sections` nor this list
// is reported as an error rather than skipped quietly -- see classify.
var exempt = []string{
	"public/changelog.mdx",
	"public/snippets",
}

// classification says how a file was handled.
type classification int

const (
	// classChecked: the file belongs to a section and its fields were verified.
	classChecked classification = iota
	// classExempt: the file is knowingly not page content.
	classExempt
	// classUnknown: the file belongs to no section and isn't exempt.
	classUnknown
)

// underPath reports whether clean is p itself or sits beneath it. It tolerates
// the relative prefixes the Makefile passes (e.g. ../../public/talos/foo.mdx),
// so a path is matched by its tail as well as its head.
func underPath(clean, p string) bool {
	return clean == p ||
		strings.HasSuffix(clean, "/"+p) ||
		strings.HasPrefix(clean, p+"/") ||
		strings.Contains(clean, "/"+p+"/")
}

// classify returns the required fields for a file and how it was classified.
func classify(file string) ([]string, classification) {
	clean := filepath.ToSlash(filepath.Clean(file))
	for _, sec := range sections {
		if underPath(clean, sec.dir) {
			return sec.fields, classChecked
		}
	}
	for _, p := range exempt {
		if underPath(clean, p) {
			return nil, classExempt
		}
	}
	return nil, classUnknown
}

func main() {
	workspace := flag.String("workspace", ".", "Path to the workspace root (repo root); ignored when file arguments are given")
	flag.Parse()

	files := flag.Args()

	if len(files) == 0 {
		if err := os.Chdir(*workspace); err != nil {
			fmt.Fprintf(os.Stderr, "Error changing to workspace %s: %v\n", *workspace, err)
			os.Exit(1)
		}

		var err error
		files, err = collectMDX(contentRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	var issues, exempted, unknown []string
	checked := 0

	for _, file := range files {
		fields, class := classify(file)
		switch class {
		case classExempt:
			exempted = append(exempted, file)
			continue
		case classUnknown:
			unknown = append(unknown, file)
			continue
		}
		checked++

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v\n", file, err)
			os.Exit(1)
		}

		fm, err := frontmatterFields(string(data))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: invalid frontmatter: %v", file, err))
			continue
		}
		for _, field := range fields {
			if strings.TrimSpace(fm[field]) == "" {
				issues = append(issues, fmt.Sprintf("%s: missing %q", file, field))
			}
		}
	}

	sort.Strings(issues)
	for _, issue := range issues {
		fmt.Println(issue)
	}

	fmt.Printf("\nChecked %d file(s): %d issue(s)\n", checked, len(issues))

	// Account for every file that wasn't checked, so the "Checked N" number
	// can always be reconciled against the number of files handed in.
	if len(exempted) > 0 {
		sort.Strings(exempted)
		fmt.Printf("Skipped %d exempt file(s): %s\n", len(exempted), strings.Join(exempted, ", "))
	}

	if len(unknown) > 0 {
		sort.Strings(unknown)
		fmt.Fprintf(os.Stderr, "\n%d file(s) belong to no known section:\n", len(unknown))
		for _, file := range unknown {
			fmt.Fprintf(os.Stderr, "  %s\n", file)
		}
		fmt.Fprintf(os.Stderr, "\nAdd the containing directory to `sections` (with the fields it\n"+
			"requires) or, if it isn't page content, to `exempt` in\n"+
			"tools/frontmatter-check/main.go. A new content tree must be\n"+
			"registered before it can pass this check.\n")
		os.Exit(1)
	}

	if len(issues) > 0 {
		os.Exit(1)
	}
}

// collectMDX returns every .mdx file under dir, sorted.
func collectMDX(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", dir, err)
	}
	sort.Strings(files)
	return files, nil
}

// frontmatterFields returns the string-valued key/value pairs found in the
// leading YAML frontmatter block (between the first pair of "---" lines). It
// parses the block as real YAML, so block scalars (`description: |`),
// quoting, and folding are all handled the way they actually resolve —
// a hand-rolled "split on the first colon" scan would treat a block
// scalar's `|`/`>` indicator as if it were the value itself.
func frontmatterFields(content string) (map[string]string, error) {
	fields := map[string]string{}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fields, nil
	}

	end := -1
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			end = i + 1
			break
		}
	}
	if end == -1 {
		return fields, nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &raw); err != nil {
		return nil, err
	}

	for key, value := range raw {
		if s, ok := value.(string); ok {
			fields[key] = strings.TrimSpace(s)
		}
	}

	return fields, nil
}
