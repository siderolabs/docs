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
	Description  string           `yaml:"description,omitempty"`
	Colors       Colors           `yaml:"colors"`
	Favicon      string           `yaml:"favicon"`
	Banner       *Banner          `yaml:"banner,omitempty"`
	Contextual   *Contextual      `yaml:"contextual,omitempty"`
	Logo         *Logo            `yaml:"logo,omitempty"`
	Thumbnails   *Thumbnails      `yaml:"thumbnails,omitempty"`
	Fonts        *Fonts           `yaml:"fonts,omitempty"`
	SEO          *SEO             `yaml:"seo,omitempty"`
	Search       *Search          `yaml:"search,omitempty"`
	Errors       *Errors          `yaml:"errors,omitempty"`
	Navbar       *Navbar          `yaml:"navbar,omitempty"`
	Footer       *Footer          `yaml:"footer,omitempty"`
	Integrations *Integrations    `yaml:"integrations,omitempty"`
	Redirects    []Redirect       `yaml:"redirects,omitempty"`
	Navigation   NavigationConfig `yaml:"navigation"`
	Homepage     *HomepageConfig  `yaml:"homepage,omitempty"`
	// ProductNav is where to write the generated map describing each product
	// to the browser: its display name, whether it carries versions, and its
	// call to action. None of this fits docs.json -- Mintlify's schema has no
	// such fields -- so it is handed over as a script instead.
	ProductNav string `yaml:"productNav,omitempty"`
	// ProductChromeCSS is where to write rules that depend on a product's page
	// paths, so those paths are never typed out by hand in a stylesheet.
	ProductChromeCSS string                 `yaml:"productChromeCss,omitempty"`
	VersionsMap      map[string][]TabConfig `yaml:"-"` // Internal use only, not from YAML
	// ProductVersionsMap is the products equivalent of VersionsMap: product key ->
	// one entry per versioned config file that declared it.
	ProductVersionsMap map[string][]ProductConfig `yaml:"-"`
	// ProductOrder lists product keys in the order they were first declared
	// across the config files. Ranging a map is randomised in Go, so the
	// generator walks this slice instead -- otherwise docs.json would come
	// out in a different order on each run and CI's `git diff --exit-code`
	// would fail at random.
	ProductOrder []string `yaml:"-"`
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

type Thumbnails struct {
	Appearance string `yaml:"appearance,omitempty" json:"appearance,omitempty"`
	Background string `yaml:"background,omitempty" json:"background,omitempty"`
}

type Fonts struct {
	Family  string     `yaml:"family,omitempty" json:"family,omitempty"`
	Weight  *int       `yaml:"weight,omitempty" json:"weight,omitempty"`
	Source  string     `yaml:"source,omitempty" json:"source,omitempty"`
	Format  string     `yaml:"format,omitempty" json:"format,omitempty"`
	Heading *FontStyle `yaml:"heading,omitempty" json:"heading,omitempty"`
	Body    *FontStyle `yaml:"body,omitempty" json:"body,omitempty"`
}

type FontStyle struct {
	Family string `yaml:"family,omitempty" json:"family,omitempty"`
	Weight *int   `yaml:"weight,omitempty" json:"weight,omitempty"`
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
}

type Navbar struct {
	Links   []NavLink   `yaml:"links" json:"links"`
	Primary *NavPrimary `yaml:"primary,omitempty" json:"primary,omitempty"`
}

type NavLink struct {
	Label string `yaml:"label" json:"label"`
	Href  string `yaml:"href" json:"href"`
}

type NavPrimary struct {
	Type  string `yaml:"type" json:"type"`
	Label string `yaml:"label" json:"label"`
	Href  string `yaml:"href" json:"href"`
}

type Footer struct {
	Socials map[string]string `yaml:"socials" json:"socials"`
}

type Integrations struct {
	GA4     *GA4Integration     `yaml:"ga4,omitempty" json:"ga4,omitempty"`
	Posthog *PosthogIntegration `yaml:"posthog,omitempty" json:"posthog,omitempty"`
}

type GA4Integration struct {
	MeasurementId string `yaml:"measurementId" json:"measurementId"`
}

type PosthogIntegration struct {
	ApiKey  string  `yaml:"apiKey" json:"apiKey"`
	ApiHost *string `yaml:"apiHost,omitempty" json:"apiHost,omitempty"`
}

type SEO struct {
	Metatags map[string]string `yaml:"metatags,omitempty" json:"metatags,omitempty"`
	Indexing string            `yaml:"indexing,omitempty" json:"indexing,omitempty"`
}

type Search struct {
	Prompt string `yaml:"prompt,omitempty" json:"prompt,omitempty"`
}

type Errors struct {
	Error404 *Error404 `yaml:"404,omitempty" json:"404,omitempty"`
}

type Error404 struct {
	Redirect    *bool  `yaml:"redirect,omitempty" json:"redirect,omitempty"`
	Title       string `yaml:"title,omitempty" json:"title,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

type NavigationConfig struct {
	Version string      `yaml:"version,omitempty"`
	Tabs    []TabConfig `yaml:"tabs"`
	// Products is the alternative to Tabs. Mintlify's schema treats the keys
	// under `navigation` as mutually exclusive, so a build uses one or the
	// other and the generator refuses a mix.
	Products []ProductConfig `yaml:"products,omitempty"`
	Global   *GlobalNav      `yaml:"global,omitempty"`
}

// ProductConfig is one product in the navigation. A product carries either its
// own version history (Talos, Talos Enterprise Linux, Talos Hypervisor) or a
// plain set of groups (Omni, Talos Director) -- never both, matching Mintlify.
type ProductConfig struct {
	Product     string `yaml:"product"`
	Name        string `yaml:"name,omitempty"`
	Icon        string `yaml:"icon,omitempty"`
	Color       string `yaml:"color,omitempty"`
	Description string `yaml:"description,omitempty"`
	Hidden      bool   `yaml:"hidden,omitempty"`
	// Family and Tag drive the generated homepage only. Mintlify's schema has
	// no notion of a product family, so neither is emitted into docs.json.
	Family string `yaml:"family,omitempty"`
	Tag    string `yaml:"tag,omitempty"`
	// Meta is the card's machine-set footer line ("SaaS and self-hosted").
	// A versioned product left without one gets "<newest version> current".
	Meta string `yaml:"meta,omitempty"`
	// Homepage set to false keeps the product out of the homepage catalog
	// while leaving it in the product switcher. Meant for products the
	// homepage reaches another way, like Kubernetes Guides in the Also row.
	Homepage *bool `yaml:"homepage,omitempty"`
	// CTA is the call to action shown while reading this product. A product
	// without one gets no button, rather than inheriting another product's.
	CTA *ProductCTA `yaml:"cta,omitempty"`
	// Sidebar set to false hides the navigation sidebar on this product's
	// pages. Meant for a single-page node like the changelog, whose sidebar
	// would hold one link to the page already being read. Defaults to true.
	Sidebar *bool         `yaml:"sidebar,omitempty"`
	Version string        `yaml:"-"` // Set internally from NavigationConfig.Version
	Groups  []GroupConfig `yaml:"groups"`
}

// HomepageConfig describes the generated landing page. The copy lives here so
// adding a product never means editing the page itself.
type HomepageConfig struct {
	Path    string `yaml:"path"`
	Eyebrow string `yaml:"eyebrow"`
	Title   string `yaml:"title"`
	Lede    string `yaml:"lede"`
	// Fonts is a stylesheet the landing page loads for itself, rather than
	// putting the request in the global stylesheet where every page pays for it.
	// Unset when the fonts are self-hosted via the design-system tokens.
	Fonts    string         `yaml:"fonts,omitempty"`
	Families []FamilyConfig `yaml:"families"`
	// Also is the secondary link row under the catalog.
	Also []NavLink  `yaml:"also,omitempty"`
	CTA  *CTAConfig `yaml:"cta,omitempty"`
}

type FamilyConfig struct {
	Family      string `yaml:"family"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	// Variant selects the card treatment. Empty is the standard product
	// card; "tools" is the lighter strip used for cross-product services
	// and clients (smaller cards, title and body only).
	Variant string `yaml:"variant,omitempty"`
	// Cards lists entries that are not products: tools, and announced
	// products whose documentation does not exist yet.
	Cards []ExtraCard `yaml:"cards,omitempty"`
}

type ExtraCard struct {
	Title       string `yaml:"title"`
	Href        string `yaml:"href,omitempty"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon,omitempty"`
	Tag         string `yaml:"tag,omitempty"`
	// Meta is the card's machine-set footer line, matching the product cards.
	Meta string `yaml:"meta,omitempty"`
	// Soon renders the card as announced-but-undocumented: dashed, a
	// "Coming soon" pill beside the tag, and not a link even if an href is
	// set. The announcement gates presence on the page; documentation gates
	// clickability.
	Soon bool `yaml:"soon,omitempty"`
}

// ProductCTA is a product's navbar call to action.
type ProductCTA struct {
	Label string `yaml:"label"`
	Href  string `yaml:"href"`
}

type CTAConfig struct {
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
	Label string `yaml:"label"`
	Href  string `yaml:"href"`
}

type TabConfig struct {
	Tab     string        `yaml:"tab"`
	Icon    string        `yaml:"icon,omitempty"`
	Version string        `yaml:"-"` // Set internally from NavigationConfig.Version
	Groups  []GroupConfig `yaml:"groups"`
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
	Description  string             `json:"description,omitempty"`
	Colors       Colors             `json:"colors"`
	Favicon      string             `json:"favicon"`
	Banner       *Banner            `json:"banner,omitempty"`
	Contextual   *Contextual        `json:"contextual,omitempty"`
	Logo         *Logo              `json:"logo,omitempty"`
	Thumbnails   *Thumbnails        `json:"thumbnails,omitempty"`
	Fonts        *Fonts             `json:"fonts,omitempty"`
	SEO          *SEO               `json:"seo,omitempty"`
	Search       *Search            `json:"search,omitempty"`
	Errors       *Errors            `json:"errors,omitempty"`
	Navbar       *Navbar            `json:"navbar,omitempty"`
	Footer       *Footer            `json:"footer,omitempty"`
	Integrations *Integrations      `json:"integrations,omitempty"`
	Redirects    []Redirect         `json:"redirects,omitempty"`
	Navigation   MintlifyNavigation `json:"navigation"`
}

type MintlifyNavigation struct {
	Tabs     []MintlifyTab     `json:"tabs,omitempty"`
	Products []MintlifyProduct `json:"products,omitempty"`
	Global   *GlobalNav        `json:"global,omitempty"`
}

type MintlifyProduct struct {
	// Family, Tag, Meta, Homepage and CTA are generator-only and never
	// reach docs.json.
	Family      string            `json:"-"`
	Tag         string            `json:"-"`
	Meta        string            `json:"-"`
	Homepage    *bool             `json:"-"`
	CTA         *ProductCTA       `json:"-"`
	Sidebar     *bool             `json:"-"`
	Product     string            `json:"product"`
	Name        string            `json:"name,omitempty"`
	Icon        string            `json:"icon,omitempty"`
	Color       string            `json:"color,omitempty"`
	Description string            `json:"description,omitempty"`
	Hidden      bool              `json:"hidden,omitempty"`
	Versions    []MintlifyVersion `json:"versions,omitempty"`
	Groups      []MintlifyGroup   `json:"groups,omitempty"`
}

type MintlifyTab struct {
	Tab      string            `json:"tab"`
	Icon     string            `json:"icon,omitempty"`
	Versions []MintlifyVersion `json:"versions,omitempty"`
	Groups   []MintlifyGroup   `json:"groups,omitempty"`
}

type MintlifyVersion struct {
	Version string          `json:"version"`
	Groups  []MintlifyGroup `json:"groups"`
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

	// Process integrations with defaults
	processedIntegrations := processIntegrations(mergedConfig.Integrations)

	// Generate Mintlify config
	mintlifyConfig := MintlifyConfig{
		Schema:       mergedConfig.Schema,
		Theme:        mergedConfig.Theme,
		Name:         mergedConfig.Name,
		Description:  mergedConfig.Description,
		Colors:       mergedConfig.Colors,
		Favicon:      mergedConfig.Favicon,
		Banner:       mergedConfig.Banner,
		Contextual:   mergedConfig.Contextual,
		Logo:         mergedConfig.Logo,
		Thumbnails:   mergedConfig.Thumbnails,
		Fonts:        mergedConfig.Fonts,
		SEO:          mergedConfig.SEO,
		Search:       mergedConfig.Search,
		Errors:       mergedConfig.Errors,
		Navbar:       mergedConfig.Navbar,
		Footer:       mergedConfig.Footer,
		Integrations: processedIntegrations,
		Redirects:    mergedConfig.Redirects,
		Navigation: MintlifyNavigation{
			Global: mergedConfig.Navigation.Global,
		},
	}

	// Process navigation tabs
	processedVersionedTabs := make(map[string]bool)

	// First, process versioned tabs
	for tabName, versions := range mergedConfig.VersionsMap {
		tab := MintlifyTab{
			Tab: tabName,
		}

		// Get icon from first version
		if len(versions) > 0 {
			tab.Icon = versions[0].Icon
		}

		// Process each version
		for _, versionConfig := range versions {
			version := MintlifyVersion{
				Version: versionConfig.Version,
			}

			for _, groupConfig := range versionConfig.Groups {
				group := MintlifyGroup{
					Group: groupConfig.Group,
				}

				// Only process explicitly defined pages
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

				version.Groups = append(version.Groups, group)
			}

			tab.Versions = append(tab.Versions, version)
		}

		mintlifyConfig.Navigation.Tabs = append(mintlifyConfig.Navigation.Tabs, tab)
		processedVersionedTabs[tabName] = true
	}

	// Then, process non-versioned tabs
	for _, tabConfig := range mergedConfig.Navigation.Tabs {
		// Skip if already processed as versioned tab
		if processedVersionedTabs[tabConfig.Tab] {
			continue
		}

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

	// Process navigation products. Products and tabs are mutually exclusive in
	// Mintlify and mergeConfigs already refused a mix, so at most one of these
	// two sections produces output.
	for _, key := range mergedConfig.ProductOrder {
		versions := mergedConfig.ProductVersionsMap[key]

		var product MintlifyProduct

		if len(versions) > 0 {
			// A versioned product: its display fields come from the first
			// version that declared it, since later versions repeat them.
			first := versions[0]
			product = MintlifyProduct{
				Family:      first.Family,
				Tag:         first.Tag,
				Meta:        first.Meta,
				Homepage:    first.Homepage,
				CTA:         first.CTA,
				Sidebar:     first.Sidebar,
				Product:     first.Product,
				Name:        first.Name,
				Icon:        first.Icon,
				Color:       first.Color,
				Description: first.Description,
				Hidden:      first.Hidden,
			}

			for _, versionConfig := range versions {
				product.Versions = append(product.Versions, MintlifyVersion{
					Version: versionConfig.Version,
					Groups:  buildGroups(versionConfig.Groups),
				})
			}
		} else {
			// An unversioned product: find its single declaration.
			var found bool
			for _, candidate := range mergedConfig.Navigation.Products {
				if candidate.Product != key {
					continue
				}
				product = MintlifyProduct{
					Family:      candidate.Family,
					Tag:         candidate.Tag,
					Meta:        candidate.Meta,
					Homepage:    candidate.Homepage,
					CTA:         candidate.CTA,
					Sidebar:     candidate.Sidebar,
					Product:     candidate.Product,
					Name:        candidate.Name,
					Icon:        candidate.Icon,
					Color:       candidate.Color,
					Description: candidate.Description,
					Hidden:      candidate.Hidden,
					Groups:      buildGroups(candidate.Groups),
				}
				found = true
				break
			}
			if !found {
				continue
			}
		}

		mintlifyConfig.Navigation.Products = append(mintlifyConfig.Navigation.Products, product)
	}

	// Write the generated landing page. Doing it here rather than in a separate
	// tool keeps it in step with the products that were just built, and means
	// CI's `git diff --exit-code` catches a homepage that has gone stale.
	if mergedConfig.Homepage != nil && mergedConfig.Homepage.Path != "" {
		// Resolved against the first config file rather than the working
		// directory: the container runs from the repo root, the local target
		// runs from tools/docs-gen, and both pass the same relative path.
		homepage := *mergedConfig.Homepage
		homepage.Path = filepath.Join(filepath.Dir(configPaths[0]), homepage.Path)

		if err := writeHomepage(&homepage, mintlifyConfig.Navigation.Products); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing homepage: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", homepage.Path)
	}

	if mergedConfig.ProductNav != "" {
		path := filepath.Join(filepath.Dir(configPaths[0]), mergedConfig.ProductNav)
		var primary *NavPrimary
		if mergedConfig.Navbar != nil {
			primary = mergedConfig.Navbar.Primary
		}
		if err := writeProductNav(path, mintlifyConfig.Navigation.Products, primary); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing product nav: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	}

	if mergedConfig.ProductChromeCSS != "" {
		path := filepath.Join(filepath.Dir(configPaths[0]), mergedConfig.ProductChromeCSS)
		if err := writeProductChromeCSS(path, mintlifyConfig.Navigation.Products); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing product chrome css: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", path)
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

// buildGroups turns configured groups into Mintlify groups, skipping any group
// with no explicitly listed pages. Folder scanning is deliberately not done
// here: every page must be named in a nav file so docs-validate can hold the
// yaml and the content directories to each other.
func buildGroups(groupConfigs []GroupConfig) []MintlifyGroup {
	var groups []MintlifyGroup

	for _, groupConfig := range groupConfigs {
		if len(groupConfig.Pages) == 0 {
			continue
		}

		pages, err := processManualPages(groupConfig.Pages, groupConfig.Folder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing manual pages for group %s: %v\n", groupConfig.Group, err)
			continue
		}

		groups = append(groups, MintlifyGroup{Group: groupConfig.Group, Pages: pages})
	}

	return groups
}

// firstPagePath finds the first page a product links to, used as its card's
// destination. Pages are an interface because a group can hold plain strings or
// nested sub-groups.
func firstPagePath(pages interface{}) string {
	switch value := pages.(type) {
	case string:
		return value
	case []string:
		if len(value) > 0 {
			return value[0]
		}
	case []interface{}:
		for _, item := range value {
			if found := firstPagePath(item); found != "" {
				return found
			}
		}
	case MintlifyGroup:
		// What processManualPages actually puts in the slice for a sub-group.
		return firstPagePath(value.Pages)
	case map[string]interface{}:
		// Not produced by processManualPages today, but cheap to keep in case
		// pages ever arrive already decoded from JSON.
		if nested, ok := value["pages"]; ok {
			return firstPagePath(nested)
		}
	}
	return ""
}

// productHref is where a product's homepage card points: its first page.
func productHref(product MintlifyProduct) string {
	groups := product.Groups
	if len(product.Versions) > 0 {
		groups = product.Versions[0].Groups
	}
	for _, group := range groups {
		if page := firstPagePath(group.Pages); page != "" {
			// A group with `folder: "/"` yields a path that already starts with
			// a slash. Prefixing another one gives "//page", which browsers
			// read as protocol-relative and resolve against a different host.
			return "/" + strings.TrimPrefix(page, "/")
		}
	}
	return ""
}

// writeHomepage renders the landing page from the homepage config and the
// products that were built. It is generated rather than hand-written so that
// adding, renaming or retiring a product updates the homepage on its own; CI
// catches a stale copy because the committed file stops matching.
func writeHomepage(homepage *HomepageConfig, products []MintlifyProduct) error {
	// Families declared in the config, so an unknown one can be rejected rather
	// than quietly dropping the product's card.
	known := make(map[string]bool, len(homepage.Families))
	for _, family := range homepage.Families {
		known[family.Family] = true
	}

	byFamily := make(map[string][]MintlifyProduct)
	for _, product := range products {
		// Hidden products are staged rather than shown, so they have no card
		// and need no family.
		if product.Hidden {
			continue
		}

		// A product can opt out of the catalog while staying in the product
		// switcher; the homepage reaches it another way (the Also row).
		if product.Homepage != nil && !*product.Homepage {
			continue
		}

		// Silently omitting these was the failure mode worth guarding: a
		// mistyped family produced a product that built, validated and passed
		// every check while simply having no card on the homepage.
		if product.Family == "" {
			return fmt.Errorf("product %q has no `family`, so it would not appear on the homepage; set one or mark the product hidden", product.Product)
		}
		if !known[product.Family] {
			return fmt.Errorf("product %q has family %q, which is not declared under homepage.families in the base config", product.Product, product.Family)
		}

		byFamily[product.Family] = append(byFamily[product.Family], product)
	}

	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: %q\n", homepage.Title)
	fmt.Fprintf(&b, "description: %q\n", homepage.Lede)
	b.WriteString("mode: \"custom\"\n")
	b.WriteString("---\n\n")
	b.WriteString("{/* Generated by tools/docs-gen from common.yaml and the product nav files. */}\n")
	b.WriteString("{/* Do not edit by hand: run `make docs.json` instead. */}\n")
	b.WriteString("{/* Styles live in home.css; the search button is wired by home.js. */}\n\n")

	// The landing page is the only page using these faces, so it loads them
	// itself. Unset when the fonts already arrive through the design-system
	// token stylesheet.
	if homepage.Fonts != "" {
		fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=%q />\n\n", homepage.Fonts)
	}

	b.WriteString("<div id=\"docs-home\">\n")

	b.WriteString("  <section className=\"home-hero\">\n")
	fmt.Fprintf(&b, "    <h1>%s</h1>\n", mdxText(homepage.Title))
	fmt.Fprintf(&b, "    <p className=\"home-promise\">%s</p>\n", mdxText(homepage.Lede))
	b.WriteString("    <button className=\"home-search\" id=\"home-search\" aria-label=\"Search the documentation\">\n")
	b.WriteString("      <span className=\"home-search-icon\" aria-hidden=\"true\">\n")
	// The ring is a path, not a <circle>: Mintlify's MDX sanitizer drops
	// circle elements but keeps paths.
	b.WriteString("        <svg width=\"16\" height=\"16\" viewBox=\"0 0 16 16\" fill=\"none\" stroke=\"currentColor\" strokeWidth=\"1.6\">\n")
	b.WriteString("          <path d=\"M11.5 7a4.5 4.5 0 1 1-9 0 4.5 4.5 0 0 1 9 0Z\" />\n")
	b.WriteString("          <path d=\"M10.5 10.5 14 14\" strokeLinecap=\"round\" />\n")
	b.WriteString("        </svg>\n")
	b.WriteString("      </span>\n")
	b.WriteString("      <span className=\"home-search-label\">Search the documentation</span>\n")
	b.WriteString("      <span className=\"home-kbd\">⌘K</span>\n")
	b.WriteString("    </button>\n")
	b.WriteString("  </section>\n\n")

	b.WriteString("  <div className=\"home-catalog\">\n")

	for _, family := range homepage.Families {
		cards := byFamily[family.Family]
		if len(cards) == 0 && len(family.Cards) == 0 {
			continue
		}

		gridClass := "home-grid"
		if family.Variant == "tools" {
			gridClass = "home-grid home-grid--tools"
		}

		b.WriteString("    <section className=\"home-domain\">\n")
		b.WriteString("      <div className=\"home-domain-head\">\n")
		fmt.Fprintf(&b, "        <h2>%s</h2>\n", mdxText(family.Title))
		if family.Description != "" {
			fmt.Fprintf(&b, "        <p>%s</p>\n", mdxText(family.Description))
		}
		b.WriteString("      </div>\n")
		fmt.Fprintf(&b, "      <div className=%q>\n", gridClass)

		for _, product := range cards {
			writeCard(&b, homeCard{
				Title:       product.Product,
				Href:        productHref(product),
				Description: product.Description,
				Tag:         product.Tag,
				Meta:        productMeta(product),
				Tool:        family.Variant == "tools",
			})
		}
		for _, card := range family.Cards {
			writeCard(&b, homeCard{
				Title:       card.Title,
				Href:        card.Href,
				Description: card.Description,
				Tag:         card.Tag,
				Meta:        card.Meta,
				Soon:        card.Soon,
				Tool:        family.Variant == "tools",
			})
		}

		b.WriteString("      </div>\n")
		b.WriteString("    </section>\n\n")
	}

	if len(homepage.Also) > 0 {
		b.WriteString("    <div className=\"home-aside\">\n")
		b.WriteString("      <span className=\"home-lbl\">Also</span>\n")
		for _, link := range homepage.Also {
			fmt.Fprintf(&b, "      <a href=%q>%s</a>\n", link.Href, mdxText(link.Label))
		}
		b.WriteString("    </div>\n")
	}

	b.WriteString("  </div>\n")
	b.WriteString("</div>\n")

	return os.WriteFile(homepage.Path, []byte(b.String()), 0o644)
}

// productMeta is a product card's footer line. A versioned product left
// without one states its newest version, so the line tracks releases without
// anyone editing it.
func productMeta(product MintlifyProduct) string {
	if product.Meta != "" {
		return product.Meta
	}
	if len(product.Versions) > 0 {
		return product.Versions[0].Version + " current"
	}
	return ""
}

// homeCard is one rendered card on the homepage.
type homeCard struct {
	Title       string
	Href        string
	Description string
	Tag         string
	Meta        string
	// Soon renders the announced-but-undocumented treatment: a div rather
	// than a link, dashed border, "Coming soon" pill.
	Soon bool
	// Tool renders the lighter tools-strip treatment: title and body only.
	Tool bool
}

// writeCard renders one homepage card in the design system's markup (see
// home.css for the classes).
func writeCard(b *strings.Builder, card homeCard) {
	classes := "home-card home-card--live"
	element := "a"
	if card.Soon {
		classes = "home-card home-card--soon"
		element = "div"
	}
	if card.Tool {
		classes += " home-card--tool"
	}

	fmt.Fprintf(b, "        <%s className=%q", element, classes)
	if !card.Soon && card.Href != "" {
		fmt.Fprintf(b, " href=%q", card.Href)
	}
	b.WriteString(">\n")

	// Tools cards carry only a title and body; the eyebrow and meta lines
	// belong to the product cards. A soon card always gets its eyebrow, since
	// the pill lives there, even before its tag is decided.
	if !card.Tool && (card.Tag != "" || card.Soon) {
		b.WriteString("          <div className=\"home-eyebrow\">")
		b.WriteString(mdxText(card.Tag))
		if card.Soon {
			if card.Tag != "" {
				b.WriteString(" ")
			}
			b.WriteString("<span className=\"home-pill\">Coming soon</span>")
		}
		b.WriteString("</div>\n")
	}
	fmt.Fprintf(b, "          <h3>%s</h3>\n", mdxText(card.Title))
	if card.Description != "" {
		fmt.Fprintf(b, "          <p>%s</p>\n", mdxText(card.Description))
	}
	if !card.Tool && !card.Soon && card.Meta != "" {
		fmt.Fprintf(b, "          <div className=\"home-meta\">%s</div>\n", mdxText(card.Meta))
	}
	fmt.Fprintf(b, "        </%s>\n", element)
}

// mdxText makes config text safe to place between JSX tags. In MDX a brace
// opens a JavaScript expression and an angle bracket opens a tag, so either
// in a description would break the page build rather than render.
var mdxTextReplacer = strings.NewReplacer(
	"{", "&#123;",
	"}", "&#125;",
	"<", "&lt;",
	">", "&gt;",
)

func mdxText(s string) string {
	return mdxTextReplacer.Replace(s)
}

// urlSegment is the first path segment of a product's pages, which is how the
// browser identifies which product is being read. It is derived from the pages
// themselves rather than configured separately, so the two cannot disagree.
func urlSegment(product MintlifyProduct) string {
	href := strings.TrimPrefix(productHref(product), "/")
	if href == "" {
		return ""
	}
	if slash := strings.Index(href, "/"); slash != -1 {
		return href[:slash]
	}
	return href
}

// writeProductNav emits what the browser needs to know about each product.
//
// Three things the navbar needs are not expressible in docs.json: which product
// is being read (for the search placeholder), whether it carries versions (so
// the version switcher is not offered where there is nothing to switch), and
// its call to action. Deriving all of it from the same config that builds the
// navigation means the script cannot fall out of step with the nav -- an
// earlier version kept a hand-written product list here, and a product missing
// from it silently lost its version switcher.
func writeProductNav(path string, products []MintlifyProduct, primary *NavPrimary) error {
	type cta struct {
		Label string `json:"label"`
		Href  string `json:"href"`
	}
	type entry struct {
		Name      string `json:"name"`
		Versioned bool   `json:"versioned"`
		CTA       *cta   `json:"cta,omitempty"`
	}

	nav := make(map[string]entry)
	for _, product := range products {
		segment := urlSegment(product)
		if segment == "" {
			if product.CTA != nil {
				fmt.Fprintf(os.Stderr, "warning: product %q has a cta but no pages to derive its url from; skipped\n", product.Product)
			}
			continue
		}

		item := entry{Name: product.Product, Versioned: len(product.Versions) > 0}
		if product.CTA != nil && product.CTA.Label != "" && product.CTA.Href != "" {
			item.CTA = &cta{Label: product.CTA.Label, Href: product.CTA.Href}
		}
		nav[segment] = item
	}

	payload, err := json.MarshalIndent(nav, "", "  ")
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("/*\n")
	b.WriteString(" * Generated by tools/docs-gen from the product navigation files.\n")
	b.WriteString(" * Do not edit by hand: run `make docs.json` instead.\n")
	b.WriteString(" *\n")
	b.WriteString(" * Read by script/contextual-chrome.js, keyed by the first path segment of\n")
	b.WriteString(" * each product's pages. A product with no `cta` block has no button.\n")
	b.WriteString(" */\n")
	b.WriteString("window.sideroProducts = ")
	b.Write(payload)
	b.WriteString(";\n")

	// The navbar.primary button Mintlify renders on every page. The script
	// finds it by this href, then relabels or hides it per product.
	if primary != nil && primary.Href != "" {
		defaultCTA, err := json.MarshalIndent(cta{Label: primary.Label, Href: primary.Href}, "", "  ")
		if err != nil {
			return err
		}
		b.WriteString("\nwindow.sideroNavbarCTA = ")
		b.Write(defaultCTA)
		b.WriteString(";\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// allPagePaths collects every page under a set of groups, walking sub-groups.
func allPagePaths(pages interface{}, out *[]string) {
	switch value := pages.(type) {
	case string:
		*out = append(*out, value)
	case []string:
		*out = append(*out, value...)
	case []interface{}:
		for _, item := range value {
			allPagePaths(item, out)
		}
	case MintlifyGroup:
		allPagePaths(value.Pages, out)
	case map[string]interface{}:
		if nested, ok := value["pages"]; ok {
			allPagePaths(nested, out)
		}
	}
}

// writeProductChromeCSS emits the rules that depend on where a product's pages
// actually live.
//
// Hiding the sidebar for the changelog needs its url, and typing that url into
// style.css meant the rule silently stopped matching if the page ever moved.
// Generating it from the same config that places the page keeps the two in step.
func writeProductChromeCSS(path string, products []MintlifyProduct) error {
	var b strings.Builder
	b.WriteString("/*\n")
	b.WriteString(" * Generated by tools/docs-gen from the product navigation files.\n")
	b.WriteString(" * Do not edit by hand: run `make docs.json` instead.\n")
	b.WriteString(" *\n")
	b.WriteString(" * Rules whose selectors depend on a product's page paths.\n")
	b.WriteString(" */\n")

	for _, product := range products {
		if product.Sidebar == nil || *product.Sidebar {
			continue
		}

		groups := product.Groups
		for _, version := range product.Versions {
			groups = append(groups, version.Groups...)
		}

		var pages []string
		for _, group := range groups {
			allPagePaths(group.Pages, &pages)
		}
		if len(pages) == 0 {
			continue
		}

		fmt.Fprintf(&b, "\n/* %s: sidebar suppressed. */\n", product.Product)
		for _, page := range pages {
			route := "/" + strings.TrimPrefix(page, "/")
			fmt.Fprintf(&b, "html[data-current-path=%q] #sidebar-content {\n  display: none;\n}\n", route)
			fmt.Fprintf(&b, "html[data-current-path=%q] #content-area {\n  width: 100%%;\n  max-width: 60rem;\n}\n", route)
		}
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// mergeConfigs reads and merges multiple YAML config files
func mergeConfigs(configPaths []string) (Config, error) {
	var mergedConfig Config
	var allTabs []TabConfig
	var allProducts []ProductConfig
	var productOrder []string
	versionsMap := make(map[string][]TabConfig)            // Map of tab name -> versions
	productVersionsMap := make(map[string][]ProductConfig) // Map of product key -> versions
	seenProduct := make(map[string]bool)                   // First-seen tracking for productOrder

	// noteProduct records a product key the first time it appears, so the
	// generator can emit products in declaration order rather than map order.
	noteProduct := func(key string) {
		if !seenProduct[key] {
			seenProduct[key] = true
			productOrder = append(productOrder, key)
		}
	}

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
			// Clear both to rebuild from every file, not just the first.
			mergedConfig.Navigation.Tabs = nil
			mergedConfig.Navigation.Products = nil
		}

		// Mintlify treats the keys under `navigation` as mutually exclusive, so
		// a single file mixing them could never produce valid output.
		if len(config.Navigation.Tabs) > 0 && len(config.Navigation.Products) > 0 {
			return Config{}, fmt.Errorf("%s declares both navigation.tabs and navigation.products; Mintlify allows only one", configPath)
		}

		// If this config has a version, group tabs by name for version support
		if config.Navigation.Version != "" {
			for _, tab := range config.Navigation.Tabs {
				// Store the version on the tab config for later use
				tab.Version = config.Navigation.Version
				versionsMap[tab.Tab] = append(versionsMap[tab.Tab], tab)
			}
		} else {
			// Collect tabs from all files in order (non-versioned)
			allTabs = append(allTabs, config.Navigation.Tabs...)
		}

		// Products follow the same shape as tabs: a file carrying a version
		// contributes one version to each product it names, and a file without
		// one contributes the product itself.
		if config.Navigation.Version != "" {
			for _, product := range config.Navigation.Products {
				product.Version = config.Navigation.Version
				noteProduct(product.Product)
				productVersionsMap[product.Product] = append(productVersionsMap[product.Product], product)
			}
		} else {
			for _, product := range config.Navigation.Products {
				noteProduct(product.Product)
				allProducts = append(allProducts, product)
			}
		}
	}

	if len(allTabs)+len(versionsMap) > 0 && len(allProducts)+len(productVersionsMap) > 0 {
		return Config{}, fmt.Errorf("config files mix navigation.tabs and navigation.products; Mintlify allows only one")
	}

	// A product declared in both a versioned and an unversioned file used to
	// build fine and silently drop the unversioned groups, because the build
	// path prefers the versioned entry. Refuse it instead.
	for _, product := range allProducts {
		if _, versioned := productVersionsMap[product.Product]; versioned {
			return Config{}, fmt.Errorf("product %q is declared both with and without a navigation.version; its unversioned groups would be dropped silently", product.Product)
		}
	}

	// Set merged tabs
	mergedConfig.Navigation.Tabs = allTabs
	mergedConfig.VersionsMap = versionsMap

	mergedConfig.Navigation.Products = allProducts
	mergedConfig.ProductVersionsMap = productVersionsMap
	mergedConfig.ProductOrder = productOrder

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

// processIntegrations processes integrations and sets default values
func processIntegrations(integrations *Integrations) *Integrations {
	if integrations == nil {
		return nil
	}

	// Create a copy to avoid modifying the original
	processed := &Integrations{
		GA4:     integrations.GA4,
		Posthog: integrations.Posthog,
	}

	return processed
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
