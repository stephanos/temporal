---
satisfies: [R6, R8]
---
# fn-165-gomad-architecture-checks-structural.3 Load types from export data and write the version-keyed standard-library allowlist

Touches: tools/gomad3/internal/puritycheck

## Description
Create `tools/gomad3/internal/puritycheck` with a loader that type-checks only the module's pure packages using `go list -export -deps` and `go/importer.ForCompiler(fset, "gc", lookup)` for every dependency, one platform per call. No standard-library source is read, parsed or hashed. Write `stdlib.go`: a table of pure standard-library symbols keyed by package with `*` and `!` syntax as in the spec, a `reviewedGoMinor` constant, and the review test that fails when the active toolchain's minor differs. Seed the table from `memorySymbols` and the host classification in the old `standard.go`, then trim it to what the gomad3 pure set and the wasm pure set reference. `time` admits only the listed pure symbols; `os`, `os/exec`, `net`, `math/rand`, `crypto/rand` admit nothing; `syscall` admits `Errno` and `E*`.

## Acceptance
- Loading the gomad3 pure set on the host platform completes in under 30 seconds and reads no file under `GOROOT/src`.
- A test with a fake `go version` minor fails naming both the reviewed and the active minor.
- Table lookup: `time.Now` is rejected, `time.Date` and `time.UTC` admitted, `bytes.*` admitted, `os.Getenv` rejected, `syscall.EINVAL` admitted, `syscall.Open` rejected.
- The table and loader together are at most 600 non-test lines.

## Done summary
internal/puritycheck loads pure packages from export data with importer.ForCompiler per platform without reading GOROOT sources; the version-keyed standard-library table rejects time.Now/os.Getenv/syscall.Open and admits time.Date/time.UTC/bytes/syscall.EINVAL; a minor mismatch names both versions.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c6afac5097
- Tests: root architecture/purity selection with GOMAD3_ARCH_ALL_PLATFORMS=1 plus ./internal/archcheck ./internal/puritycheck (0), TestStdlibSymbols (0), evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs: