## Re-review: fn-109 task 12 — detached `Artifact` vs owned `Opened` handle

I still cannot run the Quick commands in this session (no shell). Resolutions below are verified by reading the current files on disk and the updated diff.

Prior finding #1: fixed
Prior finding #2: not-fixed

**Finding #1 (nil `ReadPayload` error text).** `open.go:231` now guards `opened == nil` and returns `artifact is not open`, matching `OpenPayload`, `CopyPayload` and `TargetSharing`. A closed non-nil handle keeps the original order: listing check, bound check, then the open check inside `OpenPayload`, so documented error precedence for real handles is unchanged. The test at `opened_test.go:113` was simplified to assert one error text for both the closed and nil handles, and the explanatory comment is gone. `go-interface-changes.md` records the new nil behavior.

**Finding #2 (repeated `Manifest()` calls in test expressions).** Unchanged. The implementer's disposition in `review-round-1.md` says the migrated test lines keep their mechanical shape and that production sites bind one copy. That is accurate: every production call site binds `manifest := opened.Manifest()` once. This was and remains a P3 test-only style note. It does not block.

**New issues introduced by the fix.** None. The only code delta since round 1 is the two-line nil guard, the test tightening, and documentation. No new finding at Major or above.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R13 | met | Unchanged from round 1. Distinct `Artifact` and `*Opened` types, private manifest with deep-copy accessors, full test matrix for open/close, use after close (now uniform across all four payload methods), directory replacement, snapshot isolation, and the rejection matrix. All consumers migrated and recorded. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>

VERDICT=SHIP
