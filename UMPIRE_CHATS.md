# Umpire chat synthesis

Status: synthesis of two design conversations, not an approved implementation plan. The third URL
provided for this review duplicates the second. Recommendations from the chats are labeled as
proposals; repository decisions are taken from the current `UMPIRE_LEAN.md` and `UMPIRE.md`.

## Executive summary

The conversations converge on Umpire as more than a model-based test DSL. The stronger concept is a
semantic authority that connects API contracts, formal properties, executable scenarios, runtime
evidence, diagnostics, and human-facing explanations without creating parallel specifications.

The recurring architectural direction is:

```text
Protobuf representation + boundary validation
                       |
                       v
              Lean semantics and proofs
                       |
             proved/generated checker views
                       |
                       v
     runtime evidence DAG <-> execution controls
                       |
                       v
       qualified behavior and diagnostic claims
```

For the current repository, the most important reconciliation is that a human-readable rule language
must not become the independent “small typed semantic IR” proposed in the first chat. The current
roadmap makes Lean the sole semantic authority and treats scenario authoring as a separate intent
language. Human, Gherkin, support, and test views should therefore be renderings of Lean-owned
declarations or of an explicitly proved Lean view, not a new source of behavioral truth.
([Lean versus gRPC contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1);
[current decisions](UMPIRE_LEAN.md#2-non-negotiable-decisions))

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

## Recommended follow-ups

1. Record the contract-layer decision explicitly: Protobuf for representation, boundary validators
   for stateless validity, Lean for behavioral semantics, and Umpire for qualified conformance.
2. Pilot one Nexus dispatch-failure slice end to end. Define occurrence kinds and a total diagnostic
   policy, capture raw structured logs, add per-occurrence causal matching, and require mutations of
   every required attribute and causal link to fail.
3. Define a versioned runtime-event envelope and closure protocol using the existing source identity,
   source position, entity, lineage, causal-reference, and omission concepts. Make semantic traces a
   Lean-defined projection rather than the canonical raw record.
4. Extend one internal RPC seam with capture plus hold/release/drop and explicit realization receipts;
   then add logical timer ready/delivery gating.
5. Prototype ephemeral namespaces behind a local-test-only interface. Measure create/kill cost,
   detect surviving work, and compare contamination and throughput with normal namespaces and a pool.
6. Add stable rule IDs and generate one product statement, one support diagnostic, and several
   explorer-derived examples from a Lean-owned rule to test the human-facing workflow.
7. Defer deterministic Go scheduling until traces demonstrate races that RPC, timer, fault, and
   process controls cannot reproduce.

## Sources and limitations

- [Lean Versus gRPC Contracts](https://chatgpt.com/share/6a8b71d4-7c84-83e8-a3cb-1253c7eceff1)
- [Model Nexus Log Conformance](https://chatgpt.com/share/6a8b71e0-f334-83e8-9654-4ccf66d41bba)

Both public share pages were accessible in full. The duplicate third URL added no material. The
second conversation contains an early answer written before branch access and later, more specific
answers based on a branch inspection; this synthesis gives precedence to those later answers. The
chats are design discussions and include proposals and illustrative API shapes, not accepted
architecture or verified implementation status. Current repository documents were consulted to
separate their established constraints from the chats' proposals, but this was not a source-code
implementation audit and the chats' external citations were not independently revalidated.
