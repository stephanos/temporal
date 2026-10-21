# ModelSuite holds each Model's examples, explorations and conformance tests

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


> Local HTML render lens: `.flow/artifacts/fn-195-modelsuite-a-models-test-file-holds-its/spec.html`. Open locally; regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (2026-10-17): "the test files for a model serve two purposes: (1) test the model (example based or sampled based or property based) but we also (2) need to find a deisgn where they are translated (or only the ones we can actually translate) into conformance tests. meaning we move that repsonsibility out of the mdoel and into the test file. and the Cases are then only created from the test spec."
> user: "Quint puts those into its tests. I think we should od the same" (on `Query`, `limit`, `king, donkeykong, peaches, mario`)
> user: "I really like the `expect:` block style"
> user: "do we seprarate runs that are a single example from runs that are more like queries that explore a state?"
> user: "are we covering and clearly separting thiese: a. testing the mmodel itself as a example based test b. testing the model itself with properties c. testing the model itself with sampling/simulation d. testing the real system with examples e. testing the real system with queries or other ways to describe the state space to explore. and are c and e the same just in different contexts?"
> user: "why wouldn't we follow the test("..."): notation?"
> user: "how about using an annotation like `@...` to clearly denotate the differnet test types that aren't unit tests? or follow the same idea as the model by using sections like `object ...:`"
> user: "> client.send(Addr.king, Addr.donkeykong, Denom.banana, 1) why is that not using Send? and named params? it's hard to read what it does"
> owner decisions (AskUserQuestion, 2026-10-17): capability expectation bindings move into the test file; a run names its start when init is a set, default otherwise; `oneOf` is the only variation, `Exploration`'s `runs`/`edits`/`dropPrefix` retire; one file, fallback a separate declarations file; bare Booleans in `expect:` only; `find(steps = n)` is the horizon, part of the question; a machine with one realization binds every conformance test to it.

> user (2026-10-18): "let's file a task for standardize primary machine declarations on class Model".
> user (2026-10-18): "we should also remove support for object Model entirely for consitency".

> user (2026-10-18): "once the DSL has settled after the last task, add antoher task after to go over each umpire frameowkr sample and simpify it (wihoutht losing behavior); each one must be the best version of itself".

## Goal & Context

<!-- Goal & Context: 60% [user], 40% [paraphrase] -->

A Model's test file does two jobs. It tests the Model, by example, by exhaustive claims and by
sampling. It also declares the conformance tests that run against the real server, and those are the
only source of Cases. Today the Model carries the questions asked of it (`object queries`, `find`,
`verify`, `witness`, `limits`, `.live`, `.explore`, capability expectation bindings), the sample tests
check predicates on hand-built states, and nothing resembles Quint's `sendTest`, an init-then-send-
then-expect script. fn-190 R14/R15 already retired `limits` and reduced `queries` to witnesses and
pinned paths; this spec takes the last step and moves every question into the test file.

The five kinds of test the owner named, and where each lives:

| Form | Of the Model | Of the system |
| --- | --- | --- |
| Authored path | a. `examples` | d. `conformance` |
| Claims over a space | b. the Model's claim sections, verified by default (fn-190 R15); `explorations` for reachability | (exhaustive is impossible against a server) |
| Generated paths over a described space | c. `explorations`, `sample` | e. `conformance`, `sample` |

c and e are one activity in two contexts, as a and d are: the same seeded paths, replayed on the
Model and performed on the server, with the Model replay as each Case's oracle.

Execution is in-process on the Scala checker (fn-196). A section's tests are munit tests: a
failing `expect:` line fails the test. Phase 2, DSL completion; the lowering changes are bounded.

## Native checkpoint admission (2026-10-18)

The conductor verified each following task as done through Flow and both listed commits as
ancestors of integrated umpire HEAD `71b076b200e6727a2f1a3c4dfac6ec49e5e4a8bf`.
These native checkpoints replace the two whole-spec scheduling prerequisites. They do not
close either parent spec or earn their deferred batch review or held integration proofs.
The original fn-194.2/fn-196.2 API prerequisites remain included.

| Native task | Final source/evidence checkpoint | Committed closure |
| --- | --- | --- |
| fn-194-parametric-models-constants-and-finite.1 | `9648caeb9d9a74de4ca3de4be31d2b82ca7fc223` | `f74ab768fe3b58b1e6f5a116e48be3b3284922af` |
| fn-194-parametric-models-constants-and-finite.2 | `b33e7e36b440c9f54a4a047f8488eb596b25520d` | `6e345185f8ec9ba426707688db766008a4784770` |
| fn-194-parametric-models-constants-and-finite.3 | `bb8749d7cf56f76ece2f1cbbd583ee47cc791501` | `31e294207dd7d13716ba74c0ca2b46cb2a8fe030` |
| fn-194-parametric-models-constants-and-finite.4 | `f431b252da886d9bec88b8d7270cb66a944c3169` | `f8d5d0f39aa236a8986a3be251c15001e2e4888d` |
| fn-194-parametric-models-constants-and-finite.5 | `b53e677090f7b80c25010d196327b79d716ba62a` | `df9e05fbe0d4f1bdaf28dd92863b3c29fe8f20fb` |
| fn-196-the-scala-checker-is-the-oracle.1 | `0dd5737301eaf2193d3ae89ea71a3dac2110c7ae` | `58fa14a2056cc50b9d120c0ce1185ad6e757f5e5` |
| fn-196-the-scala-checker-is-the-oracle.2 | `8f84a96e4942fc4fb028bfd72e87ac78e379d329` | `a0cc2feb8432a4a3aa67b1d620becc9ccc78ea6f` |
| fn-196-the-scala-checker-is-the-oracle.3 | `2319050f0f6f82adee2dc1b94b98d3f301f3fdc2` | `946b550a6d7a5f25fd84ca71e246a0b6ff890b4a` |
| fn-196-the-scala-checker-is-the-oracle.5 | `8c2dada5fd7960be5db26ea0904d0a26306969e0` | `71b076b200e6727a2f1a3c4dfac6ec49e5e4a8bf` |
| fn-196-the-scala-checker-is-the-oracle.6 | `89df25410b813ffcac67c553ff3e7a4e0106ba3b` | `c449a15a0b4d91bc4af237ba4a7dfc75feaaa4c9` |
| fn-196-the-scala-checker-is-the-oracle.7 | `33b84f57806f4cd1e6b7132831b8d48f007a894d` | `115a1bcf47a9697780fe670dc73bb8d909b19b7c` |

The old fn-196.4 retirement route was subsequently closed by replacement in the one-language roadmap; its historical no-go and unmigrated runtime obligations are not completed implementation. None is released by this native conformance plan.
Go/IR/export/lifting/schema/backend/generated/regeneration and all Temporal work remain held;
no publishing or review verdict is authorized. Spec admission is local committed-checkpoint
admission, not a forced task start or remote-branch fallback. The public implementation task chain is refined below; the downstream native holds remain
unchanged. Final fn-195.7 remains blocked until fn-195.5 and the later fn-197 native checkpoints
are committed; its baseline must be taken after those DSL changes.

## Architecture & Data Models

### Native exploration checkpoints

Native exploration implementation consumes the committed authored-capture and partial
suite-lifecycle/runtime checkpoints independently of complete global suite discovery. Three
bounded leaves deliver immutable spaces/selectors, find/sample adapters and deferred refutation
with local refusal controls. Their core write surfaces overlap, so their implementation order is
serial; fixture preparation and independent review can run concurrently.

The original exploration task retains final actual Registry-selected source/library suite
integration and depends on completed suite discovery plus all three leaves. Direct native suite
selectors do not close that obligation. Full suite lifecycle/discovery, both actual gate callers,
complete held/unselected identities, corpus admission and the stronger checker policy-ordering
contract remain outstanding with their existing owners. Conformance integration stays blocked
on its actual complete inventory and capability production prerequisites. A retained sample
declaration and seeded paths supply reusable ingredients only.

Supplied exploration selector constraints evaluate once at deferred space acquisition in authored
order and remain immutable across questions. Unspecified positions retain existing search
semantics; declared positions stay within the space bound. Questions retain the test-level
from/single-start rule. This adds no per-reached-state selector reevaluation or second start API.

<!-- Architecture: 50% [user], 40% [paraphrase], 10% [inferred] -->

- **`ModelSuite(instance)`** is a munit suite over one Model instance (fn-194). It defines
  `test(name, from = <start>, expectation = <RunExpectation>, through = <realization>)` returning a
  builder whose block body is the test, shadowing munit's `test`, and four section objects the suite
  registers: `examples`, `explorations`, `conformance` and `unit`. A `unit` body is plain munit, for tests of pure helpers. A suite's top level holds only its sections and shared helper definitions; a `test` outside a section is refused, the rule the Model's lint applies to a machine object.
- **Primary machines** use `class Model`, including parameterless declarations. Each ModelSuite binds a fresh stable-named instance. Singleton primary Machine declarations are no longer supported, regardless of object name. Active native fixtures migrate with the samples. Objects remain available for namespaces, constants, actors and shared declarations. The six-sample migration is followed by fn-195.6, before the final documentation task.
- **A body in `examples` or `conformance`** is a sequence of statements: an action call by its
  record's named constructor (`client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom =
  Denom.banana, amount = 1)`), a whole action with no fields (`heater.heat`), `oneOf(step, step)` for
  a finite family, and `expect:` blocks whose every statement is one Boolean assertion on the state
  after the last step, with `outcome`, `before` and `records(fact)` in scope as in a postcondition.
- **A body in `explorations`** is a `find`, a `sample` or a `refuted` over a space. A space is a
  `val anySends = explore(steps = 6): <steps with any>` in the same section. `any` at an input or an
  unstated step is a search; `space.find(steps = n): expect: ...` stops at the first path reaching
  the target; `space.sample(count, seed)` replays that many seeded paths and judges them by the
  Model's claims; `refuted(derivedMachine, claim)` is the negative control (replaces
  `assertRefuted`).
- **`conformance`** holds the Cases. Each test is also replayed on the Model as its self-test. The
  expectation defaults to `satisfied` for an authored path and `conformant` for a `sample`; others
  are written. `through` is required only when the instance has several realizations. A capability
  expectation is a conformance test over the capability's generated path.
- **Rules.** `oneOf` enumerates, `any` searches, only `explorations` may use `any`. Case names are
  `<instance>.<slug(test name)>`, a sample's Cases `<instance>.<slug>.<seed>.<index>`.
- **The Model loses** `object queries`, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`,
  `.live`, `.explore`, `.records`/`.ends`, and the capability binding's live switch.
  `Realizes(machine, runs = ...)` loses `runs`. `satisfied`, `inconclusive(reason, because)` and
  `conformant` are constructors in the realize package.
- **Lifting (later phase).** *(2026-10-18: superseded; fn-203's Scala producer reads the `conformance` declarations directly, `expect:` trees become Contract assertions in fn-203, or oracle-sidecar questions on an fn-209 go; there is no exporter step.)* The exporter lifts the three declaration sections, excluding `unit`, of a `ModelSuite` subclass from a
  test file and nothing else in it; the `expect:` trees become Contract assertions at their
  positions. Fallback if lifting from a test file is awkward: `X.runs.scala` beside `X.test.scala`.

- **Ownership and execution.** Suite declarations and finite carriers belong to the DSL/realization
  module. They have no checker implementation types in public signatures. The existing lazy native
  runtime interface delegates execution to one checker adapter over the existing replay, exploration
  and sampling engines. MUnit uses its existing pinned version as an ordinary dependency for the
  public suite. Declaration compilation and loading remain possible without the checker.
- **Suite policy.** The framework discovers concrete suites, preserves stable instance identity and
  resolves one immutable execution budget for the exact concrete suite and passes it to every
  question, including automatic claim checking.
  Horizons and sample counts remain separate bounds. Cuts retain native unknown evidence and a
  visible inconclusive/failure disposition; they never count as verified claims.
- **Native conformance admission.** Automatic realization selection requires a complete finite
  candidate universe for the exact instance. Safe initialization, repeated acquisition and distinct
  late candidates have deterministic semantics. An initialized-so-far list does not establish
  uniqueness. Native performed/waited-class and capability-path projections precede retirement
  of their current declarations. Candidate discovery stays owner-local and collectible.
  An explicitly supplied, fully constructed realization instead binds by exact Model reference
  after checked coverage, full expectation/monitor admission and successful native replay. This
  explicit branch asserts no inventory membership or cardinality; other candidates cannot revoke
  it. Omitted `through` still requires the complete universe and refuses while discovery is incomplete.

## API Contracts

```scala
class BankTest extends ModelSuite(Bank.Model(holder = Addr.king, supply = 2)):

  object examples:
    test("the king sends bananas, donkeykong cannot send burgers"):
      client.send(fromAddr = Addr.king, toAddr = Addr.donkeykong, denom = Denom.banana, amount = 1)
      expect:
        balances(Addr.king, Denom.banana) == 1
        balances(Addr.donkeykong, Denom.banana) == 1
      client.send(fromAddr = Addr.donkeykong, toAddr = Addr.peaches, denom = Denom.burger, amount = 1)
      expect:
        outcome == Outcome.invalidCoinsOrInsufficientFunds
        balances == before.balances

  object explorations:
    val anySends = explore(steps = 6):
      client.send
    test("an overflow is reachable within four sends"):
      anySends.find(steps = 4):
        expect:
          outcome == Outcome.overflow
    test("sampled sends hold the claims"):
      anySends.sample(200, seed = 1)
    test("a leaky bank is refuted on the supply invariant"):
      refuted(BankControls.Leaky, BankControls.Leaky.invariants.totalSupplyInv)

  object conformance:
    test("peaches gets a banana"):
      client.send(fromAddr = Addr.king, toAddr = Addr.peaches, denom = Denom.banana, amount = 1)
      expect:
        balances(Addr.peaches, Denom.banana) == 1
    test("sampled sends conform"):
      explorations.anySends.sample(200, seed = 1)

  object unit:
    test("Balances.set drops a zero amount"):
      assertEquals(Balances(Map.empty).set(Addr.king, Denom.banana, 0), Balances(Map.empty))
```

Kettle names its start: `test("boils from room temperature", from = room):`. A conformance test with
a non-satisfied expectation: `test("retry completes", expectation = inconclusive(Reason.explanationsDisagree, because = "..."))`.

Errors, each a compile error or a located lint refusal at the test: a `test` outside the four sections; a statement in `examples` or
`conformance` outside the liftable subset; `any` outside `explorations`; a `find` or `sample` in
`examples`; a test without `from` on an instance whose init is a set; a conformance test whose steps
include a class no realization performs or waits for; `through` naming another Model's realization,
or omitted with several or an incomplete candidate universe; an `expect:` statement that is not Boolean; an `expect:` block with no
statements; a test name that slugs to an existing name in the same suite; `oneOf` with one
alternative; a `sample` count of zero.

## Edge Cases & Constraints

- The expectation's `because` rule stays as in fn-190: non-satisfied outcomes require text,
  satisfied ones refuse it.
- A conformance test replays on the Model first; a Model replay failure is reported as that test's
  failure and the Case is not generated.
- Repeated steps use `repeat(80)(heater.heat)`, the spelling established by the committed prototype.
  Repetition preserves ordinary action admission, execution order and per-step context.
- The sections are objects nested in a class. Registration runs after concrete suite construction
  and before MUnit snapshots tests, in fixed section order on the same receiver. Eager concrete
  helper fields initialize first. Repeated enumeration does not register duplicates; registration
  re-entry and partial initialization failures refuse consistently. Both String and TestOptions entry points use the
  same placement, naming and duplicate admission.
- Bare Booleans are accepted in `expect:` only; claim sections keep `check(...)`.

- Strict `expect` admission uses typed capture without author warning annotations or weaker
  compiler flags. Each condition retains its own source location, evaluation count and order.
  Ordinary discarded Booleans outside the block and unrelated warnings inside it remain errors.
  The proved source route compiles the macro before its legal test-source caller, or consumes a
  separately packaged macro library. Joint main-source macro/caller compilation is unproved.
  The probe admits a flat sequence and refuses nested blocks, local definitions and direct
  control-flow terms. Those are experimental bounds, not additional settled public restrictions.
  Deferred replay, the full Boolean-expression grammar and typed instance context still require
  production proof. No capture may silently discard a condition.

## Acceptance Criteria

- **R1:** `ModelSuite(instance)` registers munit tests from its `examples`, `explorations`,
  `conformance` and `unit` sections; `unit` bodies are plain munit; the top level holds only sections
  and shared helper definitions. Errors: a `test` outside a section, and a section with a non-test
  member other than a space `val`, are refused naming the member. [user]
- **R2:** An `examples` body of named action calls, whole actions, `oneOf` families and `expect:`
  blocks is replayed on the instance by the Scala checker from `from` or the single start; each
  `expect:` statement is one assertion whose failure names the line and the state. Errors: the
  liftable-subset and `from` refusals in API Contracts. [user]
- **R3:** `explorations` supports `explore(steps)` spaces with `any`, `space.find(steps)` with an
  `expect:` target, `space.sample(count, seed)` judged by the instance's claims, and
  `refuted(machine, claim)`; `assertRefuted` and `NativeControls` are retired. Errors: `any` outside
  `explorations`, zero `sample` count, `find` with no target. [user]
- **R4:** `conformance` tests are the only declarations the Case generator reads; each is replayed
  on the instance first; the expectation defaults to `satisfied` for a path and `conformant` for a
  sample; `through` binds by the decision in Architecture. Errors: the conformance refusals in API
  Contracts. [user]
- **R5:** The Model has no `queries` section, `Query`, `find`, `verify`, `witness`, `scenario`,
  `limits`, `.live`, `.explore` or capability live switch; `Realizes` has no `runs`; `satisfied`,
  `inconclusive` and `conformant` live in the realize package. Errors: the retired names are
  unreachable by authors (fn-190.16's grep). [paraphrase]
- **R6:** The six samples' tests are rewritten as `ModelSuite`s with Quint's runs as examples
  (Bank's `sendTest`, TwoPhaseCommit's `successTest`), their reachability finds as explorations,
  their negative controls as `refuted`, and pass on the Scala checker with intended answers
  unchanged. Errors: a changed answer is a finding to fix. [paraphrase]
- **R7:** The lint applies the section rules in API Contracts, each refusal located at the test.
  Errors: as listed. Source admission must examine a suite with no test calls too.
- **R8:** SEMANTICS gains a section for the test file (the five-kind table, the sections, the rules),
  §9 Scenarios and Queries is rewritten to say the Model asks nothing of itself, and the README's
  test guidance follows. Errors: none beyond review.

- **R9:** All six primary native sample machine declarations use `class Model`, including parameterless models; suites bind fresh stable-named instances with independent state and exact ownership. Singleton primary Machine declarations are refused regardless of object name, and active native fixtures/clients migrate without a compatibility escape hatch. Existing behavior, verdicts and realization bindings remain unchanged. Objects for namespaces, constants, actors and shared declarations remain; held Temporal/IR bodies are not migrated here. [user]

- **R10:** After the DSL, ModelSuite, class-only declarations and their docs settle, review and simplify every native framework sample into its clearest idiomatic form without changing behavior, claims, bounds, counterexamples or realization bindings. Preserve test purposes and verify before/after evidence; exact upstream comparisons and held consumers remain unchanged. [user]

## Early proof point

The committed prototype proves post-construction registration, replay adapters and the repeat
spelling. The strict compiler experiment proves legal test-source and library warning behavior.
The authored-capture checkpoint must prove deferred typed replay before the public suite consumes it.
The committed conformance proof records a failed complete-inventory oracle and a failed executable-capability oracle in both source and library arrangements. Conformance integration remains blocked until bounded follow-on feasibility selects complete mechanisms and their production implementations integrate. The independent native wait and expectation leaves can proceed without claiming that integration.
The authored-explicit prerequisites separately prove exact admission, coverage during complete
finite replay and public eligible-only selection. Failure in one keeps that prerequisite open;
success does not establish automatic discovery, capability paths, sampled conformance or Registry integration.

## Quick commands

```sh
mise exec -- scala-cli test --server=false model/framework/checker --require-tests --test-only checkercontrol.CheckerTest
```

Run the native-only focused recipes for each task's named suites, then the changed-scope framework
gate against its recorded source baseline. The task receipts record exact commands, selected suite
names, positive executed counts and refusals. All commands exclude held Temporal and export roots.

## Boundaries

- *(2026-10-18: the IR phase here is fn-203, Cases produced in Scala; fn-195.11's producer-facing native selector is fn-203's input.)* Lifting the sections from test files, Contract assertions at intermediate `expect:` positions, and
  Case generation from a `sample` are the IR phase; this spec ends at declarations that replay on the
  Scala checker. [paraphrase]
- The realization (commands, evidence, waits, roles, cleanup) does not change. [paraphrase]
- No power-assert rendering beyond naming the failing line and state; a diagram is a later item.
  [inferred]
- The Temporal feature Models' `queries` sections are re-homed as `ModelSuite`s when those Models
  return to the tree, by recipe, not here. [inferred]

## Decision Context

Planning confirmed the capture's R7/R8 inferences against the existing public refusal contract and
framework documentation responsibilities. Their meanings and downstream owners remain unchanged;
they are source-grounded planning requirements, rather than additional owner quotations.

Sections over a per-test `.live` switch or an annotation: an annotation attaches to a definition,
not a statement, so it would force the per-definition marker the owner rejected for `@lifted`;
sections are the Model's own idiom and let lint judge a container, and the Case rule becomes one
greppable line. [user] `test("..."):` over `val x = run:`: the owner asked why not munit's notation;
the only thing a `val` bought was a reference, and the realization no longer names runs; spaces stay
`val`s because other tests reference them. [user] `expect:` as an indented block of bare Booleans over
`.expect(x == y)` or infix chains: Scala has no bare `assert x == y`, and the owner chose the block.
[user] Named record constructors in action calls: the owner found the positional form unreadable, and
the named call is the DSL's existing checked form. [user] Examples and queries separate: an example
is a fixed path with a stable identity, a query's path is a byproduct of search and must not become
a Case by accident; a found witness is frozen into an example by hand. [paraphrase] `sample` in both
`explorations` and `conformance` because c and e are one activity. [paraphrase] The seven owner
decisions in Conversation Evidence stand as written. [user] A fourth section `unit` for plain tests rather than top-level `test`s: the owner asked for consistency; with `test` shadowed, location would otherwise carry the lifted-or-plain difference, and the Model's lint already refuses top-level declarations in a machine object. [user] Rejected: lifting munit bodies in general
(too large a liftable subset); a separate `Runs` object (an extra container the sections make
unnecessary); keeping `.live` (a per-test switch a reviewer can miss). [inferred]

### Suite discovery prerequisite (2026-10-18)

The lifecycle/native execution checkpoint passes its admitted checks but omits valid runnable
suites whose superclass uses a stable-term path. Keep S incomplete until bounded compiler
metadata supplies the actual eligible suite identities. Preserve ordinary Scala authoring,
exact source/module ownership and no-test suite admission. Source parser special cases and
initialized-so-far runtime discovery do not establish this inventory.

Task fn-195.12 proves the metadata route and its held-suite universe before production work.
The chosen direction is explicit staged preflight. Syntax, duplicate keys and positive Int
ranges refuse before child commands; exact unknown-suite validation follows admitted inert
metadata compilation and precedes suite construction or execution. The proof must reconcile
the existing stronger policy ordering and recognize held/unselected identities without
compiling owner-held inputs. Production integration requires an explicit contract amendment
after the proof, plus honestly sized producer and gate-caller prerequisites before S resumes.
No historical receipt or current production behavior is relabeled by this decision.

The bounded proof establishes classfile/typed joins and artifact-local ancestor acquisition with
the helper's original source absent. It also retains a counterexample where a nonempty typed
ancestor list omits a required helper. Task fn-195.13 implements the inert metadata leaf with
independent superclass/interface closure and artifact ownership checks. Task fn-195.14 remains
explicitly blocked and conditionally sized until the complete held-suite metadata route is
admitted and both actual caller integrations can be split honestly. The direct policy sequence
does not prove either caller. S waits for both production prerequisites; final source rules and
exact-instance realization admission keep their existing owners.

Rejected source term/type parsing because it duplicates Scala compiler semantics. Rejected
outer-bootstrap compilation because it hides the same ordering change and adds another
artifact lifecycle. Reuse the typed foundation needed by final suite admission; realization
candidate completeness remains a separate proof. All current owner holds remain unchanged.

### Native conformance prerequisites (2026-10-18)

The empirical proof passes 28 behavioral controls in each source/library arrangement. Its two independent semantic oracles still fail in each arrangement. Acquired-so-far realization values cannot establish declaration completeness, and native capability claims discard their executable predicates and effective directives. Keep those failed requirements visible when closing the investigation.

Schedule two bounded empirical follow-ons before sizing their production implementations. The realization follow-on must establish a finite declaration-owner universe, exact runtime receiver bindings, safe getter acquisition and closure before automatic selection. External owners and untouched lazy declarations are the first counterexamples to test. The capability follow-on must retain ordinary binding/law identity, executable companion predicates, effective waiver/override decisions and selected bounds through genuine generated native replay. Either follow-on may complete with a minimal counterexample and an explicit remaining blocker. Neither report closure supplies production conformance support.

Two independent native implementation leaves are ready. A kit-owned waited-class projection and immutable checked coverage preserve exact action/input identity and explicit unsupported-kit refusal. Expectation conveniences reuse the existing value hierarchy and one shared conformance admission validator, preserving raw constructor/copy compatibility. Their focused tests do not replace actual conformance registration, complete realization selection, P/X path coverage, monitor identity checks or model-first eligibility under S.

The future path default is fully satisfied. The sample default expects conformant conformance, an inconclusive singular Property with `neverEvaluated`, a satisfied Contract, completed disposition, succeeded cleanup, no monitor assertions and the explanation `A sampled path selects no individual Property`. The explicit Property-only inconclusive shorthand accepts `explanationsDisagree`, `neverEvaluated` and `unreadable`; other combinations require a full independent expectation. At conformance admission, the shared reason is absent exactly when both Property and Contract are satisfied. Explanation text is empty in that case and otherwise nonblank. Conformance and each monitor retain their own independent reason-presence rules. C resolves exact monitor ownership and duplicate assertions.

Authors keep the existing ordinary declarations. No candidate lists, registration calls, relocation rules, arbitrary classpath scanning or process-global strong registry replace automatic selection. Actual system-kit adoption and all current held integration surfaces stay held. The final documentation task reconciles the independent leaves with the completed suite interfaces.

Prospective explicit-selection decision (2026-10-18): separate exact-reference admission from
automatic complete-universe selection. Earlier feasibility reports investigated the stronger
inventory-membership rule even for explicit selection; their failures remain historical evidence,
not passing production proof. The explicit branch validates the supplied constructed realization
against the exact Model without enumerating unrelated owners. It preserves full expectation values,
canonical exact-monitor membership and checked performed/waited coverage. Deferred action inputs
are resolved only in restored replay state, never in a coverage prewalk. Eligibility requires
complete verified replay of the entire finite authored family: a verified prefix, disabled step,
failed assertion/effect, hole or cut grants none. A new execution attempt invalidates prior
publication until successful completion; idempotent MUnit enumeration does not rerun inputs or
revoke a valid declaration. Eligibility is an opaque successful-result authority, not a copyable
metadata flag. Declaration loading remains checker-independent and owner-local results remain collectible.

The three concrete production prerequisites implement only this authored-explicit branch.
Full native conformance remains blocked on complete automatic selection, generated capability
paths with effective directives/bounds, sampled conformance and actual Registry integration.
No source-registration protocol, root-owner restriction or new author annotation is introduced.

2026-10-18, one-language series: the later lifting phase is fn-203 (Cases are produced in Scala), which reads the `conformance` sections and `expect:` trees natively, with no exporter step; fn-195.11 is fn-203's input and fn-203 depends on this spec. If fn-209 (native checker binary) says go, `expect:` assertions become oracle-sidecar questions instead of Contract clauses.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Parked unknowns

- Whether claim sections also accept bare Booleans; resolved after living with `expect:`.

## Public implementation checkpoints

| Task | Mechanism | Admission |
| --- | --- | --- |
| fn-195.2 | Authored action/expect capture, exact-instance context and native replay adaptation | After committed .1 |
| fn-195.12 | Bounded compiled suite inventory and staged-policy proof | After .2; bounded proof checkpoint committed, full replacement remains unadmitted |
| fn-195.13 | In-process compiled inventory, exact artifact/source/ancestor identity and inertness | After .12; M-sized producer only, no caller orchestration |
| fn-195.14 | Complete universe and actual Registry/FrameworkLoop/ChangeCheck staged admission | Explicitly blocked after .13; complete held metadata admission and honest implementation decomposition required |
| fn-195.8 | Four-section lifecycle, real discovery, stable identity and immutable suite policy | Blocked on .12/.13/.14; committed lifecycle checkpoint is incomplete |
| fn-195.9 | Reusable exploration spaces and find/sample/refuted adaptation | After .8 |
| fn-195.10 | Bounded native realization/wait/capability proof and expectation contract | After .1; empirical report committed with completeness/capability counterexamples, R4 implementation remains incomplete |
| fn-195.15 | Finite declaration-owner universe, exact runtime receivers and safe complete selection feasibility | After committed .10; bounded empirical report, production mechanism remains unadmitted |
| fn-195.16 | Ordinary native capability retention, executable expansion and effective directives/bounds feasibility | After committed .10; bounded empirical report, production mechanism remains unadmitted |
| fn-195.17 | Framework waited-class projection and immutable performed/waited coverage | After committed .10; generic protocol and native fixture kits only, actual system-kit adoption held |
| fn-195.18 | Realize expectation conveniences and shared conformance admission validation | After committed .10; raw constructor/copy compatibility preserved |
| fn-195.22 | Exact explicit realization, full expectation and canonical monitor admission | After committed .2/.17/.18; no inventory, replay or eligibility credit |
| fn-195.23 | Actual selected-class replay coverage and opaque complete-verified eligibility | After committed .22 and .21; existing native adapter, no prewalk |
| fn-195.24 | Typed public authored-explicit conformance and eligible-only native selector | After committed .23; source/library and checker-independent declaration controls |
| fn-195.11 | Typed native conformance declarations, complete binding and model-first replay | Explicitly blocked on proved and integrated realization/capability mechanisms, checked-coverage and expectation leaves; also after .9 |
| fn-195.3 | Author vocabulary retirement and final typed suite source admission | After .11; refine retained L scope before implementation |
| fn-195.4, .6, .5 | Six-sample migration, class-only primary declarations, final docs | Existing dependency order preserved |
| fn-195.7 | Final per-sample simplification with fresh behavior baselines | Existing .5 and fn-197 native admission hold preserved |

Each newly admitted implementation checkpoint is bounded to one mechanism group. The conformance
integration remains a conditional shell until the two feasibility follow-ons select concrete production prerequisites and those implementations integrate.
The existing L-sized retirement, class-only and final-polish tasks require refinement at their own
admission boundaries. No task size is reduced by dropping behavior.

The proof and authored work have disjoint write surfaces. The conductor coordinates compiler/build
access and integrates each source checkpoint and Flow closure before dependent admission. All
native source/library and refusal proofs remain task-local. The final source-admission task also
checks suites without test invocations.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `ModelSuite(instance)` registers munit tests from its `examples`, `explorations`, `conformance` and `unit` sections; `unit` bodies are plain munit; the top level holds only sections and shared helper definitions. Errors: a `test` outside a section, and a section with a non-test member other than a space `val`, are refused naming the member. | fn-195-modelsuite-a-models-test-file-holds-its.1, fn-195-modelsuite-a-models-test-file-holds-its.12, fn-195-modelsuite-a-models-test-file-holds-its.13, fn-195-modelsuite-a-models-test-file-holds-its.14, fn-195-modelsuite-a-models-test-file-holds-its.8 | — |
| R2 | An `examples` body of named action calls, whole actions, `oneOf` families and `expect:` blocks is replayed on the instance by the Scala checker from `from` or the single start; each `expect:` statement is one assertion whose failure names the line and the state. Errors: the liftable-subset and `from` refusals in API Contracts. | fn-195-modelsuite-a-models-test-file-holds-its.1, fn-195-modelsuite-a-models-test-file-holds-its.14, fn-195-modelsuite-a-models-test-file-holds-its.2, fn-195-modelsuite-a-models-test-file-holds-its.8 | — |
| R3 | `explorations` supports `explore(steps)` spaces with `any`, `space.find(steps)` with an `expect:` target, `space.sample(count, seed)` judged by the instance's claims, and `refuted(machine, claim)`; `assertRefuted` and `NativeControls` are retired. Errors: `any` outside `explorations`, zero `sample` count, `find` with no target. | fn-195-modelsuite-a-models-test-file-holds-its.19, fn-195-modelsuite-a-models-test-file-holds-its.20, fn-195-modelsuite-a-models-test-file-holds-its.21, fn-195-modelsuite-a-models-test-file-holds-its.3, fn-195-modelsuite-a-models-test-file-holds-its.9 | — |
| R4 | `conformance` tests are the only declarations the Case generator reads; each is replayed on the instance first; the expectation defaults to `satisfied` for a path and `conformant` for a sample; `through` binds by the decision in Architecture. Errors: the conformance refusals in API Contracts. | fn-195-modelsuite-a-models-test-file-holds-its.10, fn-195-modelsuite-a-models-test-file-holds-its.11, fn-195-modelsuite-a-models-test-file-holds-its.15, fn-195-modelsuite-a-models-test-file-holds-its.16, fn-195-modelsuite-a-models-test-file-holds-its.17, fn-195-modelsuite-a-models-test-file-holds-its.18, fn-195-modelsuite-a-models-test-file-holds-its.22, fn-195-modelsuite-a-models-test-file-holds-its.23, fn-195-modelsuite-a-models-test-file-holds-its.24 | — |
| R5 | The Model has no `queries` section, `Query`, `find`, `verify`, `witness`, `scenario`, `limits`, `.live`, `.explore` or capability live switch; `Realizes` has no `runs`; `satisfied`, `inconclusive` and `conformant` live in the realize package. Errors: the retired names are unreachable by authors (fn-190.16's grep). | fn-195-modelsuite-a-models-test-file-holds-its.11, fn-195-modelsuite-a-models-test-file-holds-its.18, fn-195-modelsuite-a-models-test-file-holds-its.3 | — |
| R6 | The six samples' tests are rewritten as `ModelSuite`s with Quint's runs as examples (Bank's `sendTest`, TwoPhaseCommit's `successTest`), their reachability finds as explorations, their negative controls as `refuted`, and pass on the Scala checker with intended answers unchanged. Errors: a changed answer is a finding to fix. | fn-195-modelsuite-a-models-test-file-holds-its.4 | — |
| R7 | The lint applies the section rules in API Contracts, each refusal located at the test. Errors: as listed. Source admission must examine a suite with no test calls too. | fn-195-modelsuite-a-models-test-file-holds-its.11, fn-195-modelsuite-a-models-test-file-holds-its.12, fn-195-modelsuite-a-models-test-file-holds-its.13, fn-195-modelsuite-a-models-test-file-holds-its.14, fn-195-modelsuite-a-models-test-file-holds-its.17, fn-195-modelsuite-a-models-test-file-holds-its.18, fn-195-modelsuite-a-models-test-file-holds-its.19, fn-195-modelsuite-a-models-test-file-holds-its.2, fn-195-modelsuite-a-models-test-file-holds-its.20, fn-195-modelsuite-a-models-test-file-holds-its.21, fn-195-modelsuite-a-models-test-file-holds-its.22, fn-195-modelsuite-a-models-test-file-holds-its.23, fn-195-modelsuite-a-models-test-file-holds-its.24, fn-195-modelsuite-a-models-test-file-holds-its.3, fn-195-modelsuite-a-models-test-file-holds-its.8, fn-195-modelsuite-a-models-test-file-holds-its.9 | — |
| R8 | SEMANTICS gains a section for the test file (the five-kind table, the sections, the rules), §9 Scenarios and Queries is rewritten to say the Model asks nothing of itself, and the README's test guidance follows. Errors: none beyond review. | fn-195-modelsuite-a-models-test-file-holds-its.5 | — |
| R9 | All six primary native sample machine declarations use `class Model`, including parameterless models; suites bind fresh stable-named instances with independent state and exact ownership. Singleton primary Machine declarations are refused regardless of object name, and active native fixtures/clients migrate without a compatibility escape hatch. Existing behavior, verdicts and realization bindings remain unchanged. Objects for namespaces, constants, actors and shared declarations remain; held Temporal/IR bodies are not migrated here. | fn-195-modelsuite-a-models-test-file-holds-its.6 | — |
| R10 | After the DSL, ModelSuite, class-only declarations and their docs settle, review and simplify every native framework sample into its clearest idiomatic form without changing behavior, claims, bounds, counterexamples or realization bindings. Preserve test purposes and verify before/after evidence; exact upstream comparisons and held consumers remain unchanged. | fn-195-modelsuite-a-models-test-file-holds-its.7 | — |

Coverage maps ownership, not completion: .22/.23/.24 are authored-explicit prerequisites only. Full automatic inventory, sampled/capability conformance and actual Registry integration remain blocked under .11; .3 retains its dependency on full .11.
