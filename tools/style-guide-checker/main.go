// Command style-guide-checker lints SideroLabs documentation (.mdx) against the
// mechanically-checkable parts of the documentation style guide. It is the
// repo's single style tool.
//
// Checks:
//   - Titles   : the page title should be title case.
//   - Headings : sentence case; no stacked headings; no skipped levels; no body
//     H1; blank lines around headings.
//   - Code     : fenced blocks need a language hint; no shell prompts ($) in
//     copy-pasteable commands; avoid `sed` (BSD/GNU differ).
//   - Links    : no non-descriptive link text ("click here", "here", ...).
//   - Images   : referenced image filenames must be kebab-case.
//
// The sentence/title-case checks consult exceptions.txt for proper nouns that
// may stay capitalized; ALL-CAPS acronyms and CamelCase names are allowed
// automatically.
package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// exceptionsRaw is the editable list of proper nouns / product names that are
// allowed to keep their capitalization inside a heading or title.
//
//go:embed exceptions.txt
var exceptionsRaw string

// exceptionSet holds the lowercased words from exceptions.txt for fast lookup.
var exceptionSet = buildExceptionSet(exceptionsRaw)

func buildExceptionSet(raw string) map[string]bool {
	set := map[string]bool{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[strings.ToLower(line)] = true
	}
	return set
}

// smallWords stay lowercase in title case unless they are the first word.
var smallWords = map[string]bool{
	"a": true, "an": true, "the": true, "and": true, "but": true, "or": true,
	"nor": true, "for": true, "of": true, "to": true, "in": true, "on": true,
	"at": true, "by": true, "with": true, "as": true, "from": true, "into": true,
	"per": true, "via": true, "vs": true, "up": true, "out": true, "off": true,
}

// Level is the severity of a finding. Only Error fails the run by default;
// warnings fail only with -strict.
type Level int

const (
	Warning Level = iota
	ErrorLevel
)

func (l Level) String() string {
	if l == ErrorLevel {
		return "error"
	}
	return "warning"
}

// Finding is a single style-guide violation.
type Finding struct {
	File    string
	Line    int
	Level   Level
	Rule    string
	Message string
}

// --- Rule configuration -----------------------------------------------------

// Languages we treat as shell so we can look for prompts and non-portable
// commands. The empty string covers un-hinted blocks, which in these docs are
// almost always shell commands.
var shellLangs = map[string]bool{
	"":              true,
	"bash":          true,
	"sh":            true,
	"shell":         true,
	"zsh":           true,
	"console":       true,
	"shell-session": true,
	"sh-session":    true,
}

// Link text (after stripping emphasis/whitespace, lowercased) that gives the
// reader no idea where the link goes.
var badLinkText = map[string]bool{
	"click here": true,
	"here":       true,
	"read more":  true,
	"learn more": true,
	"click":      true,
	"this link":  true,
	"read here":  true,
	"see here":   true,
}

var imageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".svg":  true,
	".gif":  true,
	".webp": true,
	".avif": true,
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s+(.*\S)\s*$`)
	// Markdown links and images: leading "!" (group 1) marks an image.
	linkRe = regexp.MustCompile(`(!?)\[([^\]]*)\]\(([^)]+)\)`)
	// src="..." on any line, covering both same-line and multi-line <img ...>.
	srcRe = regexp.MustCompile(`src\s*=\s*["']([^"']+)["']`)
	// A leading "$ " shell prompt.
	promptRe = regexp.MustCompile(`^\s*\$\s+\S`)
	// A sed invocation at the start of a command or after a pipe/;/&/(.
	sedRe = regexp.MustCompile(`(^|[\s;|&(])sed\s`)
	// kebab-case: lowercase words joined by single hyphens.
	kebabRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	// Strips markdown emphasis so `[**click here**]` compares as "click here".
	emphasisRe = regexp.MustCompile("[*_`]")
	// A plain Capitalized word: first letter upper, rest lower (e.g. "Started").
	// Deliberately excludes acronyms (SAML) and CamelCase names (KubeSpan).
	plainCapRe = regexp.MustCompile(`^[A-Z][a-z]+$`)
	// A plain lowercase word (e.g. "started").
	plainLowerRe = regexp.MustCompile(`^[a-z]+$`)
	// An inline code span: `code`.
	inlineCodeRe = regexp.MustCompile("`[^`]*`")
	// A markdown link, reduced to its visible text.
	mdLinkTextRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

// --- Main -------------------------------------------------------------------

func main() {
	format := flag.String("format", "text", "output format: text or github")
	strict := flag.Bool("strict", false, "exit non-zero on warnings too, not just errors")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: style-guide-checker [flags] [paths...]\n\n")
		fmt.Fprintf(os.Stderr, "Lints .mdx documentation against the SideroLabs style guide.\n")
		fmt.Fprintf(os.Stderr, "Paths may be files or directories; directories are searched for .mdx files.\n")
		fmt.Fprintf(os.Stderr, "Defaults to \"public\" when no paths are given.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"public"}
	}

	files, err := collectFiles(paths)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var all []Finding
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v\n", f, err)
			os.Exit(2)
		}
		all = append(all, lint(f, string(data))...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].File != all[j].File {
			return all[i].File < all[j].File
		}
		return all[i].Line < all[j].Line
	})

	warnings, errors := report(all, *format)

	fmt.Fprintf(os.Stderr, "\nChecked %d file(s): %d error(s), %d warning(s)\n",
		len(files), errors, warnings)

	if errors > 0 || (*strict && warnings > 0) {
		os.Exit(1)
	}
}

// collectFiles expands the given paths into a sorted list of .mdx files.
func collectFiles(paths []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			files = append(files, p)
		}
	}
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("cannot access %s: %w", p, err)
		}
		if !info.IsDir() {
			if strings.HasSuffix(p, ".mdx") {
				add(p)
			}
			continue
		}
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".mdx") {
				add(path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking %s: %w", p, err)
		}
	}
	sort.Strings(files)
	return files, nil
}

// --- Linting ----------------------------------------------------------------

// lint runs every rule over a single file's contents.
func lint(file, content string) []Finding {
	lines := strings.Split(content, "\n")

	var findings []Finding
	add := func(line int, level Level, rule, msg string) {
		findings = append(findings, Finding{file, line, level, rule, msg})
	}

	// Skip a leading YAML frontmatter block so its "---" and fields are not
	// mistaken for content. Only treat it as frontmatter if it is actually
	// closed; an unterminated "---" is left to be parsed as ordinary content.
	start := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				start = i + 1
				break
			}
		}
		if start > 0 {
			// Check the page title (title case) inside the closed frontmatter.
			for i := 1; i < start-1; i++ {
				title, ok := frontmatterTitle(lines[i])
				if !ok {
					continue
				}
				if word, bad := titleCaseIssue(title); bad {
					add(i+1, Warning, "title/title-case",
						fmt.Sprintf("page title should be title case; capitalize %q (or add it to exceptions.txt if it's a proper noun)", word))
				}
				break // a page has a single title
			}
		}
	}

	// Heading-structure state. prevLevel starts at 1 because the frontmatter
	// title acts as the page's H1, so the first body heading should be H2.
	prevLevel := 1
	// pendingHeadingLine is the 1-based line of a heading that has not yet been
	// followed by body content; a second heading before any content is "stacked".
	pendingHeadingLine := 0

	// Code-fence state.
	inFence := false
	fenceMarker := ""
	fenceLang := ""

	for i := start; i < len(lines); i++ {
		lineNo := i + 1
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Fence toggling.
		if marker := fenceMarkerOf(trimmed); marker != "" {
			if !inFence {
				inFence = true
				fenceMarker = marker
				fenceLang = fenceLangOf(trimmed)
				if fenceLang == "" {
					add(lineNo, Warning, "code/no-language",
						"code block has no language hint; add one (e.g. ```bash, ```json)")
				}
				// A code block counts as content after a heading.
				pendingHeadingLine = 0
				continue
			}
			// A closing fence must use the same character and be at least as
			// long as the opening one (CommonMark), so a longer fence wrapping
			// a shorter one is not closed prematurely.
			if marker[0] == fenceMarker[0] && len(marker) >= len(fenceMarker) {
				inFence = false
				fenceMarker = ""
				fenceLang = ""
				continue
			}
		}

		if inFence {
			checkCodeLine(add, lineNo, line, fenceLang)
			continue
		}

		// --- Non-code line ---

		if m := headingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			text := m[2]

			if pendingHeadingLine != 0 {
				add(lineNo, Warning, "headings/stacked",
					fmt.Sprintf("heading directly follows the heading on line %d; add at least one sentence between them", pendingHeadingLine))
			}
			if level > prevLevel+1 {
				add(lineNo, Warning, "headings/skipped-level",
					fmt.Sprintf("heading jumps from level %d to level %d; step down one level at a time", prevLevel, level))
			}
			if level == 1 {
				add(lineNo, Warning, "headings/avoid-h1",
					"avoid H1 headings in the body; the page title is the H1, so start at ## and go deeper")
			}
			// A heading needs a blank line before it (unless it is the first line
			// of content) and after it.
			if i > start && strings.TrimSpace(lines[i-1]) != "" {
				add(lineNo, Warning, "headings/blank-line",
					"add a blank line before this heading")
			}
			if i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
				add(lineNo, Warning, "headings/blank-line",
					"add a blank line after this heading")
			}
			// Headings (H2 and deeper) should be sentence case.
			if level >= 2 {
				if word, bad := sentenceCaseIssue(text); bad {
					add(lineNo, Warning, "headings/sentence-case",
						fmt.Sprintf("heading should be sentence case; lowercase %q (or add it to exceptions.txt if it's a proper noun)", word))
				}
			}

			prevLevel = level
			pendingHeadingLine = lineNo
			continue
		}

		if trimmed != "" {
			// Any real content clears the "stacked heading" watch.
			pendingHeadingLine = 0
		}

		checkLinks(add, lineNo, line)
		checkImages(add, lineNo, line)
	}

	return findings
}

