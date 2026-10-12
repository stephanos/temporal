package campaign

import (
	"errors"
	"fmt"
	"os"
)

type journalCapacityOutcome string

type journalLimit string

type artifactLimit string

const (
	CapacityInfrastructureFailure journalCapacityOutcome = "infrastructure_failure"
)

const (
	ArtifactLimitFailureCount artifactLimit = "failure_artifacts"
	ArtifactLimitFailureBytes artifactLimit = "failure_bytes"
	artifactLimitSuccessCount artifactLimit = "success_artifacts"
	artifactLimitSuccessBytes artifactLimit = "success_bytes"
	ArtifactLimitTotalBytes   artifactLimit = "total_bytes"
)

const (
	journalLimitExecutions        journalLimit = "executions"
	journalLimitBytes             journalLimit = "journal_bytes"
	journalLimitSegmentBytes      journalLimit = "segment_bytes"
	journalLimitSegments          journalLimit = "segments"
	journalLimitIndexBytes        journalLimit = "index_bytes"
	journalLimitPartialExecutions journalLimit = "partial_executions"
	journalLimitManifestBytes     journalLimit = "manifest_bytes"
)

type JournalCapacityError struct {
	Limit    journalLimit
	Required uint64
	Maximum  uint64
	Outcome  journalCapacityOutcome
}

func (err *JournalCapacityError) Error() string {
	return fmt.Sprintf("execution journal %s requires %d, exceeding %d (%s)", err.Limit, err.Required, err.Maximum, err.Outcome)
}

type ArtifactCapacityError struct {
	Limit    artifactLimit
	Required uint64
	Maximum  uint64
	Outcome  journalCapacityOutcome
}

func (err *ArtifactCapacityError) Error() string {
	return fmt.Sprintf("artifact %s requires %d, exceeding %d (%s)", err.Limit, err.Required, err.Maximum, err.Outcome)
}

type integrityError struct {
	err error
}

func (err *integrityError) Error() string {
	return err.err.Error()
}

func (err *integrityError) Unwrap() error {
	return err.err
}

func IsIntegrityError(err error) bool {
	var integrityErr *integrityError
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
	return &integrityError{err: err}
}

func newIntegrityError(err error) error {
	if err == nil || IsIntegrityError(err) {
		return err
	}
	return &integrityError{err: err}
}
