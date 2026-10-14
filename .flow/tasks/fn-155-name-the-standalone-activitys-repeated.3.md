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
# Product integration

Product's single-fact conversions use existing Recorded effects, ten exact rejection texts have shared declarations, and worker/By-ID actions retain their ordered failure examples through the proved text-val form. Pending-pause and multi-fact branches remain explicit; must-match comments preserve the actual By-ID differences. Only Activity.scala, Standalone.scala, Product.scala and System.scala rule Because references/comments change.

Worker source 96c554b1ab70fce0ccddf0888663feeac6dcc572 is reachable through non-rewriting merge f5d28cc3ca28660c6a4ec2740ae6eea7fcad4af7. Normalized base b5736f7cc3359c495f0b36e72bc0c5b54ab1dbde, reviewed head 12344c65c04bb5d6575ea261929f2e417c767f33, integrated finalized ledger af6b810cf484732cf54b0f11186345f4ba368f27. Root independently verified all 785 proof files, seal SHA256 9da48370bb935976bc413f19de39a1ea67fe3dd5a63079a622882c8c57e239e2.

Unchanged Pins passed. Sixteen original authored-root partitions were freshly lifted and every declaration checked against the complete fresh candidate with positions only removed. Exactly twenty fresh Activity Cases match all original raw bytes without a coordinate exception. Original Nexus and manifest bytes supply inventory-pin credit only.

The raw projection remains RED: 12 race, 12 record and 21 primary paths. A separate proof accounts for six Product reset/unpause bodies across all nine complete states with action parameters arbitrary; 24 wrong-landing/status/reason/order controls reject. The inherited System resetSettles nine paths remain explicit, independently rechecked across thirteen phases with its wrong-landing negative. All other projected fields are exact. Direct helper behavior outside enabled guards is not claimed equal. Initial formatting and proof-wrapper failures remain archived beside corrected passing evidence.

Actual codex:gpt-6.1-sol:high correctness, contracts and integration draws all returned SHIP without findings (same GPT family). RID bbe2dfc4ec654428b7cb6da4563b0d50; receipt at agent/fn155-task3-review/.flow/tmp/fn155-task3-review/review-receipt.json, SHA256 46e0a15aed0a25032d341aefff11cf301e46013fc2b0270e411344708a32832c. All eight predecessor attempts remain an exact prefix. A reviewer's fresh Pins attempt was read-only-blocked and supplies no new test credit.

Integrated 56 Activity tests passed in 77.794 seconds and model lint in 103.076 seconds at f5d28cc3. After the finalized review ledger was joined, Flow honored both exact scoped receipts (fn155-task3-focused and fn155-task3-lint), since intervening changes were Flow metadata only. Root independently rechecked every recorded source hash immediately before completion; no source or test fix occurred after those commands. This is verified reuse, not a new execution or full-gate claim.

stage: memory-capture - skipped(clean first-round SHIP; no review fixes)
stage: plan-sync - skipped(policy: rolling route)

Worker evidence used pre-Task5 Realization; task 6 owes fresh joined production outputs and the full preservation/gate disposition. Original canonical RED/unobserved checks and named fn154/fn157/Batch5 deferrals remain strict. No canonical generation, full-suite, live or matching replay credit is claimed.
## Evidence
- Commits: 96c554b1ab70fce0ccddf0888663feeac6dcc572, f5d28cc3ca28660c6a4ec2740ae6eea7fcad4af7, 12344c65c04bb5d6575ea261929f2e417c767f33, af6b810cf484732cf54b0f11186345f4ba368f27
- Tests: mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/framework model/temporal --require-tests --test-only framework.*Activity* [56 tests passed; exact quoted command recorded in task3-pre-review-focused.json], make lint-model [passed; task3-pre-review-lint.json], Post-review Flow gate check honored fn155-task3-focused and fn155-task3-lint at f5d28cc3, only Flow metadata changed; root independently rechecked all source hashes immediately before completion, All 20 unique freshly lowered Activity Cases strict raw-byte exact; no coordinate exceptions; 16 fresh authored-root partition declarations match complete candidate, Six Product rule bodies across nine complete states, action parameters arbitrary; 24 actual-verifier mutations rejected; retained System resetSettles 13-phase proof and wrong-landing negative independently rechecked, Raw projection RED retained and accounted: 12 race, 12 record, 21 primary paths; every other complete projected field exact, Root independently verified all 785 private proof files, seal SHA256 9da48370bb935976bc413f19de39a1ea67fe3dd5a63079a622882c8c57e239e2, Actual Codex gpt-6.1-sol high three-axis SHIP, no findings; RID bbe2dfc4ec654428b7cb6da4563b0d50; all eight predecessor attempts preserved exactly, Task6 owes fresh joined artifacts/full gates; original strict canonical RED/deferred checks retained; no full-generation, full-suite, live or matching replay credit
- PRs: