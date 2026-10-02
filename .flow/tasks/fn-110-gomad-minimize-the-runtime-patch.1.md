---
satisfies: [R1]
---
# fn-110-gomad-minimize-the-runtime-patch.1 Record the patch, overlay, and qualification baseline

## Description
Record the comparable baseline that every later size and equivalence claim is measured against, before any patch or overlay edit. This task changes no toolchain input; it only adds an "Implementation baseline" section to the investigation document and keeps raw outputs in gitignored scratch.

**Size:** S
**Files:** `docs/research/gomad/GOMAD_PATCH_SIZE.md`
**Touches:** [docs/research/gomad/GOMAD_PATCH_SIZE.md]

### Approach
- Record the repository HEAD, `git status --short` (must show no changes under `tools/gomad3/toolchain/`), archive SHA-256, and patch SHA-256. Planning-time values to confirm, not assume: patch `3ac420beb1ea52271ff3320a196894a16110b536f9635ce19073e4d772047d0c`, 32,275 bytes, 998 lines; archive `4e408abae126d916b6164627193f2c54f0e3ca1312d693b86db45f862ab238b1`; HEAD `d4d800fb47`.
- Report separately: patch bytes and lines; number of edited upstream files (20 `diff --git` headers, equal to `patch_allowlist`); added and deleted source lines for the whole patch and for `src/runtime/proc.go` (investigation: 341/54 and 205/20); overlay file count, bytes, and lines (planning time: 57 files, 600,724 bytes, 17,086 lines; `overlay/src/runtime/gomad.go` is 45,821 bytes, 1,356 lines).
- Record the source-set inventory: `patch_allowlist` and `overlay_allowlist` from `tools/gomad3/toolchain/version/version.json`, and confirm `make -C tools/gomad3 validate` passes.
- Bring the active toolchain up to date with `make -C tools/gomad3 toolchain` and record the baseline build key from `tools/gomad3/.toolchain/build-key`. At planning time that file (key `1803c664…`, written 2026-09-29) was older than the patch file's modification time, so a rebuild may occur; check disk first.
- Record the "before" outcomes that task 5 compares against, on this host: `make -C tools/gomad3 test`, the process-simulation test (`TestRootProcessSimulationUsesRunnerTransport`, `integration` tag, in `tools/gomad3/runner/internal/execution`), `make gomad3-integration-test`, `make gomad3-smoke-qualification`, and `make gomad3-qualification` (seeds 11 and 17 in `tools/gomad3integration/qualification/temporal.json`). Keep the JSON reports under `tools/gomad3/.toolchain/fn-110/baseline/`.
- Record current D12/D14 dispositions as stated in `MILESTONES.md` (Open findings, D12/D14 rows) and the expectations in the qualification manifests, verbatim.
- State plainly that the Linux baseline was not run locally and what would supply it (`.github/workflows/gomad3.yml`, `gomad3-smoke.yml`).

### Investigation targets
**Required:**
- `docs/research/gomad/GOMAD_PATCH_SIZE.md` — measurement method ("Qualification and reproducibility")
- `tools/gomad3/toolchain/version/version.json`
- `tools/gomad3/Makefile:140-175` — validate and test tiers
- `Makefile:165-225` — `gomad3*` qualification targets
- `MILESTONES.md:93-110,185-187` — D12/D14

### Quick commands
```bash
shasum -a 256 tools/gomad3/toolchain/runtime/go1.27.1.patch tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz
wc -c -l tools/gomad3/toolchain/runtime/go1.27.1.patch
grep -c '^diff --git' tools/gomad3/toolchain/runtime/go1.27.1.patch
find tools/gomad3/toolchain/runtime/overlay -type f | xargs wc -c -l | tail -1
make -C tools/gomad3 validate
df -h . && make -C tools/gomad3 toolchain && cat tools/gomad3/.toolchain/build-key
make -C tools/gomad3 test
```

### Key context

**Working constraints (apply to every fn-110 task):**
- No `git commit`, `git add`, stash, or worktrees. The user owns commits; leave changes in the working tree and report them. Earlier fn-110 tasks may therefore be uncommitted working-tree changes — do not revert them.
- Host is `darwin/arm64` only. `linux/amd64` gates cannot run here: record every Linux-dependent check as **incomplete**, never as passing. Cross-compilation is not Linux evidence.
- Disk: about 19 GB was free at planning time and each toolchain build directory under `tools/gomad3/.toolchain/builds/<key>` is 2–6 GB. Run `df -h .` before every rebuild and stop if less than 8 GB is free. Only delete build directories that this spec's own intermediate candidates created, once superseded and not referenced by retained evidence. Never delete the baseline key recorded by task 1 or the active key in `.toolchain/build-key`. Pre-existing directories and `make clean-qualifications` need the user's confirmation.
- Patch and overlay bytes feed the build key (`tools/gomad3/toolchain/buildkey.go:48-58`), so every patch or overlay edit yields a new toolchain identity. Never relabel old artifacts.
- fn-105 D12 (Linux replay divergence) and D14 (Darwin `TestSignalWorkflowTestSuiteChasm`) keep their owners and dispositions. Do not edit qualification expectations to get a passing gate.
- Existing comments move with their code, unchanged. Add no allocations, host reads, dependencies, CLI flags, or capability grants.
- Always pass `-tags test_dep`. In testify code use `require`, not `assert`; plain `testing` files keep their existing style.

**Patch editing workflow (governed, no hand-edited hunks):**
1. Extract the verified archive `tools/gomad3/.toolchain/downloads/go1.27.1.src.tar.gz` (SHA-256 in `toolchain/version/version.json`) into a scratch directory under the gitignored `tools/gomad3/.toolchain/fn-110/`.
2. `go -C tools/gomad3 run ./cmd/gomadtool patch-materialize --root="$PWD/tools/gomad3" --source-root=<scratch>/go`
3. Edit upstream files in `<scratch>/go/src`. Do not copy overlay files into the candidate: `changedFiles` in `toolchain/patch_regenerate.go` rejects added source paths.
4. `go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3" --candidate-root=<scratch>/go`
5. `make -C tools/gomad3 generate validate`, then `make -C tools/gomad3 toolchain` (the build runs the archive-based overlay collision check at `toolchain/build.go:176`).

The descriptor requires `patch_allowlist` and `overlay_allowlist` to equal the checked trees exactly (`toolchain/version/descriptor.go:114-160`), so a task that adds an overlay file or empties a patched file updates `version.json` and regenerates in the same task.

## Acceptance
- [ ] `GOMAD_PATCH_SIZE.md` has an implementation-baseline section with HEAD, archive and patch SHA-256, baseline build key, and both allowlists' sizes
- [ ] Patch bytes/lines, edited upstream file count, added/deleted source lines, and overlay bytes/lines are reported as separate figures with the commands that produced them
- [ ] Darwin "before" outcomes for the Gomad gate, process simulation, integration, smoke, and seeds 11/17 Temporal qualification are recorded per workload, with D12/D14 dispositions quoted
- [ ] Any drifting or unavailable input (digest mismatch, missing archive, dirty toolchain tree) is reported as blocking size and equivalence claims
- [ ] Linux baseline is recorded as not run; nothing under `tools/gomad3/toolchain/` changed


## Done summary
Added the "Implementation baseline" section to docs/research/gomad/GOMAD_PATCH_SIZE.md and recorded evidence; no file under tools/gomad3/toolchain changed and nothing is staged or committed. Baseline: patch 32,652 bytes / 1,007 lines / 20 files / 71 hunks / +342 -55 (proc.go +205 -20), SHA-256 950063a8…; overlay 57 files / 601,516 bytes / 17,105 lines; build key 8d28bd44…; archive SHA-256 equals version.json; the checked-in patch reproduces byte-for-byte as the -U3 diff of the verified archive (baseline -U1 reference 24,620 bytes / 715 lines). Delta against the investigation (+377 bytes, +9 lines) is the fn-105.14 D14 lock_spinbit.go hunk.

Reused (identity-verified against identity.json): fn-108.7 gates for `make -C tools/gomad3 test`, gomad3sim, integration, smoke 4/4, core 7/7, Temporal 28/28 on seeds 11 and 17 with 56 exact replays. Run fresh: validate (exit 0), toolchain (no rebuild), test-runtime per case (4,245/4,245 pass), test-upstream per case (2/2 pass), upstream crypto/rand and syscall (pass), process simulation (25 runs).

Findings the conductor must act on:
- RED baseline: TestRootProcessSimulationUsesRunnerTransport exits 1 in 25/25 runs. TestScenarioChoicePlanRejectsChangedDecisionBeforeSelection and TestProcessExplorationConsumesExternalPlanAndPublishesRecord fail every run (also with HEAD versions of the fn-108-modified closure files); 2/25 runs add one 30 s watchdog timeout in a process-backend subtest. Tasks 2 and 5 name this test as a passing gate; they must compare per subtest. Not fixed here (out of scope).
- Runner source drift: the reused gates ran on the source the user committed as 38957053f during this task; another session (fn-109) is editing tools/gomad3/runner, tools/gomad3/target and, by the end of this task, tools/gomad3sim/exploration.go and conformance/testdata/README.md. Task 5 needs Runner source equal to 38957053f or a rerun of the before gates with toolchain 8d28bd44.
- linux/amd64: not run; supplied by host-tools-linux and core-linux (gomad3.yml) and functional-smoke-linux (gomad3-smoke.yml).
- The fn-108.7 identity.json had the core and Temporal manifest digests swapped; corrected with a note.
- Phase 5 full gates were not run for this docs-only change: `flowctl gate classify` returned FULL because of the user's commit and another session's in-flight edits in the base range, and a suite run on that moving tree would not test this task.

Tier: implementer per project routing (opus at high); no IMPLEMENTER line in the dispatch.

stage: impl-review - ran (raw codex bridge on working-tree files; commits forbidden) (model: gpt-5.6-sol) - NEEDS_WORK, NEEDS_WORK, SHIP
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: make -C tools/gomad3 validate (exit 0), make -C tools/gomad3 toolchain (exit 0, no rebuild, key 8d28bd44), sh .flow/artifacts/fn-110-gomad-minimize-the-runtime-patch/task1-measure.sh (exit 0), go run ./.toolchain/fn-110/baseline/casereport --mode=test-runtime (4245/4245 cases pass), go run ./.toolchain/fn-110/baseline/casereport --mode=test-upstream (2/2 cases pass), .toolchain/bin/go test -tags test_dep -count=1 crypto/rand syscall (exit 0), .toolchain/bin/go test -count=1 -v -tags test_dep,integration -run '^TestRootProcessSimulationUsesRunnerTransport$' ./runner/internal/execution (exit 1 in 25/25 runs: baseline red, 2 of 11 subtests fail; recorded, not fixed), REUSED fn-108.7 final gates on toolchain key 8d28bd44: make -C tools/gomad3 test, gomad3sim, make gomad3-integration-test, smoke, core, Temporal qualification (all exit 0), NOT RUN: Phase 5 full gates for this docs-only change; flowctl gate classify returned FULL due to concurrent non-task paths in the base range, NOT RUN: any linux/amd64 gate
- PRs: