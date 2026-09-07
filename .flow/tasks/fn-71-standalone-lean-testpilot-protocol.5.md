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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

