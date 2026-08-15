package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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

	"go.temporal.io/server/tools/gomadv3/internal/safefile"
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

type choiceSchema struct {
	Version uint16 `json:"version"`
	Profile string `json:"profile"`
	Trace   struct {
		Magic       string `json:"magic"`
		HeaderBytes int    `json:"header_bytes"`
		RecordBytes int    `json:"record_bytes"`
	} `json:"trace"`
	Tape struct {
		Magic          string `json:"magic"`
		HeaderBytes    int    `json:"header_bytes"`
		RecordBytes    int    `json:"record_bytes"`
		ChecksumOffset int    `json:"checksum_offset"`
	} `json:"tape"`
	Terminal struct {
		Magic  string `json:"magic"`
		States struct {
			Complete uint8 `json:"complete"`
			Overflow uint8 `json:"overflow"`
			Diverged uint8 `json:"diverged"`
		} `json:"states"`
		FrameBytes     int `json:"frame_bytes"`
		ChecksumOffset int `json:"checksum_offset"`
	} `json:"terminal"`
	Kinds struct {
		Runnable     uint8 `json:"runnable"`
		SelectPoll   uint8 `json:"select_poll"`
		SelectResult uint8 `json:"select_result"`
	} `json:"kinds"`
	Flags struct {
		Decision     uint8 `json:"decision"`
		Observation  uint8 `json:"observation"`
		SiteMissing  uint8 `json:"site_missing"`
		RankOverride uint8 `json:"rank_override"`
	} `json:"flags"`
	Modes struct {
		Seed   uint8 `json:"seed"`
		Record uint8 `json:"record"`
		Replay uint8 `json:"replay"`
		Prefix uint8 `json:"prefix"`
	} `json:"modes"`
	DivergenceReasons struct {
		Kind                uint8 `json:"kind"`
		Site                uint8 `json:"site"`
		Alternatives        uint8 `json:"alternatives"`
		Selected            uint8 `json:"selected"`
		AlternativeSet      uint8 `json:"alternative_set"`
		TapeExhausted       uint8 `json:"tape_exhausted"`
		TapeUnconsumed      uint8 `json:"tape_unconsumed"`
		IdentityMissing     uint8 `json:"identity_missing"`
		IdentityDuplicate   uint8 `json:"identity_duplicate"`
		AlternativeCapacity uint8 `json:"alternative_capacity"`
		Observation         uint8 `json:"observation"`
	} `json:"divergence_reasons"`
}

type choiceTemplateData struct {
	Package              string
	TestImport           string
	Schema               choiceSchema
	ImplementationDigest string
}

type choiceImplementationInputs struct {
	Schema          []byte
	CodecTemplate   []byte
	RuntimeTemplate []byte
	RuntimeOverlay  []byte
	ToolchainPatch  []byte
	HostTrace       []byte
	HostTape        []byte
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
		if err := safefile.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated I/O wire codec: %w", err)
		}
	}
	choiceDefinition, err := readChoiceSchema(filepath.Join(root, "protocol", "choicewire.json"))
	if err != nil {
		return err
	}
	identityInputs, err := readChoiceImplementationInputs(root)
	if err != nil {
		return err
	}
	implementationDigest := choiceImplementationIdentity(identityInputs)
	choiceOutputs := []output{
		{Package: "choicewire", Template: "choicewire.go.tmpl", Path: "internal/choicewire/wire_generated.go"},
		{Package: "choicewire", Template: "choicewire_test.go.tmpl", Path: "internal/choicewire/wire_generated_test.go"},
		{Package: "gomadchoicewire", Template: "choicewire.go.tmpl", Path: "overlay/src/internal/gomadchoicewire/wire_generated.go"},
		{Package: "gomadchoicewire_test", TestImport: "internal/gomadchoicewire", Template: "choicewire_test.go.tmpl", Path: "overlay/src/internal/gomadchoicewire/wire_generated_test.go"},
		{Package: "runtime", Template: "choicewire_runtime.go.tmpl", Path: "overlay/src/runtime/gomad_choicewire_generated.go"},
	}
	for _, target := range choiceOutputs {
		generated, generateErr := generate(filepath.Join(root, "protocol", target.Template), choiceTemplateData{Package: target.Package, TestImport: target.TestImport, Schema: choiceDefinition, ImplementationDigest: string(implementationDigest[:])})
		if generateErr != nil {
			return generateErr
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated choice wire codec is stale: %s", target.Path)
			}
			continue
		}
		if err := safefile.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated choice wire codec: %w", err)
		}
	}
	return nil
}

func readChoiceImplementationInputs(root string) (choiceImplementationInputs, error) {
	paths := []string{
		"protocol/choicewire.json",
		"protocol/choicewire.go.tmpl",
		"protocol/choicewire_runtime.go.tmpl",
		"overlay/src/runtime/gomad.go",
		"go1.26.4.patch",
		"internal/choicewire/trace.go",
		"internal/choicewire/tape.go",
	}
	values := make([][]byte, len(paths))
	for index, relative := range paths {
		value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return choiceImplementationInputs{}, fmt.Errorf("read choice implementation input %s: %w", relative, err)
		}
		values[index] = value
	}
	return choiceImplementationInputs{
		Schema: values[0], CodecTemplate: values[1], RuntimeTemplate: values[2], RuntimeOverlay: values[3], ToolchainPatch: values[4],
		HostTrace: values[5], HostTape: values[6],
	}, nil
}

func choiceImplementationIdentity(inputs choiceImplementationInputs) [sha256.Size]byte {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv3-choice-implementation-source-v2"))
	for _, input := range [][]byte{inputs.Schema, inputs.CodecTemplate, inputs.RuntimeTemplate, inputs.RuntimeOverlay, inputs.ToolchainPatch, inputs.HostTrace, inputs.HostTape} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(input)))
		_, _ = hasher.Write(size[:])
		_, _ = hasher.Write(input)
	}
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

func readChoiceSchema(path string) (choiceSchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return choiceSchema{}, fmt.Errorf("read choice wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition choiceSchema
	if err := decoder.Decode(&definition); err != nil {
		return choiceSchema{}, fmt.Errorf("decode choice wire schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return choiceSchema{}, errors.New("choice wire schema has trailing data")
	}
	if len(definition.Trace.Magic) != 8 || len(definition.Tape.Magic) != 8 || len(definition.Terminal.Magic) != 8 {
		return choiceSchema{}, errors.New("choice wire schema magic must have 8 bytes")
	}
	checks := []bool{
		definition.Version != 0,
		definition.Profile == "gomadv3-choice-trace/v2",
		definition.Trace.HeaderBytes == 64,
		definition.Trace.RecordBytes == 96,
		definition.Tape.HeaderBytes == 264,
		definition.Tape.RecordBytes == 96,
		definition.Tape.ChecksumOffset == 232,
		definition.Terminal.States.Complete == 1,
		definition.Terminal.States.Overflow == 2,
		definition.Terminal.States.Diverged == 3,
		definition.Terminal.FrameBytes == 312,
		definition.Terminal.ChecksumOffset == 280,
		definition.Kinds.Runnable == 1,
		definition.Kinds.SelectPoll == 2,
		definition.Kinds.SelectResult == 3,
		definition.Flags.Decision == 1,
		definition.Flags.Observation == 2,
		definition.Flags.SiteMissing == 4,
		definition.Flags.RankOverride == 8,
		definition.Modes.Seed == 0,
		definition.Modes.Record == 1,
		definition.Modes.Replay == 2,
		definition.Modes.Prefix == 3,
		definition.DivergenceReasons.Kind == 1,
		definition.DivergenceReasons.Site == 2,
		definition.DivergenceReasons.Alternatives == 3,
		definition.DivergenceReasons.Selected == 4,
		definition.DivergenceReasons.AlternativeSet == 5,
		definition.DivergenceReasons.TapeExhausted == 6,
		definition.DivergenceReasons.TapeUnconsumed == 7,
		definition.DivergenceReasons.IdentityMissing == 8,
		definition.DivergenceReasons.IdentityDuplicate == 9,
		definition.DivergenceReasons.AlternativeCapacity == 10,
		definition.DivergenceReasons.Observation == 11,
	}
	for _, valid := range checks {
		if !valid {
			return choiceSchema{}, errors.New("choice wire schema layout is unsupported by this generator")
		}
	}
	return definition, nil
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

func generate(path string, data any) ([]byte, error) {
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
