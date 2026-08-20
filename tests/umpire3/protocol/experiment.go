package protocol

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const DefaultDecodeLimit int64 = 1 << 20

type Model struct {
	Modules        []string `json:"modules"`
	SourceRevision string   `json:"sourceRevision"`
	SemanticHash   string   `json:"semanticHash"`
	LeanVersion    string   `json:"leanVersion"`
}

type Property struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
	Claim         string `json:"claim"`
}

type Bounds struct {
	MaxDepth   int `json:"maxDepth"`
	MaxResults int `json:"maxResults"`
}

type Assumption struct {
	Identifier    string `json:"identifier"`
	StatementHash string `json:"statementHash"`
}

type Scope struct {
	Bounds      Bounds       `json:"bounds"`
	Assumptions []Assumption `json:"assumptions"`
	Strategy    string       `json:"strategy"`
	Seed        int64        `json:"seed"`
}

type Resource struct {
	Identifier string `json:"identifier"`
	Kind       string `json:"kind"`
}

type Action struct {
	Identifier           string            `json:"identifier"`
	Kind                 string            `json:"kind"`
	Arguments            map[string]string `json:"arguments,omitempty"`
	Bindings             map[string]string `json:"bindings,omitempty"`
	RequiredCapabilities []string          `json:"requiredCapabilities"`
	PreCheckpoint        string            `json:"preCheckpoint,omitempty"`
	PostCheckpoint       string            `json:"postCheckpoint,omitempty"`
}

type Checkpoint struct {
	Identifier     string `json:"identifier"`
	Observation    string `json:"observation"`
	Ordering       string `json:"ordering"`
	OmissionPolicy string `json:"omissionPolicy"`
}

type Provenance struct {
	Kind          string `json:"kind"`
	ProofManifest string `json:"proofManifest"`
}

type Retention struct {
	RedactionClass   string `json:"redactionClass"`
	MaxArtifactBytes int64  `json:"maxArtifactBytes"`
}

type Experiment struct {
	FormatVersion string       `json:"formatVersion"`
	ExperimentID  string       `json:"experimentID"`
	Model         Model        `json:"model"`
	Property      Property     `json:"property"`
	Scope         Scope        `json:"scope"`
	Resources     []Resource   `json:"resources"`
	Actions       []Action     `json:"actions"`
	Checkpoints   []Checkpoint `json:"checkpoints"`
	Provenance    Provenance   `json:"provenance"`
	Retention     Retention    `json:"retention"`
}

var knownActionKinds = map[string]struct{}{
	"schedule-operation":     {},
	"dispatch-task":          {},
	"worker-returns-success": {},
	"request-cancellation":   {},
	"commit-cancellation":    {},
	"persist-success":        {},
	"retry-task":             {},
	"acquire-ownership":      {},
	"crash-owner":            {},
	"recover-owner":          {},
	"ack-task":               {},
	"start-update":           {},
	"accept-update":          {},
	"complete-update":        {},
	"record-update-history":  {},
	"dispatch-workflow-task": {},
	"complete-workflow-task": {},
}

var knownCapabilities = map[string]struct{}{
	"nexus":                 {},
	"nexus-worker-control":  {},
	"nexus-observation":     {},
	"failover-control":      {},
	"update":                {},
	"workflow-task-control": {},
	"history-observation":   {},
}

func DecodeExperiment(reader io.Reader, limit int64) (Experiment, error) {
	if limit <= 0 {
		return Experiment{}, errors.New("experiment decode limit must be positive")
	}
	encoded, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return Experiment{}, fmt.Errorf("read experiment: %w", err)
	}
	if int64(len(encoded)) > limit {
		return Experiment{}, fmt.Errorf("experiment exceeds %d-byte decode limit", limit)
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var experiment Experiment
	if err := decoder.Decode(&experiment); err != nil {
		return Experiment{}, fmt.Errorf("decode experiment: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return Experiment{}, err
	}
	if err := experiment.Validate(); err != nil {
		return Experiment{}, err
	}
	return experiment, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode experiment: multiple JSON values")
		}
		return fmt.Errorf("decode experiment trailer: %w", err)
	}
	return nil
}

func (e Experiment) Validate() error {
	if e.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported experiment format version %q", e.FormatVersion)
	}
	if e.ExperimentID == "" {
		return errors.New("experiment ID is required")
	}
	if len(e.Model.Modules) == 0 || e.Model.SourceRevision == "" || e.Model.LeanVersion == "" {
		return errors.New("complete model provenance is required")
	}
	if !validHash(e.Model.SemanticHash) {
		return errors.New("model semantic hash must be a sha256 digest")
	}
	if e.Property.Identifier == "" || !validHash(e.Property.StatementHash) {
		return errors.New("complete property provenance is required")
	}
	if e.Property.Claim != "implementation-conformance" {
		return fmt.Errorf("unknown requested claim %q", e.Property.Claim)
	}
	if e.Scope.Bounds.MaxDepth <= 0 || e.Scope.Bounds.MaxResults <= 0 || e.Scope.Strategy == "" {
		return errors.New("positive exploration bounds and strategy are required")
	}
	for _, assumption := range e.Scope.Assumptions {
		if assumption.Identifier == "" || !validHash(assumption.StatementHash) {
			return errors.New("every assumption requires an identifier and sha256 statement hash")
		}
	}
	if len(e.Resources) == 0 || len(e.Actions) == 0 || len(e.Checkpoints) == 0 {
		return errors.New("resources, actions, and checkpoints are required")
	}

	checkpointIDs := make(map[string]struct{}, len(e.Checkpoints))
	for _, checkpoint := range e.Checkpoints {
		if checkpoint.Identifier == "" || checkpoint.Observation == "" {
			return errors.New("checkpoint identifier and observation are required")
		}
		if checkpoint.Ordering != "causal" && checkpoint.Ordering != "source-sequence" && checkpoint.Ordering != "none" {
			return fmt.Errorf("unknown checkpoint ordering %q", checkpoint.Ordering)
		}
		if checkpoint.OmissionPolicy != "required" && checkpoint.OmissionPolicy != "optional" {
			return fmt.Errorf("unknown omission policy %q", checkpoint.OmissionPolicy)
		}
		if _, duplicate := checkpointIDs[checkpoint.Identifier]; duplicate {
			return fmt.Errorf("duplicate checkpoint %q", checkpoint.Identifier)
		}
		checkpointIDs[checkpoint.Identifier] = struct{}{}
	}

	actionIDs := make(map[string]struct{}, len(e.Actions))
	for _, action := range e.Actions {
		if action.Identifier == "" {
			return errors.New("action identifier is required")
		}
		if _, duplicate := actionIDs[action.Identifier]; duplicate {
			return fmt.Errorf("duplicate action %q", action.Identifier)
		}
		actionIDs[action.Identifier] = struct{}{}
		if _, known := knownActionKinds[action.Kind]; !known {
			return fmt.Errorf("unknown action kind %q", action.Kind)
		}
		for _, capability := range action.RequiredCapabilities {
			if _, known := knownCapabilities[capability]; !known {
				return fmt.Errorf("unknown capability %q", capability)
			}
		}
		for _, values := range []map[string]string{action.Arguments, action.Bindings} {
			for key := range values {
				if sensitiveField(key) {
					return fmt.Errorf("action %q contains sensitive field %q", action.Identifier, key)
				}
			}
		}
		for _, checkpoint := range []string{action.PreCheckpoint, action.PostCheckpoint} {
			if checkpoint != "" {
				if _, exists := checkpointIDs[checkpoint]; !exists {
					return fmt.Errorf("action %q references unknown checkpoint %q", action.Identifier, checkpoint)
				}
			}
		}
	}
	if e.Provenance.Kind != "proof" && e.Provenance.Kind != "bounded-exploration" &&
		e.Provenance.Kind != "counterexample" && e.Provenance.Kind != "curated-trace" {
		return fmt.Errorf("unknown provenance kind %q", e.Provenance.Kind)
	}
	if e.Provenance.ProofManifest == "" {
		return errors.New("proof manifest is required")
	}
	if e.Retention.RedactionClass != "semantic-only" || e.Retention.MaxArtifactBytes <= 0 {
		return errors.New("bounded semantic-only retention is required")
	}
	return nil
}

func sensitiveField(field string) bool {
	normalized := strings.ToLower(field)
	for _, fragment := range []string{"authorization", "credential", "header", "password", "payload", "secret", "token"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}

func validHash(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
}

func (e Experiment) CanonicalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encode canonical experiment: %w", err)
	}
	return encoded, nil
}

func (e Experiment) Digest() (string, error) {
	encoded, err := e.CanonicalJSON()
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
}
