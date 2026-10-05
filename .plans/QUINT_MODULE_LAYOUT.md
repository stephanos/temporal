# Quint's module shape for the Scala Models

Design study, 2026-10-05. Question: should Umpire's Scala DSL adopt Quint's module shape, so a
feature reads top to bottom in one place? Grounded in Quint's documentation and repository (sources
in section 1), `model/temporal/features/**` after fn-114.9's folder grouping, `model/irgen`,
`model/README.md`, and the specs fn-112, fn-114, fn-120, fn-122, fn-124 and fn-125.

**Owner decision (2026-10-05):** adopt option (a), a module object per machine, as layout only.
The spec that carries it out is "Read each feature top to bottom: one module object per machine"
in `.flow/specs/`. Line numbers below are from the day of the study and drift with later edits.

**Short answer:** adopt Quint's **one-module reading order**, not its syntax. The lifter already supports it; the cost is one IR/Case regeneration and golden-config entries. Details below.

## 1. Quint's structure (what is syntax, what is semantics)

Sources: [lang.md](https://github.com/quint-co/quint/blob/main/docs/content/docs/lang.md), [language-basics](https://quint.sh/docs/language-basics), [anatomy lesson](https://quint.sh/docs/lessons/anatomy), [ADR004 effects](https://github.com/quint-co/quint/blob/main/docs/content/docs/development-docs/architecture-decision-records/adr004-effect-system.md), [CHANGELOG](https://github.com/quint-co/quint/blob/main/CHANGELOG.md). Note: quint-lang.org now redirects to quint.sh; repo is `quint-co/quint`; v0.33.0 (2026-09-28); Quint spun out of Informal as its own company ([2026-04-16](https://quint.sh/posts/new_era)).

| Construct | Kind | What it does |
|---|---|---|
| `module M { … }`, `import M(N=3).*` | syntax (flattening clones the module) | one namespace holding everything; instances substitute `const` |
| `const`, `var`, `assume` | syntax + typing | parameters, state, constraints on consts |
| `pure def/val`, `def/val`, `action`, `temporal`, `run` | **semantics**: a six-mode effect system; each var primed at most once per step | mode errors are the main semantic check |
| `nondet x = oneOf(S)` / `any{}` / `all{}` | semantics | non-determinism only inside actions |
| `init`/`step` | convention (CLI flags) | not keywords |
| invariant = `val inv: bool`, `--invariant` | convention | safety |
| `temporal p = always(…)` | semantics, model checker only (Apalache/TLC; "not supported in the simulator") | liveness |
| `run t = init.then(A).expect(P).fail()` | syntax over actions (Run mode) | scenario tests |

A spec reads top to bottom: types/consts → vars → pure defs → actions → `step` → invariants → runs, all in ~60 lines (bank example). Pain points: instance ergonomics (#1572, #1715), no refinement support (#723), `nondet` scoping (#1046, #1820), temporal only via JVM checker, no recursion, big specs become one flat module ("7000-line states", [monadbft post](https://quint.sh/posts/monadbft)).

## 2. Umpire today (standalone activity)

15 files, 1,762 lines, 141 top-level declarations, 3 `DefinitionScope` pins; nexuscaller repeats it (9 files, 2,450 lines); `shared/taskqueue` borrows the activity's pin (`shared/taskqueue/Model.scala:24-25`).

| To understand `activityProduct`/`standaloneActivity` an author opens | Lines |
|---|---|
| `features/standaloneactivity/Model.scala` (enums, actions :56-84, `object Product` :111-181, machine :178-188, protocol, composition :401-405) | 405 |
| `shared/worker/Model.scala` (member machine) | 85 |
| `Properties.scala` (claims; `pause` has **no** authored claim: it comes from `Pausable`+`Pollable` law expansion in `temporal/capabilities/Pause.scala:11-24`) | 77 |
| `Queries.scala` (`object Paths`, Limits, `object Functional`) | 146 |
| `Capabilities.scala` (reaches into `Realization.scala:49` for `activityStatus`) | 56 |
| `IrFiles.scala` | 47 |
| + `Realization.scala`, `temporal/realize/Kit.scala`, `umpire/realize/Realize.scala` to reach a Case | 330+ |

Typical hop count: 5 files to Query, 9-11 to Case. Cross-file references are all **same-package top-level vals, so no `import` reveals them** (Properties→Model ≈42 symbols, Queries→Model ≈67, Realization→Model ≈70). Step-to-claim distance for `attemptResult(completed)`: step `Model.scala:270-284`, binding `:385`, Property `Properties.scala:12-14`, Scenario `Queries.scala:25-26`, Query `:101-102`, realization `Realization.scala:140,176,199`, Case `model/cases/activity-completion-case.json`. Referencing mechanisms: val names (machines, actions, Properties), `given Family` objects (`ActivityFamily`/`SystemFamily`, `Model.scala:21-27`), `given Catalog`, `DefinitionScope` pins, field selectors in `compose`.

## 3. Why it is scattered

| Split | Deliberate? | Reason (source) |
|---|---|---|
| Model vs Properties vs Queries | yes, fn-112 | Properties are "the specification a reviewer reads on its own"; monitors stay in Model because two *files* would init-cycle (`.flow/specs/fn-112:46-50`, `model/README.md:518-521`) |
| `admission/`, `compositions/` folders | yes, fn-112 | by subject first, else "three unrelated subjects in each file" (`fn-112:278-284`); subpackages so `Model$package$` names don't collide |
| Vocabulary objects `Product`/`Protocol` | yes, fn-112 | "objects let both machines say `attemptStart`" (`fn-112:278-280`) — already halfway to a module per machine |
| `Capabilities.scala` | content placement yes (fn-122.8), separate file stylistic (`.flow/tasks/fn-122…8.md:4-6`) |
| `IrFiles.scala` | the `irFile` value yes (fn-114:99-101), its own file unexplained |
| `Realization.scala` | never argued; predates fn-112 |
| Sugar vs core | framework-only rule (`fn-112:288-290`), irrelevant to feature files |
| Top-level vals | **not required**: lifter only needs a `val`; object members lift today (`Control.forgedCompletion`, nexuscaller/Model.scala:465-492; `Functional.completion`) |
| ID stability | Definition ID = owner + val name, or pin (`irgen/Context.scala:243-245`); one pin per owner, none nested (`umpire/DefinitionScope.scala`) |

Incidental: the per-kind file names themselves, the `Capabilities`/`IrFiles`/`Realization` files, and the fact that Properties are far from steps only because files, not semantics, separate them. The current folder rename already shows the price of a move: `model/ir/activity.json` now carries `temporal.features.standaloneactivity.Product$.*` function names while pinned IDs stay `temporal.standaloneactivity.Model$package$.*`.

## 4. Options

**(a) Quint-like module object per machine, feature file as the module list**

```scala
// features/standaloneactivity/StandaloneActivity.scala — types stay top-level (keeps pkg.Type IR names)
enum ProductPhase derives Finite: …
final case class ProductState(phase: ProductPhase) derives Finite
enum ProductFact derives Finite: …

object ActivityProduct:            // one pin keeps every action/monitor ID
  given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$")
  import ActivityFamily.given
  // -- actions (Quint: var/action signatures)
  val attemptStart = action(shared.worker.party).on(activity).schema[PollActivityTaskQueueResponse]
  val control = action(caller).on(activity).input(Inputs.control)…
  val timeout = timer
  // -- vocabulary + steps (Quint: pure def / action bodies)
  def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)
  def attemptStartStep(s: ProductState) = if s.phase != scheduled then disabled else accept(ProductState(started), statusStarted)
  def controlStep(s: ProductState, c: Control) = …
  // -- the machine (Quint: init/step)
  val machine = umpire.machine[ProductState, Outcome, ProductFact] {
    forEntity(activity); starts(ProductState(scheduled)); ends(s => terminal(s.phase))
    steps(attemptStart ~> attemptStartStep, control ~> controlStep, timeout ~> timeoutStep)
  }
  // -- what it promises (Quint: val inv / temporal)
  val terminalIsFinal = machine.property.once(s => terminal(s.phase)).keeps(_.phase)
  // -- capabilities, paths, queries (Quint: run … expect)
  val capabilities = umpire.capabilities(machine, limits = three)(Closable(…), Pausable(…), Pollable(…))
  val paused = machine.scenario.actions(control(Control.pause), control(Control.unpause), attemptStart)
  val pauseResume = (query find terminalIsFinal in paused limits three total 864).expect(satisfied)
```

Costs/risks: machine name changes from `activityProduct` to `ActivityProduct.machine` (name = val name; a `machine("activityProduct")` literal or a `val activityProduct` inside the object avoids it). Function symbols move → `function_name_substitutions` in `tools/umpire/internal/golden/config.json`; 1,161 source positions in `activity.json` and 21 Case files change → `umpire-gen-model`, `umpire-gen-fixtures`, `canary-gen-case`. Scala constraint: inside one object vals initialise textually, so a Property written above its machine compiles and is `null` at runtime (lifter unaffected, munit tests such as `StandaloneActivityPins.test.scala` and `Catalog.test.scala` affected) — the reading-order discipline Quint gives free must be a lint. Realization cannot be nested under a pinned module (nested pin refused; unpinned nesting changes its IDs), so `object ActivityRealization` stays a sibling. Compositions and IR files span machines, so a feature-level section (today's `compositions/`) remains. Family givens per object already work (`SystemFamily`).

**(b) Keep files, add an index/re-export module**

```scala
object StandaloneActivityIndex:
  export features.standaloneactivity.{activityProduct, activityProtocol, completes, Functional, activityFile}
```
Scala 3 `export` creates forwarder vals with a new owner: the lifter would see a second `val completes` (duplicate names are refused, README:495-503) unless taught to skip forwarders. Gains nothing in reading distance; the text stays in five files. Not recommended.

**(c) Specific Quint constructs only**

| Quint | Umpire today | Verdict |
|---|---|---|
| `nondet`/`oneOf` | `choose`/`choice` (fn-120, done; `Machine.scala:31-55`) | done |
| `run … expect` | `scenario.actions(...)` + `query find … .expect` (`Claims.scala:102-158`) | equivalent; a `run` alias would be sugar needing an `irgen/Syntax.scala` lowering |
| `const`/`assume` | fn-125 (deferred); `Assume.scala` | semantic work, deferred |
| `var`, `x' = e`, `init`/`step` | state case class + `accept(s.copy(…))`, `starts`/`steps` | already functional; priming would weaken Finite/TASTy lifting |
| `temporal`/`invariant` keywords | only safety + bounded progress (`SEMANTICS.md:123-128`) | `temporal` is a semantics change (fairness/liveness), not a keyword |

So (c) is either done or blocked on semantics; keyword renames alone would not reduce scattering.

**(d) Status quo + navigation**: a per-feature table of contents in the file headers and README; zero regeneration; does not change the 5-file hop.

## 5. Recommendation

Adopt **(a) in its per-feature form plus the (d) section discipline**, i.e. `StandaloneActivity.scala` holding, in reading order, types → `object ActivityProduct` → `object ActivityProtocol` → feature-level compositions/queries → `irFile` roots, with `Realization.scala` and the `admission/`/`compositions/` subject folders unchanged. Evidence: following the product machine from state to Query drops from 4 files / ~675 lines of context to 1 file section of ~120 lines; a step and its Property sit in the same object; the fn-112 init-cycle reason vanishes, and the "Properties read on their own" reason survives as a section (and as `properties`-only grep). Explicitly do not adopt Quint syntax: fn-120 ruled it out (`fn-120:133`), the export emits one `module umpire_slice` per IR file (`tools/umpire/export/quint.go:79,138`), not per machine, and nothing in it depends on the Scala layout.

Migration: 3 features + `shared/` ≈ 15 files merged into ~6, one golden-config batch, one IR/Case regeneration, README `:505-547` and `UMPIRE_MODULES.md:30` rewritten, one new lint (declaration order inside a module object). Make it a **new spec** (layout only, no semantics), sequenced: after fn-114.8/.9 are recorded done and fn-122.6 docs land (they describe the four-file rule you would replace); not concurrent with fn-124.8 (it forbids other edits to `model/`, `fn-124:23`); before fn-125 so settings land as `const`-like members of the new modules; fn-118.5 touches only `Realization.scala`, which this leaves alone.
## 6. As specified (fn-126, 2026-10-05)

`fn-126-read-each-feature-top-to-bottom-one` adopts (a) with these changes to the sketch above:

- **Actions stay in the feature's top-level signature section.** Product, Protocol, the composition and the realization all bind them. Inside one object, a step function named after its action would shadow it. Keeping actions at the top level also keeps every action ID under the existing file-level pin.
- **Machine `val`s keep their names** (`Product.activityProduct`), rather than `val machine` plus a name literal, which fn-114 removed. Derived Query names and law names therefore stay the same.
- **The existing vocabulary objects become the module objects**, so the function symbols of the status sets do not move.
- **The feature section is an object.** The module objects read the top-level signature, so top-level vals that read the module objects would re-create fn-112's init cycle. Capabilities that read the realization (`protocolCapabilities`) live in this object.
- **The lint covers more than order inside one object.** It also refuses initialization cycles between owners and misplaced declarations.

Later the same day, the owner widened fn-126 beyond layout. See `.plans/DSL_SIMPLIFICATION.md`, "Owner decisions":

- The machine object is the machine (`object ActivitySystem extends Machine[…]`), not a vocabulary object holding a `val` machine.
- Rules say when actions fire, and effects say what they do.
- Every kind of member sits in a section object that is transparent to Definition IDs.
- Actions are grouped by actor at the feature level.
- The feature section holds only the IR files.
- The levels are named Product and System.

**Landed.** The layout of option (a), in today's declaration forms:

- fn-126.1: the standalone activity, `record/` and `withTaskQueue/`, and the declaration-order lint;
- fn-126.2: the Nexus caller, its close policy, the Nexus operation, `shared/taskqueue` and
  `shared/worker`. No Model folder holds a per-kind file any more, the layout test keeps them
  retired, and the lint refuses a Model in a folder whose file is not named after it.

The owner's later decisions landed next:

- fn-126.3: actions grouped by actor, in actor and section objects transparent to Definition IDs;
- fn-126.4: the machine object is the machine (`object ActivityProduct extends Machine[…]`, `Derived`,
  `Composition`), with `init`, `end`, `states`, `refinement`, `effects`, `monitors`, `rules`,
  `properties`, `implements` and `queries`, rules that say when an action fires and effects what it
  does, and the section lint; the standalone activity converted;
- fn-126.5: every other Model converted (the Nexus caller and its close policy, whose nine designs are
  `Derived` objects with their own Queries, the Nexus operation, the task queue and the worker), the
  `FailureModel` and `NegativeControl` markers, and the `machine[S, O, F] { … }` builder, `steps`,
  `starts`/`ends` and the `compose(…)` value form retired, so the IR generator reads one shape.

The Product and System renames and folders per level are fn-126's last task.
