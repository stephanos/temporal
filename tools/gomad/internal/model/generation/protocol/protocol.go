package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

// BootstrapSeedOffset is where the seed starts: it is the last field before
// the checksum, which EncodeBootstrap writes immediately after it.
func (s schema) BootstrapSeedOffset() int {
	return s.Bootstrap.ChecksumOffset - 8
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
	Readiness struct {
		Flags struct {
			Known           uint16 `json:"known"`
			Default         uint16 `json:"default"`
			NilChannel      uint16 `json:"nil_channel"`
			TimerChannel    uint16 `json:"timer_channel"`
			ClosedChannel   uint16 `json:"closed_channel"`
			RepeatedChannel uint16 `json:"repeated_channel"`
		} `json:"flags"`
		CountShift   uint8  `json:"count_shift"`
		MaximumCount uint16 `json:"maximum_count"`
	} `json:"readiness"`
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
	Diagnostic struct {
		Version      uint16 `json:"version"`
		Profile      string `json:"profile"`
		Magic        string `json:"magic"`
		HeaderBytes  int    `json:"header_bytes"`
		RecordBytes  int    `json:"record_bytes"`
		MaximumBytes uint64 `json:"maximum_bytes"`
		States       struct {
			Complete uint8 `json:"complete"`
			Overflow uint8 `json:"overflow"`
		} `json:"states"`
	} `json:"diagnostic"`
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
		StallTimeout      uint16 `json:"stall_timeout"`
	} `json:"errors"`
}

type simulationModelTemplateData struct {
	Package string
	Schema  simulationModelSchema
}

type worldSnapshotSchema struct {
	Version uint16 `json:"version"`
	Profile string `json:"profile"`
	Magic   string `json:"magic"`
	Limits  struct {
		Depth     int    `json:"depth"`
		EnumBytes uint64 `json:"enum_bytes"`
	} `json:"limits"`
	Layouts struct {
		Snapshot   []string `json:"snapshot"`
		Request    []string `json:"request"`
		Readiness  []string `json:"readiness"`
		Transition []string `json:"transition"`
		Quiescence []string `json:"quiescence"`
		Delivery   []string `json:"delivery"`
	} `json:"layouts"`
	Transitions []worldSnapshotTransition `json:"transitions"`
}

type worldSnapshotTransition struct {
	Kind   string   `json:"kind"`
	Tag    uint8    `json:"tag"`
	Fields []string `json:"fields"`
}

type worldSnapshotTemplateData struct {
	Package string
	Schema  worldSnapshotSchema
}

type simulationTimeSchema struct {
	Version uint8 `json:"version"`
	Request struct {
		Magic            string `json:"magic"`
		Bytes            int    `json:"bytes"`
		GenerationOffset int    `json:"generation_offset"`
		CurrentOffset    int    `json:"current_offset"`
		DeadlineOffset   int    `json:"deadline_offset"`
		ArrivalsOffset   int    `json:"arrivals_offset"`
		ReservedOffset   int    `json:"reserved_offset"`
	} `json:"request"`
	Response struct {
		Magic            string `json:"magic"`
		Bytes            int    `json:"bytes"`
		GenerationOffset int    `json:"generation_offset"`
		TimeOffset       int    `json:"time_offset"`
		KindOffset       int    `json:"kind_offset"`
		ReservedOffset   int    `json:"reserved_offset"`
	} `json:"response"`
	Kinds struct {
		Advance  uint8 `json:"advance"`
		Retry    uint8 `json:"retry"`
		Deadlock uint8 `json:"deadlock"`
		External uint8 `json:"external"`
	} `json:"kinds"`
	Golden struct {
		Request  string `json:"request"`
		Response string `json:"response"`
	} `json:"golden"`
	InvalidRequests  []simulationTimeInvalidFrame `json:"invalid_requests"`
	InvalidResponses []simulationTimeInvalidFrame `json:"invalid_responses"`
}

type simulationTimeInvalidFrame struct {
	Name  string `json:"name"`
	Frame string `json:"frame"`
	Error string `json:"error"`
}

