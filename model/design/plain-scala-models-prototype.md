# Plain-Scala Models (fn-190.1): prototype on Kettle and verdict

Status: phase-1 prototype with a verdict, 2026-10-17. Scala 3.9.0, JVM 27, the native strict flags
(`-Werror -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -language:strictEquality`).
No production source changed. The probe is kept at
[`plain-scala-prototype/`](plain-scala-prototype/README.md) with a README and `run.sh`; its logs are
`plain-scala-prototype/logs/{compile,test,report,non-exhaustive}.{log,status}`.

Scope follows the owner's phasing (spec "Phasing", 2026-10-17): this task proves the **authoring
form and its native semantics**. Vars are snapshotted and restored per step, so effects, guards and
check-block claims execute as Scala. The stand-in plugin that lifts the form to `field`/`copy` IR
and the comparison with the retained Kettle IR moved to fn-190.9; the items that only the lift can
answer are marked **deferred (fn-190.9)** below, with what the native run already shows.

## 0. Answer in brief

**The form works natively on Kettle and the Facets fixture, under the strict flags, with no
compiler plugin.** 31 tests pass (`logs/test.status`); the run prints the schema, the transition
arms, every claim's verdict and the progress answers (`logs/report.log`).

- **State as vars, assignment-block effects, guards and claims reading vars by name** all run as
  plain Scala. A macro reads the Machine's vars (declaration order, declared types with aliases
  intact, bounds) and generates typed accessors. The runtime restores a snapshot into the vars, runs
  the transition or claim, and snapshots again.
- **`before` is the State record, typed.** It is a `transparent inline` macro that returns a record
  refined with the enclosing Machine's vars, so `before.temperature: Int` type-checks, a misspelt
  field is a type error, and `before` outside `postconditions` is refused at compile time with the
  spec's text.
- **Claim members with no kind word need one runtime trick.** An `object` nested in an `object`
  runs its body in the JVM static initialiser (`<clinit>`), which happens once. Creating a second
  instance does not rerun it (javap, §2.4). The runtime therefore evaluates a section by defining
  its class again in a child class loader, about 80 µs per evaluation. This is phase-1 only and
  goes away with the lift.
- **Four findings in the spec's Kettle** (§3): `neverBeeps` in `invariants` is refuted by the
  default checks; `boils` as written is vacuous under `only(heat, failover)` and its `within = 81`
  is the bound from 20 °C, while the bound from `oneOf(-40 to 40)` is 141; outcome and fact types
  cannot be concrete type members with defaults; and `object scenario` beside `class Scenario`
  breaks the build on a case-insensitive filesystem.

**Verdict: adopt the form.** The native runtime design for fn-190.2 is in §4.

## 1. What was built

`dsl/` is a DSL and runtime of about 780 lines (with its one test helper), separate from `model/framework`:

- `Facet`/`Machine` with `type Outcome` and `type Fact` as abstract members, so a Machine is
  `extends Machine` with no type arguments.
- `Actor`/`Internal` with `action`/`step` and no `this`.
- An `inline on(action)(body)` macro.
- `check`, `after`, `before`, `invariant`, `leadsTo`, `only`, `fair`, `scenario.starts`, `oneOf`,
  `choose` with a by-name `->`, `reject`, and `type UpTo[N <: Int] = Int`.

The fixtures:

- `kettle/Kettle.scala` is the spec's Kettle verbatim (API Contracts).
- `toplevel/Kettle.scala` is the same Kettle as top-level definitions.
- `lamp/Lamps.scala` is `dsl/Facets.test.scala`'s lamp. It has a phase var declared abstractly in
  two facets, `match` transitions, a `choose` with blocks, an `if` arm, `+=` on an `UpTo[3]`, a
  sequential read and a `reject`.

`kettle/Kettle.test.scala` keeps the Kettle cases of `samples/Kettle.test.scala` (the
enabledness table and the 99-to-100 boundary) and of `samples/Kettle.test.scala` (the four truth
tables), with the same cases and expected results, over states built from the initial state by
named updates.

## 2. Verdict table

| Item | Verdict | Evidence / what Scala forces / phase 2 |
| --- | --- | --- |
| State as `var`s on the Machine | **works** | The schema macro takes the Machine class's mutable declarations in declaration order: `State(temperature: Int, heatingOn: Boolean, beeping: Boolean)`. A facet's `var light: Lit` is abstract and the Machine defines it. A Machine with no vars is refused: `a Machine declares at least one state var`. |
| Effects as assignment blocks: `+=`, an `if` arm, `choose` with blocks | **works** | `temperature += 1`; `presses += 1` on `UpTo[3]`; `if light == Lit.on then light = Lit.off else light = Lit.on`. `choose(smashed -> { light = Lit.broken }, toggled -> { … })` gives one step per alternative, by replaying the step once per choice script. Scala forces `->` to be a **member** of `Choice` taking a by-name block. Predef's `ArrowAssoc` would run the block when the pair is built, and a member wins over the implicit conversion. |
| Guards and expectations read vars by name | **works** | A guard reads the live vars, restored to the step's start. A claim reads them restored to the state now. |
| `before` as the State record (`before.x`) | **works** | `transparent inline` macro returning `Record { val temperature: Int; … }`. A wrong field gives `value y is not a member of plain.Record{…}`, and a wrong type gives `Required: Boolean`. `before(x)` with a by-name argument would also be easy natively, but the spec chose `before.x`, so only that form is kept. |
| `before` outside `postconditions` refused | **works** | Compile error with the spec text, from the macro's owner chain. |
| `UpTo[140]` as a transparent alias of Int, bound read from the type tree | **works** | `+=` type-checks. The bound is present in the var's definition tree **and** in the symbol's info (`memberType` prints `UpTo[3]`), so a separately compiled Machine keeps it. Scala enforces no range: the native runtime checks it at every snapshot (`presses: 4 is out of UpTo[3]`). `UpTo[Int]` also satisfies `N <: Int`, so R9's literal-argument refusal is needed. Kettle's temperature is signed (−40..100) and stays `Int`, so the bound is measured on the lamp. |
| `scenario.starts(temperature = -40)` | **works** | `Dynamic.applyDynamicNamed`. Update names and value types are checked against the schema when the runtime uses the scenario (`temprature is not a state var of Model`). A compile-time check would need `applyDynamicNamed` to be a macro; that is possible but was not built. |
| Four properties as check blocks under `invariants`/`postconditions`, no kind word; `after(action):` | **works, with class reload** | Each `check` records its member (from the enclosing `val`), line and source text: `line 48: check(!beeping)`. `after(a)` runs its block only on a steps. Section members come from the section class's accessors; a `Unit` val keeps its accessor, though it has no field. See §2.4 for why sections are reloaded. |
| `boils`: `leadsTo(from =, to =, within =) in only(heat, failover)`, checked by verify | **works; spec values wrong** | The runtime gives verified with the longest path, missed deadline, deadlock or cycle. As written the claim is **vacuous** (§3.2). |
| `oneOf(-40 to 40)` start | **works natively; lift deferred (fn-190.9)** | The macro registers `temperature → -40..40` when the var initialises, giving 81 starts, 283 reachable states and 424 steps. The var's own value is the domain's head (−40). |
| No `queries`: every claim checked by default; `in only(...)` on the progress claim | **works** | `verifyAll()` checks every member from every start over the reachable graph. An `after(a)` member whose action never fires is reported vacuous. |
| Negative control as `assertRefuted` in the test | **works** | `assertRefuted(Model, invariant(check(!beeping)))`; the test needs `import Model.beeping`. |
| `object heater extends Internal` with `step`; `action` without `this` | **works** | Outside its container, the word resolves to a wildcard-imported top-level `inline def action = compiletime.error(…)`; inside, the inherited member wins. That gives the spec's exact text for an `action` outside an `Actor`, a `step` in an `Actor` and an `action` in an `Internal`. |
| Four transitions as `if g then effects.e()` | **works** | The `on` macro wraps each effect leaf in a "fired" mark. The native runtime needs this: an effect may leave the state unchanged, so the state alone cannot tell "fired" from "disabled". It refuses `a transition says when; write the state change in effects` and `a transition ends in an effect call, a rejection or (): … is none of these` at compile time. |
| One transition as a `match` on a phase var: `case a \| b if g`, `case _: Role`, `reject(r)`, `case _ => ()` | **works** | Lamp's `press`. A broken lamp answers `Rejected(Said.broken)`, and a worn-out one is disabled. A `match` missing a case fails the build under `-Werror` (`logs/non-exhaustive.status`). `compileErrors` cannot test that: exhaustiveness is checked after typer, so the probe compiles a separate file. |
| `object transitions:` without `extends Transitions`; facet order from the `extends` clause | **works: can become a criterion** | `on` is a `Facet` member. The macro refuses it anywhere but inside an object named `transitions`. Facet order follows the `extends` clause (`Faceted`: pressing, breaking; `Reordered`: breaking, pressing), with identical step tables. An action bound by two facets is refused, naming both. Natively the order comes from `getInterfaces`; the plugin will read the clause. |
| Sample as top-level definitions in `package samples.kettle`, Machine named `Kettle` | **works: can become a criterion** | Same schema, reachable graph, verdicts and progress answers as the nested form (`TopLevelKettleTest`). The IR name reads `Kettle.State` instead of `Kettle.Model.State`. |
| Outcomes/facts as member types with defaults, so the plain case is `extends Machine` | **fails as stated; alternative works** | A concrete type member cannot be overridden: `error overriding type Outcome in trait Base, which equals String; type Outcome, which equals Int has incompatible type`. Chosen alternative: **abstract** `type Outcome` and `type Fact`. A Machine that never rejects or records leaves them abstract (Kettle: `extends Machine`). One that does declares `type Outcome = Said`, and facets that both declare the same alias compile. "Accepted" is the framework's, not a type default. |
| Sequential-read substitution (`a = 1; b = a`) | **native: works; lift deferred (fn-190.9)** | `repair()`: `presses = 0; light = if presses == 0 then Lit.off else Lit.on` gives `light = off`. The lifter's substitution is fn-190.9's. |
| The bound survives the alias | **yes** | In the tree and in the symbol's info (above). |
| Synthesised record name and field order | **order: declaration order; name: phase 2** | Native record `State(temperature, heatingOn, beeping)` and `State(light, presses)`. The IR type name `<machine>.State` is set by fn-190.9; the top-level form gives `Kettle.State`. |
| A var read outside a lifted scope refused at the read | **not natively; needs the plugin** | A public var is ordinary Scala to every caller. A macro sees only the tree it is given, and the `on` macro polices only transition bodies (assignments, leaves). R4's read refusal needs the plugin's whole-unit traversal (fn-190.9). |
| An `if`/`match` chain lifts to the same arms as `when`/`where` (first match as nested `if`) | **native arms: yes; IR comparison deferred (fn-190.9)** | The `on` macro reads each body as first-match arms, each carrying the negations of the earlier ones, e.g. `!(light matches (Lit.off \| Lit.on) if presses.<(3)) && light matches (_: Terminated) => reject(Said.broken)`. Comparing with the retained IR needs the lift. |
| `before.x` lifts to the transition's first parameter, a bare name to the second; per-`check` diagnostics unchanged | **native semantics: yes; lift deferred (fn-190.9)** | Natively `before` reads the step's start record and bare names read the vars restored to now. Each `check` is reported on its own, with line and text. |

### 2.4 Why claims are reloaded, and the alternatives

`val beepsOnlyWhenOff = if beeping then check(!heatingOn)` has no call to hang a by-name on: it is
an `if` that runs when its object initialises. For `object invariants` nested in `object Model`,
Scala 3 puts that code in the class's static initialiser. In a scratch object of the same shape,
`object K { object M { object invariants { … } } }`, javap of `K$M$invariants$` shows `<init>`
calling only `Object.<init>`, with the `check` calls in `static {}`. A reflective `newInstance()` therefore runs nothing. Options:

1. **Rerun `<clinit>` in a child class loader** (chosen for phase 1). The section class is
   defined again from its bytes in a loader that delegates every other class to the parent, so it
   reads the same live Machine. Cost: about 80 µs per evaluation, 929 evaluations per Kettle
   `verifyAll`. It relies on Scala 3's codegen for static objects. Sections inside a facet trait
   are per-instance lazy objects whose body runs in the constructor, so they need constructor
   re-instantiation instead; the probe has none.
2. A context-function type on each member (`val x: Claim = …`): rejected, because it adds author
   noise.
3. A macro that splices the members' trees into closures: it needs the definitions' trees, which a
   macro has only in the same run or with `-Yretain-trees`. This is the fn-174 boundary again.
4. The plugin (phase 2) lifts the member's tree, and the IR evaluator (fn-190.3) evaluates it.
   This is the permanent answer, and it makes option 1 removable in fn-190.5.

## 3. Findings in the spec

1. **`neverBeeps` cannot stay in the Model.** API Contracts keep `val neverBeeps =
   check(!beeping)` in `invariants`, and R15 checks every claim by default. The default run
   therefore refutes it (`invariants.neverBeeps: Refuted(line 48 … State(temperature = 100,
   heatingOn = false, beeping = true))`), and the Model's default check is red. Recommendation:
   remove it from `invariants` and keep it only as `assertRefuted(Model,
   invariant(check(!beeping)))`, which the spec also states.
