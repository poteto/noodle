package schemadoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ListTargets returns available schema targets.
func ListTargets() []TargetInfo {
	return listTargets()
}

// RenderMarkdown renders a schema target as markdown documentation.
func RenderMarkdown(targetName string) (string, error) {
	target, err := targetByName(targetName)
	if err != nil {
		return "", err
	}

	jsonSchema, err := renderJSONSchema(target)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# %s Schema\n\n", target.Info.FileName)
	fmt.Fprintf(&b, "%s\n\n", target.Info.Description)
	b.WriteString("```json\n")
	b.WriteString(jsonSchema)
	b.WriteString("\n```\n")
	if len(target.Constraints) > 0 {
		b.WriteString("\n## Constraints\n\n")
		for _, constraint := range target.Constraints {
			b.WriteString("- ")
			b.WriteString(constraint)
			b.WriteString("\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// RenderPromptJSON renders the JSON-shaped schema block for prompt injection.
func RenderPromptJSON(targetName string) (string, error) {
	target, err := targetByName(targetName)
	if err != nil {
		return "", err
	}

	jsonSchema, err := renderJSONSchema(target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s schema (JSON):\n```json\n%s\n```", target.Info.FileName, jsonSchema), nil
}

func renderJSONSchema(target targetSpec) (string, error) {
	if err := validateSpec(target); err != nil {
		return "", err
	}

	root, err := describeType(target.RootType, "", target.FieldDocs, false)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(root); err != nil {
		return "", fmt.Errorf("encode schema for %s: %w", target.Info.Name, err)
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func validateSpec(target targetSpec) error {
	leafPaths, err := gatherLeafPaths(target.RootType, "")
	if err != nil {
		return fmt.Errorf("gather schema leaf paths for %s: %w", target.Info.Name, err)
	}

	missing := make([]string, 0)
	for path := range leafPaths {
		if _, ok := target.FieldDocs[path]; !ok {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("schema metadata missing for %s: %s", target.Info.Name, strings.Join(missing, ", "))
	}

	extra := make([]string, 0)
	for path := range target.FieldDocs {
		if _, ok := leafPaths[path]; !ok {
			extra = append(extra, path)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		return fmt.Errorf("schema metadata has unknown paths for %s: %s", target.Info.Name, strings.Join(extra, ", "))
	}

	return nil
}

func gatherLeafPaths(t reflect.Type, path string) (map[string]struct{}, error) {
	t = derefType(t)
	if isTimeType(t) {
		return map[string]struct{}{path: {}}, nil
	}

	switch t.Kind() {
	case reflect.Struct:
		leafs := map[string]struct{}{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonName, include, _ := jsonField(field)
			if !include {
				continue
			}
			childPath := joinPath(path, jsonName)
			childLeafs, err := gatherLeafPaths(field.Type, childPath)
			if err != nil {
				return nil, err
			}
			for key := range childLeafs {
				leafs[key] = struct{}{}
			}
		}
		return leafs, nil
	case reflect.Slice, reflect.Array:
		return gatherLeafPaths(t.Elem(), path+"[]")
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type at %q: %s", path, t.Key())
		}
		return gatherLeafPaths(t.Elem(), path+"{}")
	default:
		return map[string]struct{}{path: {}}, nil
	}
}

func describeType(t reflect.Type, path string, docs map[string]FieldDoc, optional bool) (any, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		optional = true
	}

	if isTimeType(t) {
		return describeLeaf(path, t, docs, optional)
	}

	switch t.Kind() {
	case reflect.Struct:
		out := map[string]any{}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			jsonName, include, fieldOptional := jsonField(field)
			if !include {
				continue
			}
			childPath := joinPath(path, jsonName)
			value, err := describeType(field.Type, childPath, docs, optional || fieldOptional)
			if err != nil {
				return nil, err
			}
			out[jsonName] = value
		}
		return out, nil
	case reflect.Slice, reflect.Array:
		value, err := describeType(t.Elem(), path+"[]", docs, false)
		if err != nil {
			return nil, err
		}
		return []any{value}, nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type at %q: %s", path, t.Key())
		}
		value, err := describeType(t.Elem(), path+"{}", docs, false)
		if err != nil {
			return nil, err
		}
		return map[string]any{"<key>": value}, nil
	default:
		return describeLeaf(path, t, docs, optional)
	}
}

func describeLeaf(path string, t reflect.Type, docs map[string]FieldDoc, optional bool) (string, error) {
	doc, ok := docs[path]
	if !ok {
		return "", fmt.Errorf("schema metadata missing for %q", path)
	}
	typeLabel := strings.TrimSpace(doc.Type)
	if typeLabel == "" {
		typeLabel = inferTypeLabel(t)
	}
	if optional {
		typeLabel += " (optional)"
	}
	description := strings.TrimSpace(doc.Description)
	if description == "" {
		return typeLabel, nil
	}
	return typeLabel + " - " + description, nil
}

func inferTypeLabel(t reflect.Type) string {
	t = derefType(t)
	if isTimeType(t) {
		return "RFC3339 timestamp"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		return "object"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return t.Kind().String()
	}
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func isTimeType(t reflect.Type) bool {
	t = derefType(t)
	return t.PkgPath() == "time" && t.Name() == "Time"
}

func joinPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func jsonField(field reflect.StructField) (name string, include bool, optional bool) {
	if !field.IsExported() {
		return "", false, false
	}
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	name = field.Name
	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			name = strings.TrimSpace(parts[0])
		}
		for _, opt := range parts[1:] {
			if strings.TrimSpace(opt) == "omitempty" {
				optional = true
			}
		}
	}
	return name, true, optional
}
