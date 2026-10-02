I’ll read the task specification first, then review the diff and trace its interactions with unchanged callers and shared state.
The server now reads resolved limits from prepared instruction plans, and the claim-validity helper still runs at both lock boundaries. The diff also includes the opcode-table refactor, so I’m tracing its admission and scheduler behavior.
The focused race tests could not start because the read-only sandbox denied Go’s temporary build directory. I’m finishing the static integration checks; so far, I haven’t found a regression in the authority checks, cleanup indexing, or effect dispatch.
No surviving findings in the reviewed changes.

Verification: `git diff --check` passed. Focused race tests could not start because the sandbox denied temporary-directory creation. `make lint-code-fast` stopped because another golangci-lint process was running.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | met | Task-scoped server changes use shared ceiling validation and prepared instruction bounds; authority checks and both claim checks remain, with rejection tests. |
| R5 | partial | Opcode table drives binding and effect acceptance; other D1 consolidations belong to separate tasks. |
| R8 | partial | Focused verification attempted but blocked as described above. |
| R11 | partial | No golden artifacts changed; execution was blocked. |
| R1, R2, R3, R6, R7, R9, R10 | deferred | Other epic tasks; outside this review’s scope. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>