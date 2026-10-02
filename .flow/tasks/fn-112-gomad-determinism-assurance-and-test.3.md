---
satisfies: [R3, R4]
---
# fn-112-gomad-determinism-assurance-and-test.3 Record a runtime-state digest at each choice point in a diagnostic trace

## Description
Runtime and wire half of the localiser (R4): an opt-in diagnostic trace record carrying a state digest at every choice point. This is the spec's early proof point. Runner plumbing and the differ are task 4.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`, `tools/gomad3/choice/schema/choicewire.json` and templates, generated `wire_generated.go` (host and overlay), `tools/gomad3/toolchain/version/version.json` if a new overlay file is added
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/choice/schema/**, tools/gomad3/choice/internal/wire/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/**, tools/gomad3/toolchain/version/version.json, tools/gomad3/internal/gomadtool/generation/protocol/**]

### Approach
- First step (R3): re-anchor assessment finding Q1 against the current record struct and mark it confirmed, changed, or refuted.
- Reuse the choice append path; add a separate record kind on its own inherited descriptor with its own byte bound, read at env intake beside the existing choice variables.
- Digest fields: seeded draw counters, allocation count, GC cycle and phase, virtual time, run-queue length. Use only state the overlay can read without editing a collector file; drop a field that needs one and say so.
- Recording must not allocate on the Go heap or draw from the seeded stream.
- Define the layout in the wire schema and regenerate with `make -C tools/gomad3 generate`; never hand-edit generated files.
- Add an overlay-only, diagnostics-only environment switch that perturbs one draw at a chosen ordinal, for the fixtures in tasks 4 and 5. No new patch hunk.

### Investigation targets
**Required** (read before coding):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:313-347` — record struct and append path
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:199-253` — env intake for trace descriptors
- `tools/gomad3/choice/schema/choicewire.json` — wire schema
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:344-357` — codec generator

**Optional** (reference as needed):
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:693-743` — replay divergence reporting
- `tools/gomad3/toolchain/runtime/go1.27.1.patch:614-644` — runtime rand helpers whose counters the digest reads

### Key context
- Choice records are fixed-size in one mapping capped at 64 MiB; the diagnostic trace must not share that budget.
- fn-110 tasks 2-4, fn-109 task 13, fn-114 C2/E3/E4, and fn-105 D26 edit the same files (spec Open Questions 2). Check their state before starting and rebase onto whichever landed.
- Any runtime edit changes the toolchain build key and the choice implementation digest, both part of execution evidence. Cross-build comparison therefore uses the behavioral projection in the acceptance list, never raw evidence digests.
## Acceptance
- [ ] Finding Q1 re-anchored with file and line, marked confirmed, changed, or refuted
- [ ] With diagnostics on, every choice point emits a digest record with the listed fields; dropped fields are named with the reason
- [ ] With diagnostics off, the core qualification set matches the pre-change toolchain on a behavioral projection: stdout and stderr hashes, I/O transcript hash, World identity, outcome, virtual time, peak goroutines, and choice-trace decision content. Toolchain build key, choice implementation digest, and identities derived from them are the allowed differences, and the new identities are checked to be recorded correctly
- [ ] Within the new toolchain identity, canonical bytes with diagnostics off are identical with and without the diagnostic code path compiled in use (flag absent versus never requested)
- [ ] A workload that qualifies with diagnostics off also qualifies with them on
- [ ] Recording performs no Go-heap allocation and no seeded draw, shown by a test
- [ ] Diagnostic-trace overflow stops the target with a typed failure
- [ ] `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` pass on darwin/arm64; linux status recorded
## Done summary
The overlay runtime now writes a runtime-state digest for every choice record into an opt-in diagnostic trace on its own inherited descriptor, and the choice wire schema defines that trace's header and record. Nothing changes when the trace is not requested: the seven core workloads match the pre-change toolchain on the behavioral projection.

**Q1 (R3): confirmed.** At the base snapshot `gomadChoiceRecordValue` (`toolchain/runtime/overlay/src/runtime/gomad.go:313-323`) carried ordinal, kind, flags, alternatives, selected, data, site offset, and two identities, with no stream position, allocation count, or GC cycle. Replay reports a diverging ordinal (`gomad.go:693-743`). Two fresh runs report only a field name (`qualification/qualification.go:394`, `firstDivergence`).

**Contract for tasks 4, 5, and 10**
- Enable with `GOMAD3_DIAGNOSTIC_TRACE_FD` and `GOMAD3_DIAGNOSTIC_TRACE_BYTES`. A choice trace is required. The backing starts with `EncodeDiagnosticHeader(capacity)`; the bound is 160 bytes to 64 MiB (`DiagnosticMaximumBytes`), separate from the choice trace's.
- Profile `gomad3-diagnostic-trace/v1`, magic `GOMADDG\x01`, 64-byte header, 96-byte records. Record N is the digest taken when choice record N was appended, so the two traces pair by ordinal.
- Fields: virtual time, allocation count, GC cycle, GC phase, local run-queue length (with `runnext`), and seven draw counts since process start: run-queue, scheduler (runnext and shuffle), select, runtime rand, runtime cheaprand, timer tie, clock tick.
- Header byte 12 is the state: 0 as written by the Runner, 1 complete, 2 overflow. State 2 means the trace holds fewer digests than the run had choice points, for either cause: the diagnostic trace filled (the target stops, below), or the choice trace filled and digests stopped with it (the target runs on and the trace is closed as overflowed at exit). The two are told apart by the choice terminal frame, which is absent in the first case and reports `overflow` in the second, so no third state was added.
- Overflow sets state 2, prints `runtime: Gomad diagnostic trace overflow`, and exits 125 without a choice terminal frame. Invalid configuration exits 2 with a message starting `runtime: invalid Gomad diagnostic trace`.
- `GOMAD3_DIAGNOSTIC_PERTURB_DRAW=N` takes one extra draw from the process-wide cheaprand stream before the digest of choice record N. It is rejected without a diagnostic trace. No patch hunk was added.
- Host codec: `choice/internal/wire` `EncodeDiagnosticHeader`, `DecodeDiagnosticHeader`, `EncodeDiagnosticRecord`, `DecodeDiagnosticRecord`.

**Fields dropped or narrowed**
- No listed field needed a collector file. The allocation count is read from `memstats.heapStats` and the P's `mcache`, without edits to either.
- Run-queue length covers the P's local queue only. The global queue and `gomadArrivals` change at host-timed moments (an M returning from a Runner syscall), so including them would make the digest differ between runs that agree.

**Acceptance**
- Every choice point emits a digest: `TestDiagnosticTrace/digests_every_choice_point` (190 digests for 190 choice records; two same-seed runs record identical digests). Passed 40 consecutive repetitions.
- Diagnostics off matches the pre-change toolchain: 7 of 7 core workloads, build key `4412b5b2…` against `1c34ea07…`. Stdout and stderr hashes, I/O transcript hash, World digests, outcome, virtual time, peak goroutines, semantic coverage, and choice decision content are equal. The new build key and choice implementation digest are recorded as expected in all seven. Evidence is in `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-3/`.
- Decision content is compared with identities renamed by first appearance. Raw choice bytes differ across builds because the runtime's text grew by 2928 bytes, which moves every select site offset and every identity hashed from one, and so re-sorts selected ranks.
- Same toolchain, off versus on: `TestDiagnosticTrace/recording_leaves_the_run_unchanged` requires equal stdout, stderr, choice record bytes, and choice terminal frame.
- Qualifies with diagnostics on: shown by direct launch only. The core `./basic/concurrency` test binary gave equal output and choice bytes across 10 runs off and 10 runs on. The other six core workloads need the Runner's I/O profile, so their check waits for task 4.
- No heap allocation, no seeded draw: the same subtest. The fixture prints `runtime.MemStats.Mallocs`, select and run-queue order, and map iteration order, and all are equal off and on.
- Overflow: `TestDiagnosticTrace/overflow_stops_the_target`.
- Review finding (P2, fixed): a choice-trace overflow left a truncated diagnostic trace marked complete. `gomadDiagnosticClose` now closes it as overflowed. `TestDiagnosticTrace/choice_trace_overflow_leaves_the_diagnostic_trace_marked_truncated` gives the choice trace room for two records and the diagnostic trace 1 MiB; it failed with state 1 before the fix and passes with state 2.
- I read "flag absent versus never requested" as the control variables being absent; the Runner flag is task 4.

**Gates on darwin/arm64** (all exit 0): `make -C tools/gomad3 validate`, `test-toolchain`, `overlay-test`, `test-runtime` (710 s), and the remaining `test` tiers run one target at a time: `test-harness`, `intercept-test`, `test-host`, `world-test`, `test-builder`, `test-live-capability`, `test-upstream`. Baseline before the first edit was green for `validate`, `test-toolchain`, `overlay-test`, and `test-runtime`. `make lint-code-fast` was not run; `gofmt -l` and `go vet` on the changed packages are clean.

**After the review fix** (build key `f292ea6a…`, all exit 0): `make -C tools/gomad3 validate`, `test-toolchain`, `overlay-test`, and `TestDiagnosticTrace` 20 consecutive repetitions, which includes the diagnostics-off subtest `recording_leaves_the_run_unchanged`. The diagnostic tests live in `overlay-test`; `test-runtime` has no diagnostic case, so it was not rerun. `test-runtime`, the other `test` tiers, and the core projection were last run on `1c34ea07…` and not on `f292ea6a…`; the fix changes only the state byte written at exit when a diagnostic trace is mapped.

**linux/amd64 was not run.** Commands a linux run needs: `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test`, then `make -C tools/gomad3 runner` and `.bin/gomad qualify-set --manifest=$PWD/qualification/core.json --working-dir=$PWD/qualification/corpus --artifacts=DIR --output=DIR/report.json` at the base and at the head, each followed by `project.py DIR/artifacts OUT.json`, then `compare.py BASE.json HEAD.json BUILD_KEY SOURCE_SHA256_HEX OUT.json` (scripts in the artifact directory).

**For the conductor**
- Snapshots: first pass base `6ea20042910b8505fb58725f8a1bf8e59219b5c0`, head `bf31fe284ed8e25a30ab9ae7dc9e128497568b5b`; review fix base `6dbcb0d4549b659beb669aa4871ba32f6a482ad7`, head `3063078dc0504fe344b546d60a52888baccc22fa`. The review fix is uncommitted on top of `8789deab0`.
- The toolchain build key moved from `4412b5b2…` to `1c34ea07…` and, with the review fix, to `f292ea6ac88dbffe5657d0a7dd4e7abe3c72e6b091b1494cfe315bdede03b088`; intermediate `0e1a9041…` and `113ec00b…` builds exist from the two red runs. Artifacts retained under the old key no longer match the active toolchain.
- Files outside Touches: the new test `toolchain/runtime/overlay/src/internal/gomadchoicewire/diagnostic_runtime_test.go` is inside Touches, but two generated files are not: `target/internal/livecap/protocol_generated.go` and `toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go`. `make generate` rewrites both because their digest covers `runtime/gomad.go`.
- Files that changed while I ran and that I did not change: `MILESTONES.md`, the fn-114 spec and task files, and at 22:05 four `internal/compatibilitypack` files for `modernc-libc-xsys-v047-linux-amd64`. `make validate` passes with them. At 22:10 a `wip` commit `8789deab0` by the owner took the working tree, including this task's code. The head snapshot therefore also holds those foreign changes.
- The host needs Go 1.27.1 first on `PATH` for the make targets; Homebrew's 1.26.5 fails `toolchain-build` under `GOTOOLCHAIN=local`. I used the module-cache toolchain `golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64`.
- The Runner does not yet reserve the `GOMAD3_DIAGNOSTIC_*` names in its environment allowlist (`runner/runner.go:1356`); task 4 owns that.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

Review: cross-family codex review over the snapshot range `6ea20042..bf31fe28` returned SHIP with one P2 (truncated diagnostic trace marked complete after a choice-trace overflow), fixed in `6dbcb0d4..3063078d` with a regression subtest. Review output is retained at `.flow/artifacts/fn-112-gomad-determinism-assurance-and-test/task-3/review-r1.md`. Open for task 4: qualification with diagnostics on through the Runner for the six core workloads that need the I/O profile. No commit was made by the worker; the owner's `wip` commit `8789deab0` holds the pre-fix state.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (make -C tools/gomad3 validate; test-toolchain; overlay-test; test-runtime, pre-edit, build key 4412b5b2), make -C tools/gomad3 validate, make -C tools/gomad3 test-toolchain, make -C tools/gomad3 overlay-test, make -C tools/gomad3 test-runtime, make -C tools/gomad3 test-harness, make -C tools/gomad3 intercept-test, make -C tools/gomad3 test-host, make -C tools/gomad3 world-test, make -C tools/gomad3 test-builder, make -C tools/gomad3 test-live-capability, make -C tools/gomad3 test-upstream, .toolchain/bin/go test -tags test_dep -count=40 -run TestDiagnosticTrace internal/gomadchoicewire, .bin/gomad qualify-set --manifest=qualification/core.json (base 4412b5b2 and head 1c34ea07) + project.py + compare.py: match=true 7/7, direct.py concurrency.test 17 10: pass, linux/amd64 was not run, review fix (build key f292ea6a): make -C tools/gomad3 validate, review fix: make -C tools/gomad3 test-toolchain, review fix: make -C tools/gomad3 overlay-test, review fix: .toolchain/bin/go test -tags test_dep -count=20 -run TestDiagnosticTrace internal/gomadchoicewire (new subtest choice_trace_overflow_leaves_the_diagnostic_trace_marked_truncated, red on build 113ec00b before the fix), review fix: test-runtime, remaining test tiers, and core projection not rerun on f292ea6a (last run on 1c34ea07)
- PRs: