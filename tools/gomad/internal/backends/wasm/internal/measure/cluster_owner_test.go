package measure

import (
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func TestClusterOwnerSplitReportsWaitingApartFromHostWork(t *testing.T) {
	work, waiting := clusterOwnerSplit(map[string]diagnosticSeries{
		"scheduler_handoff": {ExecuteExclusiveWallNanos: 5}, "clock": {ExecuteExclusiveWallNanos: 7},
		"handoff_wait": {ExecuteExclusiveWallNanos: 100}, "instrumentation": {ExecuteExclusiveWallNanos: 3},
	})
	if work != 12 || waiting != 103 {
		t.Fatalf("work=%d waiting=%d", work, waiting)
	}
	for _, name := range []string{"handoff_wait", "instrumentation"} {
		if !slices.Contains(diagnosticOwnerNames, name) {
			t.Fatalf("waiting owner %s outside the closed owner set", name)
		}
	}
}

func TestClusterOwnerRequestsBindEachRoleToItsAdmittedHelper(t *testing.T) {
	prepared := wasi.Request{HelperPath: "/diag", HelperSHA256: strings.Repeat("d", 64), ModulePath: "/m.wasm", ModuleSHA256: "m", Runtime: &wasi.RuntimeControl{Seed: 7}}
	helpers := map[string]diagnosticHelperAdmission{"/release": {Receipt: "/r.json", Features: []string{}}, "/diag": {Receipt: "/d.json", Features: []string{"diagnostic-profile"}}}
	release, err := clusterOwnerRequestFor(prepared, helpers, "/release", strings.Repeat("r", 64), []string{})
	if err != nil {
		t.Fatal(err)
	}
	if release.HelperPath != "/release" || release.HelperSHA256 != strings.Repeat("r", 64) || release.ModulePath != prepared.ModulePath || release.ModuleSHA256 != prepared.ModuleSHA256 {
		t.Fatalf("reference request not bound to the release helper: %+v", release)
	}
	release.Runtime.Seed = 19
	if prepared.Runtime.Seed != 7 || prepared.HelperPath != "/diag" {
		t.Fatal("reference request shares state with the prepared request")
	}
	if _, err := clusterOwnerRequestFor(prepared, helpers, "/diag", strings.Repeat("d", 64), []string{"diagnostic-profile"}); err != nil {
		t.Fatal(err)
	}
	for _, refused := range []struct {
		helper   string
		features []string
	}{{"/unadmitted", []string{}}, {"/release", []string{"diagnostic-profile"}}, {"/diag", []string{}}} {
		if _, err := clusterOwnerRequestFor(prepared, helpers, refused.helper, strings.Repeat("a", 64), refused.features); err == nil {
			t.Fatalf("accepted %s with features %v", refused.helper, refused.features)
		}
	}
	if _, err := clusterOwnerRequestFor(prepared, helpers, "/release", "short", []string{}); err == nil {
		t.Fatal("accepted a request without a helper digest")
	}
}
