---
satisfies: [R2, R3, R4, R12]
---
# fn-122-capabilities-and-their-laws.2 Add the capabilities declaration, its lifting, except and overriding, with fixtures

## Description
Build the author surface `capabilities(m, limits)(…)` with the six capability types, `except` and `overriding`, and the lifter pass that expands a declaration through the given `Catalog` into Properties and Queries named `<machine>.<law>` with computed totals, plus the law sidecar beside each IR file. No Model declares a capability yet; fixtures prove the lifting, every refusal, and each law form.

**Size:** M
**Files:** `model/umpire/Capabilities.scala` (types, `capabilities`, `except`, `overriding`, each reason-bearing; core); `model/lifter/Capabilities.scala` (new: catalog fold, expansion, generated Property/Scenario/Query records, total computation reusing fn-112.11's formula helper, sidecar writer); one dispatch case in `model/lifter/Claims.scala::fold`; `model/lifter/Syntax.scala` only if a sugar spelling is added (spec R12); `model/lifter/testdata/lifts/Capabilities.scala` + `expected/capabilities.json` + `expected/capabilities.laws.json`; refusal lines in `model/lifter/testdata/lifts/Rejects.scala` + `expected/rejects.txt`; `model/lifter/test/Fixtures.test.scala` roots.
**Touches:** [model/umpire/Capabilities.scala, model/umpire/Syntax.scala, model/lifter/Capabilities.scala, model/lifter/Syntax.scala, model/lifter/Claims.scala, model/lifter/Expressions.scala, model/lifter/testdata/**, model/lifter/test/Fixtures.test.scala]

### Approach
- Capability types are plain case classes with typed fields over the machine's `S`, `O`, `F` (a predicate of another state type does not compile, R2 errors); `Describable` takes the status table fn-112.9 declares; `reach` is a `Seq[ClassRef]` the functional laws prepend to their Scenario; `capabilities(m, limits)` requires `limits` (the bound of generated `verify` Queries) and the given `Catalog`.
- Function-valued fields bind the way fn-112.4 binds direct def arguments: a reference to a named def of the lifted sources enters the fold env and `callee` resolves the law's parameter through it; a bound def whose body is a field path is accepted as a `keeps` projection; a lambda literal is refused at its line naming the def to write (R3). Put the field-reading path beside fn-112.4's case, not in a second resolver.
- Expansion: fold the given `Catalog` as data (a `Vector` of entries naming law defs by reference); for each declared capability and each unordered pair present, emit a Property `<m>.<law>` by folding the law def with `m` and the fields bound and registering the result under `<m>.<law>` regardless of the name string its body writes, a Scenario (free from the declared start, or `reach ++ action` for functional laws) and a Query `<m>.<law>` (`verify` under `limits`, or `find` with `.expect`), with `total` computed by fn-112.11's formula. Emit only existing IR records (spec Edge Cases, no schema change).
- Sidecar: write `model/ir/<file>.laws.json` with each generated claim's law, citation, `promises`, `doesNotPromise`, bindings, every `except`/`overriding` with reason and position, and the catalog's law list with instantiating machines (spec Architecture "The law sidecar"); the gate treats it like an IR file (stale/orphan checks).
- `except(law, because)` suppresses the law's Property and Query; `overriding(law -> def, because)` lifts the entity's def under the law's name after checking its signature against the law's. Reasons go to the sidecar; task 5 forwards them to the accepted-findings file.
- Refusals, each a located lifter refusal or a compiler refusal recorded as such (fn-112's settled R16 rule): unbound action, foreign state type, lambda field, `except` of a law the catalog does not bring, `overriding` with another signature, missing reason, missing `limits`, two capabilities of one kind.
- Core and sugar (spec R12): `Capabilities.scala` is core and imports no `Syntax.scala`; if a convenience spelling is added, it goes to `model/umpire/Syntax.scala` with the core form documented, its matching to `model/lifter/Syntax.scala`, and a fixture proves IR equality with the core spelling.
- fn-114 overlap: fn-114.1 and fn-114.7 edit `model/lifter/**`; this task adds `Capabilities.scala` and one `fold` case; whichever lands second rebases, neither edits the other's cases.

### Investigation targets
**Required:**
- `model/lifter/Claims.scala` - `fold`, `register`, the Query/Limits cases and fn-112.4's function-argument binding
- `model/lifter/Expressions.scala` `callee` - where a parameter call resolves
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.11.md` - the total formula and where Go validates it
- `model/lifter/testdata/lifts/Admission.scala` and `expected/admission.json` - fixture shape
- `model/gate/Gate.scala` settle stage - stale/orphan checks the sidecar joins
**Optional:**
- `model/lifter/test/Fixtures.test.scala` - refusal assertion format `File.scala:line:col`

### Quick commands
```bash
scala-cli test model/lifter
make umpire-check-model
```

### Execution constraints
- No Model under `model/temporal` declares a capability in this task; the six checked-in IR files and every Case stay byte-identical.
## Acceptance
- [ ] `capabilities(m, limits)(…)` with `Closable`, `Terminable`, `Pausable`, `Cancelable`, `Pollable`, `Describable`, `except` and `overriding` compiles only with a reason on `except`/`overriding`, only over the machine's own state type, and only with the given `Catalog`.
- [ ] The lifting fixture shows generated Properties and Queries named `<machine>.<law>` for single, pair, cross-entity and functional law forms, with computed totals Go accepts, using only existing IR records, and the expected sidecar JSON beside the expected IR.
- [ ] Each R2 refusal has a located fixture (or a recorded compiler refusal); the lambda-field refusal names the def to write.
- [ ] Function-valued capability fields, including a field-path def as `keeps` projection, bind through fn-112.4's mechanism with one lifting and one refusal fixture.
- [ ] `Capabilities.scala` imports no `Syntax.scala`; any sugar spelling added has its fixture proving IR equality with the core form.
- [ ] Checked-in IR and Cases are byte-identical; lifter tests, model gate and lint-model pass.
## Done summary
Added the capabilities declaration, its lifting through the given Catalog, `except` and `overriding`, the law sidecar, plain-value argument binding, and fixtures for every form and refusal. No Model under model/temporal declares a capability. model/cases is byte-identical. model/ir changed in positions only, because the laws became objects (the P3 below).

**What changed**
- **`model/umpire/Capabilities.scala` (core):**
  - The types `Closable(status, terminal, rejected)`, `Terminable(terminate, settled, reach)`, `Pausable(pause, unpause, paused)`, `Cancelable(requestCancel, requested, reach)`, `Pollable(dispatch, running)` and `Describable(status: StatusTable)`, all typed through `CapabilityOf[S, +O, +F]`.
  - `capabilities(m, limits)(…)(using Catalog)`, with `except(law, because)` and `overriding(law -> def, because)`.
  - `IrRoot` admits `Capabilities[?]`, so task 3 can register declarations with `irFile`.
- **Laws are objects** (deferred P3 from fn-122.1). The object's `apply` states the law and `extends Law(cites, promises, doesNotPromise)` carries its data. `name` is the object's name, so the repeated string and the untyped `statement` are gone. Direct calls (`terminalStatesAreFinal(m)(…)`) and the fn-122.1 instances are unchanged.
- **Lifter (`model/lifter/Capabilities.scala`, plus one dispatch case in `Claims.fold` and a branch in `Lifting.liftRoot`):**
  - Folds the catalog (`Catalog.single`/`pair`/`++`, vals and givens) and the law objects (their `apply`, their constructor's text, and literals joined with `+`).
  - Binds each law parameter by name to the fields of the capabilities that bring it. Pairs are found among the declared kinds.
  - Registers the Property as `<machine>.<law>` (`generating`) regardless of the name its body writes.
  - Adds a Scenario and a Query of the same name. A law restricted by `when` is a `find` from the start through `reach` and that class; any other law is a `verify` over the free Scenario under `limits`.
  - The total is computed as Go's `QueryTotal` counts it: states × classes × steps, or states × min(steps, schedule), with composed classes counted as Go counts them.
  - Writes `<file>.laws.json` beside each IR file, or beside `out.json` for a roots lift. It holds the claims (law, capabilities, bindings, overriddenBy, position), the waivers (with reason and position) and the catalog entries (cites, promises, doesNotPromise, and the instantiating machines, one per state type).
- **Value binding (`Claims.bodyOf`/`valued`):** a declaring def's parameter that is not a model, claim, Limits, string, integer, bundle or list is bound to its argument term. `lift`, `classOf` and `resolve` read it in its place. `closedIsRejectedUniformly`, `terminateSettles` and `cancelIsRequested` now lift. `declaring` refactored into `bodyOf`, so the expansion reuses fn-112.4's binding path.
- **Refusals at their lines (`lifts/CapabilityRejects.scala`, `rejects.txt`):**
  - an unbound action;
  - a lambda field (names the def to write);
  - two capabilities of one kind, in one declaration or across two;
  - a waiver of a law not brought;
  - a blank reason;
  - an overriding def with other parameters;
  - a lambda passed for a law's function-valued parameter.
- **Compiler refusals (`crossed/Capabilities.scala`, `crossed/NoCatalog.scala`):** a foreign state type, missing `limits`, missing `because`, and no given Catalog.
- **Fixture `lifts/Capabilities.scala`** (expected `capabilities.json` and `capabilities.laws.json`):
  - single: Closable's two laws;
  - pair: Pausable × Pollable, never listed;
  - functional: terminateSettles and cancelIsRequested as finds from `reach`;
  - cross-entity: a composition's Pausable × Pollable through its members' projections;
  - `except` and `overriding`;
  - a fixture catalog given explicitly, whose law's `keeps(status)` reads the bound field-path def `Jobs.phase` (R3).
- **Go:**
  - `capabilities_test.go` recounts every generated total with `QueryTotal` and pins each answer (5 verified and 2 found on the job; the override, the keeps law and the pair verified).
  - Go readers take IR files through the new `umpiremodel.IRPaths`, which leaves out `*.laws.json`: lower `generated.go`, `umpire-gen-cases`, the isolation test, and the golden `Inputs`/`OriginalModels`.
- **README:** laws as objects, value parameters, and capabilities with their expansion, sidecar and refusals.

**Decisions (own)**
- **Laws as objects with `apply`.** This fixes the P3 with no macro, and it reads as the spec's `except(terminalStatesAreFinal, …)`.
- **Catalog keys stay the `Capability` enum.** Each capability type maps to its kind by class name.
- **The functional form is derived from the generated Property's `when`.** No per-law flag. The generated find carries no `.expect`: the expected Run depends on the realization, so task 3 adds it if Case equality needs one.
- **The cross-entity form is a composition declaring capabilities.** No catalog law is cross-entity yet.
- **Capability refusals live in their own `CapabilityRejects.scala`,** not `Rejects.scala`, to stay clear of fn-114.2's edits.
- **One declaration of each kind per machine is enforced across declarations.** Kinds are recorded only once a declaration lifts.
- **Sidecars are skipped by Go IR readers,** as the spec's `model/ir/<file>.laws.json` name requires.
- **The API Contracts' `def terminalStatesAreFinal…` became `object terminalStatesAreFinal extends Law(…)` with an `apply`.** The law's name now comes from its object, not from a duplicated string, and `except`/`overriding` take the law by value. Calls are spelled as before. Recorded per the spec's rule on name changes.
- **Only positions changed in model/ir.** The two activity IR files moved `"line"` values in Pause.scala and Laws.scala; the original baseline ignores positions and passes. Cases are byte-identical.
- **Two definitions of the sidecar filter.** `model.IRPaths` and `golden.IRFiles` both define it, because the live model package may not import the test-only golden package (TestLiveModelDependencyGraph). `original.go` uses golden's.

**Review fixes:**
- Round 1 NEEDS_WORK: only machines count as instantiating (P2, fixed); the bridge reads through IRPaths, with a test (P2, fixed); positions-only IR and the law shape are recorded above (P3); sidecar suffix helpers consolidated as far as the package graph allows (P3).
- FYI, also fixed: the parity regex now accepts one-line law objects.
- FYI, left as is:
  - `sameSignature` compares parameter names, not types. A type mismatch is still refused by the fold or by Go.
  - The sidecar lists only the laws brought to some machine of that file.

**Gates:** all pass (evidence.md):
- `umpire-gen-model` and `umpire-check-model` (lifter tests and CatalogTest);
- the full Go tooling suite;
- the affected Go packages after the review fixes;
- `lint-model` and `lint-code-fast` (0 issues).

**Review:** claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus).
- Round 1: NEEDS_WORK, 2 P2 and 2 P3.
- Round 2: SHIP.

**Shared files for the merge with fn-114.2:**
- lifter: `model/lifter/{Claims,Constants,Context,Expressions,Lift,Lifting}.scala`, `model/lifter/test/Fixtures.test.scala`, `model/lifter/testdata/lifts/expected/rejects.txt`;
- golden: `tools/umpire/internal/golden/{config.json (one appended later_inventory entry),golden.go,original.go}`;
- `model/umpire/IrFile.scala`;
- `tools/umpire/{lower/generated.go,cmd/umpire-gen-cases/main.go,cmd/umpire-ir-bridge/main.go,model/load.go,model/isolation_test.go,model/activity_parity_test.go}`;
- `model/README.md`;
- regenerate `model/ir` and `lifts/expected` after merging.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c300c86a08, f08896b0da, 0a4003aa0e, fc607cd5fc, 9ad221cf55
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -json -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0), go test -json -tags test_dep -count=1 -p 2 ./tools/umpire/model ./tools/umpire/internal/golden ./tools/umpire/lower ./tools/umpire/cmd/... (exit 0, after review fixes), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), flowctl claude impl-review --spec claude:claude-opus-5-5:high (round 2 SHIP)
- PRs: