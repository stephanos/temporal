package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVolumeSchemaRejectsIncompleteAndAmbiguousLayouts(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "internal", "model", "schema", "volumewire.json")
	definition, err := readVolumeWireSchema(path)
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name   string
		change func(*volumeWireSchema)
	}{
		{"version", func(value *volumeWireSchema) { value.Version++ }},
		{"magic", func(value *volumeWireSchema) { value.Magic["config"] = value.Magic["finish"] }},
		{"limit", func(value *volumeWireSchema) { delete(value.Limits, "operations") }},
		{"layout", func(value *volumeWireSchema) { value.Layouts = value.Layouts[1:] }},
		{"duplicate", func(value *volumeWireSchema) { value.Layouts = append(value.Layouts, value.Layouts[0]) }},
		{"empty", func(value *volumeWireSchema) { value.Layouts[0].Fields = nil }},
		{"field", func(value *volumeWireSchema) {
			value.Layouts[0].Fields = append(value.Layouts[0].Fields, value.Layouts[0].Fields[0])
		}},
		{"kind", func(value *volumeWireSchema) { value.Layouts[0].Fields[0].Kind = "unknown" }},
		{"type", func(value *volumeWireSchema) { value.Layouts[0].Fields[3].Type = "unknown" }},
		{"field-limit", func(value *volumeWireSchema) { value.Layouts[0].Fields[3].Limit = "unknown" }},
	}
	original, err := json.Marshal(definition)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			var changed volumeWireSchema
			if err := json.Unmarshal(original, &changed); err != nil {
				t.Fatal(err)
			}
			mutation.change(&changed)
			contents, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "volumewire.json")
			if err := os.WriteFile(path, contents, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := readVolumeWireSchema(path); err == nil {
				t.Fatal("accepted malformed semantic schema")
			}
		})
	}
}
