package capabilitypolicy

import (
	"slices"
	"sort"
	"strings"
)

// The first-party simulation harness bridges into the patched runtime through
// these exact sources; each is pinned by name, digest and directives.
const simulationImportPath = "go.temporal.io/server/tools/gomad/internal/model/cluster"

var builtInSimulationLinknames = map[string]Source{
	"runtime_domain.go": {
		Name: "runtime_domain.go", SHA256: "sha256:2fff74d148dd42b4d4ade4a17d87cff549c712c74782c3860ff593d9d76edf20",
		LinknameDirectives: []string{"gomadSimulationEnabled runtime.gomadDeterministicEnabled", "gomadSimulationBegin internal/gomadsim.Begin", "gomadSimulationRegister internal/gomadsim.Register", "gomadSimulationEnter internal/gomadsim.Enter", "gomadSimulationLeave internal/gomadsim.Leave", "gomadSimulationRevoke internal/gomadsim.Revoke", "gomadSimulationFinish internal/gomadsim.Finish"},
	},
	"runtime_network.go": {
		Name: "runtime_network.go", SHA256: "sha256:5fe4575e73442191b765b698a58d58cbcd2389bcf7663dd2bfc5f71767adb701",
		LinknameDirectives: []string{"gomadNetworkBegin internal/gomadio.BeginSimulation", "gomadNetworkPartition internal/gomadio.PartitionSimulation", "gomadNetworkHeal internal/gomadio.HealSimulation", "gomadNetworkDelay internal/gomadio.DelaySimulation", "gomadNetworkGroup internal/gomadio.ChangeSimulationGroup", "gomadNetworkRevoke internal/gomadio.RevokeSimulation", "gomadNetworkFinish internal/gomadio.FinishSimulation"},
	},
	"runtime_process.go": {
		Name: "runtime_process.go", SHA256: "sha256:1b90fbe867259aee59287d194d56486daa62612542edad8139d5bb6d211a8d2a",
		LinknameDirectives: []string{"gomadProcessAvailable internal/gomadsim.ProcessAvailable", "gomadProcessRole internal/gomadsim.ProcessRole", "gomadProcessBootstrap internal/gomadsim.ProcessBootstrap", "gomadProcessExchange internal/gomadsim.ProcessExchange", "gomadProcessWaitStop internal/gomadsim.ProcessWaitStop", "gomadProcessServeModel internal/gomadsim.ProcessServeModel"},
	},
	"runtime_process_model.go": {
		Name: "runtime_process_model.go", SHA256: "sha256:844e64ff23890af0ef6804ccdbef7363ec86df840c6efc3ca810777d47c01bfb",
		LinknameDirectives: []string{"gomadProcessNetworkOperation internal/gomadio.ProcessSimulationNetworkOperation", "gomadProcessVolumeOperation internal/gomadfs.ProcessSimulationVolumeOperation"},
	},
	"runtime_time_toolchain.go": {
		Name: "runtime_time_toolchain.go", SHA256: "sha256:56b58c40f4aa543e76b848a82a0e2292dce1628ad240a069f6e1071a7215bd76",
		LinknameDirectives: []string{"gomadSimulationTimeAdvance runtime.gomadSimulationTimeAdvance", "gomadSimulationTimeCurrent runtime.gomadSimulationTimeCurrent", "gomadSimulationTimeTakeArrivals runtime.gomadSimulationTimeTakeArrivals"},
	},
	"runtime_volume.go": {
		Name: "runtime_volume.go", SHA256: "sha256:152dfdebe740e5273a4d8ac0273d5fc29d214f1a8f34ca1041429189a9f57098",
		LinknameDirectives: []string{"gomadVolumeBegin internal/gomadfs.BeginSimulationVolumes", "gomadInitializeVolumeFilesystem os.gomadInitializeSimulationFilesystem", "gomadVolumeRegister internal/gomadfs.RegisterSimulationVolumes", "gomadVolumeRevoke internal/gomadfs.RevokeSimulationVolumes", "gomadVolumeEnumerate internal/gomadfs.EnumerateSimulationVolume", "gomadVolumeFinish internal/gomadfs.FinishSimulationVolumes"},
	},
}

// AllowsSimulationBridge reports whether source is one of the pinned bridge
// sources of the first-party simulation harness in the main module.
func AllowsSimulationBridge(pkg Package, source Source) bool {
	exactPackage := pkg.ImportPath == simulationImportPath || pkg.ForTest == simulationImportPath && strings.HasPrefix(pkg.ImportPath, simulationImportPath+" [")
	if !exactPackage || pkg.Policy.Module.Path != "go.temporal.io/server/tools/gomad" || !pkg.MainModule || pkg.Policy.Module.Replaced || source.MalformedLinkname {
		return false
	}
	want, ok := builtInSimulationLinknames[source.Name]
	return ok && source.SHA256 == want.SHA256 && slices.Equal(source.LinknameDirectives, want.LinknameDirectives)
}

// SimulationBridgeSources returns the pinned bridge source names in order.
func SimulationBridgeSources() []string {
	names := make([]string, 0, len(builtInSimulationLinknames))
	for name := range builtInSimulationLinknames {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
