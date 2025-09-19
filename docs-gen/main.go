package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// Config represents the YAML configuration file structure
type Config struct {
	Schema       string           `yaml:"schema"`
	Theme        string           `yaml:"theme"`
	Name         string           `yaml:"name"`
	Colors       Colors           `yaml:"colors"`
	Favicon      string           `yaml:"favicon"`
	Banner       *Banner          `yaml:"banner,omitempty"`
	Contextual   *Contextual      `yaml:"contextual,omitempty"`
	Logo         *Logo            `yaml:"logo,omitempty"`
	Navbar       *Navbar          `yaml:"navbar,omitempty"`
	Footer       *Footer          `yaml:"footer,omitempty"`
	Integrations *Integrations    `yaml:"integrations,omitempty"`
	Redirects    []Redirect       `yaml:"redirects,omitempty"`
	Navigation   NavigationConfig `yaml:"navigation"`
}

type Colors struct {
	Primary string `yaml:"primary" json:"primary"`
	Light   string `yaml:"light" json:"light"`
	Dark    string `yaml:"dark" json:"dark"`
}

type Banner struct {
	Content     string `yaml:"content" json:"content"`
	Dismissible bool   `yaml:"dismissible" json:"dismissible"`
}

type Contextual struct {
	Options []string `yaml:"options" json:"options"`
}

type Logo struct {
	Light string `yaml:"light" json:"light"`
	Dark  string `yaml:"dark" json:"dark"`
}

type Navbar struct {
	Links []NavLink `yaml:"links" json:"links"`
}

type NavLink struct {
	Label string `yaml:"label" json:"label"`
	Href  string `yaml:"href" json:"href"`
}

type Footer struct {
	Socials map[string]string `yaml:"socials" json:"socials"`
}

type Integrations struct {
	GA4 *GA4Integration `yaml:"ga4,omitempty" json:"ga4,omitempty"`
}

type GA4Integration struct {
	MeasurementId string `yaml:"measurementId" json:"measurementId"`
}

type NavigationConfig struct {
	Tabs   []TabConfig `yaml:"tabs"`
	Global *GlobalNav  `yaml:"global,omitempty"`
}

type TabConfig struct {
	Tab    string        `yaml:"tab"`
	Icon   string        `yaml:"icon,omitempty"`
	Groups []GroupConfig `yaml:"groups"`
}

type GroupConfig struct {
	Group  string      `yaml:"group"`
	Folder string      `yaml:"folder,omitempty"`
	Order  []string    `yaml:"order,omitempty"`
	Pages  []PageEntry `yaml:"pages,omitempty"`
}

type PageEntry struct {
	Page  string      `yaml:",omitempty"`
	Group string      `yaml:"group,omitempty"`
	Pages []PageEntry `yaml:"pages,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling to handle both string and object page entries
func (p *PageEntry) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as string first
	var str string
	if err := unmarshal(&str); err == nil {
		p.Page = str
		return nil
	}

	// If that fails, unmarshal as object
	type pageAlias PageEntry
	var page pageAlias
	if err := unmarshal(&page); err != nil {
		return err
	}

	*p = PageEntry(page)
	return nil
}

type GlobalNav struct {
	Anchors []Anchor `yaml:"anchors" json:"anchors"`
}

type Anchor struct {
	Anchor string `yaml:"anchor" json:"anchor"`
	Href   string `yaml:"href" json:"href"`
	Icon   string `yaml:"icon" json:"icon"`
}

type Redirect struct {
	Source      string `yaml:"source" json:"source"`
	Destination string `yaml:"destination" json:"destination"`
}

// MintlifyConfig represents the output docs.json structure
type MintlifyConfig struct {
	Schema       string             `json:"$schema"`
	Theme        string             `json:"theme"`
	Name         string             `json:"name"`
	Colors       Colors             `json:"colors"`
	Favicon      string             `json:"favicon"`
	Banner       *Banner            `json:"banner,omitempty"`
	Contextual   *Contextual        `json:"contextual,omitempty"`
	Logo         *Logo              `json:"logo,omitempty"`
	Navbar       *Navbar            `json:"navbar,omitempty"`
	Footer       *Footer            `json:"footer,omitempty"`
	Integrations *Integrations      `json:"integrations,omitempty"`
	Redirects    []Redirect         `json:"redirects,omitempty"`
	Navigation   MintlifyNavigation `json:"navigation"`
}

type MintlifyNavigation struct {
	Tabs   []MintlifyTab `json:"tabs,omitempty"`
	Global *GlobalNav    `json:"global,omitempty"`
}

type MintlifyTab struct {
	Tab    string          `json:"tab"`
	Icon   string          `json:"icon,omitempty"`
	Groups []MintlifyGroup `json:"groups"`
}

type MintlifyGroup struct {
	Group string      `json:"group"`
	Pages interface{} `json:"pages"`
}

func main() {
	var detectMissing bool
	var skipValidation bool
	flag.BoolVar(&detectMissing, "detect-missing", false, "Check for MDX files not included in config")
	flag.BoolVar(&skipValidation, "skip-validation", false, "Skip JSON schema validation")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run main.go [--detect-missing] [--skip-validation] <config1.yaml> [config2.yaml] ...")
		os.Exit(1)
	}

	configPaths := args

	// Read and merge multiple config files
	mergedConfig, err := mergeConfigs(configPaths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error processing config files: %v\n", err)
		os.Exit(1)
	}

	// If detect-missing flag is set, check for missing files
	if detectMissing {
		if err := checkMissingFiles(mergedConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error checking missing files: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Generate Mintlify config
	mintlifyConfig := MintlifyConfig{
		Schema:       mergedConfig.Schema,
		Theme:        mergedConfig.Theme,
		Name:         mergedConfig.Name,
		Colors:       mergedConfig.Colors,
		Favicon:      mergedConfig.Favicon,
		Banner:       mergedConfig.Banner,
		Contextual:   mergedConfig.Contextual,
		Logo:         mergedConfig.Logo,
		Navbar:       mergedConfig.Navbar,
		Footer:       mergedConfig.Footer,
		Integrations: mergedConfig.Integrations,
		Redirects:    mergedConfig.Redirects,
		Navigation: MintlifyNavigation{
			Global: mergedConfig.Navigation.Global,
		},
	}

	// Process navigation tabs
	for _, tabConfig := range mergedConfig.Navigation.Tabs {
		tab := MintlifyTab{
			Tab:  tabConfig.Tab,
			Icon: tabConfig.Icon,
		}

		for _, groupConfig := range tabConfig.Groups {
			group := MintlifyGroup{
				Group: groupConfig.Group,
			}

			// Only process explicitly defined pages - no automatic folder scanning
			if len(groupConfig.Pages) > 0 {
				pages, err := processManualPages(groupConfig.Pages, groupConfig.Folder)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error processing manual pages for group %s: %v\n", groupConfig.Group, err)
					continue
				}
				group.Pages = pages
			} else {
				continue
			}

			tab.Groups = append(tab.Groups, group)
		}

		mintlifyConfig.Navigation.Tabs = append(mintlifyConfig.Navigation.Tabs, tab)
	}

	// Output JSON to stdout
	jsonData, err := json.MarshalIndent(mintlifyConfig, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Validate against schema by default (unless skipped)
	if !skipValidation && mergedConfig.Schema != "" {
		if err := validateAgainstSchema(jsonData, mergedConfig.Schema); err != nil {
			fmt.Fprintf(os.Stderr, "Schema validation failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println(string(jsonData))
}

// mergeConfigs reads and merges multiple YAML config files
func mergeConfigs(configPaths []string) (Config, error) {
	var mergedConfig Config
	var allTabs []TabConfig

	for i, configPath := range configPaths {
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return Config{}, fmt.Errorf("error reading config file %s: %v", configPath, err)
		}

		var config Config
		if err := yaml.Unmarshal(configData, &config); err != nil {
			return Config{}, fmt.Errorf("error parsing config file %s: %v", configPath, err)
		}

		// Use the first config file for base settings
		if i == 0 {
			mergedConfig = config
			mergedConfig.Navigation.Tabs = nil // Clear tabs to rebuild
		}

		// Collect tabs from all files in order
		allTabs = append(allTabs, config.Navigation.Tabs...)
	}

	// Set merged tabs
	mergedConfig.Navigation.Tabs = allTabs

	return mergedConfig, nil
}

// scanFolder scans a folder for MDX files and returns page paths
func scanFolder(folder string, order []string) (interface{}, error) {
	var pages []interface{}
	var files []string
	var subGroups []MintlifyGroup

	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && path != folder {
			// Check if this subdirectory has MDX files
			subFiles, err := scanSubdirectory(path)
			if err != nil {
				return err
			}

			if len(subFiles) > 0 {
				// Create a subgroup
				baseName := strings.ReplaceAll(filepath.Base(path), "-", " ")
				groupName := strings.ToUpper(baseName[:1]) + baseName[1:]
				subGroup := MintlifyGroup{
					Group: groupName,
					Pages: subFiles,
				}
				subGroups = append(subGroups, subGroup)
			}
			return filepath.SkipDir // Don't recurse further
		}

		if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
			// Convert path to page reference (remove extension and folder prefix)
			pagePath := strings.TrimSuffix(path, ".mdx")
			files = append(files, pagePath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort files based on order if provided
	if len(order) > 0 {
		orderedFiles := make([]string, 0, len(files))
		fileSet := make(map[string]bool)

		for _, file := range files {
			fileSet[file] = true
		}

		// Add ordered files first
		for _, orderedFile := range order {
			fullPath := filepath.Join(folder, orderedFile)
			if strings.HasSuffix(fullPath, ".mdx") {
				fullPath = strings.TrimSuffix(fullPath, ".mdx")
			}
			if fileSet[fullPath] {
				orderedFiles = append(orderedFiles, fullPath)
				delete(fileSet, fullPath)
			}
		}

		// Add remaining files
		var remaining []string
		for file := range fileSet {
			remaining = append(remaining, file)
		}
		sort.Strings(remaining)
		orderedFiles = append(orderedFiles, remaining...)
		files = orderedFiles
	} else {
		sort.Strings(files)
	}

	// Add direct files as pages
	for _, file := range files {
		pages = append(pages, file)
	}

	// Add subgroups
	for _, subGroup := range subGroups {
		pages = append(pages, subGroup)
	}

	return pages, nil
}

// scanSubdirectory scans a subdirectory for MDX files
func scanSubdirectory(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mdx") {
			pagePath := strings.TrimSuffix(filepath.Join(dir, entry.Name()), ".mdx")
			files = append(files, pagePath)
		}
	}

	sort.Strings(files)
	return files, nil
}

// checkMissingFiles checks for MDX files not included in the configuration
func checkMissingFiles(config Config) error {
	// Collect all configured folders
	configuredFolders := make(map[string]bool)
	for _, tab := range config.Navigation.Tabs {
		for _, group := range tab.Groups {
			configuredFolders[group.Folder] = true
		}
	}

	// Find all MDX files in the repository
	var allMDXFiles []string
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and files
		if strings.HasPrefix(filepath.Base(path), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Collect MDX files
		if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
			allMDXFiles = append(allMDXFiles, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error walking directory: %v", err)
	}

	// Check which files are not in configured folders
	var missingFiles []string
	for _, file := range allMDXFiles {
		// Check if this file is in any configured folder
		inConfiguredFolder := false
		for folder := range configuredFolders {
			if strings.HasPrefix(file, folder+"/") {
				inConfiguredFolder = true
				break
			}
		}

		if !inConfiguredFolder {
			missingFiles = append(missingFiles, file)
		}
	}

	// Report results
	if len(missingFiles) == 0 {
		fmt.Println("✅ All MDX files are included in configured folders")
		return nil
	}

	fmt.Printf("⚠️  Found %d MDX files not included in any configured folder:\n\n", len(missingFiles))
	for _, file := range missingFiles {
		fmt.Printf("  - %s\n", file)
	}

	fmt.Printf("\nTo include these files, add their parent folders to docs-config.yaml navigation groups.\n")

	return nil
}

// validateAgainstSchema validates the generated JSON against the specified schema
func validateAgainstSchema(jsonData []byte, schemaURL string) error {
	// Create schema loader from URL
	schemaLoader := gojsonschema.NewReferenceLoader(schemaURL)

	// Create document loader from our generated JSON
	documentLoader := gojsonschema.NewBytesLoader(jsonData)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %v", err)
	}

	if !result.Valid() {
		var errorMessages []string
		for _, desc := range result.Errors() {
			errorMessages = append(errorMessages, fmt.Sprintf("  - %s", desc))
		}
		return fmt.Errorf("validation failed:\n%s", strings.Join(errorMessages, "\n"))
	}

	return nil
}

// processManualPages processes manually defined pages and subgroups
func processManualPages(pageEntries []PageEntry, basePath string) (interface{}, error) {
	var pages []interface{}

	for _, entry := range pageEntries {
		if entry.Page != "" {
			// This is a regular page entry
			pagePath := entry.Page

			// If basePath is provided and page doesn't start with it, prepend it
			if basePath != "" && !strings.HasPrefix(pagePath, basePath+"/") {
				pagePath = filepath.Join(basePath, entry.Page)
			}

			// Remove .mdx extension if present
			if strings.HasSuffix(pagePath, ".mdx") {
				pagePath = strings.TrimSuffix(pagePath, ".mdx")
			}

			pages = append(pages, pagePath)
		} else if entry.Group != "" {
			// This is a subgroup
			subGroup := MintlifyGroup{
				Group: entry.Group,
			}

			// Recursively process subgroup pages
			subPages, err := processManualPages(entry.Pages, basePath)
			if err != nil {
				return nil, fmt.Errorf("error processing subgroup %s: %v", entry.Group, err)
			}

			subGroup.Pages = subPages
			pages = append(pages, subGroup)
		}
	}

	return pages, nil
}
