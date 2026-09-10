# Umpire 4 delivery order

Build on the completed Testpilot and model authoring cutovers. Flow owns task status;
this document records delivery order. Architecture and terminology live in the
[Umpire 4 specification](UMPIRE4_SPEC.md).

## Current work

### 1. Author a live Case from a Model file — fn-83

[fn-83 — Author a live Case from a Model file](../.flow/specs/fn-83-author-a-live-case-from-a-model-file.md),
from the 2026-09-09 gap analysis of [UMPIRE4_VISION](UMPIRE4_VISION.md) against the model, the
Testpilot runtime, and the live tests. Of the vision's seven acceptance criteria, three are met
(Go SDK driven, one fault injected, deterministic file artifacts), one is half met (two
consumers), and three are unmet (explorative test, translated functional test, syntax friendly
enough for Lean newbies past the Model block). The largest distance is that **a developer who can
write the Model cannot get a running test without also writing a Producer**: the Nexus success
Model is 43 lines of command syntax, its Producer is 380 lines of raw Lean, the worker-outage
Case is 227 lines with no Model at all, adding it touched 14 files, two hand-maintained tables
name every Case, and the only tutorial documents keywords that do not exist.

The spec makes one thing true: a Model file plus one `case` block is a running, checked-in,
deterministic live test, and a fault is one line in a Scenario. Eight tasks: a generic
`Umpire.Case.Producer` extracted from the success Producer (the early proof point is that the
async-Nexus fixture regenerates byte-identical through it), two Temporal realization templates
(`nexusOperation` with sync and async responses, `workflow`), the `case` command with a Lean Case
registry that replaces both tables, `fault ... before|after <hook>` Scenario lines with the
worker-outage Case re-authored as a Model, a translation of the upstream
`TestNexusOperationSyncCompletion` using only the command syntax (its admin-service mutable-state
assertion becomes the vision's white-box Known Gap), a provisioning package plus an `umpire-run`
CLI that runs a fixture against any gRPC endpoint with exit codes 0/1/2/3, and one tutorial
`model/AUTHORING.md` whose code blocks a Go test holds equal to the translated Model file.

**Unblocked: fn-82 closed on 2026-09-10** (ten of ten tasks, completion review SHIP), so the
vocabulary and module layout this spec is written against are now in the tree. Reverse
dependencies recorded in
Flow: fn-33 (the generic Producer is the only path from a Search result to a Case), fn-70 and
fn-79 (both re-anchor on the `case` block), fn-46 (new module rows), and fn-29 (consumes the
provisioning package, not the CLI).

EVD-20, EVD-21, and AUT-09 were approved on 2026-09-10 under GOV-02. One item still needs a
human: fn-83's evidence block reads generated history names AUT-09 does not cover, so task .8
drafts one amendment for approval. Boundaries: no
exploration or fuzzing (fn-33), no canary (fn-70, fn-29), no Program surface syntax, no new fault
or instruction kinds, no clock model, no change to the typed examples, no new CI workflow.

### 2. Deepen five shallow module clusters in Umpire and Testpilot — fn-84

[fn-84 — Deepen five shallow module clusters in Umpire and Testpilot](../.flow/specs/fn-84-deepen-five-shallow-module-clusters-in.md),
from the 2026-09-09 architecture review of `model/Umpire`, `model/Testpilot`, `common/testing/testpilot`,
`tests/testcore/testpilot` and `tools/umpire`. Five independent scans, one per area, each nominated
one deepening: a shallow module cluster where one decision is spelled out in two to six places and
kept in agreement by hand becomes one deep module with a small interface that callers and tests both
cross. The spec carries exactly those five, one task each, in the order the scans recommend:

| Task | Deepening | Pin |
| ---- | --------- | --- |
| .1 | one `outage` module owned by the worker registry; `Validate` and `Open` share one `OutagePlan` | fault suite and live worker-outage tests, unchanged Verdicts |
| .2 | one Driver-contract leaf package replacing the 14-declaration facade mirror and its four adapters | conformance and facade tests compile unedited |
| .3 | `Umpire.Search.admit` owning the Property, Scenario, Query, view and search chain | `PlanResult` bytes, goldens and fingerprints unchanged |
| .4 | the offline Evidence structure module returns a verdict per audience instead of findings two callers re-judge | mutation suite diagnostics byte-identical |
| .5 | `Umpire.Case.Projection.lower` derives the Contract rule from the checked field Property | Case fixtures byte-identical under ART-11 |

No behavior changes; every task records an equivalence pin before it moves code and closes with
`make umpire-check-regression`. The plan review is SHIP after one fix round.

**Blocked on fn-83 only; fn-82 closed on 2026-09-10.** fn-82 renamed or moved files in all five
areas, including the Go `Opcode` rename in the facade and the worker Driver's admission half, and
that work has landed. Task .5 shares its seam with fn-83's generic
`Umpire.Case.Producer`, so the spec also waits for fn-83; that holds .1 and .2 as well, which is
accepted because fn-83 has no tasks yet to anchor a task-level edge. fn-22 and fn-33 depend on this
spec in turn: fn-33's exploration bridge sits on the search-view transport sites .3 replaces, and
fn-22's promotion path consumes `search`, which .3 keeps public. Dispatch is serial in task order
because every pair of tasks shares a documentation or test-root file.

Boundaries: no renames beyond what the new modules need, no change to the evaluation budget
(`CONSIDER(umpire)` on the cubic reservation stays separate), to delivery routing, to the
generators' publication tooling, to Property evaluation combinators, or to `Umpire.Json` sealing.
EVD-20 is approved (2026-09-10, GOV-02); the MOD-14 restatement .2 drafts stays pending.
Candidates the scans surfaced and the spec declined are listed in its Decision Context.

### Additional open specs

These remain open in Flow and are outside the first-canary critical path.

| Spec                                                                                              | Dependencies | Next action                                                                                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------- | ------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-46 — Lean module impact index](../.flow/specs/fn-46-export-lean-model-module-impact-index.md) | fn-45, fn-83 | Refreshed three-task plan is SHIP against current model owners. Deliver the shared loader, pure dependency/facade/test index, and opt-in export/check commands. |

## Downstream delivery

Each spec needs a fresh Testpilot plan review before implementation. Prior reviews of the
retired execution architecture do not approve the rewritten plans. Completed dependencies do not
block replanning or execution.

| Spec                                                                                              | Dependencies                       | Next action                                                                                                                                                                                                                                   |
| ------------------------------------------------------------------------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [fn-33 — Bounded exploration](../.flow/specs/fn-33-run-serial-bounded-semantic-exploration.md)    | fn-40, fn-64, fn-69, and **fn-83** | Review whole-Case candidates, serial coordination through Testpilot, lost iterations, semantic coverage, and bounded 10x behavior.                                                                                                            |
| [fn-22 — Replay and reduction](../.flow/specs/fn-22-deterministic-replay-semantic.md)             | fn-5, fn-64, and fn-69             | Resolve **MAJOR_RETHINK** before implementation: separate exact candidate identity from Contract-relative violation equivalence, prove the negative Case before reduction, and retain explicit offline semantic replay and checked promotion. |
| [fn-26 — Qualification receipts](../.flow/specs/fn-26-local-qualification-receipts-and-staged.md) | fn-48, fn-64, and fn-69            | Review offline Testpilot Case/Profile/Run/Verdict admission, receipt multiplicity, and idempotent publication. Assessment must never create or replay a Run.                                                                                  |
| [fn-29 — Production canary](../.flow/specs/fn-29-bounded-production-canary-execution-and.md)      | **fn-26**; fn-48, fn-64, fn-69, and fn-83 | After fn-26 ships, review external policy and credentials, serial Testpilot Runs, leases, lost Runs, reconciliation without redispatch, and publication.                                                                                      |

All runtime work uses `testpilot.Prepare(case, profile)` → `PreparedCase.Run(ctx, driver)` and the
server/worker authority split. New scenarios remain Case data; canary policy, credentials,
leases, recovery, and publication stay outside Testpilot and Umpire.

## Deferred and superseded

**fn-79 — Nexus operation cancellation:** [spec](../.flow/specs/fn-79-deferred-nexus-operation-cancellation.md).
Includes former fn-78.5/.8/.9 cancellation scope and fn-77’s cancellation qualification. Resume only
on an explicit user request; autonomous delivery approval does not override this deferral. Generic
fn-78 syntax/monitoring/qualification and fn-70 remain deliverable without it. Existing shutdown
and bounded cleanup cancellation behavior stays in scope.


**fn-70 — Scheduled canary proof of concept:** deferred by user decision; it was previously
queued after fn-78 as the second model consumer. Resume on an explicit user request. Nothing in
the delivery queue depends on it. On resume it inherits fn-80's `temporal.DeriveProfile`, so a
second model consumer no longer hand-writes a `ProfileSpec`. The bind-and-run helpers landed as
test-local `bindCase`/`runCase` in `tests/testpilot_run_case_test.go` rather than exported from the
fixture package, because exporting them would compile the whole server into a Quick command; a
canary test under `tests/` reuses them where they are. fn-83 extracts the provisioning that forced that
placement into `common/testing/testpilot/temporal/provision` and adds `umpire-run`; on resume fn-70
consumes both and re-anchors its catalog entry on the `case` block that replaces the fn-68 Producer.

[Spec](../.flow/specs/fn-70-scheduled-canary-proof-of-concept-as-a.md).
Retained scope on resume: fn-78 first, with fn-68, fn-71, fn-72, and fn-73 as transitive
prerequisites.
Nine tasks cover all ten requirements, with a SHIP plan review. Implementation follows fn-77
to serialize shared Producer edits; fn-77 is not a semantic prerequisite.

Implement the second consumer under `tools/canary`: manual check selection, a fresh scheduled
Workflow each minute, Activity-owned Testpilot execution, bounded results, and isolated repeated
measurements. Consume the Driver and binding interfaces delivered above; do not repeat their
implementation work. Retain the cross-consumer proof; fn-73 already owns the live proof that one
Case byte sequence runs against two environment bindings.

This is a local/development prototype. It does not depend on fn-26 or fn-29 and does not authorize
production deployment or replace fn-29's separately scoped production-canary design.


These entries are outside the delivery queue and are not prerequisites for it.

| Deferred spec                                                             | Revisit when                                                                                                   |
| ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| [fn-60](../.flow/specs/fn-60-deepen-authored-lean-canonical-json.md)      | Optional handwritten canonical JSON consolidation becomes worth prioritizing; it has no downstream dependency. |
| [fn-15](../.flow/specs/fn-15-standalone-api-and-config-input-catalogs.md) | Platform completeness is needed beyond the proven model family.                                                |
| [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md)         | Optional checker adoption becomes valuable.                                                                    |
| [fn-24](../.flow/specs/fn-24-lean-native-verification-receipts-and.md)    | A verification receipt/profile platform is justified.                                                          |
| [fn-25](../.flow/specs/fn-25-optional-callerclosure-veil-binding-and.md)  | A second verification backend is justified; caller closure remains historical.                                 |
| [fn-30](../.flow/specs/fn-30-release-evidence-graph-and-manual.md)        | Real Claim Assessment evidence supports release governance.                                                    |

[fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) is historical;
[fn-61](../.flow/specs/fn-61-simplify-the-umpire-go-execution-surface.md) and
[fn-63](../.flow/specs/fn-63-consolidate-umpire-go-tests-into-golden.md) are superseded by fn-64.
Any remaining `todo` children do not reactivate them. Broader test consolidation needs a new
Testpilot proposal with an independent oracle.
