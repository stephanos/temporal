# Umpire DSL experiment

Date: 2026-09-06. This is the decision record for isolated executable experiments under
[`experiments/umpire-dsl/`](../experiments/umpire-dsl/). It does not change the normative Umpire
specification or authorize a production migration. No Flow-Next state is used by this work.

## Intent and baseline

Determine which authoring improvements are useful independently of the checker, and whether Veil
reduces total model and verification maintenance. Use one cancellation example with both terminal
outcomes, then two operations with independent obligation clocks. Compare executable semantics,
error behavior, and the cost of a subsequent edit; a successful solver invocation alone is insufficient.

The source baseline inspected for this experiment is more capable than “no DSL,” but narrower than
the proposed cancellation language:

- [`Nexus.lean`](../model/Temporal/Feature/Nexus3/Nexus.lean) implements only
  `scheduled → started → succeeded`, using `awaitStart` and `awaitSuccess`. Its custom blocks
  compile through [`Syntax.lean`](../model/Temporal/Feature/Nexus3/Syntax.lean) and
  [`Authoring.lean`](../model/Temporal/Feature/Nexus3/Authoring.lean), which deliberately admit a
  fixed canonical success table. Cancellation and scoped progress are explicit Known Gaps.
- [`Nexus.md`](../model/Temporal/Feature/Nexus3/Nexus.md) and
  [`Integration.md`](../model/Temporal/Feature/Nexus3/Integration.md) describe the larger draft.
  Their cancellation syntax, SDK cancellation capability, and scoped monitor lowering are not
  implemented by the success slice. They are requirements to test, not working baselines.
- [`BehaviorDeclaration`](../model/Umpire/Behavior/Language.lean) already represents required,
  allowed, and forbidden actions, occurrence bounds, ordering, sequences, adjacency, exact action
  lists, and exact traces. A nicer frontend need not replace these semantics.
- [`Property/Evaluation.lean`](../model/Umpire/Property/Evaluation.lean) has executable and
  denotational meanings connected by `evaluateProperty_agrees`. A prototype's own monitor agreement
  is not a substitute for correspondence to this existing evaluator.
- [`Case/Compiler.lean`](../model/Umpire/Case/Compiler.lean) explicitly consumes already-lowered
  monitors; it has no checked-Property-to-lowering producer. This is a remaining implementation
  boundary, not functionality that Veil adoption can simply delete.
- The production Lake workspace uses Lean `v4.33.1` and Batteries `v4.33.0`. Separate experimental
  manifests must prevent checker dependencies or builds from changing that workspace.

The prior [research](UMPIRE_DSL_RESEARCH.md), [options](UMPIRE_DSL_DESIGN_OPTIONS.md), and
[opportunities](UMPIRE_DSL_OPPORTUNITIES.md) motivate hypotheses. Their old Umpire3 correspondence
examples and inspected Veil revision establish source precedent, not current compatibility,
performance, or measured savings. The experiments must supply that evidence.

## Independent authoring choices

“After” below describes the experimental contract, not approved production syntax.

| Opportunity | Status quo / before | Proposed after and expected benefit | Cost, decisive question, and decision gate |
| --- | --- | --- | --- |
| 1. Input/output ownership | Executable success actions are `awaitStart` and `awaitSuccess`; cancellation draft combines request and wait actions. | `submitCancel` is a controllable input; correlated cancellation confirmation and terminal outputs advance semantic state. Polling does not cause progress. | Changes Target vocabulary, trace identities, and Case bindings. Adopt the ownership distinction if acknowledgement alone cannot advance state and both terminal outcomes remain model-owned; reject any design where the scenario chooses the live output. |
| 2. Scenario constraints and finite algebra | Current Nexus3 exposes `actions exactly`; core declarations already support much richer conjunctive constraints. | Expose required occurrences, ordering, bounds, and adjacency with typed constructors; retain exact replay. | First preserve existing checked meaning and fingerprints. Adopt that surface if adding unrelated interleaving needs no scenario rewrite. Defer true union/repetition until finite normal forms, emptiness, and identity are implemented; reject priority-choice masquerading as union. |
| 3. Typed modes and capabilities | Frontend hides much ID plumbing but is specialized to the fixed success shape; lower-level declarations carry IDs and encoded values. | Distinct input, output predicate, scoped response clause, scenario, and query types. | More constructors and source diagnostics; no new runtime capability follows from a Lean type. Adopt small types if misuse fails at the author expression; defer a general parser until it improves real examples. |
| 4. Scoped obligations | Draft describes `for operation` and `within 1 operation_transition`; success slice reports scoped progress unsupported. | Each trigger captures a key and coordinate; only that key's admitted labeled steps tick its independent obligation. | Requires explicit endpoint and multiple-trigger policies. Adopt if interleaved operations, duplicates, counted self-loops, and late responses have the specified outcomes; reject state-inequality or raw-event clocks. |
| 5. Query validity | Umpire requires satisfiability; a conditional property can still hold without exercising its trigger. | Report satisfiable, exercised, answered, and search completeness separately; nonvacuity is query policy. | Additional result states and coverage tracking. Adopt if impossible scenarios, absent triggers, and work exhaustion remain distinct; reject any green result based on empty search or timeout. |
| 6. Shared passive monitor | Property evaluation exists; Case compiler accepts prebuilt rules; general faithful lowering remains missing. | Compile a supported response clause once and reuse its transition semantics for model traces and online/offline evidence evaluation. | Compiler correspondence and portable IR admission remain necessary. Adopt the small semantic module if chunked and whole-stream execution agree and unsupported forms reject; do not claim production lowering from this standalone test. |
| 7. Checked evidence projection | Integration draft specifies correlation and confirmation; current success surface does not implement the cancellation bridge. | Total `stutter`, `emit`, or `reject` result with operation identity and causal support. | Deduplication identity, partial evidence, ordering, and projection version need explicit policies. Adopt the boundary if unrelated/duplicate evidence stutters and conflicting or unsupported relevant evidence rejects; reject emitted outputs without support. |

A concrete authoring comparison should keep the actual success example alongside an experimental
cancellation clause: “after confirmed cancellation for operation A, observe canceled or completed
within one further admitted transition of A.” The latter adds behavior; it is not evidence of fewer
lines for an equivalent existing production feature. An ordinary observer timeout closes a pending
runtime prefix inconclusive. A deliberately closed model trace uses its declared finite endpoint
policy. Neither proves unlimited eventual completion.

## Temporal logic and developer-facing syntax

Temporal logic is already part of the core representation:
[`Property/Language.lean`](../model/Umpire/Property/Language.lean) includes `eventuallyWithin`,
`quiescentWithin`, and guarded variants. The executable semantics and `evaluateProperty_agrees`
are in [`Evaluation.lean`](../model/Umpire/Property/Evaluation.lean). Existing temporal capability
therefore must not be counted as newly supplied by the experiment. The
[property compatibility table](../model/Umpire/Property/COMPATIBILITY.md) distinguishes model
interpretation from the narrower runtime lowering surface.

Evaluate a small, readable finite temporal surface over the same scoped response clause:
`whenever … eventually … within … operationTransitions`. Its intended reading is “every matching
trigger starts an independent same-operation response obligation with this semantic bound.” This
is a typed spelling of that supported clause, not a full LTL implementation or an additional source
of monitor behavior. The experiment should check that it elaborates to exactly the constructor-built
clause and rejects the wrong mode or clock, while both forms run through the same evaluator.

Compare three developer experiences: today's checked clause constructors; typed semantic
constructors exposing trigger, response, key, clock, and bound; and the small temporal notation.
Prefer the notation only where it makes those choices easier to read without hiding them. Retain
explicit endpoint policy: a pending incomplete runtime prefix is inconclusive, while a closed
finite model trace follows its declared closure semantics. General LTL operators, unbounded
liveness, fairness assumptions, and arbitrary temporal formulas remain deferred until their
interpretations, proof methods, and runtime support are specified independently.

## Independent verification choices

| Architecture | What it brings beyond current finite checking | Costs retained or introduced | Experiment and adoption gate |
| --- | --- | --- | --- |
| Status quo finite | Complete bounded enumeration with current checked Target, Property semantics, deterministic selection, and identity machinery. | Finite domain admission and table materialization; growth with interacting state and trace dimensions. | Establish exact allowed traces, both terminal outcomes, query statuses, and repeatable output. Keep as the default unless a measured alternative earns migration. |
| Optional Veil adapter | Potential symbolic reasoning, solver infrastructure, and reconstructed invariant proofs without replacing public model authority. | Finite admission remains; initial/step/property translation, completeness, and exact replay need checked links. | Derive a view from the same model and demonstrate agreement. Adopt only supported claims with explicit assurance; a direct semantic-type import is not a symbolic integration. |
| Shared relational core | Potentially one authoritative transition meaning for finite enumeration and Veil; could avoid mandatory full enumeration for symbolic families. | Redesign admission, capability negotiation, canonical identity, and planner interface; prove each interpretation faithful. | Generate both consumers from one restricted representation and perform a transition edit once. Defer production migration until serialization/fingerprints and scenario/property product semantics also work. |
| Veil-centered authoring | Potential reuse of action frontend, generated transition infrastructure, and verification commands. | Authoring/toolchain coupling; extraction into portable Umpire artifacts; distinct Property/Behavior/Query roles and runtime bindings still needed. | First determine whether generated definitions feed the same canonical example without a second handwritten model. Defer adoption if extraction is unproven; do not reject merely because spelling differs. |

A bounded checker must carry scenario progress, obligation state, and counting coordinates when
merging states. Terminal lifecycle states are valid endpoints. Solver unknown or budget exhaustion
is unanswered. Replay can validate an individual candidate; it cannot justify an unsatisfiability
claim from an incomplete translation. An inductive-invariant proof is a separately named claim,
not a stronger label pasted onto a finite experiment.

## Executable comparison and measurements

The required case set is: canceled terminal outcome, completed terminal outcome, wrong request
transition, impossible scenario, absent property trigger, truncated pending runtime prefix,
duplicate evidence, unrelated evidence, conflicting evidence, counted self-loop, and two concurrent
operations with independent clocks. Include a response exactly at its bound and a response too late.

Use the same authoritative model data for finite and relational views. A checker requiring a second
manually maintained transition table fails the maintenance objective even if both tables agree today.
Keep each package's build entrypoint, toolchain, and manifest separate. Production packages must not
import the experiment. There is no live Temporal execution or Case promotion in this experiment.

| Measure | Collection method | Interpretation limit |
| --- | --- | --- |
| Authoring and proof effort | List actual declarations, correspondence theorems, adapter modules, and trust dependencies. | Source size is descriptive, not a developer-time measurement. |
| Subsequent edit | Add a labeled counted self-loop or transition and one scoped clause; record all touched semantic authorities and adapter changes. | A synthetic edit indicates coupling, not long-term team productivity. |
| Build/query cost | Record exact command, revision/toolchain, cache state, elapsed time, and outcome. Distinguish cold build, warm build, and query. | One tiny example cannot establish scalability or a percentage saving. |
| Diagnostics | Execute type errors and semantic failures; retain distinct statuses and useful origin/support data. | Constructor type safety alone does not establish editor UX for a future grammar. |
| Reproducibility | Repeat the same bounded evaluation and compare its serialized/printed result. | Repeatability of the toy output does not establish production artifact identity or solver witness determinism. |
| Reuse | Record production code removed separately from future machinery potentially avoided. | This isolated experiment removes no production code. |

## Results and decisions

### Current implementation measurements

Read-only focused checks ran from `model/` with existing compiled dependencies on 2026-09-06,
using Lean `v4.33.1`. The checkout HEAD was `fd0470456a3b51f8d05daba6f7df2b1067adbc87`, but
Nexus3 files already had working-tree changes, including an untracked `Syntax.lean`; this is a
measurement of that current working tree, not a pristine commit benchmark. Neither command passed
an output path or rebuilt the production workspace. Timings come from `/usr/bin/time -p` and are
single cached-dependency runs, not cold builds or isolated query timings.

