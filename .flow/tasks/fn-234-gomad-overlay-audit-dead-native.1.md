---
satisfies: [R1, R4, R5]
---
# fn-234-gomad-overlay-audit-dead-native.1 Delete the dead Go 1.27.1 intercept list and proven-unreferenced overlay leftovers

## Description
Touches: tools/gomad/expected-intercepts-go1.27.1.txt (delete), tools/gomad/internal/backends/wasm/toolchain/**, tools/gomad/internal/model/runtimesource/**, tools/gomad/README.md, tools/gomad/internal/backends/wasm/README.md.
Sweep the overlay and toolchain surface for files no Go source, Makefile target, generator, fixture loader or document reads: grep the module for each filename and the native-era names (gomadIntercept, gomad3, native overlay); record the proving search per file in the done summary. Delete the intercept list (111 net/os intercepts with no defining source; only reference is the closed task fn-114.5). Keep files any WASM gate reads even when named native (qualification/native_crash.go, runner diagnostic-identity-*-native.json): they are not leftovers. Remove documentation mentions of deleted files. Run make validate and the behavioral gate on the fixture workloads; this task changes no overlay bytes, so the gate here is a build/test/doc check.

## Acceptance
Retained implementation evidence and proving searches are indexed in [.flow/artifacts/fn234-overlay-audit-evidence-2026-10-20.md](../artifacts/fn234-overlay-audit-evidence-2026-10-20.md). Review the authenticated inputs and raw paired results there; a declaration-only check is not the behavioral gate.

Intercept list deleted; every other deletion has a recorded search proving it unreferenced; no file read by a WASM gate deleted; go build/vet/lint, toolchain and runtimesource tests, make validate green; no document names a deleted file; overlay implementation identity unchanged by this task.

## Done summary
Deleted only tools/gomad/expected-intercepts-go1.27.1.txt (131 lines/111 obsolete intercept names), root checkpoint c19e8af5ba. Exact filename/native-era/source/build/generator/fixture/document proving searches are indexed in .flow/artifacts/fn234-overlay-audit-evidence-2026-10-20.md and /home/agent/wt/evidence/fn234-overlay/HANDOVER.md. No active reader exists; no other scoped file was proven dead. WASM-used native-named inputs remain. clock_audit.d is retained as native qualification executable tooling outside the overlay/toolchain data cleanup scope. Deletion itself leaves all runtime identity inputs unchanged and is recoverable from Git.

The complete two-task batch passed Go build/vet/lint/focused/backend/installation checks, full make validate, and the dedicated old/new execution gate (68 paired labels with actual identities and own replay evidence). Root repeated focused checks, full validate, source-manifest verification and the comparator. The shared R4 evidence is durably indexed rather than inferred from structural equality. First review's unavailable-evidence finding was addressed by that committed index; independent re-review SHIP in primary session01a15c14-8e45-7281-8acc-ed9db271f27d, receipt .flow/tmp/impl-review-receipt-fn234-delete.json, R1-R5 met. No final Rust qualification claim or frozen-baseline modification.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer/reviewer pins retained. Full final-candidate test-wasm/runtime/workload/replay qualification remains with fn232.13 under MILESTONES, not supplied by these source and dedicated overlay gates.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c19e8af5ba, 05e148b9de, c62a9d43ef, dc87d554cd, 8e7bd00c9c
- Tests: make -C tools/gomad validate on worker d1a89a44ce (exit0,14s), exact environment/log in .flow/artifacts/fn234-overlay-audit-evidence-2026-10-20.md, make -C tools/gomad validate on integrated root05e148b9de (exit0), root-05e148b9de-validate.log, go test -tags test_dep -count=1 -p=2 -timeout=15m ./internal/backends/wasm/toolchain ./internal/model/runtimesource ./internal/archcheck ./internal/backends/wasm (root exit0), go test -v -tags test_dep -count=1 -p=2 -timeout=5m ./internal/backends/wasm/toolchain ./internal/model/runtimesource with downloaded Go1.27.1 (root-c62a9d43ef-focused.log,exit0), env -u BASH_ENV -u SANDBOX_START_DIR /bin/bash /home/agent/wt/evidence/fn234-overlay/compare.sh (exit0,68pairedlabels), sha256sum -c ../source.sha256 in pre/source and ../changed-source.sha256 in post/source (root exit0), independent task1+task2 Codex re-review SHIP; receipts .flow/tmp/impl-review-receipt-fn234-{delete,split}.json, git diff --check (exit0)
- PRs: