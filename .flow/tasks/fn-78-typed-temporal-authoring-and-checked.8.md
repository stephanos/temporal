---
satisfies: [R6, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.8 Qualify authored Nexus cancellation through Testpilot

## Description
**Scope transferred to deferred fn-79 by user decision. This task no longer authorizes cancellation implementation or blocks generic delivery. Generic syntax/qualification formerly in task 8 is retained in task 10. The original description below is archival.**

Expose D5's readable bounded temporal syntax over the checked scoped clause and qualify a complete authored Nexus cancellation Case through the existing public Testpilot Prepare/Run facade. Keep Nexus syntax and lowering next to the feature adapter while reusable temporal constructors remain in `Umpire.Property`.

**Size:** L
**Files:** `model/Umpire/Property/{Authoring,Syntax,Tests/**}.lean`, `model/Temporal/Feature/Nexus3/{Authoring,Syntax,Nexus,Testpilot,Tests,Integration.md}`, `model/Temporal/Tool/Testpilot.lean`, managed Testpilot fixtures/generator tests, `tests/testpilot_async_nexus_case_test.go`, `model/{README,ARCHITECTURE}.md`, `model/Umpire/ARCHITECTURE.md`
**Touches:** [model/Umpire/Property/**, model/Temporal/Feature/Nexus3/**, model/Temporal/Tool/Testpilot.lean, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/**, tests/testpilot_async_nexus_case_test.go, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

### Approach
- Add typed constructors and hygienic notation that resolve trigger, response, same-operation key, operation-transition clock, natural bound, and endpoint explicitly into the same checked clause.
- Use elaboration for context-sensitive rejection and source-local diagnostics; do not add a general expression framework or make authors maintain serialization/proof plumbing.
- Extend the existing Nexus3 Producer's checked Query/witness gate to cancellation, compile one deterministic admitted Case, and bind task 9's per-operation cancellation capability through task 1's generic server seam.
- Drive local Temporal integration through public `Prepare`/`Run`; recognize correlated cancellation and either permitted terminal result without forcing the chosen model outcome.
- Update ownership/architecture docs and replace the Nexus3 integration draft's unsupported-cancellation statements.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Property/Authoring.lean` — reusable typed Property owner
- `model/Temporal/Feature/Nexus3/Syntax.lean` — feature-local macro pattern
- `model/Temporal/Feature/Nexus3/Testpilot.lean:246-300` — checked Query/witness and Case compilation
- `model/Temporal/Feature/Nexus3/Tests.lean` — authored equality/rejection tests
- `tests/testpilot_async_nexus_case_test.go` — existing real Driver proof

### Key context
- Lean 4 elaborators should issue context/type/key/clock errors at the author syntax; macros should handle syntax-only expansion.
- Parameterized field expressions remain fn-77-owned; consume delivered interfaces without duplicating them and keep this label-only qualification independently complete.
## Acceptance
- [ ] Readable bounded temporal notation exposes or unambiguously resolves trigger, response, correlation key, semantic clock, natural bound, and endpoint policy and elaborates to the same canonical checked clause as typed constructors.
- [ ] Compile-failure tests reject state/step/trace/evidence context misuse, raw evidence or command effects in Properties, wrong keys/clocks/scopes/references, ambiguous names, incompatible bounds, and unsupported formulas at the author expression.
- [ ] Ordinary authors supply semantic choices only; derived IDs, capability version, provenance, and generated monitor mechanics remain inspectable without handwritten duplicate monitor rules.
- [ ] A checked Nexus cancellation Query produces deterministic admitted Case 1.0 bytes and runs through the existing public Prepare/Run facade with the authorized generic capability.
- [ ] Local integration recognizes correlated cancellation confirmation followed by canceled or completed without forcing the selected model result; submission alone cannot satisfy the Property.
- [ ] Controlled violating, incomplete, wrong-correlation, stopped/lost execution, and cleanup-failure fixtures preserve the distinction among violation, inconclusive, and disposition while retaining earlier proof.
- [ ] Repeated/concurrent Runs and tenfold candidate/evidence/overlapping-obligation loads demonstrate isolation and bounded failure with recorded semantic/work limits and costs.
- [ ] Existing success/rejection/cross-language/identity/lifecycle regressions remain green; unchanged fixtures retain exact bytes and IDs.
- [ ] `make umpire-build-model`, `make umpire-check-regression`, `make lint-model`, `make lint-code`, generated staleness checks, focused Go tests with `-tags test_dep`, and the scoped integration test with `test_dep integration` pass with evidence recorded.
- [ ] Umpire and model architecture docs describe the final ownership and syntax; `model/Temporal/Feature/Nexus3/Integration.md` no longer claims cancellation lowering is unsupported.
## Done summary
Administrative scope transfer only: cancellation requirements deferred to fn-79 by explicit user decision. No cancellation implementation is claimed complete. Original requirements retained in fn-79; generic syntax/qualification from task 8 retained in fn-78.10. Existing unfinished source edits preserved and worker stopped. This task closes only the scope transfer so generic fn-78 and fn-70 can proceed.
## Evidence
- Commits:
- Tests:
- PRs: