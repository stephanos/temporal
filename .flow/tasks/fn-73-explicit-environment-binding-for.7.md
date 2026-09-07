---
satisfies: [R7]
---
# fn-73-explicit-environment-binding-for.7 Document environment binding and run final gates

## Description
Update the active normative, architecture and package documentation for the shipped binding boundary and fold focused protocol/authoring checks into the existing local regression gate (R7). Then run the complete fn-73 verification matrix.

**Size:** M
**Files:** active Umpire/model/package READMEs and architecture documents, Nexus3 integration note, `.plans/UMPIRE4_ORDER.md`, `Makefile`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, model/ARCHITECTURE.md, model/README.md, model/Umpire/ARCHITECTURE.md, model/Temporal/Feature/Nexus3/Integration.md, common/testing/testpilot/README.md, common/testing/testpilot/temporal/**/README.md, tests/testcore/testpilot/README.md, Makefile]

### Approach
- Document symbolic Case declarations, physical Profile ownership, immutable Prepare resolution, binding identity and Validate-before-Open ordering in active architecture material.
- Clarify that symbolic endpoint IDs contain no physical transport address and that Behavior Fingerprints/Contract/provenance remain behavioral.
- Describe explicit legacy 1.0 versus symbolic 1.1 Driver modes and the two-environment Nexus3 evidence.
- Correct fn-70 portability wording so environment rebinding changes prepared/Driver identity rather than Case bytes, while retaining fn-70's dependency on fn-73.
- Add the existing focused Testpilot protocol/authoring checks to `umpire-check-regression`; do not expand GitHub workflow coverage or add a broad generated-API drift gate.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:55-79,148-158,296-387` — normative contracts
- `.plans/UMPIRE4_COMPONENTS.md:7-50` — current component map
- `.plans/UMPIRE4_ORDER.md:97-108` — roadmap dependency narrative
- `model/README.md:28-46,112-214` — public model and fixture workflow
- `common/testing/testpilot/README.md:3-11` — Prepare/Run facade

**Optional** (reference as needed):
- `common/testing/testpilot/temporal/README.md` and subpackage READMEs — shared Driver ownership
- `model/Temporal/Feature/Nexus3/Integration.md:40-94` — integration boundary

### Key context
Leave historical design records and completed fn-68/fn-71/fn-72 receipts unchanged. Preserve the declined decision against new CI coverage.

## Acceptance
- [ ] Active normative and public docs agree on Case 1.0/1.1, symbolic/physical ownership, immutable identity and Validate-before-Open ordering.
- [ ] Package docs explain symbolic and legacy modes, prepared resource use, transport authority and the two-environment fixture proof.
- [ ] fn-70 states that a shared Case artifact is invariant across bindings and still depends on fn-73; fn-74 remains independent and owns later error classification/activation redesign.
- [ ] `umpire-check-regression` includes the existing Testpilot protocol and authoring checks without `.github/workflows` or broad drift-gate changes.
- [ ] `make proto`, focused Testpilot protocol/authoring/fixture gates, serial Go tests with `-tags test_dep`, applicable Lake builds/model lint, the live integration gate, `make lint-model`, `make lint-code`, and `git diff --check` pass or record only verified pre-existing/resource failures under project policy.
- [ ] Completion evidence names any unchanged legacy fixtures and confirms no new third-party dependency or proof axiom.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