// checkCodeLine applies the code rules to one line inside a fenced block.
func checkCodeLine(add func(int, Level, string, string), lineNo int, line, lang string) {
	if !shellLangs[lang] {
		return
	}
	if promptRe.MatchString(line) {
		add(lineNo, ErrorLevel, "code/shell-prompt",
			"remove the shell prompt \"$\" so the command can be copy-pasted")
	}
	// Only warn about sed in explicitly shell-tagged blocks, not un-hinted ones,
	// to avoid false positives on arbitrary text.
	if lang != "" && sedRe.MatchString(line) {
		add(lineNo, Warning, "code/sed",
			"avoid `sed`; its behaviour differs between BSD and GNU")
	}
}

// checkLinks flags non-descriptive markdown link text on a line.
func checkLinks(add func(int, Level, string, string), lineNo int, line string) {
	for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
		isImage := m[1] == "!"
		if isImage {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(emphasisRe.ReplaceAllString(m[2], "")))
		if badLinkText[text] {
			add(lineNo, Warning, "links/non-descriptive",
				fmt.Sprintf("link text %q is not descriptive; say where the link goes", strings.TrimSpace(m[2])))
		}
	}
}

// checkImages flags image filenames that are not kebab-case, for both the
// markdown ![alt](path) form and the <img src="path"> form.
func checkImages(add func(int, Level, string, string), lineNo int, line string) {
	check := func(path string) {
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
			return
		}
		base := filepath.Base(path)
		// Drop any query/hash fragment.
		if idx := strings.IndexAny(base, "?#"); idx >= 0 {
			base = base[:idx]
		}
		ext := strings.ToLower(filepath.Ext(base))
		if !imageExts[ext] {
			return
		}
		name := strings.TrimSuffix(base, filepath.Ext(base))
		if !kebabRe.MatchString(name) {
			add(lineNo, Warning, "images/filename",
				fmt.Sprintf("image filename %q should be kebab-case (lowercase words joined by hyphens, e.g. accessing-exposed-service%s)", base, ext))
		}
	}
	for _, m := range linkRe.FindAllStringSubmatch(line, -1) {
		if m[1] != "!" {
			continue
		}
		// Path may be followed by a title: ![alt](path "title"). Guard against a
		// whitespace-only target like ![alt]( ), which has no fields.
		fields := strings.Fields(m[3])
		if len(fields) == 0 {
			continue
		}
		check(fields[0])
	}
	for _, m := range srcRe.FindAllStringSubmatch(line, -1) {
		check(m[1])
	}
}

