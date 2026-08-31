---
satisfies: [R4, R5, R6, R7]
---
# fn-21-nexus-duplicate-observation-control.6 Prove the paired live lifecycle and document the boundary

## Description
Complete R4-R7 with one paired live normal/faulted proof, immutable-set publication/status assertions, copy-paste usage, and honest roadmap reconciliation. This task adds no command or Make target.

**Size:** M
**Files:** `.flow/specs/fn-21-nexus-duplicate-observation-control.md`, `tools/umpire/runevaluation/live_negative_control_test.go`, `tools/umpire/runevaluation/mutation_test.go`, `model/Temporal/Tool/RunEvaluation.lean`, `tools/umpire/runevaluation/README.md`, `model/README.md`, `model/ARCHITECTURE.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [.flow/specs/fn-21-nexus-duplicate-observation-control.md, tools/umpire/runevaluation/live_negative_control_test.go, tools/umpire/runevaluation/mutation_test.go, model/Temporal/Tool/RunEvaluation.lean, tools/umpire/runevaluation/README.md, model/README.md, model/ARCHITECTURE.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Run the complete independent mutation/status suite before two bounded live controls: existing normal and exact faulted input, each through fn-19 execution, fn-20 checking, fn-18 publication, reopen, and strict set validation.
- Repair the Task5-owned duplicate-delivery adapter qualification exposed by the live proof: preserve a focused extra ordinary participant-fact mutation case, select the synthetic candidate only by the exact checked marker/fault identity, retain ordinary operational participant facts as nonsemantic raw provenance/ordering support, and scope missing-parent enforcement to the selected synthetic and history facts without weakening exact-two semantic contribution.
- Assert normal execution/Run Evaluation statuses 0/0 and satisfied; assert faulted statuses 0/2 and uniqueness-only violated; verify distinct normal/faulted input/run/result/set identities, byte-identical baseline artifacts, complete cleanup/source closure, and no cross-binding.
- Prove fn-18 republishing the exact already-constructed set and fn-20 rechecking the same immutable four-member input are idempotent. Separate live executions use fresh run IDs and may differ in timestamps/member bytes/manifests/destinations while preserving declared stable semantic and accepted-outcome identities.
- Document both existing direct/root command sequences, the requested/completed history chain, callback-one plus synthetic-one Evidence Link, three independent status dimensions, exact mutation outcomes, and why the result is a negative control rather than a Temporal defect claim.
- Update C4/C6/C7/C9, Milestone B, and pilot status only to implemented evidence; retain C10 replay/minimization/promotion and all non-local/formal/Observation Evaluation gaps. Run focused/aggregate tests without a Make or CI change.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-20-local-execution-semantic-conformance.7.md:13-33` — live proof/documentation pattern
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md:86-108` — execution publication/status/no-rerun contract
- `.flow/specs/fn-20-local-execution-semantic-conformance.md:46-59` — recheck/publication/status contract
- `model/README.md:70-98` — current author/runtime handoff
- `.plans/UMPIRE4_COMPONENTS.md:305-443` — component-status boundaries

### Acceptance
- [ ] The paired live proof produces the exact normal satisfied and faulted uniqueness-violated status/identity/publication matrix.
- [ ] Every fail-closed mutation runs first and matches the parent table; no partial/crossed set passes admission.
- [ ] Exact-set republish/recheck is idempotent, while separate executions assert only stable Behavior Fingerprints and never same transport bytes/destination.
- [ ] Documentation is copyable, labels the injection synthetic, and separates the two-event real lifecycle, real callback, synthetic contribution, request, receipt, operation, Observation Evaluation, and verdict.
- [ ] Roadmap/focused/aggregate gates pass with no new command, Make target, CI, or prohibited dependency.
## Acceptance
- [ ] R4-R6 paired live, causal evidence, status, identity, and publication proof passes.
- [ ] R7 usage/docs/roadmap and aggregate boundary verification are complete.

## Done summary
Implemented the paired bounded Nexus normal/fault live proof through execution, Run Evaluation, immutable publication, reopen, and exact independent status/identity controls. Repaired the Task5 adapter so discriminator-bearing malformed or duplicate synthetic candidates cannot disappear while ordinary lifecycle facts remain raw-only, and documented an installed-sibling direct status-2 command without treating GNU Make's generic status as the oracle.

The review fix adds an exact valid-plus-drifted-candidate conflict regression and pins both published Result Behavior Fingerprints to the stable Go controller checker identity. Focused and aggregate tagged Go suites, the paired live proof, focused and aggregate Lean gates, model lint, 243-job regression, direct status-2 execution, and focused Go lint pass; repository-wide `make lint-code` remains inherited red with 1,374 unrelated findings and no introduced task finding.

stage: impl-review - ran [Codex backend], final SHIP
## Evidence
- Commits: 6f86870bf372bbf09bbd51df80636f6fcd17fe88, 48c5685cbbcdaede4517003492a40e297ca26c59, 36d5eea6701134cd8509e736073763c4a6b06e8a, 65f89a77d1d06eec9fd43c4d246ad63d329ee69b
- Tests: RED: TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/runevaluation -run '^TestRealCheckerDuplicateDeliveryMutationMatrix$/second_synthetic_marker_drift$' (expected conflict, actual accepted), RED: TMPDIR=/private/tmp sh -c 'make --no-print-directory umpire-check-local-run-evaluation SET=/private/tmp/umpire-missing-run-set OUTPUT_ROOT=/private/tmp; test "$?" -eq 2' (generic Make failure incorrectly returned wrapper status 0), TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/runevaluation -run '^(TestRealCheckerDuplicateDeliveryMutationMatrix|TestRealCheckerDuplicateDeliveryIgnoresOrdinaryOperationalFacts)$', cd model && TMPDIR=/private/tmp mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosureFaultTests Temporal.Tool.RunEvaluationTests Temporal.Tool.RunEvaluationMutationTests, TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/runevaluation -run '^TestBoundedLiveNexusNegativeControl$', TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/temporal/nexus/..., TMPDIR=/private/tmp go test -count=1 -tags test_dep ./tools/umpire/runevaluation/..., verified installed sibling pair: direct umpire-local-run-evaluation against caller-closure-duplicate-delivery-run-set returned exact status 2 with succeeded/accepted/violated summary and /private/tmp destination, TMPDIR=/private/tmp make lint-model, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-tmp PATH=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/bin:$PATH make umpire-check-regression, INHERITED_RED: TMPDIR=/private/tmp make lint-code (1,374 repository findings; six unrelated auto-edits exactly inverted; no introduced task finding), TMPDIR=/private/tmp .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,test_dep' --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/umpire/runevaluation/... (0 issues), flowctl validate --spec fn-21-nexus-duplicate-observation-control, Codex implementation review receipt /tmp/impl-review-receipt-fn-21-nexus-duplicate-observation-control.6.json: SHIP, 0 introduced findings
- PRs: