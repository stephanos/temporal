---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-116-decide-whether-the-umpire-irs.1 Prototype CEL expressions against both Models and write the report

## Description
Run the CEL spike end to end and write the report. Implements R1-R5.

**Size:** M
**Files:** an isolated prototype Go module and a small JVM harness under git-ignored `.flow/tmp/fn116-spike/`; the report `.plans/UMPIRE_CEL_SPIKE.md`
**Touches:** [.plans/UMPIRE_CEL_SPIKE.md]

### Approach
- The owner allows no worktrees and reserves commits, so "a branch that is not merged" is an isolated prototype module under `.flow/tmp/fn116-spike/` that depends on the server module through a `replace` directive. Nothing in the main tree changes except the report.
- Translate every expression of `model/ir/nexus-caller.json` and the standalone activity IR to CEL syntax trees in Go, evaluate with `cel-go`, and compare with the reader's own evaluator on every state and class, using the reader's public interface and the fn-115 goldens as the baseline.
- Evaluate the same translated step functions with `cel-java` and compare with `cel-go`.
- Take the measurements listed under API Contracts; state any not taken with the reason.

## Acceptance
- [ ] Every expression of both Models translates to CEL or is listed with its Scala position and blocking construct; `cel-go` results equal the current evaluator's on every state and class, with each difference listed.
- [ ] `cel-java` results equal `cel-go`'s, or each disagreement is listed.
- [ ] `.plans/UMPIRE_CEL_SPIKE.md` gives the measurements, the adoption cost beyond the prototype and a recommendation with reasons; no production path, schema, lifter or checked-in IR changed.


## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: