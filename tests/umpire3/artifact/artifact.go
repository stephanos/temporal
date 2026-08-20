package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/evidence"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

const FormatVersion = "umpire3/replay-bundle/v1"

type ReplayMetadata struct {
	Profile      string          `json:"profile,omitempty"`
	Capabilities []string        `json:"capabilities"`
	Seed         int64           `json:"seed"`
	Bounds       protocol.Bounds `json:"bounds"`
	Command      string          `json:"command"`
}

type Record struct {
	FormatVersion string                `json:"formatVersion"`
	Experiment    protocol.Experiment   `json:"experiment"`
	Result        umpire3runtime.Result `json:"result"`
	Replay        ReplayMetadata        `json:"replay"`
}

func Encode(experiment protocol.Experiment, result umpire3runtime.Result, maxBytes int64) ([]byte, error) {
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validate artifact experiment: %w", err)
	}
	if maxBytes <= 0 || maxBytes > experiment.Retention.MaxArtifactBytes {
		maxBytes = experiment.Retention.MaxArtifactBytes
	}
	redacted := redactResult(result)
	digest, err := experiment.Digest()
	if err != nil {
		return nil, err
	}
	if result.ExperimentDigest != digest {
		return nil, errors.New("artifact result is not bound to the experiment")
	}
	encoded, err := json.Marshal(Record{
		FormatVersion: FormatVersion,
		Experiment:    experiment,
		Result:        redacted,
		Replay: ReplayMetadata{
			Profile: redacted.Environment.Name, Capabilities: append([]string(nil), redacted.Environment.Capabilities...),
			Seed: experiment.Scope.Seed, Bounds: experiment.Scope.Bounds,
			Command: "umpire3 replay -bundle <bundle.json>",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encode artifact: %w", err)
	}
	if int64(len(encoded)) > maxBytes {
		return nil, fmt.Errorf("artifact size %d exceeds %d-byte limit", len(encoded), maxBytes)
	}
	return encoded, nil
}

func Decode(encoded []byte, maxBytes int64) (Record, error) {
	if maxBytes <= 0 || int64(len(encoded)) > maxBytes {
		return Record{}, errors.New("replay bundle exceeds decode limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, fmt.Errorf("decode replay bundle: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Record{}, errors.New("replay bundle must contain one JSON document")
	}
	if record.FormatVersion != FormatVersion {
		return Record{}, fmt.Errorf("unsupported replay bundle format %q", record.FormatVersion)
	}
	if err := record.Experiment.Validate(); err != nil {
		return Record{}, fmt.Errorf("validate replay experiment: %w", err)
	}
	digest, err := record.Experiment.Digest()
	if err != nil {
		return Record{}, err
	}
	if record.Result.ExperimentDigest != digest {
		return Record{}, errors.New("replay result is not bound to the experiment")
	}
	if record.Result.Footprint != nil {
		if err := record.Result.Footprint.Validate(); err != nil {
			return Record{}, fmt.Errorf("validate replay learned footprint: %w", err)
		}
	}
	if record.Replay.Seed != record.Experiment.Scope.Seed ||
		record.Replay.Bounds != record.Experiment.Scope.Bounds || record.Replay.Command == "" {
		return Record{}, errors.New("replay metadata does not match the experiment")
	}
	return record, nil
}

func redactResult(result umpire3runtime.Result) umpire3runtime.Result {
	redacted := result
	redacted.Environment.ConfigurationIdentity = digestValue(result.Environment.ConfigurationIdentity)
	redacted.Environment.IsolationIdentity = digestValue(result.Environment.IsolationIdentity)
	redacted.Bindings = redactMap(result.Bindings)
	redacted.Actions = append([]umpire3runtime.ActionResult(nil), result.Actions...)
	for index := range redacted.Actions {
		redacted.Actions[index].Evidence.Reference = digestValue(redacted.Actions[index].Evidence.Reference)
		redacted.Actions[index].Evidence.CausalReferences = redactStrings(redacted.Actions[index].Evidence.CausalReferences)
		redacted.Actions[index].Evidence.EntityIdentity = digestValue(redacted.Actions[index].Evidence.EntityIdentity)
		redacted.Actions[index].Evidence.Lineage = redactStrings(redacted.Actions[index].Evidence.Lineage)
		redacted.Actions[index].Evidence.GroundedBindings = redactMap(redacted.Actions[index].Evidence.GroundedBindings)
	}
	redacted.Observations = append([]environment.Observation(nil), result.Observations...)
	for index := range redacted.Observations {
		redacted.Observations[index].CausalReference = digestValue(redacted.Observations[index].CausalReference)
		redacted.Observations[index].Reference = digestValue(redacted.Observations[index].Reference)
		redacted.Observations[index].CausalReferences = redactStrings(redacted.Observations[index].CausalReferences)
		redacted.Observations[index].EntityIdentity = digestValue(redacted.Observations[index].EntityIdentity)
		redacted.Observations[index].Lineage = redactStrings(redacted.Observations[index].Lineage)
	}
	redacted.Faults = append([]umpire3runtime.FaultResult(nil), result.Faults...)
	for index := range redacted.Faults {
		redacted.Faults[index].SourceIdentity = digestValue(redacted.Faults[index].SourceIdentity)
		redacted.Faults[index].Reference = digestValue(redacted.Faults[index].Reference)
		redacted.Faults[index].EntityIdentity = digestValue(redacted.Faults[index].EntityIdentity)
	}
	redacted.Evidence.Facts = append([]evidence.Fact(nil), result.Evidence.Facts...)
	for index := range redacted.Evidence.Facts {
		redacted.Evidence.Facts[index].Reference = digestValue(redacted.Evidence.Facts[index].Reference)
		redacted.Evidence.Facts[index].CausalReferences = redactStrings(redacted.Evidence.Facts[index].CausalReferences)
		redacted.Evidence.Facts[index].EntityIdentity = digestValue(redacted.Evidence.Facts[index].EntityIdentity)
		redacted.Evidence.Facts[index].Lineage = redactStrings(redacted.Evidence.Facts[index].Lineage)
	}
	redacted.Evidence.Actions = append([]evidence.Action(nil), result.Evidence.Actions...)
	for index := range redacted.Evidence.Actions {
		redacted.Evidence.Actions[index].Reference = digestValue(redacted.Evidence.Actions[index].Reference)
		redacted.Evidence.Actions[index].EntityIdentity = digestValue(redacted.Evidence.Actions[index].EntityIdentity)
		redacted.Evidence.Actions[index].Lineage = redactStrings(redacted.Evidence.Actions[index].Lineage)
	}
	redacted.Evidence.Relations = append([]evidence.Relation(nil), result.Evidence.Relations...)
	for index := range redacted.Evidence.Relations {
		redacted.Evidence.Relations[index].Source = digestValue(redacted.Evidence.Relations[index].Source)
		redacted.Evidence.Relations[index].Target = digestValue(redacted.Evidence.Relations[index].Target)
	}
	redacted.Footprint = redactFootprint(result.Footprint)
	redacted.Cleanup.RecoverableResources = redactMap(result.Cleanup.RecoverableResources)
	return redacted
}

func redactFootprint(report *umpire3fault.Report) *umpire3fault.Report {
	if report == nil {
		return nil
	}
	redacted := *report
	redacted.Calls = append([]umpire3fault.Call(nil), report.Calls...)
	for index := range redacted.Calls {
		redacted.Calls[index].Namespace = digestValue(redacted.Calls[index].Namespace)
		redacted.Calls[index].Participant = digestValue(redacted.Calls[index].Participant)
		redacted.Calls[index].CausalReferences = redactStrings(redacted.Calls[index].CausalReferences)
	}
	redacted.Declared = append([]umpire3fault.Footprint(nil), report.Declared...)
	redacted.AllowedNoise = append([]umpire3fault.Footprint(nil), report.AllowedNoise...)
	redacted.Drift.Missing = append([]umpire3fault.Footprint(nil), report.Drift.Missing...)
	redacted.Drift.Unexpected = append([]umpire3fault.Footprint(nil), report.Drift.Unexpected...)
	return &redacted
}

func redactStrings(values []string) []string {
	if values == nil {
		return nil
	}
	redacted := make([]string, len(values))
	for index, value := range values {
		redacted[index] = digestValue(value)
	}
	return redacted
}

func redactMap[M ~map[string]string](values M) M {
	if values == nil {
		return nil
	}
	redacted := make(M, len(values))
	for key, value := range values {
		redacted[key] = digestValue(value)
	}
	return redacted
}

func digestValue(value string) string {
	if value == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}

type FileCorpus struct {
	root string
}

func NewFileCorpus(root string) *FileCorpus {
	return &FileCorpus{root: root}
}

func (c *FileCorpus) Save(ctx context.Context, experiment protocol.Experiment, result umpire3runtime.Result) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if c.root == "" {
		return "", errors.New("corpus root is required")
	}
	digest, err := experiment.Digest()
	if err != nil {
		return "", err
	}
	encoded, err := Encode(experiment, result, experiment.Retention.MaxArtifactBytes)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(c.root, 0o700); err != nil {
		return "", fmt.Errorf("create corpus directory: %w", err)
	}
	name := strings.TrimPrefix(digest, "sha256:") + ".json"
	path := filepath.Join(c.root, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect corpus entry: %w", err)
	}
	temporary, err := os.CreateTemp(c.root, ".umpire3-artifact-*")
	if err != nil {
		return "", fmt.Errorf("create temporary artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		return "", closeWithError(temporary, fmt.Errorf("protect temporary artifact: %w", err))
	}
	if _, err := temporary.Write(encoded); err != nil {
		return "", closeWithError(temporary, fmt.Errorf("write temporary artifact: %w", err))
	}
	if err := temporary.Sync(); err != nil {
		return "", closeWithError(temporary, fmt.Errorf("sync temporary artifact: %w", err))
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close temporary artifact: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("publish artifact: %w", err)
	}
	return path, nil
}

func closeWithError(file *os.File, operationErr error) error {
	if closeErr := file.Close(); closeErr != nil {
		return errors.Join(operationErr, closeErr)
	}
	return operationErr
}
