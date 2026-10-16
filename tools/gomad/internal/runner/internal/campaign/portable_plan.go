package campaign

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	captured "go.temporal.io/server/tools/gomad/internal/evidence/artifact/capture"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	"go.temporal.io/server/tools/gomad/internal/model/strictjson"
)

const (
	// PortablePlanSchema is the portable plan lineage this build writes and
	// the only one it opens. Its embedded execution limits are the campaign
	// log's: the per-execution payload bound, the outcome count and byte
	// bounds and the partial executions, under PortablePlanStorage.
	// Plans of the earlier gomad.campaign-plan/v1 lineage bound a segmented
	// journal and are refused by name.
	PortablePlanSchema = "gomad.portable-campaign-plan/v1"
	// PortablePlanStorage is the campaign storage a plan's shards write.
	PortablePlanStorage      = CampaignLogFormat
	PortablePlanMapping      = "ordinal-modulo/v1"
	PortablePlanBundleSuffix = ".bundle"
	PortablePlanTargetFile   = "target"
	MaximumPortablePlanBytes = 1 << 20
)

type PortablePlan struct {
	Schema   string             `json:"schema"`
	Storage  string             `json:"storage"`
	Mapping  string             `json:"mapping"`
	Mounts   *PlanMountIdentity `json:"mounts,omitempty"`
	Campaign CampaignPlan       `json:"campaign"`
}

type PlanMountIdentity struct {
	Schema     string                       `json:"schema"`
	SHA256     record.SHA256                `json:"sha256"`
	Bytes      record.Uint64String          `json:"bytes"`
	Entries    record.Uint64String          `json:"entries"`
	TotalBytes record.Uint64String          `json:"total_bytes"`
	Mappings   []string                     `json:"mappings"`
	Limits     captured.CapturedInputLimits `json:"limits"`
}

type OpenedPortablePlan struct {
	Path     string
	Identity record.SHA256
	Plan     CampaignPlan
	Mounts   *PlanMountIdentity
	Prepared target.Prepared
}

func OpenPortablePlan(path string) (_ OpenedPortablePlan, retErr error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return OpenedPortablePlan{}, fmt.Errorf("resolve campaign plan path: %w", err)
	}
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return OpenedPortablePlan{}, fmt.Errorf("open campaign plan: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if info.Mode().Perm() != 0o600 || info.Size() > MaximumPortablePlanBytes {
		return OpenedPortablePlan{}, errors.New("campaign plan file mode or size is invalid")
	}
	contents, err := io.ReadAll(io.LimitReader(file, MaximumPortablePlanBytes+1))
	if err != nil {
		return OpenedPortablePlan{}, fmt.Errorf("read campaign plan: %w", err)
	}
	if len(contents) > MaximumPortablePlanBytes {
		return OpenedPortablePlan{}, errors.New("campaign plan exceeds its byte capacity")
	}
	if err := checkCampaignPlanLineage(contents); err != nil {
		return OpenedPortablePlan{}, err
	}
	var document PortablePlan
	if err := strictjson.Decode(contents, &document); err != nil {
		return OpenedPortablePlan{}, fmt.Errorf("decode campaign plan: %w", err)
	}
	if document.Schema != PortablePlanSchema || document.Storage != PortablePlanStorage || document.Mapping != PortablePlanMapping || document.Campaign.Strategy != string("seed") || document.Campaign.OnFailure != string("all") || document.Campaign.PlanSHA256 != "" || document.Campaign.Shard != nil {
		return OpenedPortablePlan{}, errors.New("campaign plan protocol identity is invalid")
	}
	if err := ValidateCampaignPlan(document.Campaign); err != nil {
		return OpenedPortablePlan{}, err
	}
	journal, err := DeriveExecutionJournalPlan(document.Campaign.Strategy, uint64(document.Campaign.SelectionCount), uint64(document.Campaign.MaxExecutions), uint64(document.Campaign.Parallel))
	if err != nil || *document.Campaign.Journal != journal {
		return OpenedPortablePlan{}, errors.Join(errors.New("campaign plan execution limits are not the campaign log limits"), err)
	}
	if (document.Mounts == nil) != (len(document.Campaign.IOROMounts) == 0) {
		return OpenedPortablePlan{}, errors.New("campaign plan mount identity is incomplete")
	}
	bundle := path + PortablePlanBundleSuffix
	bundleInfo, err := os.Lstat(bundle)
	if err != nil || !bundleInfo.IsDir() || bundleInfo.Mode()&os.ModeSymlink != 0 || bundleInfo.Mode().Perm() != 0o700 {
		return OpenedPortablePlan{}, errors.Join(errors.New("campaign plan bundle is not a private directory"), err)
	}
	if err := validateCampaignPlanBundleInventory(bundle, len(document.Campaign.IOROMounts)); err != nil {
		return OpenedPortablePlan{}, err
	}
	if err := validateCampaignPlanMountIdentity(document.Campaign, document.Mounts, bundle); err != nil {
		return OpenedPortablePlan{}, err
	}
	targetPath := filepath.Join(bundle, PortablePlanTargetFile)
	targetFile, targetInfo, err := hostfs.OpenPath(targetPath)
	if err != nil {
		return OpenedPortablePlan{}, fmt.Errorf("open campaign plan target: %w", err)
	}
	if closeErr := targetFile.Close(); closeErr != nil {
		return OpenedPortablePlan{}, closeErr
	}
	if targetInfo.Mode().Perm() != 0o500 {
		return OpenedPortablePlan{}, errors.New("campaign plan target mode is invalid")
	}
	targetRecord := document.Campaign.Prepared.Target
	prepared := target.Prepared{
		Path: targetPath, Kind: target.Kind(targetRecord.Kind), Source: targetRecord.Source, SHA256: string(targetRecord.SHA256), Size: uint64(targetRecord.Size),
		Argv: append([]string(nil), targetRecord.Argv...), BuildTags: append([]string(nil), targetRecord.BuildTags...), Adapters: append([]record.TargetAdapter{}, targetRecord.Adapters...), Compatibility: append([]record.CompatibilityPack{}, targetRecord.Compatibility...), BuildInfo: clonePlanBuildInfo(targetRecord.BuildInfo),
		GoVersion: document.Campaign.Toolchain.GoVersion, BuildKey: document.Campaign.Toolchain.BuildKey, TargetGOOS: document.Campaign.Toolchain.TargetGOOS, TargetGOARCH: document.Campaign.Toolchain.TargetGOARCH,
		CapabilityMode: target.CapabilityMode(targetRecord.CapabilityMode), CapabilityManifest: target.CapabilityManifestFromRecord(targetRecord.CapabilityManifest),
	}
	if err := prepared.Verify(); err != nil {
		return OpenedPortablePlan{}, err
	}
	return OpenedPortablePlan{Path: path, Identity: record.HashBytes(contents), Plan: document.Campaign, Mounts: document.Mounts, Prepared: prepared}, nil
}

func EncodePortablePlan(document PortablePlan) ([]byte, error) {
	if !portableCampaignPlanStringsValid(document) {
		return nil, errors.New("campaign plan contains strings that are not valid UTF-8")
	}
	return json.Marshal(document)
}

func portableCampaignPlanStringsValid(document PortablePlan) bool {
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
	return fmt.Sprintf("unsupported campaign plan format %q for storage %q: this Gomad opens only %q plans for storage %q, and no migration exists", err.Schema, err.Storage, PortablePlanSchema, PortablePlanStorage)
}

// checkCampaignPlanLineage reads only the lineage a plan names, its schema
// and storage, so a plan of another lineage is refused by name before its
// current document is decoded. A document that names no schema is left to
// typed decoding and the subsequent protocol-identity validation.
func checkCampaignPlanLineage(contents []byte) error {
	var lineage struct {
		Schema  *string `json:"schema"`
		Storage string  `json:"storage"`
	}
	if err := json.NewDecoder(bytes.NewReader(contents)).Decode(&lineage); err != nil || lineage.Schema == nil {
		return nil
	}
	if *lineage.Schema != PortablePlanSchema || lineage.Storage != PortablePlanStorage {
		return &UnsupportedCampaignPlanError{Schema: *lineage.Schema, Storage: lineage.Storage}
	}
	return nil
}

func validateCampaignPlanMountIdentity(plan CampaignPlan, identity *PlanMountIdentity, bundle string) error {
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
	limits, err := captured.DecodeLimits(captured.CapturedInputLimits{
		PathBytes: uint64(plan.IOROMountLimits.PathBytes), Requests: uint64(plan.IOROMountLimits.Requests), Files: uint64(plan.IOROMountLimits.Files),
		DirectoryEntries: uint64(plan.IOROMountLimits.DirectoryEntries), SingleFileBytes: uint64(plan.IOROMountLimits.SingleFileBytes), TotalBytes: uint64(plan.IOROMountLimits.TotalBytes),
	})
	if err != nil {
		return err
	}
	targets := make([]string, len(mappings))
	for index, mapping := range mappings {
		targets[index] = mapping.Target
	}
	if identity.Schema != "gomad.io-read-only-mounts/v1" || identity.SHA256 == "" || identity.Bytes == 0 || identity.Entries < record.Uint64String(len(mappings)) || !slices.Equal(identity.Mappings, targets) || identity.Limits != captured.CapturedInputLimitsOf(limits) {
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
	actual, _, err := CapturePlanMounts(mappings, limits)
	if err != nil {
		return fmt.Errorf("verify campaign plan mount identity: %w", err)
	}
	if !EqualPlanMountIdentity(actual, identity) {
		return errors.New("campaign plan read-only mount identity changed")
	}
	return nil
}

func EqualPlanMountIdentity(left, right *PlanMountIdentity) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Schema == right.Schema && left.SHA256 == right.SHA256 && left.Bytes == right.Bytes && left.Entries == right.Entries && left.TotalBytes == right.TotalBytes && left.Limits == right.Limits && slices.Equal(left.Mappings, right.Mappings)
}

func clonePlanBuildInfo(value record.BuildInfo) record.BuildInfo {
	value.Settings = append([]record.BuildSetting(nil), value.Settings...)
	return value
}
