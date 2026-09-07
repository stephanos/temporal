---
satisfies: [R7]
---
# fn-73-explicit-environment-binding-for.7 Document environment binding and run final gates

## Description
Update the active normative, architecture and package documentation for the shipped binding boundary and fold focused protocol/authoring checks into the existing local regression gate (R7). At the user's direction, place the reusable Temporal adapter under the generic Testpilot tree and migrate every repository consumer and path-sensitive gate. Then run the complete fn-73 verification matrix.

**Size:** M
**Files:** active Umpire/model/package READMEs and architecture documents, Nexus3 integration note, Temporal adapter packages and consumers, path-sensitive regression/vocabulary gates, `.plans/UMPIRE4_ORDER.md`, `Makefile`, `.github/workflows/umpire.yml`
**Touches:** [.plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_COMPONENTS.md, .plans/UMPIRE4_ORDER.md, model/ARCHITECTURE.md, model/README.md, model/Umpire/ARCHITECTURE.md, model/Temporal/Feature/Nexus3/Integration.md, common/testing/testpilot/README.md, common/testing/testpilot/temporal/**, tests/testcore/testpilot/README.md, tests/testcore/testpilot/artifact_test.go, tests/testpilot_async_nexus_case_test.go, tools/umpire/regression/ci_workflow_test.go, tools/umpire/internal/retiredvocabulary/check.go, Makefile, .github/workflows/umpire.yml]

### Approach
- Document symbolic Case declarations, physical Profile ownership, immutable Prepare resolution, binding identity and Validate-before-Open ordering in active architecture material.
- Clarify that symbolic endpoint IDs contain no physical transport address and that Behavior Fingerprints/Contract/provenance remain behavioral.
- Describe explicit legacy 1.0 versus symbolic 1.1 Driver modes and the two-environment Nexus3 evidence.
- Correct fn-70 portability wording so environment rebinding changes prepared/Driver identity rather than Case bytes, while retaining fn-70's dependency on fn-73.
- Relocate the repository-local Temporal adapter to `common/testing/testpilot/temporal`, retain `server`, `worker`, and `internal/delivery` beneath it, rename the root Go package to `temporal`, and remove the obsolete package path after migrating every repository consumer. Do not retain a forwarding compatibility package.
- Add the existing focused Testpilot protocol/authoring checks to `umpire-check-regression`; do not expand GitHub workflow coverage or add a broad generated-API drift gate. Update the existing workflow selector only as required by the relocation.

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
Leave historical design records and completed fn-68/fn-71/fn-72 receipts unchanged. Preserve the declined decision against new CI coverage; the workflow path-only selector migration does not add coverage.

## Acceptance
- [ ] Active normative and public docs agree on Case 1.0/1.1, symbolic/physical ownership, immutable identity and Validate-before-Open ordering.
- [ ] Package docs explain symbolic and legacy modes, prepared resource use, transport authority and the two-environment fixture proof.
- [ ] fn-70 states that a shared Case artifact is invariant across bindings and still depends on fn-73; fn-74 remains independent and owns later error classification/activation redesign.
- [ ] The reusable adapter is rooted at `common/testing/testpilot/temporal`, all repository imports and path-sensitive gates use it, and the obsolete `common/testing/temporaltestpilot` tree is absent.
- [ ] `umpire-check-regression` includes the existing Testpilot protocol and authoring checks without broad drift-gate or workflow-coverage changes; the workflow contains only the required selector migration.
- [ ] `make proto`, focused Testpilot protocol/authoring/fixture gates, serial Go tests with `-tags test_dep`, applicable Lake builds/model lint, the live integration gate, `make lint-model`, `make lint-code`, and `git diff --check` pass or record only verified pre-existing/resource failures under project policy.
- [ ] Completion evidence names any unchanged legacy fixtures and confirms no new third-party dependency or proof axiom.

## Done summary
Documented the Case 1.0/1.1 binding boundary, immutable preparation identity, Validate-before-Open ordering, and the two-environment Nexus3 proof. Added the existing protocol and authoring gates to local regression. Relocated the repository-local Temporal adapter to `common/testing/testpilot/temporal` with its server, worker, and delivery boundaries intact; migrated all repository consumers and path-sensitive checks and removed the obsolete tree.

## Evidence
- Commits:
- Tests: `make proto`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-case-runtime-conformance`; serial `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; focused workflow/documentation/dependency regressions; focused golangci-lint over `./common/testing/testpilot/...` (`0 issues`); prior focused two-environment live Nexus3 test and `make umpire-check-live-tests`; `make lint-model` build/import/Batteries/model-linter phases plus isolated successful builtin-lint retry; `make lint-code` retained 1,279 verified inherited findings outside the moved subtree; `git diff --check`; `git diff --cached --check`.
- PRs:

Legacy conformance fixtures are unchanged. No third-party dependency, Lean axiom, `sorry`, `admit`, or `native_decide` was added.
