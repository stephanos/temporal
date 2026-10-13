package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"unicode/utf8"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/internal/strictjson"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/target"
)

const (
	// campaignPlanSchema is the portable plan lineage this build writes and
	// the only one it opens. Its embedded execution limits are the campaign
	// log's: the per-execution payload bound, the outcome count and byte
	// bounds and the partial executions, under campaignPlanStorage.
	// Plans of the earlier gomad3.campaign-plan/v1 lineage bound a segmented
	// journal and are refused by name.
	campaignPlanSchema = "gomad3.portable-campaign-plan/v1"
	// campaignPlanStorage is the campaign storage a plan's shards write.
	campaignPlanStorage      = campaign.CampaignLogFormat
	campaignPlanMapping      = "ordinal-modulo/v1"
	campaignPlanBundleSuffix = ".bundle"
	campaignPlanTargetFile   = "target"
	maximumCampaignPlanBytes = 1 << 20
)

type CampaignPlanSpec struct {
	Campaign CampaignSpec
	Output   string
}

type CampaignPlanResult struct {
	Path           string        `json:"path"`
	BundlePath     string        `json:"bundle_path"`
	SHA256         record.SHA256 `json:"sha256"`
	SelectionCount uint64        `json:"selection_count"`
	TargetSHA256   record.SHA256 `json:"target_sha256"`
}

type portableCampaignPlan struct {
	Schema   string                     `json:"schema"`
	Storage  string                     `json:"storage"`
	Mapping  string                     `json:"mapping"`
	Mounts   *campaignPlanMountIdentity `json:"mounts,omitempty"`
	Campaign campaign.CampaignPlan      `json:"campaign"`
}

type campaignPlanMountIdentity struct {
	Schema     string                       `json:"schema"`
	SHA256     record.SHA256                `json:"sha256"`
	Bytes      record.Uint64String          `json:"bytes"`
	Entries    record.Uint64String          `json:"entries"`
	TotalBytes record.Uint64String          `json:"total_bytes"`
	Mappings   []string                     `json:"mappings"`
	Limits     captured.CapturedInputLimits `json:"limits"`
}

type openedCampaignPlan struct {
	path     string
	identity record.SHA256
	plan     campaign.CampaignPlan
	mounts   *campaignPlanMountIdentity
	prepared target.Prepared
}

func CreateCampaignPlan(ctx context.Context, spec CampaignPlanSpec) (_ CampaignPlanResult, retErr error) {
	return createCampaignPlanWith(ctx, spec, executionDependencies{})
}

