package backend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

var _ backends.InspectionProvider = (*Provider)(nil)

func (p *Provider) Inspect(ctx context.Context, opened *artifact.Opened, request backends.InspectionRequest) (report backends.InspectionResult, retErr error) {
	preflight := func(err error) (backends.InspectionResult, error) {
		return report, &backends.InspectionPreflightError{Err: err}
	}
	if err := request.Validate(); err != nil {
		return preflight(err)
	}
	if opened == nil {
		return preflight(errors.New("inspection requires an opened artifact"))
	}
	prepared, err := p.ValidateReplay(ctx, opened)
	if err != nil {
		return preflight(err)
	}
	pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
	if err != nil {
		return preflight(err)
	}
	if pr.Profile != wasi.CooperativeProfile {
		return preflight(errors.New("inspection requires cooperative exact replay"))
	}
	manifest := opened.Manifest()
	reference := *manifest.Target.Backend.Evidence
	data, err := opened.ReadPayload(reference.File, uint64(reference.Bytes))
	if err != nil {
		return preflight(err)
	}
	payload := target.BackendPayload{Reference: reference, Data: data}
	isCluster := len(pr.Scenario) != 0
	var execution *cluster.WASMExecutionEvidence
	var standalone wasi.Request
	if isCluster {
		_, retained, err := simulation.VerifyRetained(payload)
		if err != nil {
			return preflight(err)
		}
		execution = retained.WASM
	} else {
		execution, standalone, err = p.standaloneInspection(opened, pr, data)
		if err != nil {
			return preflight(err)
		}
	}
	query := cluster.WASMInspectionRequest{Target: cluster.NodeHandle{Node: cluster.NodeID(request.Instance), Incarnation: request.Incarnation}, Stop: cluster.WASMInspectionStop{Kind: request.Stop.Kind, Value: request.Stop.Value}, Dumps: request.Dumps, Probes: request.Probes}
	if request.Search != nil {
		query.Stop = cluster.WASMInspectionStop{Kind: "choice", Value: request.Search.First}
	}
	resolved, err := simulation.ResolveInspection(execution, query, isCluster)
	if err != nil {
		return preflight(err)
	}
	if request.OutputDirectory != "" {
		artifactPath, err := filepath.EvalSymlinks(opened.Path())
		if err != nil {
			return preflight(err)
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(request.OutputDirectory))
		if err != nil {
			return preflight(err)
		}
		output := filepath.Join(parent, filepath.Base(request.OutputDirectory))
		relative, err := filepath.Rel(artifactPath, output)
		if err != nil || relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return preflight(errors.New("inspection output must be outside artifact"))
		}
	}
	report = backends.InspectionResult{ArtifactID: string(manifest.RecordHash), Requested: request}
	report.Resolved = backends.InspectionBoundary{Instance: string(resolved.Target.Node), Incarnation: resolved.Target.Incarnation, Choice: resolved.Boundary.Decision, Trace: resolved.Boundary.Trace, Order: resolved.Boundary.Order, TimeNanos: resolved.Boundary.TimeNanos}
	module := simulation.RetainedModule{SHA256: string(manifest.Target.SHA256), Copy: func(destination string) error { return opened.CopyPayload(manifest.Target.File, destination, 0500) }}
	run := func(ctx context.Context, boundary cluster.WASMInspectionResolution, check string) (wasi.DebugOutput, error) {
		private, err := os.MkdirTemp("", "gomad-inspection-")
		if err != nil {
			return wasi.DebugOutput{}, err
		}
		var observed wasi.Result
		if isCluster {
			observed, err = simulation.InspectRetained(ctx, payload, module, private, simulation.InstalledHelper{Path: p.options.HelperPath, SHA256: strings.TrimPrefix(pr.HelperSHA256, "sha256:"), Engine: pr.Engine, ConfigureRequest: p.stressRequest}, boundary, check)
		} else {
			var control []byte
			control, err = simulation.InspectionControl(execution.Inspection, boundary, check)
			if err == nil {
				standalone.Runtime.Inspection = control
				standalone.ModulePath = filepath.Join(private, "module.wasm")
				err = module.Copy(standalone.ModulePath)
			}
			if err == nil {
				observed, err = wasi.Run(ctx, standalone)
			}
			if err == nil && observed.Termination == backends.InspectionStatusInspectedPrefix {
				var metadata *cluster.WASMInspectionEvidence
				metadata, err = cluster.DecodeWASMInspection(observed.InspectionMetadata)
				if err == nil {
					err = cluster.VerifyWASMInspectionPrefix(execution.Inspection, metadata, boundary.Boundary)
				}
			}
		}
		err = errors.Join(err, os.RemoveAll(private))
		if err != nil {
			return wasi.DebugOutput{}, err
		}
		if observed.Cancelled || observed.WatchdogTimeout {
			return wasi.DebugOutput{}, errors.New("inspection evaluation cancelled or timed out")
		}
		if !observed.Reaped {
			return wasi.DebugOutput{}, errors.New("inspection helper not reaped")
		}
		if observed.Termination != backends.InspectionStatusInspectedPrefix {
			if observed.Termination == "debug_error" || observed.Termination == "probe_fuel_limit" || observed.Termination == "fuel" || observed.Termination == "capacity" {
				return wasi.DebugOutput{}, fmt.Errorf("inspection evaluation %s: %s", observed.Termination, observed.Message)
			}
			return wasi.DebugOutput{}, fmt.Errorf("%w: inspection terminal %q: %s", cluster.ErrWASMInstrumentationDivergence, observed.Termination, observed.Message)
		}
		output, err := wasi.DecodeDebugOutput(observed.DebugOutput)
		if err != nil {
			return output, err
		}
		output.SetTarget(string(boundary.Target.Node), boundary.Target.Incarnation)
		for _, incarnation := range execution.Inspection.Incarnations {
			if incarnation.Handle != boundary.Target {
				continue
			}
			for _, entry := range incarnation.Registry.Entries {
				for i := range output.Dumps {
					if output.Dumps[i].ID == entry.ID && entry.Kind == "dump" {
						output.Dumps[i].Name = entry.Name
					}
				}
				for i := range output.Probes {
					if output.Probes[i].ID == entry.ID && entry.Kind == "probe" {
						output.Probes[i].Name = entry.Name
					}
				}
				for i := range output.Checks {
					if output.Checks[i].ID == entry.ID && entry.Kind == "check" {
						output.Checks[i].Name = entry.Name
					}
				}
			}
		}
		for i := range output.Dumps {
			if uint64(len(output.Dumps[i].Data)) > request.PerDumpBytes {
				output.Dumps[i].Data = output.Dumps[i].Data[:request.PerDumpBytes]
				output.Dumps[i].Status = "truncated"
			}
		}
		return output, nil
	}
	if search := request.Search; search != nil {
		result, err := simulation.SearchWASMInspection(ctx, execution, simulation.WASMInspectionSearchRequest{Target: resolved.Target, Cluster: isCluster, Check: search.Check, First: search.First, Last: search.Last, Mode: search.Mode, AttemptBudget: search.AttemptBudget, WallTime: search.WallTime}, func(ctx context.Context, resolution cluster.WASMInspectionResolution, check string) (simulation.WASMInspectionEvaluation, error) {
			output, err := run(ctx, resolution, check)
			if err != nil {
				return simulation.WASMInspectionEvaluation{}, err
			}
			if len(output.Checks) != 1 || output.Checks[0].Name != check {
				return simulation.WASMInspectionEvaluation{}, errors.New("inspection did not return exactly one invariant result")
			}
			status := output.Checks[0].Status
			outcome := "error"
			if status == "true" {
				outcome = "pass"
			}
			if status == "false" {
				outcome = "fail"
			}
			return simulation.WASMInspectionEvaluation{Outcome: outcome, PrefixVerified: true, Detail: status}, nil
		})
		if err != nil {
			return preflight(err)
		}
		report.Status = backends.InspectionStatusSearch
		report.Search = &backends.InspectionSearchResult{Status: result.Status, FirstFailure: result.FirstFailure, ConditionalOnMonotonicity: result.ConditionalOnMonotonicity}
		for _, e := range result.Evaluations {
			report.Search.Evaluations = append(report.Search.Evaluations, backends.InspectionSearchEvaluation{Choice: e.Choice, Outcome: e.Outcome, PrefixVerified: e.PrefixVerified, Detail: e.Detail})
		}
		if result.Remaining != nil {
			report.Search.Remaining = &backends.InspectionSearchRange{First: result.Remaining.First, Last: result.Remaining.Last}
		}
		return report, nil
	}
	output, err := run(ctx, resolved, "")
	if err != nil {
		return report, err
	}
	report.Status, report.PrefixVerified, report.WitnessMatch = backends.InspectionStatusInspectedPrefix, true, true
	encoded, heap, truncated, err := boundInspectionOutput(output, request.MaximumBytes)
	if err != nil {
		return report, err
	}
	b := resolved.Boundary
	report.Resolved = backends.InspectionBoundary{Instance: string(resolved.Target.Node), Incarnation: resolved.Target.Incarnation, Choice: b.Decision, Trace: b.Trace, Order: b.Order, TimeNanos: b.TimeNanos}
	for _, cut := range b.Cuts {
		report.PrefixCuts = append(report.PrefixCuts, backends.InspectionPrefixCut{Instance: string(cut.Handle.Node), Incarnation: cut.Handle.Incarnation, ChoiceRecords: cut.ChoiceRecords, ChoiceBytes: cut.ChoiceBytes, WASIBytes: cut.WASIBytes, NetworkBytes: cut.NetworkBytes, StorageBytes: cut.StorageBytes, StdoutBytes: cut.StdoutBytes, StderrBytes: cut.StderrBytes})
	}
	report.Truncated = truncated || output.StacksTruncated || output.HeapTruncated || len(output.Drops) != 0
	for _, dump := range output.Dumps {
		report.Truncated = report.Truncated || dump.Status == "truncated"
	}
	outputDirectory := request.OutputDirectory
	if outputDirectory == "" {
		outputDirectory, err = os.MkdirTemp(filepath.Dir(opened.Path()), "inspection-")
		if err != nil {
			return report, err
		}
	} else {
		if err := os.Mkdir(outputDirectory, 0700); err != nil {
			return report, err
		}
	}
	{
		report.OutputPath = filepath.Join(outputDirectory, "debug.json")
		if err := os.WriteFile(report.OutputPath, encoded, 0600); err != nil {
			return report, err
		}
		if len(heap) != 0 {
			if err := os.WriteFile(filepath.Join(outputDirectory, "heap.pprof"), heap, 0600); err != nil {
				return report, err
			}
		}
	}
	return report, nil
}

