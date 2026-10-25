I’ll read the task specification first, then trace the changed code and tests against its acceptance criteria.
The runtime now retains singleton witness prefixes and restricts Reach to exact targets, leaving partial-target integration for the later ambiguity task. I’m checking search order, bound handling, and whether retained evidence still validates after transport state changes.
The focused test command could not start because Cargo’s build cache is on a read-only filesystem. I’m continuing with source-level checks; the added tests cover typed facts, evidence cursors, publication retry, prefix bounds, and Current reports after nonempty histories.
No surviving findings in the reviewed changes.

`git diff --check` passed. The focused test command could not start because Cargo’s build cache is on a read-only filesystem; runtime verification remains unconfirmed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Task .1 implements bounded Exact Reach, shared traversal, typed evidence, transactional singleton prefixes, and Kettle migration. Ambiguity and current-format consumer completion remain assigned to .8–.10. |
| R2 | deferred | Capability migration belongs to .7. |
| R3 | deferred | Shared catalog and lens work belongs to .12/.2. |
| R4 | deferred | Activity binding migration belongs to .3/.11. |
| R5 | deferred | Protocol and refinement work belongs to .13/.4. |
| R6 | deferred | Executor and policy work belongs to .14/.5. |
| R7 | deferred | Count and documentation gates belong to .3/.6. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge within task .1’s scope.