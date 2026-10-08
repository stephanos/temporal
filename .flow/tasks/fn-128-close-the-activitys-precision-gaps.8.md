---
satisfies: [R8]
---
# fn-128-close-the-activitys-precision-gaps.8 Bind a caller-owned bounded Profile for pauseResume evidence

## Description
Implement R8 for exactly `pauseResume` and `heartbeatTimeoutRetriesThenCompletes` through existing caller-owned Profile binding. This correction owns two separate final Case identity/shape authorizations and scoped admission/recording/replay proof; the existing close task owns final generation, live gates and R6 accounting. The task ID, title, priority, dependencies and conductor's post-fn-129.4 parallel correction order remain unchanged.

**Size:** M
**Files:** `tools/umpire/conformance/activity_test.go`, `tools/umpire/conformance/activity_budget_test.go`, `tests/testpilot_generated_test.go`, `tests/testpilot_activity_control_test.go`, `tests/testpilot_case_profile_test.go`, existing `common/testing/testpilot/internal/verification/correlated_test.go` only for uncovered pricing boundaries.
**Touches:** [tools/umpire/conformance/activity_test.go, tools/umpire/conformance/activity_budget_test.go, tests/testpilot_generated_test.go, tests/testpilot_activity_control_test.go, tests/testpilot_case_profile_test.go, common/testing/testpilot/internal/verification/correlated_test.go]

### Approach

- The conductor dispatches after fn-129.1-4 settle, alongside the fatal correction only with disjoint ownership. Budget does not edit played_test.go or Scala source. Serialize scratch lifts/heavy suites and join fatal's source before sealing the final authorized shape. The final current-source/generated comparison remains in .6.
- Reuse existing ProfileSpec and Profile derivation. Assemble an explicit Case-scoped policy at the actual generated caller binding seam; pass the immutable Profile and matching namespace/environment/resource binding to preflight and `newTestpilotLiveCase`, and retain it for replay. Keep ordinary callers and unauthorized Cases on their existing policy. Do not add an evaluator Query-ID switch, public configuration surface or generic budget framework.
- Reuse the existing scratch current-source IR override and explicit lower seam for focused tests. After fn-129.4 and fatal-source integration, separately inventory each named final Case's Program, guarded accepted emissions and bytes, duplicates, captured cardinalities, buffered release fanout and semantic outputs. Pin complete Case/Program/Contract/Model fingerprints and exact structure/envelope independently for each; reject unexplained drift or transfer between the Cases. The pre-138 pauseResume Case and fn-129.1 scratch heartbeat Profile remain separate evidence inputs, not final allowlists by assumption.
- Derive complete accepted-set cubic projection reservation plus ordinary-rule work and all-output obligation charge for every admitted stage, including all outputs released by buffered fanout; prove cumulative total work too. Preserve duplicate charges and atomic combined-budget failure. Retain all defaults, pricing, other resource caps and the hard 100,000,000 per-event and 1,000,000,000,000 total ceilings. Keep the default and independent canary's 4,000,000 literal unchanged. Recompute within those ceilings when approved source changes affect final rows/claims/facts; if a sound authorization cannot fit, report the concrete bound rather than weakening pricing or caps.
- The captured pauseResume T14/Q23/R1/F2/E0 five-record/512-byte shape gives 12,723,923 as a conditional offline upper bound. For scratch heartbeat retry, sequences 4/6/8/13/15 and byte sizes 147/214/262/289/265 sum to 1177; T17/output-factor34 gives 8,651,232 for fifth-stage projection only. Neither shape/number authorizes the other Case or future drift. Record actual combined charged stages and minima separately. For each separate final envelope, test one-below actual combined stage charge, excess records (including a sixth where its proven envelope is five), duplicates, buffered multi-output release and overflow without changing pricing.
- Test mismatched preflight/execution binding, wrong Case/shape and failed admission before external I/O. Confirm played execution and replay preserve independent assessments, Property verdict/reason/support and exact Profile identity. Put caller/policy proofs in separate files and use existing pricing boundary tests only where they lack the required negative.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/conformance/activity_test.go:50` and scratch helper in `tools/umpire/conformance/played_test.go:274` (read only).
- `tests/testpilot_generated_test.go:219` and `tests/testpilot_activity_control_test.go:33`.
- `tests/testpilot_run_case_test.go:47` (read only unless a reviewed scope adjustment becomes necessary).
- `common/testing/testpilot/internal/verification/correlated.go:486` and reservation at `:578`.
- `common/testing/testpilot/internal/verification/evaluator.go:173`.
- `common/testing/testpilot/temporal/profile.go:95` and `tools/canary/casebinding/casebinding.go:95` (read only).

### Quick commands

```bash
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/conformance -run 'Budget|PauseResume|Heartbeat'
go test -tags test_dep -p 2 -timeout 30m ./common/testing/testpilot/internal/verification -run Correlated
```

Run existing generated-caller unit tests without starting the live cluster, and record the actual selector. Serialize heavy commands with the shared flock. Full gates, docs, live and final production-byte matching stay in .6.

## Acceptance
- [ ] Two separate closed Case/Program/Contract/Model identity/shape policies cover exactly pauseResume and heartbeatTimeoutRetriesThenCompletes after fn-129.4 and fatal-source integration. Each proves complete projection/ordinary/all-output-obligation stage and total-work bounds within retained hard ceilings; default/pricing/ordinary-caller/unauthorized-Case/canary policy remains unchanged.
- [ ] Generated preflight, executing binding and replay use the same immutable Profile and matching namespace/environment/resource inputs. Exact identity/snapshot checks and wrong-Case/drift/mismatched-binding/no-I/O admission negatives pass.
- [ ] For each named Case, actual charge-ledger proof, one-below actual combined charge, excess record, duplicate, buffered multi-output and overflow tests pass with pricing unchanged. The conditional pauseResume 12,723,923 upper bound and scratch heartbeat 8,651,232 projection-only observation remain separate from final authorizations and measured combined minima.
- [ ] Both named Cases' scratch current-source played recording and protobuf replay preserve independent Property verdict/reason/support and exact Case/Profile/resource identities, including new heartbeat SATISFIED expectations; old retry explanationsDisagree exceptions do not transfer. Final generated-byte matching, live execution and shared review remain pending .6.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
