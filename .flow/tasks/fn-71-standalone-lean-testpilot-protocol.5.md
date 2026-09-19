---
satisfies: [R5, R7]
---
# fn-71-standalone-lean-testpilot-protocol.5 Migrate Temporal Nexus3 rendering and managed fixtures

## Description
Migrate current Temporal, Nexus3, conformance, and renderer consumers to generated Testpilot values and the library-backed ProtoJSON path. Re-anchor the landed Nexus3 work, preserving its authored syntax and checked lowering while accepting only a reviewed semantic-equivalence fixture transition.

**Size:** M
**Files:** `model/Temporal/Testpilot.lean`, `model/Temporal/Testpilot/**`, `model/Temporal/Feature/Nexus3/Syntax.lean`, `model/Temporal/Feature/Nexus3/Testpilot.lean`, `model/Temporal/Feature/Nexus3/Tests.lean`, `model/Temporal/Tool/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**`, managed Testpilot fixtures and tests
**Touches:** [`model/Temporal/Testpilot*`, `model/Temporal/Feature/Nexus3/{Syntax,Testpilot,Tests}.lean`, `model/Temporal/Tool/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/**`, `common/testing/testpilot/testdata/case-runtime-conformance/*.json`, `tests/testcore/testpilot/testdata/*.json`, `tests/testcore/testpilot/artifact_test.go`]

### Approach
- Update generic Temporal helpers, GetSystemInfo/conformance Producers, and Nexus3 checked lowering to call `Testpilot.Authoring`.
- Keep Nexus3-only macros in `Temporal.Feature.Nexus3.Syntax` and preserve checked Target/Property/Behavior/Query/witness validation in its adapter.
- Move rendering to `Testpilot.ProtoJSON.canonical` and remove the independent Temporal serializer body.
- Regenerate both fixture trees transactionally. Require exact bytes where library policy agrees; otherwise review a single stable object-order/default-elision transition through strict decoded-message equivalence.
- Preserve logical Case IDs, list order, provenance bytes, supported preparation/execution, and stable Run/Verdict projections.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Testpilot/CaseSupport.lean`
- `model/Temporal/Testpilot/Conformance.lean`
- `model/Temporal/Feature/Nexus3/Syntax.lean`
- `model/Temporal/Feature/Nexus3/Testpilot.lean`
- `model/Temporal/Tool/Testpilot.lean`
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`
- `tests/testcore/testpilot/artifact_test.go`
## Acceptance
- [ ] Temporal and Nexus3 Producers construct generated Testpilot values through the neutral facade.
- [ ] Nexus3 feature macros remain isolated in `Syntax.lean`, and its checked lowering/rejection behavior is unchanged.
- [ ] The renderer delegates to the sole Testpilot ProtoJSON wrapper; no Temporal field serializer remains.
- [ ] Managed fixture generation is complete-tree, transactional, deterministic, and either byte-identical or backed by reviewed strict semantic-equivalence evidence.
- [ ] Existing Go preparation/execution and stable Run/Verdict projections pass unchanged.
## Done summary
Temporal, Nexus3, and conformance producers now construct generated Testpilot protocol values through `Testpilot.Authoring`, while Nexus3 retains its authored macros, checked lowering, rejection guards, IDs, order, and opaque provenance. The sole renderer is `Testpilot.ProtoJSON.canonical`; both managed fixture trees are complete-tree, transactional, and deterministic, and all eight migrated Cases passed strict decoded-message equivalence while expected Run/Verdict projections remained byte-identical.

The pre-edit conformance baseline was red only for the planned ProtoJSON formatting/default-elision transition. `make lint-code GOLANGCI_LINT_FIX=false` was not rerun because its accepted inherited baseline contains 1361 unrelated findings; the focused changed Go packages and full scoped Testpilot suites passed.

stage: impl-review - ran (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: red (make umpire-check-case-runtime-conformance failed pre-edit only on planned ProtoJSON formatting/default-elision transition), (cd model && mise exec -- lake build Temporal.Feature.Nexus3.Tests temporal-testpilot), TMPDIR=/private/tmp mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, make umpire-gen-case-runtime-conformance, mise exec -- go run -tags test_dep /tmp/fn71-task5-semantic-equivalence.go <old conformance fixtures> <new conformance fixtures>, mise exec -- go run -tags test_dep /tmp/fn71-task5-semantic-equivalence.go <old functional fixtures> <new functional fixtures>, make umpire-check-case-runtime-conformance, (cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot), (cd model && mise exec -- lake exe modelLintTests), TMPDIR=/private/tmp CGO_ENABLED=0 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., make lint-model, BASELINE_REUSED:make lint-code GOLANGCI_LINT_FIX=false - inherited 1361 unrelated findings; changed Go packages passed focused and scoped suites, impl-review: SHIP (codex:gpt-5.6-sol:medium; /tmp/fn71-task5-impl-review.json)
- PRs:
