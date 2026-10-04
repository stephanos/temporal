---
satisfies: [R6, R7, R8, R16]
---
# fn-112-make-the-standalone-activity-scala.2 Add captured declaration names, evidence defaults and refinement reads

Touches: [model/umpire/**, model/lifter/Context.scala, model/lifter/Declarations.scala, model/lifter/Claims.scala, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, model/README.md]

## Description
Add the general declaration/default machinery that later feature migrations consume, without moving feature files yet.

**Size:** M
**Files:** model/umpire declaration, identity-scope and claim APIs; model/lifter Context/Declarations/Claims; focused lifter fixtures and README.

### Approach
- Teach the lifter to take names from declaring vals for the R7 declaration matrix while retaining explicit-name overloads only where a name intentionally differs. Keep computed Query names supported.
- Name capture is a default, not a rule that every Property or Query has a `val` (spec R7, Decision Context "Capabilities and laws"): a Property or Query built inside a `def` over a machine argument (`admissionQueries(m)`, the R4 shared-claim defs of task 4) keeps the one explicit-name form, and the lifter must not refuse or rename a declaration because no `val` declares it. fn-122 later generates claims with no `val`, named `<machine>.<law>`; do not add a lifter check that would block that.
- Add `DefinitionScope`, pinned once per former source owner, and route symbol-based IDs through its owner plus the captured val name. Cover actions, monitors, assumptions, channels and realizations; reject duplicate/nested/conflicting scopes. Do not add per-declaration ID strings.
- Add machine type inference, declared-start Scenario defaults, fact-to-same-name evidence defaults and refined-machine Property reads. For R8, `in` drops its `using Reads[S, P]` clause (`model/umpire/Claims.scala:147`): `Machine[S, O, F]` does not carry the refined state type, so no given can be derived from the declared refinement; the lifter's existing refusal of a Property and a Scenario of unrelated machines (`model/lifter/Claims.scala:123`) stays the check, and `Reads`/`Reads.through` leave the author surface. Record the choice in the done summary (the alternative, a type parameter on `Machine`, changes every declaration's shape).
- Detect duplicate captured names and invalid/missing refinement projections with located diagnostics.
- Exercise generated symbol names and local helper calls in positive and refusal fixtures; compare against task 1 after every fixture migration.
## Acceptance
- [ ] Fixtures cover captured names for machine, derived machine, composition, Property, Scenario, Query, Limits, timer, action, monitor, assumption, hole, channel and realization declarations, plus scoped legacy owners for every symbol-based ID kind.
- [ ] Family-as-given, inferred machine types, omitted Scenario starts, evidence exceptions and implicit refinement reads lift to the original IR meaning; `in` takes no `Reads` given and the lifter's unrelated-machine refusal has a fixture.
- [ ] A Property or Query declared inside a `def` over a machine argument with the explicit-name form lifts as today; one fixture proves the lifter requires no `val` for a Property or Query.
- [ ] Duplicate/ambiguous names and unavailable refinement reads are refused at their source.
- [ ] DefinitionScope fixtures reproduce the task-1 owner/name map exactly and refuse nested, duplicate and conflicting pins without a per-declaration escape hatch.
- [ ] Task-1 equivalence, focused lifter tests and lint-model pass.
## Done summary
Added the declaration-naming and defaults machinery later fn-112 migrations consume, without moving feature files.

**What changed**
- `model/umpire`: a captured-name form for every R7 kind. These are `machine[S, O, F] { … }` (types stated once, in the call or as the val's type) with `given Family`, `restrict(actions*)`, `compose[S](members*)`, `m.property`, `m.scenario`, bare `query`, `Limits(steps =, actions =, search =)`, `action(party)`, `timer`, `internal`, `assume`, `hole`, `monitor[S,O,F,M](initial)…`, `channel[M](capacity = …)` and `Realization(machine = …)`. Also new: `DefinitionScope(former)`, `evidence(PartialFunction)` for exceptions only, `in` with no `Reads` given, and `Reads` removed. The explicit-name forms stay.
- `model/lifter`:
  - Names are read from the declaring val, threaded only through receiver chains (`fold(…, named)`). A captured form with no val, or a compiler-made val name, is refused at its line. The explicit forms need no val, and a local val inside a helper names its declaration.
  - Symbol IDs go through `definitionId`, which applies the owner's pin. Doubled, nested, self and computed pins are refused, and so are two declarations that would share an ID.
  - Defaults: a Scenario's start falls back to the machine's single start, or for a composition to the record of its members' starts. Evidence lines default to the fact's own name, in catalog order around the author's lines. A refined read comes from the Scenario machine's `refines`, keeping the archived refusal texts.
  - Duplicate names are refused for machines/compositions, actions within one machine, Limits with different bounds, assumptions, holes, channels and realizations.
  - `resolveSymbol` no longer treats `val x = timer` as an alias.
- Fixtures:
  - `lifts/Spelled.scala` and `lifts/Captured.scala` declare one Model both ways, covering every kind. The new test requires one IR, apart from positions and the owner of types/functions. It also requires every captured symbol ID to be pinned to Spelled's owner before any substitution.
  - The DefinitionScope probe now uses real pins and compares against `owners.json`.
  - `Rejects.scala` gains 20 refusals. `crossed/QueryPair.scala` was dropped (that pair is now the lifter refusal `unrelatedRead`). `werror` now tests a non-exhaustive step match, since evidence may be partial.
  - README documents all of this.
- Production Models: only the `Reads` givens and `using` were removed (forced by R8). The IR changed in line numbers only.

**Decisions (taken autonomously)**
- **R8:** `in` takes no `Reads` given. `Machine[S, O, F]` does not carry the refined type, and the alternative (a type parameter on `Machine`) changes every declaration's shape. The lifter's refusal is the check (`crossedRead`, samestate, `unrelatedRead`).
- **No `machine[Product]` type bundle.** The three types are stated once, in `machine[S, O, F]` or as the val's type (inferred from the expected type). A bundle would need match types inside a context-function body for no gain. This changes a contract name; recorded here per the spec.
- **A fact case with fields needs its own evidence line,** covering all its values. Only fieldless facts default to their name, which matches R6's `statusTimedOut(_)` line. `evidence(f: F => String)` stays for named total functions.
- **Leading default `name: String = ""`** on `channel`, `Limits` and `Realization`, because Scala allows defaults in only one overload.
- **A pin applies to the direct members of its owner.** Several owners may pin one former owner (task 8 splits files); a collision is refused instead.
- **No new `expected/*.json` files:** the fn-115 golden inventory refuses unknown files there. The twin is an assertion test instead.
- **Outside the Touches list:** `model/lifter/Compositions.scala` (captured `compose`), production `Reads` removals, and `tools`-free fixture dirs `crossed`/`werror`/`samestate` test files.
- Another session's commit 01cde2f4a6 swept two staged index entries of this task into it: the deletion of QueryPair.scala and the rename of Evidence.scala to Steps.scala. History was not rewritten.

**Review:** claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`. The writer and reviewer are the same family (Opus). Round 1 was SHIP with 3 P3s, all fixed in 480958c4cb: author evidence order is now preserved (and a guarded-only fact still gets its default), pins sort by file and line, and one shared duplicate-name helper replaces the copies. The SHIP verdict predates those P3 fixes. After them I re-ran gen-model, the lifter tests, the baseline check and lint-model.

**Deferred P3:** no fixture yet for overlapping and guarded evidence lines under defaults. The register "declared twice" message for two same-named Queries in different owners could name both.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 713868cef0, 480958c4cb
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 72 s), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 20 s), mise exec -- scala-cli test model/lifter (exit 0, 48 s), mise exec -- make lint-model (exit 0), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 199 s, at 713868cef0; 480958c4cb changes only lifter refusal texts, IR unchanged), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0)
- PRs: