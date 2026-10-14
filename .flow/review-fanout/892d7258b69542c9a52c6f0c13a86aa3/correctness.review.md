I’ll read the task spec first, then review the diff and affected callers for correctness and logic issues.

The removed APIs have no remaining callers, and the activation calculation still enforces the admission ceiling after its unused result field was removed. The replay change stores the same prepared contract used by the monitor. The race test could not start because the read-only sandbox blocks Go’s temporary build directory; I’m finishing the static checks and reviewing the remaining gate commands.

No surviving correctness or logic findings. The changed paths preserve admission checks, environment resolution, immutable views, and replay through the prepared contract.

`git diff --check` passed. Verification remains incomplete:

- Race tests: blocked by sandbox restrictions on temporary directories.
- Conformance: blocked by sandbox restrictions on Lean build artifacts.
- Lint: stopped because another golangci-lint instance was running.

## Requirements coverage

Coverage is scoped to task fn-94.4.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Owned by fn-94.1/.3. |
| R2 | met | Listed core APIs removed; retained aliases support exposed types. Driver removals and final deadcode gate belong to fn-94.5/.17. |
| R3–R7 | deferred | Assigned to other implementation tasks. |
| R8 | partial | Fixtures unchanged; verification attempts blocked as detailed above. |
| R9–R10 | deferred | Wire changes and final measurements belong to later tasks. |
| R11 | met | Identity goldens unchanged in this diff. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>