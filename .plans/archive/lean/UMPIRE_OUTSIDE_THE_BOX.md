# Umpire outside the box

Assessment note, 2026-09-29. It records a conversation about why the Lean model is the size it is,
what Lean and Veil buy, what could move to Go, what each cut would lose, and what a model of
Temporal *as a system* would need that no current tool gives. It is descriptive: it changes no rule
and approves no design. Line counts were measured on this date with `.lake/` excluded; tool facts
about P, Ivy, Quint, and FizzBee come from general knowledge and should be re-verified against the
tools' current documentation before any decision rests on them.

## 1. Where the lines are

`model/` holds 138k Lean lines. 40.7k are generated (`Temporal/API.lean`, `Temporal/API/Types.lean`,
`Temporal/DynamicConfig/Settings.lean`). The 97.6k authored lines split as follows.

| Part | Lines | Share |
| --- | --- | --- |
| Tests (`*Tests*` paths) | ~40,700 | 42% |
| `Umpire` production, the reusable toolchain | 42,800 | 44% |
| `Temporal` production (Feature, System, Case, Tool) | 9,500 | 10% |
| Feature models proper (Nexus Caller, Pair, Control, Success, Workflow Start, Worker, Info) | ~1,500 | 1.5% |

The toolchain is 20 subsystems: the command DSL (8.1k, of which `Umpire/Command/Syntax.lean` is
3.9k), Property language and evaluator (5.3k), Case producer and projection (4.2k), offline Evidence
evaluation (4.1k), Search with two backends (4.1k), Artifact formats (2.7k), Implementation Link
(2.6k), Variations (2.0k), and smaller ones. Declaration counts show what this code is: 6,606 `def`s
and 817 structures against 352 theorems, 1,484 `#guard`s, 910 `example`s, 788 `native_decide`s. It
is a compiler and checker written in Lean, with proofs at a few seams.

Why it grew this way:

- **Lean charges for what Go derives.** fn-93 counted about 100 hand-written `X.name` functions,
  13 copies of one registry pattern, 952 lines of hand-escaped JSON, and five identical error records.
  Every `enum` needs `BEq`, `DecidableEq`, `Repr`, and `Finite` instances.
- **Testing is mandated per behaviour in Lean.** The 120-line Nexus `Success` specimen carries a
  1,890-line test file. The Nexus family is 7.8k lines, of which the product models are 1.2k.
- **Six weeks, 94 specs, three architectures.** `model/` first appeared on 2026-08-19 and has 650
  commits. Authored churn is 236k added and 139k deleted, so 59% of what was written is gone. fn-93
  names about 15k surviving lines that run only from tests or facades (`Variations`, the Evidence
  chain, `ImplementationLink.Application`, retired artifact families).
- **The protocol is implemented twice by design.** SEM-18 requires Lean and Go to agree on Case
  production, so `Shared`, `Testpilot.Authoring`, and `Testpilot.Correlated` mirror Go, and
  `ModelLint` (3.2k) enforces MOD-01 to MOD-18 in Lean.

Per-feature cost is the number that matters for adoption and it is small: Workflow Start is a
110-line model plus a 513-line test and a 258-line realization. The README records a cold build of
about 15 minutes before Veil was added; the build cache is 3.3 GB. Which modules dominate build time
has not been measured. The 33k generated API lines sit in every Feature module's import closure and
are the first suspect.

## 2. The Go trace approach, as a comparison point

`chasm/lib/activity/model/vocabulary.go` names 19 events and their flags and derives nothing.
`tests/activity_parity_test.go` (883 lines) hand-writes a trace and an expected outcome per test and
drives the same trace through two realizations, standalone and workflow activity, requiring both to
match. `validateTrace` is a rule of thumb standing in for the state machine. The 14,598-line standalone
suite uses no traces. The approach produced a run of real parity fixes (nil-failure retryability,
heartbeat reset defaults, cancellation error parity, retry-state exposure, failure truncation,
timeout cause chaining) for under 2k lines, because its oracle is a second implementation.

Against the vision it wins on developer friendliness and on "translate existing tests", and it
cannot reach determinism, canary, exploration, or clock independence, because each of those needs
something that derives valid traces and expected outcomes. Umpire lacks its one real asset: a
differential oracle. A Set that realizes one activity machine through two realizations and requires
their projected Facts to agree would give exploration a second oracle that does not depend on the
model being complete. That is a Set-level feature, not a port of the driver.

## 3. What Lean and Veil buy

Today:

- A proof-carrying compile step from step functions to a finite table, with stuck-state and
  reachability checks, kernel-checked.
- Kernel-checked refinement (`refines:`/`map:`) and composition (`compose`) over finite tables.
- Veil's proven breadth-first checker as the default Search backend (fn-88, done), with the old
  depth-first traversal frozen as a differential oracle. The largest checked-in model has 158
  reachable states. Search does not run at authoring time: `query` records a declaration and results
  are pinned in tests or computed at Case production.
- A DSL that reads like a spec, at the cost of 8k lines of elaborators.
- Purity, so deterministic fixtures are free.

Not today:

- Bug finding. Everything that finds a Temporal bug is Go: runtime, Driver, Contract evaluator,
  exploration and replay loops. Lean decides what to run.
- Any vision bullet uniquely. The one Lean could serve uniquely, unbounded parametric claims, is
  the one the architecture rules out, because the `machine` command materializes the table before
  search (`UMPIRE4_DIRECTION.md` section 1).
- Developer friendliness. `FEATURE_AUTHORING_ASSESSMENT.md` puts time to first model at days.

The defensible case for Lean is the direction doc's: an oracle with proofs behind it is a stronger
claim than one with tests behind it. The case against is that the differentiator has not yet found a
Temporal bug and blocks the adoption bullets. SCP-03 mandates Lean; changing that is a GOV-02
decision.

## 4. Moving the back half to Go

The seam: Lean stops emitting Cases and emits one versioned **proto** export per Query, rendered
with the existing canonical ProtoJSON encoder (`model/Testpilot/ProtoJSON.lean`) and decoded strictly
in Go as Cases are today (`common/testing/testpilot/case.go`, `DiscardUnknown: false`). The export
carries the finite table, catalog (entities, parties, evidence lines, claims, switches, Definition
IDs, fingerprint, Known Gaps), the selected witness, lowered `require` clauses and field relations,
and set bindings. Go owns realization, Case assembly, rendering, replay reduction, inventory, lint.
Search and exploration candidate enumeration stay in Lean, because finding a path is a search and
fn-88's rationale is that witnesses stay Lean values replayed through the kernel.

| Lean today (production) | Goes to | Go added (estimate) |
| --- | --- | --- |
| `Umpire.Case` (4.2k) | Go Case compiler over the export; fixture goldens prove parity | ~2.5k |
| `Temporal.Case.Realization.*` (1.2k), `Testpilot.Authoring` (0.8k) | Go realizations | ~1k |
| `Umpire.Artifact` (2.7k) and Go `internal/artifactv2` (1k) | The export | ~0.4k, 1k Go deleted |
| `Umpire.Evidence` (4.1k), `ImplementationLink.Application` (1.0k) | Nothing; `PreparedCase.Evaluate` exists | 0 |
| `Umpire.Variations` (2.0k) | Delete; unreachable | 0 |
| `Umpire.Replay`, `Tool.ReplayBridge` (0.9k) | Go `replay` chooses edits, re-admits by row lookup | ~0.3k |
| `Umpire.Promotion` (0.45k) | Go proposal renderer | ~0.2k |
| Inventory, Evaluation Profiles, goldens, inspect, renderer (1.7k) | Go generators | ~0.8k |
| `ModelLint` (1.4k) | Go lint via `internal/leannames` | ~0.4k |
| `Shared` (0.6k) | Go-only | 0 |

