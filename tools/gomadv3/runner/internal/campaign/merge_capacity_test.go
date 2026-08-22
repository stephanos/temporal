package campaign

import (
	"errors"
	"testing"

	"go.temporal.io/server/tools/gomadv3/record"
)

func TestValidateMergedArtifactCapacityReportsSaturatedTotalOverflow(t *testing.T) {
	maximum := record.Uint64String(^uint64(0))
	err := validateMergedArtifactCapacity(ArtifactCapacityPlan{
		FailureArtifacts: 1,
		FailureBytes:     maximum,
		SuccessArtifacts: 1,
		SuccessBytes:     maximum,
		TotalBytes:       maximum,
	}, 1, ^uint64(0), 1, 1)
	var capacityErr *ArtifactCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != ArtifactLimitTotalBytes || capacityErr.Required != ^uint64(0) || capacityErr.Outcome != CapacityInfrastructureFailure {
		t.Fatalf("validateMergedArtifactCapacity() error = %#v", err)
	}
}

func TestCheckedMergedEvidenceBytesReportsTypedOverflow(t *testing.T) {
	_, err := checkedMergedEvidenceBytes(^uint64(0), 1, ArtifactLimitFailureBytes, 100)
	var capacityErr *ArtifactCapacityError
	if !errors.As(err, &capacityErr) || capacityErr.Limit != ArtifactLimitFailureBytes || capacityErr.Required != ^uint64(0) || capacityErr.Maximum != 100 {
		t.Fatalf("checkedMergedEvidenceBytes() error = %#v", err)
	}
}
