package finite

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"go.temporal.io/server/tests/umpire3/internal/artifactio"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolchecker "go.temporal.io/server/tests/umpire3/protocol/checker"
)

const CheckpointFormatVersion = "umpire3/native-checkpoint/v1"

type SearchLimits struct {
	MaxDepth       int `json:"maxDepth"`
	MaxStates      int `json:"maxStates"`
	MaxTransitions int `json:"maxTransitions"`
	MaxStateBytes  int `json:"maxStateBytes"`
}

type ExpandedNode struct {
	Replica int                             `json:"replica"`
	State   protocolchecker.FirstOrderState `json:"state"`
	Parent  int                             `json:"parent"`
	Action  protocolcatalog.ActionKind      `json:"action,omitempty"`
	Depth   int                             `json:"depth"`
}

type Checkpoint struct {
	FormatVersion  string         `json:"formatVersion"`
	ViewDigest     string         `json:"viewDigest"`
	SemanticHash   string         `json:"semanticHash"`
	Replicas       int            `json:"replicas"`
	Limits         SearchLimits   `json:"limits"`
	CompletedDepth int            `json:"completedDepth"`
	Nodes          []ExpandedNode `json:"nodes"`
	Frontier       []int          `json:"frontier"`
	Transitions    int            `json:"transitions"`
	StateBytes     int            `json:"stateBytes"`
	Digest         string         `json:"digest"`
}

func SaveCheckpoint(path string, checkpoint Checkpoint) error {
	encoded, err := checkpoint.CanonicalJSON()
	if err != nil {
		return err
	}
	return WriteArtifact(path, append(encoded, '\n'))
}

func WriteArtifact(path string, encoded []byte) error {
	return artifactio.Publish(path, encoded)
}

func LoadCheckpoint(path string, limit int64) (Checkpoint, error) {
	input, err := os.Open(path)
	if err != nil {
		return Checkpoint{}, fmt.Errorf("open native checkpoint: %w", err)
	}
	var checkpoint Checkpoint
	decodeErr := decodeStrict(input, limit, &checkpoint)
	closeErr := input.Close()
	if decodeErr != nil || closeErr != nil {
		return Checkpoint{}, fmt.Errorf("decode native checkpoint: %w", errors.Join(decodeErr, closeErr))
	}
	if err := checkpoint.validateDigest(); err != nil {
		return Checkpoint{}, err
	}
	return checkpoint, nil
}

func (c Checkpoint) CanonicalJSON() ([]byte, error) {
	if err := c.validateDigest(); err != nil {
		return nil, err
	}
	return json.Marshal(c)
}

func (c *Checkpoint) seal() error {
	c.Digest = ""
	encoded, err := json.Marshal(c)
	if err != nil {
		return err
	}
	c.Digest = digest(encoded)
	return nil
}

func (c Checkpoint) validateDigest() error {
	if c.FormatVersion != CheckpointFormatVersion || !digestPattern.MatchString(c.ViewDigest) ||
		!digestPattern.MatchString(c.SemanticHash) || c.Replicas <= 0 || c.Replicas > 10 ||
		c.Limits.MaxDepth <= 0 || c.Limits.MaxStates <= 0 || c.Limits.MaxTransitions <= 0 ||
		c.Limits.MaxStateBytes <= 0 || c.CompletedDepth < -1 || len(c.Nodes) == 0 ||
		c.Frontier == nil || c.Transitions < 0 || c.StateBytes <= 0 || !digestPattern.MatchString(c.Digest) {
		return errors.New("complete native checkpoint identity, limits, and search state are required")
	}
	expected := c
	if err := expected.seal(); err != nil || expected.Digest != c.Digest {
		return errors.New("native checkpoint digest does not match its contents")
	}
	return nil
}

func stateEncodedSize(state protocolchecker.FirstOrderState) (int, error) {
	encoded, err := json.Marshal(state)
	return len(encoded), err
}
