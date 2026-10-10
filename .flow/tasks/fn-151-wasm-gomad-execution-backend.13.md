---
satisfies: [R13]
---
# fn-151-wasm-gomad-execution-backend.13 Migrate WASM backend and tracking into temporal; retire temporal_wasm

## Description
Owner direction 2026-10-10: relocate tools/gomad_wasm into sibling temporal/tools, integrate only required shared/root support while preserving newer destination work, migrate fn-151 including historical evidence/status, update temporal/MILESTONES.md, and retire the old checkout recoverably. fn-151.12 standalone extraction is cancelled. This is relocation, not authorization to resume task5 qualification or any further feature work. No commits, pushes, PRs, new worktrees or native-owner revival.

**Touches:** tools/gomad_wasm/**, required WASM integration in tools/gomad3/** and cmd/tools/lintcode/**, root Makefile and WASI SQLite/dependency seams, fn-151 records/evidence, MILESTONES.md, AGENTS.md, retained WASM research/plan, and the recoverable old-checkout archive. Shared native runtime source/schema owners stay unchanged; unrelated destination work stays untouched.

**Quick commands:** make -C tools/gomad_wasm test-go; focused shared Runner/record/choice/host boundary tests; root SQLite/lint routing tests; make temporal-server-wasi; make -C tools/gomad3 validate; architecture/signature/source-set checks; make lint-code-fast; flowctl validate --spec fn-151 --json; git diff --check. Go tests always include -tags test_dep. Retain exact commands/exits and classify supported-host gates separately.
Migration limitation: WASI prerequisites require Prometheus 1.21.1 and newer minimum x-module versions; native compatibility pack bindings still name earlier versions. The offline pin-impact baseline could not resolve all cached module metadata, so no live native pack validity or qualification is claimed. Native pack refresh, approval and supported-host execution stay with the deferred owners; the relocation does not grant that authority.

## Acceptance
The WASM module and required integration sources exist in temporal/tools/gomad_wasm and temporal shared owners; focused module/build/lint and Flow validation run with outcomes retained. temporal/MILESTONES.md owns fn-151 while preserving its existing specs. Task1–4 acceptance stays historical; task5 stays incomplete and its failed qualification is preserved. Task12 is administratively cancelled, not claimed implemented. Unique source work and evidence remain recoverable, with temporal_wasm no longer an active development checkout.

## Done summary
Moved tools/gomad_wasm intact into temporal/tools and merged the required shared integration into temporal's newer Runner/lint owners. Native SQLite defaults remain modernc; the WASI driver uses the existing ncruces implementation behind platform tags. Required WASI prerequisites, research, plan and module/build/lint discovery moved with the component.

Migrated fn-151 records and runtime states without renumbering. temporal/MILESTONES.md and agent guidance own the combined native/WASM workflow. Tasks 1–4 retain historical acceptance; task 5 remains in progress with failed/incomplete qualification; tasks 6–11 remain queued; task 12 is cancelled administratively and satisfies no active requirement. This closes only migration task 13/R13. No further feature work or task5 qualification starts; spec readiness stays false.

Retirement is recoverable: the old checkout is archived as sibling temporal_wasm.retired-20261010, preserving original Git history, unrelated dirty Flow/docs, .tmp/.flow evidence and the existing nested wasi-pr-stack worktree. Linked-worktree pointers and preservation are checked in retirement-verification.log. No data, commits or history are deleted. Source and destination HEADs remain unchanged; no commit, push, PR, new worktree or native-owner revival occurred.

The implementation handover records actual bounded tests/build/lint commands and exits. Root independently reran record/choice/host boundaries, native SQLite/lint routing, WASI models, Flow validation and diff checks: exit0 in root-verification.log. The ordinary full WASM suite and actual unchanged SQLite guest passed, with long qualification gates explicitly skipped. Latest changed-scope lint and post-review builds pass. Both independent review axes found no remaining migration issue; BackendPayload was moved into the backend file to resolve the size finding. Reviewer and writer are from the same GPT family. See implementation-handover.md and review.md.

Native compatibility packs still bind older Prometheus/x-module versions; offline pin-impact could not resolve all baseline metadata. No live native pack compatibility or supported-host qualification is established. Those requirements remain with deferred owners. Task5's old 400 records/4 replays and watchdog failure are retained as historical partial evidence, not destination qualification.

stage: impl-review - ran (fresh independent correctness and standards reviews; model: gpt-6.1-sol at high)
stage: plan-sync - skipped(config: migration preserves queued task contracts; no feature acceptance)
## Evidence
- Commits:
- Tests: root independent: stock Go1.27.1 -C tools/gomad3 test -tags test_dep -count=1 ./record ./choice ./hostexec ./hostfs (exit0), root independent: stock Go1.27.1 test -tags test_dep -count=1 ./common/persistence/sql/sqlplugin/sqlite ./cmd/tools/lintcode (exit0), root independent: stock Go1.27.1 -C tools/gomad_wasm test -tags test_dep -count=1 -run '^(TestCaptured|TestEnvironment|TestNamespace|TestExploratory|TestCapacity|TestPathOpen|TestOffered)' ./wasi (exit0), make -C tools/gomad_wasm test-go GOMAD_WASM_STOCK_GO=<installed stock Go1.27.1> (exit0; long qualification explicitly skipped), unchanged TestTemporalGuestAdmission/sqlite with installed guest Go1.27.0 (exit0), make temporal-server-wasi (exit0), make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT_FIX=false with pinned lint tools (exit0 after review fix), WASI SQLite lint and errortype vet with -tags test_dep,sqlite3_dotlk (exit0), make -C tools/gomad3 validate and portable architecture/source-set checks (exit0; not native qualification), flowctl validate --spec fn-151 --json (valid, zero warnings), git diff --check (exit0), recoverable checkout rename, git worktree repair and unchanged HEAD/status/diff/inode checks (retirement-verification.log)
- PRs: