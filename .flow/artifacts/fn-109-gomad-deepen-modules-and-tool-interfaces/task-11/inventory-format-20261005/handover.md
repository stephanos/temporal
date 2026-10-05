# Source-inventory formatted write checkpoint

The sole production change replaces the per-file `hasher.Write([]byte(fmt.Sprintf(...)))`
with `fmt.Fprintf(hasher, "sha256:%x", digest)`, retaining `_, _` for the infallible
SHA-256 writer. Source admission is HEAD `4695a9ad18de1aa49e032dad82154f73635e9c8d`
on `gomad`. No tests, imports, comments, signatures, framing, traversal, limits,
validation/error order, consumer mappings, pins or generated outputs changed.
Root owns Flow, source-progress review, integrated lint and commits.

[source-freeze.json](source-freeze.json) binds BASE and FINAL, the selected 1,052
tracked source paths through one fingerprint, nine individual source/config/pin
hashes, five executable hashes and the offline environment. Every command has an
exclusive directory containing `receipt.json`, byte-preserved `stdout.log` and
`stderr.log`. Receipts bind exact argv/cwd, timestamps, elapsed seconds, exit,
test names/counts and before/after source/tool stability. All 15 worker commands
are terminal and retained inputs were stable for every command.

| Capture | Exit; actual result | Seconds |
| --- | --- | --- |
| [BASE inventory](base-inventory/receipt.json) | 0; 8 pass, 0 fail, 0 skip | 1.195 |
| [FINAL inventory](final-inventory/receipt.json) | 0; 8 pass, 0 fail, 0 skip | 0.774 |
| [BASE target](base-target/receipt.json) | 0; 10 pass, 0 fail, 0 skip | 2.614 |
| [FINAL target](final-target/receipt.json) | 0; 10 pass, 0 fail, 0 skip | 1.499 |
| [BASE adapters](base-adapters/receipt.json) | 1; 1 pass, 2 fail, 0 skip | 0.394 |
| [FINAL adapters](final-adapters/receipt.json) | 1; 1 pass, 2 fail, 0 skip | 1.969 |
| [BASE actual scoped lint](base-lint/receipt.json) | 1; exactly one QF1012 | 7.418 |
| [FINAL actual scoped lint](final-lint/receipt.json) | 0; `0 issues.` | 1.241 |
| [Architecture/purity/edges](architecture/receipt.json) | 0; 3 pass, 0 fail, 0 skip | 139.567 |
| [Standalone errortype](errortype/receipt.json) | 0; empty output | 3.692 |
| [Formatting](formatting/receipt.json) | 0; empty output | 0.332 |
| [Check-only validation](validate/receipt.json) | 0 | 46.325 |
| [Exact preservation check](preservation/receipt.json) | 0 | 0.784 |
| [Source diff check](source-diff-check/receipt.json) | 0; empty output | 0.023 |
| [Effective Go environment](go-environment/receipt.json) | 0; stock Go1.27.1 linux/arm64 | 0.098 |

The three sourceinventory tests ran their literal digest, typed capacity and
unsafe/empty/missing-root controls. The unchanged literal digest is
`sha256:624ffd10d3b0e4126993be4d4c60de5dba62a7bc08df9a1b9f4db1a0260c07b3`.
All four named target controls ran; their canonical golden and replacement,
nested-package and capacity checks passed. All three named adapter controls
ran: `TestPrepareGRPCReturnsTypedInventoryCapacityError` passed, while
`TestPinnedAdapterModuleInventories` and
`TestRewrittenModuleInventoriesMatchPinnedModules` failed before digest checks
because `tools/gomad3/.toolchain/bin/go` is absent. Their BASE/FINAL error output
and every focused suite's terminal test-event multiset match exactly. No host
shim, alternate assertion or download was added.

[checks.mjs](checks.mjs) reconstructs FINAL from immutable Git BASE with precisely
one replacement and verifies that this is the only changed selected tracked
source. It also compares complete scoped lint output: exactly one QF1012 was
resolved, zero findings introduced, zero remaining in this package. This is the
actual analyzer RED/GREEN; passing behavior controls do not supply a behavioral
RED. No whole-Gomad diagnostic count is inferred.

Generator ownership was inspected in `tools/gomad3/Makefile:6`–`:10`, its
check-only validation recipe, `toolchain/version/descriptor.go` and
`internal/gomadtool/generation/protocol/protocol.go:564`–`:615`. The changed host
inventory file is outside their declared generation and protocol identity
inputs. Actual check-only validation passed without changing the source
fingerprint or generated files. Its non-JSON profile-test output supplies no
native profile qualification or additional counted test pass. Architecture and
purity tests inspect darwin/arm64 and linux/amd64 source sets; execution here is
developmental linux/arm64 and establishes no supported-host qualification.

Root can use the same runner for its actual integrated gate and fresh focused
recheck. Each label creates a new directory and refuses existing output paths.
Run from the repository root:

```sh
inventory_capture=.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-11/inventory-format-20261005/capture.mjs
node "$inventory_capture" run FINAL root-integrated-lint . make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false GOLANGCI_LINT=/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 ERRORTYPE=/tmp/fn109-lint-tools.ZdNe1t50/errortype
node "$inventory_capture" run FINAL root-focused-recheck tools/gomad3 go test -count=1 -tags test_dep -json ./internal/sourceinventory
```

The worker did not run integrated lint. Root must retain its complete measured
diagnostic delta and actual unreached stages. Standalone errortype success
does not prove that the integrated errortype stage was reached. Original task10
dependency, R17/R18 preservation, matched-first-baseline, full/default/functional/
affected-consumer/formal/native Darwin acceptance remains required and open
where unproved. Missing native Linux proof remains deferred and nonblocking
under fn-128. This handover supplies bounded source progress, with no formal
SHIP/DONE or independent review verdict.

FINAL inventory file SHA-256:
`5df9e8cbf6f11f90ee804acf502b8b7dca45f83fdebcb88da9f4992253da53f4`.
FINAL selected-source fingerprint:
`8541e2f5e20623ba38928ce13b9312b717ac39d7d7b9de57614e5f898cb9e134`.
