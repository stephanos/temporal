package record

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

func FinalizeManifest(input Manifest) (Manifest, []byte, error) {
	input.RecordHash = ""
	input.Outcome.FailureSignature = ""
	if err := validateManifest(input, false); err != nil {
		return Manifest{}, nil, err
	}
	failureProjectionBytes, err := CanonicalJSON(failureProjectionOf(input))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("encode failure projection: %w", err)
	}
	input.Outcome.FailureSignature = DomainHash("gomadv3-failure-signature-v2", failureProjectionBytes)
	recordProjectionBytes, err := CanonicalJSON(recordProjectionOf(input))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("encode record projection: %w", err)
	}
	input.RecordHash = DomainHash("gomadv3-run-record-v2", recordProjectionBytes)
	if err := validateManifest(input, true); err != nil {
		return Manifest{}, nil, err
	}
	encoded, err := CanonicalJSON(input)
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("encode manifest: %w", err)
	}
	return input, encoded, nil
}

func DecodeManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := DecodeCanonicalJSON(data, &manifest); err != nil {
		return Manifest{}, err
	}
	wantRecordHash := manifest.RecordHash
	wantFailureSignature := manifest.Outcome.FailureSignature
	finalized, _, err := FinalizeManifest(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if wantRecordHash != finalized.RecordHash {
		return Manifest{}, fmt.Errorf("record hash mismatch: got %s, want %s", wantRecordHash, finalized.RecordHash)
	}
	if wantFailureSignature != finalized.Outcome.FailureSignature {
		return Manifest{}, fmt.Errorf("failure signature mismatch: got %s, want %s", wantFailureSignature, finalized.Outcome.FailureSignature)
	}
	return manifest, nil
}
