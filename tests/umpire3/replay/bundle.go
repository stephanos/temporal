package replay

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

	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/execution/evidence"
	umpire3fault "go.temporal.io/server/tests/umpire3/execution/fault"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	"go.temporal.io/server/tests/umpire3/internal/artifactio"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

const BundleFormatVersion = "umpire3/replay-bundle/v3"

type Metadata struct {
	Profile      string                         `json:"profile,omitempty"`
	Capabilities []protocolcatalog.CapabilityID `json:"capabilities"`
	Seed         int64                          `json:"seed"`
	Bounds       protocolexperiment.Bounds      `json:"bounds"`
	Command      string                         `json:"command"`
}

type Bundle struct {
	FormatVersion string                        `json:"formatVersion"`
	Experiment    protocolexperiment.Experiment `json:"experiment"`
	Result        execution.Result              `json:"result"`
	Replay        Metadata                      `json:"replay"`
}

func EncodeBundle(experiment protocolexperiment.Experiment, result execution.Result, maxBytes int64) ([]byte, error) {
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validate artifact experiment: %w", err)
	}
	if maxBytes <= 0 || maxBytes > experiment.Retention.MaxArtifactBytes {
		maxBytes = experiment.Retention.MaxArtifactBytes
	}
	if result.FormatVersion != execution.ResultFormatVersion {
		return nil, fmt.Errorf("unsupported artifact runtime result format %q", result.FormatVersion)
	}
	if err := result.ValidateAssurance(); err != nil {
		return nil, fmt.Errorf("validate artifact result assurance: %w", err)
	}
	redacted, err := redactResult(result)
	if err != nil {
		return nil, err
	}
	digest, err := experiment.Digest()
	if err != nil {
		return nil, err
	}
	if result.ExperimentDigest != digest {
		return nil, errors.New("artifact result is not bound to the experiment")
	}
	encoded, err := json.Marshal(Bundle{
		FormatVersion: BundleFormatVersion,
		Experiment:    experiment,
		Result:        redacted,
		Replay: Metadata{
			Profile:      redacted.Environment.Name,
			Capabilities: append([]protocolcatalog.CapabilityID(nil), redacted.Environment.Capabilities...),
			Seed:         experiment.Scope.Seed, Bounds: experiment.Scope.Bounds,
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

func DecodeBundle(encoded []byte, maxBytes int64) (Bundle, error) {
	if maxBytes <= 0 || int64(len(encoded)) > maxBytes {
		return Bundle{}, errors.New("replay bundle exceeds decode limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var record Bundle
	if err := decoder.Decode(&record); err != nil {
		return Bundle{}, fmt.Errorf("decode replay bundle: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return Bundle{}, errors.New("replay bundle must contain one JSON document")
	}
	if record.FormatVersion != BundleFormatVersion {
		return Bundle{}, fmt.Errorf("unsupported replay bundle format %q", record.FormatVersion)
	}
	if err := record.Experiment.Validate(); err != nil {
		return Bundle{}, fmt.Errorf("validate replay experiment: %w", err)
	}
	digest, err := record.Experiment.Digest()
	if err != nil {
		return Bundle{}, err
	}
	if record.Result.ExperimentDigest != digest {
		return Bundle{}, errors.New("replay result is not bound to the experiment")
	}
	if record.Result.FormatVersion != execution.ResultFormatVersion {
		return Bundle{}, fmt.Errorf("unsupported replay runtime result format %q", record.Result.FormatVersion)
	}
	if err := record.Result.ValidateAssurance(); err != nil {
		return Bundle{}, fmt.Errorf("validate replay result assurance: %w", err)
	}
	if resultHasStoredEvidence(record.Result) {
		if err := record.Result.ValidateEvidenceDigest(); err != nil {
			return Bundle{}, fmt.Errorf("validate replay result evidence digest: %w", err)
		}
	}
	if record.Result.Footprint != nil {
		if err := record.Result.Footprint.Validate(); err != nil {
			return Bundle{}, fmt.Errorf("validate replay learned footprint: %w", err)
		}
	}
	if record.Replay.Seed != record.Experiment.Scope.Seed ||
		record.Replay.Bounds != record.Experiment.Scope.Bounds || record.Replay.Command == "" {
		return Bundle{}, errors.New("replay metadata does not match the experiment")
	}
	return record, nil
}

func redactResult(result execution.Result) (execution.Result, error) {
	redacted := result
	redacted.Environment.ConfigurationIdentity = digestValue(result.Environment.ConfigurationIdentity)
	redacted.Environment.IsolationIdentity = digestValue(result.Environment.IsolationIdentity)
	redacted.Bindings = redactMap(result.Bindings)
	redacted.Actions = append([]execution.ActionResult(nil), result.Actions...)
	for index := range redacted.Actions {
		redacted.Actions[index].Evidence.SourceIdentity = digestValue(redacted.Actions[index].Evidence.SourceIdentity)
		redacted.Actions[index].Evidence.Reference = digestValue(redacted.Actions[index].Evidence.Reference)
		redacted.Actions[index].Evidence.CausalReferences = redactStrings(redacted.Actions[index].Evidence.CausalReferences)
		redacted.Actions[index].Evidence.EntityIdentity = digestValue(redacted.Actions[index].Evidence.EntityIdentity)
		redacted.Actions[index].Evidence.Lineage = redactStrings(redacted.Actions[index].Evidence.Lineage)
		redacted.Actions[index].Evidence.GroundedBindings = redactMap(redacted.Actions[index].Evidence.GroundedBindings)
	}
	redacted.Observations = append([]execution.Observation(nil), result.Observations...)
	for index := range redacted.Observations {
		redacted.Observations[index].SourceIdentity = digestValue(redacted.Observations[index].SourceIdentity)
		redacted.Observations[index].CausalReference = digestValue(redacted.Observations[index].CausalReference)
		redacted.Observations[index].Reference = digestValue(redacted.Observations[index].Reference)
		redacted.Observations[index].CausalReferences = redactStrings(redacted.Observations[index].CausalReferences)
		redacted.Observations[index].EntityIdentity = digestValue(redacted.Observations[index].EntityIdentity)
		redacted.Observations[index].Lineage = redactStrings(redacted.Observations[index].Lineage)
	}
	factIdentifiers := make(map[string]string, len(result.Facts))
	redacted.Facts = append([]observation.Fact(nil), result.Facts...)
	for index := range redacted.Facts {
		fact := &redacted.Facts[index]
		factIdentifiers[fact.Identifier] = digestValue(fact.Identifier)
		fact.Identifier = factIdentifiers[fact.Identifier]
		fact.Source.Identity = digestValue(fact.Source.Identity)
		fact.Source.Reference = digestValue(fact.Source.Reference)
		fact.Source.CausalReferences = redactStrings(fact.Source.CausalReferences)
		fact.Source.EntityIdentity = digestValue(fact.Source.EntityIdentity)
		fact.Source.Lineage = redactStrings(fact.Source.Lineage)
		if fact.History != nil {
			history := *fact.History
			history.WorkflowID = digestValue(history.WorkflowID)
			history.RunID = digestValue(history.RunID)
			history.OperationID = digestValue(history.OperationID)
			fact.History = &history
		}
		if fact.Mechanism != nil {
			mechanism := *fact.Mechanism
			mechanism.Resource = digestValue(mechanism.Resource)
			fact.Mechanism = &mechanism
		}
		if fact.Window != nil {
			window := *fact.Window
			fact.Window = &window
		}
	}
	for index := range redacted.Observations {
		redacted.Observations[index].SupportingFacts = redactFactIdentifiers(
			redacted.Observations[index].SupportingFacts, factIdentifiers)
	}
	redacted.Faults = append([]execution.FaultResult(nil), result.Faults...)
	for index := range redacted.Faults {
		redacted.Faults[index].SourceIdentity = digestValue(redacted.Faults[index].SourceIdentity)
		redacted.Faults[index].Reference = digestValue(redacted.Faults[index].Reference)
		redacted.Faults[index].EntityIdentity = digestValue(redacted.Faults[index].EntityIdentity)
	}
	redacted.Evidence.Facts = append([]evidence.Fact(nil), result.Evidence.Facts...)
	for index := range redacted.Evidence.Facts {
		redacted.Evidence.Facts[index].SourceIdentity = digestValue(redacted.Evidence.Facts[index].SourceIdentity)
		redacted.Evidence.Facts[index].Reference = digestValue(redacted.Evidence.Facts[index].Reference)
		redacted.Evidence.Facts[index].CausalReferences = redactStrings(redacted.Evidence.Facts[index].CausalReferences)
		redacted.Evidence.Facts[index].EntityIdentity = digestValue(redacted.Evidence.Facts[index].EntityIdentity)
		redacted.Evidence.Facts[index].Lineage = redactStrings(redacted.Evidence.Facts[index].Lineage)
	}
	redacted.Evidence.Actions = append([]evidence.Action(nil), result.Evidence.Actions...)
	for index := range redacted.Evidence.Actions {
		redacted.Evidence.Actions[index].SourceIdentity = digestValue(redacted.Evidence.Actions[index].SourceIdentity)
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
	if resultHasStoredEvidence(redacted) {
		if err := redacted.BindEvidenceDigest(); err != nil {
			return execution.Result{}, fmt.Errorf("bind redacted evidence digest: %w", err)
		}
	}
	return redacted, nil
}

func resultHasStoredEvidence(result execution.Result) bool {
	return result.EvidenceDigest != "" || result.Trace != nil ||
		len(result.Facts) != 0 || len(result.Observations) != 0 ||
		len(result.Evidence.Facts) != 0 || len(result.Evidence.Actions) != 0 ||
		len(result.Evidence.Relations) != 0 || len(result.Evidence.Claims) != 0
}

func redactFactIdentifiers(values []string, replacements map[string]string) []string {
	redacted := make([]string, len(values))
	for index, value := range values {
		if replacement, ok := replacements[value]; ok {
			redacted[index] = replacement
		} else {
			redacted[index] = digestValue(value)
		}
	}
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

func (c *FileCorpus) Save(ctx context.Context, experiment protocolexperiment.Experiment, result execution.Result) (string, error) {
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
	encoded, err := EncodeBundle(experiment, result, experiment.Retention.MaxArtifactBytes)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(digest, "sha256:") + ".json"
	path := filepath.Join(c.root, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect corpus entry: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := artifactio.Publish(path, encoded); err != nil {
		return "", err
	}
	return path, nil
}
