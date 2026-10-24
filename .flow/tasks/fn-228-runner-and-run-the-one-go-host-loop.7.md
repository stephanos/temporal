---
satisfies: [R7]
---
# fn-228-runner-and-run-the-one-go-host-loop.7 After committed fn-243.2: Runner and Run write-up, R33-R39 discharge and R40 proof checkpoint

## Description
Add the DESIGN.md section recording the vocabulary sentence, Propose, the Run schema, per-package counts before and after, live timings and make model-time; discharge independently earned R33 to R39 rows and record the delivered R40 architecture with its remaining unheld cancellation obligation assigned to fn-243.5; update README and MIGRATE for the runner package and renames. MILESTONES.md is the root agent's.

### Required integration contract

Document the actual bootstrap and polling protocol, exact replay event schema, immutable publication key, progress contract and before/after counts including relocated Runner code. Update current names and milestone references; retain historical evidence. R33-R40 close only with the recorded real live/replay and deletion-coverage proof.

### Dependency change (2026-10-23, revised 2026-10-24)

fn-228.5 is superseded by fn-243 (programmed workers) and closed with nothing built. Within this spec this task depends on .6; across specs it waits for fn-244.2 (done) and fn-243.2, so R40 is written once, in its final form: a worker in any language follows a Model-issued program, the Run records the executor kind and what happened, the Shim is superseded, the proposal vocabulary is action, bootstrap and read, and the hold fault replaces the worker gate. R37's live callback path is recorded as retired for Temporal (Answer stays for other systems). fn-243.4 checks the section afterwards and adds the design-note pointers.

### Ordering note (2026-10-24)

Runs after fn-244.2, which moves the Go Runner out of the generated protobuf tree, so the write-up names its final path.
## Acceptance
- [ ] Document the exact current atomic-Model refusal for unheld open-state cancellation, held-before-start and terminal support, and the deferred fn-243.5 proof. Keep fn-228's final R40/R7 obligation open until that evidence is committed; this task is the architecture documentation checkpoint.
- [ ] DESIGN.md section and R33 to R40 rows updated
- [ ] README and MIGRATE updated
- [ ] model-time pasted

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
