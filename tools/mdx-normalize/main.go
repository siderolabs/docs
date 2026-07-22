// Command mdx-normalize cleans up generated Markdown/MDX so it renders
// correctly on Mintlify.
//
// Tools such as `omnictl docs` and upstream `configuration.md` files produce
// constructs that Mintlify (which parses .mdx as MDX/JSX) does not handle:
//
//   - Tab-indented code blocks. Mintlify does not support indented code
//     blocks, so a line like `source <(omnictl completion bash)` is read as
//     JSX and breaks the build ("Unexpected character `(` before name"). These
//     are converted to fenced (```) code blocks.
//
//   - Tab-indented prose in a command's "Synopsis". This should stay a normal
//     paragraph, not become a code block. Command examples are distinguished
//     from prose by their intro line: examples are introduced by a line ending
//     in a colon ("...run:" or a "#### Linux:" heading), so a tab-indented
//     block with a colon intro is fenced and any other is de-indented.
//
//   - (with --strip-hr) "---" horizontal-rule separators sprinkled between
//     sections, which render as noisy horizontal lines.
//
// The file's leading YAML frontmatter block is always preserved verbatim.
// The file is normalized in place.
//
// Usage:
//
//	mdx-normalize [--strip-hr] <file.mdx>
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
	fenceRe = regexp.MustCompile("^[ ]*```")
	hrRe    = regexp.MustCompile(`^---[ \t]*$`)
	colonRe = regexp.MustCompile(`:[ \t]*$`)
	blankRe = regexp.MustCompile(`^[ \t]*$`)
)

func main() {
	stripHR := flag.Bool("strip-hr", false, "remove standalone '---' horizontal-rule separators")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: mdx-normalize [--strip-hr] [file.mdx]")
		fmt.Fprintln(os.Stderr, "  With a file argument, the file is normalized in place.")
		fmt.Fprintln(os.Stderr, "  With no argument (or '-'), reads stdin and writes stdout.")
		flag.PrintDefaults()
	}
	flag.Parse()

	// With a file argument we edit it in place; with none (or "-") we act as a
	// stdin->stdout filter. The filter mode is used from containers so the file
	// is never read/written through a bind mount, which avoids Docker Desktop
	// mount-consistency races on a just-written file.
	path := ""
	switch flag.NArg() {
	case 0:
		// stdin -> stdout
	case 1:
		if flag.Arg(0) != "-" {
			path = flag.Arg(0)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}

	var (
		data []byte
		err  error
	)
	if path == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
		os.Exit(1)
	}

	// Preserve a trailing newline: strings.Split leaves a final "" element
	// that Join turns back into the closing newline.
	out := normalize(strings.Split(string(data), "\n"), *stripHR)
	result := []byte(strings.Join(out, "\n"))

	if path == "" {
		if _, err = os.Stdout.Write(result); err != nil {
			fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if err = os.WriteFile(path, result, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
		os.Exit(1)
	}
}

// isFrontmatterDelim reports whether a line is a YAML frontmatter fence,
// tolerating trailing whitespace (some sources emit "--- ").
func isFrontmatterDelim(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
}

// escapeInlineMDX backslash-escapes MDX-significant characters ("<" and "{") in
// prose so Mintlify does not try to parse them as JSX/expressions. CLI help text
// often contains placeholders such as "<machine-id>" that would otherwise break
// the build. Characters inside inline code spans (backtick-delimited) are left
// alone, and characters already preceded by a backslash are not double-escaped.
func escapeInlineMDX(s string) string {
	if !strings.ContainsAny(s, "<{") {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 8)
	inCode := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '`' {
			inCode = !inCode
			b.WriteByte(c)
			continue
		}
		if !inCode && (c == '<' || c == '{') && !(i > 0 && s[i-1] == '\\') {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}
	return b.String()
}

func normalize(lines []string, stripHR bool) []string {
	out := make([]string, 0, len(lines))
	i := 0

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
		fenced := colonRe.MatchString(intro)
		if fenced {
			out = append(out, "```")
			out = append(out, block...)
			out = append(out, "```")
		} else {
			// De-indented prose: escape MDX-significant characters.
			for _, bl := range block {
				out = append(out, escapeInlineMDX(bl))
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
			out = append(out, escapeInlineMDX(line))
			lastNonBlank = line
		}
	}
	flush()

	return out
}
