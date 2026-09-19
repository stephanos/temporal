package record

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
)

func HashBytes(data []byte) SHA256 {
	digest := sha256.Sum256(data)
	return SHA256FromSum(digest)
}

func SHA256FromSum(digest [sha256.Size]byte) SHA256 {
	return SHA256("sha256:" + hex.EncodeToString(digest[:]))
}

func ParseSHA256(value string) (SHA256, error) {
	identity := SHA256(value)
	if _, err := identity.Bytes(); err != nil {
		return "", err
	}
	return identity, nil
}

func (identity SHA256) Bytes() ([sha256.Size]byte, error) {
	var decoded [sha256.Size]byte
	const prefix = "sha256:"
	value := string(identity)
	if len(value) != len(prefix)+hex.EncodedLen(len(decoded)) || value[:len(prefix)] != prefix {
		return decoded, fmt.Errorf("invalid SHA-256 %q", value)
	}
	hexValue := value[len(prefix):]
	if _, err := hex.Decode(decoded[:], []byte(hexValue)); err != nil || hex.EncodeToString(decoded[:]) != hexValue {
		return [sha256.Size]byte{}, fmt.Errorf("invalid SHA-256 %q", value)
	}
	return decoded, nil
}

func DomainHash(domain string, data []byte) SHA256 {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(domain))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write(data)
	return SHA256("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
}

func NoneWorld() (World, WorldPayloads) {
	payloads := WorldPayloads{
		Initial: []byte("null"),
		Final:   []byte("null"),
	}
	initialHash := HashBytes(payloads.Initial)
	transitionHash := HashBytes(payloads.Transitions)
	finalHash := HashBytes(payloads.Final)
	return World{
		Initial: WorldPayload{
			Schema:         "gomad3.world.snapshot/none",
			File:           "world/snapshot.json",
			RawSHA256:      initialHash,
			SemanticDigest: initialHash,
		},
		Transitions: WorldTransitions{
			Schema:           "gomad3.world.transitions/none",
			File:             "world/transitions.jsonl",
			RawSHA256:        transitionHash,
			TranscriptDigest: transitionHash,
		},
		Final: WorldPayload{
			Schema:         "gomad3.world.snapshot/none",
			File:           "world/final-snapshot.json",
			RawSHA256:      finalHash,
			SemanticDigest: finalHash,
		},
		Adapters: []WorldAdapter{},
		Terminal: WorldTerminal{Kind: "none"},
	}, payloads
}

func FinalizeExecutionRecord(input ExecutionRecord) (ExecutionRecord, []byte, error) {
	if input.Target.CapabilityMode == "" {
		input.Target.CapabilityMode = "closure"
	}
	input.RecordHash = ""
	input.Outcome.FailureSignature = ""
	if err := validateManifest(input, false); err != nil {
		return ExecutionRecord{}, nil, err
	}
	failureProjectionBytes, err := canonicaljson.CanonicalJSON(failureProjectionOf(input))
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode failure projection: %w", err)
	}
	input.Outcome.FailureSignature = DomainHash("gomad3-failure-signature-v1", failureProjectionBytes)
	recordProjectionBytes, err := canonicaljson.CanonicalJSON(recordProjectionOf(input))
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode record projection: %w", err)
	}
	input.RecordHash = DomainHash("gomad3-execution-record-v1", recordProjectionBytes)
	if err := validateManifest(input, true); err != nil {
		return ExecutionRecord{}, nil, err
	}
	encoded, err := canonicaljson.CanonicalJSON(input)
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode manifest: %w", err)
	}
	return input, encoded, nil
}

func DecodeExecutionRecord(data []byte) (ExecutionRecord, error) {
	var manifest ExecutionRecord
	if err := canonicaljson.DecodeCanonicalJSON(data, &manifest); err != nil {
		return ExecutionRecord{}, err
	}
	wantRecordHash := manifest.RecordHash
	wantFailureSignature := manifest.Outcome.FailureSignature
	finalized, _, err := FinalizeExecutionRecord(manifest)
	if err != nil {
		return ExecutionRecord{}, err
	}
	if wantRecordHash != finalized.RecordHash {
		return ExecutionRecord{}, fmt.Errorf("record hash mismatch: got %s, want %s", wantRecordHash, finalized.RecordHash)
	}
	if wantFailureSignature != finalized.Outcome.FailureSignature {
		return ExecutionRecord{}, fmt.Errorf("failure signature mismatch: got %s, want %s", wantFailureSignature, finalized.Outcome.FailureSignature)
	}
	return manifest, nil
}
