---
satisfies: [R2, R3, R4, R5]
---
# fn-112-make-the-standalone-activity-scala.7 Migrate queue compositions and shared properties without string keys

Touches: [model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/Claims.scala, model/lifter/test/**, model/lifter/testdata/**, model/ir/**, model/cases/**]

## Description
Apply typed composition/member/sync references and shared claims to dispatch queue and both composition families.

**Size:** M
**Files:** standaloneactivity System/Claims roots and focused feature/lifter tests.

### Approach
- Derive queue providers and composed machines with rebind/extend/withMember so each steps/sync set is declared once.
- Replace actionKeys, whenAction and fact/action key literals with typed own/synced/records selectors.
- Declare notAdmittedWhilePaused, atMostOneActive and terminalStays once, as the parameterized top-level defs task 4 shaped (spec R4): each takes the model and the state-dependent parts as function parameters (`paused`, `running`, `terminal`), never a closure over `AdmissionState`, and is written with the claim patterns (`never(…).from(…)`, `never(…)`, `once(…).keeps(…)`). Attach the one definition to the admission machines, passing the record's named status sets, and to both composition families, passing named predicates over the member projection (`_.activity`); a lambda literal is not accepted as a def argument (task 4's refusal), so the compositions' predicates are named defs beside their vocabulary. Each instance keeps its frozen name through the explicit-name form; the `*.any.*` Queries keep their computed spelling. These three defs are the first law bodies `fn-122-capabilities-and-their-laws` generalizes; they move under `temporal/laws` there, not here.
- Exercise ambiguous member/action spellings and sync qualification while retaining exact original composition keys and order.
## Acceptance
- [ ] No two feature machines share a steps list, each composed state type has one sync declaration set and both composition families derive by member replacement.
- [ ] actionKeys, whenAction and fact/action string keys are absent from the feature Model.
- [ ] Each of the three shared properties has one source definition, a parameterized top-level def written with the claim patterns, and produces the original Property rows and answers on the admission machines and both composition families; no closure over `AdmissionState` and no lambda-literal def argument remains.
- [ ] Task-1 full equivalence and focused feature/lifter tests pass.
## Done summary
Migrated the standalone activity's queue providers, both composition families and the three shared admission claims to the fn-112 DSL. No string keys are left. Tables, Definition IDs, Property rows, Query answers/totals and Case bytes are unchanged, and model/cases is byte-identical. Commit 6f6387e41a.

**What changed (System.scala)**
- **Queue providers:**
  - `dispatchQueueUnderStorageLoss = dispatchQueue.extend(storageLoss ~> storageLossView).assuming(storageLossAssumed)`.
  - `lossyMatchingQueue = matchingQueue.extend(...).refining(dispatchQueueUnderStorageLoss)(viewOf).assuming(storageLossAssumed)`.
  - `forgetfulQueue` and `volatileQueue` are `matchingQueue.rebind(crash ~> ...)`.
  - No two machines share a steps list.
- **Compositions:**
  - Each state type has one typed `compose[...]` with one set of syncs: `currentOverQueue`, and `currentOverMatching` with `.replaces(_.queue, dispatchQueue)`.
  - The other five compositions are derived with `withMember`. The lossy one replaces `dispatchQueueUnderStorageLoss`, which comes from its own refinement.
- **Scenarios:** use `c.synced(_.activity -> ...)`/`c.own(_.member, ...)` in place of `actionKeys`.
- **failedCommitKeepsTheMessage:** uses `after.records(_.activity, fact) implies (...)`.
- **Shared claims:** `notAdmittedWhilePaused(m)(paused, running)`, `atMostOneActive(m)(twoActive)` and `terminalStays(m)(terminal, phase)` are each one top-level def over `Declares[S]`, written as `never(...).from(...)`, `never(...)` and `once(...).keeps(...)`.
  - The admission designs pass the record's status sets in `object Admission`.
  - The compositions pass the same five sets as `OverQueue`/`OverMatching` companion defs over `_.activity`.
  - Frozen names come from the explicit-name form. `*.any.*` Query names are still computed.
- **Removed:** `admitsWhilePaused`, `leavesTheEnd`, `idleOverQueue`, `idleOverMatching`.

**Outside System.scala**
- **Members.scala:** dropped its production twins. Production now uses the typed forms, so comparing against the twins would be typed against typed. It keeps the cross-entity `synced(_.worker -> serve)` claim and the separator/alike-spelling switch section.
- **Fixtures.test.scala:** the Members test now pins production directly: members `[activity, queue]`, syncs `[dispatch, admit, settle]`, the replacement target of all 7 compositions, and the exact original keys of all 18 keyed Scenarios.
- **quint_test.go:** the Property-negation mutant now names `staleAdmission.property.atMostOneActive`, because each instance owns its function now. It is skipped here because quint is not installed.

**Decisions (autonomous)**
- **Status sets live on objects:** `Admission`, plus companions of `OverQueue`/`OverMatching`. Task 8 settles the final vocabulary objects (R10).
- **Parameter names:** `paused`, `running`, `terminal` (spec R4); `twoActive`/`phase` per fn-112.4.
- **heldAdmission is unchanged.** Its steps list is distinct, and `restrict` would drop its monitors and refinement.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1: SHIP.

**Deferred P3/FYI**
- The two companions repeat five one-line forwarders over `_.activity`. This is forced by the lambda-literal refusal; fn-122 can collapse it.
- `providerQueries`' `committedStays` and `delivers` are still per-def lambdas. They are for task 12 (`queueLaws`).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6f6387e41a
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; model/cases byte-identical), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), scala-cli test model/lifter (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: