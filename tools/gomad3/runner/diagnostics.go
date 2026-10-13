package runner

import (
	"context"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
)

type DiagnosticEvidence struct {
	Profile string              `json:"profile"`
	SHA256  record.SHA256       `json:"sha256"`
	Records record.Uint64String `json:"records"`
}

type DiagnosticTraceReference struct {
	Path string `json:"path"`
	DiagnosticEvidence
}

// retainDiagnosticTrace durably publishes an attempt's diagnostic trace as a
// sidecar under the campaign's diagnostics directory before the attempt's
// receipt references it.
func retainDiagnosticTrace(ctx context.Context, campaignPath string, ordinal uint64, observed choice.DiagnosticTrace) (*DiagnosticTraceReference, artifact.FileReference, error) {
	trace, err := choice.DecodeDiagnosticTrace(observed.Bytes)
	if err != nil {
		return nil, artifact.FileReference{}, err
	}
	directory := filepath.Join(campaignPath, "diagnostics")
	name := fmt.Sprintf("%020d-%x.bin", ordinal, trace.SHA256)
	file, err := artifact.PublishOwnedFile(ctx, directory, name, trace.Bytes)
	if err != nil {
		return nil, artifact.FileReference{}, err
	}
	reference := &DiagnosticTraceReference{Path: filepath.Join(directory, name), DiagnosticEvidence: DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: record.Uint64String(len(trace.Records))}}
	return reference, file, nil
}
