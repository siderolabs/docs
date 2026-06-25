package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiParagraphDescription(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create source file with multi-paragraph description
	srcFile := filepath.Join(tmpDir, "test.md")
	srcContent := `---
description: |
    This is the first paragraph of the description.
    It continues on the next line.

    This is the second paragraph after a blank line.
    It also continues on the next line.
title: TestConfig
---

# Test Content

Some body content here.`

	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Convert the file
	dstFile := filepath.Join(tmpDir, "test.mdx")
	if err := convertFile(srcFile, dstFile); err != nil {
		t.Fatalf("convertFile failed: %v", err)
	}

	// Read the output file
	output, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	// Check that description uses block scalar format (preserves paragraphs)
	if !strings.Contains(outputStr, "description: |") {
		t.Error("Description should use block scalar format")
	}

	// Check that both paragraphs are present
	if !strings.Contains(outputStr, "This is the first paragraph") {
		t.Error("First paragraph not found in description")
	}

	if !strings.Contains(outputStr, "This is the second paragraph") {
		t.Error("Second paragraph not found in description")
	}

	// Check that blank line between paragraphs is preserved
	lines := strings.Split(outputStr, "\n")
	descIdx := -1
	for i, line := range lines {
		if line == "description: |" {
			descIdx = i
			break
		}
	}

	if descIdx == -1 {
		t.Fatal("description: | line not found")
	}

	// Find the blank line between paragraphs (should be 4-5 lines after description)
	foundBlankLine := false
	for i := descIdx + 1; i < len(lines) && i < descIdx+8; i++ {
		if strings.TrimSpace(lines[i]) == "" && i > descIdx+2 {
			foundBlankLine = true
			break
		}
	}

	if !foundBlankLine {
		t.Error("Blank line between paragraphs not preserved")
	}

	// Verify title comes after the description block
	foundTitle := false
	for i := descIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "title:") {
			foundTitle = true
			break
		}
	}

	if !foundTitle {
		t.Error("title: line not found after description block")
	}
}

func TestDescriptionWithSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "special.md")
	srcContent := `---
description: |
    This has a 'single quote' in it.
    And multiple 'quotes' in the text.
title: SpecialConfig
---

Content here.`

	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	dstFile := filepath.Join(tmpDir, "special.mdx")
	if err := convertFile(srcFile, dstFile); err != nil {
		t.Fatalf("convertFile failed: %v", err)
	}

	output, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	// Verify description uses block scalar format
	if !strings.Contains(outputStr, "description: |") {
		t.Error("Description should use block scalar format")
	}

	// Verify special characters (quotes) are preserved as-is in block scalar
	if !strings.Contains(outputStr, "'single quote'") {
		t.Error("Single quotes should be preserved in block scalar")
	}

	// Verify the title line comes after the description block
	if !strings.Contains(outputStr, "title:") {
		t.Error("title: line not found")
	}
}

func TestSingleParagraphDescription(t *testing.T) {
	tmpDir := t.TempDir()

	srcFile := filepath.Join(tmpDir, "single.md")
	srcContent := `---
description: |
    This is a single paragraph description.
title: SingleConfig
---

Content here.`

	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	dstFile := filepath.Join(tmpDir, "single.mdx")
	if err := convertFile(srcFile, dstFile); err != nil {
		t.Fatalf("convertFile failed: %v", err)
	}

	output, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	outputStr := string(output)

	// Verify description uses block scalar format
	if !strings.Contains(outputStr, "description: |") {
		t.Error("Description should use block scalar format")
	}

	// Verify content is preserved
	if !strings.Contains(outputStr, "This is a single paragraph description.") {
		t.Error("Single paragraph description content not found")
	}
}
