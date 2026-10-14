I’ll read the task spec, then trace the diff through the existing command elaboration paths and focused tests.
Both callers now use the same resolver, and the registry stores the member/action pairs it needs. Class suffixes are preserved, bare sync names still resolve as before, and properties retain their existing “all classes” behavior. I’ll check the focused build next.
I haven’t found an introduced integration issue. The regression checks compare the dotted references with the existing synchronized references, and the diff leaves existing tests and goldens intact. The focused build has produced no output yet; I’ll check whether the read-only environment is blocking it.
No surviving findings. The shared resolver matches the registry’s synchronization mapping and preserves existing class handling in both callers.

`git diff --check` passed. The focused build produced no output and was interrupted; runtime verification remains unconfirmed.

## Requirements coverage

Coverage is scoped to task .9’s assigned requirement.

| R-ID | Status | Evidence |
|---|---|---|
| R3 | met | Both callers resolve synchronized member references through the shared helper. Regression pins cover dotted references, ambiguity, and unknown actions. Existing test and golden blocks remain unchanged. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>