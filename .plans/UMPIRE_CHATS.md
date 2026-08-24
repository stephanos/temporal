# Umpire chat synthesis

Status: synthesis of three design conversations, not an approved implementation plan.
Recommendations from the chats are labeled as proposals; repository decisions are taken from the
current `UMPIRE_LEAN.md` and `UMPIRE.md`.

## Executive summary

The conversations converge on Umpire as more than a model-based test DSL. The stronger concept is a
semantic authority that connects API and configuration contracts, formal properties, regression and
exploration spaces, executable experiments, runtime evidence, diagnostics, and human-facing
explanations without creating parallel specifications.

The recurring architectural direction is:

```text
Protobuf descriptors + validation + dynamic-config catalog
                              |
                              v
             Lean semantics, properties, and scenario space
                              |
              checked/versioned artifact projections
                              |
                 +------------+------------+
                 |                         |
                 v                         v
       model exploration/checking   execution controls
                                           |
                                           v
                              runtime evidence DAG
                                           |
                                           v
                         result, replay, and qualification
```

For the current repository, the most important reconciliation is that a human-readable rule language
must not become the independent “small typed semantic IR” proposed in the first chat. The current
roadmap makes Lean the sole semantic authority and treats scenario authoring as a separate intent
language. Human, Gherkin, support, and test views should therefore be renderings of Lean-owned
declarations or of an explicitly proved Lean view, not a new source of behavioral truth.
([Lean versus gRPC contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1);
[current decisions](UMPIRE_LEAN.md#2-non-negotiable-decisions))

The third conversation turns that constraint into a product and delivery strategy. It proposes an
approachable Lean-embedded language for Umpire behavioral tests, thin generated Go wrappers for
normal `go test` integration, versioned artifacts between independently useful tools, and a narrow
Nexus proof-of-value experiment before further platform expansion. Those are proposals rather than
current repository commitments.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## Ideas consistent with current repository decisions

These ideas from the chats already match explicit Umpire3 direction:

- Lean owns behavioral meaning, properties, observation interpretation, refinement, and checker
  views; Go transports raw evidence and controls bounded execution.
- Selected Protobuf structure is imported mechanically from descriptors, while product meaning is
  handwritten and proved in Lean.
- Runtime adapters emit raw facts rather than declaring whether a property passed.
- Missing or ambiguous evidence fails closed as `unknown` or `conflict`; absence is meaningful only
  after an authoritative evidence window closes.
- Cross-source ordering uses source-local order and causal references, not wall-clock comparison.
- Feature semantics and concrete system mechanisms remain independently defined and connected by
  explicit refinement, including stuttering implementation steps.

These are repository decisions rather than new decisions made by the chats. The chats add concrete
applications and proposed next increments around validation, diagnostics, tracing, control, and
test isolation. ([Model Nexus log conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba);
[Umpire3 roadmap](UMPIRE_LEAN.md))

## 1. Divide API-contract responsibilities by layer

The first chat argues for three distinct contracts and a fourth conformance layer:

| Layer | Authority | Responsibility |
| --- | --- | --- |
| Representation | `.proto` / gRPC | RPC shape, field numbers and types, presence, `oneof`, wire and language interoperability |
| Stateless well-formedness | Protovalidate/CEL | Per-message and cross-field boundary constraints, with normal runtime errors and polyglot tooling |
| Behavioral semantics | Lean | State-dependent validity, multi-RPC behavior, retries, failures, temporal properties, and invariants |
| Implementation conformance | Umpire | Whether real Temporal evidence refines a permitted Lean execution under a declared evidence profile |

Lean should not replace Protobuf or Protovalidate for ordinary validation. Doing so would rebuild
mature wire, compatibility, reflection, runtime-error, and cross-language machinery while enlarging
the trusted computing base. Protobuf details such as field presence, unknown fields, duplicate-field
behavior, maps, `oneof`, JSON mapping, and schema evolution are subtle enough that generating `.proto`
from Lean would make the Lean-to-Proto generator part of the proof boundary.

The proposed direction is instead descriptor-to-Lean projection, followed by explicit theorems that
connect boundary validation to semantic assumptions, for example:

```text
Protovalidate accepts request
              -> model preconditions hold
              -> modeled transition preserves invariants
```

The valuable theorem is not “Lean can restate `timeout > 0`”; it is “the API boundary discharges every
assumption on which this behavioral transition and its proofs depend.” Removing or weakening a
validation rule should then break that proof or a shared conformance fixture.
([Lean versus gRPC contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1))

This is substantially aligned with the current descriptor-import boundary. What remains open is how
much of Protovalidate and CEL can be imported soundly and how equivalence is demonstrated across the
generated validator, the descriptor projection, and Lean.

## 2. Produce human views from one semantic rule

The first chat proposes stable semantic rules that can be rendered for different audiences:

- a concise product statement;
- a controlled acceptance/Gherkin view;
- an engineering precondition/action/postcondition view;
- a Lean theorem or property;
- executable Umpire scenarios and model-derived examples; and
- support material describing expected API results, state changes, logs, metrics, symptoms, and
  relevant components.

The human-facing language should be more precise than free-form Gherkin. Terms such as “exists” must
resolve to explicit semantic primitives—for example, API-addressable, visible, non-terminal, or
backed by an outstanding task. English should be a rendering template, not a parser whose ambiguity
quietly becomes formal behavior.

Rules and examples should remain distinct. A rule states the universal property; exploration finds
representative witnesses such as retries after frontend failure, lost responses, task dispatch, or
worker failure. Those witnesses can become reviewed acceptance scenarios without hand-authoring a
second set of expectations.

Stable rule identifiers such as `NX-START-004` would provide traceability across Lean declarations,
scenarios, implementation sites, API documentation, support cases, coverage, observed traces, and
CI results. This is a proposal, not an existing repository convention.
([Lean versus gRPC contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1))

The unresolved design constraint is authority: the chat suggests a central typed rule IR, whereas
the current roadmap forbids an independently authored semantic IR beside Lean. A compatible design
would make the rule declaration Lean-native, or prove an explicit authoring view equivalent before
using it to generate human and executable projections.

## 3. Treat diagnostics as semantic obligations, not log transitions

The second chat's central insight is that the Nexus state machine should not contain literal logging
actions. A semantic failure occurrence should instead induce a diagnostic obligation, and logs,
metrics, or spans should be raw evidence that may satisfy it:

```text
failure occurrence -> DiagnosticRequirement

raw log / metric / span facts
              -> causal matching and policy interpretation
              -> true | false | unknown | conflict
```

This keeps prose, logger APIs, and attribute encoding out of the feature state machine. A requirement
should describe what an operator must be able to learn: what failed, which subject was affected, the
cause, correlation identity, retryability or recovery state, and an appropriate severity. A stable
machine-readable event kind is preferable to matching the human log message.

The proposed Lean design is a separate `NexusDiagnostics` model family:

- `Feature` defines “every diagnostically relevant failure has sufficient causally linked evidence.”
- `System` represents concrete failure occurrences, raw structured records, identities, lineage, and
  causal references without deciding their adequacy.
- `Refinement` relates the concrete evidence to the feature contract.

A total policy function should map every failure kind to either `required(requirement)` or
`intentionallySilent(reason)`. An optional return would conflate an intentional no-log policy with a
forgotten case. Lean exhaustiveness then establishes policy coverage for every *modeled* failure
kind.

Each runtime failure must be an occurrence with its own identity. Three failed attempts are three
obligations. Matching merely by time, entity type, or “some warning exists” could let one unrelated
record satisfy multiple failures. Umpire therefore needs a per-trigger relational observation such
as “for every selected failure, there exists evidence causally referring to it.” The existing entity,
lineage, source-order, and causal-reference envelope is the right substrate.

Raw log attributes should preserve duplicate keys rather than normalizing immediately into a map;
conflicting duplicates should remain observable as `conflict`. Requirements can check exact model
values, required presence, allowed categories, minimum severity, and forbidden sensitive attributes.
They should distinguish attempt failures from terminal failures so successful retries do not emit or
satisfy terminal-failure diagnostics.

The abstraction should remain `DiagnosticRequirement`, even if logs are the first carrier, so the
same policy can later require a failure metric or failed span without redesigning Nexus semantics.
([Model Nexus log conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba))

### Three different completeness claims

The chats identify three boundaries that must not be collapsed:

1. **Policy completeness:** every modeled failure kind has an explicit diagnostic disposition.
2. **Execution conformance:** every observed failure occurrence in a closed trace has evidence that
   satisfies its policy.
3. **Implementation/model coverage:** every relevant runtime failure site emits a typed occurrence
   recognized by Lean.

Lean exhaustiveness proves only the first. It cannot discover a new Go error branch that emits no
fact. The implementation boundary must therefore fail closed on unknown occurrence kinds, and
important failure sites may need stable identities or instrumentation in the abstraction that owns
their semantic classification. Mutation tests should delete or corrupt logs, attributes, causal
references, levels, correlation identities, and retryability and require the conformance suite to
reject every nearby mutation.

## 4. Make the evidence graph the canonical execution trace

The branch inspection recorded in the second chat characterized the then-current trace as primarily
semantic actions/outcomes plus sampled receipts and history, with a richer causal evidence graph
alongside it. The recommended destination is a continuous typed runtime-event DAG from which Lean
projects the semantic trace:

```text
typed RuntimeTrace DAG
  source identity + source sequence + entity + lineage + causal references
  RPC + HSM + task + timer + persistence + fault + log/span events
                              |
                    Lean interpretation/refinement
                              v
                system trace -> feature trace
```

The implementation trace and the execution control plane should be separate first-class inputs.
Observation records what happened; control realizes a requested schedule or fault and produces its
own receipts. This prevents a requested fault from being mistaken for a realized fault.

Important trace properties are:

- direct, typed, unsampled in-process hooks for local correctness claims;
- OTel as a useful telemetry, remote, and canary channel, but not the primary local correctness
  channel because exporters may sample, batch, or drop data;
- monotonic source-local sequence numbers and an explicit source-closed position;
- gap detection, with missing events yielding evidence failure rather than successful absence;
- partial-order checking over the causal DAG rather than equality with a collector-imposed total
  order; and
- explicit projection/refinement rules, including stuttering for implementation events such as a
  lock or read that do not correspond to a feature transition.

The useful definition of high-quality conformance is scoped rather than absolute: for the declared
abstraction and evidence profile, every relevant implementation event is accounted for by a valid
Lean execution and no evidence gaps are hidden.
([Model Nexus log conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba))

## 5. Add control incrementally

The chats separate three levels of runtime control:

1. **Trace recognition:** run the nondeterministic server, capture the execution, and ask whether some
   Lean execution permits it. Runtime determinism is unnecessary.
2. **Controlled replay:** control external actions, RPC delivery, faults, timers, randomness, and
   process lifecycle to reproduce a semantic race.
3. **Schedule exploration:** control Go scheduling and synchronization for bugs that remain below the
   semantic control points.

The recommended order is levels 1 and 2 before deterministic Go scheduling. Most distributed races
should become reproducible by controlling messages, timers, faults, and process lifecycle; modeling
every mutex in feature semantics would couple proofs to implementation structure.

Two proposed controls are especially important:

- A fault control plane should connect the existing fault vocabulary to real RPC, timer,
  persistence, process, and partition interceptors. Every requested fault needs activation and
  realization evidence tied to the concrete intercepted occurrence.
- A timer gate should separate “became ready” from “was delivered.” It should operate at the logical
  timer/task eligibility seam, not only on literal Go timers, because Temporal implements delayed
  work through higher-level queues and backoff mechanisms.

Scheduler events may still be retained in the deepest deterministic mode, but normally projected
away below system refinement.
([Model Nexus log conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba))

## 6. Make per-trace isolation cheap without weakening identity

The isolation discussion evolved. It first proposed pooled namespaces with a finer test-domain ID,
unique task queues, explicit worker shutdown, and optional Matching unload. After the user asked to
make namespace isolation itself cheaper, the final recommendation preferred **ephemeral namespaces
per trace for local/in-process Umpire**.

The proposal preserves a fresh `NamespaceID` for every trace but bypasses production registration,
replication, propagation, and asynchronous deletion. An in-memory registry overlay creates a usable
test namespace; namespace IDs and task-queue names are never reused. Teardown becomes a logical
lifecycle:

```text
ACTIVE -> QUIESCING -> DEAD
```

It stops external producers and workers, disables and releases controls, drains or invalidates work
for the namespace generation, closes trace sources, unloads Matching state, removes the registry
entry, and leaves physical persistence cleanup for batching or cluster shutdown. A generation check
could make old RPCs, timers, callbacks, and tasks non-runnable immediately.

The proposed deployment split is:

| Environment | Isolation proposal |
| --- | --- |
| Remote/real deployment | Normal namespaces |
| Shared CI server | Normal namespace or pool |
| Local in-process Umpire | Ephemeral namespace per trace |
| Deterministic Gomad | Ephemeral namespace generation per explored execution |

This proposal is attractive because it keeps namespace-strength identity while removing production
administrative lifecycle from the hot path. Its trust boundary must be explicit: normal behavior is
tested only between creation and quiescence; namespace registration, deletion, failover, and cleanup
semantics still require normal namespaces. Generation-kill must not silently hide leaked work when a
test claims to verify cleanup behavior.
([Model Nexus log conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba))

## 7. Prove value with one narrow, highly reused slice

The third chat recommends treating Umpire as an instrument for a bounded experiment rather than
arguing for model-based testing in the abstract or continuing to expand the platform. The proposed
claim is deliberately limited: for an important class of stateful Temporal behavior, Umpire should
catch more meaningful defects earlier, exercise interactions that normal tests omit, and reduce the
marginal cost of each additional behavior.

The suggested first slice is Nexus cancellation and retry combined with caller closure, callback or
result arrival, one SDK participant, and a small set of realized failures. It is narrow in modeled
domain but broad in reuse: the same semantics should support regressions, bounded exploration,
failure injection, cross-feature behavior, SDK/server execution, validation, design questions, and
formal checks. Remote qualification, production canaries, deterministic scheduling, and additional
formal backends remain on the destination map but do not block the first demonstration.

The proposed scorecard is evidence-oriented:

| Question | Measurement |
| --- | --- |
| Does it detect defects? | Historical regressions reproduced, realistic semantic mutations killed, and any new defects found |
| Does it find them earlier? | Local and CI time-to-failure compared with integration or cloud pipelines |
| Does it broaden coverage? | Semantically distinct interactions absent from hand-authored suites |
| Does reuse compound? | Person-hours for the second, third, and fourth behaviors after the model exists |
| Is it maintainable? | Cost of updating the model and generated tests after a semantic or implementation change |
| Is it trustworthy? | Negative controls, qualified evidence, minimization, and deterministic replay |
| Can another engineer use it? | Onboarding time, questions, and time to add a regression |
| Is it cheaper to execute? | CI duration, cluster minutes, and remote-environment cost |

An unknown defect is the strongest possible result but should not be the minimum success condition;
it is too dependent on luck. A fixed corpus of historical bugs, injected mutations, and known edge
cases first establishes that the detector fails for the intended semantic reason. A later,
time-bounded exploration campaign can search for unknown defects. Success and stop/go thresholds
should be written before running the experiment so that the result is not selected after the fact.

The chat also recommends testing usability directly: give a regression task to an engineer who did
not build Umpire, and measure whether the semantic declaration, generated test, and failure report
are understandable without specialist Lean knowledge. Lean and AI-assisted implementation are
enabling technologies, not the adoption or correctness pitch; the credibility chain is reviewed
semantics, deterministic generation, negative controls, qualified evidence, and checked claims.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## 8. Make regression and exploration one Lean-owned semantic space

The authoring discussion evolves through three positions. It first proposes a generated Go DSL over
Lean semantics, then requires that any Go frontend emit a pure `ScenarioSpec` which Lean validates,
and finally recommends testing whether ordinary Umpire behavioral tests can be authored directly in
an approachable Lean-embedded language. The later recommendation takes precedence in this
synthesis, while retaining a generated Go facade as a possible adoption layer.

The central abstraction is one semantic scenario space:

```text
Lean feature and composed-system models
                   |
                   v
        Lean-owned ScenarioSpace
          actions + variation axes
          partial orders + properties
                   |
          +--------+---------+
          |                  |
          v                  v
 pinned regressions    exploration policy
 known coordinates     exhaustive/pairwise/t-wise/
                       coverage-guided/budgeted
          |                  |
          +--------+---------+
                   v
        checked ExperimentSpecs
```

A regression selects and names a stable point or constrained family in the space. Exploration
selects many points under a strategy and budget. A minimized discovery can therefore be promoted by
recording its semantic coordinates rather than copying a procedural RPC sequence. Pinned
regressions always run; an exploration budget adds candidates rather than replacing known cases.

The proposed language needs only a small set of serializable combinators: setup and expected
outcomes, choice and variation, optional/repeated actions, partial-order constraints, interleaving,
fault scopes, coverage goals, and selection budgets. Arbitrary Go callbacks or `Build` functions
should be escape hatches, not the normal API, because opaque code prevents Lean validation,
minimization, explanation, replay, generation, and cross-language reuse.

Cross-feature composition belongs in Lean before test authoring. A developer should select a checked
target such as `WorkflowNexus` or `WorkflowNexusActivity`, not invent composition with a runtime
`Combine` call. Before execution, Lean can check that a scenario is well-typed, reachable or
satisfiable, respects action preconditions and identities, belongs to its declared space, targets the
stated property, and compiles to an experiment that refines the semantic declaration. Execution then
answers the separate question of whether Temporal conforms to it.

Going Lean-first is explicitly limited to Umpire-owned behavioral regressions, exploration,
cross-feature properties, and formal checks. Unit tests, race tests, persistence tests, benchmarks,
and exact handler tests remain in Go. The proposed adoption test is to build three representative
Lean-authored cases—a simple regression, a combinatorial exploration, and a cross-feature output
property—and ask ordinary Go engineers to modify them after a short orientation. If that fails,
a generated Go frontend can remain, but it must construct the same Lean-checked semantic object and
must not become a second authority.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## 9. Model API shape, configuration, and observable effects explicitly

The third chat proposes extending the descriptor boundary from the selected structures currently
used by Umpire to complete mechanical schema knowledge in Lean: services, RPCs, messages, fields,
enums, presence, `oneof`s, and annotations. This does **not** mean assigning behavioral meaning to
every field. The proposed layers remain distinct:

```text
complete generated wire schema
              |
              v
explicit field/message disposition
semantic | opaque | sensitive | ignored
              |
              v
handwritten behavioral interpretation and proofs
```

Validation annotations could provide typed predicates, boundary values, valid generators, and
invalid-neighbor generators when their semantics are supported. Complex CEL or validator behavior
still requires a deliberately supported translation subset or an opaque runtime boundary, as
described in section 1.

Dynamic configuration receives a parallel treatment. A mechanical catalog records key, type,
default, scope/precedence, and description. A separate semantic classification records its impact
(feature, validation, externally visible semantics, timing, topology, performance, or observability)
and when it is sampled (live, at creation, per request, per task, or after restart). Models declare
only the config dependencies relevant to their target.

The model consumes a resolved `ConfigView`, not an unprocessed override map. Resolution semantics
must agree with Temporal's namespace, task-queue, shard, task-type, destination, and other precedence
rules. Most experiments pin a snapshot. A scenario uses an explicit config-change action only when
behavior across the change is itself under test. If a setting is sampled at entity creation, the
model records its semantic consequence on that entity rather than retroactively applying the latest
value.

Product-visible feature gates, validation limits, and contractual policies belong in the product
model. Partition counts, cache sizes, QPS, autoscaling, and similar operational knobs normally belong
in an execution profile or lower system model. This separation prevents every behavioral scenario
from multiplying across hundreds of irrelevant settings while still allowing dedicated routing,
fairness, capacity, and scaling properties.

The conversation also broadens the semantic model from a pure state transition system to an
I/O-labelled transition system:

```text
State + ConfigView + Input + Action
                  |
                  v
      next State + Response + observable effects
```

Observable effects include history events, links, callbacks, tasks, logs, spans, and metrics. The
property vocabulary should therefore distinguish state invariants, transition contracts, input and
output contracts, observation requirements, entity relations, trace ordering or liveness, and
refinement. For example, Lean can require a Nexus-origin link under a modeled condition, while live
conformance separately establishes whether the server emitted raw Protobuf evidence that interprets
as that semantic link.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## 10. Generate familiar Go tests without moving authority into Go

The adoption proposal is to export checked Lean regressions through a canonical manifest and use a
small Go generator to create ordinary `_test.go` wrappers:

```text
Lean regression and checks
            |
            v
versioned regression manifest
 id + source + target + property + semantic hash
            |
            v
thin generated Go TestX wrapper
            |
            v
          go test
```

The wrapper should only select a checked regression or family by stable identity. It may contain a
human-readable Given/When/Then summary and a link to its Lean source, but it should not expand into a
procedural sequence that appears to be the specification. One Go test may execute several valid
experiments when the regression is intentionally partially quantified.

Stable regressions should be generated and checked in so developers retain normal test discovery,
IDE actions, `go test -run`, CI/JUnit names, ownership, and code search. CI regenerates them and
requires an empty diff; source digests and semantic hashes expose drift. Individual exploration
candidates should not be checked in because they are policy- and budget-dependent. A discovery gets
a wrapper only after minimization and promotion to a named Lean regression.

Lean can emit Go directly, but the proposed trust boundary is simpler: Lean exports checked semantic
JSON, and a mundane Go generator handles package names, imports, formatting, and build tags. The same
manifest can also generate test catalogs, documentation, validation cases, coverage maps, canary
plans, or SDK participant programs without granting those projections semantic authority.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## 11. Productize Umpire as tools connected by versioned artifacts

The final architectural proposal is a single `umpire` command backed by independently useful
subsystems. Each subsystem owns one narrow transformation and communicates through a versioned
artifact rather than importing another subsystem's internal representation.

| Subsystem | Boundary | Example CLI |
| --- | --- | --- |
| API and config import | descriptors or Go config declarations -> generated catalogs and drift reports | `umpire api/config sync|check|explain` |
| Authoring and model compilation | Lean declarations -> checked specification and ExperimentSpecs | `umpire spec check`, `umpire compile`, `umpire explain` |
| Go and documentation generation | regression catalog -> familiar projections | `umpire gen go-tests|docs` |
| Execution and SDK participants | ExperimentSpec + environment -> ExperimentRun and raw evidence | `umpire run`, `umpire participant` |
| Evidence and conformance | raw evidence + ExperimentSpec/ExperimentRun -> semantic evidence and Result | `umpire result check|explain` |
| Exploration | semantic space + strategy + budget -> selected ExperimentSpecs and coverage | `umpire explore` |
| Replay, minimization, promotion | failing bundle -> smaller replay or Lean regression | `umpire replay|minimize|promote` |
| Formal backends | model target + bounds -> receipt or counterexample | `umpire verify` |
| Deployment qualification | ExperimentSpec + authorized remote profile -> ExperimentRun and qualified Result | `umpire qualify` |

The proposed core artifact families are API, config, and semantic catalogs; regression and
exploration declarations; ExperimentSpecs; ExperimentRuns; raw and semantic evidence; Results;
replay bundles; and
verification receipts. Each should carry a format version and, where relevant, semantic and source
digests. The execution runtime need not know Lean; it consumes an `ExperimentSpec` and records an
`ExperimentRun`. The checker consumes that run plus raw evidence rather than inferring how execution
happened. Exploration can initially run entirely over the model, then compose with execution later.

The delivery sequence is similarly incremental:

1. Import schema/config, provide the Lean authoring surface and experiment compiler, and generate
   familiar Go test wrappers.
2. Execute one compiled experiment locally against Temporal and check state plus observable output
   evidence, including one SDK participant.
3. Add exploration, minimization, replay, and promotion so a previously unwritten interaction can
   become a stable regression.
4. Only after those stages demonstrate value, expand formal backends, remote qualification, canary
   execution, or deterministic scheduling.

This decomposition is a proposal, not a reason to replace already working repository seams. Its
test is whether each command remains independently valuable and whether the artifact boundaries
reduce coupling without introducing another semantic representation that can drift from Lean.
([Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc))

## Risks and tradeoffs

- **A second semantic authority:** a standalone rule IR or hand-maintained Gherkin would drift from
  Lean. Generated views need a proved binding to Lean-owned semantics.
- **Trusted-code growth:** Lean-to-Proto generation would require faithfully reproducing Protobuf
  wire and evolution semantics. Descriptor import keeps the mature representation contract primary.
- **Unmodeled implementation failures:** total matching over a Lean failure taxonomy says nothing
  about an uninstrumented Go branch. Unknown failure facts and trace gaps must fail closed.
- **False correlation:** timestamps or broad entity matching can associate unrelated evidence with a
  failure. Per-occurrence causal identity is required under concurrency.
- **Evidence perturbation and loss:** correctness instrumentation must be bounded, unsampled, and
  able to prove source closure; otherwise dropped events can manufacture conformance.
- **Noise and privacy:** “log every failure at error” creates spam. Failure class should determine
  policy, and forbidden payload, authorization, or callback-token fields should be checked as well as
  required fields.
- **Model/implementation coupling:** low-level scheduler and lock events are useful for deep replay,
  but should normally stutter below the system model.
- **Ephemeral teardown semantics:** dropping all dead-generation work is fast but test-specific. It
  must be isolated to local/deterministic profiles and cannot support claims about production
  namespace deletion or cleanup.
- **Scope and cost:** full Go scheduler determinism and exhaustive internal tracepoints are invasive.
  Continuous evidence, gap detection, RPC control, timer gating, and semantic tracepoints offer a
  higher-value intermediate layer.
- **Authoring adoption:** a Lean-embedded DSL removes semantic duplication but may still be too
  unfamiliar for ordinary contributors. The three-test usability trial should decide this before a
  large rewrite; a generated Go frontend remains a fallback.
- **Generated-view confusion:** checked-in Go wrappers improve discovery but can be mistaken for the
  specification. They must remain thin, point to the Lean source, and expose semantic hashes.
- **Configuration explosion and drift:** indiscriminately varying dynamic configs makes exploration
  intractable, while a separate resolver can disagree with Temporal. Models should declare relevant
  dependencies, and resolution needs cross-language conformance fixtures.
- **Artifact fragmentation:** many independently versioned artifacts improve subsystem isolation but
  introduce compatibility, migration, provenance, and garbage-collection work. A format version and
  digest do not by themselves prove semantic compatibility.
- **Proof-of-value bias:** case counts, LOC, or a lucky new defect can overstate value. Predeclared
  metrics, negative controls, historical bugs, realistic mutations, and explicit stop/go criteria
  are needed for a credible experiment.
- **Continued overbuilding:** schema importers, DSLs, generators, remote execution, canaries, and
  deterministic scheduling can consume the project before the core economic hypothesis is tested.
  Each new subsystem should be justified by the bounded Nexus experiment or deferred.

## Open questions

1. What Lean-native rule declaration or explicitly proved view can generate product, Gherkin,
   support, and executable outputs without violating the sole-authority decision?
2. Which subset of Protovalidate standard rules and CEL can be imported, and what cross-language
   fixtures or proofs establish equivalence to the runtime validator?
3. What is the first closed Nexus failure taxonomy, and which cases are intentionally silent?
4. What relational observation primitive gives per-failure cardinality and causal matching without
   allowing one record to discharge unrelated obligations?
5. Which component owns stable failure-site and runtime-event identities, and how are they propagated
   through internal RPCs, task queues, timers, and callbacks without changing production APIs?
6. What is the minimum event inventory and source-closure protocol needed before a live trace can be
   called complete for a particular property?
7. Which fault and timer seams provide semantic control without depending on unstable implementation
   details?
8. Can an ephemeral registry overlay and namespace-generation cancellation be implemented without
   changing active-namespace behavior or masking resource leaks?
9. Which evidence carriers are required by profile: direct hooks locally, telemetry remotely, public
   history in black-box mode, or an explicit unsupported result?
10. Can ordinary contributors author and modify the proposed Lean DSL comfortably, or is a generated
    Go facade necessary? What usability result decides between them?
11. What exactly does Lean certify about a regression or exploration space before execution, and how
    is that certificate bound to the compiled ExperimentSpec?
12. Should Lean import the complete Protobuf schema or only selected structures, and what build-time,
    review, compatibility, and trusted-code costs follow from complete import?
13. How is the dynamic-config catalog extracted, classified, and resolved, and which fixtures prove
    agreement with Temporal's Go precedence and sampling behavior?
14. What is the first output or observation property—such as a Nexus-origin link—that demonstrates
    the I/O-labelled model end to end?
15. Which artifact schemas must be stabilized first, and what compatibility and migration policy
    prevents separately shipped tools from accepting semantically incompatible inputs?
16. What predeclared scorecard and stop/go thresholds make the first Nexus proof-of-value experiment
    credible to engineers and management?

## Recommended follow-ups

1. Freeze broad capability expansion long enough to define the Nexus proof-of-value scorecard, bug
   and mutation corpus, time budget, comparison baseline, and stop/go thresholds.
2. Prototype three Lean-authored behavioral specifications: a simple regression, a combinatorial
   exploration, and a Workflow/Nexus output property. Run the proposed usability test with engineers
   who did not build Umpire before choosing Lean-only authoring or a generated Go facade.
3. Define the first artifact schemas—semantic catalog, regression/space, ExperimentSpec,
   ExperimentRun, evidence, and Result—with version, provenance, and semantic-digest rules. Use them
   to isolate the first CLI transformations rather than redesigning all packages at once.
4. Record the contract-layer decision explicitly: Protobuf for representation, boundary validators
   for stateless validity, Lean for behavioral semantics, and Umpire for qualified conformance.
5. Pilot one Nexus cancellation/dispatch-failure slice end to end. Define occurrence kinds and a
   total diagnostic policy, capture raw structured logs plus the chosen API/history output, add
   per-occurrence causal matching, and require mutations of every required value and link to fail.
6. Generate thin, checked-in Go wrappers only for stable Lean regressions. Include source identity,
   a readable contract summary, and a semantic hash; require deterministic regeneration in CI.
7. Define a versioned runtime-event envelope and closure protocol using the existing source identity,
   source position, entity, lineage, causal-reference, and omission concepts. Make semantic traces a
   Lean-defined projection rather than the canonical raw record.
8. Import and classify only the API/config material required by the pilot first. Add cross-language
   fixtures for validation, config precedence, and sampling behavior before expanding catalog scope.
9. Extend one internal RPC seam with capture plus hold/release/drop and explicit realization receipts;
   then add logical timer ready/delivery gating.
10. Prototype ephemeral namespaces behind a local-test-only interface. Measure create/kill cost,
    detect surviving work, and compare contamination and throughput with normal namespaces and a pool.
11. Add stable rule IDs and generate one product statement, one support diagnostic, and several
    explorer-derived examples from a Lean-owned rule to test the human-facing workflow.
12. Defer remote qualification, canaries, additional formal backends, and deterministic Go scheduling
    until the local slice demonstrates value or a concrete failure requires the deeper capability.

## Sources and limitations

- [Lean Versus gRPC Contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1)
- [Model Nexus Log Conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba)
- [Inspect Umpire Branch](https://chatgpt.com/share/6a8b71cb-74e4-83e8-947a-c2f6d595fefc)

All three public share pages were accessible in full. `Model Nexus Log Conformance` contains an early
answer written before branch access and later, more specific answers based on branch inspection;
this synthesis gives precedence to those later answers. `Inspect Umpire Branch` also evolves from a
Go-first authoring facade toward a Lean-first authoring experiment; its later recommendation is
represented here while the Go facade remains a fallback proposal.

The chats are design discussions and include repository snapshots, proposals, illustrative API
shapes, estimates, and external claims, not accepted architecture or verified implementation status.
Current repository documents were consulted to separate their established constraints from the
chats' proposals, but this was not a new source-code implementation audit and the chats' external
citations were not independently revalidated.
