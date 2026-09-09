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
- Run the full set: `make lint-model`, `make umpire-check-regression`, `make lint-protos`, `go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`, and `go test -tags 'test_dep integration' ./tests -run 'TestTestpilot|TestUmpire'`.

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
- [ ] `make lint-model`, `make umpire-check-regression`, `make lint-protos`, the Go packages under `tools/umpire`, the Testpilot facade, and the fixture package, and the tagged live selector all pass on the finished tree
- [ ] `.plans/UMPIRE4_ORDER.md` carries the fn-82 entry


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
