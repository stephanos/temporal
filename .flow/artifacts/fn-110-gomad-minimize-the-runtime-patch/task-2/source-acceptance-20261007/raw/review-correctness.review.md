I’ll read the task specification first, then review the changed code and focused evidence against its acceptance criteria.
The specification explicitly waives the measured 642-byte U3 excess and transfers native qualification to fn-128/fn-149. I’m checking the retained scheduler code, preservation proof, and changed maintenance paths against the remaining source requirements.
The independent read-only comparison confirms exact patch application and alpha-renaming equivalence across all 20 patched files plus the runtime overlay. The three scheduler helpers also preserve the pre-extraction bodies and comments, with the required return and local-variable adaptations. The focused Go test command could not start because the sandbox prevents Go from creating its build directory.
The independent calculation also matches the generated choice identity and the whole-byte diagnostic fixture. The focused receipts record passing source inventories, regeneration/equivalence checks, portable conformance tests, and task-scoped lint. I found no introduced correctness issue in the reviewed code.
No surviving correctness findings.

Independent read-only checks confirmed exact patch application, alpha-renaming equivalence across 20 patched files and the runtime overlay, preservation of all three scheduler implementations, and correct generated identity/diagnostic hashes. Focused receipts record passing source checks. Go tests could not be rerun because the sandbox prevents creating build directories.

## Requirements coverage

Coverage is scoped to fn-110.2 and its size clause.

| R-ID | Status | Evidence |
| --- | --- | --- |
| R2 | met | Scheduler bodies, comments, lock transitions and upstream hooks preserved; named regressions registered. Native execution is transferred and deferred. |
| R6 | met | Complete alpha/gofmt equivalence; timer-presence behavior, field layout and retained contracts preserved. |
| R8 | met | U1: 24,117 bytes; U3: 33,294 bytes. The exact 642-byte U3 excess is explicitly waived. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>