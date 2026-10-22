package wasi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	captured "go.temporal.io/server/tools/gomad/internal/model/capture"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

type volumeEnvelopeObservation struct {
	Error         string                             `json:"error"`
	Snapshots     []volume.VolumeSnapshot            `json:"snapshots"`
	Operations    map[string][]volume.Operation      `json:"operations"`
	Enumerations  map[string]volume.CrashEnumeration `json:"enumerations"`
	ImageIdentity string                             `json:"image_identity"`
	LogLength     uint64                             `json:"log_length"`
	LogDigest     []byte                             `json:"log_digest"`
	FailedStage   byte                               `json:"failed_stage"`
	CrashContext  map[string]StorageCrashContext     `json:"crash_context"`
}

func volumeEnvelopeObserve(data []byte) volumeEnvelopeObservation {
	state, err := decodeStorageState(data)
	observation := volumeEnvelopeObservation{Snapshots: state.snapshots, Operations: state.operations, Enumerations: state.enumerations, ImageIdentity: state.imageIdentity, LogLength: state.logLength, LogDigest: slices.Clone(state.logDigest[:]), FailedStage: state.failedStage, CrashContext: state.crashContext}
	if err != nil {
		observation.Error = err.Error()
	}
	return observation
}

func TestVolumeStorageEnvelopeCorpus(t *testing.T) {
	type fixture struct {
		Name     string                    `json:"name"`
		Input    []byte                    `json:"input_base64"`
		Expected volumeEnvelopeObservation `json:"expected"`
	}
	name := filepath.Join("testdata", "volume-conformance", "vectors", "storage-envelopes.json")
	if *volumeConformanceOutput == "" {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		var cases []fixture
		require.NoError(t, json.Unmarshal(data, &cases))
		require.NotEmpty(t, cases)
		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) { require.Equal(t, test.Expected, volumeEnvelopeObserve(test.Input)) })
		}
		return
	}
	valid := storageStateEnvelope(1, 5, validStorageSections()...)
	body := valid[:len(valid)-40]
	mutate := func(index int) []byte { data := slices.Clone(valid); data[index] ^= 1; return data }
	inputs := map[string][]byte{
		"valid": valid, "empty": nil, "truncated": valid[:len(valid)-1], "trailing": append(slices.Clone(valid), 0), "digest": mutate(len(valid) - 1), "body-bit": mutate(20),
		"magic": sealStorageState(append([]byte("GMSTORE\x02"), body[8:]...)), "version": storageStateEnvelope(2, 4, validStorageSections()...), "section-count": storageStateEnvelope(1, 3, validStorageSections()[:3]...), "missing-section": storageStateEnvelope(1, 5, validStorageSections()[:3]...), "extra-section": storageStateEnvelope(1, 5, append(validStorageSections(), []byte("x"))...), "body-trailing": sealStorageState(append(slices.Clone(body), 0)), "section-overflow": sealStorageState(append(slices.Clone(body[:len(body)-40-8]), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f)),
	}
	for _, test := range []struct {
		name    string
		section int
		value   string
	}{
		{"null-enumerations", 2, "null"}, {"snapshots-unknown-field", 0, `[{"ID":"data","Unknown":1}]`}, {"snapshots-trailing", 0, `[] []`}, {"operations-null", 1, `null`}, {"operations-type", 1, `{"data":[{"ID":"one"}]}`}, {"enumerations-duplicate", 2, `{"data":{"Complete":true,"Complete":false}}`}, {"identity-short", 3, "abc"}, {"identity-upper", 3, strings.Repeat("A", 64)}, {"log-binding-short", 4, "x"},
	} {
		sections := validStorageSections()
		sections[test.section] = []byte(test.value)
		inputs[test.name] = storageStateEnvelope(1, 5, sections...)
	}
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	slices.Sort(names)
	var cases []fixture
	for _, name := range names {
		observation := volumeEnvelopeObserve(inputs[name])
		switch name {
		case "valid":
			require.Empty(t, observation.Error)
			require.Len(t, observation.Snapshots, 1)
			require.Equal(t, "data", observation.Snapshots[0].ID)
			require.Len(t, observation.Operations["data"], 1)
			require.True(t, observation.Enumerations["data"].Complete)
			require.Equal(t, strings.Repeat("a", 64), observation.ImageIdentity)
		case "null-enumerations":
			require.Empty(t, observation.Error)
			require.Nil(t, observation.Enumerations)
		default:
			require.Contains(t, observation.Error, "WASI invalid: storage evidence")
		}
		cases = append(cases, fixture{name, inputs[name], observation})
	}
	volumeBoundaryWrite(t, "storage-envelopes.json", cases)
}

