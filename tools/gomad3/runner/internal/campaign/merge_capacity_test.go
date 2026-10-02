package campaign

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
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

func TestOpenMergedCampaignRejectsNoncanonicalAndInvalidRecords(t *testing.T) {
	canonical, err := canonicaljson.CanonicalJSON(MergedCampaignRecord{Schema: MergedCampaignSchema})
	if err != nil {
		t.Fatal(err)
	}
	invalidIdentity, err := canonicaljson.CanonicalJSON(MergedCampaignRecord{Schema: "gomad3.merged-campaign/v0"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		contents string
		want     string
	}{
		{name: "incomplete object", contents: `{"schema":"gomad3.merged-campaign/v1"}`, want: "JSON is not canonical"},
		{name: "trailing whitespace", contents: string(canonical) + "\n", want: "JSON is not canonical"},
		{name: "trailing data", contents: string(canonical) + "{}", want: "unexpected trailing JSON token {"},
		{name: "unknown field", contents: `{"extra":true}`, want: `decode JSON: json: unknown field "extra"`},
		{name: "malformed", contents: `{"schema":`, want: "decode JSON token: EOF"},
		{name: "invalid schema", contents: string(invalidIdentity), want: "merged campaign record is invalid"},
		{name: "invalid identity", contents: string(canonical), want: "merged campaign record is invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "merged")
			if err := os.MkdirAll(filepath.Join(path, "executions"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(path, "merge.json"), []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := OpenMergedCampaign(path); err == nil || err.Error() != test.want {
				t.Fatalf("OpenMergedCampaign() error = %v, want %s", err, test.want)
			}
		})
	}
}
