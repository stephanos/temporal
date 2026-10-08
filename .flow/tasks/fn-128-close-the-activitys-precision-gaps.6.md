---
satisfies: [R6, R7, R8]
---
# fn-128-close-the-activitys-precision-gaps.6 Close: evidence map, live Cases, MILESTONES

## Description
Close the active activity batch for R6 and the deferred final proofs in R7/R8. This task owns shared generation, gates and the requirement maps after all source tasks settle. It shares the live run with existing fn-129.5.

**Size:** M
**Files:** `model/ir/**`, `model/cases/**`, existing generated mirrors and affected goldens, `model/README.md`, `tests/testcore/testpilot/README.md`, `common/testing/testpilot/temporal/README.md` if its caller-policy wording changes, `MILESTONES.md`, Flow completion receipts.
**Touches:** [model/ir/**, model/cases/**, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala, tools/umpire/**, tests/**, common/testing/testpilot/**, model/README.md, MILESTONES.md, .flow/specs/fn-128*, .flow/specs/fn-138*, .flow/specs/fn-129*, .flow/tasks/fn-128*, .flow/tasks/fn-138*, .flow/tasks/fn-129*]

## Approach

- The conductor first joins fn-138.1-3, seals its strict R3 comparison, then joins fn-129.1-4 and both correction tasks. Preserve the independently captured baseline at `/tmp/umpire-fn138-baseline.Glr07O/.flow/tmp/activity-batch/fn138-baseline`. Re-anchor final source commits before generation. Workers do not close specs or edit the shared page.
- Collect R6's divergence map and exact old/new tables, fact catalogs, Query answers, ordered receipts, Case IDs/bytes, Contracts, Programs, expectations, coordinates and independently recomputed hash inputs. Separate fn-138's sealed equivalence from later intentional coverage/fatal changes. Explain every difference; do not normalize coordinates or bless actual output as expected.
- Run one production generation after all sources. Compare its final pauseResume Case bytes/shape against the correction's authorized scratch input. Recompute its explicit Profile authorization if an approved source delta changes counts; reject unexplained drift and keep the defaults, canary and hard ceilings unchanged. Preserve all unchanged occurrence vectors and final raw attempt-count observations.
- Run the Model gate, full tagged Go suite, Model lint/format, Case/fixture/canary checks and fast Go lint. Serialize heavy commands with `/usr/bin/flock /tmp/umpire-heavy-gates.lock`; use separate Go checks with `MODEL_GATE_ARGS=--skip-go-checks`. Capture exact commands, status and wall time. Retain inherited resource failures as failures and prove justified serial recovery, never summarize an original red suite as green.
- Format the two inherited fully-qualified ScriptRejects presets if the full format gate requires it. Preserve their restored 208 diagnostic baseline, capture exact edited-span coordinate correspondence, and update only generator-produced shifted coordinates with diagnostic content/order unchanged.
- Run one shared generated-Case live pass and retain closed Runs, exact Case/Profile/Contract identities, conformance and Property assessments, replay results, charge ledgers and HTML. Use the existing expected-Run equality check for disposition, cleanup, Contract, conformance and every Property status/reason. Preserve the exact authored Property-only `explanationsDisagree` expectations on retry, retryAfterTimeout and retryExhaustion and every other unchanged authored expectation; do not rewrite them into satisfied or expand them to other outcomes. Fatal Property satisfaction, conformant retry paths with satisfied Contracts, and bounded pauseResume execution/replay remain required. Only the named ShutdownWorker-race exception permits an additional inconclusive. Resource-limit, neverEvaluated or any other unexpected status/reason fails. A real conformance disagreement goes to a human.
- Complete fresh implementation review and separate requirement-by-requirement completion reviews for fn-128, fn-138 and fn-129. The conductor owns evidence-bearing Flowdone and verified close status. Remove only completed specs from MILESTONES, preserve completed task history inside any open spec and leave unrelated captured changes intact. The later fn-129 plan must preserve existing .1-5 IDs and replace stale Deferred and 'once more' prose before execution.
- Update the generated-Case docs for accepted fatal classification, independent Property proof, the caller's exact bounded envelope and immutable preflight/run/replay Profile agreement. Keep broad generated API drift verification and new CI coverage out.

## Investigation targets

**Required** (read before closing):
- `.plans/ACTIVITY_MODEL_COMPARISON.md` and `MILESTONES.md`.
- `model/README.md:1483` and `tests/testcore/testpilot/README.md:44`.
- `tests/testpilot_generated_test.go` and `tests/testpilot_activity_control_test.go`.
- `tools/umpire/conformance/activity_test.go` and `tools/umpire/conformance/played_test.go`.

## Acceptance
- [ ] The final R6 map accounts for every divergence and changed artifact, including the original/adopted fn-138 comparison and subsequent R7/R8 changes, with independently checked exact identity inputs.
- [ ] One shared production generation, full gates and fresh implementation review pass; generated output matches the final authorized bounded Profile input. Required docs and fixture coordinate proofs are current.
- [ ] One shared live pass and offline replay exactly match every authored expected assessment/reason and satisfy the unchanged fatal Property, with exact charge/support/reason evidence. The three retained retry Property-only explanationsDisagree outcomes match their original expectations; no unexpected status/reason passes beyond the named ShutdownWorker-race exception.
- [ ] All three specs pass requirement-by-requirement completion reviews before verified closure. MILESTONES removes completed specs only and preserves unrelated captured work.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
