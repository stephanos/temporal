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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