type simulationTimeTemplateData struct {
	Package string
	Schema  simulationTimeSchema
}

// choiceImplementationSources are the inputs of the choice codec's
// ImplementationSourceSHA256, in hashing order; gomad_semantics.go.txt is
// appended to the runtime instrumentation input.
var choiceImplementationSources = []string{
	"internal/model/choice/schema/choicewire.json",
	"internal/model/choice/schema/choicewire.go.tmpl",
	"internal/model/choice/schema/choicewire_runtime.go.tmpl",
	"internal/backends/native/toolchain/runtime/overlay/src/runtime/gomad.go",
	"internal/backends/native/toolchain/runtime/go1.27.1.patch",
	"internal/model/choice/trace.go",
	"internal/model/choice/tape.go",
	"internal/model/runtimesource/gomad_semantics.go.txt",
}

func readChoiceImplementationInputs(root string) (choiceImplementationInputs, error) {
	values := make([][]byte, len(choiceImplementationSources))
	for index, relative := range choiceImplementationSources {
		value, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			return choiceImplementationInputs{}, fmt.Errorf("read choice implementation input %s: %w", relative, err)
		}
		values[index] = value
	}
	values[3] = append(append(values[3], '\n'), values[7]...)
	return choiceImplementationInputs{
		Schema: values[0], CodecTemplate: values[1], RuntimeTemplate: values[2], RuntimeOverlay: values[3], ToolchainPatch: values[4],
		HostTrace: values[5], HostTape: values[6],
	}, nil
}

func choiceImplementationIdentity(inputs choiceImplementationInputs) [sha256.Size]byte {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomad-choice-implementation-source-v2"))
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
	if len(definition.Trace.Magic) != 8 || len(definition.Tape.Magic) != 8 || len(definition.Terminal.Magic) != 8 || len(definition.Diagnostic.Magic) != 8 {
		return choiceSchema{}, errors.New("choice wire schema magic must have 8 bytes")
	}
	checks := []bool{
		definition.Version != 0,
		definition.Profile == "gomad-choice-trace/v3",
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
		// The readiness word shares a record's two former reserved bytes: six
		// flag bits and a ten-bit ready count fill the sixteen exactly.
		definition.Readiness.Flags.Known == 1,
		definition.Readiness.Flags.Default == 2,
		definition.Readiness.Flags.NilChannel == 4,
		definition.Readiness.Flags.TimerChannel == 8,
		definition.Readiness.Flags.ClosedChannel == 16,
		definition.Readiness.Flags.RepeatedChannel == 32,
		definition.Readiness.CountShift == 6,
		definition.Readiness.MaximumCount == 1<<(16-6)-1,
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
		definition.Diagnostic.Version != 0,
		definition.Diagnostic.Profile == "gomad-diagnostic-trace/v1",
		definition.Diagnostic.HeaderBytes == 64,
		definition.Diagnostic.RecordBytes == 96,
		definition.Diagnostic.MaximumBytes == 64<<20,
		definition.Diagnostic.States.Complete == 1,
		definition.Diagnostic.States.Overflow == 2,
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
	if definition.Version != 1 || definition.Profile != "gomad.simulation-model/v1" || len(definition.Magic) != 8 || len(definition.TransportMagic) != 8 ||
		definition.Limits.FrameBytes != 128<<20 || definition.Limits.StringBytes != 4<<10 || definition.Limits.DataBytes != 64<<20 || definition.Limits.Entries != 100_000 || definition.Limits.NodeBytes != 256 || definition.Limits.ErrorBytes != 4<<10 ||
		definition.Models.Network != 1 || definition.Models.Volume != 2 ||
		!slices.Equal([]uint16{network.Listen, network.Dial, network.Accept, network.ListenerClose, network.ListenerSetDeadline, network.ConnRead, network.ConnWrite, network.ConnClose, network.ConnCloseRead, network.ConnCloseWrite, network.ConnSetDeadline, network.ConnSetReadDeadline, network.ConnSetWriteDeadline}, sequence16(1, 13)) ||
		!slices.Equal([]uint16{volume.Resolve, volume.Mkdir, volume.MkdirAll, volume.Stat, volume.Open, volume.Rename, volume.Remove, volume.RemoveAll, volume.Chmod, volume.Chtimes, volume.Chdir, volume.Getwd, volume.HandleRead, volume.HandleReadAt, volume.HandleWrite, volume.HandleWriteAt, volume.HandleTruncate, volume.HandleChmod, volume.HandleChtimes, volume.HandleChdir, volume.HandleSeek, volume.HandleStat, volume.HandleReadDir, volume.HandleClose, volume.HandleSync, volume.HandleMap, volume.MappingBytes, volume.MappingClose}, sequence16(1, 28)) ||
		!slices.Equal([]uint16{errorCodes.None, errorCodes.Generic, errorCodes.EOF, errorCodes.Deadline, errorCodes.Canceled, errorCodes.AddressInUse, errorCodes.Closed, errorCodes.ConnectionRefused, errorCodes.ResourceExhausted, errorCodes.Unsupported, errorCodes.EINVAL, errorCodes.EEXIST, errorCodes.ENOENT, errorCodes.ENOTDIR, errorCodes.EISDIR, errorCodes.EROFS, errorCodes.ENOSPC, errorCodes.EBADF, errorCodes.ENODEV, errorCodes.ESTALE, errorCodes.ENOTEMPTY, errorCodes.Capacity, errorCodes.StallTimeout}, sequence16(0, 22)) {
		return simulationModelSchema{}, errors.New("simulation model wire schema is unsupported by this generator")
	}
	return definition, nil
}

func readWorldSnapshotSchema(path string) (worldSnapshotSchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return worldSnapshotSchema{}, fmt.Errorf("read world snapshot wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition worldSnapshotSchema
	if err := decoder.Decode(&definition); err != nil {
		return worldSnapshotSchema{}, fmt.Errorf("decode world snapshot wire schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return worldSnapshotSchema{}, errors.New("world snapshot wire schema has trailing data")
	}
	if definition.Version != 1 || definition.Profile != "gomad.world-compressed-snapshot/v1" || len(definition.Magic) != 8 ||
		definition.Limits.Depth != 4 || definition.Limits.EnumBytes != 32 ||
		!sameStringSet(definition.Layouts.Snapshot, []string{"schema-version", "config.seed", "config.max-requests", "config.max-events", "config.max-queued-events", "config.max-transitions", "config.max-payload-bytes", "config.max-string-bytes", "now", "next-request-id", "next-event-id", "next-transition", "payload-bytes", "replay.cursor", "replay.expected", "transcript-digest", "state-digest"}) ||
		!sameStringSet(definition.Layouts.Request, []string{"kind", "resource.adapter", "resource.kind", "resource.key", "priority", "payload"}) ||
		!sameStringSet(definition.Layouts.Readiness, []string{"request-id", "at", "kind", "payload", "equivalence-class"}) ||
		!sameStringSet(definition.Layouts.Transition, []string{"sequence", "previous-digest", "digest"}) ||
		!sameStringSet(definition.Layouts.Quiescence, []string{"kind", "before", "after", "deliveries", "blocked"}) ||
		!sameStringSet(definition.Layouts.Delivery, []string{"request-id", "event-id", "at", "kind", "payload"}) ||
		!supportedWorldSnapshotTransitions(definition.Transitions) {
		return worldSnapshotSchema{}, errors.New("world snapshot wire schema is unsupported by this generator")
	}
	return definition, nil
}

func sameStringSet(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	want := make(map[string]struct{}, len(expected))
	for _, value := range expected {
		want[value] = struct{}{}
	}
	for _, value := range actual {
		if _, ok := want[value]; !ok {
			return false
		}
		delete(want, value)
	}
	return len(want) == 0
}

func supportedWorldSnapshotTransitions(transitions []worldSnapshotTransition) bool {
	want := map[string]struct {
		tag    uint8
		fields []string
	}{
		"register": {1, []string{"request", "request-id"}},
		"ready":    {2, []string{"readiness", "event-id"}},
		"cancel":   {3, []string{"request-id", "result.request-id", "result.event-id", "status"}},
		"quiesce":  {4, []string{"result"}},
	}
	if len(transitions) != len(want) {
		return false
	}
	for _, transition := range transitions {
		expected, ok := want[transition.Kind]
		if !ok || transition.Tag != expected.tag || !sameStringSet(transition.Fields, expected.fields) {
			return false
		}
		delete(want, transition.Kind)
	}
	return len(want) == 0
}

func readSimulationTimeSchema(path string) (simulationTimeSchema, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return simulationTimeSchema{}, fmt.Errorf("read simulation time wire schema: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition simulationTimeSchema
	if err := decoder.Decode(&definition); err != nil {
		return simulationTimeSchema{}, fmt.Errorf("decode simulation time wire schema: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return simulationTimeSchema{}, errors.New("simulation time wire schema has trailing data")
	}
	request, response := definition.Request, definition.Response
	if definition.Version != 1 || request.Magic != "GOMADTQ\x01" || request.Bytes != 40 || request.GenerationOffset != 8 || request.CurrentOffset != 16 || request.DeadlineOffset != 24 || request.ArrivalsOffset != 32 || request.ReservedOffset != 36 ||
		response.Magic != "GOMADTR\x01" || response.Bytes != 32 || response.GenerationOffset != 8 || response.TimeOffset != 16 || response.KindOffset != 24 || response.ReservedOffset != 25 ||
		definition.Kinds.Advance != 1 || definition.Kinds.Retry != 2 || definition.Kinds.Deadlock != 3 || definition.Kinds.External != 4 {
		return simulationTimeSchema{}, errors.New("simulation time wire schema is unsupported by this generator")
	}
	for _, value := range []struct {
		name, encoded string
		size          int
	}{
		{"request", definition.Golden.Request, request.Bytes}, {"response", definition.Golden.Response, response.Bytes},
	} {
		decoded, err := hex.DecodeString(value.encoded)
		if err != nil || len(decoded) != value.size {
			return simulationTimeSchema{}, fmt.Errorf("simulation time %s golden frame is invalid", value.name)
		}
	}
	for _, vector := range append(definition.InvalidRequests, definition.InvalidResponses...) {
		if _, err := hex.DecodeString(vector.Frame); err != nil || vector.Name == "" || vector.Error == "" {
			return simulationTimeSchema{}, errors.New("simulation time invalid frame vector is invalid")
		}
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

func generate(path string, data any, header string) ([]byte, error) {
	rendered, err := render(path, data, header)
	if err != nil {
		return nil, err
	}
	formatted, err := format.Source(rendered)
	if err != nil {
		return nil, fmt.Errorf("format generated I/O wire codec: %w", err)
	}
	return formatted, nil
}

// render executes a template; header is the generated-file line the template
// reads as {{header}}.
func render(path string, data any, header string) ([]byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read I/O wire template: %w", err)
	}
	functions := template.FuncMap{"bytes": byteLiterals, "hexbytes": hexByteLiterals, "title": exportedIdentifier, "header": func() (string, error) {
		if header == "" {
			return "", errors.New("template requires a generated-file header")
		}
		return header, nil
	}}
	parsed, err := template.New(filepath.Base(path)).Funcs(functions).Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse I/O wire template: %w", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, data); err != nil {
		return nil, fmt.Errorf("render I/O wire template: %w", err)
	}
	return rendered.Bytes(), nil
}

func exportedIdentifier(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

// hexByteLiterals renders bytes as language-neutral hexadecimal integer
// literals, valid in Go and Rust alike.
func hexByteLiterals(value string) string {
	var result bytes.Buffer
	for index := range len(value) {
		if index != 0 {
			result.WriteString(", ")
		}
		fmt.Fprintf(&result, "0x%02x", value[index])
	}
	return result.String()
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