| Command | Exit / wall time | Evidence |
| --- | --- | --- |
| `lake env lean Temporal/Feature/Nexus3/Tests.lean` | 0 / 1.58 s | Existing success witness, malformed-model/sequence rejection, identity, and Known Gap checks pass. Printed model/lifecycle axiom inventories contain `propext`; admission/query inventories also contain `Classical.choice` and `Quot.sound`. |
| `lake env lean Umpire/Property/Tests/LogicalTime.lean` | 0 / 0.53 s | Existing bounded eventuality/quiescence checks pass, including absent, malformed, and decreasing logical-time evidence. These are existing `native_decide` tests, not newly established axiom-free semantic correspondence. |

The current Nexus3 source has 89 lines in `Nexus.lean`, 455 in `Authoring.lean`, 130 in
`Syntax.lean`, and 314 in `Tests.lean` (988 total, counting comments and blanks). This inventory
shows a compact author surface supported by substantially more admission/frontend/test code; it
is not an estimate of avoidable boilerplate. The shared Property language/evaluator have 336/1708
lines and the Behavior language 963 lines; they serve broader functionality than this prototype.

A read-only edit-coupling inspection finds that adding cancellation to the current success slice
would require more than one new model row: the surface parser enforces the fixed success shape,
`SuccessModel` and its canonical table/admission law encode that same shape, and the vocabulary,
property/behavior constructors, and tests are specialized accordingly. No production edit was
performed to measure developer time. This is evidence for evaluating a general typed frontend,
not evidence that all these supporting checks can be removed.

### Differential check against the current Property evaluator

[`Baseline.lean`](../experiments/umpire-dsl/Baseline.lean) imports the actual cached
`Umpire.Property.Tests.Fixtures` and checks declarations through `checkProperty` before running
`evaluatePropertyOnTrace`. It compares the experiment's closed-model verdict with the conjunction
of two production `eventuallyWithin` evaluations, one per operation. The fixture bridge filters
semantic steps by operation and encodes matching trigger/response predicates as declared fixture
observations; the production clause then counts `.semanticTransitions`. This explicit assumed
bridge does not establish production correlation, generic scoped lowering, or runtime refinement.

`/usr/bin/time -p experiments/umpire-dsl/run-baseline.sh` passed **3,510 comparisons in 53.78 s**
(exit 0; user 52.85 s, system 0.34 s). The domain is all 585 event words of lengths 0–3 over four
events and two operation keys, at bounds 0, 1, and 2, with two response predicates: canceled-or-
completed, and requested-or-canceled-or-completed (allowing same-step discharge). It includes
absent and repeated triggers, both terminal outcomes, self-loops, interleaving, exact-bound and
late responses. Arbitrary words deliberately include non-model traces: this checks Property
interpretation, not only lifecycle reachability. There are 7,020 production evaluations, each
including property admission; the elapsed time is not a comparative query-performance benchmark.

The script obtains the existing model `LEAN_PATH`, adds the experiment's compiled library path,
and invokes `lean --run` without `-o`. It requires existing caches; it never builds or regenerates
the active production model. Shell syntax and whitespace checks pass. Agreement is finite
executable evidence for closed traces under the fixture projection, not a general theorem,
open-prefix comparison, or full LTL equivalence.

### Experimental semantic results

The isolated core builds with `lake build DslExperimentTests dsl-experiment`. Its executable checks
**449,376 monitor/reference comparisons**: all words of lengths 0–4 over eight operation/event
steps, four trigger predicates, four response predicates, bounds 0–2, and both closure policies.
The independent reference inspects scoped suffixes rather than reproducing monitor state updates.
All comparisons agree. Arbitrary words test the semantic interpreters; consumers must first admit
traces. `evaluateAdmitted` performs shared-model Exact Replay and rejects invalid transitions.

**1,340 evidence variants** also pass: each of 335 valid model prefixes through depth five is
projected normally, reversed, duplicated, and with unrelated noise. The admitted semantic outputs
and verdicts agree. Focused cases cover conflicts, wrong request transitions, missing parents,
causal cycles, transitive causal support, unsupported relevant evidence, and bounded buffering.
The projection caps retained records at 128 and reports incomplete evidence inconclusive.
A rejected append is atomic and preserves previously accepted state and emissions.

The evidence trust boundary is explicit: first accepted normalized envelopes are immutable.
Later conflicting evidence is rejected separately; it does not retract an earlier admitted event
or erase a proved violation. This models append-only Run semantics, not a retractable raw-source
event stream. There is no live Temporal history adapter or production authorization path here.

Queries require an explicit endpoint policy: `closedFinitePrefixes`, `runtimePrefixes`, or
`terminalWorlds`. Each report retains that choice and semantic maximum depth separately from
work-budget completeness. An exhaustive search over runtime prefixes still reports unanswered
when selected prefixes retain unresolved obligations. Impossible scenarios, absent triggers,
violating traces, and work exhaustion have distinct results. `ordered` allows intervening steps;
`either` is union. The small scenario implementation does not supply canonical production identity
or general repetition semantics.

`temporal_surface_agrees` is checked with no axioms: temporal notation elaborates to the same
`ResponseClause` constructor. `monitor_append` and `evidence_append` are audited with only
`propext`. The experiment's declared proof boundary permits standard Lean logical axioms
(`propext`, `Quot.sound`, consistent with the inspected baseline); no custom assumptions,
compiler-trust axioms, or `sorryAx` are accepted in its proof claims. These append/interface
facts and the finite differential corpus do not constitute a general Property compiler theorem.
No portable Case monitor lowering was implemented.

Final verification ran `experiments/umpire-dsl/run.sh --all` successfully: package build,
two identical native core receipts, the production-evaluator comparison, pinned Veil-core
comparison, and temporary transition edit. The independent re-review found no remaining
high-impact correctness defects after the endpoint, atomic-rejection, causal-cycle, and
admission refinements. `lake env lean -DwarningAsError=true DslExperimentTests.lean`, shell
syntax checks, and whitespace checks also pass.

| Final core measurement | Result and scope |
| --- | --- |
| `lake clean` then `/usr/bin/time -p lake build DslExperimentTests dsl-experiment` | 4.79 s wall; clean experiment artifacts, preinstalled Lean standard library, concurrent development. |
| Same build, warm | 0.14 s wall. |
| Native receipt repeatability | Two default executable runs produced byte-identical output. This is not a production fingerprint guarantee. |
| Source inventory | Model 59, Property 142, Projection 171, Query 119 lines; executable/checked/baseline tests 309 lines. Counts include comments/blanks and exclude adapters and scripts; features and assurance differ from the production baseline. |
| Repository `make lint-code GOLANGCI_LINT_FIX=false` | Failed: Go type checking could not create `/var/folders/.../go-build.../b1919/` because the device ran out of space. The command reports a typecheck failure at `tools/umpire/cmd/umpire-check-retired-vocabulary/main.go:1`; its subsequent vet step did not run. No Go files were changed by this experiment, and autofix was disabled. This is an environmental gate failure, not a passing repository lint result. |

The broad Go lint attempt took 38.68 s according to its own execution timer and returned a failing
make status. Its temporary build pressure subsided after exit. Only experiment-created downloads
were cleaned during the earlier Veil failure; no unrelated workspace/cache cleanup was performed.
There were no production edits, Flow-Next operations, commits, or live-service actions in this work.

### Veil representation and subsequent-edit results

The [detailed Veil record](../experiments/umpire-dsl/VEIL_RESULTS.md) pins
`verse-lab/veil@be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd`. The actual upstream
`EnumerableTransitionSystem` and `RelationalTransitionSystem` compile unchanged on Lean 4.33.1
in a narrow package. One shared `advance` definition supplies both the finite consumer and Veil
view. Checked initialization, transition, and state-safety correspondence use only the declared
standard logical axiom boundary. Independently traversed path lists agree at depths zero through
five (1, 2, 8, 24, 84, 216 paths); all 335 paths pass Exact Replay.

