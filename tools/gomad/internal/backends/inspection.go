package backends

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
)

const (
	InspectionStatusInspectedPrefix = "inspected_prefix"
	InspectionStatusSearch          = "invariant_search"
	DefaultInspectionBytes          = 8 << 20
	DefaultInspectionDumpBytes      = 4 << 20
	MaximumInspectionBytes          = 64 << 20
)

type InspectionProvider interface {
	Inspect(context.Context, *artifact.Opened, InspectionRequest) (InspectionResult, error)
}

type InspectionPreflightError struct{ Err error }

func (err *InspectionPreflightError) Error() string {
	return "inspection preflight: " + err.Err.Error()
}
func (err *InspectionPreflightError) Unwrap() error { return err.Err }

type InspectionStop struct {
	Kind  string `json:"kind"`
	Value uint64 `json:"value"`
}

type InspectionSearchRequest struct {
	Check         string        `json:"check"`
	First         uint64        `json:"first"`
	Last          uint64        `json:"last"`
	Mode          string        `json:"mode"`
	AttemptBudget uint64        `json:"attempt_budget"`
	WallTime      time.Duration `json:"wall_time"`
}

type InspectionRequest struct {
	Instance        string                   `json:"instance,omitempty"`
	Incarnation     uint64                   `json:"incarnation,omitempty"`
	Stop            InspectionStop           `json:"stop"`
	Dumps           []string                 `json:"dumps"`
	Probes          []string                 `json:"probes"`
	MaximumBytes    uint64                   `json:"maximum_bytes"`
	PerDumpBytes    uint64                   `json:"per_dump_bytes"`
	OutputDirectory string                   `json:"output_directory"`
	Search          *InspectionSearchRequest `json:"search,omitempty"`
}

func (request InspectionRequest) Validate() error {
	if (request.Instance == "") != (request.Incarnation == 0) {
		return errors.New("inspection target requires both instance and positive incarnation")
	}
	if request.MaximumBytes == 0 || request.MaximumBytes > MaximumInspectionBytes || request.PerDumpBytes == 0 || request.PerDumpBytes > request.MaximumBytes {
		return errors.New("inspection requires positive debug byte bounds, with per-dump <= total <= 64MiB")
	}
	if request.OutputDirectory != "" && (!filepath.IsAbs(request.OutputDirectory) || filepath.Clean(request.OutputDirectory) != request.OutputDirectory || request.OutputDirectory == string(filepath.Separator)) {
		return errors.New("inspection output directory must be an absolute clean non-root path")
	}
	if len(request.Dumps) > 256 || len(request.Probes) > 256 {
		return errors.New("inspection dump or probe selection exceeds registration capacity")
	}
	for _, names := range [][]string{request.Dumps, request.Probes} {
		for _, name := range names {
			if name == "" || len(name) > 128 {
				return errors.New("inspection names and patterns must contain 1 to 128 bytes")
			}
		}
	}
	if request.Search != nil {
		search := request.Search
		if request.Stop.Kind != "" || len(request.Dumps) != 0 || len(request.Probes) != 0 {
			return errors.New("invariant search cannot combine with a stop, dump or probe selection")
		}
		if search.Check == "" || len(search.Check) > 128 || search.First > search.Last || search.AttemptBudget == 0 || search.WallTime <= 0 || search.Mode != "scan" && search.Mode != "bisect" {
			return errors.New("invariant search requires a check, inclusive range, scan or bisect mode and positive bounds")
		}
		return nil
	}
	if request.Stop.Kind != "choice" && request.Stop.Kind != "time" && request.Stop.Kind != "failure" || request.Stop.Kind == "failure" && request.Stop.Value != 0 {
		return errors.New("inspection stop must name choice=N, time=T or failure")
	}
	return nil
}

type InspectionBoundary struct {
	Instance    string `json:"instance"`
	Incarnation uint64 `json:"incarnation"`
	Choice      uint64 `json:"choice"`
	Trace       uint64 `json:"trace"`
	Order       uint64 `json:"order"`
	TimeNanos   uint64 `json:"time_nanos"`
}

type InspectionPrefixCut struct {
	Instance      string `json:"instance"`
	Incarnation   uint64 `json:"incarnation"`
	ChoiceRecords uint64 `json:"choice_records"`
	ChoiceBytes   uint64 `json:"choice_bytes"`
	WASIBytes     uint64 `json:"wasi_bytes"`
	NetworkBytes  uint64 `json:"network_bytes"`
	StorageBytes  uint64 `json:"storage_bytes"`
	StdoutBytes   uint64 `json:"stdout_bytes"`
	StderrBytes   uint64 `json:"stderr_bytes"`
}

type InspectionSearchEvaluation struct {
	Choice         uint64 `json:"choice"`
	Outcome        string `json:"outcome"`
	PrefixVerified bool   `json:"prefix_verified"`
	Detail         string `json:"detail,omitempty"`
}

type InspectionSearchRange struct {
	First uint64 `json:"first"`
	Last  uint64 `json:"last"`
}

type InspectionSearchResult struct {
	Status                    string                       `json:"status"`
	FirstFailure              *uint64                      `json:"first_failure,omitempty"`
	Evaluations               []InspectionSearchEvaluation `json:"evaluations"`
	Remaining                 *InspectionSearchRange       `json:"remaining,omitempty"`
	ConditionalOnMonotonicity bool                         `json:"conditional_on_monotonicity"`
}

type InspectionResult struct {
	Status         string                  `json:"status"`
	ArtifactID     string                  `json:"artifact_id"`
	Requested      InspectionRequest       `json:"requested"`
	Resolved       InspectionBoundary      `json:"resolved"`
	PrefixVerified bool                    `json:"prefix_verified"`
	WitnessMatch   bool                    `json:"witness_match"`
	PrefixCuts     []InspectionPrefixCut   `json:"prefix_cuts"`
	OutputPath     string                  `json:"output_path"`
	Truncated      bool                    `json:"truncated"`
	Search         *InspectionSearchResult `json:"search,omitempty"`
}
