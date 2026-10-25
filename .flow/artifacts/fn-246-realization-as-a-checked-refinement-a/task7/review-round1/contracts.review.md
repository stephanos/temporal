I’ll read the task spec first, then review the diff for agreement between the stated contracts, implementation, and tests.
The capability migration largely preserves the existing runtime transaction path. I’m checking whether evidence validation happens before feature handlers run, and whether the consumer tests exercise the promised generic, renamed, and callback cases.
The focused test command cannot start because Cargo’s build cache is read-only in this sandbox. I’m continuing the source review, including the catalog changes and test assertions, and will distinguish recorded test evidence from checks I could run here.
I found a contract mismatch in the new evidence adapter: it converts admission refusals into search rejections. The runtime then publishes an `Unexplained` judgment and freezes the run, instead of returning the refusal without committing state. I’m tracing that path and checking the tests before assigning severity.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/umpire/protocol/runtime/src/runtime.rs:310`
- **R-IDs:** [R2]
- **Problem:** `Err(_) => EvidenceMatch::Reject` discards typed evidence admission refusals. For a `Reach` projection, this becomes `Ok(None)` in `reach`, after which `judge` prepares an `Unexplained` judgment and freezes the run. Malformed or unsupported evidence therefore becomes a committed negative judgment instead of the refusal required by R2. The migrated `NoPredicate` control already exercises this conversion: its default admission returns `Unsupported`, while the test expects `Unexplained`.
- **Suggestion:** Propagate admission errors through Reach and prefix validation as `Result<EvidenceMatch, Refusal>`. Represent an admitted observation with no matching predicate separately as `Ok(EvidenceMatch::Reject)`. Add a control asserting that failed evidence admission preserves the refusal and leaves the run unchanged.

Focused Cargo verification could not start because the sandbox’s build cache is read-only. `git diff --check` passed, and the checked artifact hashes matched.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Frontier and protocol completion belong to tasks .8–.10; .7 migrates existing singleton controls. |
| R2 | partial | Capabilities, fillers, consumers and compiler controls are present; admission refusals are discarded. |
| R3 | deferred | Registry and shared-layer completion belong to .11, .12 and .2. |
| R4 | deferred | Activity migration and live parity belong to .3 and .6. |
| R5 | deferred | Protocol and refinement proofs belong to .13, .4 and integration tasks. |
| R6 | deferred | Executor capabilities and policy enumeration belong to .14 and .5. |
| R7 | deferred | Counts and author documentation belong to .3 and .6. |

Unaddressed R-IDs: [R2]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R2"]}
```

<verdict>NEEDS_WORK</verdict>