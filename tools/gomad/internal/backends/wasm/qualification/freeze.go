package qualification

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

// Freeze obtains current source/runtime inventories from the existing backend
// preparation owner. It must precede matched comparisons. Target.PreparationRoot
// is explicitly supplied just as for backend.Prepare; no ambient checkout or
// historical proof is consulted. Its compiler resource costs remain in Run's
// result and compute-hour denominator, separate from helper execution samples.
func Freeze(ctx context.Context, spec Spec) (Spec, error) {
	spec.operatorPreparationBefore = operatorBaseline(spec.OperatorStart)
	if err := freezeSource(&spec); err != nil {
		return Spec{}, err
	}
	options := spec.Options
	previous := options.ObserveCompilerCommand
	options.ObserveCompilerCommand = func(metrics backend.CompilerCommandMetrics) {
		spec.initialCompilerCommands = append(spec.initialCompilerCommands, metrics)
		if previous != nil {
			previous(metrics)
		}
	}
	provider, err := backend.New(options)
	if err != nil {
		return Spec{}, err
	}
	started := time.Now()
	prepared, err := provider.Prepare(ctx, spec.Campaign.Target)
	spec.coldPreparationWallNanos = uint64(time.Since(started))
	if err != nil {
		return Spec{}, err
	}
	return freezePrepared(spec, provider, prepared)
}

func freezeSource(spec *Spec) error {
	if !spec.Manifest.Provisional && spec.OperatorStart == nil {
		return errors.New("final qualification requires operator baseline before setup")
	}
	if spec.Manifest.IntegratedSourceManifestPath != "" {
		encoded, err := hostfs.ReadBounded(spec.Manifest.IntegratedSourceManifestPath, 16<<20)
		if err != nil {
			return err
		}
		if record.HashBytes(encoded) != spec.Manifest.IntegratedSourceSHA256 {
			return errors.New("integrated source manifest digest changed")
		}
		if err := validateSourceManifest(encoded, true); err != nil {
			return err
		}
		spec.integratedSourceManifest = encoded
	} else if !spec.Manifest.Provisional {
		return errors.New("final WASM qualification requires source manifest")
	}
	return nil
}

func freezePrepared(spec Spec, provider *backend.Provider, prepared target.Prepared) (Spec, error) {
	options := spec.Options
	identity, err := provider.QualificationIdentity(prepared)
	if err != nil {
		return Spec{}, err
	}
	engine, err := json.Marshal(options.Engine)
	if err != nil {
		return Spec{}, err
	}
	config, err := json.Marshal(options.Config)
	if err != nil {
		return Spec{}, err
	}
	manifest := spec.Manifest
	manifest.Schema = ManifestSchema
	manifest.WorkloadSourceSHA256, manifest.RuntimeSHA256 = identity.WorkloadSourceSHA256, identity.RuntimeSHA256
	manifest.CompilerSHA256, manifest.HelperSHA256, manifest.ModelSHA256 = record.SHA256(options.CompilerSHA256), record.SHA256(options.HelperSHA256), record.SHA256(options.ModelSHA256)
	manifest.EngineSHA256, manifest.HostConfigSHA256 = record.HashBytes(engine), record.HashBytes(config)
	manifest.HostOS, manifest.HostArch = options.Engine.HostOS, options.Engine.HostArch
	manifest.Fuel, manifest.MemoryBytes = options.Fuel, options.MemoryBytes
	manifest.Profile = "controlled"
	if options.Config.Profile == wasi.CooperativeProfile {
		manifest.Profile = "cooperative"
	}
	for _, binding := range []struct{ declared, observed record.SHA256 }{{spec.Manifest.WorkloadSourceSHA256, manifest.WorkloadSourceSHA256}, {spec.Manifest.RuntimeSHA256, manifest.RuntimeSHA256}, {spec.Manifest.CompilerSHA256, manifest.CompilerSHA256}, {spec.Manifest.HelperSHA256, manifest.HelperSHA256}, {spec.Manifest.ModelSHA256, manifest.ModelSHA256}, {spec.Manifest.EngineSHA256, manifest.EngineSHA256}, {spec.Manifest.HostConfigSHA256, manifest.HostConfigSHA256}} {
		if binding.declared != "" && binding.declared != binding.observed {
			return Spec{}, errors.New("declared WASM qualification identity changed during freeze")
		}
	}
	if spec.Manifest.Profile != "" && spec.Manifest.Profile != manifest.Profile || spec.Manifest.HostOS != "" && spec.Manifest.HostOS != manifest.HostOS || spec.Manifest.HostArch != "" && spec.Manifest.HostArch != manifest.HostArch || spec.Manifest.Fuel != 0 && spec.Manifest.Fuel != manifest.Fuel || spec.Manifest.MemoryBytes != 0 && spec.Manifest.MemoryBytes != manifest.MemoryBytes {
		return Spec{}, errors.New("declared WASM qualification profile or limits changed during freeze")
	}
	if err := manifest.Validate(); err != nil {
		return Spec{}, err
	}
	spec.Manifest = manifest
	spec.preparedTarget = prepared
	spec.operatorPreparationAfter = hostexec.CurrentProcessResourceUsage()
	return spec, nil
}

// PreparedTarget returns a detached genuine backend preparation, not a custom
// producer projection. The executable still has to match its retained identity.
func PreparedTarget(spec Spec) (target.Prepared, error) {
	if spec.preparedTarget.Backend == nil {
		return target.Prepared{}, errors.New("observed frozen preparation required")
	}
	if err := spec.preparedTarget.Verify(); err != nil {
		return target.Prepared{}, err
	}
	return spec.preparedTarget.CloneBackend(), nil
}

// FreezeRetainedCluster shares one genuine measured preparation between cells.
// The private observations only originate in Freeze, never caller JSON. Current
// source/runtime/config identities are verified again; no compiler is executed.
func FreezeRetainedCluster(spec, preparation Spec) (Spec, error) {
	prepared, err := PreparedTarget(preparation)
	if err != nil {
		return Spec{}, err
	}
	if preparation.coldPreparationWallNanos == 0 || len(preparation.initialCompilerCommands) == 0 {
		return Spec{}, errors.New("actual shared preparation resource observations required")
	}
	for _, command := range preparation.initialCompilerCommands {
		if command.ResourceUsage == nil || command.WallElapsedNanos == 0 || command.ResourceUsage.PeakRSSBytes == 0 {
			return Spec{}, errors.New("shared compiler resource observations unavailable")
		}
	}
	spec.operatorPreparationBefore = operatorBaseline(spec.OperatorStart)
	if err := freezeSource(&spec); err != nil {
		return Spec{}, err
	}
	if spec.Manifest.IntegratedSourceSHA256 != preparation.Manifest.IntegratedSourceSHA256 {
		return Spec{}, errors.New("shared preparation integrated candidate differs")
	}
	provider, err := backend.New(spec.Options)
	if err != nil {
		return Spec{}, err
	}
	spec.coldPreparationWallNanos = preparation.coldPreparationWallNanos
	spec.preparationReused = true
	spec.initialCompilerCommands = append([]backend.CompilerCommandMetrics{}, preparation.initialCompilerCommands...)
	for i := range spec.initialCompilerCommands {
		spec.initialCompilerCommands[i].Arguments = append([]string{}, spec.initialCompilerCommands[i].Arguments...)
		usage := *spec.initialCompilerCommands[i].ResourceUsage
		spec.initialCompilerCommands[i].ResourceUsage = &usage
	}
	return freezePrepared(spec, provider, prepared)
}

func operatorBaseline(start *hostexec.ResourceUsage) *hostexec.ResourceUsage {
	if start == nil {
		return hostexec.CurrentProcessResourceUsage()
	}
	copy := *start
	return &copy
}
