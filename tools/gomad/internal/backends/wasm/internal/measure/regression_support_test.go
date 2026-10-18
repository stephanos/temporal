package measure

import (
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

type terminalExitObservation = wasi.TerminalExitObservation

func measurementResult(result wasi.Result, terminal [16]byte, stderr []byte) wasi.Result {
	result.SetRuntimeTerminal(terminal)
	result.SetHelperStderr(stderr)
	return result
}

func measurementTerminalBytes(result wasi.Result) []byte {
	terminal := result.RuntimeTerminal()
	return terminal[:]
}

func warmPreparationFixture() warmPreparationSeries {
	series := warmPreparationSeries{Schema: warmPreparationSchema, Role: "candidate", SourceCommit: strings.Repeat("a", 40), SourceRoot: "/source", DriverSHA256: strings.Repeat("d", 64), HostBootID: "same-boot", HostCPUInfo: `{"KernelOS":"Darwin","Machine":"arm64","KernelVersion":"Darwin Kernel Version fixture","CPUInfo":"hw.optional.arm64: 1\nmachdep.cpu.brand_string: Apple M2","Translation":"0"}`, ConfigSHA256: strings.Repeat("c", 64), Engine: wasi.PinnedEngine(), ModuleSHA256: strings.Repeat("a", 64), HelperSHA256: strings.Repeat("b", 64), RuntimeIdentity: "runtime", ModelSHA256: "model", ArtifactBytes: 100, EnvelopeBytes: 200, Inputs: map[string]string{"/source/driver.go": strings.Repeat("d", 64)}}
	series.Engine.HostOS, series.Engine.HostArch = "macos", "aarch64"
	series.ModuleReceipt = "/module-receipt.json"
	series.ComparisonSHA256 = strings.Repeat("e", 64)
	series.Inputs[series.ModuleReceipt] = strings.Repeat("a", 64)
	for range 5 {
		series.Runs = append(series.Runs, warmPreparationRun{Telemetry: wasi.Telemetry{CacheHit: true, PrepareNanos: uint64(time.Second)}, Phases: map[string]diagnosticPhase{"cache_read": {Count: 1, WallNanos: 2}, "envelope_verify": {Count: 1, WallNanos: 3}, "deserialize": {Count: 1, WallNanos: 4}, "cache_identity": {Count: 1, WallNanos: 1}}, WallNanos: 10, TerminalToReapNanos: uint64(time.Second), EvidenceSHA256: strings.Repeat("e", 64), Exited: true, Reaped: true})
	}
	return series
}
