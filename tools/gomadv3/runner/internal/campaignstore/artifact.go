package campaignstore

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"

	"go.temporal.io/server/tools/gomadv3/choice"
	romount "go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
)

type ArtifactInput struct {
	Manifest       evidence.ExecutionRecord
	TargetPath     string
	Stdout         []byte
	Stderr         []byte
	IOTranscript   []byte
	ChoiceTrace    []byte
	ReadOnlyMounts *romount.CapturedInputs
	World          evidence.WorldPayloads
	Simulation     *SimulationPayloads
}

type SimulationPayloads struct {
	Plan   []byte
	Record []byte
}

func PublishArtifact(store evidence.Store, input ArtifactInput) (evidence.Artifact, error) {
	manifest := input.Manifest
	if manifest.SchemaVersion != evidence.SchemaVersion || manifest.Runner.RecordContract != evidence.RecordContract {
		return evidence.Artifact{}, errors.New("artifact publication requires the current run-record contract")
	}
	if manifest.World.Initial.File != "world/snapshot.json" || manifest.World.Transitions.File != "world/transitions.jsonl" || manifest.World.Final.File != "world/final-snapshot.json" {
		return evidence.Artifact{}, errors.New("World payload paths must use the canonical artifact layout")
	}
	if err := validateChoiceTracePayload(manifest, input.ChoiceTrace); err != nil {
		return evidence.Artifact{}, err
	}
	manifest.Target.File = "target"
	if manifest.Target.CapabilityMode == "" {
		manifest.Target.CapabilityMode = string(target.CapabilityModeClosure)
	}
	manifest.Streams.Stdout.File = "stdout"
	manifest.Streams.Stderr.File = "stderr"
	manifest.Streams.Stdout.RetainedSHA256 = evidence.HashBytes(input.Stdout)
	manifest.Streams.Stderr.RetainedSHA256 = evidence.HashBytes(input.Stderr)
	payloads := []evidence.Payload{
		artifactSourcePayload(manifest.Target.File, input.TargetPath, 0o700, manifest.Target.SHA256, manifest.Target.Size),
		artifactDataPayload(manifest.Streams.Stdout.File, input.Stdout, 0o600),
		artifactDataPayload(manifest.Streams.Stderr.File, input.Stderr, 0o600),
	}
	if manifest.Target.CapabilityMode == string(target.CapabilityModeLinked) {
		if manifest.Target.CapabilityManifest == nil {
			return evidence.Artifact{}, errors.New("linked target capability manifest record is required")
		}
		capabilities, err := target.ReadCapabilityManifest(input.TargetPath, target.ToolchainIdentity{
			GoVersion: manifest.Toolchain.GoVersion, BuildKey: manifest.Toolchain.BuildKey,
			TargetGOOS: manifest.Toolchain.TargetGOOS, TargetGOARCH: manifest.Toolchain.TargetGOARCH,
		})
		if err != nil {
			return evidence.Artifact{}, fmt.Errorf("extract artifact target capability manifest: %w", err)
		}
		if actual := capabilities.Record(); *actual != *manifest.Target.CapabilityManifest {
			return evidence.Artifact{}, errors.New("artifact target capability manifest identity changed during publication")
		}
		payloads = append(payloads, evidence.Payload{
			Path: manifest.Target.CapabilityManifest.File, Mode: 0o600, Data: capabilities.Payload,
			SHA256: manifest.Target.CapabilityManifest.SHA256, Size: manifest.Target.CapabilityManifest.Bytes,
		})
	} else if manifest.Target.CapabilityMode != string(target.CapabilityModeClosure) || manifest.Target.CapabilityManifest != nil {
		return evidence.Artifact{}, errors.New("artifact target capability mode is invalid")
	}
	if manifest.IOProfile.Transcript != nil {
		manifest.IOProfile.Transcript.File = "io/transcript.bin"
		transcript := manifest.IOProfile.Transcript
		payloads = append(payloads, evidence.Payload{Path: transcript.File, Mode: 0o600, Data: input.IOTranscript, SHA256: transcript.SHA256, Size: transcript.Bytes})
	}
	if manifest.ChoiceProfile != nil {
		manifest.ChoiceProfile.Trace.File = "choices.bin"
		trace := manifest.ChoiceProfile.Trace
		payloads = append(payloads, evidence.Payload{Path: trace.File, Mode: 0o600, Data: input.ChoiceTrace, SHA256: trace.SHA256, Size: trace.Bytes})
	} else if input.ChoiceTrace != nil {
		return evidence.Artifact{}, errors.New("unexpected choice trace payload")
	}
	if manifest.SimulationProfile != nil {
		if input.Simulation == nil {
			return evidence.Artifact{}, errors.New("simulation exploration artifact payloads are required")
		}
		profile := manifest.SimulationProfile
		profile.Plan.File = "simulation/plan.json"
		profile.Record.File = "simulation/record.json"
		if evidence.HashBytes(input.Simulation.Plan) != profile.Plan.SHA256 || uint64(len(input.Simulation.Plan)) != uint64(profile.Plan.Bytes) {
			return evidence.Artifact{}, errors.New("simulation plan identity changed during publication")
		}
		if evidence.HashBytes(input.Simulation.Record) != profile.Record.SHA256 || uint64(len(input.Simulation.Record)) != uint64(profile.Record.Bytes) {
			return evidence.Artifact{}, errors.New("simulation record identity changed during publication")
		}
		payloads = append(payloads,
			artifactDataPayload(profile.Plan.File, input.Simulation.Plan, 0o600),
			artifactDataPayload(profile.Record.File, input.Simulation.Record, 0o600),
		)
	} else if input.Simulation != nil {
		return evidence.Artifact{}, errors.New("unexpected simulation exploration artifact payloads")
	}
	if manifest.IOProfile.ReadOnlyMounts != nil {
		if input.ReadOnlyMounts == nil {
			return evidence.Artifact{}, errors.New("read-only mount artifact payload is required")
		}
		if !capturedInputsMatch(manifest.IOProfile.ReadOnlyMounts, input.ReadOnlyMounts.Manifest) {
			return evidence.Artifact{}, errors.New("read-only mount artifact identity changed during publication")
		}
		mounts := manifest.IOProfile.ReadOnlyMounts
		payloads = append(payloads, evidence.Payload{Path: mounts.File, Mode: 0o600, Data: input.ReadOnlyMounts.Descriptor, SHA256: mounts.SHA256, Size: mounts.Bytes})
		paths := make([]string, 0, len(input.ReadOnlyMounts.Payloads))
		for payloadPath := range input.ReadOnlyMounts.Payloads {
			paths = append(paths, payloadPath)
		}
		sort.Strings(paths)
		for _, payloadPath := range paths {
			payloads = append(payloads, artifactDataPayload(payloadPath, input.ReadOnlyMounts.Payloads[payloadPath], 0o600))
		}
	} else if input.ReadOnlyMounts != nil {
		return evidence.Artifact{}, errors.New("unexpected read-only mount artifact payload")
	}
	worldPayloads := []struct {
		path string
		data []byte
		hash *evidence.SHA256
	}{
		{path: manifest.World.Initial.File, data: input.World.Initial, hash: &manifest.World.Initial.RawSHA256},
		{path: manifest.World.Transitions.File, data: input.World.Transitions, hash: &manifest.World.Transitions.RawSHA256},
		{path: manifest.World.Final.File, data: input.World.Final, hash: &manifest.World.Final.RawSHA256},
	}
	for _, payload := range worldPayloads {
		*payload.hash = evidence.HashBytes(payload.data)
		payloads = append(payloads, artifactDataPayload(payload.path, payload.data, 0o600))
	}
	return store.PublishArtifact(evidence.Publication{Record: manifest, Payloads: payloads})
}

