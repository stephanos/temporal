I’ll read the task spec first, then trace the diff and focused test evidence for correctness and edge cases.
The diff adds the Kettle catalog, live annotation wiring and generic probe path. I’m tracing how scalar oneofs and text maps behave in the shared code, then checking the scoped limits and probe classifications against the server.
The focused native test binaries passed: scalar carriers, Kettle annotations, descriptor exhaustiveness and probe correlation. Cargo and Go test commands cannot create build files in this read-only sandbox, so I’m using those binaries and the committed verification logs for test evidence.
No correctness findings survived review.

Eight focused native tests passed using existing binaries. Fresh Cargo/Go runs were blocked by the read-only filesystem; committed logs record passing live tests and `make model-check`.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Shared tooling retained; Temporal regression logs pass. |
| R2 | met | Existing live transcript preserved; all six RPCs annotated in both directions. |
| R3 | met | Exhaustive catalog, compile-fail controls, scoped limits, coverage and actual-wire gap goldens. |
| R4 | deferred | Mutant detection belongs to task .4. |
| R5 | met | Three generic gaps documented and implemented with both consumers covered. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge