# Umpire 4: TLA+ and Quint compatibility

Design notes, 2026-09-30. This proposal records ideas for exporting Umpire Models to TLA+. It does
not change the [shared specification](UMPIRE4_SPEC.md), [IR semantics](../model/scalav2/SEMANTICS.md),
or implemented behavior.

## Recommendation

Align Umpire's semantic core closely with TLA+ while retaining Scala authoring and Umpire's domain
concepts. Define a small exportable subset whose meaning survives translation exactly. Every
construct in that subset needs a documented mapping; unsupported constructs need located errors.

TLA+ supplies the semantic target. Quint can provide a typed intermediate representation and useful
tooling. Its separation of pure expressions, state expressions, actions, and temporal formulas is
a useful reference for our own checking. [Quint language reference](https://quint.sh/docs/lang).

Keep the IR authoritative. The exporter derives a representation of its behavior, and the Go
interpreter and external checker evaluate that behavior independently. An exporter must not repair
the Model, infer missing assumptions, or replace an unsupported Property with a weaker one.

Compatibility means preserving the answers to the same Umpire questions. Similar syntax, a shared
state diagram, or two successful tool invocations do not establish that. The principles below are
the constraints on the design; the backend route and encoding remain proposals.

## Core principles Umpire must preserve

### One authored behavior, several consumers

The Behavior Model defines permitted behavior. The IR gives that behavior a language-neutral
meaning; tables, exported specifications, generated Cases, and runtime Contracts derive from it.
An adapter cannot become a second place to specify retries, scheduling policy, fault handling, or
what counts as completion. Otherwise a backend can pass a different question from the one the
author asked. This follows **SEM-01**, **SEM-16**, and **EXP-01** in the shared specification.

For example, if a Model permits an acknowledgment to be lost after delivery, both the Go path and
the export must retain the resulting duplicate-delivery alternative. Making the exporter assume
successful acknowledgment would verify a different system.

TLA+ is the proposed target for expressing the semantic core, not authority to reinterpret an
existing declaration. Any intentional semantic change belongs in the shared or IR specification
and its conformance checks first. Existing rules and pending GOV-02 amendments retain their status;
this document approves neither an amendment nor an exception.

### Relations describe possibilities; strategies choose witnesses

A Machine is a transition relation. An Action request selects a class of inputs; the Model
determines which outcomes, next states, and Facts are possible. Several Steps are alternatives,
and no Step means disabled. Faults are explicit behavior in that relation, with their declared
party and scope; an exporter must not remove them to simplify checking.

Universal `verify` and existential `find` depend differently on those possibilities. Dropping one
alternative can hide a violation; adding one can manufacture a witness. An exact export therefore
needs preservation in both directions. A deliberately approximate backend would need a separately
specified claim and could not advertise exact compatibility.

Deterministic generation and nondeterministic behavior coexist. Canonical catalogs, serialization,
and a pinned search strategy can select reproducible Cases while the Model still allows many
executions. Backend traversal order may differ, but must not change permitted executions. The
ordinary Umpire producer's deterministic selection and byte-for-byte Case requirements still apply
(**PLN-02**, **ART-01**, **ART-11**).

### A Step is an indivisible decision with observable results

An atomic Step computes its complete result from the old state and inputs. All state updates
commit together; an unchanged field retains its value. Splitting one Step into several exported
actions introduces intermediate states and interleavings unless those actions are proved hidden
and harmless for every supported Property, monitor, fairness assumption, and counted-step rule.

The state change alone is insufficient. Outcomes and the ordered list of Facts are part of the
Step, even if the product state stays equal. A cancellation sent twice can be a violation while the
operation remains in the same phase throughout. Removing the second emission as a self-loop loses
the behavior the Property is meant to inspect.

A synchronized Composition step is atomic in the same sense: combine all permitted member results,
retain the specified outcome and member-qualified Facts, and expose no half-completed sync state.
Interleaving actions leave every other member unchanged. Composition must use the IR's actual
result convention, including its first member's outcome, rather than invent a new result type.

### Explored state must retain everything a future verdict can depend on

A machine state can summarize product behavior without summarizing the history a Property needs.
The explored state must additionally retain passive monitor state, activation/evaluation history,
pending bounded obligations, correlation keys where declared, and Scenario position when it affects
the available next actions. Equality of product phases alone is not equality of checking states.

For example, two paths reach `running`: on one, a cancellation obligation is pending; on the other,
it has already been discharged. Merging them can either lose the deadline violation or attach it to
the wrong path. Likewise, identical monitor values may need different bookkeeping if one verdict
was already read or violated. The current Go search retains this distinction in its monitor state.

Monitors observe every real Step and never disable one. Their declared evaluation point controls
when a verdict is read: every Step, at an end, or after a selected result. Turning `violated` into
an enabling guard removes the counterexample instead of reporting it. Never reaching an evaluation
point leaves a monitor unread; that is not evidence that its obligation held.

State reduction, symmetry, or auxiliary-variable hiding must preserve this full distinction and
the action classes referenced by fairness. A smaller state space is useful only after that
preservation obligation is established.

### Abstraction is explicit, and its direction matters

Refinement relates two product Machines through a declared state map and visible-result rules.
Each detailed result must be carried by an allowed abstract result or qualify as a stutter under
the declared projection. Equal mapped states do not make a step invisible if it emits a Fact or
outcome the abstraction sees. The existing rule is a stuttering forward simulation, not equality
of the detailed and abstract behavior sets.

This distinction matters for claims: matching every detailed step does not show that every abstract
witness is realizable by the detailed Machine. It also does not automatically preserve a bound
counted in detailed Steps, or discharge the progress and fairness obligations of an abstract claim.
Export must reproduce the current refinement rule and Query `through` semantics; extending them
to stronger temporal claims needs an explicit rule for hidden steps and progress.

An Implementation Link is a separate, explicit connection to current implementation behavior.
Refinement between product Machines is not proof that Temporal implements either one. Likewise,
replacing an opaque provider in one Composition is scoped to that Composition and requires the
declared refinement; it must not globally select a provider or silently keep relying on the
replaced provider's assumptions.

### Every conclusion carries its assumptions and finite scope

A Query asks about its selected Scenario, starts, schedules, ending conditions, and typed Limits.
Finite input classes and fault budgets define which behavior is represented. A backend must retain
that scope and report it with the answer. A proof over two operations and one duplicate delivery
does not establish the same Property for arbitrarily many operations and deliveries.

Fairness is a stated environmental assumption, not a scheduler optimization. It must be named in
the result and attached to the same classes and enabledness relation. A retry-count bound or a
modeled deadline cannot be inferred from fairness. Model Steps, logical time, evaluated Run Events,
and elapsed milliseconds are different units; a model progress bound cannot silently become a
runtime timeout (**SEM-09**, **PLN-01**, **EVD-21**).

Keep three things distinct: finite Model domains, the trace scope of the Query, and resources spent
answering it. Changing a domain or schedule changes the question; exhausting memory, search work,
or a solver budget changes whether the question was answered. Exhaustive means complete within
the declared scope (**PLN-03**, **PLN-04**). A Scenario that admits no complete Trace is
Unsatisfiable and must not become a vacuous verification success (**PLN-05**).

### Model Facts are promises; runtime Evidence establishes what happened

A Trace contains model Steps, not Evidence that Temporal executed them. An exported counterexample
can demonstrate a Model violation and provide a candidate for realization; it cannot demonstrate a
production bug by itself. Similarly, asking for an Action or receiving an RPC response need not
establish the durable commitment that a Fact promises.

The existing execution boundary remains a generated Case with one bounded Program and one
deterministic Contract, prepared against a Profile and run through an authorized Driver. Export
must not introduce a backend-specific execution language or replace that Contract. Live and offline
Contract evaluation share semantics (**SEM-17**); model monitor evaluation and runtime Contract
evaluation still have different inputs and require an explicit lowering between them.

Observations must establish the declared identity and causal relationships. Operation A's
completion cannot discharge operation B's obligation, and timestamps on different hosts cannot
invent ordering. Public observations may suffice for one Property while a durable-commit Property
requires a white-box observation. Those environments support different claims without changing
the modeled promise (**EVD-04**, **EVD-07**, **QLF-03**).

### Unknown behavior and reporting policy must remain visible

A disabled action states that behavior is forbidden at that pair. A hole states that behavior is
unknown. Malformed input states that the declaration cannot be admitted. A Known Gap states what
support or knowledge is missing for a claim. A Known Bug acknowledges an established violation.
These are different conclusions, and none is a substitute for another.

An explored hole or exhausted resource budget cannot produce an unqualified pass. A demonstrated
violation remains a violation even when another branch is unknown or later work fails. Known Bug
policy may change reporting severity after evaluation; it must preserve the failed obligation and
its witness. Conversely, not reproducing a bug during incomplete checking does not show it fixed.

This is also an ownership principle: each failure should lead back to its Property, Definition IDs,
source locations, behavior version, and assumptions. Generated helper names and external stack
traces alone are insufficient for an author to understand or repair the Model.

## What exact compatibility would mean

For an admitted finite Model and a selected Query, let `x` denote the complete checking state:
machine state, watching monitors and verdict bookkeeping, and any Scenario or obligation state
needed by that question. A real transition is labeled by its action class and full semantic Step
result. Diagnostic prose such as `because` accompanies it without deciding enabledness or a verdict.

The export defines a value encoding and a projection from backend states and transitions to those
Umpire states and Steps. Compatibility requires:

1. **Initial correspondence.** Every allowed initial checking state has a backend representation,
   and every backend initial state projects to an allowed one.
2. **Transition correspondence.** Every allowed Umpire Step has an exported representation; every
   exported real transition projects to an allowed Step with the same result and atomic effects.
   Export-only stutters project to no Step and affect no Umpire bookkeeping.
3. **Observation correspondence.** The same Property triggers and monitor evaluation points read
   the same values and reach the same verdicts, including unread and already-violated state.
4. **Scope correspondence.** The same starts, schedules, terminal conditions, assumptions, and
   semantic bounds admit the same projected Traces. Progress additionally requires the same
   enabledness, counted steps, and treatment of fair continuations.

Thus equality is about admitted labeled Traces after the declared projection, not merely reachable
product states. Two graphs can reach exactly the same states while disagreeing on which action
emitted a Fact, how many times it emitted it, or whether an obligation expired.

Artificial stuttering needs special care for bounded claims: the Umpire counter advances on real
Steps only, and a finite backend prefix containing artificial stutters may contain fewer modeled
Steps than its length suggests. An infinite sequence containing only artificial stutters must not
be decoded as a Umpire fair non-progress cycle or deadline witness. The backend's temporal wrapper
must state how such behaviors are treated; `Init /\ [][Next]_vars` alone does not encode all of
Umpire's progress and completion rules. This is an encoding obligation, not permission to add
fairness to the Model. See [Lamport on stuttering](https://lamport.azurewebsites.net/tla/rhtml/stuttering-step.html).

This exact correspondence is stronger than the one-way inclusion checked by product refinement.
Keep those two obligations separate even when both use projections and stuttering.

## Reading the existing implementations

Use these anchors to understand the contract before designing an encoding:

| Anchor | What it establishes |
| --- | --- |
| [Shared specification](UMPIRE4_SPEC.md) | Authority, explicit Limits, deterministic Artifacts, Evidence rules, and execution boundaries; pending amendments are marked. |
| [IR semantics](../model/scalav2/SEMANTICS.md) | Values, evaluation order, catalogs, Steps, channels, monitors, assumptions, refinement, holes, and Claims. |
| [IR schema](../proto/internal/temporal/server/api/modelir/v1/ir.proto) | Serialized declarations and source positions; schema presence alone does not establish evaluator support. |
| [IR admission](../model/scalav2/goir/load.go) and [row derivation](../model/scalav2/goir/machine.go) | Current validation and derivation of classes, results, channel behavior, holes, and refinement. |
| [Go search](../model/go/umpire/search.go) and [monitors](../model/go/umpire/monitor.go) | Query state identity, trigger bookkeeping, evaluation points, and bounded search outcomes. |
| [Go refinement](../model/go/umpire/refine.go) and [composition](../model/go/umpire/compose.go) | Visible-result projection, carried steps, stutters, member interleaving, and synchronized results. |
| [Go progress](../model/go/umpire/progress.go) and [replay](../model/go/umpire/replay.go) | Separate deadlock, fair-cycle and deadline findings, work accounting, and Query-aware witness validation. |
| [Activity and Nexus specimens](../model/scalav2/specimens/README.md) | Concrete Temporal questions, proposed declarations, trace oracles, and required observations. |
| [Existing Quint model](../model/quint/umpire.qnt) | A comparison prototype; it is not evidence of an IR exporter or complete cross-backend conformance. |

These files are being developed together. In particular, the IR semantics' historical “What goir
implements” section and specimen support notes can lag working-tree code. Current Go code already
contains channel and hole derivation and broader admission checks. Retaining a declaration in the
IR or reading its metadata is still different from evaluating it in a Query. Record backend support
per construct with executable evidence rather than treating either an old status paragraph or a
successful load as proof of full support.

## The existing representation is a useful starting point

Our machines compute a list of Steps from a state and action class. Each Step contains the next
state, outcome, and Facts. An empty list disables that action; several Steps describe alternative
successors. This maps naturally to a relation between old and new state, with an explicit encoding
of the action inputs and outputs.

Preserve the semantic alternatives, rather than the order in which Go enumerates them. Source
locations, Definition IDs, and Behavior Fingerprints accompany the export and the decoded witness.
The explanatory `because` string can remain diagnostic metadata.

The exported state must include everything required to distinguish future behavior and Property
results. That includes channel contents and passive monitor state, and may require auxiliary state
for step results. Two executions that owe different obligations must not collapse into one state
because their product phase happens to be equal.

## Semantic contract

| Concept | Required mapping |
| --- | --- |
| Initial state | Preserve every allowed start and monitor initialization. |
| Atomic transition | Compute one complete successor from the old state. Updates occur together. |
| Nondeterminism | Preserve every permitted result and input choice, without introducing first-result selection. |
| Disabled action | Preserve absence of successors. Keep it distinct from a real step that leaves product state unchanged. |
| Unchanged fields | Explicitly retain their old values; unconstrained next-state fields must not introduce extra behavior. |
| Composition | Preserve member interleaving and synchronized atomic steps, including their result combinations. |
| Channel | Preserve capacity, ordering, multiplicity, loss, and duplicate-delivery limits. |
| Property | Preserve its predicate, trigger, evaluation point, and relevant history. |
| Assumption | Preserve its meaning and exact action scope; never invent fairness. |
| Values | Preserve structural equality, constructor identity, arithmetic, and collection semantics. |
| Query scope | Preserve the Scenario, initial selection, action restrictions, ending condition, and semantic bounds. |

The exporter should report this contract as an explicit support profile. A Model that compiles in
Scala may still contain semantics a particular backend cannot check.

## Translation traps to resolve explicitly

### Stuttering, events, and deadlocks

TLA+ behavior specifications permit stuttering, which leaves the selected variables unchanged.
This supports reasoning about implementation steps hidden by an abstraction.
[Lamport on stuttering](https://lamport.azurewebsites.net/tla/rhtml/stuttering-step.html).

Umpire also has real steps that leave product state unchanged while producing Facts, outcomes, or
monitor updates. Preserve those effects. An artificial TLA+ stutter must not emit another Fact,
advance a monitor, consume a Query step, or charge a modeled deadline.

If step results are encoded as auxiliary state, distinguish repeated occurrences of the same event.
Keeping only the last event's value can make two identical consecutive emissions indistinguishable
from no new emission. Specify the occurrence encoding and its projection back to Umpire Steps.

Check deadlock against the enabled real transition relation and the declared ending conditions.
Allowing stuttering everywhere must not hide a nonterminal state with no permitted real successor.
Likewise, valid termination must not become an accidental liveness failure because the backend
represents behaviors as infinite sequences.

### Bounded progress and fairness

Our Progress declaration says that a destination follows within a specified number of steps. An
ordinary temporal “eventually” formula has a different meaning. Weak fairness permits arbitrarily
long finite delays and therefore does not imply a particular deadline.
[TLA+ temporal reasoning and fairness](https://lamport.azurewebsites.net/tla/book-02-02-27.pdf).

Export bounded progress through explicit finite obligation state or an equivalent bounded encoding.
Define which real steps advance each obligation and which transition starts or discharges it.
Preserve the distinctions among deadlock, a fair non-progress cycle, deadline violation, and an
unfinished search. A finite open prefix cannot become evidence of unlimited non-progress.

Fairness granularity must match the IR. Our assumptions refer to action classes. Fairness for each
class is different from fairness for a disjunction of all classes, which can permit one class to
starve while another keeps running. Preserve the declared class-level obligations and enabledness.
Quint exposes distinct weak and strong fairness operators mapped to TLA+.
[Quint fairness](https://quint.sh/docs/lang#fairness).

Any future support for unbounded temporal claims needs its own specification decision, including
compatibility with Umpire's bounded-progress rules. Export must not silently introduce that support.

### Ordered expressions and definedness

Scala pattern matching uses the first matching branch. Translate overlapping branches to ordered
conditionals or another demonstrably equivalent encoding. TLA+ `CASE` must not accidentally choose
among several true branches where our source requires priority. Quint recommends conditional chains
for this reason. [Quint's discussion of cases](https://quint.sh/docs/lang#cases-removed).

Preserve short-circuit behavior where it affects whether a hole or invalid expression is reached.
Logical equivalence on total Boolean values is insufficient when one operand can be undefined.
Guarded evaluation or explicit definedness state must retain that distinction.

### Holes, errors, and bounds

Our IR distinguishes a disabled action, a reachable semantic hole, and malformed input. A hole
makes affected checking incomplete. Exporting it as `FALSE` removes behavior and can manufacture a
passing result. Initially refuse Models containing unsupported holes, or define an explicit hole
encoding that prevents an unqualified verification result.

State-domain violations must also preserve their meaning. For example, a send that overfills a
channel currently produces an out-of-domain result. Silently disabling it would change the Model.
Either reject such a Model during admission or represent the corresponding fault explicitly.

Keep semantic bounds separate from exploration resource limits. A modeled deadline affects behavior;
a search-work limit affects whether the checker completed. Backend truncation cannot establish
absence of a counterexample. Arbitrary state constraints must not prune unwanted transitions and
then be presented as verification of the original Model.

### Integers and collections

Choose one numeric meaning for the exportable subset. The current Go evaluator uses `int64`
arithmetic, so mathematical-integer export needs either a checked no-overflow domain or an explicit
encoding of machine arithmetic. Prefer mathematical integer semantics with bounded state domains
as a design direction, but specify the interpreter change before relying on it.
See [the evaluator](../model/scalav2/goir/eval.go).

Keep ordered lists distinct from sets. An unordered channel can contain several equal deliveries;
represent it as a multiset or an equivalent canonical sequence, preserving multiplicity and retry
counts. A set of messages would lose duplicate deliveries. Define tagged constructors so values
from different enum cases cannot collide in the exported encoding.

## Proposed export subset and workflow

Start with finite types, pure expressions, explicit starts, atomic steps with finite nondeterministic
alternatives, and safety Properties. Add channels, passive monitors, composition, and bounded
progress when their complete semantics and conformance examples are available. Fairness and holes
require their own declared support; accepting their syntax is insufficient.

Two export routes are worth comparing on the same small Model:

```text
admitted IR -> generated TLA+ -> external checker
admitted IR -> generated Quint -> TLA+ compilation -> external checker
```

Select the route based on complete translation support, diagnostics, generated-model inspectability,
and witness decoding. An intermediate compiler becomes part of the trust basis and needs its version
recorded. Successful generation of a `.tla` file is only an export result, not a verification result.
Quint documents a TLA+ compilation target in its [CLI reference](https://quint.sh/docs/quint).

A small enumerated-table exporter can bootstrap checking of the transition graph. It cannot
independently validate Go's derivation of that graph if both paths consume the same Go table. A
later expression-based exporter should derive behavior from the IR independently and compare it
against Go over the same finite domain.

Keep the public export interface small. It consumes admitted IR, a selected Query, and an explicit
backend profile; it produces the specification, checker configuration, provenance, and a decoder
mapping. Internal encodings and backend-specific normalization stay inside that module.

## Agreement checks

For small finite Models, require equality under the declared value encoding of:

- Initial states and monitor initialization.
- Enabled action classes and complete successor results, including outcomes and visible Facts.
- Reachable states, after accounting for declared auxiliary state and artificial stuttering.
- Property activation, monitor evaluation points, and safety results.
- Terminal states and nonterminal deadlocks.
- Bounded obligation behavior, including reset and expiry ordering.

An external counterexample must decode to valid IR Steps and replay through the Go semantics.
Different exploration algorithms may choose different witnesses or visit different numbers of
states. Compare their semantic answers and validate each witness, rather than requiring the same
search order or shortest trace.

Include negative controls for an overlapping pattern, two enabled alternatives, simultaneous field
updates, repeated equal events, an output-producing product stutter, distinct monitor histories,
duplicate deliveries, overflow, a reachable hole, terminal stuttering, and class-level fairness.
Each control should expose the specific mistranslation it protects against.

These checks establish agreement on the declared finite scope. They do not prove arbitrary compiler
correctness or conformance of the production Temporal implementation.

## Reporting and known bugs

Keep export admission, checker completion, semantic findings, and reporting severity separate.
Record Model fingerprints, selected module variants, assumptions, semantic bounds, backend/toolchain
versions, and unsupported constructs with every result.

The [Known Bug lifecycle](../model/scalav2/specs/KNOWN_BUG.md) applies after semantic evaluation.
An acknowledged violation must still violate the exported Property and retain its counterexample.
The reporting layer can classify it as an active-known-bug warning; marking it fixed makes recurrence
an error. Export must not remove the transition or weaken the Property to make a known bug pass.

## Decisions still needed

Settle the numeric semantics, observable occurrence encoding, counted-step definition, and initial
hole policy before claiming export compatibility. Then choose one activity or Nexus slice with
finite parameters and negative controls, derive it through both Go and the export path, and measure
agreement and diagnostic quality. Broaden the subset only when a concrete Temporal question needs it.