func volumeBoundaryWrite(t *testing.T, name string, value any) {
	t.Helper()
	require.True(t, filepath.IsAbs(*volumeConformanceOutput))
	data, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	directory := filepath.Join(*volumeConformanceOutput, "vectors")
	require.NoError(t, os.MkdirAll(directory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, name), append(data, '\n'), 0o644))
}

type volumeCollectionInput struct {
	Config      Config                         `json:"config"`
	Limits      *volume.CrashEnumerationLimits `json:"limits"`
	ReturnImage bool                           `json:"return_image"`
	State       []byte                         `json:"state_base64"`
	Image       []byte                         `json:"image_base64"`
	Log         []byte                         `json:"log_base64"`
}

type volumeCollectionObservation struct {
	Error         string                             `json:"error"`
	ImagePresent  bool                               `json:"image_present"`
	ImageIdentity string                             `json:"image_identity"`
	ImageBytes    []byte                             `json:"image_base64"`
	RecoveryError string                             `json:"recovery_error"`
	Log           []byte                             `json:"log_base64"`
	Snapshots     []volume.VolumeSnapshot            `json:"snapshots"`
	Operations    map[string][]volume.Operation      `json:"operations"`
	Enumerations  map[string]volume.CrashEnumeration `json:"enumerations"`
}

func volumeCollectionObserve(input volumeCollectionInput) volumeCollectionObservation {
	sink := &frameSink{request: Request{Config: input.Config, StorageCrashLimits: input.Limits, ReturnStorageImage: input.ReturnImage}, terminal: &resultFrame{Termination: "exit"}}
	sink.helperEvidence.storageState = input.State
	sink.helperEvidence.storageImage = input.Image
	sink.helperEvidence.storage = input.Log
	var result Result
	err := sink.collectHelperStorage(&result)
	observation := volumeCollectionObservation{Log: result.StorageTranscript, Snapshots: result.StorageSnapshots, Operations: result.StorageOperations, Enumerations: result.StorageCrashEnumeration}
	if err != nil {
		observation.Error = err.Error()
	}
	if result.StorageImage != nil {
		observation.ImagePresent = true
		observation.ImageIdentity = result.StorageImage.Identity()
		observation.ImageBytes = result.StorageImage.data
		_, err := NewRecoveredEnvironment(input.Config, result.StorageImage)
		if err != nil {
			observation.RecoveryError = err.Error()
		}
	}
	return observation
}

