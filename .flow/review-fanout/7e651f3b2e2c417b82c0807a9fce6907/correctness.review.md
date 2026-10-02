I’ll read the task spec first, then inspect the diff and affected callers for correctness and spec coverage.

The runtime edit moves the same instance-detection check into a shared helper. The Makefile also replaces recursive pin-check calls with a shell function so dry runs can list the check without executing it. I’m checking both callers and the pin-check failure path.

The regression dry run exits successfully and lists both pin checks; the real manifest check also passes. The focused Go test cannot start because the read-only sandbox blocks its build directory. I’m finishing the spec-to-doc comparison and checking the additional planning files for conflicts with this task.

No introduced correctness findings.

Verified that the regression dry run succeeds, the real Veil pin passes, and a mismatched manifest fails with the expected diagnostic. The focused Go test could not start because the read-only sandbox prevents creating its build directory; closing gates remain unverified.

## Requirements coverage

Coverage scoped to fn-89.6’s assigned requirements.

| R-ID | Status | Evidence |
|---|---|---|
| R9 | partial | GOV-02 drafts preserve approved text; listed docs describe Rule instances. ORDER’s delivered update remains pending close. |
| R10 | partial | Conformance rejection exists; pin-check behavior verified. Full closing gates remain unverified. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>