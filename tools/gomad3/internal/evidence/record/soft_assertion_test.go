package record

import "testing"

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