func boundInspectionOutput(output wasi.DebugOutput, maximum uint64) ([]byte, []byte, bool, error) {
	heap := output.HeapProfile
	output.HeapProfile = nil
	truncated := output.StacksTruncated || output.HeapTruncated || len(output.Drops) != 0
	for _, dump := range output.Dumps {
		truncated = truncated || dump.Status == "truncated"
	}
	for {
		encoded, err := json.Marshal(struct {
			wasi.DebugOutput
			Truncated bool `json:"truncated"`
		}{output, truncated})
		if err != nil {
			return nil, nil, false, err
		}
		if uint64(len(encoded)+len(heap)) <= maximum {
			return encoded, heap, truncated, nil
		}
		truncated = true
		switch {
		case len(heap) != 0:
			heap = nil
			output.HeapTruncated = true
		case len(output.Stacks) != 0:
			output.Stacks = output.Stacks[:len(output.Stacks)-1]
			output.StacksTruncated = true
		case len(output.Probes) != 0:
			output.Probes = output.Probes[:len(output.Probes)-1]
		case len(output.Dumps) != 0:
			i := len(output.Dumps) - 1
			if len(output.Dumps[i].Data) != 0 {
				output.Dumps[i].Data = output.Dumps[i].Data[:len(output.Dumps[i].Data)/2]
				output.Dumps[i].Status = "truncated"
			} else {
				output.Dumps = output.Dumps[:i]
			}
		case len(output.Drops) != 0:
			output.Drops = output.Drops[:len(output.Drops)-1]
		default:
			return nil, nil, true, errors.New("inspection byte bound is too small for truncation metadata")
		}
	}
}

func (p *Provider) standaloneInspection(opened *artifact.Opened, pr provenance, evidence []byte) (*cluster.WASMExecutionEvidence, wasi.Request, error) {
	var envelope struct {
		Schema                 string `json:"schema"`
		WASI, Network, Storage []byte
		StorageSnapshots       []volume.VolumeSnapshot
		StorageImageIdentity   string `json:"storage_image_identity,omitempty"`
		Inspection             []byte `json:"inspection,omitempty"`
	}
	if err := strictjson.Decode(evidence, &envelope); err != nil || envelope.Schema != "gomad-wasm.modeled-evidence/v2" {
		return nil, wasi.Request{}, cluster.ErrWASMInspectionUnsupported
	}
	metadata, err := cluster.DecodeWASMInspection(envelope.Inspection)
	if err != nil {
		return nil, wasi.Request{}, err
	}
	manifest := opened.Manifest()
	profile := manifest.ChoiceProfile
	if profile == nil || len(metadata.Incarnations) != 1 {
		return nil, wasi.Request{}, cluster.ErrWASMInspectionUnsupported
	}
	traceBytes, err := opened.ReadPayload(profile.Trace.File, uint64(profile.Trace.Limit))
	if err != nil {
		return nil, wasi.Request{}, err
	}
	digest, err := profile.Trace.SHA256.Bytes()
	if err != nil {
		return nil, wasi.Request{}, err
	}
	if sha256.Sum256(traceBytes) != digest {
		return nil, wasi.Request{}, errors.New("inspection retained trace identity")
	}
	implementation, err := choice.ImplementationIdentity(pr.BuildKey)
	if err != nil {
		return nil, wasi.Request{}, err
	}
	targetDigest, err := manifest.Target.SHA256.Bytes()
	if err != nil {
		return nil, wasi.Request{}, err
	}
	identity := choice.ExecutionIdentity{TargetSHA256: targetDigest, ToolchainBuildKey: pr.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}
	tapeDigest, err := profile.Trace.TapeSHA256.Bytes()
	if err != nil {
		return nil, wasi.Request{}, err
	}
	n := metadata.Incarnations[0]
	for _, stream := range []struct {
		recorded record.Stream
		data     []byte
	}{{manifest.Streams.Stdout, n.Streams.Stdout}, {manifest.Streams.Stderr, n.Streams.Stderr}} {
		if stream.recorded.Truncated || uint64(stream.recorded.TotalBytes) != uint64(len(stream.data)) || record.HashBytes(stream.data) != stream.recorded.FullSHA256 {
			return nil, wasi.Request{}, errors.New("inspection canonical output differs")
		}
	}
	if !bytes.Equal(n.Streams.Choice, traceBytes) || !bytes.Equal(n.Streams.WASI, envelope.WASI) || !bytes.Equal(n.Streams.Network, envelope.Network) || !bytes.Equal(n.Streams.Storage, envelope.Storage) {
		return nil, wasi.Request{}, errors.New("inspection canonical evidence differs")
	}
	execution := &cluster.WASMExecutionEvidence{Inspection: metadata, Incarnations: []cluster.WASMIncarnationEvidence{{Handle: n.Handle, ChoiceTrace: traceBytes, WASISHA256: string(record.HashBytes(envelope.WASI)), NetworkSHA256: string(record.HashBytes(envelope.Network)), StorageSHA256: string(record.HashBytes(envelope.Storage))}}}
	config := pr.Config
	config.Args = append([]string{}, manifest.Target.Argv...)
	config.Environment = nil
	for _, entry := range manifest.Environment {
		if entry.Name != "GOMAD_CHOICE_PROFILE" && entry.Name != choice.DiagnosticProfileEnvironment {
			config.Environment = append(config.Environment, entry.Name+"="+entry.Value)
		}
	}
	config.Environment = append(config.Environment, "PWD="+config.WorkingDirectory)
	config.Limits.OutputBytes, config.Limits.TranscriptBytes = uint64(manifest.Limits.OutputBytes), uint64(manifest.Limits.IOTranscriptBytes)
	request := wasi.Request{HelperPath: p.options.HelperPath, HelperSHA256: strings.TrimPrefix(pr.HelperSHA256, "sha256:"), ModuleSHA256: strings.TrimPrefix(pr.ModuleSHA256, "sha256:"), Engine: pr.Engine, Imports: pr.Imports, InitialMemoryPages: pr.InitialMemoryPages, MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: config, Timeout: time.Duration(manifest.Limits.ExecutionTimeoutNanos), Runtime: &wasi.RuntimeControl{Seed: uint64(manifest.Seed), Choice: &backends.ChoiceRequest{Mode: choice.ModeReplay, ExecutionIdentity: identity, Limit: uint64(profile.Trace.Limit), Trace: traceBytes, TraceSHA256: digest, TapeSHA256: tapeDigest, Decisions: uint64(profile.Trace.Decisions)}, Diagnostics: false}}
	if err := p.stressRequest(&request); err != nil {
		return nil, wasi.Request{}, err
	}
	return execution, request, nil
}
