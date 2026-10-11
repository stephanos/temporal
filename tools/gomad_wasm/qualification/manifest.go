// Package qualification binds WASM support and measured costs to retained
// reports produced by Gomad's existing qualification owner.
package qualification

import (
	"errors"
	"fmt"
	"path/filepath"

	"go.temporal.io/server/tools/gomad3/record"
)

const ManifestSchema = "gomad-wasm.qualification-manifest/v1"

// Manifest is a frozen comparison cell, not an assertion that it passed.
// RuntimeSHA256 binds either the stock runtime source inventory or the hooked
// runtime inventory. IntegratedSourceSHA256 is the candidate source inventory,
// never a historical commit substituted for current dirty sources.
type Manifest struct {
	Provisional                                           bool
	IntegratedSourceManifestPath                          string
	Schema, ID, Workload, Profile                         string
	Cell                                                  string
	IntegratedSourceSHA256, CompilerSHA256, RuntimeSHA256 record.SHA256
	HelperSHA256, EngineSHA256, ModelSHA256               record.SHA256
	HostConfigSHA256                                      record.SHA256
	WorkloadSourceSHA256                                  record.SHA256
	HostOS, HostArch                                      string
	Seed, Repeat, Fuel, MemoryBytes                       uint64
	Unsupported                                           []string
}

func (m Manifest) Validate() error {
	if m.Cell != "" && m.Cell != "success" && m.Cell != "canary" || !m.Provisional && m.Cell == "" {
		return errors.New("final qualification requires explicit success/canary cell")
	}
	if !m.Provisional && !filepath.IsAbs(m.IntegratedSourceManifestPath) {
		return errors.New("final WASM qualification requires conductor integrated-source manifest reference")
	}
	if m.Schema != ManifestSchema || m.ID == "" || m.HostOS == "" || m.HostArch == "" || m.Repeat < 2 || m.Repeat > 32 || m.Fuel == 0 || m.MemoryBytes == 0 {
		return errors.New("frozen WASM identity, limits and 2..32 repeats are required")
	}
	for _, digest := range []record.SHA256{m.IntegratedSourceSHA256, m.WorkloadSourceSHA256, m.CompilerSHA256, m.RuntimeSHA256, m.HelperSHA256, m.EngineSHA256, m.ModelSHA256, m.HostConfigSHA256} {
		if _, err := record.ParseSHA256(string(digest)); err != nil {
			return fmt.Errorf("unbound WASM qualification identity: %w", err)
		}
	}
	switch m.Workload {
	case "fixture", "sqlite", "onebox", "multi-instance":
	default:
		return errors.New("unsupported WASM qualification workload")
	}
	switch m.Profile {
	case "controlled", "cooperative":
	default:
		return fmt.Errorf("unsupported WASM profile %q: no admitted backend owner", m.Profile)
	}
	return nil
}

// Matched rejects limits/source/host mismatches before drawing cost conclusions.
// Compiler/runtime/helper/engine/model may differ: those are comparison inputs,
// explicitly bound above, rather than facts allowed to float unnoticed.
func Matched(a, b Manifest) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if err := b.Validate(); err != nil {
		return err
	}
	if a.Workload != b.Workload || a.IntegratedSourceSHA256 != b.IntegratedSourceSHA256 || a.WorkloadSourceSHA256 != b.WorkloadSourceSHA256 || a.HostOS != b.HostOS || a.HostArch != b.HostArch || a.Seed != b.Seed || a.Repeat != b.Repeat || a.Fuel != b.Fuel || a.MemoryBytes != b.MemoryBytes {
		return errors.New("WASM comparison cells are not matched")
	}
	return nil
}
