I’ll read the task specification first, then trace the changed code across its callers and shared state, focusing on integration issues introduced by this range.
The main integration path runs from `act` transport capture through the worker bridge to the raw and SDK continuation loops. I’m checking whether the real SDK transport preserves the correlation and cancellation assumptions exercised by the new tests.
The focused Go tests cannot start because the sandbox blocks creation of Go’s build directory. I’m continuing with source tracing and the committed evidence, including the pinned SDK code and server poll responses.
I found no integration defect meeting the reporting threshold. The checked code preserves exact assignment matching, advances history only after a correlated completion, and joins blocked work on shutdown. I also verified all 38 checkpoint source hashes and the joined validation artifact hashes; none mismatched.
No surviving findings.

Source tracing covered program admission, transport capture, assignment correlation, raw/SDK continuations, cancellation, shutdown and Runner fault evidence. All 38 checkpoint source hashes and the checked validation artifact hashes matched.

Focused Go tests could not start because the read-only sandbox prevented creation of Go’s build directory. Committed focused test evidence was reviewed.

## Requirements coverage

Coverage is scoped to task .14.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Assigned to other epic tasks. |
| R2 | deferred | Full capability migration belongs to other tasks. |
| R3 | deferred | Shared-layer consolidation remains separately assigned. |
| R4 | deferred | Activity migration remains separately assigned. |
| R5 | deferred | Protocol refinement and proof renewal remain separately assigned. |
| R6 | met | Exact-history codec, closure admission, correlated raw/SDK continuation and focused controls implement .14’s capability scope. Strategy enumeration and live witnesses remain assigned downstream. |
| R7 | deferred | Final counts and documentation remain separately assigned. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>