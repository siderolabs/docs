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
