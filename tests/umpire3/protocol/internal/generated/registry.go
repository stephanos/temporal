// Package generated owns the embedded, reproducible protocol artifacts.
package generated

import (
	"embed"
	"fmt"
	"slices"
)

type Name string

const (
	Catalog                         Name = "catalog.json"
	CheckerCoverage                 Name = "checker-coverage.json"
	Composition                     Name = "composition.json"
	CoverageDenominator             Name = "coverage-denominator.json"
	DescriptorManifest              Name = "descriptor-manifest.json"
	FamilyDependencies              Name = "family-dependencies.json"
	MonitorPrograms                 Name = "monitor-programs.json"
	NexusExactMutationProofManifest Name = "nexus-exact-mutation-proof-manifest.json"
	NexusMutationProofManifest      Name = "nexus-mutation-rejection-proof-manifest.json"
	NexusProofManifest              Name = "nexus-proof-manifest.json"
	ParityLedger                    Name = "parity-ledger.json"
	TaskDeliveryMutatedTemporal     Name = "task-delivery-progress-mutated.temporal.json"
	TaskDeliveryTemporal            Name = "task-delivery-progress.temporal.json"
	UpdateProofManifest             Name = "update-proof-manifest.json"
)

//go:embed testdata/generated/*.json
var artifacts embed.FS

func Read(name Name) []byte {
	encoded, err := artifacts.ReadFile("testdata/generated/" + string(name))
	if err != nil {
		panic(fmt.Sprintf("read embedded Umpire3 artifact %q: %v", name, err))
	}
	return slices.Clone(encoded)
}
