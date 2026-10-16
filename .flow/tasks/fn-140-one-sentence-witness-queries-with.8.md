---
satisfies: [R3]
---
# fn-140-one-sentence-witness-queries-with.8 Validate live Case generation and complete model-only Query standings

## Description
**Size:** M
**Files:** `tools/umpire/lower/generated.go`, its regression tests and existing manifest assertions.
**Touches:** [tools/umpire/lower/generated.go, tools/umpire/lower/generated_test.go]

Owner-deferred Go half split from original fn-140.3. Preserve its entire R3 live Case-selection proof; do not admit until the owner reopens non-Scala work. Consume the actual .3 source expectation records and witness/core source fixtures.

### Approach
- At `tools/umpire/lower/generated.go:88-119`, a find with no expected_run (no `.live`) remains model-only and generates no Case, even on an executable machine. Account for it truthfully in the complete manifest instead of invoking lowering and erroring for a missing assessment.
- Preserve Query/check admission, malformed declared expectations and existing unsupported/refusal diagnostics for live Queries. Capability-generated expectations continue to select their Cases.
- Test two otherwise equivalent lowerable finds, one live and one without live, on the same realized machine: only the live find generates a Case. Cover witness and triple sources, capability selection, absent versus incomplete declared expected Runs and complete Query inventory/manifest standings.
- Pin legacy live Case bytes and membership, preserving expectation/assessment/schema reason IDs. Source-only because text must not appear in generated output or affect membership. Integrate with .7's exact-composed selector proof rather than applying a witness-specific lowering bypass.

### Investigation targets
**Required:** `tools/umpire/lower/generated.go:88-119`; `tools/umpire/lower/generated_test.go`; fn-140.3's expectation/source-text parity fixtures; fn-140.2/.7's witness/core selector fixtures; the complete Case manifest.
**Optional:** existing Lower admission and capability-generation tests.

### Quick commands (only after owner lifts deferral)
```sh
mise exec -- go test -tags test_dep ./tools/umpire/lower
```

Full production regeneration/model/Case gates/review/live remain at the shared fn-123.8 boundary.

## Acceptance
- [ ] Paired otherwise-lowerable live/no-live witness and triple regressions generate exactly the live Cases and account for every Query/standing in the complete manifest (R3).
- [ ] Capability Cases remain selected, absent expectation is distinguished from malformed/incomplete declared expectation, and existing admission/refusal errors are retained.
- [ ] Source-only text changes affect neither Case membership nor emitted expectation/assessment bytes; no schema or judge reason field is added.
- [ ] Focused Go tests pass only after the owner lifts deferral; shared gates/review/live closure is explicitly retained.


## Done summary
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
