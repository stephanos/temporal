---
satisfies: [R4, R5, R8]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.5 Add the faulty variant and the zero-Go check for the example

## Description
**Recut 2026-10-18 (one-language series):** Owner-deferred. At revival: lowering goes to fn-203, export to fn-206/207 (fn-205 deferred), Go model valuations to the Scala checker.

Show failure and enforce the no-manual-Go claim.

**Size:** M
**Files:** `model/examples/activityworkflow/` faulty variant (mirror the control machine `TrustingCaller` (formerly `object Control`) in `model/temporal/features/nexus/workflow/system/TrustingCaller.scala`: a deliberately wrong machine, a `find` Query expecting `contract = violated`, its own IR file); a source-scan test beside `tools/umpire/model/isolation_test.go` (`modelFiles` :216, `TestModelNamesNoRetiredFrontEnd` :245) run by name from the gate (`Gate.scala:296-310`).
**Touches:** [model/examples/**, model/ir/**, model/cases/**, tools/umpire/model/isolation_test.go, tools/umpire/model/*_test.go, model/check/**]

### Approach
- Faulty variant: one Query yields a model-level counterexample, declared on the Scala side (do not pin it in Go by name as `activity_system_test.go:117-179` does - that would break R4), and a violated Verdict at run level, run by the generic live runner.
- Zero-Go check: match whole identifiers or quoted strings, not substrings, and give the example distinctive names (a common prefix) so generic words like `timeout` cannot match. Derive the example's names (package, workflow type, activity type, Query names) from the example's IR, not a hard-coded list in Go, and fail on any Go source that names one; exempt generated files and the walkthrough by path pattern listed in the check. Include a table test of the matcher.
- A faulty variant that passes blocks the close (R5 errors).

### Investigation targets
**Required:**
- `TrustingCaller` (formerly `object Control`) in `model/temporal/features/nexus/workflow/system/TrustingCaller.scala`, `tests/testpilot_nexus_control_case_test.go:33`
- `tools/umpire/model/isolation_test.go:200-280`
- `model/check/Gate.scala:290-315`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/model/ -run 'Example|Isolation'
make umpire-check-model && make umpire-check-live-tests
```
## Acceptance
- [ ] The faulty variant yields a counterexample at model level and a violated Verdict at run level.
- [ ] A gate-run check fails when any non-exempt Go source names the example, its workflow or activity type, or a Query; exemptions are listed by path pattern; a mutation test proves it fires.
- [ ] Both run in the model gate and the live test job.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: