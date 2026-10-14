I’ll read the task spec first, then review the diff and related code for contract and consistency issues.
Both resolvers now consult the same synchronization mapping, and the new tests compare dotted references with the existing bare sync names. I’m checking the ambiguity path and running the focused Compose test module.
I found no introduced contract or consistency issues. The focused build has produced no output, so I can’t confirm it passed; the diff’s whitespace check passed.
No surviving findings.

`git diff --check` passed. The focused Compose build produced no output and was interrupted; runtime validation remains unconfirmed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R3 | met | Within task .9’s scope, both resolvers share synchronization lookup and located ambiguity errors. Tests pin dotted Property and Scenario references, unknown references, and ambiguous resolution. Existing tests and goldens are unchanged. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>