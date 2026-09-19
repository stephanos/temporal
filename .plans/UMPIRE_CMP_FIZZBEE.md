# Umpire and FizzBee compared

Research note, 2026-09-12. It compares [FizzBee](https://fizzbee.io/) with Umpire as specified in
[UMPIRE4_SPEC](UMPIRE4_SPEC.md) and as implemented under `model/`, and answers three questions:
what Umpire can take from FizzBee, whether Umpire can express more of a model as ordinary Lean code
instead of command macros, and where the Lean-based approach is stronger. It is descriptive; it
changes no rule and approves no design. FizzBee facts were read from fizzbee.io, the
`fizzbee-io/fizzbee` repository, and its shipped agent skills on the date above; FizzBee is a young
tool whose docs describe some features as work in progress, and nothing was installed or run.

## 1. The two tools in one paragraph each

**FizzBee** is a specification language plus an explicit-state model checker written in Go. A spec is
a `.fizz` file: a handful of keywords (`action`, `role`, `func`, `atomic`, `serial`, `parallel`,
`oneof`, `any`, `require`, `fair`, `always`, `exists`, `transition`, `eventually always`,
`always eventually`) wrapped around bodies written in Starlark, a Python subset. State is whatever
the `Init` action assigns; roles are classes with `self` state, functions and actions; an RPC is a
function call between roles. Actions are non-atomic by default, and at every yield point the checker
also explores a crash, message loss, or another action interleaving. Bounds come from a YAML
front matter (`max_actions`, `max_concurrent_actions`, per-action `action_options`). The checker
reports safety violations with a trace, checks liveness under weak or strong fairness through
Markov-chain reachability, detects deadlocks, reduces state with typed symmetry values, and renders
state graphs, sequence diagrams and an interactive "whiteboard" explorer in an online playground.
A separate model-based-testing (MBT) product scaffolds adapter interfaces (Go, TypeScript, Rust,
Java), then a server replays random walks over the checked state graph against the real system
through the adapter, sequentially and in parallel with linearizability checking, seeded for replay.
It ships AI-assistant skills (`fizz-spec`, `fizz-check`, `fizz-debug`, `fizz-mbt`).

**Umpire** is a Lean 4 library and a Go runtime. A Behavior Model is Lean code under `model/`: a
finite Model with enumerated state, Action, Model Outcome and Fact domains and a table of permitted
steps; Properties over Traces; Scenarios that constrain Traces; Queries that ask bounded questions
that Search answers exhaustively within typed Limits. A Producer lowers one checked Query and a
Temporal-owned realization into a versioned Case, one bounded Program and one deterministic
Contract, whose bytes are checked in as fixtures. Testpilot prepares the Case against an immutable
Profile and runs it through a Driver against a real Temporal server and SDK worker; every attempt
appends one Run and reaches one three-valued Verdict. Definition IDs, Behavior Fingerprints,
Artifact checksums, Provenance and Known Gaps tie every artifact back to the model, and evidence
that is missing or ambiguous never becomes success. Authoring goes through five command macros
(`model`, `property`, `scenario`, `limits`, `query`) that elaborate to typed Lean records;
[fn-85](../.flow/specs/fn-85-model-side-effects-as-typed-actions-and.md) extends them with
entities, actions with typed inputs, machines, observations, refinement and sets.

## 2. The same design, written twice

The two-phase-commit example on FizzBee's landing page and the checked-in Nexus success Model in
[`Success/Model.lean`](../model/Temporal/Feature/Nexus/Success/Model.lean) are the smallest
representative specimens of each surface.

FizzBee:

```python
role Participant:
  action Init:
      self.status = "init"
  func placehold():
      vote = any ["accepted", "aborted"]
      self.status = vote
      return self.status
  func finalize(decision):
      self.status = decision

role Coordinator:
    action Init:
        self.status = "init"
    action Checkout:
        require(self.status == "init")
        self.status = "inprogress"
        for p in participants:
              vote = p.placehold()
              if vote == "aborted":
                  self.finalize("aborted")
                  return
        self.finalize("committed")
    func finalize(decision):
        self.status = decision
        for p in participants:
            p.finalize(decision)

action Init:
    coordinator = Coordinator()
    participants = []
    for i in range(2):
        participants.append(Participant())

always assertion ParticipantsConsistent:
  for p1 in participants:
    for p2 in participants:
      if p1.status == 'committed' and p2.status == 'aborted':
        return False
  return True
```

Umpire today:

```lean
entity operation

enum State  | scheduled | started | succeeded
enum Outcome | acknowledged | completed

structure Lifecycle where
  state : State
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

action awaitStart
  party: caller
  on: operation

action awaitSuccess
  party: caller
  on: operation

def awaitStartStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .scheduled then [] else
  [{ outcome := .acknowledged, state := { state := .started }, facts := [] }]

def awaitSuccessStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .started then [] else
  [{ outcome := .completed, state := { state := .succeeded }, facts := [] }]

machine lifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  steps:
    awaitStart: awaitStartStep
    awaitSuccess: awaitSuccessStep

property successfulResult
  model: lifecycle
  when: awaitSuccess
  require:
    state: succeeded
    outcome: completed

scenario successfulCompletion
  model: lifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

limits shortTrace
  steps: 2
  actions: 2
  search: 16

query completion
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace
```

Three differences are visible before any semantics are discussed. FizzBee's logic is code: a
guard is `require`, a state change is an assignment, a choice is `any`, and an invariant is a
function returning a Bool. Umpire's logic is a table and a set of keyed fields (`state:`,
`outcome:`, `fact:`), which is exact and inspectable but is a notation an engineer has to learn
rather than one they already know. And FizzBee's spec is complete in one file with no separate
"how does this reach a real system" step, while Umpire's file continues with a `case` block, a
realization, a generated fixture and a Go test; that is the price of running against a real
server rather than an adapter, and it is where fn-83 and fn-85 are spending their effort.

## 3. Side by side

| Dimension | FizzBee | Umpire |
| --- | --- | --- |
| Host language | Custom ANTLR grammar with Starlark bodies interpreted by a Go checker | Lean 4; command macros elaborate to typed records; logic checked by Lean |
| State | Arbitrary Starlark values (lists, dicts, sets, role instances); bounded only by checker options | Finite enumerated domains per Model; counts and slot pools planned in fn-85 |
| Transitions | Imperative code with yield points; the checker derives the state graph by executing it | Declarative rows `before + action → after, outcome, facts`; fn-85 adds guards, field updates and first-matching-row semantics |
| Nondeterminism | `any`/`oneof` in bodies; implicit crash and message loss at every yield point | The Model owns outcomes (SEM-07): authors request an Action, the table says what can result; faults are declared actions of a party (EVD-20) |
| Concurrency | First-class: interleavings of non-atomic actions, `parallel` blocks, `max_concurrent_actions` | Not modeled as interleaved imperative steps; interleavings across entity instances become paths (fn-85); goroutine and network scheduling are explicit non-goals |
| Safety | `always assertion` (state predicate), `transition assertion` (before/after), `exists assertion` (reachability) | Property clauses over Traces at a step and across steps; `find` Query is the reachability witness |
| Liveness | `eventually always`, `always eventually` under weak or strong fairness; Markov-chain reachability, unbounded | Bounded only (SEM-09): `within N` model steps, `rule_events` or milliseconds at runtime; unbounded "eventually" is rejected |
| Checker | Explicit-state BFS, deadlock detection, symmetry reduction, liveness graph analysis | Exhaustive finite Search within typed Limits; `limitReached` is inconclusive (PLN-04); completeness and executability carried as proofs |
| Faults | Implicit: thread crash, process crash with ephemeral-state loss, message loss, partition; `@state(ephemeral=[...])` | Explicit: `workerStop`/`workerResume` instructions the Driver realizes, one `FAULT_INJECTED` Run Event each |
| Real-system execution | MBT adapter per role: `ActionX(args) (any, error)` plus optional `GetState`; server drives random walks over the checked graph; sequential and parallel runs; linearizability via porcupine | One deterministic Case per Query: a Program DAG of typed instructions plus a Contract of monitor rules; Driver executes; Run and Verdict recorded |
| Evidence | Return values and whole-state snapshots compared to model state; `IGNORE` sentinel for nondeterministic fields | Declared typed Observations projected from history events and RPC responses; correlation keys per instance; no ignore lists (ART-11); fail closed (EVD-04) |
| Result | Pass or fail; `ErrNotImplemented` actions are skipped, so an unimplemented adapter passes | `satisfied`, `violated`, `inconclusive`; Run disposition and cleanup status reported separately (QLF-05) |
| Test selection | Random with seeds; no shrinking; `--max-actions` to shorten traces | Deterministic witness selection (PLN-02); byte-identical fixtures; Exploration engine exists in Lean but emits no Case yet |
| Identity and provenance | The seed | Definition IDs, Behavior Fingerprints, Artifact checksums, Provenance, Known Gaps |
| API awareness | None; the adapter is untyped glue written by hand | Generated `Temporal.API` and `Temporal.DynamicConfig` catalogs; field Properties read protobuf descriptors; evidence names resolve to history event kinds |
| Refinement | A hand-written adapter called a refinement mapping; checked only by running | Forward simulation between machines and the Feature-to-System Implementation Link, checked by Lean before anything runs |
| Tooling | Playground, state graph, sequence diagrams, interactive explorer, VS Code, AI skills | `umpire-inspect`, `umpire-case`, `INVENTORY.md`; Lean editor; no diagrams |
| Trust | Interpreter correctness | Lean kernel; `native_decide` only where policy allows; axiom inventories pinned in tests |
| Time to first model | Hours for a Python programmer (per its own testimonials) | Days, and only the Model block is friendly today (fn-83's own assessment) |

## 4. What to take from FizzBee

Ranked by expected value to Umpire against cost. Each item names what FizzBee does, what Umpire
does today, the proposal, and which rules it touches.

### 4.1 Logic as Lean code, declarations as commands

This is the central answer to "can we express things more like code". Yes, and Lean is a better
body language than Starlark for it, provided the finite table stays the canonical form.

FizzBee's grammar contributes only structure. `action`, `role`, `func`, the block modifiers and the
assertion kinds tell the checker how to schedule and what to check. Everything an author reasons
about, the guard, the update, the choice, the predicate, is ordinary code. That is the whole reason
engineers find it approachable.

Umpire's command surface today has the opposite balance. The `steps:` table and the `require:`
block with `state:`, `outcome:` and `fact:` keys carry the logic, and fn-85 grows the row grammar
further: guards with alternatives and wildcards, `+1` counter updates, bound pattern variables,
`reject`, `result:`, `after (timer)`. Every one of those is a construct that Lean already has as
`match`, `if`, record update, `Option`, and `List`. The row grammar is re-implementing a small
functional language inside a macro, with its own diagnostics, its own shadowing analysis, and its
own learning curve.

The proposal is to move the logic into Lean functions over Lean structures and keep commands for
what a command is good at: naming things, binding identity, and linking to generated vocabulary.

```lean
-- Declarations stay commands: they carry Definition IDs and resolve generated names.
enum Phase | scheduled | started | backingOff | succeeded | failed
enum Outcome | acknowledged | retried | completed | rejected

inductive Reply
  | syncSuccess
  | async
  | handlerError (retryable : Bool)
  deriving DecidableEq, Repr

/-- Per-instance state. Every field is finite, so the enumerator is derived. -/
structure Operation where
  phase : Phase
  attempts : Fin 4
  deriving DecidableEq, Repr

-- Logic is code. The Model owns outcomes, so a step returns what the server may do (SEM-07),
-- and a list is how a step says "either of these".
def handlerReply (op : Operation) : Reply → List (Operation × Outcome)
  | .async =>
      if op.phase = .scheduled then [({ op with phase := .started }, .acknowledged)] else []
  | .handlerError false =>
      if op.phase = .scheduled then [({ op with phase := .failed }, .rejected)] else []
  | .handlerError true =>
      if op.phase = .scheduled ∧ op.attempts < 3 then
        [({ op with phase := .backingOff, attempts := op.attempts + 1 }, .retried)]
      else []
  | .syncSuccess =>
      if op.phase = .scheduled then [({ op with phase := .succeeded }, .completed)] else []

-- Predicates are code too.
def neverRetriesPastLimit (op : Operation) : Bool := op.attempts ≤ 3

-- The command names the pieces and derives the table, the fingerprint, and the diagnostics.
machine nexusProtocol
  for: operation
  state: Operation
  ends: [succeeded, failed]
  step handlerReply
    evidence: nexusOperationStarted when phase: started
  invariant neverRetriesPastLimit
```

How this stays inside the current rules:

- **AUT-05 and PLN-02** require portable, inspectable data with deterministic identity. A Lean
  function is opaque to artifacts and to Go. The resolution is the one AUT-08 already describes:
  the function is the authoring form, and the `machine` command enumerates it over the derived
  finite domain at elaboration into the same `FiniteTable` rows the `steps:` block produces
  today. The table is what gets fingerprinted, searched, lowered and inspected. Two functions that
  enumerate to the same table have the same Behavior Fingerprint, which is exactly the "changes
  with behavior, not with source" property the spec asks for. The enumeration bound already exists
  (`transitionBound` in `Umpire/Command/Syntax.lean`).
- **AUT-09** admits domains a macro derives from the author's own declarations, currently
  enum-like inductives. It would need to be read as also covering a `structure` whose fields are
  all enum-like, `Bool` or `Fin n`, which is a mechanical extension of the same idea. An
  `attempts : Nat` field must be rejected with a located error, not silently bounded.
- **AUT-07 and AUT-07a** say `Umpire.Command` is the only surface and must define no behavior of
  its own. A command that enumerates an author's function defines no behavior; it transcribes one.
  The `steps:` row form can remain as sugar for the same table, so nothing existing is invalidated.
- **Diagnostics get less precise and must be compensated.** Today a shadowed or unreachable row is
  reported at the row. With a function, the same facts are found on the enumerated table and must
  be reported with a concrete witness: "`handlerReply` admits no step from `{ phase := .backingOff,
  attempts := 3 }` on `.async`; if that is intended, list `backingOff` under `ends:`". Pinning
  these with `#guard_msgs` is the existing practice and still works.
- **Trust.** Enumeration proofs run under `native_decide` or `decide` on small domains; the
  repository's policy already accepts `native_decide` at the `checked` seam. Nothing new is trusted.
- **Elaboration cost.** The Race prototype measured elaboration for hand-written tables; the
  structure-of-enums domain multiplies (a five-phase, four-attempt operation is twenty states,
  fine; five such instances is twenty to the fifth, not fine). Bounds and symmetry (4.5) are the
  same answer FizzBee gives.

What this buys: the `steps:` mini-language, its wildcard guards, its `+1` updates and its pattern
binders do not need to be designed, documented and taught. A Temporal engineer who can write a Go
`switch` can write `handlerReply`. The commands shrink to what a reader cannot infer from Lean:
which function is the machine, which entity it tracks, which recorded event confirms which step,
and what the Definition ID is. That is AUT-01's target.

What it does not buy: the model does not become imperative. A FizzBee action mutates state across
yield points; the checker discovers the intermediate states by executing it. Umpire's step
function returns the successors of one atomic step, which is the relation SEM-07 wants and the
Contract can monitor. Non-atomic actions are the one FizzBee idea this proposal deliberately
leaves out; see section 6.

### 4.2 Diagrams from the checked table

FizzBee's strongest adoption lever is not the language. It is that every run produces a state
graph and a sequence diagram, and the playground lets a reader click through actions and `any`
choices. Nothing in Umpire renders anything, yet all the inputs are already canonical data: a
`CheckedTable` is a labeled graph, a Scenario's witness Trace is a path through it, and a Case's
Program is a DAG of instructions over declared roles, which is a sequence diagram waiting to be
drawn.

Proposal: `umpire-inspect` gains a `--mermaid` rendering of a Model's table as a state diagram with
the witness path highlighted, and of a Case's Program as a sequence diagram with one lifeline per
role. The generator that owns `INVENTORY.md` and the per-feature `COVERAGE.md` embeds them. This is
a Generated View under ART-07, bound to the source checksum; it defines nothing. Cost is small;
value for design review, for onboarding, and for the "share your spec for review" step FizzBee
puts at the end of its tutorial is high.

### 4.3 Roles as the unit of organization, with instances

FizzBee's `role` is a class: per-instance state under `self`, actions and functions on it,
constructor parameters, `self.__id__`, dynamic creation and deletion, and RPC as a function call
on another role instance. Engineers draw block diagrams and each box becomes a role; the tutorial
says so explicitly.

fn-85's `entity`, `party` and `machine ... for: operation` are the same shape and should be kept.
Two details are worth borrowing outright. First, a role's `Init` is where its state is declared,
so state and behavior sit in one block rather than in an `enum`, a `structure` and a `machine`
declared apart; with 4.1, a machine's state type and its step function can be declared inside one
`namespace Operation`, which reads as a class. Second, roles carry parameters at construction
(`Server(ID=i)`), which is how a spec distinguishes instances without a global registry. fn-85's
entity `key:` serves the runtime; an author-facing constructor parameter serves the model.

### 4.4 Implicit fault exploration in Search, explicit realization at runtime

FizzBee explores a crash at every yield point and a loss on every non-atomic call without the
author listing placements, and marks fields `ephemeral` so a crash resets them. That is why its
2PC example finds the response-lost case with no fault code.

Umpire cannot adopt this at runtime: the server is black-box and EVD-20 rightly requires a fault to
be a declared, Driver-realized instruction that proves itself with a Run Event. But fn-85 already
allows a row with no guard (`+ workerStop`) to match in every state, which is the model-side half
of the idea. Two additions make it the whole idea:

- A Scenario may say `fault: workerStop anywhere` and let Search place it, rather than naming a
  hook. The result is still a finite set of placements the Producer lowers one at a time, so
  PLN-02 holds.
- A machine's `state:` block may mark fields `ephemeral`, and a declared crash action resets them.
  For Temporal this is a precise fit: history is durable, worker memory is not, and the difference
  is what most worker-outage bugs are about.

### 4.5 Symmetry for entity instances and counters

FizzBee's typed symmetric values (`nominal` for interchangeable IDs, `ordinal` for ordered IDs,
`interval` for counters) canonicalize states so that `{k0:v0, k1:v1}` and `{k1:v1, k0:v0}` are one
state, and its `symmetric role` does the same for interchangeable instances held in a `bag`.

fn-85 introduces several instances per entity and `count` fields, which is exactly where finite
enumeration blows up. Entity instances are nominal by construction (an operation is named by its
scheduled event, and nothing compares two of them for order), so Search can canonicalize by sorting
instance states before hashing. Counters that only ever compare against a bound are interval
values. This is a Search concern, not an authoring one, and it is the single technique most likely
to keep fn-85's five-operation Query inside Limits.

### 4.6 Seeded exploratory runs and parallel-run checking

FizzBee MBT does not enumerate; it samples thousands of random walks over the checked graph,
reports the first failure with its seed, and, in parallel mode, checks that concurrent invocations
of real operations are linearizable against the sequential model. It has no shrinking and says so.

Umpire's Exploration engine is pure Lean with no Case output, and fn-85's `exploratory` set only
enumerates coverage targets. When fn-33 gives it execution, the FizzBee shape is the right one to
copy: sample paths from the checked table under a seed, produce one Case per sampled path through
the same Producer, and record the seed in Provenance so the path is replayable. The parallel mode
is a genuinely different capability, checking real concurrent operations against a sequential
model, and Umpire has nothing like it; it belongs on the list for the canary work, not before.

### 4.7 Onboarding: tutorial-first, runnable everywhere, with an agent skill

FizzBee's docs start from a clock with two actions, every snippet has a playground link, and the
tool installs skills that teach an AI assistant its language, checker and debugging. Umpire has
[`AutoClose.lean`](../model/Temporal/Feature/Nexus/Experimental/AutoClose.lean), which is a good
tutorial for the expert path, and fn-83 R5 plans an `AUTHORING.md` whose snippets are asserted
equal to regions of a real Model file, which is better than a playground link because it cannot
drift. Add to that a repository skill for the command surface listing every located diagnostic and
its fix; it is cheap and it is the form of "documentation" both humans and agents actually read.

