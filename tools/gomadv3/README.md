# Gomad v3

Gomad v3 is an opt-in Go 1.26.4 toolchain with a small deterministic-runtime
patch and source overlay. It uses native Go goroutines, channels, `select`,
maps, synchronization, `go run`, and `go test`.

Build the cached toolchain from the repository root:

```sh
make gomadv3-go
```

Run a command or test package with a seed:

```sh
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./cmd/example
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./path/to/package
```

The stable Go command is `tools/gomadv3/.toolchain/bin/go`. The build verifies
the official Go source checksum, snapshots and validates `go1.26.4.patch` and
`overlay`, rejects upstream overlay collisions, copies the exact overlay
snapshot, applies the exact patch snapshot with zero fuzz, and caches immutable
builds by the Go version, source checksum, patch and overlay checksums, host OS
and architecture, bootstrap Go version, and canonical build environment.
Same-key builds use an atomic owner lock, and ambient Go experiment,
architecture, C/C++ tool, and compiler/linker tuning is cleared before
`make.bash`. Set `GOMADV3_BOOTSTRAP_GO` to choose a bootstrap `go` command.

## Contract

When `GOMADSEED` is absent, the toolchain follows the upstream runtime paths.
When it is present, the runtime parses it as a `uint64`, forces the initial
`GOMAXPROCS` to one, disables asynchronous preemption, and seeds existing
runtime choice paths. Seed `0` is valid; empty, malformed, and overflowing
values fail before user initialization.

For a fixed toolchain, architecture, program, deterministic external inputs,
and seed, supported runtime-controlled choices repeat across fresh processes.
Different seeds explore different choices when alternatives exist. Runtime
choices must finish before output or other external I/O is performed.

Deterministic mode supports Unix-like hosts. Windows, cgo, plugins, foreign
threads, the race detector, timers, signals, finalizers, and host-dependent
network, filesystem, process, and other I/O readiness are outside the contract.
The runtime system monitor is disabled with asynchronous preemption, so a
CPU-bound goroutine may run forever and host-driven readiness is unsupported.
Calling `runtime.GOMAXPROCS` to raise the value after startup is unsupported.

The mode is intended only for trusted tests. Deterministic map seeds remove a
hash-randomization defense and must not be enabled in production. Each process
uses one P, so run different seeds in separate processes for parallelism. The
shared runtime random state also means program changes can change later choices.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomadv3 test
```

Generated source, binaries, downloads, and toolchain builds remain under
`tools/gomadv3/.toolchain` and are not committed.
