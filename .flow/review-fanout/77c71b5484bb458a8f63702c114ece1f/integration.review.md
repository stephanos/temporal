I’ll read the task spec first, then review the diff and its integration with unchanged code, using only focused verification.

The shared helpers keep core test support separate from facade-dependent fixtures. I’m checking that the replacements preserve the old fakes’ behavior and test coverage. The race suite and dependency check could not start because the read-only sandbox blocks Go’s temporary and cache writes.

The replacements retain the delivery lifecycle assertions and strengthen the composite-session close-order check. I’m now checking unchanged callers for removed helper references and for behavior that depended on the old fixture limits.

No blocking introduced findings. Shared helpers preserve the core import boundary, and migrated callers retain their lifecycle and ownership assertions.

Verification was limited: race tests and dependency listing were blocked by sandbox write restrictions; lint exited reporting another golangci-lint process running. No passing checks are claimed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | met | Task-scoped helpers and fakes consolidated; facade-dependent fixtures isolated in `facadetest`. Live-test consolidation belongs to fn-94.15. |
| R8 | deferred | Focused verification attempted but blocked by the environment. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>