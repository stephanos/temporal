---
satisfies: [R1]
---
# fn-82-unify-the-umpire-and-testpilot.9 Glossary rewrite and spec-name resolution test

## Description
Glossary rewrite and the resolution test (R1, spec §R1): `.plans/UMPIRE4_SPEC.md` defines each
term of the vocabulary table once, doc-only terms are removed or marked planned, retired rules
move to an appendix, the vocabulary policy gets new rule IDs for GOV-02 approval, and a Go test
proves every backticked module or declaration name in the spec resolves.

**Size:** M
**Files:** `.plans/UMPIRE4_SPEC.md`, `tools/umpire/vocabulary/spec_names_test.go` (new), `model/ModelLint/ImportGraph.lean` + tests (Verify reservations), `tools/umpire/internal/retiredvocabulary/check.go`
**Touches:** [.plans/UMPIRE4_SPEC.md, tools/umpire/vocabulary/**, model/ModelLint/**, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Rewrite the glossary sections ("Core concepts", "Where things live", "Trace concepts", "Planning and Artifact concepts", "Runtime concepts", "Exploration concepts", "Verification and claim concepts") to the vocabulary table: Model, Machine, Table, Vocabulary, Step, Fact, Trace, Scenario, Property/Clause/Branch, Correlated, Deadline, Limit, Query, Search, Plan, Verdict, Monitor, Evidence, Projection, Variations, Inventory, Provenance. Remove Test, Case Artifact, Execution, Stage Status, Assurance Method, Claim Assessment, Exact Replay; collapse Scenario/Behavior. Mark `Temporal.Verify`, `Umpire.Verify.Veil`, and the Veil consumers as planned under fn-24/fn-25, and remove their reservations from `ImportGraph.lean:31,35,111,127-134,141-143,218-219,258`.
- Move the twenty `Retired:` tombstones (SEM-10..15, ART-03/05/06/08, EVD-02/03/06/09/10, QLF-04) to a closing appendix, IDs intact. Draft new rules under new IDs: one word per concept, compound-only retirement through the gate, glossary resolution check; mark them awaiting GOV-02 approval.
- Repair backticks: the languages are modules, so cite `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query` as modules and `CheckedProperty` etc. as declarations; PLN-06 cites `Plan.Steps`; AUT-07/AUT-08 cite `FiniteMachine`, `Machine`, `DraftModel`, `checkModel`; MOD-12/13/14 keep their Go paths.
- Add `spec_names_test.go` beside `retired_vocabulary_test.go`: read the spec, extract backticked `Umpire.*`, `Testpilot.*`, `Temporal.*`, `Shared.*` names, and assert each is a module in `model/` or a declaration found by the `Tools.LeanSourceInventory` output (invoke `lake exe` or parse `model/` sources the way `tools/umpire/regression` already does); a name tagged planned resolves only while its owning spec is open in `.flow/specs`.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:1-110,181-260,339-360,418-455` — the glossary sections to rewrite
- `model/Tools/LeanSourceInventory.lean` — the inventory the resolution test can reuse
- `tools/umpire/vocabulary/retired_vocabulary_test.go` — test style and repository-root handling to mirror
- `model/ModelLint/ImportGraph.lean:25-40,101-145,210-225,250-262` — Verify reservations

**Optional** (reference as needed):
- `.flow/specs/fn-24-lean-native-verification-receipts-and.md`, `fn-25-optional-callerclosure-veil-binding-and.md` — owners of the planned Verify terms

### Key context
- GOV-01 forbids renumbering or reusing rule IDs; retired rules keep theirs in the appendix.
- The gate scans `.plans/UMPIRE4_*.md`, so the rewrite must use no retired compound.
- fn-80 .9 also drafted rules under new IDs; take the next free IDs after those.

## Acceptance
- [ ] Every capitalized term in the spec names a declaration or module (backticked and resolvable) or is marked planned with an open owning spec; Test, Case Artifact, Execution, Stage Status, Assurance Method, Claim Assessment, Exact Replay, and the separate Behavior term are gone
- [ ] Retired rules sit in an appendix with IDs intact; the vocabulary rules carry new IDs marked awaiting GOV-02 approval
- [ ] `ModelLint` reserves no Verify module and `make lint-model` passes
- [ ] `go test -tags test_dep ./tools/umpire/vocabulary/...` runs the resolution test; an unresolvable backticked name fails it, and a planned term whose owning spec is closed fails it
- [ ] `make umpire-check-retired-vocabulary` passes on the rewritten spec


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
