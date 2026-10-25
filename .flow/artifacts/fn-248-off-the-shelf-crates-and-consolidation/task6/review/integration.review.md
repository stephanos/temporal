I’ll read the task spec first, then trace the diff through the decoder’s callers and focused test evidence.
The derived envelope preserves the decoder’s strict checks, including duplicate-key refusal and the distinction between absent and null `params`. Committed evidence records 28 focused tests and a required-mode live replay passing; I’m checking the caller path and whether the focused suite can run in this sandbox.
No surviving findings. The derived envelope preserves strict nested decoding, duplicate rejection, required fields, null refusal, resource budgets, and typed replay integration.

Verification: existing test binaries passed 3 new controls, 5 decoder tests, and 6 replay tests. Fresh Cargo verification was blocked by the read-only build-lock path. Committed joined evidence records 28 focused tests and required-mode live replay passing.

## Requirements coverage

Coverage is scoped to task .6.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Derived strict envelope; presence-aware `params`; existing locked derive dependency; refusal controls retained. Other crate replacements belong to sibling tasks. |
| R2 | deferred | Task .2. |
| R3 | deferred | Task .3. |
| R4 | deferred | Task .1. |
| R5 | deferred | Final integration task .7; focused joined validation is recorded here. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>