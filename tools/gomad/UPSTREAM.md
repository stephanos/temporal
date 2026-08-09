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

## Go 1.26 compatibility

The nested module declares Go 1.26.0. The local port is verified with Go 1.26.1
on Darwin/ARM64 against gosim's runtime, translator, translated behavior,
nemesis, and race-detector suites.

Local compatibility changes cover:

- moved and added runtime/syscall entry points, ARM64 CPU probes, DIT and caller
  intrinsics, wait-group semaphores, environment clearing, and `vgetrandom`;
- Go 1.26 FIPS packages and assembly boundaries, including per-simulation
  indicator/bypass state and constant-time helpers;
- `reflect.TypeAssert`, `Value.Seq`, and `Value.Seq2` for translated maps;
- named-map generic constraints and Go 1.26's `internal/sync.HashTrieMap`;
- `internal/race`, `internal/synctest`, `weak`, and new time runtime hooks; and
- Linux `O_DIRECTORY` support required by the crash/disk tests.

The principal acceptance commands, run from this directory, are:

```text
go build -tags=test_dep -o .gosim/gosimtool ./cmd/gosim
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./gosimruntime
.gosim/gosimtool prepare-selftest
.gosim/gosimtool test ./internal/tests/behavior ./nemesis
.gosim/gosimtool test -race ./internal/tests/behavior ./nemesis
```

The port has deliberate compatibility approximations. Weak pointers keep a
strong reference, `internal/synctest` bubble bookkeeping is not modeled, FIPS
state is not a complete clone of the Go runtime's state model, the internal
race adapters lose some object/PC fidelity, and the `internal/sync` hash-trie
adapter uses a constant hash. The latter is collision-correct but can degrade
to linear performance. These choices should be revisited if production code
depends on the omitted semantics or profiles show the hash trie as a hotspot.
