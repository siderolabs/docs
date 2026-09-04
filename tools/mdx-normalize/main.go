// Command mdx-normalize cleans up generated Markdown/MDX so it renders
// correctly on Mintlify.
//
// Tools such as `omnictl docs`, `talosctl docs` and upstream `configuration.md`
// files produce constructs that Mintlify (which parses .mdx as MDX/JSX) does
// not handle:
//
//   - Tab-indented code blocks. Mintlify does not support indented code
//     blocks, so a line like `source <(talosctl completion bash)` is read as
//     JSX and breaks the build ("Unexpected character `(` before name"). These
//     are converted to fenced (```) code blocks.
//
//   - Tab-indented prose in a command's "Synopsis". This should stay a normal
//     paragraph, not become a code block. Command examples are distinguished
//     from prose by their intro line: examples are introduced by a line ending
//     in a colon ("...run:" or a "#### Linux:" heading), so a tab-indented
//     block with a colon intro is fenced and any other is de-indented.
//
//   - Tab-indented *lists* nested under a list item (e.g. sub-bullets under
//     "- For each node:"). These look like colon-intro examples but must stay a
//     list, not become a code block, so they are re-indented instead of fenced.
//
//   - (with --strip-hr) "---" horizontal-rule separators sprinkled between
//     sections, which render as noisy horizontal lines.
//
//   - (with --escape-inline) MDX-significant characters ("<" and "{") in prose,
//     such as "<machine-id>" placeholders in CLI help, which Mintlify would try
//     to parse as JSX/expressions. It escapes every "<"/"{" in prose (only
//     inline code spans and already-escaped characters are skipped). This is
//     OPT-IN, and applied only to plain-markdown CLI help, because it must never
//     run on files that contain real HTML/JSX (e.g. the <table> markup in the
//     Talos configuration reference or the schema-generated Omni pages) —
//     escaping those would destroy the tables. Such files are normalized with
//     escaping off, which leaves their HTML, MDX expressions ({"<"}) and
//     comments ({/* ... */}) untouched.
//
// The file's leading YAML frontmatter block is always preserved verbatim.
//
// Every transformation is idempotent: running the tool again on its own output
// is a no-op.
//
// Usage:
//
//	mdx-normalize [--strip-hr] [--escape-inline] [file.mdx ...]
//
// With one or more file arguments, each file is normalized in place. With no
// argument (or "-") it reads stdin and writes stdout.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

var (
	fenceRe    = regexp.MustCompile("^[ ]*```")
	hrRe       = regexp.MustCompile(`^---[ \t]*$`)
	colonRe    = regexp.MustCompile(`:[ \t]*$`)
	blankRe    = regexp.MustCompile(`^[ \t]*$`)
	listItemRe = regexp.MustCompile(`^[ \t]*([-*+]|[0-9]+[.)])[ \t]`)
)

func main() {
	stripHR := flag.Bool("strip-hr", false, "remove standalone '---' horizontal-rule separators")
	escapeInline := flag.Bool("escape-inline", false, "backslash-escape '<' and '{' in prose (for plain-markdown CLI help; never use on files with real HTML/JSX)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdx-normalize [--strip-hr] [--escape-inline] [file.mdx ...]")
		fmt.Fprintln(os.Stderr, "  With file arguments, each file is normalized in place.")
		fmt.Fprintln(os.Stderr, "  With no argument (or '-'), reads stdin and writes stdout.")
		flag.PrintDefaults()
	}
	flag.Parse()

	// With no argument (or a single "-") we act as a stdin->stdout filter; this
	// mode is used from containers so the file is never read/written through a
	// bind mount, which avoids Docker Desktop mount-consistency races on a
	// just-written file. With file arguments each file is normalized in place.
	args := flag.Args()
	if len(args) == 0 || (len(args) == 1 && args[0] == "-") {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
			os.Exit(1)
		}
		if _, err = os.Stdout.Write(normalizeBytes(data, *stripHR, *escapeInline)); err != nil {
			fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
			os.Exit(1)
		}
		return
	}

	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
			os.Exit(1)
		}
		if err = os.WriteFile(path, normalizeBytes(data, *stripHR, *escapeInline), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
			os.Exit(1)
		}
	}
}

// normalizeBytes normalizes a whole file's bytes, preserving a trailing newline
// (strings.Split leaves a final "" element that Join turns back into it).
//
// A CRLF file is converted to LF for the line-oriented passes below — every
// pattern they match ("---" frontmatter and horizontal rules, a blank line, a
// colon intro) is anchored at end-of-line, and a trailing "\r" would defeat all
// of them — and converted back afterwards, so a CRLF file is normalized
// correctly and stays CRLF. (A file with mixed endings is normalized to CRLF
// throughout; nothing in the corpus mixes them.)
func normalizeBytes(data []byte, stripHR, escapeInline bool) []byte {
	s := string(data)
	crlf := strings.Contains(s, "\r\n")
	if crlf {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	out := strings.Join(normalize(strings.Split(s, "\n"), stripHR, escapeInline), "\n")
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return []byte(out)
}

// isFrontmatterDelim reports whether a line is a YAML frontmatter fence,
// tolerating trailing whitespace (some sources emit "--- ").
func isFrontmatterDelim(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

// blockIsList reports whether the first non-blank line of a block is a list
// item, i.e. the block is a (usually nested) list rather than a code example.
func blockIsList(block []string) bool {
	for _, l := range block {
		if strings.TrimSpace(l) == "" {
			continue
		}
		return listItemRe.MatchString(l)
	}
	return false
}

// escapeInlineMDX backslash-escapes MDX-significant characters ("<" and "{") in
// prose so Mintlify does not try to parse them as JSX/expressions. CLI help text
// often contains placeholders such as "<machine-id>" that would otherwise break
// the build. It escapes every "<"/"{", including expression-looking text such as
// {"cni":"flannel"}, because it runs only on plain-markdown CLI help with no
// intentional JSX. The only exceptions, so the pass stays idempotent, are:
//   - characters inside inline code spans (backtick-delimited),
//   - characters already preceded by a backslash.
//
// Code spans are only honoured when the line's backticks are balanced (an even
// count). An unpaired backtick — common in CLI help, e.g. "use `omnictl and
// then <machine-id> is required" — is not a code span at all, and treating it
// as one would leave the rest of the line unescaped and ship exactly the
// character this pass exists to escape.
func escapeInlineMDX(s string) string {
	if !strings.ContainsAny(s, "<{") {
		return s
	}
	balanced := strings.Count(s, "`")%2 == 0
	var b strings.Builder
	b.Grow(len(s) + 8)
	inCode := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '`' {
			if balanced {
				inCode = !inCode
			}
			b.WriteByte(c)
			continue
		}
		// Escape a "<" or "{" in prose unless it is inside an inline code span or
		// already backslash-escaped (so the pass stays idempotent). Everything is
		// escaped, including expression-looking prose such as {"cni":"flannel"}:
		// this pass runs only on plain-markdown CLI help that contains no
		// intentional JSX, so an unescaped brace or bracket can only break the
		// build.
		if !inCode && (c == '<' || c == '{') && !(i > 0 && s[i-1] == '\\') {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

// blockHasFence reports whether a pending block already contains a fence line.
// Such a block is (or contains) a code block already, so wrapping it in another
// fence would nest fences — and would not survive a second pass, since the
// added fence changes how the same input is read.
func blockHasFence(block []string) bool {
	for _, l := range block {
		// Not fenceRe: that only allows leading spaces, and a block line here
		// may still be tab-indented (one tab has been stripped).
		if strings.HasPrefix(strings.TrimLeft(l, " \t"), "```") {
			return true
		}
	}
	return false
}

func normalize(lines []string, stripHR, escapeInline bool) []string {
	out := make([]string, 0, len(lines))
	i := 0

	maybeEscape := func(s string) string {
		if escapeInline {
			return escapeInlineMDX(s)
		}
		return s
	}

	// Copy a leading YAML frontmatter block through untouched so its title,
	// description, and closing "---" are never altered.
	if len(lines) > 0 && isFrontmatterDelim(lines[0]) {
		out = append(out, lines[0])
		for i = 1; i < len(lines); i++ {
			out = append(out, lines[i])
			if isFrontmatterDelim(lines[i]) {
				i++
				break
			}
		}
	}

	var (
		inFence      bool
		block        []string // pending tab-indented block, one leading tab stripped
		intro        string   // last non-blank line before the block started
		lastNonBlank string
		pendBlanks   int  // blank lines held while a block is open
		hrSkip       bool // just dropped an HR; swallow one following blank
	)

	flush := func() {
		if len(block) == 0 {
			return
		}
		switch {
		case listItemRe.MatchString(intro) && blockIsList(block):
			// A list nested under a list item (e.g. sub-bullets under
			// "- For each node:"). Keep it a list rather than a code block. Each
			// remaining leading tab (block lines already had one tab stripped)
			// becomes two spaces so deeper nesting is preserved, and the whole
			// block is indented one level under the intro. The item text is still
			// escaped (when enabled) like any other prose, so the pass stays
			// idempotent and placeholders inside list items don't break the build.
			for _, bl := range block {
				if bl == "" {
					out = append(out, "")
					continue
				}
				n := 0
				for n < len(bl) && bl[n] == '\t' {
					n++
				}
				out = append(out, "  "+strings.Repeat("  ", n)+maybeEscape(bl[n:]))
			}
		case blockHasFence(block):
			// The block already contains a fenced code block (a tab-indented
			// fence is not matched by fenceRe, so it lands here rather than
			// being tracked as an open fence). Emit it de-indented instead of
			// wrapping it in another fence, which would nest fences and would
			// not be idempotent.
			out = append(out, block...)
		case colonRe.MatchString(intro):
			// A command example introduced by "...:" — fence it.
			out = append(out, "```")
			out = append(out, block...)
			out = append(out, "```")
		default:
			// De-indented prose.
			for _, bl := range block {
				out = append(out, maybeEscape(bl))
			}
		}
		block = block[:0]
		for ; pendBlanks > 0; pendBlanks-- {
			out = append(out, "")
		}
	}

	for ; i < len(lines); i++ {
		line := lines[i]

		switch {
		case fenceRe.MatchString(line):
			flush()
			out = append(out, line)
			inFence = !inFence
			lastNonBlank = ""
			hrSkip = false

		case inFence:
			out = append(out, line)

		case strings.HasPrefix(line, "\t") && strings.TrimSpace(line) != "":
			if len(block) == 0 {
				intro = lastNonBlank
			}
			// Blank lines seen while the block was open were internal to it.
			for ; pendBlanks > 0; pendBlanks-- {
				block = append(block, "")
			}
			block = append(block, strings.TrimPrefix(line, "\t"))

		case stripHR && hrRe.MatchString(line):
			flush()
			hrSkip = true

		case blankRe.MatchString(line):
			switch {
			case len(block) > 0:
				pendBlanks++
			case hrSkip:
				hrSkip = false
			default:
				out = append(out, line)
			}

		default:
			hrSkip = false
			flush()
			out = append(out, maybeEscape(line))
			lastNonBlank = line
		}
	}
	flush()

	return out
}
