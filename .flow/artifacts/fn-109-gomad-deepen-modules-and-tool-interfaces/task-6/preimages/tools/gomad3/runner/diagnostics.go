package runner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

func retainDiagnosticTrace(campaignPath string, ordinal uint64, observed choice.DiagnosticTrace) (_ *DiagnosticTraceReference, retErr error) {
	trace, err := choice.DecodeDiagnosticTrace(observed.Bytes)
	if err != nil {
		return nil, err
	}
	directory := filepath.Join(campaignPath, "diagnostics")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	if err := syncDirectory(campaignPath); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(directory, ".diagnostic-*.partial")
	if err != nil {
		return nil, err
	}
	temporary := file.Name()
	defer func() {
		retErr = errors.Join(retErr, file.Close())
		if err := os.Remove(temporary); err != nil && !os.IsNotExist(err) {
			retErr = errors.Join(retErr, err)
		}
		retErr = errors.Join(retErr, syncDirectory(directory))
	}()
	if _, err := file.Write(trace.Bytes); err != nil {
		return nil, err
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	path := filepath.Join(directory, fmt.Sprintf("%020d-%x.bin", ordinal, trace.SHA256))
	if err := os.Link(temporary, path); err != nil {
		if !os.IsExist(err) {
			return nil, err
		}
		existing, readErr := choice.ReadDiagnosticTrace(path)
		if readErr != nil || existing.SHA256 != trace.SHA256 {
			return nil, errors.Join(errors.New("retained diagnostic trace identity changed"), readErr)
		}
	}
	return &DiagnosticTraceReference{Path: path, DiagnosticEvidence: DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: record.Uint64String(len(trace.Records))}}, nil
}
