package record

import (
	"sort"
	"testing"
)

func TestExternalBackendObservedRecordBindsProvenance(t *testing.T) {
	input := manifestFixture()
	payload := []byte(`{"profile":"fixture-observed/v1"}`)
	input.Target.Backend = &BackendMetadata{Name: "fixture", ReplayMode: ReplayObserved, Provenance: BackendPayload{
		Schema: "fixture-provenance/v1", File: "backend/provenance.json", SHA256: HashBytes(payload), Bytes: Uint64String(len(payload)),
	}}
	input.ReplayMode = ReplayObserved
	input.Files = append(input.Files, File{Path: "backend/provenance.json", Mode: "0600", SHA256: HashBytes(payload), Size: Uint64String(len(payload))})
	sort.Slice(input.Files, func(i, j int) bool { return input.Files[i].Path < input.Files[j].Path })
	first, _, err := FinalizeExecutionRecord(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Target.Backend.Provenance.SHA256 = HashBytes([]byte(`{"profile":"changed"}`))
	for index := range input.Files {
		if input.Files[index].Path == input.Target.Backend.Provenance.File {
			input.Files[index].SHA256 = input.Target.Backend.Provenance.SHA256
		}
	}
	second, _, err := FinalizeExecutionRecord(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.RecordHash == second.RecordHash || first.Outcome.FailureSignature == second.Outcome.FailureSignature {
		t.Fatal("backend provenance did not bind execution and failure identity")
	}
	input.Target.Backend = nil
	if _, _, err := FinalizeExecutionRecord(input); err == nil {
		t.Fatal("native record accepted observed replay")
	}
}

func TestExternalBackendMetadataRejectsInvalidReferences(t *testing.T) {
	payload := []byte("fixture")
	base := BackendMetadata{Name: "fixture", ReplayMode: ReplayObserved, Provenance: BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: HashBytes(payload), Bytes: Uint64String(len(payload))}}
	for _, test := range []struct {
		name   string
		change func(*BackendMetadata)
	}{
		{"name", func(m *BackendMetadata) { m.Name = "native" }},
		{"replay-mode", func(m *BackendMetadata) { m.ReplayMode = "invalid" }},
		{"schema", func(m *BackendMetadata) { m.Provenance.Schema = "" }},
		{"escape", func(m *BackendMetadata) { m.Provenance.File = "backend/../target" }},
		{"digest", func(m *BackendMetadata) { m.Provenance.SHA256 = "invalid" }},
		{"empty", func(m *BackendMetadata) { m.Provenance.Bytes = 0 }},
		{"duplicate", func(m *BackendMetadata) { m.Evidence = &m.Provenance }},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := base
			test.change(&m)
			if err := ValidateBackendMetadata(m); err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
	input := manifestFixture()
	input.Target.Backend = CloneBackendMetadata(&base)
	if _, _, err := FinalizeExecutionRecord(input); err == nil {
		t.Fatal("external exact replay claim accepted")
	}
	input.ReplayMode = ReplayObserved
	if _, _, err := FinalizeExecutionRecord(input); err == nil {
		t.Fatal("missing backend provenance file accepted")
	}
}

func TestExternalCooperativeBackendAdmitsExactReplayMetadata(t *testing.T) {
	metadata := BackendMetadata{Name: "fixture-cooperative", ReplayMode: ReplayExact, Provenance: BackendPayload{
		Schema: "fixture/v1", File: "backend/provenance.json", SHA256: HashBytes([]byte("fixture")), Bytes: 7,
	}}
	if err := ValidateBackendMetadata(metadata); err != nil {
		t.Fatal(err)
	}
}
