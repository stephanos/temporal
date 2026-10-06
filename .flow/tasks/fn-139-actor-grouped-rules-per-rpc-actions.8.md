---
satisfies: [R10]
---
# fn-139-actor-grouped-rules-per-rpc-actions.8 Rejection to RPC status code table and conformance check

## Description
Adds the one table, shared by every Model, from each `Rejection` to the RPC status code a Run observes, in the realization layer. Conformance then checks each rejecting step's observed code against it, replacing per-realization answer strings such as the activity's `closedAnswer = "activity_notFound"` (R10). Runs after the DSL batch's regeneration, with fn-133.8's carrier metadata already in place.

**Size:** M (split it at start if the Go side needs a new IR field and a reader change as well as the conformance check)
**Files:** model/temporal/realize/ (new shared table, e.g. Rejections.scala), the realization metadata path fn-133.8 establishes (Scala carrier and IR export), model/temporal/features/activity/standalone/system/WithTaskQueue.scala (`closedAnswer` removed), any other realization with a rejection answer string, tools/umpire/conformance/ (expected-code check and tests), common/testing/testpilot/temporal/ (where a Run's observed status code is captured, if it is not captured yet)
**Touches:** [model/temporal/realize/**, model/temporal/features/**, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, tools/umpire/**, common/testing/testpilot/temporal/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- The table is Scala, in the Temporal realization layer, written as an exhaustive `match` over `Rejection` with no wildcard, so a new case without a code does not compile. `notFound` → NOT_FOUND, `alreadyExists` → ALREADY_EXISTS, `failedPrecondition` → FAILED_PRECONDITION, `invalidArgument` → INVALID_ARGUMENT. Cite the server sources in comments (spec, Resolved via Research, practice-scout).
- Carry the table to Go through the realization metadata fn-133.8 adds to the IR, not through a second Go-side table. If that path cannot carry it, a new IR field is needed (proto, lifter, Go reader). Stop, split the task, and say so.
- Conformance: for each step whose Model outcome is `rejected(r)`, compare the observed gRPC status code with the table's code for `r`. A mismatch fails the step's conformance, naming both codes. Compare codes only, never message text. Also cover the reverse cases: an accepted step that observes an error, and a rejecting step that observes OK.
- Find where a Run's error is captured today (common/testing/testpilot/temporal/server/session.go:171-177 maps context errors to codes; control/activity.go handles ObsoleteMatchingTask). Capture the observed status code where steps are recorded, if it is not captured yet.
- Remove `closedAnswer` and any other per-realization rejection strings once the check replaces them.
- A Go test proves the table is total: every `Rejection` case in the IR has a code.

### Investigation targets
**Required:**
- model/temporal/features/activity/standalone/system/WithTaskQueue.scala:60-110, 180-195 (`closedAnswer`)
- tools/umpire/conformance/ (package layout, activity_test.go)
- common/testing/testpilot/temporal/server/session.go:160-180 and control/activity.go
- fn-133.8's realization metadata in the IR (read its task and done summary)
- proto/internal/temporal/server/api/umpire/v1/ir.proto (realization section)

## Acceptance
- [ ] One Scala table maps every `Rejection` to an RPC status code. Removing a case's code fails compilation, and a Go test checks that the exported table is total.
- [ ] Conformance fails a rejecting step whose observed code differs from the table's, naming the expected and observed codes, with a Go test for the mismatch, for an accepted step observing an error, and for a rejecting step observing OK.
- [ ] No realization carries its own rejection answer string (`closedAnswer` is gone).
- [ ] The Go unit tests for the touched packages and the Scala munit and fixture suites pass. The IR change is declared in the commit message for the batch diff check.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