func createCampaignPlanWith(ctx context.Context, spec CampaignPlanSpec, dependencies executionDependencies) (_ CampaignPlanResult, retErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	config := campaignRequestFromSpecWith(spec.Campaign, dependencies)
	selection, environment, err := validateCampaignRequest(config)
	if err != nil {
		return CampaignPlanResult{}, err
	}
	config.profile, err = config.Backend.Describe(ctx, backend.Configuration{Target: config.Target, ClockTick: config.ClockTick, Environment: config.Environment, RequiredSemanticProbes: config.RequiredSemanticProbes})
	if err != nil {
		return CampaignPlanResult{}, err
	}
	if config.strategy() != StrategySeed || config.OnFailure != PolicyAll {
		return CampaignPlanResult{}, errors.New("portable campaign plans require a seed campaign with on-failure=all")
	}
	if config.Shard.Count != 0 || config.PlanSHA256 != "" || config.ResumeCampaign != "" {
		return CampaignPlanResult{}, errors.New("portable campaign plan input cannot already be a shard or resume")
	}
	output, err := filepath.Abs(spec.Output)
	if err != nil || spec.Output == "" {
		return CampaignPlanResult{}, errors.Join(errors.New("resolve campaign plan output"), err)
	}
	bundle := output + campaignPlanBundleSuffix
	for _, path := range []string{output, bundle} {
		if _, err := os.Lstat(path); err == nil {
			return CampaignPlanResult{}, fmt.Errorf("campaign plan output already exists: %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return CampaignPlanResult{}, fmt.Errorf("inspect campaign plan output: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o700); err != nil {
		return CampaignPlanResult{}, fmt.Errorf("create campaign plan parent: %w", err)
	}
	if err := os.Mkdir(bundle, 0o700); err != nil {
		return CampaignPlanResult{}, fmt.Errorf("create campaign plan bundle: %w", err)
	}
	published := false
	defer func() {
		if !published {
			retErr = errors.Join(retErr, os.RemoveAll(bundle))
		}
	}()
	if config.IOROMountLimits == (captured.Limits{}) {
		config.IOROMountLimits = captured.DefaultLimits()
	}
	mounts, err := captured.ParseMappings(config.IOROMounts, config.Target.WorkingDir)
	if err != nil {
		return CampaignPlanResult{}, err
	}
	mounts, portableMounts := canonicalCampaignPlanMounts(mounts)
	mountIdentity, capturedMounts, err := captureCampaignPlanMounts(mounts, config.IOROMountLimits)
	if err != nil {
		return CampaignPlanResult{}, fmt.Errorf("capture campaign plan mount identity: %w", err)
	}
	config.Target.PreparationRoot = bundle
	prepared, err := config.prepareTarget(ctx, preparationRequest{
		Target: config.Target, Environment: recordedEnvironmentStrings(environment), Preparer: config.Preparer, Provider: config.Backend,
	})
	if err != nil {
		return CampaignPlanResult{}, err
	}
	targetPath := filepath.Join(bundle, campaignPlanTargetFile)
	if filepath.Clean(prepared.Path) != targetPath {
		preparationDirectory := filepath.Dir(prepared.Path)
		relative, relativeErr := filepath.Rel(bundle, preparationDirectory)
		if relativeErr != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) || len(relative) >= 3 && relative[:3] == ".."+string(filepath.Separator) {
			return CampaignPlanResult{}, errors.Join(errors.New("prepared campaign target is outside its private preparation directory"), relativeErr)
		}
		if err := os.Rename(prepared.Path, targetPath); err != nil {
			return CampaignPlanResult{}, fmt.Errorf("publish prepared campaign target: %w", err)
		}
		if err := os.RemoveAll(preparationDirectory); err != nil {
			return CampaignPlanResult{}, fmt.Errorf("remove campaign target preparation files: %w", err)
		}
	}
	if err := os.Chmod(targetPath, 0o500); err != nil {
		return CampaignPlanResult{}, fmt.Errorf("make campaign target immutable: %w", err)
	}
	prepared.Path = targetPath
	if err := config.Backend.ValidatePrepared(config.Target, prepared, recordedEnvironmentStrings(environment)); err != nil {
		return CampaignPlanResult{}, err
	}
	if err := materializeCampaignPlanMounts(bundle, capturedMounts); err != nil {
		return CampaignPlanResult{}, fmt.Errorf("materialize campaign plan mounts: %w", err)
	}
	if config.Guide {
		config.Corpus, err = guidedCorpusPath(config.Corpus)
		if err != nil {
			return CampaignPlanResult{}, err
		}
		guidance, err := openGuidance(ctx, config, prepared, environment, "plan")
		if err != nil {
			return CampaignPlanResult{}, err
		}
		snapshot := guidance.Snapshot()
		selection, config.guidancePlan, err = selectGuidedSeeds(selection, snapshot, config.Corpus, config.GuideRegression)
		err = errors.Join(err, guidance.Close())
		if err != nil {
			return CampaignPlanResult{}, err
		}
		config.Seeds = selection.String()
		config.GuideSnapshotSHA256 = snapshot.SnapshotSHA256
	}
	journalPlan, err := campaign.DeriveExecutionJournalPlan(string(StrategySeed), selection.Count(), 0, uint64(config.Parallel))
	if err != nil {
		return CampaignPlanResult{}, fmt.Errorf("derive campaign journal capacity: %w", err)
	}
	plan, err := campaignPlanRecord(config, journalPlan, ".prepared/target", prepared, environment, portableMounts, selection.Count())
	if err != nil {
		return CampaignPlanResult{}, err
	}
	if err := campaign.ValidateCampaignPlan(plan); err != nil {
		return CampaignPlanResult{}, err
	}
	document := portableCampaignPlan{Schema: campaignPlanSchema, Storage: campaignPlanStorage, Mapping: campaignPlanMapping, Mounts: mountIdentity, Campaign: plan}
	encoded, err := encodePortableCampaignPlan(document)
	if err != nil {
		return CampaignPlanResult{}, fmt.Errorf("encode campaign plan: %w", err)
	}
	if len(encoded) > maximumCampaignPlanBytes {
		return CampaignPlanResult{}, fmt.Errorf("campaign plan exceeds its %d-byte capacity", maximumCampaignPlanBytes)
	}
	if err := syncRegularFile(targetPath); err != nil {
		return CampaignPlanResult{}, err
	}
	if err := hostfs.SyncDirectory(bundle); err != nil {
		return CampaignPlanResult{}, err
	}
	if err := publishPlanFile(output, encoded); err != nil {
		return CampaignPlanResult{}, err
	}
	published = true
	return CampaignPlanResult{
		Path: output, BundlePath: bundle, SHA256: record.HashBytes(encoded), SelectionCount: selection.Count(), TargetSHA256: record.SHA256(prepared.SHA256),
	}, nil
}

func openCampaignPlan(path string) (_ openedCampaignPlan, retErr error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return openedCampaignPlan{}, fmt.Errorf("resolve campaign plan path: %w", err)
	}
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return openedCampaignPlan{}, fmt.Errorf("open campaign plan: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if info.Mode().Perm() != 0o600 || info.Size() > maximumCampaignPlanBytes {
		return openedCampaignPlan{}, errors.New("campaign plan file mode or size is invalid")
	}
	contents, err := io.ReadAll(io.LimitReader(file, maximumCampaignPlanBytes+1))
	if err != nil {
		return openedCampaignPlan{}, fmt.Errorf("read campaign plan: %w", err)
	}
	if len(contents) > maximumCampaignPlanBytes {
		return openedCampaignPlan{}, errors.New("campaign plan exceeds its byte capacity")
	}
	if err := checkCampaignPlanLineage(contents); err != nil {
		return openedCampaignPlan{}, err
	}
	var document portableCampaignPlan
	if err := strictjson.Decode(contents, &document); err != nil {
		return openedCampaignPlan{}, fmt.Errorf("decode campaign plan: %w", err)
	}
	if document.Schema != campaignPlanSchema || document.Storage != campaignPlanStorage || document.Mapping != campaignPlanMapping || document.Campaign.Strategy != string(StrategySeed) || document.Campaign.OnFailure != string(PolicyAll) || document.Campaign.PlanSHA256 != "" || document.Campaign.Shard != nil {
		return openedCampaignPlan{}, errors.New("campaign plan protocol identity is invalid")
	}
	if err := campaign.ValidateCampaignPlan(document.Campaign); err != nil {
		return openedCampaignPlan{}, err
	}
	journal, err := campaign.DeriveExecutionJournalPlan(document.Campaign.Strategy, uint64(document.Campaign.SelectionCount), uint64(document.Campaign.MaxExecutions), uint64(document.Campaign.Parallel))
	if err != nil || *document.Campaign.Journal != journal {
		return openedCampaignPlan{}, errors.Join(errors.New("campaign plan execution limits are not the campaign log limits"), err)
	}
	if (document.Mounts == nil) != (len(document.Campaign.IOROMounts) == 0) {
		return openedCampaignPlan{}, errors.New("campaign plan mount identity is incomplete")
	}
	bundle := path + campaignPlanBundleSuffix
	bundleInfo, err := os.Lstat(bundle)
	if err != nil || !bundleInfo.IsDir() || bundleInfo.Mode()&os.ModeSymlink != 0 || bundleInfo.Mode().Perm() != 0o700 {
		return openedCampaignPlan{}, errors.Join(errors.New("campaign plan bundle is not a private directory"), err)
	}
	if err := validateCampaignPlanBundleInventory(bundle, len(document.Campaign.IOROMounts)); err != nil {
		return openedCampaignPlan{}, err
	}
	if err := validateCampaignPlanMountIdentity(document.Campaign, document.Mounts, bundle); err != nil {
		return openedCampaignPlan{}, err
	}
	targetPath := filepath.Join(bundle, campaignPlanTargetFile)
	targetFile, targetInfo, err := hostfs.OpenPath(targetPath)
	if err != nil {
		return openedCampaignPlan{}, fmt.Errorf("open campaign plan target: %w", err)
	}
	if closeErr := targetFile.Close(); closeErr != nil {
		return openedCampaignPlan{}, closeErr
	}
	if targetInfo.Mode().Perm() != 0o500 {
		return openedCampaignPlan{}, errors.New("campaign plan target mode is invalid")
	}
	targetRecord := document.Campaign.Prepared.Target
	prepared := target.Prepared{
		Path: targetPath, Kind: target.Kind(targetRecord.Kind), Source: targetRecord.Source, SHA256: string(targetRecord.SHA256), Size: uint64(targetRecord.Size),
		Argv: append([]string(nil), targetRecord.Argv...), BuildTags: append([]string(nil), targetRecord.BuildTags...), Adapters: cloneAdapters(targetRecord.Adapters), Compatibility: cloneCompatibility(targetRecord.Compatibility), BuildInfo: cloneBuildInfo(targetRecord.BuildInfo),
		GoVersion: document.Campaign.Toolchain.GoVersion, BuildKey: document.Campaign.Toolchain.BuildKey, TargetGOOS: document.Campaign.Toolchain.TargetGOOS, TargetGOARCH: document.Campaign.Toolchain.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(targetRecord.CapabilityMode), CapabilityManifest: target.CapabilityManifestFromRecord(targetRecord.CapabilityManifest),
	}
	if err := verifyPrepared(prepared); err != nil {
		return openedCampaignPlan{}, err
	}
	return openedCampaignPlan{path: path, identity: record.HashBytes(contents), plan: document.Campaign, mounts: document.Mounts, prepared: prepared}, nil
}

func encodePortableCampaignPlan(document portableCampaignPlan) ([]byte, error) {
	if !portableCampaignPlanStringsValid(document) {
		return nil, errors.New("campaign plan contains strings that are not valid UTF-8")
	}
	return json.Marshal(document)
}

func portableCampaignPlanStringsValid(document portableCampaignPlan) bool {
	plan := document.Campaign
	values := []string{
		document.Schema, document.Storage, document.Mapping,
		plan.Schema, string(plan.PlanSHA256), plan.Strategy, plan.Selection,
		string(plan.ChoiceExplorationImplementationSHA256), string(plan.SimulationExplorationImplementationSHA256),
		plan.OnFailure, plan.RunnerBuild,
		plan.Toolchain.GoVersion, plan.Toolchain.BuildKey, plan.Toolchain.TargetGOOS, plan.Toolchain.TargetGOARCH,
		plan.Prepared.Path,
		plan.IOProfile.Name, string(plan.IOProfile.ImplementationSHA256), string(plan.IOProfile.InventorySHA256),
		plan.Coverage, plan.KeepSuccesses,
	}
	values = append(values, plan.IOROMounts...)
	values = append(values, plan.RequiredSemanticProbes...)
	for _, entry := range plan.Environment {
		values = append(values, entry.Name, entry.Value)
	}
	if profile := plan.ChoiceProfile; profile != nil {
		values = append(values, profile.Name, string(profile.ImplementationSHA256))
	}
	if guidance := plan.Guidance; guidance != nil {
		values = append(values, guidance.Corpus, string(guidance.SnapshotSHA256), guidance.RequestedSelection)
	}
	if mounts := document.Mounts; mounts != nil {
		values = append(values, mounts.Schema, string(mounts.SHA256))
		values = append(values, mounts.Mappings...)
	}
	appendPortableTargetStrings(&values, plan.Prepared.Target)
	for _, value := range values {
		if !utf8.ValidString(value) {
			return false
		}
	}
	return true
}

func appendPortableTargetStrings(values *[]string, targetRecord record.Target) {
	*values = append(*values,
		targetRecord.Kind, targetRecord.Source, targetRecord.File, string(targetRecord.SHA256), targetRecord.CapabilityMode,
		targetRecord.BuildInfo.GoVersion, targetRecord.BuildInfo.Path, targetRecord.BuildInfo.MainModule,
	)
	*values = append(*values, targetRecord.Argv...)
	*values = append(*values, targetRecord.BuildTags...)
	for _, adapter := range targetRecord.Adapters {
		*values = append(*values, adapter.Module, adapter.Version, adapter.Sum)
	}
	for _, pack := range targetRecord.Compatibility {
		*values = append(*values, pack.ID, string(pack.SHA256))
	}
	for _, setting := range targetRecord.BuildInfo.Settings {
		*values = append(*values, setting.Key, setting.Value)
	}
	if backendRecord := targetRecord.Backend; backendRecord != nil {
		*values = append(*values,
			backendRecord.Name, backendRecord.ReplayMode,
			backendRecord.Provenance.Schema, backendRecord.Provenance.File, string(backendRecord.Provenance.SHA256),
		)
		if evidence := backendRecord.Evidence; evidence != nil {
			*values = append(*values, evidence.Schema, evidence.File, string(evidence.SHA256))
		}
	}
	if manifest := targetRecord.CapabilityManifest; manifest != nil {
		*values = append(*values,
			manifest.Schema, manifest.File, string(manifest.SHA256), string(manifest.ProducerImplementationSHA256),
			string(manifest.GuardImplementationSHA256), string(manifest.CapabilityUniverseSHA256),
		)
	}
}

// UnsupportedCampaignPlanError refuses a portable plan whose lineage is not
// the one this build writes: a plan an earlier Gomad wrote for another
// campaign storage, or one that names no storage. It is never decoded or
// migrated.
type UnsupportedCampaignPlanError struct {
	Schema  string
	Storage string
}

func (err *UnsupportedCampaignPlanError) Error() string {
	return fmt.Sprintf("unsupported campaign plan format %q for storage %q: this Gomad opens only %q plans for storage %q, and no migration exists", err.Schema, err.Storage, campaignPlanSchema, campaignPlanStorage)
}

// checkCampaignPlanLineage reads only the lineage a plan names, its schema
// and storage, so a plan of another lineage is refused by name before its
// current document is decoded. A document that names no schema is left to
// the strict decoder, which refuses it as malformed or invalid.
func checkCampaignPlanLineage(contents []byte) error {
	var lineage struct {
		Schema  *string `json:"schema"`
		Storage string  `json:"storage"`
	}
	if err := json.NewDecoder(bytes.NewReader(contents)).Decode(&lineage); err != nil || lineage.Schema == nil {
		return nil
	}
	if *lineage.Schema != campaignPlanSchema || lineage.Storage != campaignPlanStorage {
		return &UnsupportedCampaignPlanError{Schema: *lineage.Schema, Storage: lineage.Storage}
	}
	return nil
}

func validateCampaignPlanMountIdentity(plan campaign.CampaignPlan, identity *campaignPlanMountIdentity, bundle string) error {
	if identity == nil {
		if len(plan.IOROMounts) != 0 {
			return errors.New("campaign plan mount identity is missing")
		}
		return nil
	}
	mappings, err := captured.ParseMappings(plan.IOROMounts, bundle)
	if err != nil {
		return err
	}
	limits, err := captured.DecodeLimits(deterministicCapturedInputLimits(plan.IOROMountLimits))
	if err != nil {
		return err
	}
	targets := make([]string, len(mappings))
	for index, mapping := range mappings {
		targets[index] = mapping.Target
	}
	if identity.Schema != "gomad3.io-read-only-mounts/v1" || identity.SHA256 == "" || identity.Bytes == 0 || identity.Entries < record.Uint64String(len(mappings)) || !slices.Equal(identity.Mappings, targets) || identity.Limits != captured.CapturedInputLimitsOf(limits) {
		return errors.New("campaign plan mount identity is invalid")
	}
	if _, err := record.ParseSHA256(string(identity.SHA256)); err != nil {
		return fmt.Errorf("campaign plan mount identity: %w", err)
	}
	for index, mapping := range mappings {
		if plan.IOROMounts[index] != campaignPlanMountValue(index, mapping.Target) {
			return errors.New("campaign plan mount mapping is not portable")
		}
	}
	actual, _, err := captureCampaignPlanMounts(mappings, limits)
	if err != nil {
		return fmt.Errorf("verify campaign plan mount identity: %w", err)
	}
	if !equalCampaignPlanMountIdentity(actual, identity) {
		return errors.New("campaign plan read-only mount identity changed")
	}
	return nil
}

func equalCampaignPlanMountIdentity(left, right *campaignPlanMountIdentity) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Schema == right.Schema && left.SHA256 == right.SHA256 && left.Bytes == right.Bytes && left.Entries == right.Entries && left.TotalBytes == right.TotalBytes && left.Limits == right.Limits && slices.Equal(left.Mappings, right.Mappings)
}

func publishPlanFile(path string, contents []byte) (retErr error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".campaign-plan-")
	if err != nil {
		return fmt.Errorf("create campaign plan staging file: %w", err)
	}
	temporaryPath := temporary.Name()
	linked := false
	defer func() {
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			retErr = errors.Join(retErr, removeErr)
		}
		if retErr != nil && linked {
			if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				retErr = errors.Join(retErr, removeErr)
			} else {
				retErr = errors.Join(retErr, hostfs.SyncDirectory(filepath.Dir(path)))
			}
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if _, err := temporary.Write(contents); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Sync(); err != nil {
		return errors.Join(err, temporary.Close())
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(temporaryPath, path); err != nil {
		return fmt.Errorf("publish campaign plan: %w", err)
	}
	linked = true
	if err := os.Remove(temporaryPath); err != nil {
		return err
	}
	return hostfs.SyncDirectory(filepath.Dir(path))
}

func syncRegularFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}
