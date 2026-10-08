---
satisfies: [R7]
---
# fn-128-close-the-activitys-precision-gaps.7 Observe accepted fatal settlement through typed public evidence

## Description
Implement R7 against the settled activity source. This correction supplies accepted fatal-settlement evidence; the existing close task owns final generation, live gates and R6 accounting.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/system/System.scala`, `model/temporal/features/activity/standalone/system/Realization.scala`, `model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala`, `tools/umpire/conformance/played_test.go`, `tools/umpire/conformance/activity_fatal_evidence_test.go`.
**Touches:** [model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/activity/standalone/system/Realization.scala, model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala, tools/umpire/conformance/played_test.go, tools/umpire/conformance/activity_fatal_evidence_test.go]

### Approach

- The conductor dispatches only after fn-138.3's strict original/adopted R3 comparison is sealed and fn-129.1-4 sources are integrated. Re-anchor the named rows and current fact catalog after those source changes.
- Extend the System Fact catalog and the accepted fatal effects only. Enumerate all applicable accepted fatal rows and derived owners after by-ID/reset changes, preserving reset/cancel behavior. Reuse the existing statusFailed fact and Property. Hide only the new classification in Product refinement; preserve exact legal row structure and old ordered fact subsequences. Record baseline and mutated rows separately from the previously sealed fn-138 comparison.
- Reuse the typed Describe read and vocabulary in Realization.scala. Request last failure, test FAILED status plus present application oneof and true retained nonRetryable Boolean, and bound the classification with singleton Taking. Preserve generic statusFailed vocabulary for exhausted failures. Keep the producer's existing confirming precedence and one primary kind per decisive source; an offered SDK fatal response supplies no proof of server acceptance.
- Extend played_test's explicit scripted server-failure input and IncludeLastFailure validation without branching on Query names or expected verdicts. Honor IncludeLastFailure when supplying independently configured server failure and assert the fatal poll requests it. Update the fatal poll/kind fixture entry while keeping the final raw read's existing request and payload. Put the focused positive and negative proof in activity_fatal_evidence_test.go. Fatal owns played_test.go; the parallel budget worker does not edit it.
- Produce a serialized scratch current-source lift/lower and reuse `UMPIRE_ACTIVITY_IR_DIR` and the existing explicit-Model lower seam for focused recording/replay. Do not regenerate production artifacts here. Preserve full unfiltered IR and ordered Check receipts separately; the played helper's FIND selection is not all-Query evidence.
- Test actual typed recording predicates and the SDK-offer/server-other distinction. Preserve failure1/poll2, timeout1/poll2, final raw attempt reads, Case expectations and all other assessment bindings. Compare online/offline verdict, reason and support identities. Retain exact source and Case/Profile inputs for task .6's final live proof.

### Investigation targets

**Required** (read before coding):
- `model/temporal/features/activity/standalone/system/System.scala:52` and accepted fatal/refinement/Property declarations.
- `model/temporal/features/activity/standalone/system/Realization.scala:38` and fatal await.
- `model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala`.
- `tools/umpire/conformance/played_test.go:109` and scratch loading at `:274`.
- `tools/umpire/conformance/activity_test.go:40`.

### Quick commands

```bash
mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only '*ActivityRetryRegression*' --require-tests
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/conformance -run 'Fatal|NonRetryable'
```

Re-anchor the source paths if the scheduled mechanical moves have changed them; record the exact invoked command. Heavy scratch lifts and Go commands use the shared flock. Full suites, production generation, docs, live and final review stay in .6.

## Acceptance
- [ ] Focused source row comparison proves that accepted fatal rows alone gain classification; states/actions/guards/outcomes/next states and old fact subsequences stay unchanged. Product refinement and existing failure/timeout occurrence vectors pass.
- [ ] Typed source recording requires FAILED and the requested present application failure with nonRetryable true. Retryable exhaustion, missing failure/arm/flag, false flag, wrong operation, malformed fields and offered-fatal/server-other races cannot establish classification.
- [ ] Scratch current-source lowering, played recording and offline replay satisfy the unchanged fatal Property with matching reasons/support and final raw attempt-count observation. Contract-only or conformance-only success is insufficient.
- [ ] Source/Case/Profile/receipt deltas and exact inputs are retained for R6; focused tests and scoped lint/format pass. Shared production/live/full review remains explicitly pending .6.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
