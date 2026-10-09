---
satisfies: [R2, R4]
---
# fn-155-name-the-standalone-activitys-repeated.3 Product Recorded conversions and shared rejection reasons

## Description
Implements spec section B and the cross-level parts of D: Product single-fact steps use the shared phase effects; rejection reasons citing the same source are defined once; the worker/By-ID must-match comment; By-ID failure reuses the kind's examples. It depends on task 2 because both edit the System rules.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/product/Product.scala`, `Standalone.scala` (reasons, examples), the System-machine owner (rule `because` references)
**Touches:** [model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/Standalone.scala, model/temporal/features/activity/standalone/system/System.scala] (System rule `because` references only; never Dispatch* or Realization.scala)

### Approach
- **Single-fact Product steps.** Convert only these: `retryPaused`, `resetAppliedPaused` and other single-fact, reasonless arms become `pause(s)`/`retry(s)`. Leave `heartbeatExpires` and other multi-fact or reasoned steps in method form.
- **Shared reasons.** Put them in `Standalone.scala`, or wherever task 1's probe showed a cross-object `.because` lifts. Each level keeps its baseline text. Where the texts differ today, keep two values, or one value and an explicit mapping entry.
- **Worker/By-ID comment.** Add a comment at both blocks naming which rows must match. Do not merge the blocks, since `StandaloneActivityPins.test.scala:1050-1099` pins binding order.
- **By-ID examples.** `respondFailedByID` reuses `temporal.features.activity.worker.respondFailed`'s examples if task 1's probe accepted it. Otherwise keep them repeated and record the refusal in `mapping.md` (spec R4). Check `model/cases` for changed Case names.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/product/Product.scala:85-160` — effects to convert
- `model/temporal/features/activity/Activity.scala` — kind's `respondFailed` examples
- `model/framework/Syntax.scala:105-160` — `Recorded`, `Draft`, `effect`

### Acceptance
- [ ] The Pins step-table test passes unchanged, including fact order
- [ ] `project.py` on a scratch lift shows only positions, mapped identities and recorded structural-review entries
- [ ] `make umpire-check-cases` shows no Case name change, or the change is mapped

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
