package main

import (
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
	if got := truncateLines(in, 2); got != "a\nb" {
		t.Fatalf("truncateLines cap 2 = %q, want %q", got, "a\nb")
	}
	if got := truncateLines(in, 10); got != in {
		t.Fatalf("truncateLines above length should be unchanged, got %q", got)
	}
}

func TestBuildPromptChangedIncludesDiffAndFiles(t *testing.T) {
	p := buildPrompt("changed", "origin/main", true, []string{"public/omni/x.mdx"}, nil, "- old\n+ new")

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
	p := buildPrompt("changed", "HEAD", true, []string{"public/x.mdx"}, nil, diff)
	if !strings.Contains(p, "````diff\n") || !strings.Contains(p, "\n````\n") {
		t.Error("expected the diff to be wrapped in a four-backtick fence")
	}
}

func TestBuildPromptListsNewFiles(t *testing.T) {
	p := buildPrompt("changed", "HEAD", true, []string{"public/new.mdx"}, []string{"public/new.mdx"}, "")
	if !strings.Contains(p, "newly added files") {
		t.Error("expected a note calling out newly added files")
	}
}

func TestBuildPromptOmitsDiffWhenNotIncluded(t *testing.T) {
	p := buildPrompt("all", "HEAD", false, []string{"public/omni/x.mdx"}, nil, "")
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

func TestModeName(t *testing.T) {
	if modeName(true) != "all" || modeName(false) != "changed" {
		t.Fatal("modeName mapping is wrong")
	}
}
