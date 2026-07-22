package main

import (
	"strings"
	"testing"
)

func run(t *testing.T, in string, stripHR bool) string {
	t.Helper()
	return strings.Join(normalize(strings.Split(in, "\n"), stripHR), "\n")
}

func TestColonIntroBecomesFencedCode(t *testing.T) {
	in := "To load completions in your current shell session:\n\n\tsource <(omnictl completion bash)\n\nnext paragraph\n"
	want := "To load completions in your current shell session:\n\n```\nsource <(omnictl completion bash)\n```\n\nnext paragraph\n"
	if got := run(t, in, false); got != want {
		t.Errorf("colon intro not fenced:\n got: %q\nwant: %q", got, want)
	}
}

func TestHeadingColonIntroBecomesFencedCode(t *testing.T) {
	// A command example without any MDX-breaking characters must still be
	// fenced when introduced by a colon (here a "#### Linux:" heading).
	in := "#### Linux:\n\n\tomnictl completion bash > /etc/bash_completion.d/omnictl\n"
	want := "#### Linux:\n\n```\nomnictl completion bash > /etc/bash_completion.d/omnictl\n```\n"
	if got := run(t, in, false); got != want {
		t.Errorf("heading-colon intro not fenced:\n got: %q\nwant: %q", got, want)
	}
}

func TestSentenceIntroBecomesProse(t *testing.T) {
	in := "Create or update resources using YAML file(s) as input.\n\n\tIf a file is specified, only that file will be processed.\n\tIf a directory is specified, all files are processed.\n"
	want := "Create or update resources using YAML file(s) as input.\n\nIf a file is specified, only that file will be processed.\nIf a directory is specified, all files are processed.\n"
	if got := run(t, in, false); got != want {
		t.Errorf("prose block not de-indented:\n got: %q\nwant: %q", got, want)
	}
}

func TestExistingFenceUntouched(t *testing.T) {
	// A tab-indented line inside a real fenced block must not be re-fenced.
	in := "```bash\n\tsource <(cmd)\n```\n"
	if got := run(t, in, false); got != in {
		t.Errorf("existing fenced block altered:\n got: %q\nwant: %q", got, in)
	}
}

func TestFrontmatterPreserved(t *testing.T) {
	// Closing delimiter has a trailing space, and strip-hr is on: the
	// frontmatter must survive intact.
	in := "---\ntitle: Image Factory Configuration\ndescription: something\n--- \n\n## CLI Usage\n"
	want := in
	if got := run(t, in, true); got != want {
		t.Errorf("frontmatter not preserved:\n got: %q\nwant: %q", got, want)
	}
}

func TestStripHR(t *testing.T) {
	in := "HTTP configuration.\n\n---\n\n### `http.httpListenAddr`\n"
	want := "HTTP configuration.\n\n### `http.httpListenAddr`\n"
	if got := run(t, in, true); got != want {
		t.Errorf("HR not stripped:\n got: %q\nwant: %q", got, want)
	}
}

func TestStripHRDisabledByDefault(t *testing.T) {
	in := "a\n\n---\n\nb\n"
	if got := run(t, in, false); got != in {
		t.Errorf("HR stripped without --strip-hr:\n got: %q\nwant: %q", got, in)
	}
}

func TestEscapesInlinePlaceholderInProse(t *testing.T) {
	in := "use 'omnictl get machinestatus <machine-id> -o yaml' to inspect it\n"
	want := "use 'omnictl get machinestatus \\<machine-id> -o yaml' to inspect it\n"
	if got := run(t, in, false); got != want {
		t.Errorf("inline < not escaped:\n got: %q\nwant: %q", got, want)
	}
}

func TestDoesNotEscapeInsideFencedCode(t *testing.T) {
	in := "```\nomnictl get machinestatus <machine-id>\n```\n"
	if got := run(t, in, false); got != in {
		t.Errorf("escaped inside fenced code:\n got: %q\nwant: %q", got, in)
	}
}

func TestDoesNotEscapeInsideInlineCodeSpan(t *testing.T) {
	in := "run `omnictl get machinestatus <machine-id>` now\n"
	if got := run(t, in, false); got != in {
		t.Errorf("escaped inside inline code span:\n got: %q\nwant: %q", got, in)
	}
}

func TestEscapesBraceInProse(t *testing.T) {
	in := "the value {placeholder} is required\n"
	want := "the value \\{placeholder} is required\n"
	if got := run(t, in, false); got != want {
		t.Errorf("inline { not escaped:\n got: %q\nwant: %q", got, want)
	}
}

func TestStripHRKeepsFencedSeparators(t *testing.T) {
	// A "---" inside a code block (e.g. a YAML document separator) must stay.
	in := "```yaml\nfoo: 1\n---\nbar: 2\n```\n"
	if got := run(t, in, true); got != in {
		t.Errorf("fenced --- was stripped:\n got: %q\nwant: %q", got, in)
	}
}