func TestVolumeStorageCollectionCorpus(t *testing.T) {
	type fixture struct {
		Name     string                      `json:"name"`
		Input    volumeCollectionInput       `json:"input"`
		Expected volumeCollectionObservation `json:"expected"`
	}
	name := filepath.Join("testdata", "volume-conformance", "vectors", "storage-collection.json")
	if *volumeConformanceOutput == "" {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		var cases []fixture
		require.NoError(t, json.Unmarshal(data, &cases))
		require.NotEmpty(t, cases)
		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) { require.Equal(t, test.Expected, volumeCollectionObserve(test.Input)) })
		}
		return
	}
	limits := volume.CrashEnumerationLimits{States: 1, Operations: 1, Depth: 1, Bytes: 1, Work: 1}
	var cases []fixture
	for _, name := range []string{"missing-state", "stream-without-state", "unrequested-enumeration", "missing-enumeration", "missing-image", "wrong-image", "unrequested-image", "truncated-log", "identity-only-image", "valid-pending-context", "enumeration-unknown-volume", "context-snapshot", "context-operation", "volume-set-count", "volume-set-id", "decoder-error"} {
		input := volumeCollectionInput{Config: storageConfig(), State: storageStateEnvelope(1, 5, validStorageSections()...)}
		if name != "missing-state" && name != "stream-without-state" && name != "unrequested-enumeration" {
			input.Limits = &limits
			input.State = storageStateEnvelope(3, 6, contextStorageSections()...)
		}
		want := ""
		switch name {
		case "missing-state":
			input.State = nil
			want = "missing helper storage evidence"
		case "stream-without-state":
			input.State = nil
			input.Log = []byte("x")
			want = "storage evidence without state"
		case "unrequested-enumeration":
			want = "enumerations"
		case "missing-enumeration":
			sections := validStorageSections()
			sections[2] = []byte("null")
			input.State = storageStateEnvelope(1, 5, sections...)
			want = "enumerations"
		case "missing-image":
			input.ReturnImage = true
			want = "recovery image"
		case "wrong-image":
			input.ReturnImage = true
			input.Image = []byte("x")
			want = "recovery image"
		case "unrequested-image":
			input.Image = []byte("x")
			want = "unrequested recovery image"
		case "truncated-log":
			input.Log = []byte("x")
			want = "storage log length or digest"
		case "identity-only-image":
			input.Log = []byte("{}\n")
			sections := contextStorageSections()
			sections[4] = storageLogBinding(input.Log)
			input.State = storageStateEnvelope(3, 6, sections...)
		case "valid-pending-context", "context-snapshot", "context-operation":
			sections := contextStorageSections()
			sections[0] = []byte(strings.ReplaceAll(string(sections[0]), `"PendingOperations":0`, `"PendingOperations":1`))
			var operations map[string][]volume.Operation
			require.NoError(t, json.Unmarshal(sections[1], &operations))
			if name == "context-operation" {
				operations["data"][0].Kind = "resize"
				want = "crash context operation"
			}
			if name == "context-snapshot" {
				operations["data"] = nil
				want = "crash context"
			}
			encoded, err := json.Marshal(map[string]StorageCrashContext{"data": {PendingSHA256: "sha256:" + strings.Repeat("a", 64), Operations: operations["data"]}})
			require.NoError(t, err)
			sections[5] = encoded
			input.State = storageStateEnvelope(3, 6, sections...)
		case "enumeration-unknown-volume":
			sections := contextStorageSections()
			sections[2] = []byte(`{"other":{"States":null,"ChoiceExploration":null,"Complete":true,"Capacity":""}}`)
			input.State = storageStateEnvelope(3, 6, sections...)
			want = "enumerations"
		case "volume-set-count":
			sections := contextStorageSections()
			sections[0] = []byte(`[]`)
			input.State = storageStateEnvelope(3, 6, sections...)
			want = "volume set"
		case "volume-set-id":
			sections := contextStorageSections()
			sections[0] = []byte(strings.ReplaceAll(string(sections[0]), `"ID":"data"`, `"ID":"other"`))
			input.State = storageStateEnvelope(3, 6, sections...)
			want = "volume set"
		case "decoder-error":
			input.State = []byte("x")
			want = "storage evidence"
		}
		observation := volumeCollectionObserve(input)
		if want != "" {
			require.Contains(t, observation.Error, want)
			require.False(t, observation.ImagePresent)
		} else if name == "identity-only-image" {
			require.Empty(t, observation.Error)
			require.True(t, observation.ImagePresent)
			require.Equal(t, strings.Repeat("a", 64), observation.ImageIdentity)
			require.Nil(t, observation.ImageBytes)
			require.Equal(t, []byte("{}\n"), observation.Log)
			require.NotEmpty(t, observation.RecoveryError)
		} else {
			require.Empty(t, observation.Error)
			require.True(t, observation.ImagePresent)
		}
		cases = append(cases, fixture{name, input, observation})
	}
	volumeBoundaryWrite(t, "storage-collection.json", cases)
}

func TestVolumeStorageRunAdmissionCorpus(t *testing.T) {
	type fixture struct {
		Name      string              `json:"name"`
		Config    Config              `json:"config"`
		Selection map[string][]uint64 `json:"selection"`
		Message   string              `json:"message"`
		Error     string              `json:"error"`
		Started   bool                `json:"started"`
		Accepted  bool                `json:"accepted"`
	}
	observe := func(test fixture) fixture {
		if test.Selection != nil {
			err := ValidateStorageCrashSelection(Request{Config: test.Config, StorageCrashSelection: test.Selection})
			test.Error = ""
			if err != nil {
				test.Error = err.Error()
			}
			test.Started = false
		} else {
			test.Accepted = storageCrashConfigured(test.Config.Storage, test.Message)
		}
		return test
	}
	name := filepath.Join("testdata", "volume-conformance", "vectors", "storage-run-admission.json")
	if *volumeConformanceOutput == "" {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		var cases []fixture
		require.NoError(t, json.Unmarshal(data, &cases))
		require.NotEmpty(t, cases)
		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) { require.Equal(t, test, observe(test)) })
		}
		return
	}
	var cases []fixture
	for _, test := range []struct {
		name      string
		selection map[string][]uint64
		want      string
	}{{"unknown-volume", map[string][]uint64{"unknown": {1}}, "storage crash selection volume"}, {"duplicate", map[string][]uint64{"data": {1, 1}}, "duplicate operation"}, {"zero", map[string][]uint64{"data": {0}}, "zero operation"}} {
		observation := observe(fixture{Name: test.name, Config: storageConfig(), Selection: test.selection})
		require.Contains(t, observation.Error, test.want)
		require.False(t, observation.Started)
		cases = append(cases, observation)
	}
	for _, name := range []string{"configured-crash", "different-crash-name", "missing-crash-fault"} {
		config := storageConfig()
		message := "modeled storage crash: stop"
		if name != "missing-crash-fault" {
			config.Storage.Faults = []StorageFault{{Name: "stop", Resource: "/workspace/db", Operation: "sync", Occurrence: 1, Action: "crash-before"}}
		}
		if name == "different-crash-name" {
			message = "modeled storage crash: other"
		}
		observation := observe(fixture{Name: name, Config: config, Message: message})
		require.Equal(t, name == "configured-crash", observation.Accepted)
		cases = append(cases, observation)
	}
	volumeBoundaryWrite(t, "storage-run-admission.json", cases)
}

