I’ll read the task spec first, then review the diff and affected callers for contract and consistency issues.
The diff removes accessors and redirects their tests to retained code paths. Activation-limit validation still uses the calculation whose result is no longer stored. I’m checking the replay wiring and remaining callers; unrelated working-tree edits are outside this review.
No surviving findings. The removed accessors have no remaining Go callers. Replay retains the same prepared contract, activation-limit checks remain intact, and the revised tests exercise retained production paths.

`git diff --check` passed. Fixtures and goldens are unchanged. Go tests could not start because the read-only sandbox denied temporary-directory creation; conformance regeneration and lint were not run.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by fn-94.1/.3. |
| R2 | partial | Core removals verified; Driver removals and final deadcode sweep belong to fn-94.5/.17. |
| R3 | deferred | Owned by fn-94.7/.8. |
| R4 | deferred | Owned by fn-94.6/.9/.10. |
| R5 | deferred | Owned by fn-94.11/.12. |
| R6 | deferred | Owned by fn-94.13. |
| R7 | deferred | Owned by fn-94.14/.15. |
| R8 | partial | Fixtures unchanged; runtime verification blocked by sandbox permissions. |
| R9 | deferred | Owned by fn-94.16. |
| R10 | deferred | Final measurement belongs to fn-94.17. |
| R11 | met | This change preserves existing identity goldens. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>