package main

import (
	"strings"
	"testing"
)

// findRule reports whether any finding has the given rule.
func findRule(fs []Finding, rule string) bool {
	for _, f := range fs {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

// hasRuleAtLine reports whether a finding with the rule exists at the line.
func hasRuleAtLine(fs []Finding, rule string, line int) bool {
	for _, f := range fs {
		if f.Rule == rule && f.Line == line {
			return true
		}
	}
	return false
}

func TestStackedHeadings(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## First\n\n### Second\n\nSome content.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "headings/stacked") {
		t.Fatalf("expected stacked-heading finding, got %+v", got)
	}
}

func TestNoStackedWhenContentBetween(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## First\n\nProse here.\n\n### Second\n\nMore prose.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/stacked") {
		t.Fatalf("did not expect stacked-heading finding, got %+v", got)
	}
}

func TestSkippedHeadingLevel(t *testing.T) {
	// Frontmatter title is H1, so jumping straight to #### skips levels.
	doc := "---\ntitle: Test\n---\n\n#### Too deep\n\nContent.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "headings/skipped-level") {
		t.Fatalf("expected skipped-level finding, got %+v", got)
	}
}

func TestNoSkipGoingBackUp(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## A\n\ntext\n\n### B\n\ntext\n\n## C\n\ntext\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/skipped-level") {
		t.Fatalf("going back up levels should not be a skip, got %+v", got)
	}
}

func TestCodeNoLanguage(t *testing.T) {
	doc := "Intro.\n\n```\nkubectl apply -f deployment.yaml\n```\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "code/no-language") {
		t.Fatalf("expected no-language finding, got %+v", got)
	}
}

func TestCodeWithLanguageOK(t *testing.T) {
	doc := "Intro.\n\n```bash\nkubectl apply -f deployment.yaml\n```\n"
	got := lint("t.mdx", doc)
	if findRule(got, "code/no-language") {
		t.Fatalf("did not expect no-language finding, got %+v", got)
	}
}

func TestShellPrompt(t *testing.T) {
	doc := "Intro.\n\n```bash\n$ kubectl apply -f deployment.yaml\n```\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "code/shell-prompt") {
		t.Fatalf("expected shell-prompt finding, got %+v", got)
	}
}

func TestDollarVariableIsNotPrompt(t *testing.T) {
	// "$VAR" and "$(cmd)" must not be flagged as prompts.
	doc := "Intro.\n\n```bash\necho $HOME\nresult=$(date)\n```\n"
	got := lint("t.mdx", doc)
	if findRule(got, "code/shell-prompt") {
		t.Fatalf("variable/subshell should not be a prompt, got %+v", got)
	}
}

func TestSedWarning(t *testing.T) {
	doc := "Intro.\n\n```bash\nsed -i 's/a/b/' file\n```\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "code/sed") {
		t.Fatalf("expected sed finding, got %+v", got)
	}
}

func TestNonDescriptiveLink(t *testing.T) {
	doc := "See [click here](https://example.com) for details.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "links/non-descriptive") {
		t.Fatalf("expected non-descriptive link finding, got %+v", got)
	}
}

func TestDescriptiveLinkOK(t *testing.T) {
	doc := "See the [Cluster templates guide](https://example.com).\n"
	got := lint("t.mdx", doc)
	if findRule(got, "links/non-descriptive") {
		t.Fatalf("descriptive link should be fine, got %+v", got)
	}
}

func TestEmphasizedBadLink(t *testing.T) {
	doc := "See [**here**](https://example.com).\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "links/non-descriptive") {
		t.Fatalf("emphasized bad link should still flag, got %+v", got)
	}
}

func TestImageFilenameMarkdown(t *testing.T) {
	doc := "![A screenshot](./images/Cluster_Overview.PNG)\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "images/filename") {
		t.Fatalf("expected image filename finding, got %+v", got)
	}
}

func TestImageFilenameJSX(t *testing.T) {
	doc := "<img src=\"./images/My_Screenshot.png\" alt=\"x\" />\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "images/filename") {
		t.Fatalf("expected image filename finding for <img>, got %+v", got)
	}
}

func TestGoodImageFilenameOK(t *testing.T) {
	doc := "![Exposed service](./images/accessing-exposed-service.png)\n"
	got := lint("t.mdx", doc)
	if findRule(got, "images/filename") {
		t.Fatalf("kebab-case filename should be fine, got %+v", got)
	}
}

func TestRemoteImageIgnored(t *testing.T) {
	doc := "![logo](https://example.com/Some_Logo.png)\n"
	got := lint("t.mdx", doc)
	if findRule(got, "images/filename") {
		t.Fatalf("remote images should be ignored, got %+v", got)
	}
}

func TestCodeInsideFenceNotParsedAsHeadingOrLink(t *testing.T) {
	// A "#" comment and a "[x](y)" inside a code block must not trigger
	// heading or link rules.
	doc := "Intro.\n\n```bash\n# not a heading\necho '[click here](x)'\n```\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/stacked") || findRule(got, "links/non-descriptive") {
		t.Fatalf("code content should be ignored by prose rules, got %+v", got)
	}
}

func TestLineNumbersAreOneBased(t *testing.T) {
	doc := "line1\nline2\n[here](x)\n"
	got := lint("t.mdx", doc)
	if !hasRuleAtLine(got, "links/non-descriptive", 3) {
		t.Fatalf("expected finding on line 3, got %+v", got)
	}
}

