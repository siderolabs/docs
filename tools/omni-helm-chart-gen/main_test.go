package main

import (
	"strings"
	"testing"
)

const testRef = "v1.12.2"

// body strips the generated frontmatter and header comment, so tests can
// compare only the converted README content.
func body(t *testing.T, readme string) string {
	t.Helper()

	out := convert(readme, testRef)

	_, rest, ok := strings.Cut(out, "*/}\n\n")
	if !ok {
		t.Fatalf("output has no generated header comment:\n%s", out)
	}

	return rest
}

func TestHeaderHasFrontmatterAndRef(t *testing.T) {
	out := convert("Some text.\n", testRef)

	for _, want := range []string{
		"---\ntitle: Omni Helm Chart\ndescription: ",
		"automatically generated from the Omni Helm chart README (v1.12.2)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	if !strings.HasPrefix(out, "---\n") {
		t.Errorf("output does not start with frontmatter:\n%s", out)
	}
}

func TestDropsTitleBadgesHomepageAndMaintainers(t *testing.T) {
	in := "# Omni Helm Chart (v2)\n\n" +
		"![Version: 2.12.2](https://img.shields.io/badge/Version-2.12.2-informational?style=flat)\n\n" +
		"A Helm chart.\n\n" +
		"**Homepage:** <https://www.siderolabs.com/omni/>\n\n" +
		"## Maintainers\n\n" +
		"| Name | Email |\n| ---- | ----- |\n| Sidero Labs | <info@siderolabs.com> |\n\n" +
		"## Installation\n\nSteps.\n"

	want := "A Helm chart.\n\n## Installation\n\nSteps.\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestAlertsBecomeCallouts(t *testing.T) {
	for kind, callout := range callouts {
		in := "Intro.\n\n> [!" + kind + "]\n> First line.\n>\n> Second line.\n\nAfter.\n"
		want := "Intro.\n\n<" + callout + ">\nFirst line.\n\nSecond line.\n</" + callout + ">\n\nAfter.\n"

		if got := body(t, in); got != want {
			t.Errorf("%s alert:\n got: %q\nwant: %q", kind, got, want)
		}
	}
}

func TestCalloutClosesAtEndOfFile(t *testing.T) {
	in := "> [!NOTE]\n> Last thing in the file."
	want := "<Note>\nLast thing in the file.\n</Note>\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestPlainBlockquoteIsLeftAlone(t *testing.T) {
	in := "> Just a quote.\n"

	if got := body(t, in); got != in {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, in)
	}
}

func TestAutolinksBecomeMarkdownLinks(t *testing.T) {
	in := "See <https://example.com/a?b=c> or mail <info@siderolabs.com>.\n"
	want := "See [https://example.com/a?b=c](https://example.com/a?b=c) or mail [info@siderolabs.com](mailto:info@siderolabs.com).\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestPlaceholdersAreNotTreatedAsLinks(t *testing.T) {
	// Inline code placeholders must not become links.
	in := "Where `<prefix>` is random.\n"

	if got := body(t, in); got != in {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, in)
	}
}

func TestRelativeValuesLinkBecomesAbsolute(t *testing.T) {
	in := "See the [`values.yaml`](values.yaml) file.\n"
	want := "See the [`values.yaml`](https://github.com/siderolabs/omni/blob/v1.12.2/deploy/helm/omni/values.yaml) file.\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestCodeBlocksAreUntouched(t *testing.T) {
	// Nothing inside a fence is converted: not alerts, links, headings, or blank runs.
	in := "Example:\n\n```yaml\n# Omni Helm Chart\n> [!NOTE]\nurl: <https://example.com>\nsee: (values.yaml)\n\n\nend: true\n```\n\nAfter.\n"

	if got := body(t, in); got != in {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, in)
	}
}

func TestIndentedCodeBlocksInListsAreUntouched(t *testing.T) {
	in := "1. Enable it:\n   ```yaml\n   url: <https://example.com>\n   ```\n\n2. Next step.\n"

	if got := body(t, in); got != in {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, in)
	}
}

func TestBlankRunsAreCollapsed(t *testing.T) {
	in := "\n\nOne.\n\n\n\nTwo.\n\n\n"
	want := "One.\n\nTwo.\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestCRLFInput(t *testing.T) {
	in := "One.\r\n\r\nTwo.\r\n"
	want := "One.\n\nTwo.\n"

	if got := body(t, in); got != want {
		t.Errorf("unexpected output:\n got: %q\nwant: %q", got, want)
	}
}

func TestConvertIsDeterministic(t *testing.T) {
	in := "# T\n\n> [!WARNING]\n> Careful.\n\nSee <https://example.com>.\n"

	if a, b := convert(in, testRef), convert(in, testRef); a != b {
		t.Errorf("two runs differ:\n%q\n%q", a, b)
	}
}
