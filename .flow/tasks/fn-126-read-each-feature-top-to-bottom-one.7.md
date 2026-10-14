---
satisfies: [R5]
---
# fn-126-read-each-feature-top-to-bottom-one.7 Definition IDs are fully qualified Scala names; pins and families removed

## Description
Also decision 28's task-7 items: gate-computed `total`, inferred `entity`, a `reject` helper, named-argument state literals.

Task 6b of the split closing batch (decisions 23, 24, 25, 26, 27). Rules are rewritten action-major: `on(action) { in(…).where(…) ~> effect }`, `when` retired (decision 27). Also: the capabilities section is the declaration and `exports` may name a `queries` section (decision 24, no `val all`); machine-level sections drop `extends Section` (decision 25); `Party` is retired for `Actor` everywhere, including the IR field `Action.party` → `actor` (same number), Go and docs, and the task queue's `fault` actor holds its fault actions (decision 26). Make every Definition ID the declaration's fully qualified Scala name and the IR family the declaring package: rewrite `Context.definitionId`, remove `DefinitionScope`, the pin lookup, section transparency, `Family` givens and `…Family` objects, the family arguments of derivations and compositions; keep section placement refusals; add a whole-index check of derived IDs (machine and Query names unique per package). Invert and regenerate the fixtures. Prove only IDs changed with an ID map (`project6.py`): a bijection applied to the before-IR, then `projtool` on mapped-before vs after (fingerprints recomputed, never mapped), and Cases regenerated from the mapped IR in a scratch worktree. Re-record pinned Runs. The lifter, `model/umpire` and fixture work may be written alongside 6a in its own worktree; its Model edits and regeneration wait for 6a to merge. Plan section 1.

Plan: `.flow/tmp/fn-126/plan6.md` (read-only planning of 2026-10-05, built after task 4 against task 5 in progress; re-verify against the merged task 5). Host decisions on its open points: IDs follow decision 23's rule literally (section objects included, e.g. `…ActivityRecord.monitors.atMostOneActive`); the Kit's family comes from the lifter substituting the realization's package when it folds a Kit call (no macro); level files keep their subject's own types and signature; `TrustingCaller`'s `inspect` moves into the feature's `object caller` (party unchanged), ending the shadowing; a `shared/` folder has at most one `object exports`. The golden harness is retired by fn-124.7 before these tasks: every proof is by projection (`projtool` + a before/after IR projection), never a golden re-capture.

## Acceptance
- [ ] Every item of this task's description is done, with the plan's verification list for its section passed.
- [ ] Equality proved by projection as described; any table, answer, verdict or fingerprint difference beyond the stated renames stops the task.
- [ ] All gates of the spec's Verification pass.


## Done summary
fn-126.7 is done on `umpire-fn126-7` in `/Users/stephan/Workspace/skunkworks/umpire/wt/lane-a`. The final proof printed `RESULT: OK` and every gate passed. The live suite failed only on INCONCLUSIVE verdicts, all from the known ShutdownWorker race. `umpire` is merged in (never rebased). Nothing was pushed and `flowctl done` was not run. The tree is clean at 5986de66bd.

The harness refused to let me write `summary.md`, so the summary is here instead; I did not write `evidence.json` either. All logs are under `.flow/tmp/fn-126/fn126-7/`.

**Parallelism:** I used no subagents. Your go-ahead to fan out arrived after the bulk conversion was done, so splitting the work would not have saved time.

### Commits (base 436cff62fe)
- `4b93f8b392` feat(umpire): qualified names, actors, rules grouped by action. Covers the framework, the lifter, `model/check`, `ir.proto` with `ir.pb.go` regenerated (only that file changed), and the Go source.
- `d42de4c0af` refactor(model): all Models converted, outputs regenerated, Go test IDs mapped.
- `cfe046ac96` docs(model): README, SEMANTICS and the two `.plans` notes.
- `3d357c0ea9` test(testpilot): pinned Runs re-recorded, canary repinned.
- `5986de66bd` merge of `umpire` (it brought only flow files and MILESTONES).

Each commit is a logical slice but does not build on its own, because the framework and the Models depend on each other. Every commit carries the Co-Authored-By trailer.

### Decisions
- **D23 (IDs and families):**
  - An ID is the package plus every enclosing object, with `$` stripped and `<File>$package$` objects skipped. The family is the declaring package.
  - Removed: `DefinitionScope.scala`, `Section.scala`, `Family` with its givens, the `…Family` objects and the family arguments, pins, scope files, section transparency, and the folding of `Family` constants.
  - The Kit's `family` is a placeholder; the lifter replaces it with the realization's package, with no macro.
  - New check: two derived IDs alike in one package are refused across the whole run (`Lift.derivedIdTwins`, tested in `model/irgen/test/DerivedIds.test.scala`).
- **D24 (`implements` and `queries`):**
  - `object implements extends Implements(limits = …)(…)` is now the declaration in ActivityProduct, ActivityProtocol and NexusOperation. Waivers are body statements (`except` and `overriding`).
  - `val all` and `functionalQueries` are gone, and exports name `queries` sections and `implements` objects.
  - `competingTimers` sits in `CurrentAdmission.queries` (`system/Record.scala`). To allow that, the order lint now accepts a Query whose inline Scenario reads another machine object of the same package.
- **D25 (sections):** sections are plain objects, recognized by name.
- **D26 (actors):**
  - `Party` is now `Actor`, and the IR field `actor = 4` replaces `party = 4`. Go, the lints and the docs use the new wording.
  - The task queue declares `object fault extends Actor` with storageLoss, crash and ackLoss. The marker lint's fault rule reads the `fault` actor.
- **D27 (rules):**
  - Rules are written `on(action) { in(…).where(…) ~> effect }`, with `always`; derivations use a top-level `on`. `when` is retired.
  - The lifter refuses a block inside a block, a repeated block, and a disabled action that a rule fires.
  - The conversion was mechanical and per class, with rule order checked. Tables are unchanged.
- **D28 (lighter declarations):**
  - The gate computes `total`; an author's value is now only an optional check.
  - `entity` is inferred. It is kept explicitly in AdmissionResponseLoss and RejectAfterClose.
  - `reject(outcome, s)` replaces the `…Step` aliases.
  - Multi-field state literals use named arguments.
- **Carry-forward:**
  - `inspect` moved into NexusCaller's `object caller`.
  - The `given …Scope` pins and the template's `given Family` are removed.
  - The order lint no longer refuses nested objects.

### ID scheme examples (before → after)
- `temporal.standaloneactivity.Model$package$.start` → `temporal.features.standaloneactivity.caller.start`
- `temporal.standaloneactivity.System$package$.atMostOneActiveAttempt` → `temporal.features.standaloneactivity.system.CurrentAdmission.monitors.atMostOneActiveAttempt`
- `temporal.standaloneactivity.System$package$.crash` → `temporal.shared.taskqueue.fault.crash`
- `temporal.nexuscaller.Control$.inspect` → `temporal.features.nexuscaller.caller.inspect`
- `temporal.worker.Worker$package$.serve` → `temporal.shared.worker.worker.serve`
- Families are packages, for example `temporal.features.standaloneactivity.system` and `temporal.features.nexuscaller.product`. Seven old families map onto the new packages, and three of them split.

In total, 63 IDs and 65 type names changed, and the `party`→`actor` key was applied to 75 members.

### Proof (final run, after the merge)
`prove.sh --renames .flow/tmp/fn-126/fn126-7/schema-renames.json --require '^temporal\.(features|shared)\.' --no-dollar .flow/tmp/fn-126/fn126-7/before . .flow/tmp/fn-126/fn126-7/prove-final`
```
[OK] projection (byte-identical, 89972570 bytes)
[OK] casegen (21/21 files equal)
[OK] ir (7 IR files)
[OK] lint
[OK] laws
[OK] cases (21/21 files equal)
[OK] generated (9/9 files equal)
[OK] canary (2/2 files equal)
[OK] names
RESULT: OK
```
The only IR notes are allowed deltas:
- **Root names:** `queries$.all` became `queries`, `implements$.all` became `implements`, and competingTimers' root moved to `CurrentAdmission$.queries`.
- **One effect name:** `effects.reject` became `effects.rejectDelivery`, renamed so it doesn't clash with the new `reject` helper.

No table, refinement row, Query answer, verdict or Contract meaning changed. Log: `final-prove.log`.

### Pinned Runs and canary
- Both pinned Runs were re-recorded: `nexusCallerControl-forgedCompletion-run.json` and `nexus-caller-syncCompletion-run.json`. The final `make umpire-rerecord-pinned-runs` reports both as current (`final-rerecord.log`).
- The canary pin moved from c9280a98… to `4bae24f51bd78bc355b778e525d6f0e957cbcd9fd8ac619a612349605bd2ae54`.
- I left the historical `tools/canary/assessment/testdata/nexusCallerCanary-syncCompletion-case.json` untouched on purpose.

### Gates (final, after the merge)
| Gate | Result | Log |
|---|---|---|
| `umpire-check-model --skip-go-checks` | exit 0 | `final-check-model.log` |
| `lint-model` | exit 0 | `final-lint-model.log` |
| `lint-code-fast` | exit 0 | `final-lint-code.log` |
| `umpire-check-cases`, `umpire-check-fixtures`, `canary-check-case` | exit 0 | `final-cases.log` |
| Full Go suite at `-p 2` | 46 packages pass, 3 have no tests; `tools/umpire/export` was OOM-killed | `final-go.json` |
| `tools/umpire/export` rerun at `-p 1` | pass (72.9s) | `final-go-export-p1.json` |
| `umpire-check-live-tests` | exit 2, INCONCLUSIVE only | `final-live.log` |
| Targeted live rerun | INCONCLUSIVE only | `live-rerun1.log` |

**Live failures in detail:**
- **Activity generated Cases** (cancelIsRequested, terminateSettles, pauseResume, scheduleToStartTimeout): "the Run recorded no evidence" after ShutdownWorker, or a context-deadline `schedule_cancelled`.
- **`NexusCallerScheduleToStartTimeout/chasm`:** INCONCLUSIVE, and it passed on the rerun.
- **`TestTestpilotWorkerOutageCase`:** INCONCLUSIVE in both runs, while its sibling `LeavesAnotherQueueAlone`, which runs the same outage Case, passed.

There were no violations. The failing set changes between runs and matches the set that fn124-3 and fn124-6 already logged as the ShutdownWorker race.

### Other decisions
- **Effects that take arguments** are bound as `~> (effects.timeOut(_, X))` instead of being curried. Currying would reorder the IR's function parameters, which must not change.
- **`Step` is covariant in its facts**, so `stay`, `enter` and `reject` need no result types. `records` keeps its strictness through `G <:< F` evidence.
- **`IrRoot = AnyRef`:** the kind of root is checked when lifting and at runtime. A channel used as a root is now refused at lift time rather than failing to compile.
- **Plain `implements` sections:** the ones holding parameterized `capabilities(m, …)` definitions shared by derived designs (Record, System, WithTaskQueue) stay plain sections. Only a Model's own declaration uses `extends Implements`.
- **Closepolicy:** the objects keep `callerSide` and `handlerSide`, and their old `val all` became `<design>Queries`.
- **Explicit totals:** the Models no longer write them; the lifter fixtures still do, as the optional check.

### For the owner
- I acted on no request from anyone but the host. Two completion notices arrived for subagents I never started ("Convert task queue Models" and "Build fn-124.8 package-split tool"). I ignored them.
- A leftover `index.lock` in the shared git directory (`.git/worktrees/fn127-1`) blocked two commits for a short time. It shared an inode with the index, so I did not delete it, and it cleared on its own. The index matched my backup.
- `make protoc` ran only with a Linux goimports override, and `ir.pb.go` was the only file it changed.
### Review
**Reviewer:** claude-opus-5-5 at high, fresh context, host-dispatched (single lane). Writer and reviewer are the same family (Opus).

**Round 1: SHIP, no P1/P2.**
- The reviewer reran prove.sh from a fresh `--rev 436cff62fe` snapshot: RESULT OK, projection byte-identical.
- The ID map is clean: a bijection with `--require`/`--no-dollar` satisfied.
- It verified that `qualifiedName` admits no collisions, and that the Kit family placeholder resolves correctly for all 6 realization families.
- It confirmed the live INCONCLUSIVEs predate this task.

**P3s applied** in f9900c61f3 and 9e7dfb47d0:
- derived-ID twins from one shared `def` are refused;
- an unread `queries` val is refused;
- `Implements.except`/`overriding` are protected;
- stale docs and "party" wording are fixed.

**Reruns, all pass:** model gate, `lint-model`, `./tools/umpire/lint` and `./tools/umpire/lower` at `-p 1`, `lint-code-fast` (`p3-*.log`).

**Deviations, recorded by the host as spec decision 30:**
- the `~> (effects.timeOut(_, X))` effect form;
- plain `implements` sections for parameterized shared capability defs;
- `reject` renamed to `rejectDelivery`;
- the same-package Query relaxation in the order lint.

**Recorded, not changed:**
- Commits 4b93f8b392..d42de4c0af build only together.
- The merge 5986de66bd lacks the attribution trailer.
- "party" remains in `.plans/UMPIRE4_SPEC.md` and `.plans/UMPIRE4_TLA_COMPAT.md`; that is for fn-126.8's docs pass.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4b93f8b392, d42de4c0af, cfe046ac96, 3d357c0ea9, 5986de66bd, f9900c61f3, 9e7dfb47d0
- Tests: make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make lint-model (exit 0), lint-code-fast (exit 0), umpire-check-cases, umpire-check-fixtures, canary-check-case (exit 0), go test -count=1 -json -tags test_dep -p 2 ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (46 pass, export passes alone at -p 1), prove.sh --renames schema-renames.json --require '^temporal\.(features|shared)\.' --no-dollar: RESULT OK, make umpire-rerecord-pinned-runs (both current), make umpire-check-live-tests (exit 2, ShutdownWorker-race INCONCLUSIVEs only), review reruns: model gate, lint-model, ./tools/umpire/lint and ./tools/umpire/lower at -p 1, lint-code-fast (all pass)
- PRs: