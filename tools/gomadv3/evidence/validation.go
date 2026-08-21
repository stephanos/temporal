package evidence

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateManifest(manifest ExecutionRecord, requireIdentities bool) error {
	if manifest.SchemaVersion != SchemaVersion && manifest.SchemaVersion != PriorSchemaVersion && manifest.SchemaVersion != PreviousSchemaVersion && manifest.SchemaVersion != LegacySchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d", manifest.SchemaVersion)
	}
	if err := validateArtifactReplay(manifest.ArtifactKind, manifest.ReplayMode, manifest.Outcome.Domain); err != nil {
		return err
	}
	if manifest.CreatedAt == "" || manifest.CampaignID == "" {
		return fmt.Errorf("manifest creation time and batch ID are required")
	}
	if _, err := time.Parse(time.RFC3339Nano, manifest.CreatedAt); err != nil {
		return fmt.Errorf("invalid manifest creation time: %w", err)
	}
	if requireIdentities {
		if err := validateSHA256(manifest.RecordHash); err != nil {
			return fmt.Errorf("invalid record hash: %w", err)
		}
		if err := validateSHA256(manifest.Outcome.FailureSignature); err != nil {
			return fmt.Errorf("invalid failure signature: %w", err)
		}
	}
	expectedContract := RecordContract
	if manifest.SchemaVersion == PriorSchemaVersion {
		expectedContract = PriorRecordContract
	} else if manifest.SchemaVersion == PreviousSchemaVersion {
		expectedContract = PreviousRecordContract
	} else if manifest.SchemaVersion == LegacySchemaVersion {
		expectedContract = LegacyRecordContract
		if manifest.ChoiceProfile != nil || manifest.Limits.ChoiceTraceBytes != 0 {
			return errors.New("schema v2 manifest cannot contain a choice trace")
		}
	}
	if manifest.Runner.RecordContract != expectedContract || manifest.Runner.RunnerBuild == "" || manifest.Runner.HostOS == "" || manifest.Runner.HostArch == "" {
		return fmt.Errorf("invalid Runner identity")
	}
	if manifest.Toolchain.GoVersion == "" || !isLowerHex(manifest.Toolchain.BuildKey, 64) || manifest.Toolchain.TargetGOOS == "" || manifest.Toolchain.TargetGOARCH == "" {
		return fmt.Errorf("invalid toolchain identity")
	}
	if err := validateTarget(manifest.SchemaVersion, manifest.Target); err != nil {
		return err
	}
	if err := validateIOProfile(manifest.IOProfile); err != nil {
		return err
	}
	if err := validateChoiceProfile(manifest.SchemaVersion, manifest.ChoiceProfile, manifest.Limits.ChoiceTraceBytes, manifest.ArtifactKind, manifest.Outcome.Reason); err != nil {
		return err
	}
	if err := validateSimulationProfile(manifest.SchemaVersion, manifest.ReplayMode, manifest.SimulationProfile); err != nil {
		return err
	}
	choiceProfile := ""
	if manifest.ChoiceProfile != nil {
		choiceProfile = manifest.ChoiceProfile.Name
	}
	if err := validateEnvironment(manifest.Environment, uint64(manifest.Seed), manifest.IOProfile.Name, choiceProfile); err != nil {
		return err
	}
	if manifest.Limits.RunTimeoutNanos == 0 || manifest.Limits.OverallTimeoutNanos == 0 || manifest.Limits.OutputBytes == 0 || manifest.Limits.WorldTransitionBytes == 0 {
		return fmt.Errorf("invalid zero execution limit")
	}
	if manifest.Limits.TerminateGraceNanos > manifest.Limits.RunTimeoutNanos || manifest.Limits.TerminateGraceNanos > manifest.Limits.OverallTimeoutNanos {
		return fmt.Errorf("termination grace exceeds an execution deadline")
	}
	if err := validateWorld(manifest.World); err != nil {
		return err
	}
	if manifest.Outcome.Reason == "" || manifest.Outcome.Termination == "" {
		return fmt.Errorf("outcome reason and termination are required")
	}
	if err := validateOutcome(manifest.Outcome); err != nil {
		return err
	}
	files, err := validateFiles(manifest.Files)
	if err != nil {
		return err
	}
	if err := validateFileReference(files, manifest.Target.File, manifest.Target.SHA256, manifest.Target.Size); err != nil {
		return fmt.Errorf("target file: %w", err)
	}
	if manifest.Target.CapabilityManifest != nil {
		capabilities := manifest.Target.CapabilityManifest
		if err := validateFileReference(files, capabilities.File, capabilities.SHA256, capabilities.Bytes); err != nil {
			return fmt.Errorf("target capability manifest file: %w", err)
		}
	}
	if err := validateStream(files, "stdout", manifest.Streams.Stdout); err != nil {
		return err
	}
	if err := validateStream(files, "stderr", manifest.Streams.Stderr); err != nil {
		return err
	}
	if manifest.IOProfile.Transcript != nil {
		transcript := manifest.IOProfile.Transcript
		if err := validateFileReference(files, transcript.File, transcript.SHA256, transcript.Bytes); err != nil {
			return fmt.Errorf("I/O transcript file: %w", err)
		}
		if transcript.Bytes > manifest.Limits.IOTranscriptBytes {
			return errors.New("I/O transcript file exceeds the recorded limit")
		}
	}
	if manifest.IOProfile.ReadOnlyMounts != nil {
		mounts := manifest.IOProfile.ReadOnlyMounts
		if err := validateFileReference(files, mounts.File, mounts.SHA256, mounts.Bytes); err != nil {
			return fmt.Errorf("read-only mount descriptor file: %w", err)
		}
	}
	if manifest.ChoiceProfile != nil {
		trace := manifest.ChoiceProfile.Trace
		if err := validateFileReference(files, trace.File, trace.SHA256, trace.Bytes); err != nil {
			return fmt.Errorf("choice trace file: %w", err)
		}
	}
	if manifest.SimulationProfile != nil {
		profile := manifest.SimulationProfile
		if err := validateFileReference(files, profile.Plan.File, profile.Plan.SHA256, profile.Plan.Bytes); err != nil {
			return fmt.Errorf("simulation plan file: %w", err)
		}
		if err := validateFileReference(files, profile.Record.File, profile.Record.SHA256, profile.Record.Bytes); err != nil {
			return fmt.Errorf("simulation record file: %w", err)
		}
	}
	if err := validateFileReference(files, manifest.World.Initial.File, manifest.World.Initial.RawSHA256, files[manifest.World.Initial.File].Size); err != nil {
		return fmt.Errorf("initial World file: %w", err)
	}
	if err := validateFileReference(files, manifest.World.Transitions.File, manifest.World.Transitions.RawSHA256, files[manifest.World.Transitions.File].Size); err != nil {
		return fmt.Errorf("World transitions file: %w", err)
	}
	if files[manifest.World.Transitions.File].Size > manifest.Limits.WorldTransitionBytes {
		return fmt.Errorf("World transitions file exceeds the recorded limit")
	}
	if err := validateFileReference(files, manifest.World.Final.File, manifest.World.Final.RawSHA256, files[manifest.World.Final.File].Size); err != nil {
		return fmt.Errorf("final World file: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, manifest.Host.StartedAt); err != nil {
		return fmt.Errorf("invalid host start time: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, manifest.Host.FinishedAt); err != nil {
		return fmt.Errorf("invalid host finish time: %w", err)
	}
	return nil
}

func validateChoiceProfile(schemaVersion uint32, profile *ChoiceProfile, limit Uint64String, artifactKind, outcomeReason string) error {
	const choiceTraceHeaderBytes = 64
	if profile == nil {
		if limit != 0 {
			return errors.New("choice trace limit requires a choice profile")
		}
		return nil
	}
	trace := profile.Trace
	recordBytes := Uint64String(96)
	if schemaVersion == PreviousSchemaVersion {
		recordBytes = 48
		if profile.Name != "gomadv3-choice-trace/v1" || trace.Schema != "gomadv3.choice-trace/v1" || trace.TapeSHA256 != "" || trace.Decisions != 0 {
			return errors.New("invalid schema v3 choice trace identity")
		}
	} else if schemaVersion != SchemaVersion && schemaVersion != PriorSchemaVersion || profile.Name != "gomadv3-choice-trace/v2" || trace.Schema != "gomadv3.choice-trace/v2" {
		return errors.New("invalid choice trace identity")
	}
	if trace.File != "choices.bin" {
		return errors.New("invalid choice trace identity")
	}
	switch trace.TerminalState {
	case "complete":
		if schemaVersion == SchemaVersion || schemaVersion == PriorSchemaVersion {
			if err := validateSHA256(trace.TapeSHA256); err != nil {
				return fmt.Errorf("invalid choice tape hash: %w", err)
			}
			if trace.Decisions > trace.Records || trace.BranchingRecords > trace.Decisions {
				return errors.New("invalid choice trace decision counts")
			}
		}
	case "overflow":
		if artifactKind != ArtifactRunnerFailure || outcomeReason != "choice_trace_overflow" {
			return errors.New("choice trace overflow requires a matching Runner failure")
		}
		if trace.TapeSHA256 != "" || trace.Decisions != 0 {
			return errors.New("overflow choice trace cannot claim an exact tape")
		}
	default:
		return errors.New("invalid choice trace terminal state")
	}
	if err := validateSHA256(profile.ImplementationSHA256); err != nil {
		return fmt.Errorf("invalid choice profile implementation hash: %w", err)
	}
	if err := validateSHA256(trace.SHA256); err != nil {
		return fmt.Errorf("invalid choice trace hash: %w", err)
	}
	if limit < choiceTraceHeaderBytes+recordBytes || trace.Limit != limit || trace.Bytes > limit-choiceTraceHeaderBytes || trace.Bytes%recordBytes != 0 || trace.Records != trace.Bytes/recordBytes || trace.BranchingRecords > trace.Records {
		return errors.New("invalid choice trace limits or counts")
	}
	return nil
}

func validateSimulationProfile(schemaVersion uint32, replayMode string, profile *SimulationProfile) error {
	if profile == nil {
		return nil
	}
	if schemaVersion != SchemaVersion {
		return errors.New("historical manifest contains simulation exploration evidence")
	}
	if replayMode != ReplayExact {
		return errors.New("simulation exploration evidence requires exact replay")
	}
	if profile.Name != "gomadv3-simulation-exploration/v1" || profile.Plan.Schema != "gomadv3.simulation-exploration-plan/v1" || profile.Plan.File != "simulation/plan.json" || profile.Record.Schema != "gomadv3.cluster-record/v7" || profile.Record.File != "simulation/record.json" {
		return errors.New("invalid simulation exploration identity")
	}
	for _, identity := range []struct {
		name  string
		value SHA256
	}{
		{name: "controller", value: profile.ControllerSHA256},
		{name: "execution", value: profile.ExecutionSHA256},
		{name: "candidate", value: profile.CandidateSHA256},
		{name: "outcome", value: profile.OutcomeSHA256},
		{name: "plan", value: profile.Plan.SHA256},
		{name: "record", value: profile.Record.SHA256},
	} {
		if err := validateSHA256(identity.value); err != nil {
			return fmt.Errorf("invalid simulation %s hash: %w", identity.name, err)
		}
	}
	if profile.FailureSHA256 != "" {
		if err := validateSHA256(profile.FailureSHA256); err != nil {
			return fmt.Errorf("invalid simulation failure hash: %w", err)
		}
	}
	if profile.Plan.Bytes == 0 || profile.Record.Bytes == 0 || profile.Record.Limit == 0 || profile.Record.Bytes > profile.Record.Limit {
		return errors.New("invalid simulation exploration payload bounds")
	}
	return nil
}

func validateIOProfile(profile IOProfile) error {
	if profile.Name == "" {
		return errors.New("I/O profile identity is required")
	}
	if err := validateSHA256(profile.ImplementationSHA256); err != nil {
		return fmt.Errorf("invalid I/O profile implementation hash: %w", err)
	}
	if err := validateSHA256(profile.InventorySHA256); err != nil {
		return fmt.Errorf("invalid I/O profile inventory hash: %w", err)
	}
	if HashBytes([]byte(profile.Inventory)) != profile.InventorySHA256 {
		return errors.New("I/O profile inventory hash does not match its bytes")
	}
	var decoded any
	if err := DecodeCanonicalJSON([]byte(profile.Inventory), &decoded); err != nil {
		return fmt.Errorf("decode I/O profile inventory: %w", err)
	}
	if profile.Transcript != nil {
		if profile.Transcript.Schema != "gomadv3.io-transcript/v1" || profile.Transcript.File != "io/transcript.bin" {
			return errors.New("invalid I/O transcript identity")
		}
		if err := validateSHA256(profile.Transcript.SHA256); err != nil {
			return fmt.Errorf("invalid I/O transcript hash: %w", err)
		}
	}
	if mounts := profile.ReadOnlyMounts; mounts != nil {
		if mounts.Schema != "gomadv3.io-read-only-mounts/v1" || mounts.File != "io/mounts.json" || len(mounts.Mappings) == 0 {
			return errors.New("invalid read-only mount identity")
		}
		if err := validateSHA256(mounts.SHA256); err != nil {
			return fmt.Errorf("invalid read-only mount descriptor hash: %w", err)
		}
		if mounts.Bytes == 0 || mounts.Limits.PathBytes == 0 || mounts.Limits.Requests == 0 || mounts.Limits.Files == 0 || mounts.Limits.DirectoryEntries == 0 || mounts.Limits.SingleFileBytes == 0 || mounts.Limits.TotalBytes == 0 || mounts.Limits.SingleFileBytes > mounts.Limits.TotalBytes || mounts.TotalBytes > mounts.Limits.TotalBytes {
			return errors.New("invalid read-only mount limits")
		}
		previous := ""
		for index, target := range mounts.Mappings {
			if target == "/" || !strings.HasPrefix(target, "/") || path.Clean(target) != target || index > 0 && (target <= previous || strings.HasPrefix(target, previous+"/") || strings.HasPrefix(previous, target+"/")) {
				return errors.New("read-only mount mappings must be absolute, sorted, unique, and non-overlapping")
			}
			previous = target
		}
	}
	return nil
}

func validateArtifactReplay(artifactKind, replayMode, outcomeDomain string) error {
	switch artifactKind {
	case ArtifactSuccess:
		if replayMode != ReplayExact || outcomeDomain != "success" {
			return fmt.Errorf("successful run requires exact replay and success outcome")
		}
	case ArtifactTargetFailure:
		if replayMode != ReplayExact || outcomeDomain != "target" {
			return fmt.Errorf("target failure requires exact replay and target outcome")
		}
	case ArtifactWatchdogTimeout:
		if replayMode != ReplayDiagnostic || outcomeDomain != "watchdog" {
			return fmt.Errorf("watchdog failure requires diagnostic replay and watchdog outcome")
		}
	case ArtifactRunnerFailure:
		if replayMode != ReplayNone || outcomeDomain != "runner" {
			return fmt.Errorf("Runner failure requires no replay and Runner outcome")
		}
	default:
		return fmt.Errorf("unknown artifact kind %q", artifactKind)
	}
	return nil
}

func validateTarget(schemaVersion uint32, target Target) error {
	if target.Kind != "exec" && target.Kind != "go-run" && target.Kind != "go-test" {
		return fmt.Errorf("unknown target kind %q", target.Kind)
	}
	if target.Source == "" || target.File == "" || target.Size == 0 || target.BuildInfo.GoVersion == "" || target.BuildInfo.Path == "" {
		return fmt.Errorf("incomplete target identity")
	}
	if err := validateSHA256(target.SHA256); err != nil {
		return fmt.Errorf("invalid target hash: %w", err)
	}
	if len(target.Argv) == 0 || target.Argv[0] != "gomadv3-target" {
		return fmt.Errorf("target argv must start with gomadv3-target")
	}
	if !sortedUniqueStrings(target.BuildTags) || !sortedTargetAdapters(target.Adapters) || !sortedCompatibilityPacks(target.Compatibility) || !sortedBuildSettings(target.BuildInfo.Settings) {
		return errors.New("target build tags, adapters, compatibility packs, and settings must be canonical")
	}
	return validateTargetCapability(schemaVersion, target)
}

func validateTargetCapability(schemaVersion uint32, target Target) error {
	if schemaVersion != SchemaVersion {
		if target.CapabilityMode != "" || target.CapabilityManifest != nil {
			return errors.New("historical target contains linked capability evidence")
		}
		return nil
	}
	switch target.CapabilityMode {
	case "closure":
		if target.CapabilityManifest != nil {
			return errors.New("closure target contains a linked capability manifest")
		}
	case "linked":
		manifest := target.CapabilityManifest
		if manifest == nil || manifest.Schema != "gomadv3.live-capability-manifest/v1" || manifest.File != "target-capabilities.json" || manifest.Bytes == 0 {
			return errors.New("linked target capability manifest identity is incomplete")
		}
		if err := validateSHA256(manifest.SHA256); err != nil {
			return fmt.Errorf("invalid target capability manifest hash: %w", err)
		}
		if err := validateSHA256(manifest.ProducerImplementationSHA256); err != nil {
			return fmt.Errorf("invalid target capability producer hash: %w", err)
		}
		if err := validateSHA256(manifest.CapabilityUniverseSHA256); err != nil {
			return fmt.Errorf("invalid target capability universe hash: %w", err)
		}
	default:
		return fmt.Errorf("unknown target capability mode %q", target.CapabilityMode)
	}
	return nil
}

func ValidateCurrentTarget(target Target) error {
	return validateTarget(SchemaVersion, target)
}

func ValidateCurrentTargetCapability(target Target) error {
	return validateTargetCapability(SchemaVersion, target)
}

func sortedTargetAdapters(adapters []TargetAdapter) bool {
	if adapters == nil {
		return false
	}
	for index, adapter := range adapters {
		if adapter.Module == "" || adapter.Version == "" || adapter.Sum == "" || index > 0 && adapters[index-1].Module >= adapter.Module {
			return false
		}
	}
	return true
}

func ValidateTargetAdapters(adapters []TargetAdapter) error {
	if !sortedTargetAdapters(adapters) {
		return errors.New("target adapters are not canonical")
	}
	return nil
}

func sortedCompatibilityPacks(packs []CompatibilityPack) bool {
	if packs == nil {
		return false
	}
	for index, pack := range packs {
		if pack.ID == "" || validateSHA256(pack.SHA256) != nil || index > 0 && packs[index-1].ID >= pack.ID {
			return false
		}
	}
	return true
}

func ValidateCompatibilityPacks(packs []CompatibilityPack) error {
	if !sortedCompatibilityPacks(packs) {
		return errors.New("compatibility packs are not canonical")
	}
	return nil
}

func validateEnvironment(environment []Environment, seed uint64, ioProfile, choiceProfile string) error {
	reserved := map[string]struct{}{
		"GOMADV3_CHILD_SEED": {}, "CGO_ENABLED": {}, "GODEBUG": {}, "GOMAXPROCS": {}, "GOEXPERIMENT": {},
		"LIBPATH": {}, "SHLIB_PATH": {},
	}
	previous := ""
	foundSeed := false
	foundTimezone := false
	foundIOProfile := false
	hasIOProfile := false
	foundChoiceProfile := false
	hasChoiceProfile := false
	for index, entry := range environment {
		if !environmentNamePattern.MatchString(entry.Name) || strings.IndexByte(entry.Value, 0) >= 0 {
			return fmt.Errorf("invalid environment entry %q", entry.Name)
		}
		if index > 0 && entry.Name <= previous {
			return fmt.Errorf("environment entries must be sorted and unique")
		}
		previous = entry.Name
		if _, found := reserved[entry.Name]; found || strings.HasPrefix(entry.Name, "LD_") || strings.HasPrefix(entry.Name, "DYLD_") {
			return fmt.Errorf("environment name %q is reserved", entry.Name)
		}
		switch entry.Name {
		case "GOMADSEED":
			foundSeed = entry.Value == fmt.Sprintf("%d", seed)
		case "GOMADV3_IO_PROFILE":
			hasIOProfile = true
			foundIOProfile = ioProfile != "" && entry.Value == ioProfile
		case "GOMADV3_CHOICE_PROFILE":
			hasChoiceProfile = true
			foundChoiceProfile = choiceProfile != "" && entry.Value == choiceProfile
		case "TZ":
			foundTimezone = entry.Value == "UTC"
		}
	}
	if !foundSeed || !foundTimezone {
		return fmt.Errorf("environment must contain the recorded GOMADSEED and TZ=UTC")
	}
	if hasIOProfile != (ioProfile != "") || hasIOProfile && !foundIOProfile {
		return errors.New("environment must match the recorded I/O profile")
	}
	if hasChoiceProfile != (choiceProfile != "") || hasChoiceProfile && !foundChoiceProfile {
		return errors.New("environment must match the recorded choice profile")
	}
	return nil
}

func validateWorld(world World) error {
	switch world.Terminal.Kind {
	case "none", "delivered", "idle", "deadlock":
		if world.Terminal.Detail != "" {
			return fmt.Errorf("World quiescence terminal has detail")
		}
	case "capacity", "replay-divergence", "invalid-input":
		if world.Terminal.Detail == "" {
			return fmt.Errorf("World error terminal omitted detail")
		}
	default:
		return fmt.Errorf("invalid World terminal kind %q", world.Terminal.Kind)
	}
	for name, payload := range map[string]WorldPayload{"initial": world.Initial, "final": world.Final} {
		if payload.Schema == "" || payload.File == "" {
			return fmt.Errorf("incomplete %s World payload", name)
		}
		if err := validateSHA256(payload.RawSHA256); err != nil {
			return fmt.Errorf("invalid %s World raw hash: %w", name, err)
		}
		if err := validateSHA256(payload.SemanticDigest); err != nil {
			return fmt.Errorf("invalid %s World semantic digest: %w", name, err)
		}
	}
	if world.Transitions.Schema == "" || world.Transitions.File == "" {
		return fmt.Errorf("incomplete World transitions payload")
	}
	if err := validateSHA256(world.Transitions.RawSHA256); err != nil {
		return fmt.Errorf("invalid World transitions raw hash: %w", err)
	}
	if err := validateSHA256(world.Transitions.TranscriptDigest); err != nil {
		return fmt.Errorf("invalid World transcript digest: %w", err)
	}
	previous := ""
	for index, adapter := range world.Adapters {
		if adapter.Schema == "" || index > 0 && adapter.Schema <= previous {
			return fmt.Errorf("World adapter schemas must be nonempty, sorted, and unique")
		}
		if err := validateSHA256(adapter.InitialDigest); err != nil {
			return fmt.Errorf("invalid World adapter initial digest: %w", err)
		}
		if err := validateSHA256(adapter.FinalDigest); err != nil {
			return fmt.Errorf("invalid World adapter final digest: %w", err)
		}
		previous = adapter.Schema
	}
	return nil
}

func validateOutcome(outcome Outcome) error {
	switch outcome.Termination {
	case "exit":
		if outcome.ExitCode == nil || outcome.Signal != nil || outcome.Deadline != nil {
			return fmt.Errorf("exit termination has incompatible fields")
		}
	case "signal":
		if outcome.ExitCode != nil || outcome.Signal == nil || *outcome.Signal == "" || outcome.Deadline != nil {
			return fmt.Errorf("signal termination has incompatible fields")
		}
	case "timeout":
		if outcome.ExitCode != nil || outcome.Signal != nil || outcome.Deadline == nil || *outcome.Deadline == "" {
			return fmt.Errorf("timeout termination has incompatible fields")
		}
	case "none":
		if outcome.ExitCode != nil || outcome.Signal != nil || outcome.Deadline != nil {
			return fmt.Errorf("none termination has incompatible fields")
		}
	default:
		return fmt.Errorf("unknown outcome termination %q", outcome.Termination)
	}
	return nil
}

func validateFiles(files []File) (map[string]File, error) {
	indexed := make(map[string]File, len(files))
	previous := ""
	for index, file := range files {
		if err := validateRelativePath(file.Path); err != nil {
			return nil, err
		}
		if index > 0 && file.Path <= previous {
			return nil, fmt.Errorf("manifest files must be sorted and unique")
		}
		previous = file.Path
		if file.Mode != "0600" && file.Mode != "0700" {
			return nil, fmt.Errorf("invalid file mode %q for %s", file.Mode, file.Path)
		}
		if err := validateSHA256(file.SHA256); err != nil {
			return nil, fmt.Errorf("invalid file hash for %s: %w", file.Path, err)
		}
		indexed[file.Path] = file
	}
	return indexed, nil
}

func validateFileReference(files map[string]File, fileName string, hash SHA256, size Uint64String) error {
	file, ok := files[fileName]
	if !ok {
		return fmt.Errorf("unlisted file %q", fileName)
	}
	if file.SHA256 != hash || file.Size != size {
		return fmt.Errorf("file identity mismatch for %q", fileName)
	}
	return nil
}

func validateStream(files map[string]File, name string, stream Stream) error {
	if err := validateSHA256(stream.RetainedSHA256); err != nil {
		return fmt.Errorf("invalid %s retained hash: %w", name, err)
	}
	if err := validateSHA256(stream.FullSHA256); err != nil {
		return fmt.Errorf("invalid %s full hash: %w", name, err)
	}
	if uint64(stream.RetainedBytes) > uint64(stream.TotalBytes) || uint64(stream.DiscardedBytes) != uint64(stream.TotalBytes)-uint64(stream.RetainedBytes) {
		return fmt.Errorf("invalid %s stream byte accounting", name)
	}
	if stream.Truncated != (stream.DiscardedBytes != 0) {
		return fmt.Errorf("invalid %s stream truncation state", name)
	}
	if !stream.Truncated && stream.FullSHA256 != stream.RetainedSHA256 {
		return fmt.Errorf("invalid %s untruncated stream hashes", name)
	}
	file, ok := files[stream.File]
	if !ok || file.SHA256 != stream.RetainedSHA256 {
		return fmt.Errorf("invalid %s retained file", name)
	}
	if !stream.Truncated && file.Size != stream.RetainedBytes {
		return fmt.Errorf("invalid %s retained file size", name)
	}
	return nil
}

func validateRelativePath(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." || strings.Contains(value, "\\") {
		return fmt.Errorf("invalid artifact path %q", value)
	}
	return nil
}

func validateSHA256(value SHA256) error {
	_, err := ParseSHA256(string(value))
	return err
}

func isLowerHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func sortedUniqueStrings(values []string) bool {
	return sort.SliceIsSorted(values, func(i, j int) bool { return values[i] < values[j] }) && !hasAdjacentDuplicate(values)
}

func hasAdjacentDuplicate(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func sortedBuildSettings(settings []BuildSetting) bool {
	for index := 1; index < len(settings); index++ {
		if settings[index].Key <= settings[index-1].Key {
			return false
		}
	}
	return true
}
