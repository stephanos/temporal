# WASM layout repair checkpoint — 2026-10-12

The owner requested wrapping up the current repair and stopping, then cleared the active goal. Work is stopped at this checkpoint. There are no active worker or test handles. This is not Flow task/spec completion or productionization acceptance.

## Committed repair

Source checkpoint: `a2b42872f8` on `gomad` (private source `98194848abeed92162472988ebfbaeec584d8d50`, base `b4696f1a78`).

- Both choice and simulation exploration retain the selected Provider.
- Resume distinguishes a native/static I/O descriptor from validated backend provenance, preserving canonical mismatch and current-options/payload checks.
- Minimization carries the validated Describe I/O descriptor through both publication paths.
- Exact affected import formatting and four lint findings are repaired. Launcher diagnostic-write failures use the existing status-3 error path; ordinary invalid-flag output/status is preserved.
- Compiled WASI model: `sha256:bdc12bc6c6867e8cc6bed1e4d2ed59202709b721e43b410d06da6ffbf0befdf3`. It was observed before updating the test pin. No guest has qualified this final model.

No API/schema growth, new dependency, Rust rebuild or native qualification occurred. Native work remains deferred. Unrelated untracked files and the old helper cache are preserved.

## Verified on the integrated source

All commands used stock Go 1.27.1, `test_dep`, bounded memory and serial package/test execution.

| Evidence | Result | Local raw log |
| --- | --- | --- |
| Root architecture/purity/public-signature, three-platform vet, WASM structural/hostile-reply and lifecycle selections | exit 0; root 45.404s; typed host pass 2.865s; budgets 2030 production/801 test lines | `/tmp/gomad-root-final-source-v2.log` |
| Complete `make -C tools/gomad3 validate` | exit 0; version, protocol, boundary, patch, script, compatibility and qualification-manifest checks passed | `/tmp/gomad-root-final-validate-v2.log` |
| Canonical `make lint-code-fast`, base `e43c7a59c7`, fix=false, unchanged configuration | exit 0; root ordinary/integration scopes and 81 Gomad host packages; zero new issues; errortype vet included | `/tmp/gomad-root-final-canonical-lint-v3.log` |
| Six exact Runner strategy/resume/minimization source regressions and five CLI/error-writer selections | exit 0; Runner 0.460s, CLI 0.005s; in-memory Providers, no guest | `/tmp/gomad-root-stop-checkpoint-regressions.log` |

Worker RED/GREEN details: `/tmp/fn164-finalfix-summary.md`, `/tmp/fn164-finalfix-evidence.json`. Full conductor history: `/tmp/wasm-section-batch-summary.md`.

## Unfinished acceptance

The prior complete `test-wasm` run exited 2 on model `803a462…`: five backend tests failed in Provider propagation/resume, while every other package passed, including installed-command checks, simulation and 100 fresh stock instances. Raw log: `/tmp/gomad-root-final-test-wasm.log`. Those failures are retained; source regressions verify the fixes, but neither that run nor earlier model `41b129…` receipts qualifies the final `bdc12bc…` model.

Final-model guest verification and independent batch review remain open. All fn-164/fn-165 tasks retain `in_progress`; none was marked done. fn-156 remains queued with its historical receipts under their original identities. Current-source 32×100 scheduling/32 exact replays, final workload/operator corpus, installed retained-cluster replay, matched costs and final review remain unfulfilled.

## Resume only on owner request

First verify this checkpoint and available capacity, then run final-model WASM guest verification and independent integrated review. Complete Flow tasks only from current evidence and verified done state. After the layout/checker batch is accepted, freeze one final fn-156 source identity and re-anchor the existing qualification recipes; do not relaunch obsolete sealed scripts or reuse historical source/model receipts as current acceptance.

Prepared R1 path-only delta: `/tmp/fn156-folded-r1-paths-preparation.patch`. No new source capture, native driver build or qualification guest was released. PR, push and CI dispatch remain unauthorized.
