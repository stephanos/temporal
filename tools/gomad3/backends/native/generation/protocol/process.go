package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
)

type processSchema struct {
	Version         uint16         `json:"version"`
	Profile         string         `json:"profile"`
	BootstrapSchema string         `json:"bootstrap_schema"`
	TerminalSchema  string         `json:"terminal_schema"`
	FrameBytes      uint64         `json:"frame_bytes"`
	NodeBytes       uint64         `json:"node_bytes"`
	ErrorBytes      uint64         `json:"error_bytes"`
	Kinds           []string       `json:"kinds"`
	Frame           []processField `json:"frame"`
	Bootstrap       []processField `json:"bootstrap"`
	Terminal        []processField `json:"terminal"`
	GoldenWait      string         `json:"golden_wait"`
}

type processField struct {
	Name string `json:"name"`
	Type string `json:"type"`
	JSON string `json:"json"`
}

func (definition processSchema) FrameKinds() []struct{ Name, Value string } {
	kinds := make([]struct{ Name, Value string }, 0, len(definition.Kinds))
	for _, kind := range definition.Kinds {
		var name strings.Builder
		for _, part := range strings.Split(kind, "_") {
			name.WriteString(exportedIdentifier(part))
		}
		kinds = append(kinds, struct{ Name, Value string }{Name: name.String(), Value: kind})
	}
	return kinds
}

type processTemplateData struct {
	Package    string
	Schema     processSchema
	WaitPrefix string
}

func readProcessSchema(path string) (processSchema, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return processSchema{}, fmt.Errorf("read process wire schema: %w", err)
	}
	var definition processSchema
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return processSchema{}, fmt.Errorf("decode process wire schema: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return processSchema{}, errors.New("process wire schema contains trailing data")
	}
	if definition.Version != 3 || definition.Profile == "" || definition.BootstrapSchema == "" || definition.TerminalSchema == "" || definition.FrameBytes == 0 || definition.NodeBytes == 0 || definition.ErrorBytes == 0 || !slices.Contains(definition.Kinds, "wait") || !slices.Contains(definition.Kinds, "response") || len(definition.Frame) < 3 || definition.Frame[0].Name != "Profile" || definition.Frame[1].Name != "Kind" {
		return processSchema{}, errors.New("process wire schema is invalid")
	}
	return definition, nil
}

func generateProcessProtocols(root string, check bool) error {
	definition, err := readProcessSchema(filepath.Join(root, "simulation", "schema", "processwire.json"))
	if err != nil {
		return err
	}
	profileJSON, err := json.Marshal(definition.Profile)
	if err != nil {
		return fmt.Errorf("encode process profile: %w", err)
	}
	waitPrefix := fmt.Sprintf(`{"%s":%s,"%s":"wait",`, definition.Frame[0].JSON, profileJSON, definition.Frame[1].JSON)
	if !strings.HasPrefix(definition.GoldenWait, waitPrefix) {
		return errors.New("process wire wait vector does not match frame prefix")
	}
	outputs := []output{
		{Package: "gomad3sim", Template: "processwire.go.tmpl", Path: "../gomad3sim/runtime_process_wire.go"},
		{Package: "gomad3sim", Template: "processwire_test.go.tmpl", Path: "../gomad3sim/runtime_process_wire_generated_test.go"},
		{Package: "gomadsim", Template: "processwire_transport.go.tmpl", Path: "backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated.go"},
		{Package: "gomadsim_test", Template: "processwire_transport_test.go.tmpl", Path: "backends/native/toolchain/runtime/overlay/src/internal/gomadsim/process_wire_generated_test.go"},
	}
	for _, target := range outputs {
		generated, err := generate(filepath.Join(root, "simulation", "schema", target.Template), processTemplateData{Package: target.Package, Schema: definition, WaitPrefix: waitPrefix})
		if err != nil {
			return err
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated process wire codec is stale: %s", target.Path)
			}
			continue
		}
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated process wire codec: %w", err)
		}
	}
	return nil
}
