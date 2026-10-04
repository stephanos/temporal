I’ll read the task specification first, then review the changed documentation against the implementation and recorded evidence.
The change adds guidance and migration instructions without changing executable source. I’m checking the API names, backend limits, and replay claims against their owners, while keeping the recorded native qualification gaps separate from documentation correctness.
The document checks pass: links and fences resolve, requirement IDs and shell examples are unchanged, and the retained hashes match. I’m checking one remaining detail: whether `choice-replay=exact` proves a matching replay or only identifies the selected replay mode.
No surviving correctness findings.

The documented owners, API migrations, backend limits, and replay claims match the implementation. Fresh read-only checks verified 66 guide links, 15 evidence links, balanced fences, unchanged requirement IDs and shell examples, matching document hashes, and all 973 non-guide source hashes. Retained focused-test and `make validate` logs pass.

Native qualification and fn-105.5 closure remain explicitly open; this verdict covers the documentation changes.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R9 | met | Guides document current owners and migrations, distinguish the four evidence claims, preserve residual findings, and reuse fn-111 evidence. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
