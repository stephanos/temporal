I’ll read the task spec first, then review the diff for agreement between the implementation, tests, and documented promises.
The RPC mapping follows the task contract: polling and Answer leave the state unchanged, and Judge uses the heating receipt to advance one action. I’m checking the selected-start reset behavior, rejection paths, and build integration next.
No surviving findings.

The implementation matches the selected-start and RPC projection contracts. The canonical Kettle model is unchanged, and the reference exploration golden is byte-identical.

Verification: 19 tests passed using existing compiled Rust test binaries. Fresh Cargo and Go test runs were blocked by the read-only filesystem.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Delivered in task .1; this change consumes the shared envelopes. |
| R2 | met | Selected-start validation, transactional Reset controls, independent Go server, exact Judge/Answer mapping, transcript golden, and build integration. |
| R3 | deferred | Catalog and Annotate integration belong to task .3. |
| R4 | deferred | Mutants intentionally remain inactive until task .4. |
| R5 | deferred | Second-catalog validation belongs to task .3. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>