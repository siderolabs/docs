package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Product defines a GitHub repo and how it appears in the changelog.
type Product struct {
	Repo       string
	Label      string
	MDXTag     string
	MinVersion []int // exclusive lower bound; nil means no filter
}

// Release is the subset of the GitHub releases API response we need.
type Release struct {
	TagName     string `json:"tag_name"`
	Prerelease  bool   `json:"prerelease"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// updateBlock holds a rendered <Update> block and its publish timestamp for sorting.
type updateBlock struct {
	content     string
	publishedAt string
}

var (
	prereleaseTagRe  = regexp.MustCompile(`(?i)-(alpha|beta|rc)`)
	ghAlertRe        = regexp.MustCompile(`(?i)^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*$`)
	blockLineStripRe = regexp.MustCompile(`^>\s?`)
	headingPrefixRe  = regexp.MustCompile(`^#+\s*`)
	anchorTagRe      = regexp.MustCompile(`(?i)^<a\s+name=`)
	mdxHeadingLinkRe = regexp.MustCompile(`^## \[`)
	versionSplitRe   = regexp.MustCompile(`[.\-]`)
)

var skipHeadingNames = map[string]bool{
	"Contributors":       true,
	"Changes":            true,
	"Dependency Changes": true,
	"Images":             true,
}

var alertTagMap = map[string]string{
	"NOTE":      "Note",
	"TIP":       "Tip",
	"IMPORTANT": "Note",
	"WARNING":   "Warning",
	"CAUTION":   "Warning",
}

var products = []Product{
	{Repo: "talos", Label: "Talos Linux", MDXTag: "Talos", MinVersion: []int{1, 6, 0}},
	{Repo: "omni", Label: "Omni", MDXTag: "Omni", MinVersion: []int{1, 1, 0}},
	{Repo: "image-factory", Label: "Image Factory", MDXTag: "Image Factory"},
	{Repo: "discovery-service", Label: "Discovery Service", MDXTag: "Discovery Service"},
}

const frontmatter = `---
title: "Product Updates"
description: "Product updates and announcements"
rss: true
---

`

func main() {
	output := flag.String("output", "public/changelog.mdx", "Output file path")
	flag.Parse()

	// GITHUB_TOKEN is read from the environment. Set it to avoid the 60 req/hr
	// unauthenticated rate limit: GITHUB_TOKEN=<token> make changelog
	token := os.Getenv("GITHUB_TOKEN")

	fmt.Fprintln(os.Stderr, "Generating changelog for all products...")

	type fetchResult struct {
		blocks []updateBlock
		err    error
		label  string
	}

	results := make(chan fetchResult, len(products))

	var wg sync.WaitGroup
	for _, p := range products {
		wg.Add(1)
		go func(p Product) {
			defer wg.Done()
			fmt.Fprintf(os.Stderr, "  Fetching %s releases from GitHub...\n", p.Label)
			blocks, err := fetchReleases(p, token)
			results <- fetchResult{blocks: blocks, err: err, label: p.Label}
		}(p)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var allBlocks []updateBlock
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching %s: %v\n", r.label, r.err)
			os.Exit(1)
		}
		allBlocks = append(allBlocks, r.blocks...)
	}

	fmt.Fprintln(os.Stderr, "  All products fetched.")
	fmt.Fprintln(os.Stderr, "  Merging and sorting...")

	sort.Slice(allBlocks, func(i, j int) bool {
		return allBlocks[i].publishedAt > allBlocks[j].publishedAt
	})

	var blockStrings []string
	for _, b := range allBlocks {
		blockStrings = append(blockStrings, b.content)
	}

	content := frontmatter + strings.Join(blockStrings, "\n\n")

	if err := os.WriteFile(*output, []byte(content), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Done! changelog.mdx written to: %s\n", *output)
}

// fetchReleases paginates the GitHub releases API and returns processed update blocks.
func fetchReleases(p Product, token string) ([]updateBlock, error) {
	var releases []Release

	for page := 1; ; page++ {
		url := fmt.Sprintf(
			"https://api.github.com/repos/siderolabs/%s/releases?per_page=100&page=%d",
			p.Repo, page,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "changelog-gen")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, body)
		}

		var batch []Release
		if err := json.Unmarshal(body, &batch); err != nil {
			return nil, fmt.Errorf("parsing releases page %d: %w", page, err)
		}

		if len(batch) == 0 {
			break
		}

		releases = append(releases, batch...)

		if len(batch) < 100 {
			break
		}
	}

	var blocks []updateBlock
	for _, r := range releases {
		if r.Prerelease || isPrereleaseTag(r.TagName) {
			continue
		}

		if p.MinVersion != nil && versionGreater(p.MinVersion, parseVersion(r.TagName)) {
			continue
		}

		cleaned := cleanBody(r.Body)
		indented := indentBody(cleaned)

		block := fmt.Sprintf("<Update label=%q tags={[%q]}>\n", r.TagName, p.MDXTag)
		block += fmt.Sprintf("  [Release notes →](%s)\n\n", r.HTMLURL)
		block += indented + "\n"
		block += "</Update>"

		blocks = append(blocks, updateBlock{
			content:     block,
			publishedAt: r.PublishedAt,
		})
	}

	return blocks, nil
}

// isPrereleaseTag returns true if the tag contains alpha/beta/rc suffixes,
// regardless of the GitHub API prerelease flag.
func isPrereleaseTag(tag string) bool {
	return prereleaseTagRe.MatchString(tag)
}

// convertGHAlerts converts GitHub alert blockquotes to Mintlify callout components.
//
// GitHub format:
//
//	> [!NOTE]
//	> Some text
//
// Mintlify format:
//
//	<Note>
//	Some text
//	</Note>
func convertGHAlerts(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		m := ghAlertRe.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m != nil {
			tag := alertTagMap[strings.ToUpper(m[1])]
			var blockLines []string
			i++
			for i < len(lines) && strings.HasPrefix(lines[i], ">") {
				blockLines = append(blockLines, blockLineStripRe.ReplaceAllString(lines[i], ""))
				i++
			}
			out = append(out, "<"+tag+">")
			out = append(out, blockLines...)
			out = append(out, "</"+tag+">")
		} else {
			out = append(out, lines[i])
			i++
		}
	}
	return strings.Join(out, "\n")
}

// isSkipHeading returns true if the heading line belongs to a section we want to drop.
func isSkipHeading(line string) bool {
	if !strings.HasPrefix(line, "### ") && !strings.HasPrefix(line, "## ") {
		return false
	}
	heading := strings.TrimSpace(headingPrefixRe.ReplaceAllString(line, ""))
	for name := range skipHeadingNames {
		if strings.HasPrefix(heading, name) {
			return true
		}
	}
	return strings.HasPrefix(heading, "Changes since") || strings.HasPrefix(heading, "Changes from")
}

// cleanBody applies all cleanup rules to a release body before embedding it in MDX.
func cleanBody(body string) string {
	// 1. Convert GitHub alert blocks to Mintlify components.
	body = convertGHAlerts(body)

	lines := strings.Split(body, "\n")
	var filtered []string
	skipBlock := false
	inDetails := false

	// 2. Strip <details> blocks, standalone <p>/<img> tags, and skipped heading sections.
	for _, line := range lines {
		if strings.Contains(line, "<details") {
			inDetails = true
			continue
		}
		if strings.Contains(line, "</details>") {
			inDetails = false
			continue
		}
		if inDetails {
			continue
		}

		stripped := strings.TrimSpace(line)
		if stripped == "<p>" || stripped == "</p>" {
			continue
		}
		if strings.HasPrefix(stripped, "<img ") {
			continue
		}

		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ") {
			skipBlock = isSkipHeading(line)
		}
		if !skipBlock {
			filtered = append(filtered, line)
		}
	}

	// 3. Strip boilerplate lines.
	var cleaned []string
	for _, line := range filtered {
		// ## [Product X.Y.Z](url) title lines break MDX — already linked via "Release notes →"
		if mdxHeadingLinkRe.MatchString(line) {
			continue
		}
		// <a name="..."></a> anchor tags break MDX.
		if anchorTagRe.MatchString(strings.TrimSpace(line)) {
			continue
		}
		if strings.HasPrefix(line, "Welcome to the") {
			continue
		}
		if strings.Contains(line, "Please try out the release binaries") {
			continue
		}
		if strings.HasPrefix(line, "Previous release can be found at") {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "*This is a pre-release") {
			continue
		}
		// Bare issue tracker URLs.
		if strings.Contains(line, "github.com/siderolabs/") &&
			strings.Contains(line, "/issues") &&
			strings.HasPrefix(strings.TrimSpace(line), "http") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	// 4. Strip leading/trailing blank lines.
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[0]) == "" {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	// 5. Collapse consecutive blank lines to one.
	var result []string
	blankRun := 0
	for _, line := range cleaned {
		if strings.TrimSpace(line) == "" {
			blankRun++
			if blankRun == 1 {
				result = append(result, line)
			}
		} else {
			blankRun = 0
			result = append(result, line)
		}
	}

	text := strings.Join(result, "\n")

	// 6. Escape <= — MDX/JSX interprets it as an opening tag.
	text = strings.ReplaceAll(text, "<=", "&lt;=")

	return text
}

// indentBody indents non-blank lines with two spaces for readability inside <Update> blocks.
func indentBody(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = "  " + line
		} else {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// parseVersion extracts a semver tuple from a tag name (e.g. "v1.2.3") for sorting.
func parseVersion(tag string) []int {
	v := strings.TrimPrefix(tag, "v")
	parts := versionSplitRe.Split(v, -1)
	result := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}

// versionGreater returns true if version a sorts higher than version b (descending).
func versionGreater(a, b []int) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return len(a) > len(b)
}
