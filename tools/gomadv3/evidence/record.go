package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
			Schema:         "gomadv3.world.snapshot/none",
			File:           "world/snapshot.json",
			RawSHA256:      initialHash,
			SemanticDigest: initialHash,
		},
		Transitions: WorldTransitions{
			Schema:           "gomadv3.world.transitions/none",
			File:             "world/transitions.jsonl",
			RawSHA256:        transitionHash,
			TranscriptDigest: transitionHash,
		},
		Final: WorldPayload{
			Schema:         "gomadv3.world.snapshot/none",
			File:           "world/final-snapshot.json",
			RawSHA256:      finalHash,
			SemanticDigest: finalHash,
		},
		Adapters: []WorldAdapter{},
		Terminal: WorldTerminal{Kind: "none"},
	}, payloads
}

func FinalizeExecutionRecord(input ExecutionRecord) (ExecutionRecord, []byte, error) {
	if input.SchemaVersion == SchemaVersion && input.Target.CapabilityMode == "" {
		input.Target.CapabilityMode = "closure"
	}
	input.RecordHash = ""
	input.Outcome.FailureSignature = ""
	if err := validateManifest(input, false); err != nil {
		return ExecutionRecord{}, nil, err
	}
	failureProjectionBytes, err := CanonicalJSON(failureProjectionOf(input))
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode failure projection: %w", err)
	}
	failureDomain := "gomadv3-failure-signature-v5"
	recordDomain := "gomadv3-run-record-v5"
	if input.SchemaVersion == PriorSchemaVersion {
		failureDomain = "gomadv3-failure-signature-v4"
		recordDomain = "gomadv3-run-record-v4"
	} else if input.SchemaVersion == PreviousSchemaVersion {
		failureDomain = "gomadv3-failure-signature-v3"
		recordDomain = "gomadv3-run-record-v3"
	} else if input.SchemaVersion == LegacySchemaVersion {
		failureDomain = "gomadv3-failure-signature-v2"
		recordDomain = "gomadv3-run-record-v2"
	}
	input.Outcome.FailureSignature = DomainHash(failureDomain, failureProjectionBytes)
	recordProjectionBytes, err := CanonicalJSON(recordProjectionOf(input))
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode record projection: %w", err)
	}
	input.RecordHash = DomainHash(recordDomain, recordProjectionBytes)
	if err := validateManifest(input, true); err != nil {
		return ExecutionRecord{}, nil, err
	}
	encoded, err := CanonicalJSON(input)
	if err != nil {
		return ExecutionRecord{}, nil, fmt.Errorf("encode manifest: %w", err)
	}
	return input, encoded, nil
}

func DecodeExecutionRecord(data []byte) (ExecutionRecord, error) {
	var manifest ExecutionRecord
	if err := DecodeCanonicalJSON(data, &manifest); err != nil {
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
	if manifest.SchemaVersion != SchemaVersion {
		manifest.Target.CapabilityMode = "closure"
	}
	return manifest, nil
}
