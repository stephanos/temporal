package trace

import (
	"bytes"
	"testing"
)

type testState struct {
	Value int `json:"value"`
}

type testAction struct {
	Name string `json:"name"`
}

type testDelta struct {
	Value int `json:"value"`
}

func TestCodecProducesStableSemanticAndCorpusDigests(t *testing.T) {
	codec := testCodec()
	first := testCorpus()
	second := testCorpus()
	second.Coverage[0], second.Coverage[1] = second.Coverage[1], second.Coverage[0]
	second.Traces[0], second.Traces[1] = second.Traces[1], second.Traces[0]

	if err := codec.Finalize(&first); err != nil {
		t.Fatal(err)
	}
	if err := codec.Finalize(&second); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := codec.Encode(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := codec.Encode(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("canonical corpus differs:\n%s\n%s", firstBytes, secondBytes)
	}
	if first.SemanticDigest == "" || first.CorpusDigest == "" || first.SemanticDigest == first.CorpusDigest {
		t.Fatalf("unexpected digests: semantic=%q corpus=%q", first.SemanticDigest, first.CorpusDigest)
	}
}

func TestCodecRejectsSemanticPayloadWithStaleDigest(t *testing.T) {
	codec := testCodec()
	corpus := testCorpus()
	if err := codec.Finalize(&corpus); err != nil {
		t.Fatal(err)
	}
	encoded, err := codec.Encode(corpus)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(encoded, []byte(`"increment"`), []byte(`"decrement"`), 1)
	if _, err := codec.Decode(changed); err == nil {
		t.Fatal("Decode() accepted changed semantic payload")
	}
}

func testCodec() Codec[testState, testAction, testDelta, string] {
	return Codec[testState, testAction, testDelta, string]{
		Subject:              "counter",
		CorpusSchema:         "counter.corpus/v1",
		SemanticDigestDomain: "counter.corpus-semantic/v1",
		CorpusDigestDomain:   "counter.corpus/v1",
	}
}

func testCorpus() Corpus[testState, testAction, testDelta, string] {
	return Corpus[testState, testAction, testDelta, string]{
		Schema: "counter.corpus/v1",
		Generation: GenerationContract{
			Schema: "counter.generation/v1", ModelIdentity: "sha256:model", BoundsIdentity: "sha256:bounds",
		},
		BaselineIdentity: "sha256:baseline",
		Coverage:         []string{"action.increment", "action.noop"},
		Traces: []BehaviorTrace[testState, testAction, testDelta]{
			{
				Name: "increment", InitialState: testState{},
				Steps: []StepRecord[testAction, testDelta]{{
					Ordinal: 0, Action: testAction{Name: "increment"}, PreStateIdentity: "state:0", PostStateIdentity: "state:1", ObservableDelta: testDelta{Value: 1},
				}},
			},
			{Name: "empty", InitialState: testState{Value: 5}},
		},
		Rejections: []RejectionCase[testState, testAction, string]{{
			Name: "unknown", InitialState: testState{}, Action: testAction{Name: "unknown"}, PreStateIdentity: "state:0", Code: "unknown_action",
		}},
	}
}