2. **`boils` is vacuous, and 81 is the wrong bound.** If the environment governs the whole run, as
   in TLA⁺, the starts are `heatingOn = false`, and only `pressButton`, which `only(heat, failover)`
   excludes, turns heating on. `from` never holds, so the claim is vacuous. If `from` states are
   instead taken over the whole graph, with only the later paths restricted, the longest path is
   **141** (from −40 °C, 140 heats and the failover), so `within = 81` misses. 81 is exactly the
   bound from 20 °C, the old `init`. Measured:
   - `only(heat, failover) & starts(temperature = 20, heatingOn = true)` gives `Verified(81)`.
   - `& starts(heatingOn = true)` gives `MissedDeadline(141, 81)`.

   Recommendation: keep the strict reading, which needs no new semantics, and write
   `leadsTo(from = heatingOn, to = beeping, within = 141) in (only(heater.heat, heater.failover) &
   starts(heatingOn = true))`. The owner decides whether `only` should instead restrict only the
   paths after `from`. Scenario composition (`&` here) also needs a spelling in R13.
3. **Outcome and fact types cannot be concrete type members with defaults** (§2). They are
   abstract members with no default.
4. **`scenario` cannot be a top-level `object` beside a `Scenario` class.** Scala warns that
   `plain.Scenario` and `plain.scenario` differ only in case and overwrite each other on a
   case-insensitive filesystem, which is the owner's macOS. Under `-Werror` that is an error.
   Keep `scenario` a member or a `val`, as production does today (`Machine.scenario`).

## 4. Recommended native runtime for fn-190.2

- **Schema, at compile time.** An inline macro at the runtime's entry (`Runtime(Model)`, or a
  `given` derived per Machine) reads the Machine object's mutable declarations in declaration
  order. For each var it emits the name, the declared type with aliases intact, the `UpTo` bound,
  a typed getter, a setter and an instance test. No reflection touches state, and `Machine` loses
  its state type parameter.
- **One live state, snapshotted and restored.** A snapshot is an immutable record of the values in
  declaration order. Restoring writes each var through its setter. The runtime captures the
  initial state the first time it reads a Machine, before any step. It restores before every step
  or evaluation and restores the initial state after every public operation, so the singleton is
  always back at init (tested). This rules out running two evaluations of one Machine at once.
  munit's sequential suites are fine; parallel suites on one Machine are not.
- **Steps.** Restore the start state, run the binding's body under a context (start record, action,
  choice script), and read the result. A fired leaf gives a step to a new snapshot, `reject` gives
  a rejection, and anything else means disabled. Each effect leaf gets its fired mark from the `on`
  macro. `choose` picks from the script, and the runtime replays the step once per unexplored
  choice. Bounds are checked at every snapshot.
- **`before` is the State record.** `before` is a `transparent inline` macro typed as the record
  refined with the enclosing Machine's vars. It reads the context's start record and is refused
  outside `postconditions`.
- **Claims read the state now through the vars.** Before evaluating a section, the runtime restores
  the state now into the vars, sets the context (start record, action), and reruns the section's
  static initialiser in a child loader. Each `check` records member, line, text and result. A
  member's first failing check is its diagnostic. `after(a)` runs only on a steps. Outside an
  evaluation, `check` does nothing.
- **Starts.** `oneOf` registers the var's domain as the var initialises. The starts are the product
  of the domains, and `scenario.starts(...)` overrides vars by name (checked against the schema).
- **Tests** build states as `native.state("temperature" -> 20, …)`, which is the initial state plus
  named updates, as the spec's evaluator will. Name members with a macro over the selection
  (`holds(Model.invariants.beepsOnlyWhenOff, …)`) rather than strings, which the probe used for
  brevity.
- **Removal.** fn-190.5 deletes the reload, the context and the setter closures. The schema
  macro's reading of names, types and bounds is what the plugin does in fn-190.9.

## 5. Not proved here

- Every lift item, sent to fn-190.9: `field`/`copy` IR, substitution, the `<machine>.State` IR
  name, `oneOf` into the IR start list, `before.x` to the first parameter, the IR comparison
  against the retained Kettle IR, and the read refusal.
- Inputs: `on(a): s => …` was not built. The `on` macro would unwrap the lambda and pass the input
  to the body.
- `records(fact)`, `crash`, monitors, refinement, `fair` (it is carried on the scenario but not
  used), and the cycle report that names the never-taken action.
- Claims inside a facet trait (§2.4) and the compile-time check of `scenario.starts` names.

## 6. Rerun

`bash "$PWD/model/design/plain-scala-prototype/run.sh"` from the repository root. Expected:
`compile: exit 0`, `test: exit 0, 31 passed, 0 failed`, `report: exit 0`, `non-exhaustive match:
refused (exit 1)`. Run it through `bash` with an absolute path. In the sandbox, `./run.sh` from
the probe directory reports "No such file or directory".