About 24k production lines and 12k to 15k test lines leave Lean, landing near 60k. Go gains about 6k
and loses about 1k. `Testpilot.ProtoJSON`, `Carried`, and `Protocol` (about 300 lines) stay for the
export. SEM-18 needs an amendment: Lean produces the deterministic export, Go is the Producer, which
the Producer definition already allows.

## 5. The second cut

What remains is shaped by one feature family.

- **Property language to what is used.** Feature models use `when:` 18 times, `holds:` 17 times,
  `relates:` twice. The language supports six operators, cardinality, key, presence, five field
  roots, and captures. Trim to Bool predicates, `when:`, and equality relations: about 1.5k lines.
- **Retire the frozen reference backend** once the monitor lowering covers every used form:
  `Search.lean` (1.4k), `Branches` (0.7k), the 840-line differential test. The kernel replay gate
  stays as the trust boundary.
- **Prune single-use commands under SCP-01.** `from/restrict/extend` serves only the negative
  control. Keep `compose` (two models, and it is the fault story) and `examples:` (class splitting).
- **Drop or park the configuration models.** `System.Configuration`, `Callback`, `Matching` have no
  non-test consumer.
- **Change test style.** 258 `#guard_msgs` pins and row-by-row table pins become one fixture diff
  per machine and one rejection pin per error kind.

That lands near 30k to 35k, about 18k of it production. A Lean-hosted DSL does not go below roughly
10k of core (elaborators 4k to 5k, table generation 1.5k, ids 1k, proofs 1k to 2k, exporter 0.3k)
plus 300 to 800 lines per feature plus tests.

## 6. What each cut loses

Three losses are real and should be decided rather than absorbed.

1. **The evaluator-agreement proof becomes a test.** Today the search-time monitor, the Contract
   lowering, and the runtime monitor share Lean semantics with agreement proofs, so "a satisfied
   search predicts a satisfied Verdict" is kernel-checked. After the move it is a differential test
   against the Go evaluator. Nothing visible changes; the assurance behind a green result weakens.
2. **White-box scaffolding goes.** The Evidence chain and `ImplementationLink.Application` are the
   path from internal records to a model verdict. The mode has been off since fn-81 and nothing
   reaches the code. Reviving white box means rebuilding that path in Go, where the records are.
3. **Exploration narrows.** Variations (parameter sweeps) is unreachable and can go. `examples:`
   class claims are how exploration splits a class on a divergent member and should stay. Without
   both, exploration only walks rows and stops testing whether the author's groupings and defaults
   were right.

Everything else is unused today, redundant with Go, or a test-style preference. Two cuts not to make
on line count alone: `compose`, and API resolution from the descriptor set instead of the 33k
generated lines, which is a build-time bet that has not been measured.

## 7. The model Temporal needs

The target is a model of the system that grows gradually, like gradual typing: it never describes
everything, it says what is described and what is a hole, and it lets a person zero in on one
subgraph around a feature, bug, or incident. Four properties, and what each forces:

1. **Gradual.** Holes are first-class, with a declared interface. Step results are three-valued:
   allowed, invalid, unspecified. Today an empty row list means both "rejected" and "nobody said".
2. **Two levels that connect.** Feature machines say what the user sees. A system layer says how
   work moves. A binding says "this step rides this task over this route". The current System side is
   a second state machine, not a topology, so there is nothing to place a fault on.
3. **Scoped verification.** A focus names a subgraph; everything outside collapses to its declared
   interface (assume-guarantee). The summary assumed about history when verifying matching is what is
   verified when the focus moves to history.
4. **One truth, many consumers.** A checked proto IR read by a fast bounded checker, the real-server
   runner (Testpilot), and exporters to TLA+ or Veil for deeper questions.

Three layers in one IR:

- **Topology.** Nodes: frontend, history, matching, worker, persistence, finer when a focus needs
  it (history's task categories `Transfer`, `Timer`, `Outbound`, `Visibility`, `Replication` exist
  in Go). Edges: `rpc` (78 history and 43 matching internal RPCs are enumerable from proto), `task`
  (durable, may be delayed, duplicated, reordered, dropped), `persist`, `timer`. A **task kind** has
  an origin and a **route**, an ordered list of edges. Workflow, activity, and Nexus tasks are three
  kinds with three routes, so "Nexus takes a different path" becomes a diffable fact.
- **Machines.** Entity, parties, state, actions, timers owned by `system`, evidence lines, plus
  three-valued rows and a `rides` clause naming the route.
- **Focus.** Subgraph, machines to drive, budget. Faults are derived by walking routes inside the
  focus: per `task` edge drop, delay past each timer that could fire, duplicate, reorder; per
  `persist` edge crash before and after; per `rpc` edge timeout and retry.

Gaps come out of the checker in three kinds: **topology holes** (an opaque node or edge a scenario
traverses), **inventory holes** (task kinds, RPCs, persistence operations the code has that no route
mentions; the existing Generated Data pipeline extends to these), and **semantic holes** (reachable
state-action pairs whose row is unspecified).

## 8. Language, IR, checker: three decisions, not one

- **IR**: proto, checked by a Go loader with source-location diagnostics. Milliseconds of feedback,
  one place for semantic checks, and the strict schema with precise errors that an AI author iterates
  against well.
- **Checker**: a Go breadth-first search over the focused product state space first. Reachable state
  counts here are in the hundreds. Veil and TLA+ become exporters, added when a question needs them.
- **Syntax**: the smallest decision; pick for readers, since most will read and not write. Options:
  Lean macros (best-looking, worst loop, least familiar to models); Go builders (same language as the
  server, reads worse, no sum types); a compact standalone text DSL parsed in Go (about a thousand
  lines of parser; P-shaped); Rust with Stateright or FizzBee's Starlark (real checkers, another
  toolchain, no topology or gradual layer). Recommended: the standalone DSL over the proto IR.

## 9. What existing tools pick

Two axes explain them.

**What a state is.** Global state with guarded actions (TLA+, Quint, Alloy; Umpire's `machine`).
Communicating state machines with typed event queues (P; Stateright actors; FizzBee sits between with
global state plus `role`s). Relational state with imperative actions in a decidable fragment (Ivy;
Veil is Ivy in Lean).

**How a claim is checked.** Explicit-state enumeration (TLC, P, FizzBee, Stateright, Veil concrete;
P samples schedules by default rather than exhausting). Symbolic bounded (Apalache). Inductive
invariants for any N (Ivy, Veil symbolic; the author must find the invariant).

| Property kind | TLA+ / Quint | P | Ivy / Veil | FizzBee |
| --- | --- | --- | --- | --- |
| State invariant | `Inv` at every state | `assert` in a spec monitor or handler | Proven inductive or checked bounded | `always` |
| Trace property | Temporal formula | Spec monitor observing announced events | History relations in state, then invariant | Assertions plus auxiliary state |
| Liveness | Temporal formula under fairness | Hot states in a monitor | Weak | `eventually`, `always eventually` via Markov reachability |
| Refinement | Refinement mapping | Module refinement over observable events | Isolates, `implements` | Not known |
| Faults | Authored actions | Authored network or failure machine | Authored actions | Implicit at every yield: crash, loss |
| Parametric N | Symmetry sets; Apalache bounded | No | Yes | No |
| Against the real system | Trace validation | Generated code or harness | No | MBT adapter replaying walks |

## 10. Fit for Temporal

Temporal is components joined by durable tasks and RPCs: delivered late, twice, or out of order,
retried, with crashes before or after persistence writes, and timers whose relative order matters.
The claims are "this interleaving cannot happen", "within these timeouts it closes", "the standalone
and workflow paths agree", and "the real server does what the model says".

- **P's abstraction is the right system layer**: machines, typed events, queues, spec monitors, and
  module refinement for scoping. Its channels are reliable FIFO by default, so duplication and
  reordering need a network machine, and its checker samples.
- **TLA+-shaped guarded actions are the right feature layer**, which is what Umpire's machines are.
- **FizzBee's implicit faults are the right fault story**: faults derived from a channel's declared
  failure semantics at every yield, which no other tool does.
- **Spec monitors are the right property style** for "terminal is final", "at most one started
  attempt", "no reply while the worker is stopped". Invariants cover the rest.
- **Ivy's parametric proofs are the poorest fit.** The questions are about a handful of tasks, not
  arbitrary N, and invariant discovery is the wall.
- **Conformance is already solved** by Testpilot, which is closer to P's harness than to FizzBee's MBT.

No single tool is the answer. The fit is P's machines-and-events with per-channel fault semantics
for the system graph, TLA+-shaped machines for features, monitors for properties, module refinement
for scoping, and the existing Go runtime for conformance. Of existing tools P is nearest and would
need holes and topology-derived faults layered on. Quint is the choice for a mature typed checker if
components are encoded as state; FizzBee for the fastest first model with fault exploration, without
refinement. P's module refinement is its least used part and should be prototyped before relied on.

## 11. What carries over

From Umpire, the design vocabulary: entities and parties, timers as `system` actions, evidence lines
with fail-closed reading, Known Gaps, Definition IDs and fingerprints, deterministic Cases,
`driven`/`observed` bindings, the functional/canary/exploratory split. The whole Go runtime:
preparation, Driver, Contract evaluator, exploration and replay loops. What does not carry over
under the shape in section 7 is the host: the proof-carrying table compile, the refinement kernel
check, and Veil become optional exporters rather than the spine.

## 12. A first slice

Focus on matching with three task origins. It exercises every mechanism once. Illustrative syntax:

```
topology temporal
  node frontend | history | matching | worker | persistence
  edge history -> matching     : task
  edge frontend -> matching    : rpc      -- polls
  edge matching -> persistence : persist taskqueue
  node persistence opaque

task workflowTask  origin history  route [history->matching, matching->persistence, frontend->matching]
task activityTask  origin history  route [history->matching, matching->persistence, frontend->matching]
task nexusTask     origin frontend route [frontend->matching, matching->persistence, frontend->matching]

machine standaloneActivity
  ...
  steps:
    poll: pollStep rides activityTask
    scheduleToStart: scheduleToStartStep

focus matchingEntry
  nodes [matching]
  drive [standaloneActivity, workflowStart, nexusCaller]
  depth 4
```

The checker's first outputs would be the fault list derived from the routes and the gap list, which
on day one says that persistence is opaque, that the Nexus route skips history, and that the
`Outbound` and `Timer` categories appear in no route. That is the conversation the model should
produce, and it is one no protocol-shaped model produces.

## Related notes

- [UMPIRE4_VISION](../UMPIRE4_VISION.md), the bullets everything above is measured against
- [UMPIRE4_DIRECTION](../archive/lean/UMPIRE4_DIRECTION.md), Veil, Specula, tracing, FizzBee
- [UMPIRE_CMP_FIZZBEE](../archive/lean/UMPIRE_CMP_FIZZBEE.md)
- [VEIL_BACKEND_RESEARCH](../archive/lean/VEIL_BACKEND_RESEARCH.md)
- [FEATURE_AUTHORING_ASSESSMENT](../archive/lean/FEATURE_AUTHORING_ASSESSMENT.md)
- `.flow/specs/fn-88-veil-concrete-checker-as-the-umpire.md`, `fn-93-simplify-the-lean-model.md`,
  `fn-94-simplify-the-testpilot-go-runtime.md`
