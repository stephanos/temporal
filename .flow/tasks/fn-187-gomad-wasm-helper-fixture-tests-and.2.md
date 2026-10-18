---
satisfies: [R2]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.2 Move measurement producers out of test files into wasmmeasure

## Description
Inventory the env-gated producers in wasi/integration_test.go and sibling files (runtime cost profile, matched release cluster baseline/candidate, warm preparation, namespace measurement sample, diagnostic cluster pair, guest profile helper build, terminal-exit compiler controls, cold memory guard). Create internal/backends/wasm/cmd/wasmmeasure with one subcommand per producer and explicit flags replacing every GOMAD_WASM_* variable; share the admission, hashing, host-evidence and report-writing helpers by moving them into a non-test package. Preserve evidence identity assertions and refusal rules from QUALIFICATION.md. Delete the test wrappers. Update QUALIFICATION.md, the wasm README and the rerun sections of the fn-172/174/178 artifacts with the new invocations without rewriting their results. Preserve the existing profile collector in the moved command; fn-185 task .1 follows this spec and extends it there.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/cmd/wasmmeasure/**, tools/gomad/internal/backends/wasm/internal/measure/**, tools/gomad/internal/backends/wasm/wasi/integration_test.go and sibling *_test.go, QUALIFICATION.md, README.md, .flow/artifacts/fn17{2,4,8}-*.md

## Acceptance
- [x] No test under wasi skips on a GOMAD_WASM_* measurement variable; producers run only through wasmmeasure
- [x] Each subcommand refuses without explicit host and identity flags and keeps the admission rules
- [x] integration_test.go shrinks by the producer code; count retained
- [x] Docs and artifact rerun sections point at the command
## Done summary
Moved measurement producers into the ordinary internal/measure package and wasmmeasure command: 22 subcommands, 75 distinct explicit inputs/187 bindings, no testing facade or test-binary execution. The selected checkout builds and verifies its production executor. All 114 nonproducer roots are conserved (46 WASI, 68 measure); historical results are unchanged and rerun guidance uses the command.

Four small Go package suites, 46 selected WASI roots, affected canonical lint/errortype and primary generated validation pass. Independent review reproduced and corrected premature warm-pair returns and lost terminal-exit failure evidence with controlled red-to-green regressions, then returned SHIP. [Batch handover](../artifacts/fn187-functional-2026-10-17.md) records exact evidence and identity binding.

Uncommitted for the owner; no push/PR/CI. No producer measurements, performance/resource or workload qualification were run; broader fn-187/fn-189 acceptance remains open.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: Rust 1.94.1 release locked helper_corpus: default, diagnostic-profile, diagnostic-fuel-off and guest-profile; retained engine/cluster feature suites; exact commands/logs in batch handover, Two GOMAD_HELPER_CORPUS_REGENERATE=1 runs with byte-identical before/after manifests, go test -p=1 -parallel=1 -tags=test_dep -count=1: cmd/wasmmeasure internal/measurement internal/measure internal/stockgo, Primary 46-root WASI corpus/identity/admission/fixture/transport selection: 46 pass, zero skip/fail, Canonical affected five-package make lint-code plus errortype, fixes=false: zero issues, make -C tools/gomad validate, go list -deps ./internal/backends/wasm/cmd/wasmmeasure: no native or testing dependency, Independent read-only actual staged-patch review: NEEDS_WORK then SHIP; patch 567ab2a5d171cf63c657e9a6d238f323bba429ced5b58f458619fbd85540ca5a, All 261 nondeleted reviewed files SHA-256 match integrated candidate; git diff --check
- PRs: