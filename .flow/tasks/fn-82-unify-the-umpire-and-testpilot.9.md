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
**Files:** `.plans/UMPIRE4_SPEC.md`, `tools/umpire/vocabulary/spec_names_test.go` (new), `tools/umpire/internal/leannames/*.go` (new resolver + tests), `model/ModelLint/ImportGraph.lean` + tests (Verify reservations), `tools/umpire/internal/retiredvocabulary/check.go`
**Touches:** [.plans/UMPIRE4_SPEC.md, tools/umpire/vocabulary/**, tools/umpire/internal/leannames/**, model/ModelLint/**, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Rewrite the glossary sections ("Core concepts", "Where things live", "Trace concepts", "Planning and Artifact concepts", "Runtime concepts", "Exploration concepts", "Verification and claim concepts") to the vocabulary table: Model, Machine, Table, Vocabulary, Step, Fact, Trace, Scenario, Property/Clause/Branch, Correlated, Deadline, Limit, Query, Search, Plan, Verdict, Monitor, Evidence, Projection, Variations, Inventory, Provenance. Remove Test, Case Artifact, Execution, Stage Status, Assurance Method, Claim Assessment, Exact Replay; collapse Scenario/Behavior. Mark `Temporal.Verify`, `Umpire.Verify.Veil`, and the Veil consumers as planned under fn-24/fn-25, and remove their reservations from `ImportGraph.lean:31,35,111,127-134,141-143,218-219,258`.
- Move the twenty `Retired:` tombstones (SEM-10..15, ART-03/05/06/08, EVD-02/03/06/09/10, QLF-04) to a closing appendix, IDs intact. Draft new rules under new IDs: one word per concept, compound-only retirement through the gate, glossary resolution check; mark them awaiting GOV-02 approval.
- Repair backticks: the languages are modules, so cite `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query` as modules and `CheckedProperty` etc. as declarations; PLN-06 cites `Plan.Steps`; AUT-07/AUT-08 cite `FiniteMachine`, `Machine`, `DraftModel`, `checkModel`; MOD-12/13/14 keep their Go paths.
- Add `spec_names_test.go` beside `retired_vocabulary_test.go` with a small resolver in a new `tools/umpire/internal/leannames` package: walk `model/**/*.lean` (skipping `.lake`), derive module names from paths (`model/Umpire/Model/Check.lean` is `Umpire.Model.Check`), and index declaration heads by tracking `namespace`/`end` nesting and matching `structure`, `inductive`, `class`, `abbrev`, `def`, `theorem`, `macro`, and `syntax` lines into qualified names. The test reads the spec, extracts backticked `Umpire.*`, `Testpilot.*`, `Temporal.*`, `Shared.*` names, and asserts each is a module, a namespace, or an indexed declaration (a trailing field or constructor segment resolves against its parent declaration). A name tagged planned resolves only while its owning spec is open in `.flow/specs`. `Tools.LeanSourceInventory` maps files to modules only and knows no declarations, so it is not the resolver.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md:1-110,181-260,339-360,418-455` — the glossary sections to rewrite
- `tools/umpire/vocabulary/retired_vocabulary_test.go` — test style and repository-root handling to mirror
- `tools/umpire/internal/retiredvocabulary/check.go:93-131` — the tree walk to mirror in the resolver (skip `.lake`, symlinks)
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
- [ ] `go test -tags test_dep ./tools/umpire/vocabulary/... ./tools/umpire/internal/leannames/...` runs the resolver tests and the resolution test; the resolver finds a nested `def` inside a `namespace`, a `structure` field, and a module name; an unresolvable backticked name fails the spec test, and a planned term whose owning spec is closed fails it
- [ ] `make umpire-check-retired-vocabulary` passes on the rewritten spec
## Done summary
`.plans/UMPIRE4_SPEC.md` now defines each concept once, under the word the vocabulary table fixes, and a Go test proves every dotted Lean name it cites in backticks exists.

- Glossary: the six concept sections are rewritten to the table. Terms with no code and no owner are gone — Test, Case Artifact, Execution, Stage Status, Assurance Method, Claim Assessment, Exact Replay — and Scenario absorbs Behavior. The rules that used those words are rewritten rather than left citing them, so the removal is not cosmetic.
- Retired rules: the sixteen `Retired:` tombstones (SEM-10..15, ART-03/05/06/08, EVD-02/03/06/09/10, QLF-04) sit in a closing appendix with their IDs and text intact, so a design that cites one still resolves and GOV-01 keeps the numbers reserved. The task's description says twenty; sixteen is what its own enumerated list holds.
- Planned terms: `Temporal.Verify`, `Umpire.Verify.Veil` and their consumers exist nowhere in the tree. MOD-05 and VER-01..VER-06 are marked planned under fn-24 and fn-25, and `ModelLint` no longer carries the `temporalVerify`/`umpireVeil`/`optInVerify` classes, their classifiers, `verifyConsumers`, or the `verificationIsolation` rule. The two companion documents that described that machinery in the present tense say what it is now.
- New rules, all **awaiting GOV-02 approval**: SEM-19 one word per concept, SEM-20 compound-only retirement through the gate, MOD-15 the resolvable-glossary check.
- The check is `tools/umpire/vocabulary/spec_names_test.go` over a new `tools/umpire/internal/leannames`. The index is syntactic — module names from paths, declaration heads from `namespace`-tracked lines, fields and constructors from the block that declares them — which is what lets an ordinary Go test run it. `leannames.Unresolved` is the decision, and both refusals are pinned against a fixture: a name the tree does not have, and a planned term whose owner is closed.

Four review rounds against claude-fable-5-1 at high found real defects, each fixed: the lint driver still built the retired `modelLintTests` target; `Resolve`'s parent fallback accepted any invented segment under a structure, which is how the spec's one bad citation (`Umpire.Scenario.TraceAddress`) passed the very test built to catch it; four glossary words ended up naming two concepts each, which SEM-19 forbids; the planned marker skipped the index entirely, so a typo inside a planned rule passed and a real name would have started failing when its owner closed; comment-depth counting on the raw line let `+/-10%` inside a description string drop every later declaration in that file; and the Verdict entry named a constructor the code does not have.

The fifth round could not run against claude-fable-5-1 — the account's Fable quota was exhausted mid-task — so the final SHIP is pinned to claude-sonnet-4-5 at high, still off the implementing model. Its report is thinner than the fable rounds', and I record that rather than claim equivalent scrutiny.

Carried debt: `ModelCoordinate` has not become `TraceAddress`, and `PropertyEndpointAnswer.unresolved` has not become `inconclusive`; the glossary names what the code has and says SEM-19 makes each a rename. The resolver reads the tree syntactically, so a name only the elaborator produces is invisible to it.
## Evidence
- Commits: d7d0bf1dda, e7642fe8f7, c5323914a1, e3ea77cf2c, 5f36fa91ce
- Tests: CC=/usr/bin/cc go test -tags test_dep -count=1 ./tools/umpire/..., make umpire-check-regression, make lint-model, make lint-code GOLANGCI_LINT_FIX=false, make umpire-check-retired-vocabulary, CC=/usr/bin/cc go vet -tags test_dep ./..., go run ./tools/planindex, cd model && lake exe umpire-lint-tests
- PRs:
stage: plan-sync - skipped(config: planSync.enabled != true)
