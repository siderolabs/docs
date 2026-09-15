package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterMDXFiltersAndSorts(t *testing.T) {
	changed := "public/omni/b.mdx\npublic/omni/a.mdx\npublic/docs.json\n"
	untracked := "public/omni/c.mdx\nREADME.md\n"

	got := filterMDX(changed, untracked)
	want := []string{"public/omni/a.mdx", "public/omni/b.mdx", "public/omni/c.mdx"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFilterMDXDeduplicates(t *testing.T) {
	// A file that is both "modified" and reported again should appear once.
	got := filterMDX("public/a.mdx\n", "public/a.mdx\n")
	if len(got) != 1 || got[0] != "public/a.mdx" {
		t.Fatalf("expected a single deduplicated entry, got %v", got)
	}
}

func TestFilterMDXIgnoresBlankAndNonMDX(t *testing.T) {
	got := filterMDX("\n  \npublic/a.md\npublic/a.mdxx\npublic/real.mdx\n")
	if len(got) != 1 || got[0] != "public/real.mdx" {
		t.Fatalf("expected only public/real.mdx, got %v", got)
	}
}

func TestParseVerdict(t *testing.T) {
	cases := map[string]string{
		"nothing here":                          "",
		"DOC_ACCURACY_VERDICT: PASS":            "PASS",
		"report...\nDOC_ACCURACY_VERDICT: FAIL": "FAIL",
		"  DOC_ACCURACY_VERDICT: PASS  ":        "PASS", // surrounding whitespace tolerated
		"DOC_ACCURACY_VERDICT: MAYBE":           "",     // unknown value ignored
		"`DOC_ACCURACY_VERDICT: FAIL`":          "FAIL", // wrapped in inline-code backticks
		"**DOC_ACCURACY_VERDICT: PASS**":        "PASS", // wrapped in bold
		"DOC_ACCURACY_VERDICT:FAIL":             "FAIL", // no space after the colon
		"DOC_ACCURACY_VERDICT: FAILURE":         "",     // only exact PASS/FAIL count
	}
	for in, want := range cases {
		if got := parseVerdict(in); got != want {
			t.Errorf("parseVerdict(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseVerdictLastWins(t *testing.T) {
	// If the model somehow prints two verdicts, the final one is authoritative.
	out := "DOC_ACCURACY_VERDICT: PASS\nmore text\nDOC_ACCURACY_VERDICT: FAIL\n"
	if got := parseVerdict(out); got != "FAIL" {
		t.Fatalf("expected FAIL to win, got %q", got)
	}
}

func TestTruncateLines(t *testing.T) {
	in := "a\nb\nc\nd"
	if got := truncateLines(in, 10); got != in {
		t.Fatalf("truncateLines above length should be unchanged, got %q", got)
	}
	got := truncateLines(in, 2)
	if !strings.HasPrefix(got, "a\nb\n") {
		t.Fatalf("truncateLines cap 2 must keep the first 2 lines, got %q", got)
	}
	// A silent cut would have the reviewer report on a diff it never saw in
	// full; the cut must be visible in the text the model reads, with the count.
	if !strings.Contains(got, "truncated") || !strings.Contains(got, "2 more line(s)") {
		t.Errorf("truncation must be marked with the omitted line count, got %q", got)
	}
}

func TestRunSkipsWhenOverMaxFiles(t *testing.T) {
	ws := t.TempDir()
	var diff strings.Builder
	for _, name := range []string{"a", "b", "c"} {
		f := "public/omni/" + name + ".mdx"
		diff.WriteString("diff --git a/" + f + " b/" + f + "\n")
		diff.WriteString("--- a/" + f + "\n+++ b/" + f + "\n")
		diff.WriteString("@@ -1 +1 @@\n-old\n+new\n")
	}
	diffPath := filepath.Join(ws, "pr.diff")
	if err := os.WriteFile(diffPath, []byte(diff.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	// 3 changed files against a cap of 2: nothing is reviewed (no `claude` is
	// even looked up), and the run still exits 0 — this check is advisory.
	code := run(runConfig{
		workspace:   ws,
		diffFile:    diffPath,
		maxFiles:    2,
		report:      "_out/report.md",
		findingsOut: "_out/findings.json",
	})
	if code != 0 {
		t.Errorf("a skipped review must not fail the check, got exit %d", code)
	}

	data, err := os.ReadFile(filepath.Join(ws, "_out/findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got findingsReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	// SKIPPED, never PASS: the CI comment must be able to tell "reviewed and
	// found nothing" apart from "too large, not reviewed".
	if got.Verdict != "SKIPPED" {
		t.Errorf("verdict = %q, want SKIPPED", got.Verdict)
	}
	if !strings.Contains(got.Reason, "3") || !strings.Contains(got.Reason, "2") {
		t.Errorf("reason should name the file count and the cap, got %q", got.Reason)
	}
	if len(got.Findings) != 0 {
		t.Errorf("a skipped run has no findings, got %d", len(got.Findings))
	}
}

func TestBuildPromptChangedIncludesDiffAndFiles(t *testing.T) {
	p := buildPrompt("changed", "origin/main", true, false, []string{"public/omni/x.mdx"}, nil, "- old\n+ new")

	// The embedded reviewer instructions must lead the prompt.
	if !strings.Contains(p, "Documentation accuracy reviewer") {
		t.Error("prompt is missing the embedded reviewer instructions")
	}
	for _, want := range []string{
		"Mode: **changed**",
		"- public/omni/x.mdx",
		"base: `origin/main`",
		"````diff", // four backticks so .mdx code fences don't close the block
		"+ new",
	} {
		if !strings.Contains(p, want) {
			t.Errorf("changed-mode prompt missing %q", want)
		}
	}
}

func TestBuildPromptDiffFenceSurvivesInnerFences(t *testing.T) {
	// A real .mdx diff contains three-backtick code fences; the outer fence must
	// use four backticks so the inner ones don't terminate the diff block.
	diff := " ```bash\n-docker run --rm foo\n+docker run foo\n ```"
	p := buildPrompt("changed", "HEAD", true, false, []string{"public/x.mdx"}, nil, diff)
	if !strings.Contains(p, "````diff\n") || !strings.Contains(p, "\n````\n") {
		t.Error("expected the diff to be wrapped in a four-backtick fence")
	}
}

func TestBuildPromptListsNewFiles(t *testing.T) {
	p := buildPrompt("changed", "HEAD", true, false, []string{"public/new.mdx"}, []string{"public/new.mdx"}, "")
	if !strings.Contains(p, "newly added files") {
		t.Error("expected a note calling out newly added files")
	}
}

func TestBuildPromptOmitsDiffWhenNotIncluded(t *testing.T) {
	p := buildPrompt("all", "HEAD", false, false, []string{"public/omni/x.mdx"}, nil, "")
	if !strings.Contains(p, "Mode: **all**") {
		t.Error("all-mode prompt should say Mode: **all**")
	}
	if strings.Contains(p, "````diff\n") {
		t.Error("prompt without includeDiff should not contain a diff block")
	}
}

func TestExplicitFiles(t *testing.T) {
	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws, "public/a.mdx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := explicitFiles(ws, []string{"public/a.mdx", "public/a.mdx"}) // dupe collapses
	if err != nil || len(got) != 1 || got[0] != "public/a.mdx" {
		t.Fatalf("expected [public/a.mdx], got %v (err %v)", got, err)
	}
	if _, err := explicitFiles(ws, []string{"public/a.txt"}); err == nil {
		t.Error("expected an error for a non-.mdx path")
	}
	if _, err := explicitFiles(ws, []string{"public/missing.mdx"}); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestExplicitFilesRejectsTraversal(t *testing.T) {
	ws := t.TempDir()
	// A path escaping the workspace must be rejected even if it ends in .mdx.
	if _, err := explicitFiles(ws, []string{"../secret.mdx"}); err == nil {
		t.Error("expected an error for a parent-escaping path")
	}
	if _, err := explicitFiles(ws, []string{"/etc/passwd.mdx"}); err == nil {
		t.Error("expected an error for an absolute path")
	}
}

func TestChunk(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e"}
	got := chunk(in, 2)
	if len(got) != 3 || len(got[0]) != 2 || len(got[2]) != 1 || got[2][0] != "e" {
		t.Fatalf("chunk(size 2) = %v, want [[a b] [c d] [e]]", got)
	}
	// Every element is covered exactly once, in order.
	var flat []string
	for _, c := range got {
		flat = append(flat, c...)
	}
	if strings.Join(flat, "") != "abcde" {
		t.Fatalf("chunk lost or reordered elements: %v", got)
	}
	// A size below 1 is clamped rather than looping forever.
	if got := chunk([]string{"x"}, 0); len(got) != 1 || got[0][0] != "x" {
		t.Fatalf("chunk with size 0 = %v, want [[x]]", got)
	}
}

func TestModeName(t *testing.T) {
	if modeName(true) != "all" || modeName(false) != "changed" {
		t.Fatal("modeName mapping is wrong")
	}
}

// sampleDiff is a unified diff touching two .mdx files (one modified, one added),
// a non-.mdx file, and a deleted .mdx — enough to exercise every filter in
// collectDiffFile.
const sampleDiff = `diff --git a/public/omni/a.mdx b/public/omni/a.mdx
index 1111111..2222222 100644
--- a/public/omni/a.mdx
+++ b/public/omni/a.mdx
@@ -1,3 +1,3 @@
 intro
-old line
+new line
diff --git a/public/omni/b.mdx b/public/omni/b.mdx
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/public/omni/b.mdx
@@ -0,0 +1,2 @@
+brand new
+content
diff --git a/public/docs.json b/public/docs.json
index 4444444..5555555 100644
--- a/public/docs.json
+++ b/public/docs.json
@@ -1 +1 @@
-{}
+{"x":1}
diff --git a/public/omni/gone.mdx b/public/omni/gone.mdx
deleted file mode 100644
index 6666666..0000000
--- a/public/omni/gone.mdx
+++ /dev/null
@@ -1 +0,0 @@
-was here
`

func TestCollectDiffFile(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "pr.diff")
	if err := os.WriteFile(path, []byte(sampleDiff), 0o644); err != nil {
		t.Fatal(err)
	}

	files, sections, err := collectDiffFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Only the modified and added .mdx files: docs.json is not .mdx, and the
	// deleted file (new side /dev/null) has nothing to review.
	want := []string{"public/omni/a.mdx", "public/omni/b.mdx"}
	if len(files) != len(want) || files[0] != want[0] || files[1] != want[1] {
		t.Fatalf("collectDiffFile files = %v, want %v", files, want)
	}
	// Each file's section carries its own header and changed lines, nothing else.
	if !strings.Contains(sections["public/omni/a.mdx"], "+new line") ||
		strings.Contains(sections["public/omni/a.mdx"], "brand new") {
		t.Errorf("section for a.mdx is wrong:\n%s", sections["public/omni/a.mdx"])
	}
	if !strings.Contains(sections["public/omni/b.mdx"], "+brand new") {
		t.Errorf("section for b.mdx is missing its added content:\n%s", sections["public/omni/b.mdx"])
	}
}

func TestCollectDiffFileIgnoresPlusPlusPlusInHunk(t *testing.T) {
	// An added documentation line whose text starts with "++ " renders in the
	// diff as "+++ ...". Inside the hunk body it must be treated as content, not
	// as a file header — otherwise the file is silently dropped from review.
	diff := "diff --git a/public/omni/a.mdx b/public/omni/a.mdx\n" +
		"--- a/public/omni/a.mdx\n" +
		"+++ b/public/omni/a.mdx\n" +
		"@@ -1,2 +1,3 @@\n" +
		" intro\n" +
		"+++ this added line begins with a plus-plus\n" +
		"+real content\n"
	ws := t.TempDir()
	path := filepath.Join(ws, "pr.diff")
	if err := os.WriteFile(path, []byte(diff), 0o644); err != nil {
		t.Fatal(err)
	}
	files, _, err := collectDiffFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "public/omni/a.mdx" {
		t.Fatalf("expected the real file header to win, got %v", files)
	}
}

func TestOverallVerdict(t *testing.T) {
	cases := []struct {
		name                                    string
		anyFail, anyCritical, notFinish, anyGap bool
		want                                    string
	}{
		{"clean completed run", false, false, false, false, "PASS"},
		{"critical from verdict line", true, false, false, false, "FAIL"},
		{"critical from parsed finding", false, true, false, false, "FAIL"},
		{"claude crashed mid-run", false, false, true, false, "INCOMPLETE"},
		{"no verdict line produced", false, false, false, true, "INCOMPLETE"},
		// A confirmed critical wins even if a later batch also failed to finish —
		// the critical is real and actionable, and FAIL is not a clean result.
		{"critical plus a crash", true, false, true, false, "FAIL"},
	}
	for _, c := range cases {
		if got := overallVerdict(c.anyFail, c.anyCritical, c.notFinish, c.anyGap); got != c.want {
			t.Errorf("%s: overallVerdict = %q, want %q", c.name, got, c.want)
		}
	}
	// The load-bearing guarantee: PASS is impossible unless the run truly
	// completed clean — any incompleteness downgrades it to INCOMPLETE, never PASS.
	if overallVerdict(false, false, true, false) == "PASS" {
		t.Fatal("a run that did not finish must never be reported as PASS")
	}
}

func TestReviewerAllowedToolsScopesWebFetch(t *testing.T) {
	has := func(tools []string, name string) bool {
		for _, x := range tools {
			if x == name {
				return true
			}
		}
		return false
	}
	allowed := reviewerAllowedTools()

	// The local read tools are allowed as-is.
	for _, name := range []string{"Read", "Grep", "Glob"} {
		if !has(allowed, name) {
			t.Errorf("%s must be in the allowlist", name)
		}
	}
	// WebFetch is allowed only domain-scoped — never as a bare, any-host entry.
	if has(allowed, "WebFetch") {
		t.Error("a bare, unscoped WebFetch must never be in the allowlist — that re-opens arbitrary egress")
	}
	for _, d := range allowedWebFetchDomains {
		if !has(allowed, "WebFetch(domain:"+d+")") {
			t.Errorf("WebFetch must be scoped to %s", d)
		}
	}
	// No write/exec tool ever appears, in either list.
	for _, banned := range []string{"Bash", "Edit", "Write"} {
		if has(allowed, banned) || has(reviewerTools, banned) {
			t.Errorf("%s must never be available to the reviewer", banned)
		}
	}
}

func TestParseDiffTarget(t *testing.T) {
	cases := map[string]string{
		"+++ b/public/omni/a.mdx":       "public/omni/a.mdx",
		"+++ b/public/omni/a.mdx\t2024": "public/omni/a.mdx", // trailing timestamp dropped
		"+++ /dev/null":                 "",                  // deletion
		"+++ public/no-b-prefix.mdx":    "public/no-b-prefix.mdx",
	}
	for in, want := range cases {
		if got := parseDiffTarget(in); got != want {
			t.Errorf("parseDiffTarget(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildPromptDataOnlyForbidsDiskRead(t *testing.T) {
	p := buildPrompt("changed", "origin/main", true, true, []string{"public/omni/a.mdx"}, nil, "+new line")
	if !strings.Contains(p, "do not read these files from disk") {
		t.Error("data-only prompt must tell the reviewer not to read the changed files from disk")
	}
	if strings.Contains(p, "read each one") {
		t.Error("data-only prompt must not use the disk-read instruction")
	}
	// The diff is still the review target.
	if !strings.Contains(p, "+new line") {
		t.Error("data-only prompt should still include the diff")
	}
}

func TestParseFindings(t *testing.T) {
	out := "some human report...\n\n" +
		"DOC_ACCURACY_FINDINGS\n" +
		"```json\n" +
		`[{"file":"public/a.mdx","line":42,"severity":"CRITICAL","summary":"rm -rf / wipes the host"},` +
		`{"file":"public/b.mdx","line":0,"severity":"NOTICE","summary":"binds to 0.0.0.0 in a local example"}]` +
		"\n```\n" +
		"DOC_ACCURACY_VERDICT: FAIL\n"

	got := parseFindings(out)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d (%v)", len(got), got)
	}
	if got[0].File != "public/a.mdx" || got[0].Line != 42 || got[0].Severity != "CRITICAL" {
		t.Errorf("first finding wrong: %+v", got[0])
	}
	if got[1].Severity != "NOTICE" || got[1].Line != 0 {
		t.Errorf("second finding wrong: %+v", got[1])
	}
}

func TestParseFindingsMissingOrMalformed(t *testing.T) {
	if got := parseFindings("no marker here"); got != nil {
		t.Errorf("expected nil when marker absent, got %v", got)
	}
	// Marker present but the fenced block isn't valid JSON.
	bad := "DOC_ACCURACY_FINDINGS\n```json\nnot json\n```\n"
	if got := parseFindings(bad); got != nil {
		t.Errorf("expected nil for malformed JSON, got %v", got)
	}
	// Entries missing file/summary are dropped.
	partial := "DOC_ACCURACY_FINDINGS\n```json\n" +
		`[{"file":"","line":1,"severity":"WARNING","summary":"x"},{"file":"public/a.mdx","line":1,"severity":"WARNING","summary":""}]` +
		"\n```\n"
	if got := parseFindings(partial); got != nil {
		t.Errorf("expected incomplete entries dropped to nil, got %v", got)
	}
}

func TestSeverityLevel(t *testing.T) {
	cases := map[string]string{
		"CRITICAL": "error", "critical": "error",
		"WARNING": "warning", " warning ": "warning",
		"NOTICE": "notice", "": "notice", "weird": "notice",
	}
	for in, want := range cases {
		if got := severityLevel(in); got != want {
			t.Errorf("severityLevel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmitAnnotations(t *testing.T) {
	var b strings.Builder
	emitAnnotations(&b, []Finding{
		{File: "public/a.mdx", Line: 42, Severity: "CRITICAL", Summary: "boom",
			Details: "etcd data is lost on recreate", Fix: "add -v /var/lib/etcd:/var/lib/etcd"},
		{File: "public/b.mdx", Line: 0, Severity: "NOTICE", Summary: "minor, note this"},
	})
	got := b.String()
	// The title carries the severity; the body is summary + escaped Why/Fix lines.
	if !strings.Contains(got, "::error file=public/a.mdx,line=42,title=Doc accuracy — CRITICAL::boom%0AWhy: etcd data is lost on recreate%0AFix: add -v /var/lib/etcd:/var/lib/etcd") {
		t.Errorf("missing detailed critical annotation, got:\n%s", got)
	}
	// Line 0 omits the line property; no details/fix means the body is just the summary.
	if !strings.Contains(got, "::notice file=public/b.mdx,title=Doc accuracy — NOTICE::minor, note this") {
		t.Errorf("missing lineless notice annotation, got:\n%s", got)
	}
}

func TestAnnotationBody(t *testing.T) {
	// Summary only.
	if got := annotationBody(Finding{Summary: "just this"}); got != "just this" {
		t.Errorf("summary-only body = %q", got)
	}
	// Details and fix become escaped Why/Fix lines.
	got := annotationBody(Finding{Summary: "s", Details: "d", Fix: "f"})
	if got != "s%0AWhy: d%0AFix: f" {
		t.Errorf("full body = %q", got)
	}
	// Blank fix is dropped, not shown as an empty line.
	if got := annotationBody(Finding{Summary: "s", Fix: "   "}); got != "s" {
		t.Errorf("blank fix should be omitted, got %q", got)
	}
}

func TestEscapeData(t *testing.T) {
	if got := escapeData("a\nb%c\rd"); got != "a%0Ab%25c%0Dd" {
		t.Errorf("escapeData = %q", got)
	}
	if got := escapeProp("re/mote:main,x"); got != "re/mote%3Amain%2Cx" {
		t.Errorf("escapeProp = %q", got)
	}
}

func TestParseRemoteRef(t *testing.T) {
	remotes := []string{"origin", "upstream"}
	cases := []struct {
		base, remote, branch string
		ok                   bool
	}{
		{"upstream/main", "upstream", "main", true},
		{"origin/feature/x", "origin", "feature/x", true}, // only first slash splits
		{"HEAD", "", "", false},
		{"main", "", "", false},      // bare local branch
		{"fork/main", "", "", false}, // unknown remote
		{"deadbeef", "", "", false},  // a SHA
		{"origin/", "", "", false},   // trailing slash, no branch
		{"/main", "", "", false},     // leading slash
	}
	for _, c := range cases {
		r, b, ok := parseRemoteRef(c.base, remotes)
		if r != c.remote || b != c.branch || ok != c.ok {
			t.Errorf("parseRemoteRef(%q) = (%q,%q,%v), want (%q,%q,%v)",
				c.base, r, b, ok, c.remote, c.branch, c.ok)
		}
	}
}

func TestPrintSummaryKeepsFindingsOnOneLine(t *testing.T) {
	// A finding is model-authored text about attacker-controlled diff content.
	// In CI this summary goes to a stdout the runner scans for ::commands at the
	// start of a line, so an embedded newline must not survive into the output.
	var buf strings.Builder
	printSummary(&buf, []Finding{{
		File:     "public/omni/a.mdx",
		Line:     4,
		Severity: "WARNING",
		Summary:  "stale tag\n::stop-commands::hide\n",
		Fix:      "bump it\n::error file=x::forged",
	}})
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "::") {
			t.Errorf("a finding must not be able to start a line with '::', got %q", line)
		}
	}
	if !strings.Contains(buf.String(), "stale tag ::stop-commands::hide") {
		t.Errorf("the text should be kept, just flattened; got %q", buf.String())
	}
}

func TestSortedFindings(t *testing.T) {
	in := []Finding{
		{File: "b.mdx", Line: 5, Severity: "NOTICE", Summary: "n"},
		{File: "a.mdx", Line: 9, Severity: "CRITICAL", Summary: "c2"},
		{File: "a.mdx", Line: 2, Severity: "CRITICAL", Summary: "c1"},
		{File: "a.mdx", Line: 1, Severity: "WARNING", Summary: "w"},
	}
	got := sortedFindings(in)
	// CRITICALs first (by file then line), then WARNING, then NOTICE.
	wantOrder := []string{"c1", "c2", "w", "n"}
	for i, w := range wantOrder {
		if got[i].Summary != w {
			t.Fatalf("position %d = %q, want %q (full: %+v)", i, got[i].Summary, w, got)
		}
	}
	// Input slice must not be mutated.
	if in[0].Summary != "n" {
		t.Error("sortedFindings mutated its input")
	}
}

func TestDiffBaseUsesForkPointWhenBehind(t *testing.T) {
	ws := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		out, err := git(ws, args...)
		if err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
		return strings.TrimSpace(out)
	}
	commit := func(name, body string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(ws, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		run("add", "-A")
		run("-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false",
			"commit", "-q", "-m", "c")
		return run("rev-parse", "HEAD")
	}

	run("init", "-q", "-b", "main")
	fork := commit("base.txt", "base") // shared history: the fork point

	// main advances with its own commits (simulating work merged while the branch
	// was open), then the branch forks from `fork` and adds one commit.
	run("checkout", "-q", "-b", "feature", fork)
	commit("feature.txt", "branch work")

	run("checkout", "-q", "main")
	commit("main1.txt", "main work 1")
	commit("main2.txt", "main work 2")

	run("checkout", "-q", "feature")

	// The branch is now behind main. Diffing against main's tip would include
	// main1/main2 (not the branch's work); diffBase must return the fork point so
	// only feature.txt is in the diff.
	if got := diffBase(ws, "main"); got != fork {
		t.Fatalf("diffBase = %q, want fork point %q", got, fork)
	}
	changed, err := git(ws, "diff", "--name-only", diffBase(ws, "main"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(changed) != "feature.txt" {
		t.Errorf("expected only feature.txt changed vs fork point, got %q", changed)
	}

	// A non-existent base has no merge-base: fall back to the base string itself.
	if got := diffBase(ws, "no-such-ref"); got != "no-such-ref" {
		t.Errorf("expected fallback to base on merge-base failure, got %q", got)
	}
}

func TestWriteFindingsEmptyIsValidJSON(t *testing.T) {
	ws := t.TempDir()
	if err := writeFindings(ws, "_out/findings.json", "PASS", nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ws, "_out/findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got findingsReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("empty findings must still be valid JSON: %v (%q)", err, data)
	}
	if got.Verdict != "PASS" || len(got.Findings) != 0 {
		t.Errorf("expected {PASS, []}, got %+v", got)
	}
}

func TestWriteFindingsStoresGivenVerdict(t *testing.T) {
	ws := t.TempDir()
	findings := []Finding{{File: "public/a.mdx", Line: 1, Severity: "WARNING", Summary: "w"}}
	// The verdict is authoritative and passed in — even with no CRITICAL finding,
	// a FAIL from the reviewer's verdict line must be preserved (the case where a
	// malformed findings block parsed to fewer entries than the run actually found).
	if err := writeFindings(ws, "_out/findings.json", "FAIL", findings); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(ws, "_out/findings.json"))
	var got findingsReport
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Verdict != "FAIL" {
		t.Errorf("writeFindings must store the given verdict, got %q", got.Verdict)
	}
	if len(got.Findings) != 1 {
		t.Errorf("expected the finding preserved, got %d", len(got.Findings))
	}
}

func TestHasCritical(t *testing.T) {
	if hasCritical([]Finding{{Severity: "WARNING"}, {Severity: "NOTICE"}}) {
		t.Error("no CRITICAL present, want false")
	}
	if !hasCritical([]Finding{{Severity: "warning"}, {Severity: "critical"}}) {
		t.Error("a lower-case critical must still count (case-insensitive)")
	}
}