func TestVolumeRecoveredConstructorCorpus(t *testing.T) {
	type imageInput struct {
		Data     []byte `json:"data_base64"`
		Model    []byte `json:"model_base64"`
		Identity string `json:"identity"`
	}
	type observation struct {
		Error    string `json:"error"`
		Present  bool   `json:"present"`
		Identity string `json:"identity"`
	}
	type fixture struct {
		Name     string      `json:"name"`
		Config   Config      `json:"config"`
		Image    *imageInput `json:"image"`
		Fresh    bool        `json:"fresh,omitempty"`
		Expected observation `json:"expected"`
	}
	observe := func(test fixture) observation {
		var image *StorageImage
		if test.Image != nil {
			image = &StorageImage{data: test.Image.Data, model: test.Image.Model, identity: test.Image.Identity}
		}
		var e *Environment
		var err error
		if test.Fresh {
			e, err = NewEnvironment(test.Config)
		} else {
			e, err = NewRecoveredEnvironment(test.Config, image)
		}
		var got observation
		if err != nil {
			got.Error = err.Error()
		}
		if e != nil {
			got.Present = true
			got.Identity = e.identity
		}
		return got
	}
	name := filepath.Join("testdata", "volume-conformance", "vectors", "recovered-constructor.json")
	if *volumeConformanceOutput == "" {
		data, err := os.ReadFile(name)
		require.NoError(t, err)
		var cases []fixture
		require.NoError(t, json.Unmarshal(data, &cases))
		require.NotEmpty(t, cases)
		for _, test := range cases {
			t.Run(test.Name, func(t *testing.T) { require.Equal(t, test.Expected, observe(test)) })
		}
		return
	}
	config := storageConfig()
	source := newTestEnvironment(t, config)
	image, err := source.CrashStorage(nil)
	require.NoError(t, err)
	var cases []fixture
	for _, name := range []string{"nil-image", "identity-only", "model-mismatch", "invalid-profile", "authentic-helper-image", "nonempty-storage-mount"} {
		input := &imageInput{Data: slices.Clone(image.data), Model: slices.Clone(image.model), Identity: image.Identity()}
		target := config
		switch name {
		case "nil-image":
			input = nil
		case "identity-only":
			input.Data = nil
		case "model-mismatch":
			input.Model = []byte("{}")
		case "invalid-profile":
			target.Profile = "invalid"
		case "nonempty-storage-mount":
			source := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(source, "input"), []byte("frozen"), 0o600))
			inputs, err := captured.CaptureReadOnlyMountInputs([]captured.Mapping{{Source: source, Target: "/workspace/frozen"}}, captured.DefaultLimits())
			require.NoError(t, err)
			target.CapturedInputs = inputs
		}
		test := fixture{Name: name, Config: target, Image: input, Fresh: name == "nonempty-storage-mount"}
		test.Expected = observe(test)
		if test.Fresh {
			require.Equal(t, "WASI invalid: storage mount must be an empty writable directory", test.Expected.Error)
		}
		if name == "authentic-helper-image" {
			require.Empty(t, test.Expected.Error)
			require.True(t, test.Expected.Present)
		} else {
			require.NotEmpty(t, test.Expected.Error)
			require.False(t, test.Expected.Present)
		}
		cases = append(cases, test)
	}
	volumeBoundaryWrite(t, "recovered-constructor.json", cases)
}