func capturedInputsMatch(recorded *evidence.ReadOnlyMounts, captured romount.CapturedInputsManifest) bool {
	limits := recorded.Limits
	return recorded.Schema == captured.Schema && recorded.File == captured.File && recorded.SHA256 == evidence.SHA256(captured.SHA256) && uint64(recorded.Bytes) == captured.Bytes &&
		uint64(recorded.Entries) == captured.Entries && uint64(recorded.NotExist) == captured.NotExist && uint64(recorded.TotalBytes) == captured.TotalBytes && slices.Equal(recorded.Mappings, captured.Mappings) &&
		uint64(limits.PathBytes) == captured.Limits.PathBytes && uint64(limits.Requests) == captured.Limits.Requests && uint64(limits.Files) == captured.Limits.Files &&
		uint64(limits.DirectoryEntries) == captured.Limits.DirectoryEntries && uint64(limits.SingleFileBytes) == captured.Limits.SingleFileBytes && uint64(limits.TotalBytes) == captured.Limits.TotalBytes
}

func artifactSourcePayload(path, source string, mode os.FileMode, digest evidence.SHA256, size evidence.Uint64String) evidence.Payload {
	return evidence.Payload{Path: path, Mode: mode, SourcePath: source, SHA256: digest, Size: size}
}

func artifactDataPayload(path string, data []byte, mode os.FileMode) evidence.Payload {
	return evidence.Payload{Path: path, Mode: mode, Data: data, SHA256: evidence.HashBytes(data), Size: evidence.Uint64String(len(data))}
}

func validateChoiceTracePayload(manifest evidence.ExecutionRecord, payload []byte) error {
	if manifest.ChoiceProfile == nil {
		if payload != nil {
			return errors.New("unexpected choice trace payload")
		}
		return nil
	}
	trace := manifest.ChoiceProfile.Trace
	var implementation [32]byte
	if manifest.SchemaVersion == evidence.SchemaVersion {
		var err error
		implementation, err = choice.ImplementationIdentity(manifest.Toolchain.BuildKey)
		if err != nil {
			return fmt.Errorf("derive choice trace implementation identity: %w", err)
		}
		if manifest.ChoiceProfile.ImplementationSHA256 != evidence.SHA256FromSum(implementation) {
			return errors.New("choice trace implementation identity does not match the pinned toolchain")
		}
	}
	if evidence.HashBytes(payload) != trace.SHA256 || uint64(len(payload)) != uint64(trace.Bytes) {
		return errors.New("choice trace identity changed during publication")
	}
	digest, err := trace.SHA256.Bytes()
	if err != nil {
		return fmt.Errorf("decode choice trace identity: %w", err)
	}
	targetIdentity, err := manifest.Target.SHA256.Bytes()
	if err != nil {
		return fmt.Errorf("decode choice trace target identity: %w", err)
	}
	terminalState := choice.TerminalComplete
	if trace.TerminalState == "overflow" {
		terminalState = choice.TerminalOverflow
	}
	decoded, err := choice.DecodeStoredTrace(manifest.ChoiceProfile.Name, payload, choice.TerminalMetadata{
		State: terminalState, Limit: uint64(trace.Limit), Records: uint64(trace.Records), SHA256: digest,
	})
	if errors.Is(err, choice.ErrOverflow) && terminalState == choice.TerminalOverflow {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("validate choice trace payload: %w", err)
	}
	projection, err := choice.ProjectTrace(decoded, uint64(trace.Limit), targetIdentity)
	if err != nil {
		return fmt.Errorf("project choice trace payload: %w", err)
	}
	if projection.Summary.Branching != uint64(trace.BranchingRecords) {
		return errors.New("choice trace branching count does not match its payload")
	}
	if manifest.SchemaVersion == evidence.PreviousSchemaVersion || terminalState == choice.TerminalOverflow {
		return nil
	}
	tape, err := choice.ProjectReplayPlan(decoded, choice.ExecutionIdentity{
		TargetSHA256: targetIdentity, ToolchainBuildKey: manifest.Toolchain.BuildKey,
		GOOS: manifest.Toolchain.TargetGOOS, GOARCH: manifest.Toolchain.TargetGOARCH, ImplementationSHA256: implementation,
	})
	if err != nil {
		return fmt.Errorf("derive choice tape: %w", err)
	}
	if trace.TapeSHA256 != evidence.SHA256FromSum(tape.SHA256) || uint64(trace.Decisions) != uint64(len(tape.Decisions)) {
		return errors.New("choice tape identity does not match its trace")
	}
	return nil
}
