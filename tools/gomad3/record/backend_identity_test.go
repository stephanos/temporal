package record

import "testing"

func TestPreparedBackendIdentityExcludesOnlyExecutionEvidence(t *testing.T) {
	prepared := manifestFixture().Target
	prepared.Backend = &BackendMetadata{Name: "fixture", ReplayMode: ReplayObserved, Provenance: BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: HashBytes([]byte("source")), Bytes: 6}}
	executed := prepared
	executed.Backend = CloneBackendMetadata(prepared.Backend)
	executed.Backend.Evidence = &BackendPayload{Schema: "evidence/v1", File: "backend/evidence.bin", SHA256: HashBytes([]byte("run")), Bytes: 3}
	matches, err := SamePreparedTargetIdentity(prepared, executed)
	if err != nil || !matches || executed.Backend.Evidence == nil {
		t.Fatalf("preparation comparison changed evidence or identity: %v %v", matches, err)
	}
	matches, err = SameTargetIdentity(prepared, executed)
	if err != nil || matches {
		t.Fatalf("full execution comparison lost evidence: %v %v", matches, err)
	}
	executed.Backend.Provenance.SHA256 = HashBytes([]byte("changed"))
	matches, err = SamePreparedTargetIdentity(prepared, executed)
	if err != nil || matches {
		t.Fatalf("changed preparation provenance accepted: %v %v", matches, err)
	}
	prepared.Backend = nil
	executed = prepared
	matches, err = SamePreparedTargetIdentity(prepared, executed)
	if err != nil || !matches {
		t.Fatalf("native preparation comparison changed: %v %v", matches, err)
	}
}
