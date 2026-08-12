package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/template"
)

type schema struct {
	Version   uint16 `json:"version"`
	Bootstrap struct {
		Magic          string `json:"magic"`
		Kind           uint16 `json:"kind"`
		FrameBytes     int    `json:"frame_bytes"`
		ChecksumOffset int    `json:"checksum_offset"`
	} `json:"bootstrap"`
	Transcript struct {
		ProducedMagic  string `json:"produced_magic"`
		ExpectedMagic  string `json:"expected_magic"`
		HeaderBytes    int    `json:"header_bytes"`
		RecordBytes    int    `json:"record_bytes"`
		OperationBytes int    `json:"operation_bytes"`
	} `json:"transcript"`
	Terminal struct {
		Magic  string `json:"magic"`
		States struct {
			Complete         uint8 `json:"complete"`
			Overflow         uint8 `json:"overflow"`
			ReplayDivergence uint8 `json:"replay_divergence"`
		} `json:"states"`
		FrameBytes     int `json:"frame_bytes"`
		ChecksumOffset int `json:"checksum_offset"`
	} `json:"terminal"`
	Mount struct {
		RequestMagic    string `json:"request_magic"`
		ResponseMagic   string `json:"response_magic"`
		LookupOperation uint16 `json:"lookup_operation"`
		Statuses        struct {
			OK        uint16 `json:"ok"`
			Unmounted uint16 `json:"unmounted"`
			NotExist  uint16 `json:"not_exist"`
		} `json:"statuses"`
		Kinds struct {
			File      uint8 `json:"file"`
			Directory uint8 `json:"directory"`
		} `json:"kinds"`
		RequestHeaderBytes  int `json:"request_header_bytes"`
		ResponseHeaderBytes int `json:"response_header_bytes"`
		ChildHeaderBytes    int `json:"child_header_bytes"`
	} `json:"mount"`
	Golden struct {
		Bootstrap     string `json:"bootstrap"`
		MountRequest  string `json:"mount_request"`
		MountResponse string `json:"mount_response"`
	} `json:"golden"`
}

type templateData struct {
	Package    string
	TestImport string
	Schema     schema
}

type output struct {
	Package    string
	TestImport string
	Template   string
	Path       string
}

func main() {
	check := flag.Bool("check", false, "check generated files without changing them")
	root := flag.String("root", ".", "Gomad v3 module root")
	flag.Parse()
	if err := run(*root, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, check bool) error {
	definition, err := readSchema(filepath.Join(root, "protocol", "iowire.json"))
	if err != nil {
		return err
	}
	outputs := []output{
		{Package: "iowire", Template: "iowire.go.tmpl", Path: "internal/iowire/wire_generated.go"},
		{Package: "iowire", Template: "iowire_test.go.tmpl", Path: "internal/iowire/wire_generated_test.go"},
		{Package: "gomadwire", Template: "iowire.go.tmpl", Path: "overlay/src/internal/gomadwire/wire_generated.go"},
		{Package: "gomadwire_test", TestImport: "internal/gomadwire", Template: "iowire_test.go.tmpl", Path: "overlay/src/internal/gomadwire/wire_generated_test.go"},
	}
	for _, target := range outputs {
		generated, generateErr := generate(filepath.Join(root, "protocol", target.Template), templateData{Package: target.Package, TestImport: target.TestImport, Schema: definition})
		if generateErr != nil {
			return generateErr
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated I/O wire codec is stale: %s", target.Path)
			}
			continue
		}
		if err := writeAtomic(path, generated); err != nil {
			return err
		}
	}
	return nil
}

func readSchema(path string) (schema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return schema{}, fmt.Errorf("read I/O wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition schema
	if err := decoder.Decode(&definition); err != nil {
		return schema{}, fmt.Errorf("decode I/O wire schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return schema{}, errors.New("I/O wire schema has trailing data")
	}
	for name, magic := range map[string]string{
		"bootstrap": definition.Bootstrap.Magic, "produced transcript": definition.Transcript.ProducedMagic,
		"expected transcript": definition.Transcript.ExpectedMagic, "terminal": definition.Terminal.Magic,
		"mount request": definition.Mount.RequestMagic, "mount response": definition.Mount.ResponseMagic,
	} {
		if len(magic) != 8 {
			return schema{}, fmt.Errorf("I/O wire schema %s magic has %d bytes", name, len(magic))
		}
	}
	if !supportedSchema(definition) {
		return schema{}, errors.New("I/O wire schema layout is unsupported by this generator")
	}
	return definition, nil
}

func supportedSchema(definition schema) bool {
	checks := []bool{
		definition.Version != 0,
		definition.Bootstrap.Kind == 1,
		definition.Bootstrap.FrameBytes == 212,
		definition.Bootstrap.ChecksumOffset == 180,
		definition.Transcript.HeaderBytes == 64,
		definition.Transcript.RecordBytes == 128,
		definition.Transcript.OperationBytes == 22,
		definition.Terminal.States.Complete == 1,
		definition.Terminal.States.Overflow == 2,
		definition.Terminal.States.ReplayDivergence == 3,
		definition.Terminal.FrameBytes == 104,
		definition.Terminal.ChecksumOffset == 72,
		definition.Mount.LookupOperation == 1,
		definition.Mount.Statuses.OK == 0,
		definition.Mount.Statuses.Unmounted == 1,
		definition.Mount.Statuses.NotExist == 2,
		definition.Mount.Kinds.File == 1,
		definition.Mount.Kinds.Directory == 2,
		definition.Mount.RequestHeaderBytes == 24,
		definition.Mount.ResponseHeaderBytes == 40,
		definition.Mount.ChildHeaderBytes == 8,
	}
	for _, valid := range checks {
		if !valid {
			return false
		}
	}
	return true
}

func generate(path string, data templateData) ([]byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read I/O wire template: %w", err)
	}
	parsed, err := template.New(filepath.Base(path)).Funcs(template.FuncMap{"bytes": byteLiterals}).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse I/O wire template: %w", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render I/O wire template: %w", err)
	}
	formatted, err := format.Source(rendered.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format generated I/O wire codec: %w", err)
	}
	return formatted, nil
}

func byteLiterals(value string) string {
	var result bytes.Buffer
	for index := range len(value) {
		if index != 0 {
			result.WriteString(", ")
		}
		result.WriteString(strconv.QuoteRune(rune(value[index])))
	}
	return result.String()
}

func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create generated I/O wire directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".iowire-*")
	if err != nil {
		return fmt.Errorf("create generated I/O wire file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write generated I/O wire file: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set generated I/O wire mode: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close generated I/O wire file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish generated I/O wire file: %w", err)
	}
	return nil
}
