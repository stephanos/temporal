# Upstream provenance

This directory is a source snapshot of
[`jellevandenhooff/gosim`](https://github.com/jellevandenhooff/gosim).

- Upstream commit: `ffd3a613542675755e4cbf8186b5edaf404ed95c`
- Upstream branch: `main`
- Imported: 2026-08-09

The upstream `.git` directory is intentionally excluded. The original
`go.mod` is retained so gosim remains a nested module and does not add its
dependencies to the Temporal server module.

Local compatibility changes should be kept small and documented here so a
future upstream refresh can distinguish them from the imported snapshot.

## Go 1.26 compatibility experiment

The imported module still declares Go 1.23.2. With Go 1.26.3 on Darwin/ARM64,
the gosim runtime unit target passes with the required `linkname` build tag and
linker flag, but translated self-tests do not yet build.

Local adapters currently cover:

- `internal/runtime/syscall/linux.Syscall6`;
- `internal/cpu.getpfr0`;
- `internal/runtime/sys` DIT and caller intrinsics;
- `crypto/internal/fips140/subtle.xorBytes`;
- `sync.runtime_SemacquireWaitGroup`; and
- `syscall.runtimeClearenv`.

The remaining hard failures include the FIPS SHA-256 and SHA-512 assembly entry
points `blockSHA2` and `blockSHA512`. Translation also discovers new linkname
surfaces in `crypto/subtle`, `weak`, `internal/synctest`, `internal/sync`,
`internal/runtime/maps`, `internal/syscall/unix`, and `time`. This is a partial
porting experiment, not a Go 1.26-compatible gosim release.
