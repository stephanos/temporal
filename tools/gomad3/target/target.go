package target

import (
	"fmt"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
)

type Kind string

const (
	KindExec   Kind = "exec"
	KindGoRun  Kind = "go-run"
	KindGoTest Kind = "go-test"
)

type CapabilityMode string

const (
	CapabilityModeClosure CapabilityMode = "closure"
	CapabilityModeLinked  CapabilityMode = "linked"
	CapabilityModeGuarded CapabilityMode = "guarded"
)

type Spec struct {
	SoftAssertionReportOnly bool `json:",omitempty"`
	Backend             string `json:",omitempty"`
	Kind                Kind
	Source              string
	Provenance          string
	Args                []string
	BuildTags           []string
	WorkingDir          string
	PreparationRoot     string
	ToolchainRoot       string
	BuildOverlay        string
	BuildModFile        string
	AdapterReplacements []AdapterReplacement
	CapabilityMode      CapabilityMode
}

type ModuleIdentity struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}

type AdapterReplacement struct {
	Original                         ModuleIdentity
	ReplacementPath                  string
	PreparedPackage                  string
	ProfileName                      string
	ProfileImplementationSHA256      string
	Adapter                          ModuleIdentity
	OriginalSourceInventorySHA256    string
	ReplacementSourceInventorySHA256 string
	PreparedSourceSetSHA256          string
}

type ToolchainIdentity struct {
	GoVersion    string
	BuildKey     string
	TargetGOOS   string
	TargetGOARCH string
}

type Prepared struct {
	SoftAssertionReportOnly bool `json:",omitempty"`
	Backend            *record.BackendMetadata `json:",omitempty"`
	BackendPayloads    []BackendPayload        `json:",omitempty"`
	Path               string
	Kind               Kind
	Source             string
	SHA256             string
	Size               uint64
	Argv               []string
	BuildTags          []string
	Adapters           []record.TargetAdapter
	Compatibility      []record.CompatibilityPack
	BuildInfo          record.BuildInfo
	GoVersion          string
	BuildKey           string
	TargetGOOS         string
	TargetGOARCH       string
	CapabilityMode     CapabilityMode
	CapabilityManifest *CapabilityManifest
}

type CapabilityManifest struct {
	Schema                       string        `json:"schema"`
	SHA256                       record.SHA256 `json:"sha256"`
	Bytes                        uint64        `json:"bytes"`
	Facts                        uint64        `json:"facts"`
	ProducerImplementationSHA256 string        `json:"producer_implementation_sha256"`
	GuardImplementationSHA256    string        `json:"guard_implementation_sha256"`
	CapabilityUniverseSHA256     string        `json:"capability_universe_sha256"`
	Payload                      []byte        `json:"-"`
}

// Verify checks the executable's file identity. The preparing backend owns
// verification of its capability policy and embedded manifest.
func (prepared Prepared) Verify() error {
	hash, size, err := hostfs.HashExecutable(prepared.Path)
	if err != nil {
		return fmt.Errorf("verify prepared target: %w", err)
	}
	if hash != prepared.SHA256 || size != prepared.Size {
		return fmt.Errorf("prepared target changed after preparation")
	}
	return nil
}

func (prepared Prepared) RecordTarget() record.Target {
	buildInfo := prepared.BuildInfo
	buildInfo.Settings = append([]record.BuildSetting(nil), prepared.BuildInfo.Settings...)
	recorded := record.Target{
		SoftAssertionReportOnly: prepared.SoftAssertionReportOnly,
		Backend: record.CloneBackendMetadata(prepared.Backend),
		Kind:    string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
		Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...),
		Adapters: append([]record.TargetAdapter{}, prepared.Adapters...), Compatibility: append([]record.CompatibilityPack{}, prepared.Compatibility...), BuildInfo: buildInfo,
		CapabilityMode: string(prepared.CapabilityMode),
	}
	if recorded.CapabilityMode == "" {
		recorded.CapabilityMode = string(CapabilityModeClosure)
	}
	if manifest := prepared.CapabilityManifest; manifest != nil {
		recorded.CapabilityManifest = manifest.Record()
	}
	return recorded
}

func (manifest CapabilityManifest) Record() *record.TargetCapabilityManifest {
	return &record.TargetCapabilityManifest{
		Schema: manifest.Schema, File: "target-capabilities.json", SHA256: manifest.SHA256,
		Bytes: record.Uint64String(manifest.Bytes), Facts: record.Uint64String(manifest.Facts),
		ProducerImplementationSHA256: record.SHA256(manifest.ProducerImplementationSHA256),
		GuardImplementationSHA256:    record.SHA256(manifest.GuardImplementationSHA256),
		CapabilityUniverseSHA256:     record.SHA256(manifest.CapabilityUniverseSHA256),
	}
}

func CapabilityManifestFromRecord(manifest *record.TargetCapabilityManifest) *CapabilityManifest {
	if manifest == nil {
		return nil
	}
	return &CapabilityManifest{
		Schema: manifest.Schema, SHA256: manifest.SHA256, Bytes: uint64(manifest.Bytes), Facts: uint64(manifest.Facts),
		ProducerImplementationSHA256: string(manifest.ProducerImplementationSHA256),
		GuardImplementationSHA256:    string(manifest.GuardImplementationSHA256),
		CapabilityUniverseSHA256:     string(manifest.CapabilityUniverseSHA256),
	}
}

func (prepared Prepared) RecordToolchain() record.Toolchain {
	return record.Toolchain{
		GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey,
		TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH,
	}
}
