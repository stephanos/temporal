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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
