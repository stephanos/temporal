---
satisfies: [R2, R5, R6, R7, R8, R9, R14, R15]
---
# fn-112-make-the-standalone-activity-scala.6 Migrate product, protocol and admission models to the new DSL

Touches: [model/temporal/standaloneactivity/Model.scala, model/temporal/standaloneactivity/System.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/standaloneactivity/StandaloneActivityPins.test.scala, model/gate/Roots.scala, model/ir/**, model/cases/**]

## Description
Apply the new declaration, derivation, step, evidence, input and counter surfaces to the product/protocol/admission subjects, including fn107 additions, and decide the protocol machine's default-disabled pairs explicitly.

**Size:** M
**Files:** standaloneactivity Model/System/Claims roots before layout; feature pin tests and generated IR comparisons.

### Approach
- Start only after fn-112.11's total schema and fn-120.1's later choice schema are done. The same-spec task dependency records fn-112.11; the conductor checks fn-120.1's completion as a cross-spec entry gate. Replace copied admission machines and private step wrappers with derivation and canonical helpers. Use the settled named-choice syntax for existing branching; keep result count/order, tables and Case bytes exact. Do not wait for fn-120's lint/explorer work or strict unnamed-branch refusal.
- Capture declaration names/defaults, reduce evidence blocks to the two exceptions and migrate ProtocolState.attempts to UpTo[2] while preserving Active.
- When a `a == b || a == c …` chain becomes `phase.in(…)`, keep each status set as one named def (`terminal`, `paused`, `running` for the product; `held` for the protocol; the admission record's own), as `terminalPhase` and `running` already are in `Model.scala`, rather than repeating the `in(…)` list at use sites; task 8 moves those defs onto the vocabulary objects (R10). The R4 shared-claim defs of task 7 take these named sets as arguments.
- Write the product's two transition claims with the claim patterns of task 4: `terminalIsFinal` as `once(terminal).keeps(_.phase)` and `pausedIsNotDispatched` as `never(s => running(s.state)).from(paused)` over the named status sets, keeping their frozen names and the same Property rows.
- **Model gaps (spec Decision Context "Model gaps found by the modality study"; `.plans/MODALITIES.md` section 6).** `protocolControlStep`'s two wildcard arms (`case _ => Nil`, `Model.scala:~386, ~391`) disable 168 pairs by default, among them seven caller RPCs the server answers with `FailedPrecondition` (`control-pause` in `paused`, `pauseRequested`, `cancelRequested`; `control-unpause` in `scheduled`, `backingOff`, `started`, `cancelRequested`; `chasm/lib/activity/handler.go` pause/unpause). The freeze (R1) forbids a rejecting row, since it is a new table row and new fingerprints, so write each disabled phase as an explicit arm returning `disabled` with the reason in a comment beside it (no wildcard arm remains in the feature Model; the function body changes under the R1 function projection, the table does not), and list the seven in the done summary as the follow-up "rejecting rows with `because`" that a later spec takes once the freeze lifts. `terminated` and `cancelRequestedWhileStarted` are false on the 120 `notFound` stutter rows of the terminal phases and only pinned `find` Queries ask them; rephrasing them (`when c holds (s.outcome == accepted implies …)`, or a free `verify`) changes Property verdicts over the table and possibly Contract clauses, which R1 forbids, so they stay as they are and are listed in the done summary as the same follow-up for the owner.
- Preserve `.results("Delivery")`, all finite catalogs and function-table rows; remove the dead Delivery enum only if unused by the frozen outputs.
- State retry saturation at attemptBound on the Property instead of changing retry behavior.
## Acceptance
- [ ] Product/protocol/admission declarations contain no duplicate steps list or private single-step helper and use the settled name/evidence/input/default forms.
- [ ] fn-112.11 and fn-120.1 are complete, in that order, before branching rewrites start; every rewritten branch uses final named-choice syntax.
- [ ] The fn107 held-delivery and response-loss machines are included.
- [ ] Each phase-set membership is one named def (`terminal`, `paused`, `running`, `held`, and the admission record's) used at every site; `terminalIsFinal` and `pausedIsNotDispatched` are written with `once/keeps` and `never/from` over those defs with their original rows.
- [ ] No wildcard arm remains in a feature step function: each default-disabled phase of `protocolControlStep` is an explicit `disabled` arm with its reason, the table is unchanged, and the done summary lists the seven server-rejected pause/unpause pairs and the two witness-only Properties (`terminated`, `cancelRequestedWhileStarted`) as follow-ups the freeze defers, with the reason.
- [ ] Result metadata, branch count/order, finite catalogs, IDs, tables, Query answers and Case bytes equal task 1. IR differences are limited to R1's exact source-position/moved-function/function-projection allowances and authored Query.total, inert choice-name and queue-entity metadata deltas; entity-sensitive fingerprints match the approved delta and all other fingerprints remain exact.
- [ ] Focused feature roots, model gate and relevant Go golden tests pass.
## Done summary
Migrated the standalone activity's product, protocol and admission models (and the fn-107 held-delivery and response-loss machines) to the fn-112 DSL. Tables, Definition IDs, fingerprints, Query answers/totals and Case bytes are unchanged; model/cases is byte-identical. Commits f9b2cdf4ea, 5698740e4c, c64ef46e48, 7f3c0c2856.

**What changed**
- **Model.scala / Claims.scala (product, protocol):**
  - Captured names for actions, timers, machines, the restriction, the composition (now with typed selectors), Properties, Scenarios, Queries and Limits.
  - Input tokens live in `object Inputs`, because the timers and the `control` action already own those names; `start(Inputs.scheduleToStart := expires)`.
  - `attempts: UpTo[2]`; the hand-written `given Finite[ProtocolState]` and the dead `enum Delivery` are gone; `results("Delivery")` stays.
  - `accept`/`disabled`/`stay`/`.because`/`in`/`records` everywhere; `productStep` and `moves` are gone.
  - Status sets as named defs: product `terminal`, `paused`, `running`; protocol `terminalPhase`, `live` (was `running`), `held` (was `attemptHeld`).
  - `terminalIsFinal = once(terminal).keeps(_.phase)`; `pausedIsNotDispatched = never(s => running(s.state)).from(paused)`.
  - Protocol evidence lists only `statusTimedOut(_)` and `attemptCount`; product has none.
  - `retryCompletes` states that the count saturates at `attemptBound`.
  - `protocolControlStep` has no wildcard arm: each phase is an explicit `disabled` arm with its reason.
  - `stoppedBeforeRetry`/`startedByPollingWorker` use `own`/`synced` instead of string keys.
- **System.scala (an opus subagent):**
  - `staleAdmission = currentAdmission.rebind(attemptStart ~> admitStale)`, `currentRecord = currentAdmission.unmonitored`, `staleRecord = staleAdmission.unmonitored`.
  - `pauses`, `timesOut`, `views` and `behind` are gone; `pauseStep` has explicit arms.
  - Named choices for `admitted`, `loseAdmissionAnswer`, `enqueueView` and `enqueueDetail`.
  - Evidence defaults: the three evidence functions are deleted, and admission keeps only `statusTimedOut(_)`.
  - Captured names, `records`, starts omitted. The queue providers and compositions keep their steps/sync lists and string keys for task 7.
- **Go:**
  - fn-115's migration golden reads Function references as tokens (`functions_by_reference`, sharing the baseline's `functionless`), with mutation controls (an opus subagent).
  - The P export writes `OP_CONTAINS` over a written-out list as equalities.
  - Test expectations now carry choice names and captured declaration names, and derive a position line from the IR.
- **README:** how two families share one package.

**Decisions (autonomous)**
- **Families are object givens imported per file** (`ActivityFamily`, `SystemFamily`). Two package-level givens clash, and a package-level given wins over an imported one. Members.scala imports `SystemFamily.given`.
- **All-default `start(unset, unset, unset)` stays positional.** `start()` needs a change to core `Action.apply()` (task 5's surface). Deferred to task 10.
- **R7 exception:** `stoppedBeforeRetry` keeps its explicit start. A defaulted composition start takes the worker's position in Worker.scala, and the migration golden compares each position's file (`positions_by_file`). Task 8, whose file moves meet the same rule, can drop it.
- **Queue outcome:** `given Accepted[QueueOutcome] = Accepted(internal)` lives in the `QueueOutcome` companion, since a second package-level `Accepted` breaks `choose` inference.
- **Outside Touches:** `tools/umpire/{internal/golden,model,lower,export}`, `model/lifter/testdata/lifts/{Members.scala,expected}`, `model/README.md`.

**Follow-ups the behavior freeze defers (owner decides their home)**
- Seven server-rejected caller pairs (168 rows) are explicit `disabled` arms. The server answers each with FailedPrecondition (`operator_commands.go` "non-pausable"/"non-unpausable state"):
  - `control-pause` in `paused`, `pauseRequested`, `cancelRequested`;
  - `control-unpause` in `scheduled`, `backingOff`, `started`, `cancelRequested`.
  - The follow-up is rejecting rows with `because`.
- `terminated` and `cancelRequestedWhileStarted` are false on the 120 `notFound` stutter rows of the terminal phases and only pinned `find` Queries ask them. Rephrasing them changes Property verdicts, which R1 forbids.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the same family (Opus).
- Round 1: SHIP with 3 P3s. The pause-arm comment wording was fixed in 7f3c0c2856; P3 2 is recorded above as the R7 exception.

**Deferred P3/FYI**
- `attemptBound` and `UpTo[2]` both state the bound. A shared alias needs lifter support for type aliases.
- `accept` answering `internal` for queue steps; tasks 7/12 may prefer a documented queue-local helper.
- `admitHeld` duplicates `admitCurrent` minus the commit failure (pre-existing).
- With `functions_by_reference`, `alpha_normalized_parameters` and `function_name_substitutions` are inert (task 10).
- `TestQuintKeepsEveryNamedAlternative` hard-codes the enqueue names, so task 7 must update it.
- `idleOverQueue`/`idleOverMatching` are now used only by Members.scala.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f9b2cdf4ea, 5698740e4c, c64ef46e48, 7f3c0c2856
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0; model/cases byte-identical), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), scala-cli test model/lifter (exit 0), make lint-model (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: