# fn-189 seven WASM gate repairs

This follow-up covers the seven failing WASM tests recorded in the
[upstream checkpoint](fn189-upstream-sync-2026-10-17.md). That checkpoint was
subsequently committed as merge `0077477965`; its pending-merge description is
historical. The owner requested these seven repairs only. The repairs remain
uncommitted, and later milestones and native execution remain paused/deferred.

## Causes and changes

Cooperative preparation inferred an aliased compiler GOROOT while its runtime
overlay used canonical paths. On Darwin, `/var` and `/private/var` disagreed,
so Go missed the runtime instrumentation and import validation correctly refused
the resulting module. Preparation now canonicalizes GOROOT before refreshing
the compiler inventory, building the overlay, listing sources and compiling.
The source/cache identity changes with the effective compiler paths. A portable
compiler-symlink regression explicitly forces the legacy alias and verifies a
different cache key, then retains exact detached replay and refusal checks.
The build-selection fixture also uses the canonical temporary source path.

Warm-preparation fixtures combined the current engine platform with Linux/arm64
CPU evidence, and admission still enforced the superseded platform restriction.
Warm host evidence now binds the recorded OS/architecture, kernel, CPU and
translation status to the engine. Existing Linux/arm64 evidence remains valid.
Actual collection uses bounded host commands and checks the same boot and CPU
bindings after collection. Darwin memory admission checks free plus inactive
pages against the unchanged 8GiB bound; Linux retains its MemAvailable check.
Malformed, missing, duplicate and overflowing memory evidence refuses dispatch.
Five-run requirements, comparison bindings and performance thresholds remain.
Environment-gated measurements were not collected, so this supplies no new
performance or workload qualification claim.

## Verification

Host: Darwin/arm64, stock Go 1.27.1. Commands use `-tags test_dep`, unset
ambient GOROOT, and explicitly select the pinned Darwin stock compiler under
`tools/gomad/.toolchain/stock/7932b4a12ed12e59/go/bin/go` where required.
Raw logs are retained locally under `.flow/tmp/fn189/`.

| Check | Result | Log |
| --- | --- | --- |
| Original replay/backend and operator failures reproduced | Red | `fix-replay-backend-red.log`, `fix-replay-command-red.log`, `fix-replay-symlink-red.log` |
| Original warm controls and added memory negatives reproduced | Red | `fix-warm-red.log`, `fix-warm-fixtures-red.log`, `fix-warm-memory-red.log` |
| Build-selection and cooperative detached replay tests | Pass | `fix-replay-backend-final.log` |
| Installed operator discovery/inspection/replay and typed detached replay | Pass | `fix-replay-command-green.log` |
| Warm preparation, current host evidence, memory and diagnostic controls | Pass | `fix-warm-focused.log` |
| Final portable compiler-alias detached replay regression | Pass | `fix-replay-portability.log` |
| `make -C tools/gomad test-wasm GOMAD_STOCK_GO=<pinned stock compiler>` | Exit 0; slim refusals, namespace conformance, projection checks and all WASM packages pass | `fix-full-wasm.log` |
| Canonical `make lint-code-fast GOLANGCI_LINT_BASE_REV=0077477965 GOLANGCI_LINT_FIX=false` | Exit 0, zero issues | `fix-lint-final.log` |

Independent review found a Linux portability issue in the new regression's
setup. Explicit legacy GOROOT resolves it; the affected detached replay test
and canonical lint passed afterward. Review has no outstanding findings.
The full WASM run started before that test-only portability correction; the
affected regression was rerun against the final source as listed above.

No Rust helper, runtime overlay, protocol or model implementation changed.
Helper SHA-256 remains
`f543978ac9d2f913f5b044049b7f45a4553cf2fc1533c3a16bb1b3b23cfed1f6`;
model identity remains
`sha256:9482b9c5e810464896c29d569c269d9b7e6d841872949ffa6cd20cfd5a7b0744`.

Fn-189 remains open until its full current-candidate workload ladder and retained
R1-R4 budgets and gates are established. Seven repaired tests alone do not
complete the milestone.
