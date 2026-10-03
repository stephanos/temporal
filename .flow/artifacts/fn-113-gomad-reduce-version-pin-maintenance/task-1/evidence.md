# fn-113 task 1 evidence

Host: linux/arm64 with the local development harness on (linux/arm64 added to
`version.json` and the boundary manifest, `syscall.Dup2` shimmed). Everything here is
linux/arm64 development evidence, not darwin/arm64 or linux/amd64 evidence. Source: branch
`gomad-next-b` on `6be0755fe` plus this task's change (commit recorded in the task's
Evidence section). Go: pinned go1.27.1.

## Files

- `measure-pins.sh`, `pin-baseline.txt`, `baseline.md`: R1 baseline at `6be0755fe`.
- `root-bump-candidate-go.mod.diff`: a real candidate made from the repository root module in a
  scratch copy with `go get github.com/klauspost/compress@v1.18.6 google.golang.org/grpc@v1.84.0`.
- `root-bump-report.txt`, `root-bump-report.json`: `gomadtool pin-impact` on that candidate
  against the root module (exit 1, 3.0 s, network module proxy). It names the gRPC adapter and
  30 rules of the 4 packs that activate on `klauspost/compress@v1.18.5`; the candidate's
  `go.mod` and `go.sum` hashes were unchanged after the run. The JSON lists
  `linux/arm64` in `platforms` and 133 interception fingerprints only because the harness adds
  a linux/arm64 platform override; the committed manifest has 132.

## Commands and results (linux/arm64, harness on)

| Command | Result |
| --- | --- |
| `go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/...` | pass |
| `go -C tools/gomad3 test -tags test_dep -count=1 -run TestPackageArchitecture .` | pass |
| `make -C tools/gomad3 validate` | pass (4 to 11 s) |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` | see `test-host.txt` |
| `go -C tools/gomad3 run ./cmd/gomadtool pin-impact` (root module against `HEAD`) | exit 0: 9 of 15 adapters and 48 of 54 rules selected and unaffected, 6 and 6 not selected |

Acceptance tests, all in `upgrade/pinimpact` and `cmd/gomadtool`:

- Fixture bump of one adapted (`getsentry/sentry-go`) and one packed
  (`klauspost/compress`) module yields exactly the adapter and the two
  `temporal-leaf-xxhash-darwin-arm64` rules, and the adapter registry's
  `PrepareTargetBuildAdapters` and pack selection's `AllowsCapability` reject the same pins
  while accepting the baseline: `TestFixtureBumpMatchesBuildRejections`.
- Same version with changed sum (`TestSameVersionWithChangedSum`, also rejected by the
  registry's sum check), removed modules (`TestRemovedModules`: stale, plus the rule whose pack
  activation lost a module), replaced modules (`TestReplacedModules`, rejected by the registry's
  replacement check), and indirect-only bumps, tidy and untidy (`TestIndirectOnlyBump`).
- Unknown pins: a candidate needing a newer Go, a root without the descriptors, and an
  unloadable external pack directory are reported unknown and invalidate the candidate, with
  path-free reasons (`TestUnknownPinsCountAsInvalidated`).
- Exit statuses 0/1/2/3, Git and directory baselines, and unchanged candidate `go.mod`/`go.sum`
  through the CLI with a file-based module proxy (`TestRunPinImpact*`); resolution outside the
  module with a private module cache, including relative local replacements
  (`TestGoResolver*`).

## Still owed on qualified platforms

R2's acceptance is platform-independent in design (pins are judged by exact module identity
for every platform's packs), but the gates above ran only on linux/arm64. Owed:

```sh
# darwin/arm64 and linux/amd64, harness absent
go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/...
go -C tools/gomad3 test -tags test_dep -count=1 -run TestPackageArchitecture .
make -C tools/gomad3 validate
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
go -C tools/gomad3 run ./cmd/gomadtool pin-impact   # expect exit 0 on an unmodified checkout
```

## Harness off (committed state on linux/arm64)

`make -C tools/gomad3 validate` stops at `undefined: syscall.Dup2` in `runner/internal/execution`
(the known harness limitation), so it passes only with the harness on. With the harness off,
the three pin-impact tests that run the adapter registry's build check fail before the check,
with `deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64`; the
report itself and the remaining tests do not depend on the host platform.
