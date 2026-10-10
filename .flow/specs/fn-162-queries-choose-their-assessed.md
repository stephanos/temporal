# Queries choose their assessed Properties; each Property declared once

## Conversation Evidence

> user (turn 31): "temporal/model/temporal/features/activity/standalone/system/Heartbeat.scala has 3 separate machines; will/should we merge them?"
> user (turn 32): "why would we have both HeartbeatCompletion and HeartbeatThenCompletion spread across system.scala and its own file?"
> user (turn 34): "remind me, why not just 1 file?"
> user (turn 35): "yes, go for the real fix"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 40% [user], 60% [paraphrase] -->

The standalone activity's system level declares many *subjects*: copies of `ActivitySystem` (`Derived(ActivitySystem.unmonitored)` or `.rebind()`, with nothing changed) that exist only to hold one or a few Properties and Queries. `Heartbeat.scala` has three of them. The copies exist because a realized Case is assessed against its subject machine's Properties, so cloning the machine is the only way to give a Case a focused set of claims. As a result, the same Property is declared again and again (an identical "completes" predicate sits in `HeartbeatCompletion`, `HeartbeatRetry`, `Completion`, `Pausing`, `DispatchEligibility` and others), and the one-machine-per-file rule turns every clone into a file.

The fix: each Property is declared once, in the facet whose promise it states, and each Query (or the realization that runs it) names the Properties its Case is assessed against. A facet's claims then live with the facet, with no machine copies. [paraphrase]

## Architecture & Data Models
<!-- scope: technical -->

- **Today:** a Query runs over a scenario of a machine, and a realization `Realizes(<subject machine>)`. The model assessment of a Run appears to check the subject machine's Properties, which is what the regression test "focused subjects independently own the exact complete property truth sets" pins. How exactly the IR, the lowering and the Go assessment select a Case's Properties is still to be confirmed. [inferred]
- **Target:** a Property is declared once, on the machine or in its facet (fn-161), and identified by that declaration. A Query states its **assessment set**: the Properties a Case lowered from it is assessed against, by default the Property the Query finds or verifies. A realization runs Queries of the machine it realizes, so it no longer needs a subject clone. Machine-wide claims that every Run is assessed against, if any, are declared as such explicitly. [paraphrase]
- Derived machines that change behaviour (rebind, restrict, extend, assuming, refining) stay. Only derivations that change nothing and exist as claim containers go away. [paraphrase]

## Edge Cases & Constraints
<!-- scope: technical -->

- A Query's verdict and its Case's expected outcome must not change. A Case assessed against a narrower or equal set of Properties than today must reach the same verdict. A Case whose set grows needs explaining. [inferred]
- Property and Query ids in the IR change when they move from subject objects to the facet or machine. Cases, manifests, pinned fixture trees and Go tests that name them change too. [paraphrase]
- Capability-generated Properties (`Closable`, `Pausable`, …) keep their own identities and their per-machine scope. [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Before changing anything, the way a Case's assessed Properties are selected today is traced end to end: model, IR, lowering, `umpire-run --model` and `umpire-assess --model`. It is recorded in the spec. [inferred] Errors: if selection depends on something other than the subject machine, the target design is revisited with the user before implementation.
- **R2:** A Query declares the Properties its Case is assessed against, by default the Property it finds or verifies, and the IR carries that set. The lowering and the Go assessment assess a Case against exactly that set. [paraphrase] Errors: a Query naming a Property of another machine fails the lift, naming both; a Case whose IR has no assessment set is refused by the assessment, not silently assessed against nothing.
- **R3:** Each Property is declared once. A predicate identical to another declaration of the same machine is refused or reported by the lifter. [paraphrase] Errors: the duplicate is named with both locations.
- **R4:** The standalone activity's claim-container derivations (derivations that change nothing and only hold Properties and Queries) are removed. Their Properties go to the facet or machine that states the promise, and their Queries keep their scenarios, limits, totals and expectations. [user] ("yes, go for the real fix") Errors: no error surface beyond R5.
- **R5:** Every Query keeps its answer and total, and every Case keeps its expected outcome. IR and Case differences are limited to moved Property and Query ids and the assessment sets. The subject regression test is replaced by one pinning that each Case's assessment set is exactly its Query's. [inferred] Errors: any other difference is explained before it is accepted.

## Boundaries
<!-- scope: business -->

- Changing what any Property or Query claims is out of scope. [inferred]
- Derivations that change behaviour stay. [paraphrase]
- Nexus and the task queue are migrated only if they have claim-container derivations; their behaviour-changing derivations stay. [inferred]

## Decision Context
<!-- scope: both — conditionally substructured -->

The user chose the real fix over relaxing the one-machine-per-file rule so a facet file could hold its claim subjects. [user] Relaxing would keep the clones and the duplicated Properties, and only hide them in fewer files. Scoping claims per Query removes the reason the clones exist. [paraphrase] Merging the three heartbeat subjects into one subject was rejected: every heartbeat Case would then be assessed against all three Properties, losing focus. [paraphrase]

## Parked unknowns

- Whether anything other than the subject machine decides a Case's assessed Properties today (e.g. realization-level evidence lists or capability Properties). R1's trace resolves this. [inferred]
