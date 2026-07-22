package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const defaultSchemaURL = "https://raw.githubusercontent.com/siderolabs/omni/refs/heads/main/internal/pkg/config/schema.json"

func main() {
	url := defaultSchemaURL
	if len(os.Args) > 1 {
		url = os.Args[1]
	}

	schema, err := fetchSchema(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching schema: %v\n", err)
		os.Exit(1)
	}

	g := newGenerator(schema)
	g.generate()

	fmt.Print(g.output.String())
}

func fetchSchema(source string) (*Schema, error) {
	var data []byte

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request for %s: %w", source, err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", source, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetching %s: status %d", source, resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}
	} else {
		var err error

		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", source, err)
		}
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("parsing schema: %w", err)
	}

	return &schema, nil
}

// Schema represents a JSON Schema object.
type Schema struct {
	Title                string            `json:"title"`
	Description          string            `json:"description"`
	Type                 string            `json:"type"`
	Ref                  string            `json:"$ref"`
	Required             []string          `json:"required"`
	Properties           OrderedMap        `json:"properties"`
	Definitions          OrderedMap        `json:"definitions"`
	XCliFlag             string            `json:"x-cli-flag"`
	Items                *Schema           `json:"items"`
	Enum                 []json.RawMessage `json:"enum"`
	Minimum              *float64          `json:"minimum"`
	Maximum              *float64          `json:"maximum"`
	MinLength            *int              `json:"minLength"`
	Pattern              string            `json:"pattern"`
	XPatternMessage      string            `json:"x-pattern-message"`
	Deprecated           bool              `json:"deprecated"`
	Const                json.RawMessage   `json:"const"`
	Default              json.RawMessage   `json:"default"`
	AdditionalProperties *Schema           `json:"additionalProperties"`
}

// OrderedMap preserves JSON key order for properties and definitions.
type OrderedMap struct {
	Keys   []string
	Values map[string]*Schema
}

func (m *OrderedMap) UnmarshalJSON(data []byte) error {
	m.Values = make(map[string]*Schema)

	dec := json.NewDecoder(bytes.NewReader(data))

	t, err := dec.Token()
	if err != nil {
		return err
	}

	if t != json.Delim('{') {
		return fmt.Errorf("expected '{', got %v", t)
	}

	for dec.More() {
		t, err = dec.Token()
		if err != nil {
			return err
		}

		key := t.(string)

		var val Schema
		if err := dec.Decode(&val); err != nil {
			return fmt.Errorf("decoding value for key %q: %w", key, err)
		}

		m.Keys = append(m.Keys, key)
		m.Values[key] = &val
	}

	return nil
}

// section is a group of leaf fields under a heading.
type section struct {
	path        string // dotted config path (e.g., "services.api")
	level       int    // heading level: 2 = ##, 3 = ###
	description string
	fields      []field
}

// field is a single leaf config value.
type field struct {
	path        string // dotted config path (e.g., "services.api.endpoint")
	typeName    string
	cliFlag     string
	description string
	required    bool
	deprecated  bool
	constraints string
	defaultVal  string
}

type generator struct {
	root     *Schema
	output   *strings.Builder
	sections []section
}

func newGenerator(schema *Schema) *generator {
	return &generator{
		root:   schema,
		output: &strings.Builder{},
	}
}

func (g *generator) generate() {
	g.collectSections()
	g.writeFrontmatter()
	g.writeIntro()

	for _, sec := range g.sections {
		g.emitSection(sec)
	}
}

// collectSections walks the schema and builds flat sections of leaf fields.
func (g *generator) collectSections() {
	for _, key := range g.root.Properties.Keys {
		prop := g.root.Properties.Values[key]
		resolved := g.resolveProperty(prop)
		g.walkObject(key, 2, prop.Description, resolved)
	}
}

// walkObject processes an object schema, creating a section for its leaf fields
// and recursing into nested objects as sub-sections.
func (g *generator) walkObject(path string, level int, description string, schema *Schema) {
	requiredSet := toSet(schema.Required)

	var leaves []field
	var nested []struct {
		key         string
		description string
		schema      *Schema
	}

	for _, key := range schema.Properties.Keys {
		prop := schema.Properties.Values[key]

		if isObjectRef(prop) {
			resolved := g.resolveProperty(prop)

			desc := prop.Description
			if desc == "" && prop.Ref != "" {
				if def := g.resolveRef(prop.Ref); def != nil {
					desc = def.Description
				}
			}

			nested = append(nested, struct {
				key         string
				description string
				schema      *Schema
			}{key, desc, resolved})

			continue
		}

		leaves = append(leaves, g.makeField(path+"."+key, prop, requiredSet[key]))
	}

	sec := section{
		path:        path,
		level:       level,
		description: description,
		fields:      leaves,
	}
	g.sections = append(g.sections, sec)

	subLevel := min(level+1, 4)

	for _, n := range nested {
		g.walkObject(path+"."+n.key, subLevel, n.description, n.schema)
	}
}

