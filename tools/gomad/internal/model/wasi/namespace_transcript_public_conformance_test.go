package wasi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
)

type namespaceTranscriptPublicResult struct {
	Reply        json.RawMessage `json:"reply"`
	Transcript   []byte          `json:"transcript_base64"`
	SHA256       string          `json:"sha256"`
	Tag          uint8           `json:"tag"`
	BinaryInput  []byte          `json:"binary_input_base64"`
	BinaryOutput []byte          `json:"binary_output_base64"`
}

type namespaceTranscriptPublicCorpus struct {
	Schema           string                           `json:"schema"`
	Name             string                           `json:"name"`
	Kind             string                           `json:"kind"`
	Op               string                           `json:"op"`
	Input            json.RawMessage                  `json:"input"`
	Limit            uint64                           `json:"limit"`
	ReferenceLimit   uint64                           `json:"reference_limit"`
	DigestTranscript bool                             `json:"digest_transcript"`
	Reference        *namespaceTranscriptPublicResult `json:"reference"`
	Expected         namespaceTranscriptPublicResult  `json:"expected"`
}

func runNamespaceTranscriptPublic(fixture namespaceTranscriptPublicCorpus, limit uint64, digest bool) (namespaceTranscriptPublicResult, error) {
	var result namespaceTranscriptPublicResult
	config := testConfig()
	config.Limits.TranscriptBytes = limit
	environment, err := NewEnvironment(config)
	if err != nil {
		return result, err
	}
	// Run applies Request.DigestTranscript to the same reference environment.
	request := Request{DigestTranscript: digest}
	if request.DigestTranscript {
		environment.digestOnly, environment.transcriptHash = true, sha256.New()
	}
	reply, err := environment.Handle(Call{Type: "call", ID: 1, Op: fixture.Op, Input: fixture.Input})
	if err != nil {
		return result, err
	}
	result.Reply, err = json.Marshal(reply)
	if err != nil {
		return result, err
	}
	result.Transcript = environment.Transcript()
	digestBytes := environment.TranscriptSHA256()
	result.SHA256 = hex.EncodeToString(digestBytes[:])
	result.Tag, err = transcriptTag(fixture.Op)
	if err != nil || int(result.Tag) >= len(transcriptOperations) {
		return result, fmt.Errorf("invalid namespace public transcript operation")
	}
	result.BinaryInput, err = appendTranscriptObject(nil, fixture.Input, transcriptOperations[result.Tag].input)
	if err != nil {
		return result, err
	}
	if reply.Errno == 0 && !reply.Unsupported {
		result.BinaryOutput, err = appendTranscriptObject(nil, reply.Output, transcriptOperations[result.Tag].output)
	}
	return result, err
}

func checkNamespaceTranscriptPublic(fixture namespaceTranscriptPublicCorpus) error {
	if fixture.Schema != namespacePublicCorpusSchema || fixture.Name == "" || fixture.Limit == 0 {
		return fmt.Errorf("invalid namespace public transcript schema/name/limit")
	}
	compactReply := func(result *namespaceTranscriptPublicResult) error {
		var compact bytes.Buffer
		if err := json.Compact(&compact, result.Reply); err != nil {
			return err
		}
		result.Reply = compact.Bytes()
		return nil
	}
	if err := compactReply(&fixture.Expected); err != nil {
		return err
	}
	switch fixture.Kind {
	case "digest":
		if !fixture.DigestTranscript || fixture.Limit != 1 || fixture.Reference == nil || fixture.ReferenceLimit < 2 {
			return fmt.Errorf("invalid digest transcript corpus fields")
		}
		referenceExpected := *fixture.Reference
		if err := compactReply(&referenceExpected); err != nil {
			return err
		}
		fixture.Reference = &referenceExpected
		reference, err := runNamespaceTranscriptPublic(fixture, fixture.ReferenceLimit, false)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(reference, *fixture.Reference) {
			return fmt.Errorf("reference transcript differs from committed bytes/reply/digest")
		}
	case "codec":
		if fixture.DigestTranscript || fixture.Reference != nil || fixture.ReferenceLimit != 0 {
			return fmt.Errorf("invalid codec transcript corpus fields")
		}
	default:
		return fmt.Errorf("invalid namespace public transcript kind")
	}
	actual, err := runNamespaceTranscriptPublic(fixture, fixture.Limit, fixture.DigestTranscript)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, fixture.Expected) {
		return fmt.Errorf("namespace public transcript differs from committed bytes/reply/digest/codec")
	}
	if fixture.Kind == "digest" {
		if len(actual.Transcript) != 0 || actual.SHA256 != fixture.Reference.SHA256 || !bytes.Equal(actual.Reply, fixture.Reference.Reply) {
			return fmt.Errorf("digest-only transcript differs from retained reference")
		}
	} else {
		records, err := DecodeTranscript(actual.Transcript)
		if err != nil || len(records) != 1 || records[0].Op != fixture.Op {
			return fmt.Errorf("codec transcript record mismatch: %v", err)
		}
	}
	return nil
}

func namespaceTranscriptPublicInputs() []namespaceTranscriptPublicCorpus {
	result := []namespaceTranscriptPublicCorpus{}
	for _, input := range []struct {
		op, input string
		limit     uint64
	}{
		{"fd_fdstat_set_flags", `{"fd":3,"flags":2}`, 75},
		{"path_open", `{"fd":3,"dirflags":0,"path_base64":"eA==","oflags":0,"rights_base":2,"rights_inheriting":0,"fdflags":2}`, 96},
		{"path_readlink", `{"fd":3,"path_base64":"eA==","length":0}`, 92},
	} {
		result = append(result, namespaceTranscriptPublicCorpus{Schema: namespacePublicCorpusSchema, Name: input.op + "-unsupported-digest-only-limit1", Kind: "digest", Op: input.op, Input: json.RawMessage(input.input), Limit: 1, ReferenceLimit: input.limit, DigestTranscript: true})
	}
	for _, input := range []struct{ name, op, input string }{
		{"fd_pwrite-stdout-ab-offset9-record", "fd_pwrite", `{"fd":1,"data_base64":"YWI=","offset":9}`},
		{"fd_write-stdout-ab-record", "fd_write", `{"fd":1,"data_base64":"YWI="}`},
	} {
		result = append(result, namespaceTranscriptPublicCorpus{Schema: namespacePublicCorpusSchema, Name: input.name, Kind: "codec", Op: input.op, Input: json.RawMessage(input.input), Limit: testConfig().Limits.TranscriptBytes})
	}
	return result
}

func generateNamespaceTranscriptPublic(t *testing.T, input namespaceTranscriptPublicCorpus) namespaceTranscriptPublicCorpus {
	t.Helper()
	var err error
	if input.Kind == "digest" {
		reference, err := runNamespaceTranscriptPublic(input, input.ReferenceLimit, false)
		if err != nil {
			t.Fatal(err)
		}
		input.Reference = &reference
	}
	input.Expected, err = runNamespaceTranscriptPublic(input, input.Limit, input.DigestTranscript)
	if err != nil {
		t.Fatal(err)
	}
	return input
}

func namespaceTranscriptPublicConformanceCorpus(t *testing.T) {
	t.Helper()
	for _, input := range namespaceTranscriptPublicInputs() {
		t.Run(input.Name, func(t *testing.T) {
			var fixture namespaceTranscriptPublicCorpus
			path := filepath.Join("testdata", "namespace-transcript-conformance", input.Name+".json")
			namespacePublicFixture(t, path, func() any { return generateNamespaceTranscriptPublic(t, input) }, &fixture)
			if fixture.Name != input.Name {
				t.Fatal("namespace transcript case name mismatch")
			}
			if err := checkNamespaceTranscriptPublic(fixture); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNamespacePublicTranscriptRejectsChangedExpectedBytes(t *testing.T) {
	inputs := namespaceTranscriptPublicInputs()
	for _, input := range []namespaceTranscriptPublicCorpus{inputs[0], inputs[3]} {
		t.Run(input.Name, func(t *testing.T) {
			fixture := generateNamespaceTranscriptPublic(t, input)
			if err := checkNamespaceTranscriptPublic(fixture); err != nil {
				t.Fatal(err)
			}
			if input.Kind == "digest" {
				fixture.Expected.SHA256 = "wrong"
			} else {
				fixture.Expected.BinaryInput[0] ^= 1
			}
			if err := checkNamespaceTranscriptPublic(fixture); err == nil {
				t.Fatal("changed committed digest/codec bytes were accepted")
			}
		})
	}
}
