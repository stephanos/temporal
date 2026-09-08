---
satisfies: [R2, R3, R4, R7, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.6 Execute bounded canary Activities through the shared Driver with early live proof

## Description
Execute bounded canary Activities through the shared Driver with early live proof.

**Size:** M
**Files:** tools/canary/activity.go; tools/canary/activity_test.go; tools/canary/worker.go; tools/canary/worker_test.go; tests/canary_activity_test.go
**Touches:** [tools/canary/activity.go, tools/canary/activity_test.go, tools/canary/worker.go, tools/canary/worker_test.go, tests/canary_activity_test.go]

### Approach
- Worker owns caller SDK clients, frozen Profiles/PreparedCases and shared temporal.Driver registrations; Activity obtains admitted pinned entry, reserves sink capacity, claims the measurement identity durably, invokes PreparedCase.Run once for a new claim, then snapshots/publishes result. Duplicate/unresolved claims reject without another Run; a retained identical result may be read without execution.
- Heartbeat every5s with20s HeartbeatTimeout; propagate Activity context to Run while alive. Budget30s ordinary plus three5s phases=45s, reporting<=10s, Activity StartToClose60s. Worker concurrency1 and bounded drain60s; Driver/client closure bounded and observed. Do not assume outer cancellation prevents fresh Testpilot cleanup phases.
- Preserve actual authoritative statuses; violated/inconclusive are data. No Run returned after process loss means uncertain, never synthesize closure. Publication failure returns bounded diagnostic for Workflow failure/pause without retry. Introduce simple injected execution/sink seams for counting and lifecycle tests, not another interpreter.
- Add TestCanaryActivitySharedDriverBindings: use canary Activity entrypoint in a real SDK Activity environment/live cluster, execute two isolated measurements and alternate namespace/queue bindings with exact artifact bytes, compare existing functional consumer semantics/correlated support. Show server60s target lifetime and cancellation cleanup; missing resource/invalid binding rejects at appropriate stage. This early proof precedes scheduling.

### Investigation targets
**Required:**
- common/testing/testpilot/temporal/driver.go:27,44,123 — client/Driver composition and Close.
- tests/testpilot_async_nexus_case_test.go:47,132 — real two-environment consumer.
- service/worker/migration/activities.go:219 — heartbeat precedent.
- service/worker/parentclosepolicy/processor.go:86 — bounded SDK concurrency.
- common/testing/testpilot/internal/execution/runtime.go:54 — cleanup phases.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./tools/canary ./common/testing/testpilot/...`
`mise exec -- go test -race -count=1 -tags test_dep ./tools/canary`
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^TestCanaryActivitySharedDriverBindings$' -timeout 10m > /tmp/fn70-task6-live.jsonl`
Then verify both terminal exit 0 and JSON records: each exact named top-level test has Action=run and Action=pass, no matching test/subtest Action=skip or fail, and package pass. Parse with a bounded script; a missing record is failure. This command names a required new test, not a currently existing one.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] Real early proof executes two distinct Runs plus alternate binding via public Activity/shared Driver and unchanged Case semantics.
- [ ] Heartbeat cancellation,45s runtime budget, bounded reporting/drain and finite server lifetime have concrete tests.
- [ ] A new local measurement claim reserves capacity and invokes Run once; duplicate/unresolved claims and violation/inconclusive/report failures cause no redispatch. This does not claim exactly-once external effects.
- [ ] Source/Run/resource identities stay isolated and caller-owned clients/Driver close within bounds.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
