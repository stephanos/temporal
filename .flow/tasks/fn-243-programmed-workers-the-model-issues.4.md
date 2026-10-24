---
satisfies: [R5]
---
# fn-243-programmed-workers-the-model-issues.4 DESIGN.md R40 architecture checkpoint and code pointers; unheld proof deferred

## Description
After .2 and .3. fn-228.7 writes DESIGN.md's Runner and Run section after this spec, so R40 is written once in its final form there. This task checks that section states: a worker in any language follows a Model-issued program and the Run records what happened; the Shim is superseded; the proposal vocabulary is action, bootstrap, read; the hold fault; the executor kinds and where they live; and that the design note is marked design of record. It adds pointers from `programmed-workers.md` to the shipped code and goldens.
## Acceptance
- [ ] Check R40 against the actually admitted held-before-start and terminal-cancellation slice, name the exact temporary refusal and fn-243.5 obligation, and avoid claiming full unheld cancellation conformance or closing the parent spec.
- [ ] DESIGN.md states R40 in its final form with the proposal vocabulary, hold fault and executor kinds; the design note points to the code
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
