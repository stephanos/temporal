I’ll read the task spec first, then review the diff and affected callers for contract and consistency issues.
The code change moves the existing ceiling check behind the public facade without changing either driver’s rejection path. I’m checking that the new helper’s documented behavior matches the shared validator, and that the documentation points to the right helpers.
I found one documentation contradiction: the new delivery summary says fn-94 preserved identity bytes, but its wire-change commit changed the catalog identity golden and re-recorded pinned Runs. The facade change appears behavior-preserving. Both the focused regression test and deadcode check were blocked because the read-only sandbox cannot create Go build directories.
- **Severity:** P3
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `.plans/UMPIRE4_ORDER.md:26`
- **R-IDs:** [R11]
- **Problem:** “no Run, Verdict or identity-byte change” incorrectly describes fn-94. Commit `2839a4ed2e` changed the catalog identity golden from `95533e4d…` to `1f757edd…` and re-recorded pinned Runs. The new summary hides the campaign’s intentional identity change.
- **Suggestion:** Qualify the preservation claim with lane G’s catalog-identity change and accompanying re-record.

The facade helper preserves both drivers’ existing validation and rejection behavior. No runtime defect found.

Verification: `git diff --check` passed. The focused boundary regression test and deadcode check could not execute because the read-only sandbox prevents creating Go build directories. Full gates were not run.

## Requirements coverage

Task fn-94.17 owns R2, R8 and R10; the other epic requirements belong to earlier tasks.

| R-ID | Status | Evidence |
|---|---|---|
| R2 | partial | Deadcode execution blocked by sandbox; cleanliness unverified. |
| R8 | partial | Forbidden driver imports removed; focused regression execution blocked. |
| R10 | partial | Measured production 19,375; tests 20,337; live 2,369; proto 1,411. Against fn-94.2’s baseline, production grew 26 lines and tests shrank 235; both floors missed. Final receipt remains TBD. |

The outstanding receipt is a nonblocking process observation.

Unaddressed R-IDs: []
Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>