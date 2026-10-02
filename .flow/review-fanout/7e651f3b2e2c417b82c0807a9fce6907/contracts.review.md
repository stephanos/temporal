I’ll read the task specification first, then review the diff for agreement between the code, tests, and documented contracts.
The runtime change extracts the existing instance check into a shared helper. I’m checking the new documentation against admission and evaluation, and tracing the Makefile change that keeps the Veil check from executing during a dry run. The range also includes fn-94 planning files, which I’m checking for contract contradictions.
The documented instance types, ordering, state isolation, and producer fallback match the code I traced. The regression dry run exits successfully and lists both Veil checks. The focused Go test cannot start because the read-only sandbox blocks creation of its build directory, so I can’t confirm the test result here.
No surviving findings.

The regression dry run passes. The extracted Veil checker accepts the pinned manifest and rejects a planted mismatch with the expected diagnostic. The focused Go test could not start because the read-only sandbox prevents build-directory creation.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R9 | met | GOV-02 drafts preserve original text; updated documentation agrees with instance admission, evaluation, and producer behavior. |
| R10 | partial | Conformance rejection exists; dry-run fix verified. Full closing gates remain unverified here. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>