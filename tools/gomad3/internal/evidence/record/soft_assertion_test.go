package record

import (
	"testing"

	"go.temporal.io/server/tools/gomad3/record/softassert"
)

func TestSoftAssertionLiteralBracesKeepDistinctSignatures(t *testing.T) {
	var signatures []SHA256
	for _, output := range []string{`failed assertion: invalid {"left":1}`, `failed assertion: invalid {"right":1}`} {
		observed := softassert.Observe([]byte(output), nil, true)
		input := manifestFixture()
		input.Outcome.Reason = "soft_assertion"
		input.Outcome.SoftAssertions = &SoftAssertions{Messages: observed.Messages}
		finalized, _ := finalizedManifest(t, input)
		signatures = append(signatures, finalized.Outcome.FailureSignature)
	}
	if signatures[0] == signatures[1] {
		t.Fatal("literal static JSON text collapsed distinct assertion signatures")
	}
}

func TestSoftAssertionSignatureExcludesDynamicEvidence(t *testing.T) {
	input := manifestFixture()
	input.Outcome.Reason = "soft_assertion"
	input.Outcome.SoftAssertions = &SoftAssertions{Messages: []string{"invalid state"}}
	first, _ := finalizedManifest(t, input)
	input.Seed++
	input.Environment[1].Value = "8"
	input.Streams.Stdout.FullSHA256 = HashBytes([]byte("different timestamp and dynamic tags"))
	input.Streams.Stdout.RetainedSHA256 = input.Streams.Stdout.FullSHA256
	input.Files[1].SHA256 = input.Streams.Stdout.FullSHA256
	second, _ := finalizedManifest(t, input)
	if first.Outcome.FailureSignature != second.Outcome.FailureSignature || first.RecordHash == second.RecordHash {
		t.Fatal("static assertion signature included dynamic evidence, or record omitted it")
	}
	input.Outcome.SoftAssertions.Messages = []string{"different invariant"}
	third, _ := finalizedManifest(t, input)
	if first.Outcome.FailureSignature == third.Outcome.FailureSignature {
		t.Fatal("signature omitted the static assertion message")
	}
}
