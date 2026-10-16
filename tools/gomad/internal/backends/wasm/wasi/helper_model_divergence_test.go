package wasi

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestHelperOrdinaryDivergenceDoesNotRequireInspection(t *testing.T) {
	environment := newTestEnvironment(t, testConfig())
	digest := sha256.Sum256(nil)
	divergence := &choice.Divergence{Reason: choice.DivergenceTapeExhausted}
	sink := &frameSink{environment: environment, terminal: &resultFrame{Termination: "divergence", Divergence: divergence, LocalModel: &helperModelResult{Now: environment.now, TranscriptSHA256: hex.EncodeToString(digest[:])}}}
	result := Result{Termination: "divergence"}
	if err := sink.collectHelper(&result); err != nil {
		t.Fatal(err)
	}
	if result.Termination != "divergence" || result.ChoiceDivergence != divergence {
		t.Fatalf("ordinary replay divergence lost: %+v", result)
	}
}

func TestHelperRejectsUnsealedSuccessfulInspection(t *testing.T) {
	sink := &frameSink{request: Request{Runtime: &RuntimeControl{Inspection: []byte(`{}`)}}, terminal: &resultFrame{LocalModel: &helperModelResult{}}}
	if err := sink.collectHelper(&Result{Termination: "inspected_prefix"}); err == nil {
		t.Fatal("accepted an unsealed successful inspection")
	}
}
