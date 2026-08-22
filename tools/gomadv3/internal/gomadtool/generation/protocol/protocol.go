package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"go.temporal.io/server/tools/gomadv3/internal/gomadtool/generation/boundary"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
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

type liveCapabilitySchema struct {
	Version     uint32 `json:"version"`
	Schema      string `json:"schema"`
	Symbol      string `json:"symbol"`
	GuardSymbol string `json:"guard_symbol"`
	Header      struct {
		Magic string `json:"magic"`
		Bytes uint32 `json:"bytes"`
	} `json:"header"`
	Limits struct {
		PayloadBytes uint64 `json:"payload_bytes"`
		Facts        uint64 `json:"facts"`
		StringBytes  uint64 `json:"string_bytes"`
		OwnerFacts   uint64 `json:"owner_facts"`
	} `json:"limits"`
	FactKinds         []string `json:"fact_kinds"`
	Dispositions      []string `json:"dispositions"`
	GuardExemptions   []string `json:"guard_exemptions"`
	ForbiddenImports  []string `json:"forbidden_imports"`
	ForbiddenPrefixes []string `json:"forbidden_prefixes"`
}

type liveCapabilityImplementationInputs struct {
	Schema             []byte
	CodecTemplate      []byte
	CompilerEmitter    []byte
	LinkerProjector    []byte
	Encoder            []byte
	GuardFlag          []byte
	GuardSource        []byte
	RuntimeGuard       []byte
	InterceptionSource []byte
	BoundaryTable      []byte
	HostValidator      []byte
	ProjectionContract []byte
}

type liveCapabilityTemplateData struct {
	Package                   string
	Schema                    liveCapabilitySchema
	ImplementationDigest      string
	GuardImplementationDigest string
	UniverseDigest            string
	BoundaryManifestDigest    string
	Boundaries                []liveCapabilityBoundary
}

type simulationModelSchema struct {
	Version        uint16 `json:"version"`
	Profile        string `json:"profile"`
	Magic          string `json:"magic"`
	TransportMagic string `json:"transport_magic"`
	Limits         struct {
		FrameBytes  uint64 `json:"frame_bytes"`
		StringBytes uint64 `json:"string_bytes"`
		DataBytes   uint64 `json:"data_bytes"`
		Entries     uint64 `json:"entries"`
		NodeBytes   uint64 `json:"node_bytes"`
		ErrorBytes  uint64 `json:"error_bytes"`
	} `json:"limits"`
	Models struct {
		Network uint8 `json:"network"`
		Volume  uint8 `json:"volume"`
	} `json:"models"`
	NetworkOperations struct {
		Listen               uint16 `json:"listen"`
		Dial                 uint16 `json:"dial"`
		Accept               uint16 `json:"accept"`
		ListenerClose        uint16 `json:"listener_close"`
		ListenerSetDeadline  uint16 `json:"listener_set_deadline"`
		ConnRead             uint16 `json:"conn_read"`
		ConnWrite            uint16 `json:"conn_write"`
		ConnClose            uint16 `json:"conn_close"`
		ConnCloseRead        uint16 `json:"conn_close_read"`
		ConnCloseWrite       uint16 `json:"conn_close_write"`
		ConnSetDeadline      uint16 `json:"conn_set_deadline"`
		ConnSetReadDeadline  uint16 `json:"conn_set_read_deadline"`
		ConnSetWriteDeadline uint16 `json:"conn_set_write_deadline"`
	} `json:"network_operations"`
	VolumeOperations struct {
		Resolve        uint16 `json:"resolve"`
		Mkdir          uint16 `json:"mkdir"`
		MkdirAll       uint16 `json:"mkdir_all"`
		Stat           uint16 `json:"stat"`
		Open           uint16 `json:"open"`
		Rename         uint16 `json:"rename"`
		Remove         uint16 `json:"remove"`
		RemoveAll      uint16 `json:"remove_all"`
		Chmod          uint16 `json:"chmod"`
		Chtimes        uint16 `json:"chtimes"`
		Chdir          uint16 `json:"chdir"`
		Getwd          uint16 `json:"getwd"`
		HandleRead     uint16 `json:"handle_read"`
		HandleReadAt   uint16 `json:"handle_read_at"`
		HandleWrite    uint16 `json:"handle_write"`
		HandleWriteAt  uint16 `json:"handle_write_at"`
		HandleTruncate uint16 `json:"handle_truncate"`
		HandleChmod    uint16 `json:"handle_chmod"`
		HandleChtimes  uint16 `json:"handle_chtimes"`
		HandleChdir    uint16 `json:"handle_chdir"`
		HandleSeek     uint16 `json:"handle_seek"`
		HandleStat     uint16 `json:"handle_stat"`
		HandleReadDir  uint16 `json:"handle_read_dir"`
		HandleClose    uint16 `json:"handle_close"`
		HandleSync     uint16 `json:"handle_sync"`
		HandleMap      uint16 `json:"handle_map"`
		MappingBytes   uint16 `json:"mapping_bytes"`
		MappingClose   uint16 `json:"mapping_close"`
	} `json:"volume_operations"`
	Errors struct {
		None              uint16 `json:"none"`
		Generic           uint16 `json:"generic"`
		EOF               uint16 `json:"eof"`
		Deadline          uint16 `json:"deadline"`
		Canceled          uint16 `json:"canceled"`
		AddressInUse      uint16 `json:"address_in_use"`
		Closed            uint16 `json:"closed"`
		ConnectionRefused uint16 `json:"connection_refused"`
		ResourceExhausted uint16 `json:"resource_exhausted"`
		Unsupported       uint16 `json:"unsupported"`
		EINVAL            uint16 `json:"einval"`
		EEXIST            uint16 `json:"eexist"`
		ENOENT            uint16 `json:"enoent"`
		ENOTDIR           uint16 `json:"enotdir"`
		EISDIR            uint16 `json:"eisdir"`
		EROFS             uint16 `json:"erofs"`
		ENOSPC            uint16 `json:"enospc"`
		EBADF             uint16 `json:"ebadf"`
		ENODEV            uint16 `json:"enodev"`
		ESTALE            uint16 `json:"estale"`
		ENOTEMPTY         uint16 `json:"enotempty"`
		Capacity          uint16 `json:"capacity"`
	} `json:"errors"`
}

type simulationModelTemplateData struct {
	Package string
	Schema  simulationModelSchema
}

type liveCapabilityBoundary struct {
	Package     string
	Target      string
	Hook        string
	Operation   string
	Probe       string
	ProbeID     uint64
	Disposition string
}

type output struct {
	Package    string
	TestImport string
	Template   string
	Path       string
}

func GenerateProtocols(root string, check bool) error {
	definition, err := readSchema(filepath.Join(root, "deterministicio", "schema", "iowire.json"))
	if err != nil {
		return err
	}
	outputs := []output{
		{Package: "wire", Template: "iowire.go.tmpl", Path: "deterministicio/internal/wire/wire_generated.go"},
		{Package: "wire", Template: "iowire_test.go.tmpl", Path: "deterministicio/internal/wire/wire_generated_test.go"},
		{Package: "gomadwire", Template: "iowire.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go"},
		{Package: "gomadwire_test", TestImport: "internal/gomadwire", Template: "iowire_test.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go"},
	}
	for _, target := range outputs {
		generated, generateErr := generate(filepath.Join(root, "deterministicio", "schema", target.Template), templateData{Package: target.Package, TestImport: target.TestImport, Schema: definition})
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
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated I/O wire codec: %w", err)
		}
	}
	choiceDefinition, err := readChoiceSchema(filepath.Join(root, "choice", "schema", "choicewire.json"))
	if err != nil {
		return err
	}
	identityInputs, err := readChoiceImplementationInputs(root)
	if err != nil {
		return err
	}
	implementationDigest := choiceImplementationIdentity(identityInputs)
	choiceOutputs := []output{
		{Package: "wire", Template: "choicewire.go.tmpl", Path: "choice/internal/wire/wire_generated.go"},
		{Package: "wire", Template: "choicewire_test.go.tmpl", Path: "choice/internal/wire/wire_generated_test.go"},
		{Package: "gomadchoicewire", Template: "choicewire.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go"},
		{Package: "gomadchoicewire_test", TestImport: "internal/gomadchoicewire", Template: "choicewire_test.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated_test.go"},
		{Package: "runtime", Template: "choicewire_runtime.go.tmpl", Path: "toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go"},
	}
	for _, target := range choiceOutputs {
		generated, generateErr := generate(filepath.Join(root, "choice", "schema", target.Template), choiceTemplateData{Package: target.Package, TestImport: target.TestImport, Schema: choiceDefinition, ImplementationDigest: string(implementationDigest[:])})
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
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated choice wire codec: %w", err)
		}
	}
	modelDefinition, err := readSimulationModelSchema(filepath.Join(root, "simulation", "schema", "modelwire.json"))
	if err != nil {
		return err
	}
	modelOutputs := []output{
		{Package: "gomadmodelwire", Template: "modelwire.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go"},
		{Package: "gomadmodelwire", Template: "modelwire_test.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated_test.go"},
		{Package: "execution", Template: "modeltransport.go.tmpl", Path: "runner/internal/execution/simulation_model_wire_generated.go"},
		{Package: "gomadsim", Template: "modeltransport.go.tmpl", Path: "toolchain/runtime/overlay/src/internal/gomadsim/model_transport_generated.go"},
	}
	for _, target := range modelOutputs {
		generated, generateErr := generate(filepath.Join(root, "simulation", "schema", target.Template), simulationModelTemplateData{Package: target.Package, Schema: modelDefinition})
		if generateErr != nil {
			return generateErr
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated simulation model wire codec is stale: %s", target.Path)
			}
			continue
		}
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated simulation model wire codec: %w", err)
		}
	}
	if err := generateLiveCapabilityProtocols(root, check); err != nil {
		return err
	}
	return nil
}

func generateLiveCapabilityProtocols(root string, check bool) error {
	definition, err := readLiveCapabilitySchema(filepath.Join(root, "target", "internal", "livecap", "livecap.json"))
	if err != nil {
		return err
	}
	boundaryDigest, projectedBoundaries, err := boundary.CapabilityProjection(root)
	if err != nil {
		return err
	}
	universeDigest, err := liveCapabilityUniverseIdentity(definition, boundaryDigest)
	if err != nil {
		return err
	}
	identityInputs, err := readLiveCapabilityImplementationInputs(root)
	if err != nil {
		return err
	}
	implementation := liveCapabilityImplementationIdentity(identityInputs)
	implementationDigest := fmt.Sprintf("sha256:%x", implementation)
	boundaries := make([]liveCapabilityBoundary, len(projectedBoundaries))
	for index, entry := range projectedBoundaries {
		boundaries[index] = liveCapabilityBoundary{
			Package: entry.Package, Target: entry.Target, Hook: entry.Hook, Operation: entry.Operation,
			Probe: entry.Probe, ProbeID: entry.ProbeID, Disposition: entry.Disposition,
		}
	}
	outputs := []output{
		{Package: "livecap", Template: "livecap.go.tmpl", Path: "target/internal/livecap/protocol_generated.go"},
		{Package: "gomadcap", Template: "livecap.go.tmpl", Path: "toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go"},
	}
	for _, target := range outputs {
		generated, generateErr := generate(filepath.Join(root, "target", "internal", "livecap", target.Template), liveCapabilityTemplateData{
			Package: target.Package, Schema: definition, ImplementationDigest: implementationDigest, GuardImplementationDigest: implementationDigest,
			UniverseDigest: universeDigest, BoundaryManifestDigest: boundaryDigest, Boundaries: boundaries,
		})
		if generateErr != nil {
			return generateErr
		}
		path := filepath.Join(root, filepath.FromSlash(target.Path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated) {
				return fmt.Errorf("generated live capability protocol is stale: %s", target.Path)
			}
			continue
		}
		if err := hostfs.Replace(path, generated, 0o644); err != nil {
			return fmt.Errorf("write generated live capability protocol: %w", err)
		}
	}
	return nil
}

func readLiveCapabilityImplementationInputs(root string) (liveCapabilityImplementationInputs, error) {
	paths := []string{
		"target/internal/livecap/livecap.json",
		"target/internal/livecap/livecap.go.tmpl",
		"toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadcap.go",
		"toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go",
		"toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/base/gomadguard.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go",
		"toolchain/runtime/overlay/src/runtime/gomad.go",
		"toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go",
		"deterministicio/boundary_generated.go",
		"target/internal/livecap/livecap.go",
		"target/internal/livecap/project.go",
	}
	values := make([][]byte, len(paths))
	for index, relative := range paths {
		value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return liveCapabilityImplementationInputs{}, fmt.Errorf("read live capability implementation input %s: %w", relative, err)
		}
		values[index] = value
	}
	return liveCapabilityImplementationInputs{
		Schema: values[0], CodecTemplate: values[1], CompilerEmitter: values[2], LinkerProjector: values[3],
		Encoder: values[4], GuardFlag: values[5], GuardSource: values[6], RuntimeGuard: values[7],
		InterceptionSource: values[8], BoundaryTable: values[9], HostValidator: values[10], ProjectionContract: values[11],
	}, nil
}

func readChoiceImplementationInputs(root string) (choiceImplementationInputs, error) {
	paths := []string{
		"choice/schema/choicewire.json",
		"choice/schema/choicewire.go.tmpl",
		"choice/schema/choicewire_runtime.go.tmpl",
		"toolchain/runtime/overlay/src/runtime/gomad.go",
		"toolchain/runtime/go1.26.4.patch",
		"choice/trace.go",
		"choice/tape.go",
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

func liveCapabilityImplementationIdentity(inputs liveCapabilityImplementationInputs) [sha256.Size]byte {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv3-live-capability-implementation-source-v2"))
	for _, input := range [][]byte{
		inputs.Schema, inputs.CodecTemplate, inputs.CompilerEmitter, inputs.LinkerProjector, inputs.Encoder,
		inputs.GuardFlag, inputs.GuardSource, inputs.RuntimeGuard, inputs.InterceptionSource,
		inputs.BoundaryTable, inputs.HostValidator, inputs.ProjectionContract,
	} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(input)))
		_, _ = hasher.Write(size[:])
		_, _ = hasher.Write(input)
	}
	var result [sha256.Size]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

func liveCapabilityUniverseIdentity(definition liveCapabilitySchema, boundaryManifestSHA256 string) (string, error) {
	if len(boundaryManifestSHA256) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(boundaryManifestSHA256, "sha256:") {
		return "", errors.New("live capability boundary manifest identity is invalid")
	}
	projection := struct {
		Schema                 string   `json:"schema"`
		Version                uint32   `json:"version"`
		GuardSymbol            string   `json:"guard_symbol"`
		BoundaryManifestSHA256 string   `json:"boundary_manifest_sha256"`
		FactKinds              []string `json:"fact_kinds"`
		Dispositions           []string `json:"dispositions"`
		GuardExemptions        []string `json:"guard_exemptions"`
		ForbiddenImports       []string `json:"forbidden_imports"`
		ForbiddenPrefixes      []string `json:"forbidden_prefixes"`
	}{
		Schema: definition.Schema, Version: definition.Version, GuardSymbol: definition.GuardSymbol, BoundaryManifestSHA256: boundaryManifestSHA256,
		FactKinds: definition.FactKinds, Dispositions: definition.Dispositions, GuardExemptions: definition.GuardExemptions,
		ForbiddenImports: definition.ForbiddenImports, ForbiddenPrefixes: definition.ForbiddenPrefixes,
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("encode live capability universe: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", digest), nil
}

func readLiveCapabilitySchema(path string) (liveCapabilitySchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return liveCapabilitySchema{}, fmt.Errorf("read live capability schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition liveCapabilitySchema
	if err := decoder.Decode(&definition); err != nil {
		return liveCapabilitySchema{}, fmt.Errorf("decode live capability schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return liveCapabilitySchema{}, errors.New("live capability schema has trailing data")
	}
	if definition.Version != 2 || definition.Schema != "gomadv3.live-capability-manifest/v2" || definition.Symbol != "runtime.gomadCapabilities" || definition.GuardSymbol != "runtime.gomadCapabilityGuard" ||
		definition.Header.Magic != "GOMADCAPABILITY\x00" || definition.Header.Bytes != 112 ||
		definition.Limits.PayloadBytes != 16<<20 || definition.Limits.Facts != 100_000 || definition.Limits.StringBytes != 4<<10 || definition.Limits.OwnerFacts != 4_096 ||
		!slices.Equal(definition.FactKinds, []string{"boundary", "capability", "foreign", "guard", "linkname"}) ||
		!slices.Equal(definition.Dispositions, []string{"denied", "guarded", "modeled", "pack"}) ||
		!slices.Equal(definition.GuardExemptions, []string{"syscall.Clearenv", "syscall.Environ", "syscall.Errno.Error", "syscall.Errno.Is", "syscall.Errno.Temporary", "syscall.Errno.Timeout", "syscall.Getenv", "syscall.Setenv", "syscall.Unsetenv", "syscall.Write"}) ||
		!slices.Equal(definition.ForbiddenImports, []string{"os/exec", "os/signal", "os/user", "plugin", "runtime/cgo", "syscall"}) ||
		!slices.Equal(definition.ForbiddenPrefixes, []string{"golang.org/x/sys"}) {
		return liveCapabilitySchema{}, errors.New("live capability schema is unsupported by this generator")
	}
	return definition, nil
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

func readSimulationModelSchema(path string) (simulationModelSchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return simulationModelSchema{}, fmt.Errorf("read simulation model wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition simulationModelSchema
	if err := decoder.Decode(&definition); err != nil {
		return simulationModelSchema{}, fmt.Errorf("decode simulation model wire schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return simulationModelSchema{}, errors.New("simulation model wire schema has trailing data")
	}
	network := definition.NetworkOperations
	volume := definition.VolumeOperations
	errorCodes := definition.Errors
	if definition.Version != 1 || definition.Profile != "gomadv3.simulation-model/v1" || len(definition.Magic) != 8 || len(definition.TransportMagic) != 8 ||
		definition.Limits.FrameBytes != 128<<20 || definition.Limits.StringBytes != 4<<10 || definition.Limits.DataBytes != 64<<20 || definition.Limits.Entries != 100_000 || definition.Limits.NodeBytes != 256 || definition.Limits.ErrorBytes != 4<<10 ||
		definition.Models.Network != 1 || definition.Models.Volume != 2 ||
		!slices.Equal([]uint16{network.Listen, network.Dial, network.Accept, network.ListenerClose, network.ListenerSetDeadline, network.ConnRead, network.ConnWrite, network.ConnClose, network.ConnCloseRead, network.ConnCloseWrite, network.ConnSetDeadline, network.ConnSetReadDeadline, network.ConnSetWriteDeadline}, sequence16(1, 13)) ||
		!slices.Equal([]uint16{volume.Resolve, volume.Mkdir, volume.MkdirAll, volume.Stat, volume.Open, volume.Rename, volume.Remove, volume.RemoveAll, volume.Chmod, volume.Chtimes, volume.Chdir, volume.Getwd, volume.HandleRead, volume.HandleReadAt, volume.HandleWrite, volume.HandleWriteAt, volume.HandleTruncate, volume.HandleChmod, volume.HandleChtimes, volume.HandleChdir, volume.HandleSeek, volume.HandleStat, volume.HandleReadDir, volume.HandleClose, volume.HandleSync, volume.HandleMap, volume.MappingBytes, volume.MappingClose}, sequence16(1, 28)) ||
		!slices.Equal([]uint16{errorCodes.None, errorCodes.Generic, errorCodes.EOF, errorCodes.Deadline, errorCodes.Canceled, errorCodes.AddressInUse, errorCodes.Closed, errorCodes.ConnectionRefused, errorCodes.ResourceExhausted, errorCodes.Unsupported, errorCodes.EINVAL, errorCodes.EEXIST, errorCodes.ENOENT, errorCodes.ENOTDIR, errorCodes.EISDIR, errorCodes.EROFS, errorCodes.ENOSPC, errorCodes.EBADF, errorCodes.ENODEV, errorCodes.ESTALE, errorCodes.ENOTEMPTY, errorCodes.Capacity}, sequence16(0, 21)) {
		return simulationModelSchema{}, errors.New("simulation model wire schema is unsupported by this generator")
	}
	return definition, nil
}

func sequence16(first, last uint16) []uint16 {
	result := make([]uint16, last-first+1)
	for value := first; value <= last; value++ {
		result[value-first] = value
	}
	return result
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
	parsed, err := template.New(filepath.Base(path)).Funcs(template.FuncMap{"bytes": byteLiterals, "title": exportedIdentifier}).Parse(string(source))
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

func exportedIdentifier(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
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
