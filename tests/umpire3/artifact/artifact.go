package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

type Record struct {
	FormatVersion string                `json:"formatVersion"`
	Experiment    protocol.Experiment   `json:"experiment"`
	Result        umpire3runtime.Result `json:"result"`
}

func Encode(experiment protocol.Experiment, result umpire3runtime.Result, maxBytes int64) ([]byte, error) {
	if err := experiment.Validate(); err != nil {
		return nil, fmt.Errorf("validate artifact experiment: %w", err)
	}
	if maxBytes <= 0 || maxBytes > experiment.Retention.MaxArtifactBytes {
		maxBytes = experiment.Retention.MaxArtifactBytes
	}
	redacted := redactResult(result)
	encoded, err := json.Marshal(Record{
		FormatVersion: protocol.FormatVersion,
		Experiment:    experiment,
		Result:        redacted,
	})
	if err != nil {
		return nil, fmt.Errorf("encode artifact: %w", err)
	}
	if int64(len(encoded)) > maxBytes {
		return nil, fmt.Errorf("artifact size %d exceeds %d-byte limit", len(encoded), maxBytes)
	}
	return encoded, nil
}

func redactResult(result umpire3runtime.Result) umpire3runtime.Result {
	redacted := result
	redacted.Bindings = redactMap(result.Bindings)
	redacted.Actions = append([]umpire3runtime.ActionResult(nil), result.Actions...)
	for index := range redacted.Actions {
		redacted.Actions[index].Evidence.Reference = digestValue(redacted.Actions[index].Evidence.Reference)
		redacted.Actions[index].Evidence.GroundedBindings = redactMap(redacted.Actions[index].Evidence.GroundedBindings)
	}
	redacted.Observations = append([]environment.Observation(nil), result.Observations...)
	for index := range redacted.Observations {
		redacted.Observations[index].CausalReference = digestValue(redacted.Observations[index].CausalReference)
	}
	redacted.Cleanup.RecoverableResources = redactMap(result.Cleanup.RecoverableResources)
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