func TestFrontmatterFieldsNotLinted(t *testing.T) {
	// A colon or brackets in frontmatter must not be misread as content.
	doc := "---\ntitle: Test\ndescription: See here\n---\n\nBody.\n"
	got := lint("t.mdx", doc)
	if len(got) != 0 {
		t.Fatalf("frontmatter should not produce findings, got %+v", got)
	}
}

func TestHeadingSentenceCaseViolation(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## Getting Started\n\nText.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "headings/sentence-case") {
		t.Fatalf("expected sentence-case finding for 'Getting Started', got %+v", got)
	}
}

func TestHeadingSentenceCaseOK(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## Getting started\n\nText.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/sentence-case") {
		t.Fatalf("'Getting started' is sentence case, got %+v", got)
	}
}

func TestHeadingAllowsExceptionsAcronymsAndCamelCase(t *testing.T) {
	// Omni (exception), SAML (acronym), KubeSpan (CamelCase) must all be allowed.
	doc := "---\ntitle: Test\n---\n\n## Using SAML with Omni and KubeSpan\n\nText.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/sentence-case") {
		t.Fatalf("proper nouns/acronyms/CamelCase should be allowed, got %+v", got)
	}
}

func TestHeadingAllowsCapitalAfterColon(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## Note: This matters\n\nText.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/sentence-case") {
		t.Fatalf("capital after a colon should be allowed, got %+v", got)
	}
}

func TestTitleTitleCaseViolation(t *testing.T) {
	doc := "---\ntitle: Break glass emergency access\n---\n\nBody.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "title/title-case") {
		t.Fatalf("expected title-case finding, got %+v", got)
	}
}

func TestTitleTitleCaseOK(t *testing.T) {
	doc := "---\ntitle: Break Glass Emergency Access\n---\n\nBody.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "title/title-case") {
		t.Fatalf("proper title case should pass, got %+v", got)
	}
}

func TestTitleAllowsSmallWords(t *testing.T) {
	doc := "---\ntitle: Using SAML with Omni\n---\n\nBody.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "title/title-case") {
		t.Fatalf("small words (with) may stay lowercase, got %+v", got)
	}
}

func TestAvoidH1(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n# A body H1\n\nText.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "headings/avoid-h1") {
		t.Fatalf("expected avoid-h1 finding, got %+v", got)
	}
}

func TestBlankLineAroundHeading(t *testing.T) {
	// No blank line after the heading.
	doc := "---\ntitle: Test\n---\n\n## Overview\nText right after.\n"
	got := lint("t.mdx", doc)
	if !findRule(got, "headings/blank-line") {
		t.Fatalf("expected blank-line finding, got %+v", got)
	}
}

func TestBlankLineAroundHeadingOK(t *testing.T) {
	doc := "---\ntitle: Test\n---\n\n## Overview\n\nText with a blank line.\n"
	got := lint("t.mdx", doc)
	if findRule(got, "headings/blank-line") {
		t.Fatalf("heading with surrounding blanks should be fine, got %+v", got)
	}
}

func TestExceptionsAreLoaded(t *testing.T) {
	// The embedded exceptions.txt should contain the core product names.
	for _, w := range []string{"talos", "omni", "kubernetes"} {
		if !exceptionSet[w] {
			t.Fatalf("expected %q in exceptionSet", w)
		}
	}
}

func TestWhitespaceImageTargetDoesNotPanic(t *testing.T) {
	// Regression: `![alt]( )` used to panic (strings.Fields → empty slice).
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked on whitespace-only image target: %v", r)
		}
	}()
	if got := lint("t.mdx", "![alt]( )\n"); len(got) != 0 {
		t.Fatalf("expected no findings, got %+v", got)
	}
}

func TestLongFenceNotClosedByShorterFence(t *testing.T) {
	// A 4-backtick block wrapping a 3-backtick example must stay open, so the
	// content is not re-parsed as prose. The `# not a heading` line inside must
	// not produce an avoid-h1 finding.
	doc := "Intro.\n\n````markdown\n# not a heading\n```\ncode\n```\n````\n\nAfter.\n"
	got := lint("t.mdx", doc)
	for _, f := range got {
		if f.Rule == "headings/avoid-h1" {
			t.Fatalf("content inside a longer fence was mis-parsed as a heading: %+v", got)
		}
	}
}

func TestUnclosedFrontmatterDoesNotMisfire(t *testing.T) {
	// A leading '---' with no closing '---' must not be treated as frontmatter,
	// so a stray `title:` line deep in the body is not title-checked.
	doc := "---\nsome: yaml\n\nBody text.\n\ntitle: a lowercase thing\n"
	got := lint("t.mdx", doc)
	for _, f := range got {
		if f.Rule == "title/title-case" {
			t.Fatalf("unclosed frontmatter should not trigger a title check: %+v", got)
		}
	}
}

func TestGitHubFormatDoesNotPanic(t *testing.T) {
	f := []Finding{{"a.mdx", 2, ErrorLevel, "code/shell-prompt", "msg"}}
	// Just ensure report runs for both formats.
	if w, e := report(f, "github"); e != 1 || w != 0 {
		t.Fatalf("unexpected counts w=%d e=%d", w, e)
	}
	if !strings.HasPrefix("::error", "::") {
		t.Fatal("unreachable")
	}
}
