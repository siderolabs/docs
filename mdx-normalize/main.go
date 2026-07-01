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
		fmt.Fprintln(os.Stderr, "usage: mdx-normalize [--strip-hr] <file.mdx>")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	path := flag.Arg(0)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
		os.Exit(1)
	}

	// Preserve a trailing newline: strings.Split leaves a final "" element
	// that Join turns back into the closing newline.
	lines := strings.Split(string(data), "\n")
	out := normalize(lines, *stripHR)

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "mdx-normalize: %v\n", err)
		os.Exit(1)
	}
}

// isFrontmatterDelim reports whether a line is a YAML frontmatter fence,
// tolerating trailing whitespace (some sources emit "--- ").
func isFrontmatterDelim(line string) bool {
	return strings.TrimRight(line, " \t") == "---"
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
		}
		out = append(out, block...)
		if fenced {
			out = append(out, "```")
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
			out = append(out, line)
			lastNonBlank = line
		}
	}
	flush()

	return out
}
