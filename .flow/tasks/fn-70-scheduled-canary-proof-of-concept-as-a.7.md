---
satisfies: [R6, R7, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.7 Orchestrate one Activity with deterministic replay and uncertain loss outcomes

## Description
Orchestrate one Activity with deterministic replay and uncertain loss outcomes.

**Size:** M
**Files:** tools/canary/workflow.go; tools/canary/workflow_test.go; tools/canary/workflow_replay_test.go; tools/canary/testdata/**; tests/canary_workflow_loss_test.go
**Touches:** [tools/canary/workflow.go, tools/canary/workflow_test.go, tools/canary/workflow_replay_test.go, tools/canary/testdata/**, tests/canary_workflow_loss_test.go]

### Approach
- Check Workflow accepts only serializable pinned identities and returns bounded result. ScheduleToClose75s, StartToClose60s, HeartbeatTimeout20s, WaitForCancellation=true, Activity RetryPolicy.MaximumAttempts=1; declared Workflow action execution/run timeout90s and execution MaximumAttempts1 are shared constants for task8.
- Classify actual violated/inconclusive results as data; operational missing result, timeout, exhausted retention or publication failure becomes nonretryable orchestration failure with bounded uncertain metadata, preserving any existing Run reference/violation.
- Use SDK WorkflowTestSuite and actual WorkflowReplayer partial/terminal history to prove no repeated execution on replay. Process loss after target dispatch and after publication must not create a synthetic Run or change uncertainty after late result.
- Add TestCanaryWorkflowLossAndTimeout with live worker loss/cancellation and server-enforced target expiry; task8 adds automatic Schedule pause through the task3 policy. Do not rely on Workflow cancellation handlers after direct termination.

### Investigation targets
**Required:**
- common/testing/testpilot/temporal/worker/sdk_test.go:76 — actual replay APIs.
- go.mod:84 — pinned SDK1.44.0.
- tests/testpilot_async_nexus_case_test.go:112 — authoritative incomplete Run example.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./tools/canary`
Add TestCanaryWorkflowReplayNoRedispatch and execute actual history replay, not mocks alone.
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^TestCanaryWorkflowLossAndTimeout$' -timeout 10m > /tmp/fn70-task7-live.jsonl`
Then verify both terminal exit 0 and JSON records: each exact named top-level test has Action=run and Action=pass, no matching test/subtest Action=skip or fail, and package pass. Parse with a bounded script; a missing record is failure. This command names a required new test, not a currently existing one.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] Workflow contains deterministic SDK orchestration only; partial/full replay does not invoke external execution again.
- [ ] One Activity attempt and90s Workflow bound are explicit; violation/inconclusive remain retained measurements.
- [ ] Lost/timed-out/report-failed measurements fail orchestration without invented closure or retry and keep independent statuses.
- [ ] Live loss/target expiry and focused cancellation/late-result tests pass with bounded summaries.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
