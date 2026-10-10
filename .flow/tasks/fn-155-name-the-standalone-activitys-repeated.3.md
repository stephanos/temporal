---
satisfies: [R2, R4]
---
# fn-155-name-the-standalone-activitys-repeated.3 Product Recorded conversions and shared rejection reasons

## Description
Implements spec section B and the cross-level parts of D: Product single-fact steps use the shared phase effects; rejection reasons citing the same source are defined once; the worker/By-ID must-match comment; By-ID failure reuses the kind's example text values. It depends on task 2 because both edit the System rules.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/product/Product.scala`, `Standalone.scala` (reasons, examples), `model/temporal/features/activity/Activity.scala` (shared example text values), the System-machine owner (rule `because` references)
**Touches:** [model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/Standalone.scala, model/temporal/features/activity/Activity.scala, model/temporal/features/activity/standalone/system/System.scala]

Scope: (System rule `because` references only; Activity.scala example text values only; never Dispatch* or Realization.scala)

### Approach
- **Single-fact Product steps.** Convert only these: `retryPaused`, `resetAppliedPaused` and other single-fact, reasonless arms become `pause(s)`/`retry(s)`. Leave `heartbeatExpires` and other multi-fact or reasoned steps in method form.
- **Shared reasons.** Put them in `Standalone.scala`, or wherever task 1's probe showed a cross-object `.because` lifts. Each level keeps its baseline text. Where the texts differ today, keep two values, or one value and an explicit mapping entry.
- **Worker/By-ID comment.** Add a comment at both blocks naming which rows must match. Do not merge the blocks, since `StandaloneActivityPins.test.scala:1050-1099` pins binding order.
- **By-ID examples.** Task 1 proved shared kind-level example text values in `Activity.scala`, referenced by both `worker.respondFailed` and `service.respondFailedByID`, preserve the complete lifted example lists and order. Use that exact form, retaining the existing two example declarations per action. It did not prove an API for reusing an action's example collection; do not invent one. Follow the pinned scratch patch and record any refusal/fallback in `mapping.md` (spec R4). Check `model/cases` for changed Case names.

### Investigation targets
**Required:**
- `model/temporal/features/activity/standalone/product/Product.scala:85-160` — effects to convert
- `model/temporal/features/activity/Activity.scala` — kind's `respondFailed` examples
- `model/framework/Syntax.scala:105-160` — `Recorded`, `Draft`, `effect`
- `.flow/tmp/fn-155/task1/probes/combined-source.patch` and `.flow/tmp/fn-155/mapping.md` in task 1's sealed workspace — proven shared example text values, complete lift and projection evidence

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