func (g *generator) makeField(path string, prop *Schema, required bool) field {
	return field{
		path:        path,
		typeName:    scalarType(prop),
		cliFlag:     prop.XCliFlag,
		description: prop.Description,
		required:    required,
		deprecated:  prop.Deprecated,
		constraints: constraintNotes(prop),
		defaultVal:  formatDefault(prop),
	}
}

func (g *generator) writeFrontmatter() {
	g.writef(`---
title: Omni Configuration
description: Complete reference for all Omni configuration options and their corresponding CLI flags.
---

{/* This file is automatically generated from the Omni configuration schema. Do not edit manually. */}

`)
}

func (g *generator) writeIntro() {
	g.writef(`Omni can be configured using a configuration file passed via ` + "`--config-path`" + ` (repeatable).
This page documents all available configuration options, their types, and corresponding CLI flags.

`)
}

func (g *generator) emitSection(sec section) {
	hashes := strings.Repeat("#", sec.level)
	g.writef("%s %s\n\n", hashes, sec.path)

	if sec.description != "" {
		g.writef("%s\n\n", escapeMDX(sec.description))
	}

	if len(sec.fields) == 0 {
		return
	}

	g.writef("<table>\n")
	g.writef("  <thead>\n    <tr>\n      <th>Field</th>\n      <th>Type</th>\n      <th>Description</th>\n    </tr>\n  </thead>\n")
	g.writef("  <tbody>\n")

	for _, f := range sec.fields {
		name := fmt.Sprintf("`%s`", lastSegment(f.path))
		desc := buildDescription(f)
		g.writef("    <tr>\n      <td style={{whiteSpace: 'nowrap', padding: '8px'}}>%s</td>\n      <td style={{whiteSpace: 'nowrap', padding: '8px'}}>%s</td>\n      <td>%s</td>\n    </tr>\n", name, f.typeName, desc)
	}

	g.writef("  </tbody>\n</table>\n\n")
}

func (g *generator) resolveProperty(prop *Schema) *Schema {
	if prop.Ref == "" {
		return prop
	}

	base := g.resolveRef(prop.Ref)
	if base == nil {
		return prop
	}

	if len(prop.Properties.Keys) > 0 {
		return mergeSchemas(base, prop)
	}

	return base
}

func (g *generator) resolveRef(ref string) *Schema {
	name := extractDefName(ref)
	if name == "" {
		return nil
	}

	if def, ok := g.root.Definitions.Values[name]; ok {
		return def
	}

	return nil
}

func (g *generator) writef(format string, args ...any) {
	fmt.Fprintf(g.output, format, args...)
}

// isObjectRef returns true if the property is an object reference ($ref)
// that should be rendered as a nested section rather than a table row.
// Map types (object with additionalProperties) are leaf values.
func isObjectRef(prop *Schema) bool {
	if prop.Ref != "" {
		return true
	}

	if prop.Type == "object" && prop.AdditionalProperties == nil && len(prop.Properties.Keys) > 0 {
		return true
	}

	return false
}

func scalarType(prop *Schema) string {
	if isDuration(prop) {
		return "duration"
	}

	if prop.Type == "array" && prop.Items != nil {
		return prop.Items.Type + "[]"
	}

	if prop.Type == "object" && prop.AdditionalProperties != nil {
		return fmt.Sprintf("map[string]%s", prop.AdditionalProperties.Type)
	}

	if prop.Type != "" {
		return prop.Type
	}

	return "any"
}

func buildDescription(f field) string {
	var parts []string

	if f.required {
		parts = append(parts, "**Required.**")
	}

	if f.deprecated {
		parts = append(parts, "**Deprecated.**")
	}

	if f.description != "" {
		parts = append(parts, f.description)
	}

	if f.constraints != "" {
		parts = append(parts, f.constraints)
	}

	desc := strings.Join(parts, " ")
	desc = strings.ReplaceAll(desc, "\n", " ")
	desc = strings.ReplaceAll(desc, "<", "&lt;")
	desc = strings.ReplaceAll(desc, ">", "&gt;")
	desc = escapeURLPatterns(desc)

	if f.cliFlag != "" {
		desc += fmt.Sprintf("<br />**CLI flag:** `--%s`", f.cliFlag)
	}

	if f.defaultVal != "" {
		desc += fmt.Sprintf("<br />**Default:** %s", f.defaultVal)
	}

	return desc
}

func formatDefault(prop *Schema) string {
	if prop.Default == nil {
		return ""
	}

	raw := strings.TrimSpace(string(prop.Default))

	// Unquote JSON strings.
	var s string
	if err := json.Unmarshal(prop.Default, &s); err == nil {
		if s == "" {
			return ""
		}

		return fmt.Sprintf("`%s`", s)
	}

	return fmt.Sprintf("`%s`", raw)
}

// escapeURLPatterns wraps bare URL-like patterns in backticks so that
// broken link checkers do not flag them as real URLs.
func escapeURLPatterns(s string) string {
	re := regexp.MustCompile(`"(https?(?:\(s\))?://[^"]+)"`)

	return re.ReplaceAllString(s, "`$1`")
}

func constraintNotes(prop *Schema) string {
	var notes []string

	if len(prop.Enum) > 0 {
		vals := make([]string, 0, len(prop.Enum))

		for _, e := range prop.Enum {
			vals = append(vals, fmt.Sprintf("`%s`", strings.Trim(string(e), `"`)))
		}

		notes = append(notes, fmt.Sprintf("Values: %s.", strings.Join(vals, ", ")))
	}

	if prop.Minimum != nil {
		notes = append(notes, fmt.Sprintf("Minimum: `%g`.", *prop.Minimum))
	}

	if prop.Maximum != nil {
		notes = append(notes, fmt.Sprintf("Maximum: `%g`.", *prop.Maximum))
	}

	if prop.MinLength != nil && *prop.MinLength > 0 {
		notes = append(notes, "Must not be empty.")
	}

	if prop.XPatternMessage != "" {
		notes = append(notes, fmt.Sprintf("Format: %s.", prop.XPatternMessage))
	}

	if prop.Const != nil {
		notes = append(notes, fmt.Sprintf("Constant: `%s`.", strings.Trim(string(prop.Const), `"`)))
	}

	return strings.Join(notes, " ")
}

// mergeSchemas merges base definition properties with inline overrides.
func mergeSchemas(base, overlay *Schema) *Schema {
	result := &Schema{
		Description: base.Description,
		Type:        base.Type,
		Required:    base.Required,
		Properties: OrderedMap{
			Keys:   make([]string, len(base.Properties.Keys)),
			Values: make(map[string]*Schema, len(base.Properties.Values)),
		},
	}

	copy(result.Properties.Keys, base.Properties.Keys)

	for k, v := range base.Properties.Values {
		cp := *v
		result.Properties.Values[k] = &cp
	}

	for _, key := range overlay.Properties.Keys {
		overlayProp := overlay.Properties.Values[key]

		if baseProp, exists := result.Properties.Values[key]; exists {
			merged := *baseProp

			if overlayProp.XCliFlag != "" {
				merged.XCliFlag = overlayProp.XCliFlag
			}

			if overlayProp.Description != "" {
				merged.Description = overlayProp.Description
			}

			if overlayProp.Type != "" {
				merged.Type = overlayProp.Type
			}

			if overlayProp.Default != nil {
				merged.Default = overlayProp.Default
			}

			result.Properties.Values[key] = &merged
		} else {
			result.Properties.Keys = append(result.Properties.Keys, key)
			result.Properties.Values[key] = overlayProp
		}
	}

	if overlay.Required != nil {
		seen := toSet(result.Required)

		for _, r := range overlay.Required {
			if !seen[r] {
				result.Required = append(result.Required, r)
			}
		}
	}

	return result
}

func extractDefName(ref string) string {
	const prefix = "#/definitions/"

	if !strings.HasPrefix(ref, prefix) {
		return ""
	}

	return strings.TrimPrefix(ref, prefix)
}

func escapeMDX(s string) string {
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")

	return s
}

func isDuration(prop *Schema) bool {
	return prop.Type == "string" && strings.Contains(prop.Pattern, "ns|us|µs|ms|s|m|h")
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}

	return path
}

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}

	return m
}