### 4.8 Small syntactic conveniences

- `require` as a guard reads better than a keyed `when:`; with 4.1 it is simply an `if` in the
  step function.
- `exists assertion` is Umpire's `find` Query; the name is clearer for a reachability witness.
- `transition assertion (before, after)` is a two-state predicate; with 4.1 it is
  `Operation → Operation → Bool` and needs no new clause kind.
- Per-action occurrence bounds in one place (`action_options: X: max_actions: 1`) are already
  expressible as Scenario occurrence constraints; surfacing them next to `limits` would be tidy.

## 5. Where the Lean approach is superior

These are the things FizzBee cannot do, or does by hand-written glue that nothing checks, and they
are the reasons Umpire exists as a regression gate rather than a design tool.

1. **Checked declarations against the real API.** An Umpire Property can require that the
   submitted `workflow_type.name` equals the one a `WorkflowExecutionStarted` event records, read
   through the generated `GetWorkflowExecutionHistory` response schema, and a misspelled field or a
   type mismatch fails where it is written. Evidence names resolve against generated history event
   kinds. FizzBee has no model of the system's API; the adapter maps action names to methods by
   hand and compares whatever `GetState` returns. FizzBee's own limitations page lists wrong line
   numbers in errors, restrictions on where a function may be called, and that extracting a local
   variable silently changes semantics; Lean has none of these.

2. **Proofs where FizzBee has runs.** The executable and denotational meanings of a Property are
   connected by a theorem. The finite adapter carries completeness and executability proofs. A
   protocol machine that refines a product machine is checked by forward simulation before any
   Case exists, and the Feature-to-System Implementation Link is the same kind of object. FizzBee's
   "refinement mapping" is an adapter; the only check that it is a refinement is that tests pass.

3. **Honest results.** A Search that hits a Limit is `limitReached`, not "no counterexample"
   (PLN-04). Evidence that is missing, ambiguous, stale or causally unrelated cannot establish
   success (EVD-04). A Run has a disposition, a cleanup status and a Verdict, and a proved violation
   survives cleanup failure (QLF-05). FizzBee MBT has pass and fail; an action whose adapter
   returns `ErrNotImplemented` is skipped, so an empty adapter passes a thousand runs, which its
   quick start demonstrates and defends as a feature. For a gate on a production server, Umpire's
   three values are the correct number.

4. **Determinism and identity.** Identical definitions, Limits and seed give identical Plans and
   fixture bytes (PLN-02, ART-11), and every artifact carries Definition IDs, Behavior Fingerprints,
   checksums, Provenance and Known Gaps. Reviewers diff a fixture and see exactly what changed in
   behavior. FizzBee's reproducibility is a seed and a state-graph directory with a timestamp in
   its name.

5. **One semantic authority across Lean and Go.** `Shared.CorrelatedObligation` is the monitor
   semantics both the Lean Producer and the Go runtime implement, and the Contract rule is derived
   from the checked Property rather than written twice. FizzBee's checker and its MBT server share
   the state graph, but the adapter's state mapping is a second, unchecked description of the
   system.

6. **Bounded liveness that runs against a real system.** SEM-09 forbids unbounded "eventually",
   and a Contract rule's Deadline counts Run Events, not wall clock (EVD-07, EVD-21), so a slow CI
   host cannot turn a healthy outage into a violated one. FizzBee's liveness is a fairness-plus-
   Markov argument about the model; its `fizz-spec` skill sets `liveness: "false"` in the
   front matter of MBT-ready specs, so nothing about progress reaches the system under test.

7. **Per-instance correlation.** A Correlated rule is tracked per operation under an explicit key,
   so one operation's completion cannot discharge another's obligation. FizzBee compares whole
   state snapshots, and its `IGNORE` sentinel exists precisely because that comparison is too
   coarse; ART-11 forbids that sentinel for the same reason.

8. **Extensible without forking the tool.** A new notation, a helper, a domain-specific check or a
   lint is a Lean file in the same repository as the models, with hygienic macros, located errors
   and `#guard_msgs` regressions. FizzBee's grammar is ANTLR plus a Go interpreter; a new construct
   is a change to the product.

9. **Environment discipline.** Cases bind symbolic namespaces, task queues and endpoints; Profiles
   bind physical ones; preparation is immutable; Drivers are authorized by Opcode; faults must hold
   resources no other Run shares. FizzBee MBT targets a library or a service in a test process and
   has no notion of any of this, which is fine for its purpose and disqualifying for a canary.

## 6. Where FizzBee is ahead, and which of those Umpire should not chase

- **Interleaving of non-atomic actions.** This is FizzBee's core insight: a real operation is a
  sequence of steps that can each fail, and the checker should find the partial executions. Umpire
  models a server whose steps it cannot observe or interrupt, and lists goroutine and network
  scheduling as non-goals. The right response is 4.4 (faults as explored actions) rather than an
  interleaving semantics for step functions; the latter would be a second behavioral language.
- **Unbounded state with checker-side bounds.** Lists and dicts with `max_actions` are more
  convenient than declaring `Fin 4` up front. Umpire's finiteness is what makes exhaustive Search,
  enumeration proofs and byte-identical fixtures possible; keep it, and make the bound a located
  error rather than a silent truncation.
- **Fairness and unbounded liveness.** Useful for a design question ("does this protocol always
  terminate?"), not checkable against a run. Umpire can add it only as an optional verification
  path with its own trust class (VER-06); it should not enter Contracts.
- **Probabilistic and performance modeling.** Out of scope for a regression tool.
- **A playground.** Lean's editor integration is the playground; what is missing is the rendering
  (4.2), not the interactivity.

## 7. Recommended next steps

In order of value against cost, and each small enough to be one spec or one task in an open spec:

1. Prototype 4.1 on one existing Model: a `machine` whose step is a Lean function over a
   structure of enums, enumerated at elaboration into today's `FiniteTable`, with the fingerprint
   shown to equal the hand-written table's. Measure elaboration time and diagnostic quality. This
   belongs in fn-85's task breakdown as an alternative to the row grammar, before the row grammar
   is built.
2. Add Mermaid rendering to `umpire-inspect` and embed it in the generated `COVERAGE.md` (4.2).
3. Record `ephemeral` field marking and `fault ... anywhere` placement as fn-85 design options
   (4.4).
4. Canonicalize entity-instance states in Search (4.5) before the five-operation Nexus Query is
   attempted.
5. Ship an authoring skill alongside fn-83's `AUTHORING.md` (4.7).

## Sources

- FizzBee site: [landing page](https://fizzbee.io/),
  [getting started](https://fizzbee.io/design/tutorials/getting-started/),
  [quick start](https://fizzbee.io/design/tutorials/quick-start/),
  [roles](https://fizzbee.io/design/tutorials/roles/),
  [implicit fault injection](https://fizzbee.io/design/tutorials/fault-injection/),
  [channels](https://fizzbee.io/design/tutorials/channels/),
  [symmetry reduction](https://fizzbee.io/design/tutorials/symmetry_reduction/),
  [visualizations](https://fizzbee.io/design/tutorials/visualizations/),
  [limitations](https://fizzbee.io/design/tutorials/limitations/),
  [model-based testing](https://fizzbee.io/testing/),
  [MBT quick start](https://fizzbee.io/testing/tutorials/quick-start/).
- FizzBee repository: [README](https://github.com/fizzbee-io/fizzbee/blob/main/README.md),
  [quick start for TLA+ users](https://github.com/fizzbee-io/fizzbee/blob/main/docs/fizzbee-quick-start-for-tlaplus-users.md),
  [language design notes](https://github.com/fizzbee-io/fizzbee/blob/main/docs/language_design_for_review.md),
  the `fizz-spec` and `fizz-mbt` skills under `.claude/skills/`.
- Umpire: [UMPIRE4_SPEC](UMPIRE4_SPEC.md), [UMPIRE_DSL_RESEARCH](UMPIRE_DSL_RESEARCH.md),
  [UMPIRE_DSL_EXPERIMENT](UMPIRE_DSL_EXPERIMENT.md),
  [`Nexus/DESIGN.md`](../model/Temporal/Feature/Nexus/DESIGN.md), and the fn-83, fn-85 and fn-86
  Flow specs.
