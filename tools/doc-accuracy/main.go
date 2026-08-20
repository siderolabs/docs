package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// reviewerPrompt is the static review instruction set. It lives in a Markdown
// file so it can be read and edited without touching Go, and is embedded so the
// built binary is self-contained (mirroring style-guide-checker/exceptions.txt).
//
//go:embed reviewer-prompt.md
var reviewerPrompt string

// verdictPrefix is the marker the reviewer prints on its final line so this
// tool can turn the review into an exit code. Keep it in sync with the
// "Final verdict" section of reviewer-prompt.md.
const verdictPrefix = "DOC_ACCURACY_VERDICT:"

// maxDiffLines caps the diff inlined into the prompt so a huge changeset can't
// blow past the model's context window. The reviewer still has Read/Grep to
// inspect anything truncated.
const maxDiffLines = 4000

func main() {
	base := flag.String("base", "HEAD", "git ref to diff against in changed mode")
	all := flag.Bool("all", false, "review every .mdx under public/ instead of only changed files")
	model := flag.String("model", "", "model to pass to `claude --model` (empty: claude's default)")
	report := flag.String("report", "_out/doc-accuracy-report.md", "also write the full report to this path (relative to the workspace)")
	workspace := flag.String("workspace", ".", "path to the docs repo root")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: doc-accuracy [flags] [files...]\n\n")
		fmt.Fprintf(os.Stderr, "AI-reviews the documentation for accuracy and harmful changes, driving\n")
		fmt.Fprintf(os.Stderr, "the `claude` CLI with the embedded reviewer prompt. Reviews changed .mdx\n")
		fmt.Fprintf(os.Stderr, "files by default, the whole public/ tree with -all, or exactly the .mdx\n")
		fmt.Fprintf(os.Stderr, "files given as arguments (paths relative to the workspace).\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	os.Exit(run(*workspace, *base, *all, *model, *report, flag.Args()))
}

// run executes one review and returns the process exit code. When explicit is
// non-empty, exactly those files are reviewed; otherwise the file set comes from
// git (all tracked docs with -all, else the changed ones).
func run(workspace, base string, all bool, model, report string, explicit []string) int {
	ws, err := filepath.Abs(workspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: resolving workspace:", err)
		return 2
	}

	var (
		files, newFiles []string
		mode            string
		includeDiff     bool
	)
	if len(explicit) > 0 {
		files, err = explicitFiles(ws, explicit)
		mode = "file"
	} else {
		files, newFiles, err = collectFiles(ws, base, all)
		mode = modeName(all)
		includeDiff = !all // only changed mode has a meaningful diff to show
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: selecting docs to review:", err)
		return 2
	}

	if len(files) == 0 {
		fmt.Printf("No .mdx files to review (mode: %s, base: %s). Nothing to do.\n", mode, base)
		return 0
	}

	fmt.Printf("==> Reviewing %d .mdx file(s) for accuracy (mode: %s)\n", len(files), mode)
	for _, f := range files {
		fmt.Printf("      %s\n", f)
	}

	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'claude' CLI not found on PATH.")
		fmt.Fprintln(os.Stderr, "  This tool uses Claude Code headless to review the docs.")
		fmt.Fprintln(os.Stderr, "  Install it from https://claude.com/claude-code and sign in.")
		return 2
	}

	// In changed mode, show the reviewer exactly what changed so it can zero in
	// on removed flags/mounts/safeguards — the highest-risk kind of edit.
	var diff string
	if includeDiff {
		raw, err := git(ws, append([]string{"diff", "--unified=3", base, "--"}, files...)...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: computing diff:", err)
			return 2
		}
		diff = truncateLines(raw, maxDiffLines)
	}

	prompt := buildPrompt(mode, base, includeDiff, files, newFiles, diff)

	// Stream the model's output to stdout while capturing it so we can read the
	// verdict back and save the report. The report file is written only after
	// the run (see below), so a failed run doesn't destroy a previous report.
	var captured bytes.Buffer
	out := io.MultiWriter(os.Stdout, &captured)

	fmt.Println()
	runErr := runClaude(ws, model, prompt, out)

	// Persist whatever the reviewer produced — even on failure — so there is a
	// record to read.
	if err := writeReport(ws, report, captured.String()); err != nil {
		fmt.Fprintln(os.Stderr, "\nWarning: could not save report:", err)
	} else {
		fmt.Printf("\n==> Full report saved to: %s\n", report)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "\nError: claude did not complete: %v\n", runErr)
		return 2
	}

	switch parseVerdict(captured.String()) {
	case "FAIL":
		fmt.Println("\n❌ Doc accuracy check FAILED: the reviewer found critical issues above.")
		return 1
	case "PASS":
		fmt.Println("\n✅ Doc accuracy check passed: no critical issues found.")
		return 0
	default:
		fmt.Fprintln(os.Stderr, "\nError: the reviewer did not emit a verdict line; treating as inconclusive.")
		fmt.Fprintln(os.Stderr, "  Read the report above to see what happened.")
		return 2
	}
}

// modeName is the human label for the review mode.
func modeName(all bool) string {
	if all {
		return "all"
	}
	return "changed"
}

// collectFiles returns the repo-relative .mdx files to review. In "all" mode
// that is every tracked .mdx under public/ (and newFiles is nil); otherwise
// files is those added/modified vs. base plus any new untracked ones, and
// newFiles is just the untracked ones — which have no prior version and so do
// not appear in `git diff`, so the caller flags them for full review.
func collectFiles(ws, base string, all bool) (files, newFiles []string, err error) {
	if all {
		tracked, err := git(ws, "ls-files", "--", "public")
		if err != nil {
			return nil, nil, err
		}
		return filterMDX(tracked), nil, nil
	}

	changed, err := git(ws, "diff", "--name-only", "--diff-filter=AM", base, "--", "public")
	if err != nil {
		return nil, nil, err
	}
	untracked, err := git(ws, "ls-files", "--others", "--exclude-standard", "--", "public")
	if err != nil {
		return nil, nil, err
	}
	return filterMDX(changed, untracked), filterMDX(untracked), nil
}

// explicitFiles validates a caller-supplied list of paths (relative to ws):
// each must end in .mdx and exist. Paths are de-duplicated and sorted.
func explicitFiles(ws string, paths []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		if !strings.HasSuffix(p, ".mdx") {
			return nil, fmt.Errorf("not an .mdx file: %s", p)
		}
		if _, err := os.Stat(filepath.Join(ws, p)); err != nil {
			return nil, fmt.Errorf("file not found: %s", p)
		}
		seen[p] = true
		files = append(files, p)
	}
	sort.Strings(files)
	return files, nil
}

// filterMDX keeps only the .mdx paths from one or more git output blobs,
// de-duplicated and sorted. Blank lines are ignored.
func filterMDX(blobs ...string) []string {
	seen := map[string]bool{}
	var files []string
	for _, blob := range blobs {
		for _, line := range strings.Split(blob, "\n") {
			p := strings.TrimSpace(line)
			if p == "" || !strings.HasSuffix(p, ".mdx") || seen[p] {
				continue
			}
			seen[p] = true
			files = append(files, p)
		}
	}
	sort.Strings(files)
	return files
}

// buildPrompt assembles the full reviewer prompt: the embedded instructions
// followed by the dynamic context for this run (mode, file list, any newly
// added files, and — when includeDiff is set — the diff).
func buildPrompt(mode, base string, includeDiff bool, files, newFiles []string, diff string) string {
	var b strings.Builder
	b.WriteString(reviewerPrompt)
	b.WriteString("\n\n---\n\n## This review\n\n")
	fmt.Fprintf(&b, "Mode: **%s**\n\n", mode)
	b.WriteString("Review these files (read each one):\n\n")
	for _, f := range files {
		fmt.Fprintf(&b, "- %s\n", f)
	}

	if len(newFiles) > 0 {
		b.WriteString("\nThese are newly added files with no prior version, so they do not appear\n")
		b.WriteString("in the diff below — read and review each one in full:\n\n")
		for _, f := range newFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}

	if includeDiff {
		fmt.Fprintf(&b, "\n### Diff of what changed (base: `%s`)\n\n", base)
		b.WriteString("Focus your review on these edits — a removed or altered flag, argument,\n")
		b.WriteString("mount, or value inside a code block is the highest-risk change.\n\n")
		// Fence with four backticks: the diff is of .mdx files, whose own
		// three-backtick code fences would otherwise close this block early.
		b.WriteString("````diff\n")
		b.WriteString(diff)
		if !strings.HasSuffix(diff, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("````\n")
	}

	return b.String()
}

// parseVerdict scans the reviewer output for the verdict line and returns
// "PASS", "FAIL", or "" if none was emitted. The last verdict line wins.
//
// It splits the (already in-memory) output on newlines rather than using a
// bufio.Scanner, so a single very long line can't hit the scanner's token-size
// limit and silently drop the verdict. It also tolerates the markdown the model
// sometimes wraps the line in — backticks (`...`), bold (**...**), or leading
// whitespace — so a correctly-stated verdict is never misread as inconclusive.
func parseVerdict(output string) string {
	verdict := ""
	for _, line := range strings.Split(output, "\n") {
		idx := strings.Index(line, verdictPrefix)
		if idx < 0 {
			continue
		}
		// Take the run of uppercase letters immediately after the prefix,
		// skipping any spaces/backticks/asterisks the model added.
		rest := strings.TrimLeft(line[idx+len(verdictPrefix):], " \t`*")
		word := rest
		if i := strings.IndexFunc(rest, func(r rune) bool { return r < 'A' || r > 'Z' }); i >= 0 {
			word = rest[:i]
		}
		switch word {
		case "PASS":
			verdict = "PASS"
		case "FAIL":
			verdict = "FAIL"
		}
	}
	return verdict
}

// truncateLines returns at most limit lines of s.
func truncateLines(s string, limit int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= limit {
		return s
	}
	return strings.Join(lines[:limit], "\n")
}

// writeReport writes the report content to report (resolved relative to ws),
// creating the parent directory. Called after the run so a failed review never
// destroys a previously saved report.
func writeReport(ws, report, content string) error {
	path := report
	if !filepath.IsAbs(path) {
		path = filepath.Join(ws, report)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// git runs `git -C dir args...` and returns stdout. A non-zero exit is an error
// that includes git's stderr, so failures like a bad -base ref are diagnosable.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(errOut.String()); msg != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// streamEvent is the subset of a claude stream-json event we care about: the
// assistant messages and the text blocks inside them.
type streamEvent struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// runClaude drives the claude CLI headless as a read-only reviewer: it may read
// the repo and consult upstream sources, but Edit/Write/NotebookEdit are
// disallowed so it can never modify the docs it is reviewing. The prompt is fed
// on stdin.
//
// It uses stream-json output and writes every assistant *text* block to out.
// Plain `--output-format text` returns only the model's final message, which
// the model sometimes reduces to just the verdict — discarding the findings it
// narrated in earlier messages. Reassembling the text blocks captures the whole
// review regardless of how the model split it across messages.
func runClaude(ws, model, prompt string, out io.Writer) error {
	args := []string{"-p"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args,
		"--output-format", "stream-json", "--verbose",
		"--permission-mode", "bypassPermissions",
		"--allowedTools", "Read", "Grep", "Glob", "WebFetch",
		"Bash(git diff:*)", "Bash(git log:*)", "Bash(git show:*)",
		"--disallowedTools", "Edit", "Write", "NotebookEdit",
	)

	cmd := exec.Command("claude", args...)
	cmd.Dir = ws
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// json.Decoder reads one JSON value at a time regardless of line length, so
	// a large single-line message can't overflow a fixed scanner buffer.
	dec := json.NewDecoder(stdout)
	for {
		var ev streamEvent
		if err := dec.Decode(&ev); err != nil {
			break // io.EOF, or a malformed event: stop parsing, drain below.
		}
		if ev.Type != "assistant" {
			continue
		}
		for _, block := range ev.Message.Content {
			if block.Type == "text" && block.Text != "" {
				fmt.Fprintln(out, block.Text)
			}
		}
	}

	// Drain anything left unparsed so claude never blocks writing to a full pipe.
	_, _ = io.Copy(io.Discard, dec.Buffered())
	_, _ = io.Copy(io.Discard, stdout)

	return cmd.Wait()
}
