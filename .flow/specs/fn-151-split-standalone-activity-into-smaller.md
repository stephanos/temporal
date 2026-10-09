# Split standalone activity into smaller subject models

## Goal

Make the standalone activity System easier to understand through smaller, cohesive subject files and focused derived models. This captures the owner's 2026-10-08 request to add Flow todos after discussing the layout; the three existing tasks now carry the executable move, ownership and verification handoffs.

## Owner direction

- Prefer smaller files and models.
- Ignore hard-coded line limits; choose boundaries by subject, cohesion and readability.
- Filenames must tell readers what the file contains. SettlementByID and Admission were challenged as unclear names.
- Use Dispatch* filenames for the dispatch-to-start subject, including its deeper protocol models and task-queue compositions. ActivityRecord/Admission/AttemptStarting do not communicate that vertical slice adequately.
- Use a shared subject prefix where several files belong together; do not require multiple files for every subject.
- The scope is the standalone activity system under model/temporal/features/activity/standalone/system.

## Acceptance Criteria

- **R1:** Existing derived timeout-retry, heartbeat, by-ID and reset models and the worker composition live in files named for their subjects. Each model retains its own properties, scenarios and queries. Pure moves preserve top-level package names and declaration identities.
- **R2:** Focused verification currently embedded in ActivitySystem is extracted into smaller derived subject models where the claims and scenarios form a coherent unit. The lifecycle machine retains its state, transitions, refinement, shared capabilities and lifecycle-wide invariants. No fixed file-size target, generic Properties/Queries/Rules split, or duplicated transition definitions is introduced.
- **R3:** Names and documentation distinguish focused verification models from the core transition definitions. By-ID and task-token response transitions remain accounted for. Group the dispatch-to-start models currently in Record.scala and WithTaskQueue.scala under Dispatch* filenames, including eligibility, queued delivery, pause-before-start races, duplicate delivery and lost start responses. Do not merely rename Record.scala to ActivityRecord.scala or Admission.scala. Other multi-file subjects use consistent subject prefixes.
- **R4:** The refactor preserves modeled behavior, query coverage, bounds, expectations and realization semantics. Any unavoidable identity change from moving a claim or query to a new derived owner is explicitly mapped and checked through exports, realizations, regressions and generated artifacts. IR and Cases are regenerated and reviewed. The model gate must pass except for the named resource interruptions explicitly deferred on 2026-10-09 to fn-157 under the owner's roughly-one-hour gate policy; retain those RED results and require complete independent equivalence, lint, corrected-consumer checks and review before structural closure.

## Early proof point

Task fn-151-split-standalone-activity-into-smaller.1 seals exact declaration identity/body equality and mapped source coordinates before ownership changes. Before claim extraction in task .2, an ordinary lift must prove the existing zero-change rebind retains the complete lifecycle tables, refinement, visibility and monitor semantics; if it fails, stop dependent extraction and reconsider the derivation without changing the DSL or removing those semantics.

## Subject grouping and filename prefixes

Dispatch is the agreed vertical slice for getting an activity to a worker. Group its deeper protocol models with its lifecycle verification rather than treating Record or Admission as a sibling lifecycle subject. Retain separate machines and their semantics; grouping them does not merge their states or transitions.

Use subject-first filenames so related files sort together. The Dispatch* family is owner-approved; the other concrete suffixes below are implementation guidance, to refine against cohesive model boundaries:

| Subject prefix | Example files and contents |
| --- | --- |
| Dispatch* | Dispatch.scala for start eligibility/delay/backoff models; DispatchRaces.scala for held delivery and lost start responses; DispatchWithTaskQueue.scala for queue compositions. Place the existing ActivityRecord and faulty-control models within this family. DispatchWithWorker.scala is appropriate for the worker composition if its claims remain about starting attempts. |
| Retry* | RetryFailures.scala and RetryTimeouts.scala for failure and attempt-timeout retry/exhaustion models. |
| Heartbeat* | Heartbeat.scala, or HeartbeatCompletion.scala and HeartbeatRetries.scala when separate complete models improve navigation. |
| Reset* | Reset.scala, or ResetSettlement.scala and ResetKeepingPause.scala for distinct reset models. |
| Response* | ResponseByID.scala for completion, failure and cancellation by activity ID. Use the same prefix for any later focused worker-response models; do not invent an empty counterpart. |

Cancellation.scala, Pausing.scala, Completion.scala and Timeouts.scala can remain single subject files. System.scala retains the lifecycle machine. Realization.scala retains the execution bindings unless a separate need is demonstrated.

Prefixes group subjects, not member kinds: each file owns complete models with their properties, scenarios and queries. Pick one primary subject for a model that crosses subjects (for example heartbeat-triggered retry); do not duplicate it under both prefixes. No fixed file count or line limit applies.

## Boundaries and dependencies

The owner prioritized this spec on 2026-10-09 as the next delivery after the structural batch fn-142/fn-143/fn-145 closes, before fn-140/fn-123 authoring. This supersedes the previous wait for fn-128.6 and fn-129.5. Re-anchor to the closed structural baseline, including its joined Activity changes and regenerated artifacts. Preserve the inherited Activity completion, fatal-failure and pause/resume obligations for Batch 5 with strict assertions unchanged; the split does not fix them. Heavyweight Quint JSON memory work remains deferred to fn-154. Prove the split's behavior and identity mapping against that baseline, distinguishing inherited failures from regressions. The subsequent fn-155 abstraction spec consumes this split's closed baseline, followed by fn-140's query-authoring migration against the actual split owners; neither later refactor belongs to this spec, and this refactor does not change the DSL.

This follows the completed layout work in fn-126 and kind/form organization in fn-132. The three tasks remain serial (.1 then .2 then .3), with one regeneration and integrated verification/review at .3. Parallel preparation may inspect independent file-layout and claim/consumer concerns; code joins and shared heavy gates remain serialized.

No server behavior changes, new activity coverage requirements, task-token value model, or new composed attempt/dispatch lifecycle is implied. The current realization bindings remain in scope for reference/identity updates, not an unrelated redesign.

## Decision Context

Maintainability (plan review): duplication - identical completes predicates on three replacement owners and scheduleToStartFires on two are deliberate independently owned claim instances, checked by exact predicate equivalence; structure - none identified.

Use the existing refinement-preserving derivation, subject to the early ordinary-lift proof; unmonitored is not an equivalent substitute because it also removes refinement. Shared predicates are instantiated independently on their consuming owners, not aliased across owners. The mapping records deliberate one-to-many claim ownership while preserving every original Query once in its original artifact context. Model-only competing deadline Queries retain their no-realization standing and absent live expectation; grouping source files does not grant execution coverage.

The equivalence proof preserves complete tables, Query receipts, bounded witnesses, expectations and controller/evidence declarations separately from intentionally mapped identities. Negative comparison specimens detect omitted roots, weakened predicates, lost refinement/monitors and altered bounds or execution instructions. Reuse established comparisons; no new general verification framework, domain reduction or memory-fit claim is introduced. All strict inherited Activity failures remain recorded for Batch 5, and fn-154 retains heavyweight Quint memory work.

## Resource-gate disposition, 2026-10-09

The conductor applies the owner's MILESTONES.md rule for gates stuck for roughly an hour. `fn-157-bound-native-verification-memory-and` records the five kernel-confirmed canonical Go victims, the no-update Model gate's generator OOM and scratch-storage exhaustion, and the ordinary Case/fixture generator OOMs. This changes R4 and task .3's gate acceptance; it defers their resource-fit restoration and preserves their actual RED command results. It grants no memory-fit, missing-test or passing-gate credit. Fn-154's Quint JSON work and Batch 5's strict completion/fatal/pause assertions retain separate ownership.

Structural closure still requires complete independent native and seven-Model Case equivalence against the sealed baseline, exact owner/identity mappings, exercised negative controls, passing production regeneration and both lints, passing exact corrected consumers, and independent implementation and spec-completion review. The corrected completion guard mutation needs an actual focused result. Canary controller/publication failures remain outside this resource deferral and require evidence-backed affected-package classification before review. Unexplained new semantic, identity or assertion failures block closure. No native optimization, reduced domains or weakened assertions enters this split. Task .3 records missing selections and links every deferred restoration obligation.

Use the existing cumulative split lint target with `GOLANGCI_LINT_BASE_REV=4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd` and `GOLANGCI_LINT_FIX=false`. It covers all three tasks' Go changes. Retain the default lint invocation's 743-finding RED separately; its February local-main comparison selects existing branch debt, and source/log classification identifies no fn151-introduced finding. A passing cumulative check grants no default full-branch lint pass. Restore the nine accidental default-autofix edits to proven pre-lint bytes before any accepted verification.

The focused corrected completion guard negative started its exact parent and child, then exited 1 after 39.446 seconds when the kernel OOM-killed conformance at 23:02:45 UTC. Its expected GuardError and assertion remain UNOBSERVED. Fn-157 explicitly retains this required negative's restoration with unchanged mutation and assertion. This recurring native-resource interruption is deferred under the same owner policy and grants no negative-test pass credit. Both complete Canary packages passed on restored source through the corrected runtime wrapper; classify their canonical failures as environment-sensitive with the low-level ENOTEMPTY cause still unknown, preserving the canonical RED and the documented inference about temporary-directory selection.

## Quick commands

```bash
mise exec -- scala-cli compile --server=false model/project.scala model/framework model/temporal
```

Tasks .1 and .2 use focused source checks and scratch seals. Task .3 owns the single production regeneration, Model/artifact checks and canonical Go coverage with recorded actual results; a focused pass does not replace a RED full-suite result.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Existing derived timeout-retry, heartbeat, by-ID and reset models and the worker composition live in files named for their subjects. Each model retains its own properties, scenarios and queries. Pure moves preserve top-level package names and declaration identities. | fn-151-split-standalone-activity-into-smaller.1 | — |
| R2 | Focused verification currently embedded in ActivitySystem is extracted into smaller derived subject models where the claims and scenarios form a coherent unit. The lifecycle machine retains its state, transitions, refinement, shared capabilities and lifecycle-wide invariants. No fixed file-size target, generic Properties/Queries/Rules split, or duplicated transition definitions is introduced. | fn-151-split-standalone-activity-into-smaller.2 | — |
| R3 | Names and documentation distinguish focused verification models from the core transition definitions. By-ID and task-token response transitions remain accounted for. Group the dispatch-to-start models currently in Record.scala and WithTaskQueue.scala under Dispatch* filenames, including eligibility, queued delivery, pause-before-start races, duplicate delivery and lost start responses. Do not merely rename Record.scala to ActivityRecord.scala or Admission.scala. Other multi-file subjects use consistent subject prefixes. | fn-151-split-standalone-activity-into-smaller.1, fn-151-split-standalone-activity-into-smaller.2 | — |
| R4 | The refactor preserves modeled behavior, query coverage, bounds, expectations and realization semantics. Any unavoidable identity change from moving a claim or query to a new derived owner is explicitly mapped and checked through exports, realizations, regressions and generated artifacts. IR and Cases are regenerated and reviewed. The model gate must pass except for the named resource interruptions explicitly deferred on 2026-10-09 to fn-157 under the owner's roughly-one-hour gate policy; retain those RED results and require complete independent equivalence, lint, corrected-consumer checks and review before structural closure. | fn-151-split-standalone-activity-into-smaller.2, fn-151-split-standalone-activity-into-smaller.3 | — |
