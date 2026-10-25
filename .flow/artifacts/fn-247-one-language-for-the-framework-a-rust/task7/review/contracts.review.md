I’ll read the task specification first, then review the diff for agreement between the implementation, tests, and documented contracts.
The retired-name filter keeps the raw record format and external `grep -E` recheck. The lifecycle comparison and changed-edge control also remain intact. I’m checking the joined dispatcher and gate next to confirm how the new Rust controls run.
The focused Cargo checks could not start: the read-only sandbox prevents Cargo from opening its build lock. I’m using the committed test logs alongside source tracing, and checking that the retired verifier has no active callers.
I found a caller-level status regression. The Rust filter can exit before consuming the stream; when enough input remains, `git grep` gets SIGPIPE and the shell returns 141 instead of the filter’s error code. A focused reproduction with the actual shell function and a filter stub returning 9 confirmed this. The old Python filter consumed all input first.
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/retired_names.rs:85`
- **R-IDs**: [R2]
- **Problem**: Returning a grep error immediately leaves stdin unread. With sufficient pending input, upstream `git grep` receives SIGPIPE; `retired-names.sh:21` prioritizes that status and returns 141 instead of the filter’s error. The previous Python implementation consumed stdin before filtering. A caller reproduction using real `git grep` and an early-exiting filter stub returning 9 produced status 141. Existing tests exercise grep failures directly against xtask, missing this shell interaction.
- **Suggestion**: Drain the remaining input before returning the filter error, or adjust shell status precedence to preserve filter errors when upstream receives SIGPIPE. Add a shell control with input exceeding pipe capacity and an injected grep failure.

Focused Cargo tests and the bootstrap scan could not run because the read-only sandbox prevents build-lock/output creation. Committed joined logs record six passing filter controls and both lifecycle controls.

## Requirements coverage

Coverage below concerns task .7’s responsibilities.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Process implementation belongs to other tasks. |
| R2 | partial | Native filter preserves records and exemptions; shell error propagation has the finding above. |
| R3 | met | Python filter/tests and hash verifier/manifest removed; Rust controls added; lifecycle comparison and mutation control retained. |
| R4 | deferred | Broader documentation and final validation belong to other tasks. |
| R5 | deferred | Canonical validation pipeline belongs to final .3. |
| R6 | deferred | Process syscall migration belongs to other tasks. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>