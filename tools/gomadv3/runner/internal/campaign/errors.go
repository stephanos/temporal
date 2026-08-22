package campaign

import (
	"errors"
	"fmt"
	"os"
)

type JournalCapacityOutcome string

type JournalLimit string

type ArtifactLimit string

const (
	CapacityInfrastructureFailure JournalCapacityOutcome = "infrastructure_failure"
)

const (
	ArtifactLimitFailureCount ArtifactLimit = "failure_artifacts"
	ArtifactLimitFailureBytes ArtifactLimit = "failure_bytes"
	ArtifactLimitSuccessCount ArtifactLimit = "success_artifacts"
	ArtifactLimitSuccessBytes ArtifactLimit = "success_bytes"
	ArtifactLimitTotalBytes   ArtifactLimit = "total_bytes"
)

const (
	JournalLimitExecutions        JournalLimit = "executions"
	JournalLimitBytes             JournalLimit = "journal_bytes"
	JournalLimitSegmentBytes      JournalLimit = "segment_bytes"
	JournalLimitSegments          JournalLimit = "segments"
	JournalLimitIndexBytes        JournalLimit = "index_bytes"
	JournalLimitPartialExecutions JournalLimit = "partial_executions"
	JournalLimitManifestBytes     JournalLimit = "manifest_bytes"
)

type JournalCapacityError struct {
	Limit    JournalLimit
	Required uint64
	Maximum  uint64
	Outcome  JournalCapacityOutcome
}

func (err *JournalCapacityError) Error() string {
	return fmt.Sprintf("execution journal %s requires %d, exceeding %d (%s)", err.Limit, err.Required, err.Maximum, err.Outcome)
}

type ArtifactCapacityError struct {
	Limit    ArtifactLimit
	Required uint64
	Maximum  uint64
	Outcome  JournalCapacityOutcome
}

func (err *ArtifactCapacityError) Error() string {
	return fmt.Sprintf("artifact %s requires %d, exceeding %d (%s)", err.Limit, err.Required, err.Maximum, err.Outcome)
}

type IntegrityError struct {
	err error
}

func (err *IntegrityError) Error() string {
	return err.err.Error()
}

func (err *IntegrityError) Unwrap() error {
	return err.err
}

func IsIntegrityError(err error) bool {
	var integrityErr *IntegrityError
	return errors.As(err, &integrityErr)
}

func classifyIntegrityError(err error) error {
	if err == nil || IsIntegrityError(err) {
		return err
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return &IntegrityError{err: err}
}

func newIntegrityError(err error) error {
	if err == nil || IsIntegrityError(err) {
		return err
	}
	return &IntegrityError{err: err}
}