// --- Case checks ------------------------------------------------------------

// frontmatterTitle returns the value of a `title:` line and true if the line
// is one. Surrounding quotes are stripped.
func frontmatterTitle(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, "title:") {
		return "", false
	}
	v := strings.TrimSpace(strings.TrimPrefix(t, "title:"))
	v = strings.Trim(v, `"'`)
	if v == "" {
		return "", false
	}
	return v, true
}

// stripHeadingInline removes inline code and reduces links to their text so the
// case checks only look at real words.
func stripHeadingInline(s string) string {
	s = inlineCodeRe.ReplaceAllString(s, " ")
	s = mdLinkTextRe.ReplaceAllString(s, "$1")
	s = emphasisRe.ReplaceAllString(s, "")
	return s
}

// coreWord trims surrounding punctuation and a trailing possessive "'s".
func coreWord(w string) string {
	w = strings.TrimFunc(w, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	w = strings.TrimSuffix(w, "'s")
	w = strings.TrimSuffix(w, "’s")
	return w
}

// endsSentence reports whether a word ends with sentence-ending punctuation,
// after which a capital letter is expected.
func endsSentence(w string) bool {
	w = strings.TrimRight(w, `"')`)
	return strings.HasSuffix(w, ".") || strings.HasSuffix(w, ":") ||
		strings.HasSuffix(w, "?") || strings.HasSuffix(w, "!")
}

// sentenceCaseIssue returns the first wrongly-capitalized word in a heading and
// true if the heading is not sentence case.
func sentenceCaseIssue(text string) (string, bool) {
	words := strings.Fields(stripHeadingInline(text))
	for i, w := range words {
		if i == 0 {
			continue // the first word is expected to be capitalized
		}
		core := coreWord(w)
		if core == "" || !plainCapRe.MatchString(core) {
			continue
		}
		if exceptionSet[strings.ToLower(core)] {
			continue
		}
		if endsSentence(words[i-1]) {
			continue // start of a new sentence, capital is fine
		}
		return core, true
	}
	return "", false
}

// titleCaseIssue returns the first word that should be capitalized in a title
// and true if the title is not title case. It flags a title only when a
// significant (non-small, non-exception) word is left lowercase.
func titleCaseIssue(text string) (string, bool) {
	words := strings.Fields(stripHeadingInline(text))
	for i, w := range words {
		core := coreWord(w)
		if core == "" || !plainLowerRe.MatchString(core) {
			continue
		}
		if i != 0 && smallWords[strings.ToLower(core)] {
			continue // small joining words stay lowercase
		}
		if exceptionSet[strings.ToLower(core)] {
			continue // e.g. a CLI tool name like omnictl
		}
		return core, true
	}
	return "", false
}

// fenceMarkerOf returns the fence marker (backticks or tildes) if the trimmed
// line opens or closes a fenced code block, or "" otherwise.
func fenceMarkerOf(trimmed string) string {
	for _, c := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, c) {
			// Count the run of the fence character.
			n := 0
			for n < len(trimmed) && trimmed[n] == c[0] {
				n++
			}
			return trimmed[:n]
		}
	}
	return ""
}

// fenceLangOf returns the language hint of an opening fence line, or "".
func fenceLangOf(trimmed string) string {
	info := strings.TrimLeft(trimmed, "`~")
	info = strings.TrimSpace(info)
	if info == "" {
		return ""
	}
	return strings.Fields(info)[0]
}

// --- Reporting --------------------------------------------------------------

func report(findings []Finding, format string) (warnings, errors int) {
	var lastFile string
	for _, f := range findings {
		if f.Level == ErrorLevel {
			errors++
		} else {
			warnings++
		}
		switch format {
		case "github":
			fmt.Printf("::%s file=%s,line=%d,title=%s::%s\n",
				f.Level, f.File, f.Line, f.Rule, f.Message)
		default:
			if f.File != lastFile {
				fmt.Printf("\n%s\n", f.File)
				lastFile = f.File
			}
			fmt.Printf("  %d: [%s] %s: %s\n", f.Line, f.Level, f.Rule, f.Message)
		}
	}
	return warnings, errors
}
