package native

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"slices"

	nativebuild "go.temporal.io/server/tools/gomad3/internal/backends/native/build"
	"go.temporal.io/server/tools/gomad3/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func (provider *Provider) ValidateReplay(ctx context.Context, opened *artifact.Opened) (target.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	if opened == nil {
		return target.Prepared{}, errors.New("replay artifact is required")
	}
	manifest := opened.Manifest()
	if manifest.Target.Backend != nil {
		return target.Prepared{}, errors.New("native replay artifact contains external backend metadata")
	}
	profile := transcript.Default()
	if !profile.MatchesRecorded(manifest.IOProfile.Name, string(manifest.IOProfile.ImplementationSHA256), string(manifest.IOProfile.InventorySHA256), manifest.IOProfile.Inventory) {
		return target.Prepared{}, errors.New("artifact I/O profile identity does not match this Runner")
	}
	if manifest.ReplayMode != record.ReplayExact && manifest.ReplayMode != record.ReplayDiagnostic {
		return target.Prepared{}, fmt.Errorf("artifact replay mode %q cannot be executed", manifest.ReplayMode)
	}
	if manifest.Runner.HostOS != runtime.GOOS || manifest.Runner.HostArch != runtime.GOARCH || manifest.Toolchain.TargetGOOS != runtime.GOOS || manifest.Toolchain.TargetGOARCH != runtime.GOARCH {
		return target.Prepared{}, fmt.Errorf("artifact platform does not match host %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	identity, err := nativebuild.ReadToolchainIdentity(provider.config.ToolchainRoot)
	if err != nil {
		return target.Prepared{}, err
	}
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	if identity.GoVersion != manifest.Toolchain.GoVersion || identity.BuildKey != manifest.Toolchain.BuildKey || identity.TargetGOOS != manifest.Toolchain.TargetGOOS || identity.TargetGOARCH != manifest.Toolchain.TargetGOARCH {
		return target.Prepared{}, fmt.Errorf("artifact toolchain identity does not match the pinned toolchain")
	}
	targetFile, err := opened.OpenPayload(manifest.Target.File, uint64(manifest.Target.Size))
	if err != nil {
		return target.Prepared{}, err
	}
	capability, err := verifyReplayCapabilityManifest(opened, targetFile, identity)
	if err != nil {
		return target.Prepared{}, errors.Join(err, targetFile.Close())
	}
	path := targetFile.Name()
	info, err := buildinfo.Read(targetFile)
	closeErr := targetFile.Close()
	if err != nil {
		return target.Prepared{}, errors.Join(fmt.Errorf("read stored target build info: %w", err), closeErr)
	}
	if closeErr != nil {
		return target.Prepared{}, fmt.Errorf("close stored target build info: %w", closeErr)
	}
	if err := validateBuildInfo(info, manifest.Target.BuildInfo); err != nil {
		return target.Prepared{}, err
	}
	if err := nativebuild.VerifyCompatibility(manifest.Target.Compatibility); err != nil {
		return target.Prepared{}, fmt.Errorf("verify replay compatibility: %w", err)
	}
	if err := profile.VerifyAdapters(replayAdapters(manifest.Target.Adapters)); err != nil {
		return target.Prepared{}, fmt.Errorf("verify replay adapters: %w", err)
	}
	return target.Prepared{
		Path: path, Kind: target.Kind(manifest.Target.Kind), Source: manifest.Target.Source, SHA256: string(manifest.Target.SHA256), Size: uint64(manifest.Target.Size),
		Argv: append([]string(nil), manifest.Target.Argv...), BuildTags: append([]string(nil), manifest.Target.BuildTags...), Adapters: append([]record.TargetAdapter(nil), manifest.Target.Adapters...), Compatibility: append([]record.CompatibilityPack(nil), manifest.Target.Compatibility...),
		BuildInfo: manifest.Target.BuildInfo, GoVersion: identity.GoVersion, BuildKey: identity.BuildKey, TargetGOOS: identity.TargetGOOS, TargetGOARCH: identity.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(manifest.Target.CapabilityMode), CapabilityManifest: capability,
	}, nil
}

func verifyReplayCapabilityManifest(opened *artifact.Opened, targetFile *os.File, identity target.ToolchainIdentity) (*target.CapabilityManifest, error) {
	manifest := opened.Manifest()
	recorded := manifest.Target.CapabilityManifest
	switch manifest.Target.CapabilityMode {
	case "closure":
		if recorded != nil {
			return nil, errors.New("closure replay target contains a linked capability manifest")
		}
		return nil, nil
	case "linked", "guarded":
		if recorded == nil {
			return nil, errors.New("executable replay target capability manifest is missing")
		}
		actual, err := nativebuild.ReadCapabilityManifestFile(targetFile, identity)
		if err != nil {
			return nil, fmt.Errorf("extract stored target capability manifest: %w", err)
		}
		if *actual.Record() != *recorded {
			return nil, errors.New("stored target embedded capability manifest does not match the execution record")
		}
		payload, err := opened.ReadPayload(recorded.File, uint64(recorded.Bytes))
		if err != nil {
			return nil, fmt.Errorf("read stored target capability manifest: %w", err)
		}
		if !bytes.Equal(payload, actual.Payload) {
			return nil, errors.New("stored target capability manifest payload does not match its embedded record")
		}
		return actual, nil
	default:
		return nil, fmt.Errorf("unknown replay target capability mode %q", manifest.Target.CapabilityMode)
	}
}

func replayAdapters(adapters []record.TargetAdapter) []transcript.Adapter {
	result := make([]transcript.Adapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = transcript.Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func validateBuildInfo(info *debug.BuildInfo, expected record.BuildInfo) error {
	actual := nativebuild.ProjectBuildInfo(info)
	if nativebuild.HasCoverageInstrumentation(actual) {
		return errors.New("stored target uses unsupported coverage instrumentation")
	}
	if expected.GoVersion != actual.GoVersion || expected.Path != actual.Path || expected.MainModule != actual.MainModule ||
		(expected.Settings == nil) != (actual.Settings == nil) || !slices.Equal(expected.Settings, actual.Settings) {
		return fmt.Errorf("stored target build info does not match manifest")
	}
	return nil
}
