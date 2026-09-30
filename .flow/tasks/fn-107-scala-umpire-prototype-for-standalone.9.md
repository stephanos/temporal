---
satisfies: [R6, R7]
---
# fn-107-scala-umpire-prototype-for-standalone.9 Run shared activity and Nexus Cases through existing consumers

Touches: [tests/testcore/testpilot/**, tests/testpilot_scala*_test.go, tools/canary/testharness/**, tools/canary/casebinding/**, tools/canary/preflight/**]

## Description
Consume Scala-produced Cases through existing functional and isolated canary harnesses after the activity SDK adapter prerequisite lands.

**Size:** M
**Files:** shared functional fixture/consumer test; canary testharness.go, casebinding.go/preflight.go and focused seam tests as needed.

### Approach
- Consume the activity activation support from the existing Testpilot SDK adapter prerequisite. Do not add another SDK runner.
- Bind one public activity completion Case/script/property to both consumers. Keep profile authority and independent per-run resources at existing seams.
- Add a test-harness-only binding path carrying explicit canonical Case bytes and a checked Profile into existing casebinding/preflight. Production Bind/Check remain pinned and expose no runtime Case override. The harness follows the same admission, namespace, identity, and capability checks, using isolated resources. Prove functional/canary Case identity and script bytes match and that a harness binding cannot affect production's pinned binding.
- Translate selected activity completion/retry/pause-unpause and Nexus sync/async behavioral assertions into shared fixtures. Include the applicable existing standalone/workflow parity comparison.
- Preserve existing workflow/Nexus worker behavior with focused regression tests. Exclude scheduled canary orchestration and live CallerClosePolicy qualification.

### Investigation targets
**Required:** common/testing/testpilot/temporal/worker/sdk.go:23; common/testing/testpilot/temporal/worker/interpreter.go:60; tools/canary/casebinding/casebinding.go; tools/canary/preflight/preflight.go; tools/canary/testharness/testharness.go.
**Optional:** tests/activity_parity_test.go:544; tests/nexus_workflow_test.go:512.

### Quick commands
`mise exec -- go test -tags test_dep ./common/testing/testpilot/temporal/worker/... ./tests/testcore/testpilot/... ./tools/canary/testharness/... ./tools/canary/casebinding/... ./tools/canary/preflight/...`; run selected functional cases in the configured integration environment.

## Acceptance
- [ ] Existing Testpilot executes a real Go SDK activity from the Scala-declared script.
- [ ] Functional and isolated canary consumers run byte-identical completion Case identity/script through the existing consumer's harness-only Case/Profile seam, with independent resources; production's pinned Case/Profile cannot be overridden.
- [ ] Selected activity/Nexus translations preserve their assertions and existing SDK consumers continue passing.
- [ ] Missing capabilities reject before I/O; repeat/concurrent runs do not alias learned identities.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
