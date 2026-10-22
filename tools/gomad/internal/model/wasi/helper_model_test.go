package wasi

import "testing"

func TestHelperModelRejectsRetiredNetworkEvidence(t *testing.T) {
	environment := newTestEnvironment(t, testConfig())
	environment.helperOwned = true
	sink := frameSink{environment: environment, started: &Started{}}
	frame, err := newValidatedFrame([]byte(`{"type":"evidence","stream":"network","data_base64":"AQ=="}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := sink.receiveHelperEvidence(frame); err == nil {
		t.Fatal("retired network v1 helper evidence admitted")
	}
	if len(sink.helperEvidence.choice)+len(sink.helperEvidence.diagnostic)+len(sink.helperEvidence.transcript) != 0 {
		t.Fatal("retired network v1 helper evidence mutated retained streams")
	}
}
