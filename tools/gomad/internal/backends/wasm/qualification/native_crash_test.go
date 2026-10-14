package qualification

import (
	"testing"

	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/model/volume"
)

func TestNativeCrashCaptureIsBoundedDetachedAndKeepsDivergence(t *testing.T) {
	handle := gomad3sim.NodeHandle{Node: "sqlite", Incarnation: 2}
	enumeration := volume.CrashEnumeration{Complete: true, States: []volume.CrashState{{Volume: "sqlite", Identity: "native", PendingSHA256: "pending", Entries: []volume.CrashEntry{{Path: "/marker", Data: []byte("old")}}}}}
	result := Result{Manifest: Manifest{Workload: "sqlite", Repeat: 2}}
	incarnations := []simulation.IncarnationResult{{Handle: handle, Result: wasi.Result{StorageCrashEnumeration: map[string]volume.CrashEnumeration{"sqlite": enumeration}}}}
	for _, phase := range []string{"execute", "replay"} {
		for iteration := uint64(1); iteration <= 2; iteration++ {
			if err := captureNativeCrashCatalogues(&result, incarnations, phase, iteration); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(result.NativeCrashCatalogues) != 1 || len(result.NativeCrashCatalogues[0].Observations) != 4 {
		t.Fatal("matched native catalogue was duplicated or observations lost")
	}
	if _, err := MatchedNativeCrashEnumeration(result, handle, "sqlite"); err != nil {
		t.Fatal(err)
	}
	altered := result
	altered.NativeCrashCatalogues = append([]NativeCrashCatalogue{}, result.NativeCrashCatalogues...)
	altered.NativeCrashCatalogues[0].Observations = append([]NativeCrashObservation{}, result.NativeCrashCatalogues[0].Observations...)
	altered.NativeCrashCatalogues[0].Observations[0].Iteration = 3
	if err := validateNativeCrashCatalogues(altered); err == nil {
		t.Fatal("foreign repeat admitted")
	}
	altered.NativeCrashCatalogues[0].Observations[0] = altered.NativeCrashCatalogues[0].Observations[1]
	if err := validateNativeCrashCatalogues(altered); err == nil {
		t.Fatal("duplicate observation admitted")
	}
	altered.NativeCrashCatalogues[0].SHA256 = "foreign"
	if err := validateNativeCrashCatalogues(altered); err == nil {
		t.Fatal("native catalogue digest mutation admitted")
	}
	enumeration.States[0].Entries[0].Data[0] = 'n'
	if string(result.NativeCrashCatalogues[0].Enumeration.States[0].Entries[0].Data) != "old" {
		t.Fatal("native state aliases caller bytes")
	}
	divergent := Result{Manifest: result.Manifest}
	if err := captureNativeCrashCatalogues(&divergent, incarnations, "execute", 1); err != nil {
		t.Fatal(err)
	}
	incarnations[0].Result.StorageCrashEnumeration["sqlite"].States[0].Entries[0].Data[0] = 'x'
	if err := captureNativeCrashCatalogues(&divergent, incarnations, "execute", 2); err != nil {
		t.Fatal(err)
	}
	if len(divergent.NativeCrashCatalogues) != 2 {
		t.Fatal("divergent native catalogue discarded")
	}
	if _, err := MatchedNativeCrashEnumeration(divergent, handle, "sqlite"); err == nil {
		t.Fatal("divergent/incomplete observations passed")
	}
	tooLarge := volume.CrashEnumeration{States: make([]volume.CrashState, 33)}
	incarnations[0].Result.StorageCrashEnumeration["sqlite"] = tooLarge
	if err := captureNativeCrashCatalogues(&result, incarnations, "execute", 1); err == nil {
		t.Fatal("oversized native state count cloned")
	}
	tooLarge.States = []volume.CrashState{{Entries: []volume.CrashEntry{{Data: make([]byte, 8<<20)}}}}
	incarnations[0].Result.StorageCrashEnumeration["sqlite"] = tooLarge
	if err := captureNativeCrashCatalogues(&result, incarnations, "execute", 1); err == nil {
		t.Fatal("oversized native entry payload cloned")
	}
}
