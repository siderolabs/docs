package main

import (
	"bytes"
	"context"
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
	"time"
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

// reviewBatchSize is how many files each review run covers. A single model run
// cannot meaningfully review a large set (the whole tree is ~1700 files, and
// even a branch vs. a fresh main is routinely 100+), so every mode is split into
// batches of this size, each its own claude invocation.
const reviewBatchSize = 20

// claudeTimeout bounds a single review run so a stalled `claude` can't hang the
// command forever.
const claudeTimeout = 10 * time.Minute

func main() {
	base := flag.String("base", "HEAD", "git ref to diff against in changed mode")
	all := flag.Bool("all", false, "review every .mdx under public/ instead of only changed files")
	diffFile := flag.String("diff-file", "", "review the .mdx changes in this unified-diff file instead of the local git working tree; the changed docs are read from the diff as data and never from disk (used by CI to review a fork PR without checking out its ref)")
	model := flag.String("model", "", "model to pass to `claude --model` (empty: claude's default)")
	report := flag.String("report", "_out/doc-accuracy-report.md", "also write the full report to this path (relative to the workspace)")
	findingsOut := flag.String("findings-out", "_out/doc-accuracy-findings.json", "write machine-readable findings to this path (relative to the workspace)")
	format := flag.String("format", "text", "extra output on stdout: text (none) or github (::error/::warning annotations)")
	fetch := flag.Bool("fetch", true, "in changed mode, git-fetch a remote-tracking base (e.g. upstream/main) first so it isn't stale")
	maxFiles := flag.Int("max-files", 0, "skip the review entirely when more than this many .mdx files changed, instead of fanning out into many model runs (0: no cap)")
	verbose := flag.Bool("verbose", false, "stream the full reviewer report to the terminal; by default only the compact findings summary is shown (the full report is always saved to -report)")
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

	os.Exit(run(runConfig{
		workspace:   *workspace,
		base:        *base,
		all:         *all,
		diffFile:    *diffFile,
		fetch:       *fetch,
		maxFiles:    *maxFiles,
		model:       *model,
		report:      *report,
		findingsOut: *findingsOut,
		format:      *format,
		verbose:     *verbose,
		explicit:    flag.Args(),
	}))
}

// runConfig is the resolved configuration for one review invocation.
type runConfig struct {
	workspace   string
	base        string
	all         bool
	diffFile    string
	fetch       bool
	maxFiles    int
	model       string
	report      string
	findingsOut string
	format      string
	verbose     bool
	explicit    []string
}

// run executes one review and returns the process exit code. When explicit is
// non-empty, exactly those files are reviewed; otherwise the file set comes from
// git (all tracked docs with -all, else the changed ones).
func run(cfg runConfig) int {
	ws, err := filepath.Abs(cfg.workspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: resolving workspace:", err)
		return 2
	}

	// Keep the base current before diffing so "changed since main" reflects the
	// real mainline, not a stale local snapshot. Best-effort: a failed fetch
	// (offline, unknown remote) warns and the review proceeds against whatever
	// snapshot exists. Only meaningful in changed mode with a remote-tracking base.
	if cfg.fetch && cfg.diffFile == "" && len(cfg.explicit) == 0 && !cfg.all {
		maybeFetchBase(ws, cfg.base)
	}

	var (
		files, newFiles []string
		mode            string
		includeDiff     bool
		// dataOnly is set in diff-file mode: the changed docs come entirely from
		// the supplied diff (as data), and the reviewer is told not to read them
		// from disk — the working tree is the base, not the proposed change. It
		// may still Read/Grep the base checkout for trusted source to ground the
		// review. sectionByFile then holds each changed file's slice of that diff,
		// so the per-batch diff can be reassembled without a local `git diff`.
		dataOnly      bool
		sectionByFile map[string]string
	)
	// diffRef is the ref the changed set and diff are computed against. In changed
	// mode it is the *fork point* (merge-base of base and HEAD), not the base tip,
	// so the review covers only what this branch introduced — never files that
	// advanced on the base while the branch sat behind it.
	diffRef := cfg.base
	switch {
	case cfg.diffFile != "":
		files, sectionByFile, err = collectDiffFile(cfg.diffFile)
		mode = "changed"
		includeDiff = true
		dataOnly = true
	case len(cfg.explicit) > 0:
		files, err = explicitFiles(ws, cfg.explicit)
		mode = "file"
	default:
		if !cfg.all {
			diffRef = diffBase(ws, cfg.base)
		}
		files, newFiles, err = collectFiles(ws, diffRef, cfg.all)
		mode = modeName(cfg.all)
		includeDiff = !cfg.all // only changed mode has a meaningful diff to show
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: selecting docs to review:", err)
		return 2
	}

	if len(files) == 0 {
		// An empty set is a legitimate pass, but say so unambiguously — it must
		// not read the same as "reviewed everything and found nothing."
		fmt.Printf("Reviewed 0 .mdx file(s) (mode: %s, base: %s).\n", mode, cfg.base)
		if mode == "changed" {
			fmt.Println("  Nothing changed vs the base. If you expected changes, check that")
			fmt.Println("  -base is the right ref and has been fetched (e.g. upstream/main).")
		}
		return 0
	}

	// A cap on how many files one run may review. Every mode fans out into
	// reviewBatchSize batches, each its own claude invocation with its own
	// budget, so a very large changeset turns into a lot of model runs for a
	// review no reader would sit through. Past the cap the run is skipped
	// outright rather than silently trimmed: it records a SKIPPED verdict the
	// caller can report ("too large to review") and exits 0, because this is
	// advisory tooling and not a merge gate. 0 (the default, and what local runs
	// use) means no cap.
	if cfg.maxFiles > 0 && len(files) > cfg.maxFiles {
		reason := fmt.Sprintf("%d changed .mdx file(s) exceeds the -max-files cap of %d; not reviewed", len(files), cfg.maxFiles)
		fmt.Printf("==> Skipping review: %s.\n", reason)
		fmt.Println("      Review a change this size by hand, or raise -max-files.")
		if err := writeReport(ws, cfg.report, "Review skipped: "+reason+".\n"); err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not save report:", err)
		}
		if err := writeFindingsReport(ws, cfg.findingsOut, findingsReport{Verdict: "SKIPPED", Reason: reason, Findings: []Finding{}}); err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not save findings:", err)
		}
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

	newSet := map[string]bool{}
	for _, f := range newFiles {
		newSet[f] = true
	}

	// Every mode is batched: a large changed set (branch vs. a fresh main) is
	// just as unreviewable in one call as the whole tree, so both are chunked.
	batches := chunk(files, reviewBatchSize)

	var (
		report_  bytes.Buffer
		findings []Finding
		anyFail  bool
		anyGap   bool // a batch that produced no verdict
		runErr   error
	)
	for i, batch := range batches {
		if len(batches) > 1 {
			fmt.Printf("\n==> Batch %d/%d (%d file(s))\n", i+1, len(batches), len(batch))
		}

		// Diff per batch so each prompt stays bounded even when the whole changed
		// set is large; the diff for a batch covers only that batch's files.
		var diff string
		if includeDiff {
			var raw string
			if dataOnly {
				// The diff was supplied, not computed: stitch together just this
				// batch's file sections, in batch order.
				var b strings.Builder
				for _, f := range batch {
					b.WriteString(sectionByFile[f])
				}
				raw = b.String()
			} else {
				raw, err = git(ws, append([]string{"diff", "--unified=3", diffRef, "--"}, batch...)...)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: computing diff:", err)
					return 2
				}
			}
			diff = truncateLines(raw, maxDiffLines)
		}

		// New (untracked) files in this batch have no prior version, so they do
		// not appear in the diff — flag just this batch's new files for full read.
		var batchNew []string
		for _, f := range batch {
			if newSet[f] {
				batchNew = append(batchNew, f)
			}
		}

		prompt := buildPrompt(mode, cfg.base, includeDiff, dataOnly, batch, batchNew, diff)

		// Capture the full report for the file; only mirror it live to the terminal
		// in verbose mode. By default the terminal shows just the compact summary
		// printed after the run, not the model's whole prose report.
		var captured bytes.Buffer
		var out io.Writer = &captured
		if cfg.verbose {
			out = io.MultiWriter(os.Stdout, &captured)
			fmt.Println()
		} else {
			fmt.Println("      reviewing… (pass -verbose to stream the full report)")
		}
		if err := runClaude(ws, cfg.model, prompt, out); err != nil {
			runErr = err
			break
		}
		if report_.Len() > 0 {
			report_.WriteString("\n\n---\n\n")
		}
		report_.Write(captured.Bytes())
		findings = append(findings, parseFindings(captured.String())...)

		switch parseVerdict(captured.String()) {
		case "FAIL":
			anyFail = true
		case "PASS":
			// nothing
		default:
			anyGap = true
		}
	}

	// Persist whatever the reviewer produced — even on failure — so there is a
	// record to read. Written once, after the run, so a failed run never
	// destroys a previous report.
	if err := writeReport(ws, cfg.report, report_.String()); err != nil {
		fmt.Fprintln(os.Stderr, "\nWarning: could not save report:", err)
	} else {
		fmt.Printf("\n==> Full report saved to: %s\n", cfg.report)
	}

	// Persist the machine-readable findings for downstream tooling (e.g. the CI
	// step that turns criticals into a PR comment). The verdict is authoritative:
	// FAIL if any batch's verdict line said so, or if a parsed finding is CRITICAL
	// — so a malformed findings block can never downgrade a real FAIL to a clean
	// report in the file the PR comment reads.
	fileVerdict := overallVerdict(anyFail, hasCritical(findings), runErr != nil, anyGap)
	if err := writeFindings(ws, cfg.findingsOut, fileVerdict, findings); err != nil {
		fmt.Fprintln(os.Stderr, "\nWarning: could not save findings:", err)
	}

	// A compact, stable one-line-per-finding summary — the scannable view, most
	// severe first — so the terminal isn't only the model's free-form prose.
	printSummary(os.Stdout, findings)

	// GitHub-annotation output: one workflow command per finding, so each shows
	// up pinned to its line in the PR's Files-changed view.
	if cfg.format == "github" {
		emitAnnotations(os.Stdout, findings)
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "\nError: claude did not complete: %v\n", runErr)
		return 2
	}

	// Aggregate across batches, worst outcome wins: FAIL > inconclusive > PASS.
	switch {
	case anyFail:
		fmt.Println("\n❌ Doc accuracy check FAILED: the reviewer found critical issues above.")
		return 1
	case anyGap:
		fmt.Fprintln(os.Stderr, "\nError: a review produced no verdict line; treating as inconclusive.")
		fmt.Fprintln(os.Stderr, "  Read the report above to see what happened.")
		return 2
	default:
		fmt.Println("\n✅ Doc accuracy check passed: no critical issues found.")
		return 0
	}
}

// chunk splits s into consecutive slices of at most size elements.
func chunk(s []string, size int) [][]string {
	if size < 1 {
		size = 1
	}
	var out [][]string
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
	}
	return out
}

// Finding is one machine-readable issue the reviewer emits after its human
// report (see findingsMarker). It exists so this tool can turn the review into
// GitHub annotations and a PR comment without scraping prose.
//
// Details and Fix are the same "why" and "suggested correction" the model writes
// in its human report, carried here so an inline annotation can show them — the
// annotation is where a reader gets the full explanation; the PR comment stays a
// terse summary that points at it. Both are optional: an older or terser review
// that emits only file/line/severity/summary still parses.
type Finding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"`
	Summary  string `json:"summary"`
	Details  string `json:"details,omitempty"`
	Fix      string `json:"fix,omitempty"`
}

// findingsMarker introduces the fenced JSON block of findings in the reviewer's
// output. Keep it in sync with the "Machine-readable findings" section of
// reviewer-prompt.md.
const findingsMarker = "DOC_ACCURACY_FINDINGS"

// parseFindings extracts the fenced JSON findings block the reviewer emits after
// its human report. Anything missing or malformed yields nil: annotations are a
// best-effort convenience layered on top of the verdict, never a reason to fail.
func parseFindings(output string) []Finding {
	idx := strings.LastIndex(output, findingsMarker)
	if idx < 0 {
		return nil
	}
	block := extractFencedBlock(output[idx:])
	if block == "" {
		return nil
	}
	var fs []Finding
	if err := json.Unmarshal([]byte(block), &fs); err != nil {
		return nil
	}
	var out []Finding
	for _, f := range fs {
		if f.File == "" || strings.TrimSpace(f.Summary) == "" {
			continue // drop entries too incomplete to annotate
		}
		out = append(out, f)
	}
	return out
}

// extractFencedBlock returns the body of the first ``` fenced code block in s,
// dropping an optional info string (e.g. "json") on the opening fence line.
func extractFencedBlock(s string) string {
	start := strings.Index(s, "```")
	if start < 0 {
		return ""
	}
	rest := s[start+3:]
	nl := strings.IndexByte(rest, '\n')
	if nl < 0 {
		return ""
	}
	rest = rest[nl+1:] // skip the rest of the opening fence line (the info string)
	end := strings.Index(rest, "```")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// findingsReport is the machine-readable artifact this tool writes: the overall
// verdict alongside the findings that justify it, so a consumer (the CI comment
// step) gets both from one file. verdict is derived from the findings — FAIL iff
// any is CRITICAL — matching the reviewer's own rule that only criticals fail.
type findingsReport struct {
	Verdict string `json:"verdict"`
	// Reason explains a verdict that isn't a review result — currently only
	// SKIPPED, where no review ran at all (too many changed files). Empty for
	// PASS / FAIL / INCOMPLETE, which the findings themselves account for.
	Reason   string    `json:"reason,omitempty"`
	Findings []Finding `json:"findings"`
}

// overallVerdict is the authoritative result written to the findings file:
//   - FAIL when a critical was found (a real, actionable result),
//   - INCOMPLETE when the run did not finish cleanly — a batch that crashed or
//     timed out (didNotFinish) or produced no verdict line (anyGap),
//   - PASS otherwise.
//
// The key property: PASS is returned *only* when the review actually completed
// with nothing critical. A crashed or verdict-less run is INCOMPLETE, never PASS,
// so downstream tooling (the PR comment) can never mistake a failed run for a
// clean review and overwrite a real FAIL with "nothing flagged".
func overallVerdict(anyFail, anyCritical, didNotFinish, anyGap bool) string {
	switch {
	case anyFail || anyCritical:
		return "FAIL"
	case didNotFinish || anyGap:
		return "INCOMPLETE"
	default:
		return "PASS"
	}
}

// hasCritical reports whether any finding is CRITICAL.
func hasCritical(findings []Finding) bool {
	for _, f := range findings {
		if strings.EqualFold(strings.TrimSpace(f.Severity), "CRITICAL") {
			return true
		}
	}
	return false
}

// writeFindings writes the findings report as JSON to path (resolved relative to
// ws): an object with a verdict and the findings array. An empty run still
// writes a valid object ({"verdict":"PASS","findings":[]}) so downstream tooling
// always has something to parse.
//
// verdict is the run's *authoritative* verdict, taken from the reviewer's verdict
// line — not re-derived from findings. This matters when the findings JSON block
// is malformed and parses to nothing: the run can still be a FAIL, and the file
// must say so, or a consumer (the PR comment) would report the PR as clean while
// the check actually failed.
func writeFindings(ws, path, verdict string, findings []Finding) error {
	return writeFindingsReport(ws, path, findingsReport{Verdict: verdict, Findings: findings})
}

// writeFindingsReport writes an already-assembled report as JSON to path
// (resolved relative to ws), creating the parent directory. It is the single
// writer of the findings file, used both for a completed review and for the
// SKIPPED report a capped-out run writes instead of reviewing.
func writeFindingsReport(ws, path string, report findingsReport) error {
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(ws, path)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	if report.Findings == nil {
		report.Findings = []Finding{}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(full, append(data, '\n'), 0o644)
}

// severityRank orders severities for display: most severe first. Unknown
// severities sort last, alongside NOTICE.
func severityRank(sev string) int {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case "CRITICAL":
		return 0
	case "WARNING":
		return 1
	default:
		return 2
	}
}

// sortedFindings returns findings ordered most-severe-first, then by file and
// line, so the summary reads the same way every run regardless of the order the
// model happened to emit them in.
func sortedFindings(findings []Finding) []Finding {
	out := append([]Finding(nil), findings...)
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := severityRank(out[i].Severity), severityRank(out[j].Severity); ri != rj {
			return ri < rj
		}
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// oneLine collapses all whitespace runs (newlines included) into single spaces
// and trims the result.
//
// Findings text is written by the model, which is reasoning about attacker-
// controlled diff content, and this summary goes to stdout — which in CI the
// runner parses for `::workflow commands` at the start of a line. A finding
// carrying an embedded newline could otherwise forge one (a fake `::error`, or
// `::stop-commands` to suppress the real annotations). Keeping each finding on
// one line removes the line start it would need. This is the plain-text
// counterpart to escapeData, which does the same job for annotations.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// printSummary writes a compact one-line-per-finding summary to w, most severe
// first. Nothing is printed when there are no structured findings.
func printSummary(w io.Writer, findings []Finding) {
	if len(findings) == 0 {
		return
	}
	fmt.Fprintf(w, "\n==> Findings (%d), most severe first:\n", len(findings))
	for _, f := range sortedFindings(findings) {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(w, "  %-8s %s — %s\n", strings.ToUpper(strings.TrimSpace(f.Severity)), loc, oneLine(f.Summary))
		// A one-line fix under the finding keeps the summary actionable without
		// turning it into the full report (which stays in -report / the JSON).
		if fix := oneLine(f.Fix); fix != "" {
			fmt.Fprintf(w, "           fix: %s\n", fix)
		}
	}
}

// severityLevel maps a reviewer severity to a GitHub annotation level. Unknown
// values fall back to "notice" so a finding is never silently dropped.
func severityLevel(sev string) string {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case "CRITICAL":
		return "error"
	case "WARNING":
		return "warning"
	default:
		return "notice"
	}
}

// emitAnnotations writes one GitHub workflow command per finding so each shows
// up inline on its line in the PR. A finding whose line is unknown attaches to
// the top of the file. The annotation is the *detailed* view — its body carries
// the finding's "why" and suggested fix, so the PR summary comment can stay a
// terse list that points here. See
// https://docs.github.com/actions/using-workflows/workflow-commands-for-github-actions
func emitAnnotations(w io.Writer, findings []Finding) {
	for _, f := range findings {
		level := severityLevel(f.Severity)
		title := "Doc accuracy"
		if sev := strings.ToUpper(strings.TrimSpace(f.Severity)); sev != "" {
			title += " — " + sev
		}
		if f.Line > 0 {
			fmt.Fprintf(w, "::%s file=%s,line=%d,title=%s::%s\n",
				level, escapeProp(f.File), f.Line, escapeProp(title), annotationBody(f))
		} else {
			fmt.Fprintf(w, "::%s file=%s,title=%s::%s\n",
				level, escapeProp(f.File), escapeProp(title), annotationBody(f))
		}
	}
}

// annotationBody is the escaped message for one annotation: the summary, then a
// "Why:" line (details) and a "Fix:" line when the finding carries them. The
// three render as separate lines in the annotation box; escapeData turns the
// newlines into the %0A the workflow-command format requires.
func annotationBody(f Finding) string {
	msg := f.Summary
	if d := strings.TrimSpace(f.Details); d != "" {
		msg += "\nWhy: " + d
	}
	if fix := strings.TrimSpace(f.Fix); fix != "" {
		msg += "\nFix: " + fix
	}
	return escapeData(msg)
}

// escapeData applies GitHub's workflow-command escaping to a message so newlines
// and percent signs can't break the command.
func escapeData(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// escapeProp escapes a workflow-command property value, which additionally may
// not contain a literal ':' or ','.
func escapeProp(s string) string {
	s = escapeData(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

// maybeFetchBase best-effort refreshes a remote-tracking base (e.g.
// "upstream/main") so the changed set reflects the real mainline, not a stale
// local snapshot. Non-remote bases (HEAD, a local branch, a SHA) are left alone.
// A failed fetch warns and returns; the caller proceeds with what exists.
func maybeFetchBase(ws, base string) {
	remotes, err := git(ws, "remote")
	if err != nil {
		return
	}
	remote, branch, ok := parseRemoteRef(base, strings.Fields(remotes))
	if !ok {
		return
	}
	fmt.Printf("==> Refreshing base %s (git fetch %s %s)\n", base, remote, branch)
	if _, err := git(ws, "fetch", "--quiet", remote, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not refresh %s (%v); using the last-known snapshot.\n", base, err)
	}
}

// parseRemoteRef splits a base like "upstream/main" into remote and branch when
// its first segment is one of remotes. Returns ok=false for HEAD, a bare local
// branch, a SHA, or any base whose first segment isn't a configured remote.
func parseRemoteRef(base string, remotes []string) (remote, branch string, ok bool) {
	i := strings.Index(base, "/")
	if i <= 0 || i == len(base)-1 {
		return "", "", false
	}
	remote, branch = base[:i], base[i+1:]
	for _, r := range remotes {
		if r == remote {
			return remote, branch, true
		}
	}
	return "", "", false
}

// modeName is the human label for the review mode.
func modeName(all bool) string {
	if all {
		return "all"
	}
	return "changed"
}

// diffBase resolves the ref that changed mode diffs against. It returns the
// merge-base of base and HEAD — the point this branch forked from base — so the
// changed set is only what the branch introduced (plus uncommitted edits), and
// stays correct even when the branch is behind base (e.g. main moved ahead while
// you worked). Falls back to base itself when there is no common ancestor
// (unrelated histories) or the merge-base can't be resolved. When base is HEAD,
// merge-base is HEAD, so the "uncommitted only" behaviour is unchanged.
func diffBase(ws, base string) string {
	out, err := git(ws, "merge-base", base, "HEAD")
	if err != nil {
		return base
	}
	if mb := strings.TrimSpace(out); mb != "" {
		return mb
	}
	return base
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

	changed, err := git(ws, "diff", "--name-only", "--diff-filter=AMR", base, "--", "public")
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
// each must end in .mdx, stay inside the workspace, and exist. Paths are
// de-duplicated and sorted.
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
		clean := filepath.Clean(p)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("path must be inside the workspace: %s", p)
		}
		if _, err := os.Stat(filepath.Join(ws, clean)); err != nil {
			return nil, fmt.Errorf("file not found: %s", p)
		}
		seen[p] = true
		files = append(files, clean)
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

// collectDiffFile reads a unified diff from path and returns the changed .mdx
// files and, keyed by file, that file's slice of the diff. It is the diff-file
// (CI) equivalent of collectFiles: the changed set and the diff both come from
// the supplied patch, so no local `git diff` — and no checkout of the change —
// is needed. Non-.mdx sections and deletions (whose new side is /dev/null) are
// dropped: there is no proposed document to review.
func collectDiffFile(path string) (files []string, sectionByFile map[string]string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	sectionByFile = map[string]string{}
	for _, s := range splitDiffByFile(string(raw)) {
		if s.file == "" || !strings.HasSuffix(s.file, ".mdx") {
			continue
		}
		// A file touched twice in one diff (unusual, but possible) accumulates
		// both sections so no hunk is lost.
		sectionByFile[s.file] += s.text
	}
	for f := range sectionByFile {
		files = append(files, f)
	}
	sort.Strings(files)
	return files, sectionByFile, nil
}

// diffSection is one file's portion of a unified diff: the repo-relative path of
// its new (+++) side, and the raw text of that section including its `diff --git`
// header. file is "" for a deletion (new side is /dev/null) or a section with no
// +++ line.
type diffSection struct {
	file string
	text string
}

// splitDiffByFile splits a unified diff into per-file sections, one per
// `diff --git` header. Any preamble before the first header is ignored.
//
// The `+++ ` file header is only read *before* the section's first `@@` hunk.
// Once inside the hunk body, a `+++ `-prefixed line is content, not a header —
// an added documentation line whose own text starts with "++ " renders as
// "+++ ..." in the diff, and must not be mistaken for the file path (doing so
// would silently drop the file from review).
func splitDiffByFile(diff string) []diffSection {
	var sections []diffSection
	var cur *diffSection
	sawHunk := false
	flush := func() {
		if cur != nil {
			sections = append(sections, *cur)
			cur = nil
		}
	}
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			cur = &diffSection{}
			sawHunk = false
		}
		if cur == nil {
			continue // text before the first file header — not part of any section
		}
		if strings.HasPrefix(line, "@@ ") {
			sawHunk = true
		}
		if !sawHunk && strings.HasPrefix(line, "+++ ") {
			cur.file = parseDiffTarget(line)
		}
		cur.text += line + "\n"
	}
	flush()
	return sections
}

// parseDiffTarget extracts the repo-relative path from a unified-diff `+++` line
// (e.g. "+++ b/public/x.mdx" → "public/x.mdx"). It strips the conventional "b/"
// prefix and any trailing tab-separated timestamp, and returns "" for /dev/null
// (a deletion).
func parseDiffTarget(line string) string {
	p := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
	if i := strings.IndexByte(p, '\t'); i >= 0 {
		p = p[:i] // drop a trailing "\t<timestamp>" some diffs append
	}
	if p == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(p, "b/")
}

// buildPrompt assembles the full reviewer prompt: the embedded instructions
// followed by the dynamic context for this run (mode, file list, any newly
// added files, and — when includeDiff is set — the diff).
//
// When dataOnly is set (diff-file mode) the proposed documentation is present
// only in the diff below, not on disk: the working tree is the base branch, so
// the reviewer is told to review the changes from the diff and never to read
// these files from disk (reading them would show the pre-change base version).
// It may still Read/Grep the rest of the repo for trusted source to ground the
// review.
func buildPrompt(mode, base string, includeDiff, dataOnly bool, files, newFiles []string, diff string) string {
	var b strings.Builder
	b.WriteString(reviewerPrompt)
	b.WriteString("\n\n---\n\n## This review\n\n")
	fmt.Fprintf(&b, "Mode: **%s**\n\n", mode)
	if dataOnly {
		b.WriteString("These files changed in this pull request. Their **proposed** content is\n")
		b.WriteString("given entirely by the diff below — the working tree holds the *base*\n")
		b.WriteString("branch, not this change, so **do not read these files from disk**; that\n")
		b.WriteString("would show the old version. Review the changes from the diff. You may\n")
		b.WriteString("still Read/Grep the rest of the repository for trusted source to ground\n")
		b.WriteString("your review.\n\n")
		b.WriteString("Files changed:\n\n")
	} else {
		b.WriteString("Review these files (read each one):\n\n")
	}
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

// truncateLines returns at most limit lines of s, and says so when it cuts.
//
// The marker matters: in diff-file mode the reviewer is told *not* to read the
// changed files from disk (the working tree is the base), so a silently
// shortened diff would leave it reviewing less than the change while reporting
// as though it had seen all of it. The marker is inside the fenced diff, so the
// reviewer sees the gap and can flag it in its report.
func truncateLines(s string, limit int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= limit {
		return s
	}
	return strings.Join(lines[:limit], "\n") + fmt.Sprintf(
		"\n... [diff truncated: %d more line(s) of this batch omitted]\n"+
			"... Review only what is shown above, and say in your report that the\n"+
			"... diff was truncated and part of the change was not reviewed.",
		len(lines)-limit)
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
// assistant messages (and their text blocks), plus the final result event whose
// `result` field carries the final message text (a fallback so the verdict is
// captured even if the assistant text blocks were missed) and whose status
// fields explain a run that ended without a review (e.g. hitting a turn limit).
type streamEvent struct {
	Type       string `json:"type"`
	Result     string `json:"result"`
	Subtype    string `json:"subtype"`
	IsError    bool   `json:"is_error"`
	NumTurns   int    `json:"num_turns"`
	StopReason string `json:"stop_reason"`
	Message    struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

// reviewerTools is the set of built-in tools that *exist* in the session (via
// `--tools`). `git`, Bash, Edit, and Write are intentionally absent: the diff is
// inlined into the prompt, and their absence is what makes the tool read-only by
// construction. WebFetch exists but is domain-scoped for egress (see below).
var reviewerTools = []string{"Read", "Grep", "Glob", "WebFetch"}

// allowedWebFetchDomains are the only hosts WebFetch may reach. They are the
// upstream GitHub hosts the reviewer grounds against (source, raw files, and the
// large-file object store raw redirects to). A fetch to any other host is denied,
// not aborted — see runClaude. This is the egress boundary: injected content
// cannot make the reviewer POST a secret to an attacker's server, because the
// tool can only reach GitHub, which does not expose fetch logs to leak it back.
var allowedWebFetchDomains = []string{
	"github.com",
	"raw.githubusercontent.com",
	"objects.githubusercontent.com",
}

// reviewerAllowedTools is the permission allowlist passed to `--allowedTools`:
// the three local read tools, plus WebFetch scoped to each allowed domain. Under
// the default permission mode this list is authoritative — a call to anything
// else (a WebFetch to another host) is denied and returned to the model as a
// tool error, and the run continues.
func reviewerAllowedTools() []string {
	allowed := []string{"Read", "Grep", "Glob"}
	for _, d := range allowedWebFetchDomains {
		allowed = append(allowed, "WebFetch(domain:"+d+")")
	}
	return allowed
}

// runClaude drives the claude CLI headless as a read-only reviewer: it may read
// the repo and fetch the upstream GitHub repos, but nothing else. The prompt is
// fed on stdin. It is bounded by claudeTimeout so a stalled `claude` can't hang forever.
//
// Read-only, and egress-scoped, by *construction*:
//   - `--tools` limits the built-in tools that exist to Read/Grep/Glob/WebFetch,
//     so Bash, Edit, Write, etc. simply do not exist in the session — there is
//     no shell and nothing that can modify a file.
//   - `--allowedTools` (with the default permission mode) scopes what those
//     tools may do: Read/Grep/Glob are allowed, and WebFetch is allowed only for
//     allowedWebFetchDomains. A WebFetch to any other host is *denied, not
//     aborted* — the model receives a tool error and continues to a verdict.
//     (Verified against this CLI: an out-of-scope fetch returns permission_denied
//     and the headless run still completes. bypassPermissions is deliberately
//     NOT used — it would ignore the allowlist and re-open unrestricted egress.)
//   - `--strict-mcp-config` (with no --mcp-config) drops every MCP server, so
//     the runner's own connectors (e.g. a Google Drive integration) aren't
//     reachable either.
//
// Why the egress scope matters: `ANTHROPIC_API_KEY` is in this process's
// environment and the claude child inherits it, and Read can open absolute paths
// (e.g. /proc/self/environ). Without an egress limit, injected doc content could
// in principle make the reviewer read the key and WebFetch it to an attacker.
// Scoping WebFetch to GitHub closes that: the only reachable hosts don't hand an
// attacker the fetched data back. This does not rely on the model declining to
// comply — the capability to reach an arbitrary host is simply absent. (Note the
// separate connection from the claude CLI to the Anthropic API is unaffected;
// that is how the model runs, and the model cannot redirect it.)
//
// It uses stream-json output and writes every assistant *text* block to out.
// Plain `--output-format text` returns only the model's final message, which
// the model sometimes reduces to just the verdict — discarding the findings it
// narrated in earlier messages. Reassembling the text blocks captures the whole
// review regardless of how the model split it across messages.
func runClaude(ws, model, prompt string, out io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), claudeTimeout)
	defer cancel()

	args := []string{"-p"}
	if model != "" {
		args = append(args, "--model", model)
	}
	// --tools bounds which tools exist; --allowedTools bounds what they may do
	// (WebFetch is scoped to GitHub). No --permission-mode: the default mode
	// honours the allowlist and, headless, denies anything outside it instead of
	// prompting — bypassPermissions would ignore the allowlist entirely.
	args = append(args,
		"--output-format", "stream-json", "--verbose",
		"--strict-mcp-config",
		"--tools",
	)
	args = append(args, reviewerTools...)
	args = append(args, "--allowedTools")
	args = append(args, reviewerAllowedTools()...)

	cmd := exec.CommandContext(ctx, "claude", args...)
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

	// When DOC_ACCURACY_RAW is set, tee the raw event stream to that file so an
	// empty or truncated review can be diagnosed without guessing.
	var rawSink io.Writer = io.Discard
	if p := os.Getenv("DOC_ACCURACY_RAW"); p != "" {
		if f, err := os.Create(p); err == nil {
			defer func() { _ = f.Close() }()
			rawSink = f
		}
	}

	// json.Decoder reads one JSON value at a time regardless of line length, so
	// a large single-line message can't overflow a fixed scanner buffer.
	dec := json.NewDecoder(io.TeeReader(stdout, rawSink))
	wroteText := false
	finalResult := ""
	var meta streamEvent // the final result event, for status if no review came back
	for {
		var ev streamEvent
		if err := dec.Decode(&ev); err != nil {
			break // io.EOF, or a malformed event: stop parsing, drain below.
		}
		switch ev.Type {
		case "assistant":
			for _, block := range ev.Message.Content {
				if block.Type == "text" && block.Text != "" {
					fmt.Fprintln(out, block.Text)
					wroteText = true
				}
			}
		case "result":
			finalResult = ev.Result
			meta = ev
		}
	}

	// Fallback: if no assistant text was captured (e.g. the model produced only a
	// terse final message), fall back to the result event's text so the verdict
	// isn't lost.
	if !wroteText && finalResult != "" {
		fmt.Fprintln(out, finalResult)
	}

	// Drain anything left unparsed so claude never blocks writing to a full pipe.
	_, _ = io.Copy(io.Discard, dec.Buffered())
	_, _ = io.Copy(io.Discard, stdout)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("claude timed out after %s", claudeTimeout)
		}
		return err
	}

	// The run finished cleanly but produced no review at all — surface the
	// result event's status (e.g. a turn limit) instead of a silent empty report.
	if !wroteText && finalResult == "" {
		reason := firstNonEmpty(meta.Subtype, meta.StopReason, "unknown reason")
		return fmt.Errorf("claude produced no review output (result: %s, is_error=%v, turns=%d); "+
			"set DOC_ACCURACY_RAW=<path> to capture the raw event stream", reason, meta.IsError, meta.NumTurns)
	}
	return nil
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
