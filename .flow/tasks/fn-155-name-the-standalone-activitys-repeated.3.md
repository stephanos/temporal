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

### Implementation evidence pointers

Worker evidence is retained at `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn155-task3/.flow/tmp/fn-155/task3/`: `handover-summary.md`, `handover-evidence.json`, `mapping.md`, `structural.py`, `structural-review.json`, `case-proof.json` and `proof-manifest.json`. The manifest seals 785 private files; SHA-256 `9da48370bb935976bc413f19de39a1ea67fe3dd5a63079a622882c8c57e239e2`. The conductor's integration receipt is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn-155-name-the-standalone-activitys-repeated/.flow/tmp/fn155-integration/task3-integrated.json`, binding worker source `96c554b1ab70fce0ccddf0888663feeac6dcc572` to integrated source `f5d28cc3ca28660c6a4ec2740ae6eea7fcad4af7` on normalized base `b5736f7cc3359c495f0b36e72bc0c5b54ab1dbde`.

The unchanged raw projection stays RED (12 race, 12 record, 21 standalone paths). The separate structural proof accounts for six Product rule bodies over all nine complete states with action inputs arbitrary and 24 rejected mutation controls, plus Task2's retained resetSettles nine-path, thirteen-phase proof and rejected wrong landing. Sixteen freshly lifted source partitions match the complete fresh Model declarations; all twenty unique fresh Activity Cases are raw-byte-exact to sealed originals, with no coordinate exception. Mapping records direct-helper differences outside enabled guards and preserves pending-pause and ordered multi-fact behavior.

Worker Pins and model lint are scoped GREEN. Canonical all-machine generation, managed-tree agreement, the original manifest, Nexus inventory, fixtures and full Go gates retain their recorded RED/unobserved or inventory-only status; no canonical, live or full-gate credit is claimed. Task6 owns joined-candidate regeneration and gates. Initial formatting and wrapper type-shape failures remain retained alongside corrected scoped evidence. Formal task review and closure remain pending.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
