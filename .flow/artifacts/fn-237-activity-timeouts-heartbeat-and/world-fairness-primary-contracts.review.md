I’ll read the task contract first, then trace the changed code, tests, and documentation for consistency.
The main path selects a scope before checking catalog membership in both backends. I’m checking rebinding, generated names, and metadata agreement next, where the new contract crosses existing boundaries.
The 6 front-end and 9 TLA binding tests pass using the existing test binaries. Cargo cannot rebuild them in this read-only sandbox, so that result covers the cached binaries. I’m checking the recorded gate evidence and remaining compatibility cases.
No reportable findings in the reviewed range. The implementation, tests, and authoring guide agree on conditional selection, rebinding, exact catalog membership, and metadata validation.

Validation: 22 focused tests passed using cached binaries; required-TLC branches were skipped locally. Cargo rebuilding was blocked by the read-only filesystem. Recorded required-TLC and full-gate evidence was inspected.

## Requirements coverage

Coverage is scoped to task fn-237.6’s assigned prerequisite.

| R-ID | Status | Evidence |
|---|---|---|
| R2 | met | World-only conditional admission, shared typed selection, independent native/TLA binding, exact fairness metadata, mutation controls, and maxima 1/2/3 fixture evidence. Activity semantics remain assigned to subsequent tasks. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge