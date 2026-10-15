package wasi

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

// storageStateEnvelope mirrors the helper's encode_state for decoder tests.
func storageStateEnvelope(version, count uint32, sections ...[]byte) []byte {
	data := []byte(storageStateMagic)
	data = binary.LittleEndian.AppendUint32(data, version)
	data = binary.LittleEndian.AppendUint32(data, count)
	for index, section := range sections {
		data = binary.LittleEndian.AppendUint32(data, uint32(index+1))
		data = binary.LittleEndian.AppendUint64(data, uint64(len(section)))
		data = append(data, section...)
	}
	return sealStorageState(data)
}

func sealStorageState(body []byte) []byte {
	digest := sha256.Sum256(body)
	data := binary.LittleEndian.AppendUint64(append([]byte(nil), body...), uint64(len(body)))
	return append(data, digest[:]...)
}

func validStorageSections() [][]byte {
	return [][]byte{
		[]byte(`[{"ID":"data","Mount":"/workspace","CapacityBytes":1024,"Persisted":[],"Volatile":[],"PendingOperations":0,"PendingSHA256":"","NextOperation":1,"Identity":""}]`),
		[]byte(`{"data":[{"ID":1,"Kind":"alloc","Dependencies":null,"Inode":4,"Offset":0,"Bytes":0,"PayloadSHA256":"","EffectSHA256":"sha256:x","Path":"","Destination":""}]}`),
		[]byte(`{"data":{"States":null,"ChoiceExploration":null,"Complete":true,"Capacity":""}}`),
		[]byte(strings.Repeat("a", 64)),
	}
}

func TestStorageEvidenceDecodesHelperEnvelope(t *testing.T) {
	state, err := decodeStorageState(storageStateEnvelope(1, 4, validStorageSections()...))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.snapshots) != 1 || state.snapshots[0].ID != "data" || len(state.operations["data"]) != 1 || !state.enumerations["data"].Complete || state.imageIdentity != strings.Repeat("a", 64) {
		t.Fatalf("decoded state = %+v", state)
	}
	sections := validStorageSections()
	sections[2] = []byte("null")
	state, err = decodeStorageState(storageStateEnvelope(1, 4, sections...))
	if err != nil || state.enumerations != nil {
		t.Fatalf("null enumerations = %+v %v", state.enumerations, err)
	}
}

func TestStorageEvidenceRefusesCorruptTruncatedAndTrailingEnvelopes(t *testing.T) {
	valid := storageStateEnvelope(1, 4, validStorageSections()...)
	body := valid[:len(valid)-40]
	mutate := func(edit func([]byte) []byte) []byte { return edit(append([]byte(nil), valid...)) }
	cases := map[string][]byte{
		"empty":            nil,
		"truncated":        valid[:len(valid)-1],
		"trailing":         append(append([]byte(nil), valid...), 0),
		"digest":           mutate(func(data []byte) []byte { data[len(data)-1] ^= 1; return data }),
		"body-bit":         mutate(func(data []byte) []byte { data[20] ^= 1; return data }),
		"magic":            sealStorageState(append([]byte("GMSTORE\x02"), body[8:]...)),
		"version":          storageStateEnvelope(2, 4, validStorageSections()...),
		"section-count":    storageStateEnvelope(1, 3, validStorageSections()[:3]...),
		"missing-section":  storageStateEnvelope(1, 4, validStorageSections()[:3]...),
		"extra-section":    storageStateEnvelope(1, 4, append(validStorageSections(), []byte("x"))...),
		"body-trailing":    sealStorageState(append(append([]byte(nil), body...), 0)),
		"section-overflow": sealStorageState(append(append([]byte(nil), body[:len(body)-64-8]...), 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f)),
	}
	for _, tamper := range []struct {
		name    string
		section int
		value   string
	}{
		{"snapshots-unknown-field", 0, `[{"ID":"data","Unknown":1}]`},
		{"snapshots-trailing", 0, `[] []`},
		{"operations-null", 1, `null`},
		{"operations-type", 1, `{"data":[{"ID":"one"}]}`},
		{"enumerations-duplicate", 2, `{"data":{"Complete":true,"Complete":false}}`},
		{"identity-short", 3, "abc"},
		{"identity-upper", 3, strings.Repeat("A", 64)},
	} {
		sections := validStorageSections()
		sections[tamper.section] = []byte(tamper.value)
		cases[tamper.name] = storageStateEnvelope(1, 4, sections...)
	}
	for name, data := range cases {
		if _, err := decodeStorageState(data); err == nil || !strings.Contains(err.Error(), "WASI invalid: storage evidence") {
			t.Errorf("%s: accepted or unclassified refusal: %v", name, err)
		}
	}
}

func TestStorageEvidenceCollectionRefusesInconsistentHelperState(t *testing.T) {
	config := storageConfig()
	limits := volume.CrashEnumerationLimits{States: 1, Operations: 1, Depth: 1, Bytes: 1, Work: 1}
	for _, test := range []struct {
		name    string
		request Request
		edit    func(*helperEvidence)
		want    string
	}{
		{"missing-state", Request{Config: config}, func(e *helperEvidence) { e.storageState = nil }, "missing helper storage evidence"},
		{"stream-without-state", Request{Config: config}, func(e *helperEvidence) { e.storageState, e.storage = nil, []byte("x") }, "storage evidence without state"},
		{"unrequested-enumeration", Request{Config: config}, func(*helperEvidence) {}, "enumerations"},
		{"missing-enumeration", Request{Config: config, StorageCrashLimits: &limits}, func(e *helperEvidence) {
			sections := validStorageSections()
			sections[2] = []byte("null")
			e.storageState = storageStateEnvelope(1, 4, sections...)
		}, "enumerations"},
		{"missing-image", Request{Config: config, StorageCrashLimits: &limits, ReturnStorageImage: true}, func(*helperEvidence) {}, "recovery image"},
		{"wrong-image", Request{Config: config, StorageCrashLimits: &limits, ReturnStorageImage: true}, func(e *helperEvidence) { e.storageImage = []byte("x") }, "recovery image"},
		{"unrequested-image", Request{Config: config, StorageCrashLimits: &limits}, func(e *helperEvidence) { e.storageImage = []byte("x") }, "unrequested recovery image"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sink := &frameSink{request: test.request, terminal: &resultFrame{Termination: "exit"}}
			sink.helperEvidence.storageState = storageStateEnvelope(1, 4, validStorageSections()...)
			test.edit(&sink.helperEvidence)
			var result Result
			err := sink.collectHelperStorage(&result)
			if err == nil || !strings.Contains(err.Error(), test.want) || result.StorageImage != nil {
				t.Fatalf("collect = %v, want refusal containing %q", err, test.want)
			}
		})
	}
	sink := &frameSink{request: Request{Config: config, StorageCrashLimits: &limits}, terminal: &resultFrame{Termination: "exit"}}
	sink.helperEvidence.storageState = storageStateEnvelope(1, 4, validStorageSections()...)
	sink.helperEvidence.storage = []byte("{}\n")
	var result Result
	if err := sink.collectHelperStorage(&result); err != nil || result.StorageImage == nil || result.StorageImage.Identity() != strings.Repeat("a", 64) || result.StorageImage.data != nil || string(result.StorageTranscript) != "{}\n" {
		t.Fatalf("valid helper storage evidence = %+v %v", result, err)
	}
	if _, err := NewRecoveredEnvironment(config, result.StorageImage); err == nil {
		t.Fatal("identity-only image admitted for recovery")
	}
}
