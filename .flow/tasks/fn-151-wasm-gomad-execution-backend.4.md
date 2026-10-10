---
satisfies: [R4]
---
# fn-151-wasm-gomad-execution-backend.4 Integrate WASM preparation artifacts and replay

## Description
Typed prepared targets/private executor and engine/profile provenance integrate with existing campaign/artifact/replay owners. Discover intentional failure and replay100fresh executions; reject changed/corrupt inputs and preserve native canonical bytes/defaults.

## Acceptance
Typed prepared targets/private executor and engine/profile provenance integrate with existing campaign/artifact/replay owners. Discover intentional failure and replay100fresh executions; reject changed/corrupt inputs and preserve native canonical bytes/defaults.

## Done summary
Implemented the explicit WASM preparation/provider boundary, source-bound private build cache, campaign publication, retained observed replay and interrupted resume while preserving native defaults and canonical bytes. Guest execution stays sealed; new WASM owners are in tools/gomad_wasm.

Root accepted R4 against the 45-source freeze d57de026faff0027189ca771f811adf10271189fb2031a74c111640f6d7874de. Acceptance receipt: .tmp/wasm-backend-runner/root-task4-acceptance.json (SHA256 7bbf61329cf8900c9d384cc372629062e58f0426efaf4c7c323e11ee68f2b94c); task4-final-handover.json binds actual commands, exits, source/tool identities and logs. Independent gpt-6.1-sol/high review (same model family as writer) found no remaining P1/P2.

The retained full backend gate passed in 351.532s, including 100 fresh observed replays and 300 complete raw stdout/stderr/model-evidence comparisons. Those executions bind the 44-source candidate; the final 45-source getter/style correction has a reviewed compatibility bridge, focused publication/replay/resume checks and an additional exact-artifact replay with the compiler unavailable. This is observed stock replay, not forced scheduling replay or strict-time qualification.

Nonmutating lint, both-platform source architecture/purity, generated validation and scoped identity/cache/validation/lifecycle controls passed. Root rechecked all 111 protected files and exact 4,375-byte native canonical before/after equality. Broad stock-source execution passed 41 of 54 packages; all 244 failed leaves are accounted for in stock-host-classification-final.json. Canonical TMPDIR controls cover nine path leaves; coordinator regressions are corrected; corpus, watchdog and simulation failures reproduce with HEAD owners; unchanged deleted-CWD tests fail during setup before Gomad invocation. Missing native installation/hooks remain unqualified under fn-128/fn-149. No full host/native pass is claimed.

User requested wrap-up on 2026-10-09. Tasks 5–11 remain todo in MILESTONES.md and their Flow tasks; no further runtime implementation started. User retains commits; no staging, commit, push, PR or CI action performed for this backend.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: tools/gomad_wasm: stock Go1.27.1 test -tags test_dep -count=1 -timeout=15m -v ./backend (351.532s; frozen44-source candidate), make lint-code-fast GOLANGCI_LINT_BASE_REV=60bbfbdd7c91e014f3a7aca7c813c893ad6ddc89 GOLANGCI_LINT_FIX=false, tools/gomad3: stock Go1.27.1 test -tags test_dep -count=1 -v -run "^(TestPackageArchitecture|TestPureModulesHaveNoHostEffects|TestPublicPackagesDoNotExportTypeAliases|TestWASIPureEnvironmentAndPublicBoundary)$" ., env -u GOROOT GOENV=off GOFLAGS= GOTOOLCHAIN=local GOWORK=off GOEXPERIMENT=nogreenteagc PATH=<pinned Go1.27.1>/bin:/opt/homebrew/bin:/usr/bin:/bin make -C tools/gomad3 validate (exact environment in task4-final-handover.json), tools/gomad3: stock Go1.27.1 test -tags test_dep -count=1 -v -run "^TestVerifyRegisteredAdapterScratchCleanup$/.*/^(primary-and-cleanup|cleanup)$" ./deterministicio (owned cache/canonical TMPDIR), tools/gomad_wasm: stock Go1.27.1 run -tags test_dep .tmp/wasm-backend-runner/lint-retained-compatibility/main.go <retained100 artifact> <outputRoot> <verified helper> (exact argv in final-additional-checks-receipt.json), python3: root final acceptance verified45source hashes,111protected sources,8artifact payloads,303raw byte comparisons and retained command/log hashes
- PRs: