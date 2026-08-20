package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type ProofDependency struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
}

type ProofManifest struct {
	FormatVersion string            `json:"formatVersion"`
	Identifier    string            `json:"identifier"`
	Theorem       string            `json:"theorem"`
	StatementHash string            `json:"statementHash"`
	SemanticHash  string            `json:"semanticHash"`
	LeanVersion   string            `json:"leanVersion"`
	Assumptions   []ProofDependency `json:"assumptions"`
}

func DecodeProofManifest(reader io.Reader, limit int64) (ProofManifest, error) {
	var manifest ProofManifest
	if err := decodeStrictJSON(reader, limit, "proof manifest", &manifest); err != nil {
		return ProofManifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return ProofManifest{}, err
	}
	return manifest, nil
}

func (m ProofManifest) Validate() error {
	if m.FormatVersion != FormatVersion || m.Identifier == "" || m.Theorem == "" || m.LeanVersion == "" {
		return errors.New("complete proof manifest identity is required")
	}
	if !validHash(m.StatementHash) || !validHash(m.SemanticHash) {
		return errors.New("proof statement and semantic hashes must be sha256 digests")
	}
	for _, assumption := range m.Assumptions {
		if assumption.Identifier == "" || !validHash(assumption.StatementHash) {
			return errors.New("complete proof assumption is required")
		}
	}
	return nil
}

func (m ProofManifest) Digest() (string, error) {
	if err := m.Validate(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("encode proof manifest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}

func (m ProofManifest) CanonicalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode proof manifest: %w", err)
	}
	return encoded, nil
}