This is a successful representation adapter, not an invocation of Veil's BFS, symbolic checker,
or proof reconstruction. The initial core compilation took 6.57 s; narrow package builds took
1.37 s then 0.14 s warm; `lake exe probe` took 0.48 s including startup. These are individual
local runs under concurrent development. Full dependency update failed after 223.96 s because
its mathlib cache expected Lean 4.32.0. A source build at 4.33.1 failed after 27.17 s on Batteries
API incompatibility and then disk exhaustion. Installing matching 4.32 also ran out of disk.
Full-checker compatibility and solver claims remain unestablished; a matching-version build was
not shown to be semantically or technically impossible.

The executed subsequent-edit probe passed in **1.34 s**. In an experiment-owned temporary copy,
it added `started + completed → succeeded` as **one branch in one model file**, with **zero
adapter source edits and zero safety-clause edits**. The unchanged adapter correspondence was
rechecked; both consumers admitted the new transition, replay accepted it, and the existing safety
condition detected it. The original model and its rejection regression remain intact.

This edit demonstrates no adapter change was needed for this added transition, not complete pipeline migration:
the evidence projector intentionally still rejects unsolicited completion from `started`. Adopting
that behavior would also require deliberate evidence support and revising the baseline regression.
The probe does not measure scoped compiler changes, production maintenance savings, or symbolic
performance. It removes no production code.

### Decisions

“Adopt” here selects a design direction for subsequent implementation; it does not migrate the
active Umpire codebase or amend its normative specification.

| Opportunity | Decision | Evidence and remaining boundary |
| --- | --- | --- |
| 1. Input/output ownership | Adopt explicit ownership. | Confirmation advances the model; commands and observation retries do not. Keep the two terminal outputs model-owned. Production Target/Case vocabulary and capability changes remain separate work. |
| 2. Scenario constraints/algebra | Adopt typed existing constraints; defer general algebra. | Ordered/exact/union examples preserve selected trace meaning in the prototype. General repetition, canonicalization, source mapping, and semantic fingerprints are not implemented. |
| 3. Typed modes/capabilities | Adopt the small typed frontend and bounded temporal notation. | Constructor/syntax equivalence is checked; modes keep evidence and effects out of the property fragment. This does not establish usability of a general grammar or authorize runtime effects. |
| 4. Scoped obligations | Adopt the explicit key, clock, bound, and closure interface. | Exhaustive small-word differential tests and the current-evaluator bridge agree. A production compiler correspondence proof and supported portable representation are still required. |
| 5. Query validity | Adopt separate satisfiability, exercise, answer, completeness, and endpoint fields. | Examples distinguish impossible/unexercised/unanswered results; semantic depth never substitutes for work completeness. Production query identity and receipts need deliberate migration. |
| 6. Shared passive monitor | Adopt the shared semantic module direction. | Both closure modes and chunked evaluation are checked; finite baseline comparisons reuse actual checked Property evaluation. General compiler correctness and Case lowering remain unproved/unimplemented. |
| 7. Evidence projection | Adopt explicit stutter/emit/reject with atomic admission and support. | Duplicate/noise/reordering corpus and focused causal/conflict cases pass. Immutable accepted evidence is an explicit premise; a real Temporal projection and versioned refinement link remain required. |

Keep the **current finite checker as default**. Retain the **optional Veil adapter as a feasible
representation seam**, and continue evaluating the **shared relational representation** only when
an actual checker can consume scenario/obligation product state. Defer **full Veil adoption and
Veil-centered authoring**: extraction from its generated declarations, compatible complete builds,
solver assurance, and portable identity were not demonstrated. The present relational view still
derives from finite enumeration; it does not prove symbolic-first admission or scalability.

Reject **a second handwritten behavioral model**, **wall-time/raw-event obligation clocks**,
**state equality as the test for stuttering**, and **green answers from empty or incomplete
searches**. Defer full LTL, unbounded liveness/fairness, and general scenario algebra until each has
an explicit semantics and supported assurance method. Existing bounded temporal logic should be
extended and reused, not described as absent or replaced merely for its notation.

The remaining discriminating work is a compatible full-checker run on the shared model plus
scenario/obligation state; a checked Property-to-monitor compiler and production evidence link;
and a migration design covering semantic fingerprints, portable Case admission, and runtime
capabilities. These experiments justify those focused steps. They do not yet justify wholesale
replacement or a claim of net production code savings.
