---
satisfies: [R3, R4, R8, R9]
---
# fn-20-local-execution-semantic-conformance.8 Prove causal ordering against skewed wall clocks

## Description
Add one independent, test-only skew fixture at the Temporal Lean checker boundary.

**Touches:** `model/Temporal/Tool/RunEvaluationTests.lean`

Use strict TDD: first add the exact executable assertion and capture its RED before adding the private fixture/oracle. The fixture assigns deliberately contradictory wall-clock timestamps to the existing caller-closure history while keeping the admitted facts source-local ordinals and causal parents unchanged. Compare the authoritative causal/source-local projection with an explicitly test-only hypothetical timestamp-authoritative projection. Do not add timestamps to production RawEvidence, alter production ordering, weaken closure, or invent wall-clock authority.

## Acceptance
- [ ] The fixture contains deliberately skewed wall-clock timestamps whose ascending order contradicts both the admitted history source ordinals and causal-parent chain.
- [ ] The authoritative unchanged request evaluates to accepted/applied/satisfied and asserts the exact Evidence Link ordering support plus Result semantic status/checksum.
- [ ] An independent test-only timestamp-authoritative projection over the same fixture changes Observation Evaluation or Property Result, proving the scenario is discriminating rather than decorative.
- [ ] Production request/schema/evaluator code is unchanged unless the RED demonstrates a real defect; no timestamp becomes semantic authority.
- [ ] The focused Lean target and the full fn20/model gates pass.

## Done summary
Added a private skewed-wall-clock caller-closure fixture that keeps admitted fact ordinals and causal parents authoritative, asserts exact Evidence Link ordering and the satisfied Result checksum, and proves a test-only timestamp-authoritative projection changes the Result to incomplete. No production request, schema, adapter, or evaluator code changed; strict RED and all applicable fn20/model gates are recorded, with the stale static smoke path and unrelated Go lint finding explicitly classified as inherited.

stage: impl-review - ran (Codex SHIP; session 01a05a58-d6ba-79b0-bec7-6f0df20d0024; receipt /tmp/impl-review-receipt-fn-20-local-execution-semantic-conformance.8.json)

stage: plan-sync - skipped(config: planSync.enabled != true)

## Evidence
- Commits: d8f293a19da171653fa0fad634e79d286946571d
- Tests: baseline: green - cd model && mise exec -- lake build Umpire.Observation.Tests.Check, baseline: green - cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker, baseline: green - go test -count=1 ./tools/umpire/runevaluation/..., baseline: green - go test -count=1 ./tools/umpire/cmd/umpire-local-run-evaluation/..., baseline: red (inherited) - make umpire-check-local-run-evaluation SET=tools/umpire/temporal/nexus/testdata/caller-closure-run-set OUTPUT_ROOT=/tmp/umpire-local-results (SET must be a directory; the tracked static set is absent and the current live set is materialized dynamically), baseline: green - make umpire-check-regression, RED_EXPECTED: cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests (exit 1: Expected type must not contain free variables skewedWallClockFixtureIsDiscriminating; /Users/stephan/Workspace/temporal/umpire/.flow/tmp/fn20-8-red.log), cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, cd model && mise exec -- lake build Umpire.Observation.Tests.Check, cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests temporal-run-evaluation-checker, go test -count=1 ./tools/umpire/runevaluation/..., go test -count=1 ./tools/umpire/cmd/umpire-local-run-evaluation/..., INHERITED_RED: make umpire-check-local-run-evaluation SET=tools/umpire/temporal/nexus/testdata/caller-closure-run-set OUTPUT_ROOT=/tmp/umpire-local-results (exit 2: SET must be a directory; no canonical static replacement; current dynamic live-set evaluation passed in the full runevaluation package suite), make umpire-check-regression, make lint-model, go test -tags test_dep -count=1 ./tools/umpire/runevaluation/..., go test -tags test_dep -count=1 ./tools/umpire/cmd/umpire-local-run-evaluation/..., INHERITED_RED: GOLANGCI_LINT_BASE_REV=e7519e9020f74f5821244722e6d3ade3b46fdbea GOLANGCI_LINT_FIX=false make lint-code (golangci: 0 task-diff issues; unchanged tools/umpire/runtime/errors.go:60:9 et:unw+), git diff --check, impl-review Codex SHIP session 01a05a58-d6ba-79b0-bec7-6f0df20d0024 receipt /tmp/impl-review-receipt-fn-20-local-execution-semantic-conformance.8.json
- PRs:
