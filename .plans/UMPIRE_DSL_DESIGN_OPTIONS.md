# Umpire DSL evolution and Veil reuse: design options

Research date: 2026-09-06. This is an assessment of alternatives, not an implementation decision.
The user explicitly allows the Umpire DSLs to change. Existing language and finite-model rules
are therefore migration constraints to evaluate, not reasons to reject a better design.
No model, runtime, dependency, or normative specification was changed for this investigation.

> The Umpire3 sources this assessment cites were removed by fn-81. They are named below as
> provenance rather than as links.

Supporting investigations: [Veil implementation details](VEIL_BACKEND_RESEARCH.md) and
[DSL opportunities with conceptual examples](UMPIRE_DSL_OPPORTUNITIES.md).

## Assessment

Veil can plausibly run under Umpire and avoid building substantial verification infrastructure.
The strongest expected savings are in symbolic transition reasoning, inductive-invariant checking,
verification-condition generation, solver integration, proof reconstruction, and counterexample
inspection. Those are primarily future costs avoided. They are not evidence that importing Veil
would delete most of today's Umpire code.

For the five-state Nexus3 draft, a finite checker is straightforward. Veil becomes more compelling
for richer data, interacting operations, and parameterized protocols. Performance and net code
savings have not been measured. Source-level feasibility does not establish compatibility.

The most promising direction is a coherent typed authoring language with distinct kinds of
declarations, a shared model representation, a small monitorable property fragment, and Veil as
a verification engine. The grammar and current internal data structures can both change.

## What the current code actually requires

- [Core.lean](../model/Umpire/Core.lean) defines `TransitionKernel` using executable initial/step
  lists, authoritative relations, and soundness/completeness proofs. `Vocabulary` also
  enumerates states, actions, outcomes, and facts.
- [Target/Language.lean](../model/Umpire/Target/Language.lean) requires complete finite domain
  evidence at admission. `describeBehavior` visits the state/action product and materializes
  transition rows for canonical behavior metadata and fingerprints. A symbolic search adapter
  alone would not remove this admission cost.
- [Planning/Engine.lean](../model/Umpire/Planning/Engine.lean) exposes an indexed finite kernel
  tied by proofs to the Target. Its private backend is a candidate-pull interface. A symbolic
  decision engine does not naturally satisfy that interface without enumeration or redesign.
- [Behavior/Language.lean](../model/Umpire/Behavior/Language.lean) already supports allowed and
  forbidden actions, occurrence bounds, partial orders, sequences, adjacency, and exact traces.
  Friendlier scenario composition can expose existing semantics; not all such features require
  a new search engine.
- [Property/Evaluation.lean](../model/Umpire/Property/Evaluation.lean) has an executable evaluator,
  denotational semantics, and `evaluateProperty_agrees`. A replacement must preserve the assurance
  supplied by these definitions rather than discard it as boilerplate.
- [Case/Compiler.lean](../model/Umpire/Case/Compiler.lean) assembles already-lowered monitors.
  It does not translate the full Property language. Exact Property-to-Contract lowering is still
  an important cost to address. The broader claim in `Umpire/ARCHITECTURE.md` is not supported by
  this implementation.

The older Umpire3 semantic binding (`Umpire3/Veil/Semantics.lean`) and Nexus concrete
correspondence (`Temporal/Families/NexusCancellation/Targets/Veil/SoundConcreteSemantics.lean`),
both under the `tools/umpire3` tree fn-81 removed,
show a real local precedent for using Veil. They also show the cost of separate representations:
initial-state, transition, property, encoding, and coverage correspondence all need evidence.
These files were inspected, not built. They do not establish compatibility of current Umpire,
and this assessment does not recommend restoring retired execution formats or runtimes.

## Three ways to incorporate Veil

At inspected Veil commit `be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd`, there are real reusable
`RelationalTransitionSystem` and `EnumerableTransitionSystem` types, an action computation
`VeilM`, and a concrete checker interface. The symbolic trace commands are more tightly coupled
to Veil's generated module declarations and elaboration context. Importing the semantic type
alone does not automatically expose all DSL automation to arbitrary Umpire data.
[Transition-system source](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/TransitionSystem.lean),
[symbolic trace implementation](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Symbolic/TraceLang.lean).

| Option | Arrangement | Savings and costs | Assessment |
| --- | --- | --- | --- |
| Optional checker | Existing Umpire model produces a checked Veil view | Adds symbolic checks; retains current finite admission, planner, and translation maintenance | Smallest adoption experiment; limited immediate deletion |
| Shared relational core | Model meaning is relational; executable finite search and Veil verification consume the same meaning | Can remove mandatory full enumeration and duplicate transition semantics; changes admission, fingerprints, planning capabilities and evidence | Best candidate to investigate for long-term simplification |
| Veil-centered authoring | Veil declarations own model behavior; Umpire adds scenarios, properties and runtime bindings | Reuses more frontend/action infrastructure; couples authoring to Veil and requires a reliable path from its definitions to portable artifacts | Valid alternative worth comparing, not rejected merely because today's DSL differs |

A shared relational core does not mean an arbitrary Lean proposition becomes automatically
enumerable, SMT-translatable, serializable, or executable. Each interpretation needs explicit
support. A restricted typed representation with a relational interpretation is one approach;
reusing Veil-generated definitions is another. The experiment must establish which approach
actually avoids duplicate models.

For symbolic models, replace mandatory full-table identity computation with a deliberately
specified fingerprint of normalized checked declarations and their behavior-affecting dependencies.
That fingerprint need not decide semantic equivalence between arbitrary programs. Source location
and comments should remain irrelevant, while changes to called behavior must remain visible.
This is a design change requiring explicit compatibility treatment, not an existing capability.

## The verification integration should answer questions directly

The current candidate-pull seam is useful for deterministic finite exploration. A symbolic checker
should instead receive the checked model, scenario, property, semantic bounds, and backend work
budget, and return a typed answer with its assumptions and assurance method.

A basic visited-state reachability search is not interchangeable with enumeration of all traces.
For history-sensitive requirements, its state must also retain relevant scenario progress,
property-obligation state, and counting coordinates. Otherwise merging two paths that reach the
same product state can discard a relevant distinction. Terminal model states must remain valid
terminal states rather than accidentally become deadlock failures under backend defaults.

For bounded trace checking, the conceptual formulas are:

```text
Allowed(trace) = Model(trace) AND Scenario(trace) AND SemanticBounds(trace)

scenario satisfiable:  EXISTS trace. Allowed(trace)
property witness:     EXISTS trace. Allowed(trace) AND Property(trace)
counterexample:       EXISTS trace. Allowed(trace) AND NOT Property(trace)
```

Universal verification needs both a nonempty scenario and absence of counterexamples under the
declared semantics. Solver timeout/unknown is not absence. A resource budget is not part of
`Allowed`; hitting it leaves the question unanswered.

An invariant proof may establish a stronger, unbounded reachability theorem under declared
assumptions, but that is a separately named claim. A failed inductiveness check may start from an
unreachable state and is not automatically a reachable system counterexample.

If translation deliberately overapproximates behavior, proving a universal property can transfer
only with the correct inclusion and property-preservation argument. A satisfying solver trace
may be spurious. Exact canonical replay validates an individual witness; it cannot validate an
unsatisfiability claim or repair a translation that excluded real behaviors.

Solver-selected witnesses also need a deliberate determinism policy. Replaying and canonically
serializing one witness does not make the solver select the same witness on a later run.
Keep deterministic regression selection separate, or establish canonical witness selection and
pin solver configuration as part of the reproducibility contract.

## Opportunities to change authoring

### One expression language, explicit contexts

Borrow Quint's mode distinction while retaining Lean's types. Let state predicates, transition
predicates, trace requirements, and scenario expressions share typed values and familiar operators.
Reject context mistakes at elaboration. Separate meanings do not require duplicated expression
grammars or a separate framework for every declaration kind.

Authors could write `result.state == cancelRequested` instead of assembling IDs, field tags and
encoded literals. Such syntax must elaborate to a typed representation. Arbitrary Lean predicates
may be useful for model-only proofs, but that does not grant them a runtime-monitor interpretation.
Expose checker and monitor support explicitly and reject unsupported requested interpretations.
[Quint modes](https://quint.sh/docs/lang).

### Make input/output ownership part of the model

The draft already explains that waiting does not cause completion. A stronger design can encode
this distinction: model transitions describe commands and system-produced events, while a scenario
can wait for evidence of an event. A completion transition can then occur independently of whether
the controller has begun observing it.

This is a semantic change to evaluate, not a renaming. It affects interleavings, query counts and
how request/acknowledgment pairs become model steps. An output-labeled transition is still selected
by the model checker during exploration; that does not make it controllable by a live test driver.
Borrow input/output roles from model-based testing and passive observers from P and Ivy.
[P monitors](https://p-org.github.io/P/manual/monitors/),
[Ivy interface specifications](https://microsoft.github.io/ivy/examples/testing/specification.html).

### Scenario combinators with precise trace semantics

Expose named occurrence ordering, choice, bounded repetition, and explicitly defined interleaving.
Keep exact sequences as a useful special case. Specify whether sequencing means adjacent steps or
merely ordered occurrences with intervening activity; use distinct forms when both are needed.

For Nexus, there are different questions about cancellation after acknowledgment, completion before
cancellation, and resolution after cancellation. Broadening a scenario cannot invent transitions
missing from the model. Start by expressing the existing two-outcome race more clearly, then add
the additional behavior deliberately.

Quint's runs offer readable composition. Choreo offers an additional example of an authoring layer
over a lower-level modeling language. Borrowing those ideas does not imply copying Quint's run
evaluation rules verbatim or importing its runtime into Lean.
[Quint language reference](https://quint.sh/docs/lang).

### Properties as scoped obligations with one semantic compiler

Represent a bounded response using explicit trigger, response, correlation key, counting coordinate,
bound, and endpoint policy. Compile it into an obligation machine. Use that machine's semantics
for finite trace evaluation, the product model checked by Veil, and supported runtime lowering.

For `within 1 operation_transition`, only an admitted step of the same operation advances the
coordinate. A response at the trigger coordinate or the following coordinate is allowed by the
draft. Reaching an ordinary runtime timeout before that semantic evidence arrives is inconclusive.
Closed-model trace endings and incomplete runtime prefixes need explicit, different treatment.

This could reduce future duplication more than changing property syntax alone. It still requires
a proved compiler relationship and a checked evidence projection; the same-looking monitor over
raw events does not imply the same property. P and Ivy supply useful observer/monitor precedents,
not a ready-made implementation of Nexus's scoped finite semantics.
[P monitors](https://p-org.github.io/P/manual/monitors/),
[Ivy monitors](https://microsoft.github.io/ivy/examples/testing/specification.html).

### Queries should report what was exercised

Keep witness and universal check distinct. Report scenario satisfiability, trigger coverage,
property result, and search completeness independently. A nonempty scenario can still avoid the
trigger of a conditional requirement. Existing case-analysis coverage can inform this interface.

Alloy's run/check workflow and ability to fork traces provide useful interaction patterns. A fork
must expose which conditions are fixed and which vary. This is a generated inspection view of the
model, not another editable source of behavior.
[Alloy 6](https://alloytools.org/alloy6.html).

### Make evidence projection a deep module

Treat correlated runtime evidence to semantic steps as a small explicit interface that can retain
partial evidence, emit supported model steps, ignore justified duplicates/unrelated activity, or
report conflicts and gaps. Preserve causal order and supporting event identities.

TLA+ refinement with stuttering is the useful theoretical precedent. Equal abstract state alone
does not imply that no semantic event occurred: a labeled self-loop may count as a transition.
Step-bounded obligations therefore require preservation of their semantic counting coordinate.
[Lamport on refinement](https://lamport.azurewebsites.net/pubs/state-machine.pdf).

## What would remain Umpire-owned

| Responsibility | Expected impact of Veil adoption |
| --- | --- |
| SMT translation, solver invocation and reconstruction | Strong reuse candidate; avoid developing an independent stack |
| Transition/action semantics and generated obligations | Strong candidate if representations are shared; less saving with separate handwritten models |
| Concrete model traversal | Partial replacement possible; scenario/property product state, deterministic selection and limits need adaptation |
| DSL identity and source diagnostics | Derive more automatically, but Umpire's identity/provenance policy remains |
| Scoped property semantics and monitor lowering | Umpire still supplies them; common obligation semantics can reduce internal duplication |
| Runtime evidence correlation | Umpire/Temporal still supplies it |
| Case schema, authorization, immutable Runs, cleanup and Go SDK execution | Remain outside Veil |

## A decisive comparison

Compare the optional-checker and shared-core/Veil-centered arrangements on the same small Nexus
example. Do not write a separate model for each checker by hand. Include both terminal outcomes,
a wrong request transition, an impossible scenario, an unexercised property trigger, a truncated
pending response, duplicate/unrelated evidence, and a labeled self-loop. Add a small multi-operation
example so the scoped counting requirement is actually exercised.

Measure authored declarations and proof obligations, adapter code, cold/warm build and query cost,
proof reconstruction support, deterministic artifacts, and per-feature maintenance. Record code
actually removed separately from new verification machinery no longer needed. The success criterion
is one model, preserved outcomes/claims, and less total maintenance—not just a green solver result.

This research did not run that experiment. There is enough evidence to justify it, but not enough
to assert a percentage reduction or recommend an immediate wholesale migration.
