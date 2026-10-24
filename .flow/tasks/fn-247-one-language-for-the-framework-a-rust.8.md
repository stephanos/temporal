---
satisfies: [R4, R5]
---
# fn-247-one-language-for-the-framework-a-rust.8 Consolidate required Model validation and timing into one canonical pipeline

## Description
Deliver the user-approved canonical validation savings (R4, R5) after completed .1 and before the supervisor port. Reuse the existing timing sequence; the later final task consumes this target after all ports integrate.

**Size:** M
**Files:** `Makefile`, `model/scripts/time-gate.sh`, `model/scripts/test-time-gate.sh`, focused orchestration controls if needed, `model/README.md`, `model/GUEST_BUILDS.md`, `AGENTS.md` Important Commands only.
**Touches:** [Makefile, model/scripts/time-gate.sh, model/scripts/test-time-gate.sh, model/scripts/test-model-validate.sh, model/README.md, model/GUEST_BUILDS.md, AGENTS.md]

### Approach

- Add one phony `model-validate` target using the existing `model-time`/`time-gate.sh` sequence. Prefer recursive `$(MAKE) UMPIRE_BACKENDS=require model-time` or equivalent minimal wiring that forces required mode even against incoming command-line/MAKEFLAGS skip. Do not duplicate the Model stage recipes or introduce a generic cache/report framework. Preserve independent Apalache propagation.
- Keep `model-check` and `model-time` public behavior available, including normal default/skip selection and user-selected required modes. Make the canonical invocation's printed command/modes honest; report required timing separately from historical default timing. The existing timer still runs one full model-check followed by one touched-checker test benchmark.
- Exercise the actual new Make entrypoint and existing helper with disposable fake gate/benchmark children. Use a task-owned temporary inner-source fixture and fake tools, so controls never touch production source or perform a hidden full gate. Assert exactly one gate and one benchmark on success, enforced required mode for absent/environment/command-line/MAKEFLAGS conflicts, Apalache mode propagation, and unchanged legacy target mode selection.
- Retain existing direct step controls for exits0/1/37, both child diagnostic texts, cleanup and soft-warning thresholds. Add whole-wrapper/Make controls for gate failure stopping before benchmark, benchmark failure, diagnostics and established Make status2 versus direct step37. Successful soft overruns warn while failed children remain failures; fake controls must be labelled as controls, not execution receipts.
- Preserve default/skip evidence with `skipped_backend_requires_selected_complete_goldens` plus focused actual generic/Temporal fixture entry-mode checks for UMPIRE_BACKENDS absent and explicit skip. These checks must confirm no backend execution while retaining selected complete-golden validation. `test-capabilities.sh` is unrelated filesystem/clock/thread coverage. Use the existing entrypoints or an equivalently direct mode control; do not weaken fixtures to pass.
- Run the ONE real new canonical `make model-validate` pipeline after focused controls, recording every current stage, required TLC execution/mutants/goldens/live host coverage, one benchmark, exact status/diagnostics and measured durations. Do not run separate full default/required/time validations. Final .3 later adds required Apalache within its one integrated canonical run. No guest/golden/hash/package changes or acceptance refreshes are permitted.
- Update only current command guidance in Model README/GUEST_BUILDS and the Important Commands pointer. Preserve historical timing receipts and legacy commands. Read the writing-for-agents skill before editing AGENTS; keep its update to this delivered command. Root owns milestone bookkeeping and holds it unchanged during validation.

### Investigation targets

**Required**:
- `Makefile:1271` — shared stage wrapper, complete gate and legacy timing target.
- `model/scripts/time-gate.sh:11` — step status/diagnostics, full gate at40 and benchmark at47-50.
- `model/scripts/test-time-gate.sh:5` — existing exit/diagnostic/cleanup/soft-warning controls.
- `model/umpire/dev/testing/tests/consumers.rs:172` — selected complete-goldens skip control.
- `model/umpire/dev/fixtures/fixtures.rs:47` and `model/temporal/testing/fixtures.rs:56` — actual environment-mode selection.
**Optional**:
- `model/README.md:149` and `model/GUEST_BUILDS.md:94` — current timing/command contracts.

### Quick commands

```bash
bash model/scripts/test-time-gate.sh
cd model && mise exec -- cargo test -p umpire-testing --test consumers skipped_backend_requires_selected_complete_goldens -- --exact
make model-validate
```

Add the actual focused orchestration/mode-control invocation to evidence after implementation. A newly introduced `test-model-validate.sh`, if used, must be wired into the existing timing-controls entrypoint so the ordinary gate executes it. The final command is the sole real full pipeline for this task; run it only after focused controls pass.

## Acceptance
- [ ] The actual canonical entrypoint forces TLC-required mode through environment/command-line/MAKEFLAGS conflicts and records one full gate plus one existing benchmark; legacy command behavior and independent Apalache propagation remain covered by focused actual-entry controls.
- [ ] Focused fake-child controls prove invocation counts, gate fail-fast, benchmark failure, both diagnostic texts, temporary-output cleanup, direct step exits0/1/37 and GNU Make failure mapping, with soft-warning thresholds preserved.
- [ ] Existing skipped-consumer goldens control plus focused actual absent/skip entry-mode coverage pass without backend execution or dropped golden checks. Filesystem/clock/thread capability coverage remains intact and is not counted as mode coverage.
- [ ] ONE real `make model-validate` run executes all current gate stages and required TLC semantic/mutant/live checks, then one benchmark; actual commands/modes/statuses/stages/platform/timings are retained. Failed or skipped required work never produces a success receipt, and required timings are not relabelled historical default timing.
- [ ] Current command guidance identifies the delivered canonical target while preserving legacy commands/history. No Model/protocol/Go/guest/golden/hash/package changes occur; no source milestone bookkeeping changes during validation.


## Done summary
Added `make model-validate`, reusing the existing timer for exactly one TLC-required gate and one edited-checker benchmark. Required mode survives environment, command-line and MAKEFLAGS conflicts; legacy modes, independent Apalache selection, failure diagnostics/statuses, cleanup and soft warnings retain focused controls. Current command guidance is updated, README is 999 lines, and no Model/protocol/Go/guest/golden/hash/package or milestone content changed.

baseline: green (existing timing controls and selected-complete-goldens consumer test). New actual-Make/timer controls failed first for the absent target, then passed; the ordinary timing-controls stage executes them. Actual generic and Temporal fixture entry tests passed with TLC absent and explicitly skipped, with Java resolution/execution sentinels untouched. Initial wrong-directory/lost-handle observations and the absent external timer were inconclusive and are recorded honestly in the durable evidence.

The ONE real `BASH_ENV=/dev/null make model-validate` invocation exited 0 at executable commit `3e38c6080db279521afb1267f3964f96cdd9f9f9`: all 19 existing stages, required TLC agreement/semantic/mutant/selected-golden checks, wasm/guest and all three Go host parity stages, followed by one benchmark. Gate 1007.800 s, benchmark 9.272 s, wall 1017.380 s; the 60 s gate target warned and the 10 s benchmark target did not. Required timings are separate from historical defaults. Retained workspace output totals 720 passed, zero failed and four existing ignored controls across 177 suite summaries. Shared fixtures: 47 passed; Temporal fixtures: 33 passed/one ignored. Linux/aarch64, GNU Make 4.4.1, Rust/Cargo 1.99.1, Go 1.27.0 and Temurin 27 were observed. No native macOS execution is claimed. Apalache remains independently opt-in; final task .3 retains its combined-backend run.

Durable stage/test/mode/tool/hash evidence and review receipts are under `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task8/`. The guest remains SHA-256 `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`. After measurement only current-command prose and Flow evidence changed; executable-byte identity against the measured commit, shell syntax and diff checks passed. The classifier reported FULL/unmatched, so no docs-only skip or repeated full run is claimed; the recorded full receipt remains tied to the measured executable commit.

Three explicit `codex:gpt-6.1-sol:high` draws returned SHIP; dispatcher and finalizer each exited 0. Reviewers are from the same model family as the writer. Read-only reviewer sandboxes blocked mktemp runtime reruns; they checked shell syntax, source contracts and retained passing evidence. The primary freeze was released only after both commands terminated. No source worktree or remote operation was created.

stage: impl-review - ran (three SHIP draws; finalizer SHIP; task8/review-receipt.json)
Tier: session (jev-unavailable(no_key)); AGENTS implementer pin applies. One judge dispatch already recorded in .flow/tmp/fn2478-judge.json; no retry.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3e38c6080db279521afb1267f3964f96cdd9f9f9, d5ccc400dc0c67109273ccae0a69d04f38a986f4
- Tests: BASH_ENV=/dev/null bash model/scripts/test-time-gate.sh, BASH_ENV=/dev/null bash model/scripts/test-model-validate.sh, cd model && mise exec -- cargo test -p umpire-testing --test consumers skipped_backend_requires_selected_complete_goldens -- --exact, cd model && env -u UMPIRE_BACKENDS mise exec -- cargo test -p umpire-fixtures -p umpire-temporal-tests --test fixtures fixtures -- --exact --nocapture, cd model && UMPIRE_BACKENDS=skip mise exec -- cargo test -p umpire-fixtures -p umpire-temporal-tests --test fixtures fixtures -- --exact --nocapture, BASH_ENV=/dev/null make model-validate, bash -n model/scripts/time-gate.sh model/scripts/test-time-gate.sh model/scripts/test-model-validate.sh, git diff --check
- PRs: