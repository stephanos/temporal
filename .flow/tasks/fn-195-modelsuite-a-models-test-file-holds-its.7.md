---
satisfies: [R10]
---
# fn-195-modelsuite-a-models-test-file-holds-its.7 Polish every framework sample against the settled DSL without changing behavior

## Description
**Owner request (2026-10-18):** "once the DSL has settled after the last task, add antoher task after to go over each umpire frameowkr sample and simpify it (wihoutht losing behavior); each one must be the best version of itself".

After fn-195.5, the preceding DSL/class-only migrations, and the committed framework-only fn-197 native checkpoints through .3, review every native framework sample against the settled authoring surface. Make Bank, Voting, TwoPhaseCommit, Kettle, ReliableBroadcast and TwoLayeredCache the clearest idiomatic versions of their respective models. Review supporting author-facing Scala fragments too. Remove migration scaffolding, redundant state views, trivial indirection, duplication, obsolete comments and unnecessary DSL ceremony where doing so improves the model. Keep meaningful domain names and useful abstractions; this is not line-count golf or a requirement to change an already good file.

**Size:** L

**Touches:** `model/framework/samples/**`, `model/framework/README.md`, `model/framework/SEMANTICS.md`, `model/framework/REFERENCE.md`

Only native Scala samples/tests and live realization fixtures may change under the samples glob. Related documentation changes are limited to sample links/examples that this task changes. Exact upstream comparison files and historical uncompiled snapshots remain untouched.

For each sample, record a brief before/after rationale and preserve its actual mathematical behavior. Capture a fresh baseline of starts, input domains, transitions/outcomes/facts, claim verdicts, bounds, negative-control paths and realization bindings before editing. Compare reachable graphs/steps where complete and retain honest cuts/unknowns where histories or budgets prevent completeness. Preserve the signed Bank behavior and its negative amount control, Voting's majority/alternate quorum behavior, TwoPC's fairness/deadline/Eager control, cache history monitors and Kettle/broadcast edge cases as they stand at this task's admission. Do not repair a perceived domain bug or change a stutter/fairness rule under the label simplification.

Use the settled ModelSuite forms for tests and class-only primary declarations. A sample should teach the authoring style we want users to copy; avoid reintroducing old Query/Scenario/state-record adapters. Simplify tests too without deleting their purposes or weakening evidence. Any needed DSL feature change is a separately scoped finding, not permission to redesign the DSL here. Update user-facing reference examples without Scala-file references.

Use disjoint per-sample subagents where useful, with one integration/commit owner and serialized compiler gates. Verify current native/lint gates and fresh source/separate-library realization clients under unchanged strict flags. Held Temporal, IR/export/lifter, Go, generated outputs, upstream exact comparison files and regeneration remain out of scope. Keep this task blocked until the conductor verifies the committed native fn-195.5 and fn-197 checkpoints through .3 in umpire. Fn-197's mini/Temporal and actual lifting obligations must first be explicitly split/re-scoped; those held proofs are not native admission requirements and are not credited complete. Capture the final per-sample baseline only after these native checkpoints.

### Admission hold and release

The same-spec dependency on fn-195.5 remains. The explicit blocked status prevents ready/start from offering polish while fn-197 still changes authoring or the candidate ReliableBroadcast sample. No cross-spec task edge or whole-spec cycle is added. After verifying integrated native closures, record their exact commits in this durable description/admission record, then release the never-started task with flowctl task reset (without cascade) and recheck ready. Reset clears Evidence, so preserve prerequisite receipts outside that section. The conductor reconciles the old Blocked summary at release; no prior implementation work is discarded.
## Acceptance
- [ ] Admission records integrated native fn-195.5 and fn-197 checkpoints through .3 before the fresh final sample baseline; actual Temporal/lifting proofs remain held rather than credited.
- [ ] All six primary native samples and supporting author-facing Scala fragments have a recorded clarity review; concrete unnecessary scaffolding is removed without gratuitous edits.
- [ ] Before/after evidence preserves starts, domains, transition behavior, outcomes/facts, claims, bounds, counterexamples and realization ownership; cuts/unknowns remain explicit.
- [ ] Samples consistently demonstrate the settled class-only primary declarations and ModelSuite authoring style, with no retired migration adapters.
- [ ] Test purposes remain intact; full native/lint and fresh source/separate-library realization gates pass.
- [ ] Related examples/links stay accurate; exact upstream comparison files, historical snapshots and held consumers are unchanged.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: