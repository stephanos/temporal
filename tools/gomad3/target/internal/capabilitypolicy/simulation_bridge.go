package capabilitypolicy

import (
	"slices"
	"sort"
	"strings"
)

// The first-party simulation harness bridges into the patched runtime through
// these exact sources; each is pinned by name, digest and directives.
const simulationImportPath = "go.temporal.io/server/tools/gomad3sim"

var builtInSimulationLinknames = map[string]Source{
	"runtime_domain.go": {
		Name: "runtime_domain.go", SHA256: "sha256:67364238aab705541b2db0db17ad5388949dc9db988c85e548b5db288ad3b74f",
		LinknameDirectives: []string{"gomadSimulationEnabled runtime.gomadDeterministicEnabled", "gomadSimulationBegin internal/gomadsim.Begin", "gomadSimulationRegister internal/gomadsim.Register", "gomadSimulationEnter internal/gomadsim.Enter", "gomadSimulationLeave internal/gomadsim.Leave", "gomadSimulationRevoke internal/gomadsim.Revoke", "gomadSimulationFinish internal/gomadsim.Finish"},
	},
	"runtime_network.go": {
		Name: "runtime_network.go", SHA256: "sha256:9e90c09d6ff0d6d2ad576453a0c018c12336a08981eceabc20a6bc3193139ecc",
		LinknameDirectives: []string{"gomadNetworkBegin internal/gomadio.BeginSimulation", "gomadNetworkPartition internal/gomadio.PartitionSimulation", "gomadNetworkHeal internal/gomadio.HealSimulation", "gomadNetworkDelay internal/gomadio.DelaySimulation", "gomadNetworkGroup internal/gomadio.ChangeSimulationGroup", "gomadNetworkRevoke internal/gomadio.RevokeSimulation", "gomadNetworkFinish internal/gomadio.FinishSimulation"},
	},
	"runtime_process.go": {
		Name: "runtime_process.go", SHA256: "sha256:db3266da1521d4d3e87ad59de1390a21a58c33cac8b39829a9cb3243bc52cedb",
		LinknameDirectives: []string{"gomadProcessAvailable internal/gomadsim.ProcessAvailable", "gomadProcessRole internal/gomadsim.ProcessRole", "gomadProcessBootstrap internal/gomadsim.ProcessBootstrap", "gomadProcessExchange internal/gomadsim.ProcessExchange", "gomadProcessWaitStop internal/gomadsim.ProcessWaitStop", "gomadProcessServeModel internal/gomadsim.ProcessServeModel"},
	},
	"runtime_process_model.go": {
		Name: "runtime_process_model.go", SHA256: "sha256:2d46945f0cd4b45afe49f5b84f19cc87a90370d2a3600b3ba332b6a7d7e7924d",
		LinknameDirectives: []string{"gomadProcessNetworkOperation internal/gomadio.ProcessSimulationNetworkOperation", "gomadProcessVolumeOperation internal/gomadfs.ProcessSimulationVolumeOperation"},
	},
	"runtime_time_toolchain.go": {
		Name: "runtime_time_toolchain.go", SHA256: "sha256:211c01f57125ba62115b1ffce5d2479d3c22116d51a41aefcfb1a576e8b393a9",
		LinknameDirectives: []string{"gomadSimulationTimeAdvance runtime.gomadSimulationTimeAdvance", "gomadSimulationTimeCurrent runtime.gomadSimulationTimeCurrent", "gomadSimulationTimeTakeArrivals runtime.gomadSimulationTimeTakeArrivals"},
	},
	"runtime_volume.go": {
		Name: "runtime_volume.go", SHA256: "sha256:feb0d31fa9c0d7d6a7c85c3aac6d9d12666bbfe94874e6e2077ca4c78b430a0c",
		LinknameDirectives: []string{"gomadVolumeBegin internal/gomadfs.BeginSimulationVolumes", "gomadInitializeVolumeFilesystem os.gomadInitializeSimulationFilesystem", "gomadVolumeRegister internal/gomadfs.RegisterSimulationVolumes", "gomadVolumeRevoke internal/gomadfs.RevokeSimulationVolumes", "gomadVolumeEnumerate internal/gomadfs.EnumerateSimulationVolume", "gomadVolumeFinish internal/gomadfs.FinishSimulationVolumes"},
	},
}

// AllowsSimulationBridge reports whether source is one of the pinned bridge
// sources of the first-party simulation harness in the main module.
func AllowsSimulationBridge(pkg Package, source Source) bool {
	exactPackage := pkg.ImportPath == simulationImportPath || pkg.ForTest == simulationImportPath && strings.HasPrefix(pkg.ImportPath, simulationImportPath+" [")
	if !exactPackage || pkg.Policy.Module.Path != "go.temporal.io/server" || !pkg.MainModule || pkg.Policy.Module.Replaced || source.MalformedLinkname {
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
