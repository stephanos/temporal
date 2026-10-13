package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	backend "go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
	"go.temporal.io/server/tools/gomad3/internal/host/fs"
	"go.temporal.io/server/tools/gomad3/internal/runner/internal/campaign"
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
	bundle := output + campaign.PortablePlanBundleSuffix
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
	mounts, portableMounts := campaign.CanonicalPlanMounts(mounts)
	mountIdentity, capturedMounts, err := campaign.CapturePlanMounts(mounts, config.IOROMountLimits)
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
	targetPath := filepath.Join(bundle, campaign.PortablePlanTargetFile)
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
	if err := campaign.MaterializePlanMounts(bundle, capturedMounts); err != nil {
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
	document := campaign.PortablePlan{Schema: campaign.PortablePlanSchema, Storage: campaign.PortablePlanStorage, Mapping: campaign.PortablePlanMapping, Mounts: mountIdentity, Campaign: plan}
	encoded, err := campaign.EncodePortablePlan(document)
	if err != nil {
		return CampaignPlanResult{}, fmt.Errorf("encode campaign plan: %w", err)
	}
	if len(encoded) > campaign.MaximumPortablePlanBytes {
		return CampaignPlanResult{}, fmt.Errorf("campaign plan exceeds its %d-byte capacity", campaign.MaximumPortablePlanBytes)
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
