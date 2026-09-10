---
satisfies: [R7]
---
# fn-82-unify-the-umpire-and-testpilot.10 Close-out documents, roadmap, and full gate run

## Description
Close-out (R7, spec §R7): rewrite the three model documents and the roadmap under the new
vocabulary, confirm the gate holds every compound name the spec retires, and run the full gate
set once more on the finished tree.

**Size:** M
**Files:** `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_ORDER.md`, `tools/umpire/CLEANUP_INVENTORY.md`, `tools/umpire/CONTEXT.md`, `model/Umpire/Property/COMPATIBILITY.md`, `tools/umpire/internal/retiredvocabulary/check.go`
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/Umpire/Property/COMPATIBILITY.md, .plans/UMPIRE4_ORDER.md, tools/umpire/*.md, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Earlier tasks respelled tokens in these documents to keep the gate green; this task rewrites them so they read as one description of the tree: the import map in `model/ARCHITECTURE.md`, the facade table and the Model ownership table in `model/Umpire/ARCHITECTURE.md:20-75`, the authoring walkthrough and the command list in `model/README.md`, and the `Case production` and `Runtime handoff` sections. Keep every statement checkable against the code (memory: golden fields updated by vocabulary alone do not prove ownership statements match executable interfaces).
- `.plans/UMPIRE4_ORDER.md`: add the fn-82 entry with its dependencies and the retired-name policy; respell the Nexus2/Nexus3 and `temporal-testpilot` mentions.
- Audit `buildRetiredRules` against the spec's vocabulary table and every task's retired list; add any compound still missing.
- Run the full set: `make lint-model`, `make umpire-check-regression`, `make buf-breaking`, `make lint-protos`, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`, and `go test -tags 'test_dep integration' ./tests -run 'TestTestpilot|TestUmpire'`.

### Investigation targets
**Required** (read before coding):
- `model/README.md`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md` — the documents to rewrite
- `.plans/UMPIRE4_ORDER.md:1-120` — roadmap entries and their format
- `tools/umpire/internal/retiredvocabulary/check.go:264-338` — the rule list to audit

**Optional** (reference as needed):
- `.flow/specs/fn-82-unify-the-umpire-and-testpilot.md` §The vocabulary — the normative table to audit against

### Key context
- No new CI workflow and no generated-API drift gate (declined-concept ledger).
- Historical `.plans` documents other than `UMPIRE4_SPEC.md` and `UMPIRE4_ORDER.md` are not edited.
## Acceptance
- [ ] The three model documents and the roadmap describe the tree under the new vocabulary with no retired compound and every named module or declaration present in the tree
- [ ] `buildRetiredRules` contains every compound name the spec's vocabulary table retires
- [ ] `make lint-model`, `make umpire-check-regression`, `make buf-breaking`, `make lint-protos`, the Go packages under `tools/umpire`, the Testpilot facade, and the fixture package, and the tagged live selector all pass on the finished tree
- [ ] `.plans/UMPIRE4_ORDER.md` carries the fn-82 entry
## Done summary
The three model documents, the roadmap and the tool context now read as one description of the tree, the gate holds every compound the spec's vocabulary table retires and can hold, and the full gate set ran on the finished tree.

- `model/ARCHITECTURE.md`: the import map named Observation, Space and Planning, which are now `Umpire.Evidence`, `Umpire.Variations` and `Umpire.Search`; the ownership bullets and the Artifact section follow.
- `model/Umpire/ARCHITECTURE.md`: the facade table credited `Umpire.Case` with provenance and with "temporary aliases for generated Testpilot protocol types". Task .7 moved provenance to `Umpire.Provenance` and deleted the aliases, so the table lists the four Case owners — Compiler, Coverage, Correlated, Projection — and Provenance separately, and the protocol section says what `Umpire.Provenance.make` actually encodes.
- `model/README.md`: the authoring walkthrough, the command list and the Case-production section.
- `tools/umpire/CONTEXT.md`: it still defined **Horizon** and told the reader to avoid "Deadline", which is the word the wire field took in task .7. It defines Deadline, and gains Rule, Correlated and Opcode; the Profile entry's "capabilities" are Opcodes.
- `.plans/UMPIRE4_ORDER.md`: merged into the maintainer's existing fn-82 entry rather than overwritten. The progress paragraph records all ten tasks and what .5 through .10 landed, and a new paragraph states the retired-name policy, including why three rules need a narrower boundary than the shared compound pattern.

Gate audit: I extracted the Retires column of the spec's vocabulary table mechanically and diffed it against `buildRetiredRules`. Eight compounds were missing and are added, each proved to fire by planting it: `ExperimentSpaceDeclaration` and `TargetBehaviorDomainAvailability` (longer names built on held ones, which the identifier boundary excludes), the four retired Limit units in their wire spelling (`candidateEvaluations`, `selectedActions`, `semanticTransitions`, `experimentSpecs`), `observationPositions`, and `testpilot.Capability` — whose boundary keeps the live `CapabilityBridge` and `CapabilityEffect` Driver seam out of it.

Four names the table retires are still what the code calls them, so the gate cannot hold them: `ModelCoordinate`, `ModelTraceStep`, `PropertyEndpointAnswer` and `expectationFact`. `PlannerRun` is the same case inside `Umpire.Variations`. Each is a rename the spec scheduled and no task performed; the glossary names what exists rather than what was planned.

Full gate set on the finished tree: `make lint-model` (163 findings, all in the generated `Temporal.API.Proto`, against a 169 baseline; the import graph passes), `make umpire-check-regression` green including the six live Testpilot identities, `make buf-breaking`, `make lint-protos`, `make umpire-check-retired-vocabulary`, `go test -tags test_dep` over `./tools/umpire/...`, `./common/testing/testpilot/...` and `./tests/testcore/testpilot/...`, `go vet` at 15 against a 15 baseline, `make lint-code` at 128 against a 128 baseline, `go run ./tools/planindex` at 48 against a 48 baseline, and the tagged live selector `go test -tags 'test_dep integration' ./tests -run 'TestTestpilot|TestUmpire'`.

Review is SHIP. It could not run against claude-fable-5-1 — the account's Fable quota was exhausted during task .9 — so it is pinned to claude-sonnet-4-5 at high, still off the implementing model. Its one P3 was half right: "Step" was lowercase among capitalized glossary terms and is fixed; "Model Outcome" is a defined glossary term and keeps its name.
## Evidence
- Commits: b553f9bbe9, 0b0b79daf4
- Tests: make lint-model, make umpire-check-regression, make buf-breaking, make lint-protos, make umpire-check-retired-vocabulary, CC=/usr/bin/cc go test -tags test_dep -count=1 ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/..., CC=/usr/bin/cc go test -count=1 -tags 'test_dep integration' ./tests -run 'TestTestpilot|TestUmpire', make lint-code GOLANGCI_LINT_FIX=false, CC=/usr/bin/cc go vet -tags test_dep ./..., go run ./tools/planindex
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
