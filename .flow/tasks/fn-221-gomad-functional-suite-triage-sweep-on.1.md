---
satisfies: [R1]
---
# fn-221-gomad-functional-suite-triage-sweep-on.1 Build the sweep harness and prove it on two packages

## Description
Add the explicit-input sweep driver under `internal/backends/wasm`, refusing inherited `GOMAD_*` variables and unbounded runs. Enumerate top-level functions, registered suite methods (including nested registration paths) and nested modules under `tests`, reconcile wrappers against `go test -list`, and require the selected method's own terminal verdict. Reuse an authenticated module by compile-only inputs while retaining fresh per-execution provenance; test compiler counts, invalidation and distinct runtime inputs. Drive the existing explore go-test path with one seed, parallelism one and bounded preparation/execution/global deadlines. Use fn-220's temporal installation now that it has landed, recording that choice. Classify the eight execution classes from the spec, with skips and explicit budget/cap not-run states kept distinct. Early proof remains `tests/gomadfunctional` plus one ordinary tests package capped at ten isolated units; wrapper-only v1 evidence cannot establish it. The full uncapped survey belongs to task .2.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/** (harness/cache), tools/gomad/docs/wasm/README.md, .flow/artifacts/fn221-*

**Touches:** tools/gomad/internal/backends/wasm/** (harness/cache), tools/gomad/docs/wasm/README.md, .flow/artifacts/fn221-*
## Acceptance
- [ ] Harness refuses inherited GOMAD_* variables and unbounded runs
- [ ] Top-level counts reconcile against go test -list; method declarations and registration paths are inventoried separately
- [ ] Every function in the two proof packages has exactly one disposition, with cap/budget not-run rows explicitly distinguished from the executed classifications
- [ ] The capped proof classifies the selected isolated units by their own verdicts, not just suite wrapper verdicts
- [ ] Configuration source (fn-220 or throwaway) recorded
- [ ] Compile-only reuse preserves per-execution provenance; compiler-count and invalidation regressions pass
## Done summary
Implemented the method-aware developer harness at `internal/backends/wasm/cmd/wasmsweep` with complete function/registration discovery, nested-module discovery, explicit input/environment refusal, bounded preparation/execution, selected-method verdicts, authenticated compile-only reuse and resumable v2 evidence. Independent reviews covered both compile-cache correctness and the repaired harness; the final two-package evidence review returned SHIP with no remaining findings.

Corrected proof: `.flow/artifacts/fn221-method-proof-2026-10-20.md` and its two v2 JSON sidecars, frozen source `cc84570e2838dda3ec185764b5a099b284796ec9`, fn-220 temporal-install configuration. Functional package 1/1 PASS. Ordinary root package reconciled 148/148 top-level functions, inventoried 1173 methods and 1324 isolated units, ran the capped ten selected methods (10 PASS) and explicitly labeled the remaining 1314 not-run. Real cluster-backed activity fixtures started. Every timed root method reused the same module with engine cache hit and zero new engine entries. Same-candidate resume exited 0 without re-executing any unit or warm-up: two headers, one warm-up, ten execution records.

Focused source tests, compiler-count/invalidation/provenance regressions and harness/command lint passed before the frozen proof. Root freshly verified both committed v2 sidecars with the integrated harness (exit 0 each). Integrated Gomad lint passed at `28d2c88147`; architecture checks passed at `57d8b5f1ad`. These are source/task evidence, not a current-candidate qualification claim. Earlier wrapper-only v1 evidence is explicitly historical and does not establish method coverage. The full sweep and ranked recommendations remain open under .2/.3; no support list or qualification row was changed by fn-221.

stage: impl-review - ran (model: gpt-6.1-sol high; independent read-only reviewer, same model family as writer)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cc84570e2838dda3ec185764b5a099b284796ec9, 6546e4f31a, 211b3236e1, 798efffc13
- Tests: GOMAXPROCS=2 go test -p=1 -tags test_dep ./internal/backends/wasm/internal/sweep ./internal/backends/wasm/cmd/wasmsweep ./internal/backends/wasm/command ./internal/backends/wasm/installation (exit 0), go test -p=1 -tags test_dep -run '^Test(Stress.*|Compilation.*|CachePublication.*|OptionsObservationIsNotSerialized)$' ./internal/backends/wasm/backend (exit 0), CONFIG_SOURCE=temporal-install PER_TEST_TIMEOUT=3m PREPARE_TIMEOUT=8m OVERALL_BUDGET=30m bash /home/agent/wt/evidence/fn221/proof-point.sh /home/agent/wt/fn221 (exit 0; retained commands and limits in report), same-candidate root proof resume and verifier (exit 0; no additional guest executions), go run -tags test_dep ./internal/backends/wasm/cmd/wasmsweep --verify ../../.flow/artifacts/fn221-method-proof-gomadfunctional-2026-10-20.json (exit 0), go run -tags test_dep ./internal/backends/wasm/cmd/wasmsweep --verify ../../.flow/artifacts/fn221-method-proof-root-2026-10-20.json (exit 0), make lint-code-gomad GOLANGCI_LINT=/home/agent/wt/fn222/.bin/golangci-lint-v2.13.0 ERRORTYPE=/home/agent/wt/fn222/.bin/errortype GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=9cb5bc3149 (exit 0 at 28d2c88147), go test -p=1 -tags test_dep ./internal/archcheck (exit 0 at 57d8b5f1ad)
- PRs:
