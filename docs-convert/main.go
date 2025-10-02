package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func convertFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	scanner := bufio.NewScanner(src)
	writer := bufio.NewWriter(dst)
	defer writer.Flush()

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Process lines
	i := 0
	inFrontmatter := false
	for i < len(lines) {
		line := lines[i]

		// Track frontmatter boundaries
		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				fmt.Fprintln(writer, line)
				i++
				continue
			} else {
				// End of frontmatter
				inFrontmatter = false
				fmt.Fprintln(writer, line)
				i++
				continue
			}
		}

		// Handle multi-line description in frontmatter
		if inFrontmatter && strings.HasPrefix(line, "description: |") {
			// Collect all indented lines that follow
			descriptionParts := []string{}
			i++
			for i < len(lines) && len(lines[i]) > 0 && (lines[i][0] == ' ' || lines[i][0] == '\t') {
				// Remove leading whitespace and add to parts
				descriptionParts = append(descriptionParts, strings.TrimSpace(lines[i]))
				i++
			}
			// Join all parts into a single line
			singleLineDescription := strings.Join(descriptionParts, " ")
			// Escape single quotes in the description
			singleLineDescription = strings.Replace(singleLineDescription, "'", "''", -1)
			// Quote the description with single quotes to handle special characters
			fmt.Fprintf(writer, "description: '%s'\n", singleLineDescription)
			continue
		}

		// Check if this line starts an Accordion with inline code
		if strings.Contains(line, "<details><summary>") {
			// Find the end of the Accordion (</details>)
			endLine := i
			for endLine < len(lines) && !strings.Contains(lines[endLine], "</details>") {
				endLine++
			}

			if endLine < len(lines) {
				// Collect all lines in the Accordion
				accordionLines := append([]string{line}, lines[i+1:endLine+1]...)
				fullContent := strings.Join(accordionLines, "\n")

				// Replace tags
				fullContent = strings.Replace(fullContent, "<details><summary>", "<Accordion title=\"", 1)
				fullContent = strings.Replace(fullContent, "</summary>", "\">", 1)
				fullContent = strings.Replace(fullContent, "</details>", "</Accordion>", 1)

				// Extract and convert all highlight blocks
				inlineBlocks := []string{}
				remaining := fullContent
				for strings.Contains(remaining, "{{< highlight yaml >}}") {
					start := strings.Index(remaining, "{{< highlight yaml >}}")
					end := strings.Index(remaining, "{{< /highlight >}}")
					if end == -1 {
						break
					}

					// Extract code between tags
					codeStart := start + len("{{< highlight yaml >}}")
					code := strings.TrimSpace(remaining[codeStart:end])
					// Replace actual newlines with the string "\n" to keep everything on one line
					// This prevents table cells from breaking
					code = strings.Replace(code, "\n", "\\n", -1)
					// Escape angle brackets for MDX inside inline code
					code = strings.Replace(code, "<", "\\<", -1)
					code = strings.Replace(code, ">", "\\>", -1)
					// Escape pipe characters to prevent breaking markdown tables
					code = strings.Replace(code, "|", "\\|", -1)
					inlineBlocks = append(inlineBlocks, "`"+code+"`")

					// Remove this block from remaining
					remaining = remaining[:start] + "%%INLINE%%"  + remaining[end+len("{{< /highlight >}}"):]
				}

				// Replace placeholders with inline code, adding <br /> between them
				for i, block := range inlineBlocks {
					if i > 0 {
						remaining = strings.Replace(remaining, "%%INLINE%%", "<br />"+block, 1)
					} else {
						remaining = strings.Replace(remaining, "%%INLINE%%", block, 1)
					}
				}

				// Convert <br> to <br /> for MDX compatibility
				remaining = strings.Replace(remaining, "<br>", "<br />", -1)

				fmt.Fprintln(writer, remaining)
				i = endLine + 1
				continue
			}
		}

		// Handle regular lines - convert Hugo shortcodes to code blocks
		line = strings.Replace(line, "{{< highlight yaml >}}", "```yaml", -1)
		line = strings.Replace(line, "{{< /highlight >}}", "```", -1)

		// Convert <br> to <br /> for MDX compatibility
		line = strings.Replace(line, "<br>", "<br />", -1)

		// Skip markdownlint-disable comments
		if strings.Contains(line, "<!-- markdownlint-disable -->") {
			i++
			continue
		}

		// Remove {#anchor-id} from headings (MDX doesn't support this syntax)
		if strings.HasPrefix(strings.TrimSpace(line), "#") && strings.Contains(line, "{#") {
			start := strings.Index(line, "{#")
			end := strings.Index(line, "}")
			if end > start && end != -1 {
				line = strings.TrimSpace(line[:start] + line[end+1:])
			}
		}

		// Escape placeholder-like angle brackets (e.g., <src-path>, <dest-path>)
		// but preserve actual HTML tags (a, br, Accordion)
		// Simple heuristic: if it contains a hyphen or underscore, it's likely a placeholder
		line = escapeAngleBracketPlaceholders(line)

		fmt.Fprintln(writer, line)
		i++
	}

	return nil
}

func escapeAngleBracketPlaceholders(line string) string {
	result := ""
	i := 0
	for i < len(line) {
		if line[i] == '<' {
			// Find the closing >
			end := i + 1
			for end < len(line) && line[end] != '>' {
				end++
			}
			if end < len(line) {
				// Extract the content between < and >
				content := line[i+1 : end]
				// Check if it's likely a placeholder or text that should not be parsed as HTML
				// Known HTML tags we want to preserve: a, br, Accordion and their closing tags
				isKnownTag := strings.HasPrefix(content, "a ") ||
					strings.HasPrefix(content, "br") ||
					strings.HasPrefix(content, "Accordion") ||
					content == "/a" ||
					content == "/br" ||
					content == "/Accordion"

				// If it's not a known HTML tag, escape using JSX expressions
				if !isKnownTag {
					result += `{"<"}` + content + `{">"}`
					i = end + 1
					continue
				}
			}
		}
		result += string(line[i])
		i++
	}
	return result
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: convert-docs <source_dir> <dest_dir>")
		os.Exit(1)
	}

	srcDir := os.Args[1]
	dstDir := os.Args[2]

	fmt.Printf("Converting docs from %s to %s\n", srcDir, dstDir)

	// Remove and recreate destination directory
	os.RemoveAll(dstDir)
	os.MkdirAll(dstDir, 0755)

	// Walk through source directory
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip _index.md files
		if strings.Contains(relPath, "_index.md") {
			fmt.Printf("Skipping %s\n", relPath)
			return nil
		}

		// Convert .md to .mdx
		dstPath := filepath.Join(dstDir, strings.TrimSuffix(relPath, ".md")+".mdx")

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		fmt.Printf("Converting %s -> %s\n", relPath, strings.TrimSuffix(relPath, ".md")+".mdx")

		return convertFile(path, dstPath)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Count converted files
	count := 0
	filepath.Walk(dstDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".mdx") {
			count++
		}
		return nil
	})

	fmt.Println("Conversion complete!")
	fmt.Printf("Converted files: %d\n", count)
}
