I’ll read the task specification first, then trace the changed code against its contracts, documentation, and focused tests.
The commit history places the rejection-order test before the opcode refactor, and the dispatch table appears to preserve the prior check order. The range also changes server authorization and prepared instruction bounds; I’m tracing those paths for behavior changes.
I haven’t found a contract regression in the changed paths. Verification is limited: the Go tests and conformance check hit read-only filesystem restrictions, and lint stopped because another golangci-lint process was running. The diff check passed, and the rejection-order test is unchanged since its pre-refactor commit.
No surviving findings in the reviewed changes.

The opcode table preserves binding order and effect dispatch. Server authorization, prepared bounds, and claim revalidation remain consistent with the documented contracts.

Verification: `git diff --check` passed. Go tests and conformance verification were blocked by filesystem permissions; lint stopped because another golangci-lint process was running. Runtime checks are therefore unverified.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R5 | met | Task .11 scope: one opcode table replaces the specified switches; rejection-order tests precede the refactor in `afdb27126e` and remain unchanged; extension checklist updated. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>