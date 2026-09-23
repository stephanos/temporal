# Temporal Lean model architecture

This directory contains neutral formal primitives, the reusable Umpire library, generated Temporal
structure, handwritten Temporal behavior, and the first Case Producer. The normative rules live in
[the Umpire 4 specification](../.plans/UMPIRE4_SPEC.md); the reusable API is described in
[Umpire/ARCHITECTURE.md](Umpire/ARCHITECTURE.md).

## Libraries and imports

| Import | Responsibility |
| --- | --- |
| `Shared` | Neutral transition and trace-replay primitives. |
| `Testpilot` | Generated Testpilot protocol, producer-neutral authoring, and ProtoJSON policy. |
| `Umpire` | Temporal-independent modeling, planning, promotion, and Testpilot provenance. |
| `Temporal` | Generated Temporal structure, handwritten semantics, and Temporal Case production. |

Most consumers start with `import Umpire` or `import Temporal`. Focused imports should follow the
owner boundaries below:

```text
Shared
  └── correlated projection and obligation, inert named values

Umpire.Core ──▶ Model.Types ──▶ Model.Canonical ──▶ Model.Check
                                                       ├──▶ Property / Scenario semantics
                                                       │       └──▶ Query semantics ──▶ Search ──▶ Search.Admission
                                                       ├──▶ Model.Elab ──▶ Model authoring facade
                                                       └──▶ Model.Table ──▶ Model authoring facade

Checked Models ──▶ Evidence / ImplementationLink
Admitted Queries ──▶ Variations / Exploration / Promotion

Testpilot.Protocol ──▶ Testpilot.Authoring ──▶ Temporal.Testpilot
         │                       ▲                       ▲
         └────▶ Testpilot.ProtoJSON       Umpire.Case ──▶ Umpire.Provenance

Umpire.Command ──▶ Temporal.Case ──▶ Temporal.Feature.* Model files

Temporal.API ───────────────────────────┐
Temporal.DynamicConfig ────────────────┤
Temporal.Feature ──────────────────────┼──▶ Temporal
Temporal.System ───────────────────────┤
Temporal.Testpilot ────────────────────┘
```

`Umpire.*` never imports `Temporal.*`. `Temporal.Feature.*` and `Temporal.System.*` remain separate
except for the exact checked Implementation Link leaf. `Temporal.Tool.*` owns developer commands and
is not imported by the production aggregate. A production module under `Temporal.Feature` or
`Umpire.Examples` reaches Umpire's authoring owners (`Umpire.Model`, `Umpire.Property`,
`Umpire.Scenario`, `Umpire.Query`, `Umpire.Operation`, `Umpire.Case`) only through
`Umpire.Command`; `Temporal.Case` and the Implementation Link are outside that rule. `make
lint-model` checks these edges against the full source inventory and compiled module metadata, the
authoring-path rule as a direct-import rule and the rest as reachability.

`make umpire-export-model-module-index` projects the same source inventory and compiled metadata
into a `temporal-model-module-index/v1` document (`ModelLint.ModuleIndex`, exported by
`temporal-model-module-index`): per module, its classification under the lint policy, its direct
and reverse first-party imports, and which of the reviewed public facades and focused test roots
reach it. The roots are explicit policy in `ModelLint.ModuleIndex.defaultIndexPolicy`, never file
names. The index is a navigation aid over this graph, produced on demand and never checked in; it
makes no semantic claim and is not an input of any check.

`Umpire.Model.Check` is the narrow checked-model import. It owns pure admission together with
private checked construction; `Model.Canonical` owns pure canonicalization, and `Model.Elab`
owns syntax capture and located elaboration. Property, Scenario, Query, and Search semantic
modules cannot transitively import the model elaborator or `Lean.Elab.Term`. The ordinary authoring
facades remain `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`, and `Umpire.Query`; importing
`Umpire.Search` alone does not provide their authoring conveniences. See the
[Model ownership table](Umpire/ARCHITECTURE.md#model-ownership-and-semantic-imports) for the checked
API and replacement-proof contracts.

## Generated structure

`Temporal.API` contains descriptor-derived message, enum, field, presence, map, oneof, recursion,
and unary-method facts. `Temporal.DynamicConfig` contains the complete generated registry snapshot.
Neither generated family defines behavior. Handwritten Feature and System modules interpret only
the facts they explicitly import.

The owner commands validate a complete candidate before replacing managed output:

```sh
make umpire-gen-lean-api
make umpire-gen-lean-dynamic-config-catalog
```

## Semantic model

The retained semantic APIs keep these responsibilities separate:

- The Model owns valid setup, state, Action, Model Outcome, Fact, Step, and Capability domains.
- Property states a claim over Traces.
- A Scenario constrains allowed Trace shape without choosing Model-owned outcomes.
- Query asks one bounded question, and Search answers it. `Search.admit` checks the Property,
  Scenario, Known Gaps, Query, and search view in one order and returns an `AdmittedQuery` or the
  first stage that rejected.
- Variations compile a finite checked universe, and Exploration plans one target Query per coverage
  target of an exploratory set, without performing runtime I/O.
- Evidence and Implementation Link retain the offline mapping path for model analysis.
- Promotion validates one exact scenario-neutral Plan source for human review.

Plans and Generated Views remain useful model outputs. They are not inputs to Testpilot and do not
establish that a runtime Action occurred.

A Model file is written in `Umpire.Command`'s commands -- entities, domains, actions, observations,
machines with a step function per action, properties as predicates, scenarios, limits, queries and
sets -- and a platform-owned `case … realizes <set>` block produces one Case per Query of a
functional set through a realization in `Temporal.Case`. [AUTHORING.md](AUTHORING.md) walks the
Nexus caller-side Model through them, quoting its marked regions under a drift test.

Operation-correlated bounded response authoring lowers through the existing Property checker.
`correlated_response%` and typed `PropertyCorrelatedClause` values share canonical meaning and fingerprints;
key, scope, bound, and ending remain explicit semantic choices.
Checked projection, source Property, and portable Contract are connected by `Umpire.Case.Correlated`
certificates. Shared table/projection/obligation modules contain no feature callback; generic
Testpilot interprets the admitted correlated capability and maintains fresh state for each Run.

The non-cancellation correlated corpus now includes RPC Programs that emit typed observations through
the public Prepare/Run path. It qualifies correlation, inclusive deadlines, preserved violation
proof, incomplete/lost execution, cleanup failure, and bounded tenfold loads. The existing Nexus
success Case remains the live Driver integration. Cancellation Models, evidence adapters, operation
capabilities, and authored Cases are deferred to fn-79, independently of Run-context cancellation
and bounded cleanup.

Typed operation authoring is written on the commands: an `action`'s `schema:` line binds it to the
generated RPC or SDK-command declaration, a `property`'s `relates:` line compares exact field
operands over the declared schemas, and an action's `input:` and `examples:` lines declare its
finite domains and abstraction claims. The generator and `Temporal.API` own structural declarations
and supported exact value representations; `Umpire` owns admission, canonical meaning and Property semantics;
`Temporal.Feature` owns the product requirements. `Umpire.Case.Coverage` owns the request direction
of the checked lowering, and `Umpire.Case.Projection.lower` derives the evidence direction, the
monitor rule itself, from the checked field Property. The
[authoring walkthrough](README.md#typed-operation-authoring) records the supported and unsupported
value forms, and the [Umpire architecture](Umpire/ARCHITECTURE.md#typed-field-lowering) records the
lowering and identity ownership.

## Testpilot protocol and Producers

The checked-in `.proto` closure rooted at
`proto/internal/temporal/server/api/testpilot/v1/case.proto` is the sole Case wire schema.
`Testpilot.Protocol` exposes the generated Lean declarations, and `Testpilot.Authoring` constructs
those generated values through context-safe helpers:

```text
Case
├── version, identity, typed producer provenance rows the runtime never reads
├── Program
│   ├── symbolic roles
│   ├── Case 1.0 symbolic environment declarations and direct resource references
│   ├── typed private Slots and declared Observations
│   ├── controller / workflow / activity / Nexus-handler DAGs
│   ├── cleanup graph
│   └── instruction timeouts and attempts
└── Contract
    ├── deterministic safety and bounded-liveness rules
    ├── bounded captures
    ├── expiry-before-transition deadlines
    └── correlated windows
```

A Case carries no resource ceilings. Structural, runtime, work and storage ceilings belong to the
Profile, and `testpilot.Prepare` checks the Case's behavior bounds and structure against them.

`Umpire.Case` retains only Umpire's producer-specific definitions, fingerprints, sources, and Known
Gaps and lowers them into the Case's typed provenance rows. It does not own a parallel Program, Contract,
Run, or field serializer. The Producer is `Umpire.Case.Producer`, reached through the platform's `case … realizes` block: it
assembles a Case's Program and Contract from a checked Query's witness and a realization through
`Testpilot.Authoring`, and uses `Umpire.Case.Compiler` for source-bound rule validation, Case-local
names and model value spellings, exact provenance rows, and final assembly from generated values. The
Contract's evidence is the witness's, plus the evidence of every other result of each witnessed row
(the alternatives, one kind each, projected to their rows under the witness's silent prefix), so a
Run that takes another result of an authorized row is read and judged against it rather than left
unread; a kind any other result or row already declares, or one the realization does not admit, is
a production error. `Testpilot.ProtoJSON` delegates canonical
encoding to `Protobuf.Json`.

`Umpire.Case.Producer` is the one Producer: every checked-in functional Case is produced from a
command Model through a realization (`Temporal.Case.Realization.{asyncNexus,workflowStart,
workflowOutage,unaryRpc}`). The system-info Model's Case proves that the IR is not tied to Nexus;
the caller Model's Cases use controller RPCs plus SDK workflow and Nexus-handler entrypoints without
adding a scenario opcode. `Temporal.Testpilot` supplies the six conformance Cases, which cover the
root Go facade's satisfied, violated, inconclusive, static-rejection, cleanup-failure, and cross-Run
classes.

`Temporal.Tool.Testpilot` is a build-time renderer only. Coordinates, credentials, clients,
workers, capabilities, and live IDs remain Driver inputs.

`Temporal.Tool.ExplorationBridge` (`umpire-explore`) is the run-time counterpart for an
exploratory set: it opens the set's `Umpire.Exploration` campaign and answers `initialize`,
`next`, `observe` and `finish` frames over stdin and stdout, producing each candidate's Case
through the same Producer and the realization the set's `case` block names, under the candidate's
own identity (`temporal.case.<set>.<digest>`), and crediting the campaign from the closed Run the
coordinator hands back. It registers nothing in the Temporal Case Registry and reads no Run
Event: credit is the planned witness path. The campaign retains each counterexample's candidate
and `Umpire.Exploration.Promotion` compiles it through `Umpire.Promotion` into a review-only
regression source under fresh names keyed by the candidate's digest; the `finished` frame carries
the source's digest, path and bytes (or why it did not compile), and the bridge writes no file.

Exact Case 1.0 is the only admitted format. A resource-bearing Program's roles and expressions reference
symbolic text IDs for namespaces, task queues, and named Nexus endpoints, and preparation derives the
closed set they form; a resource-free Program references none. These references
express resource relationships, not physical values; an endpoint role or binding ID is not a network
address. The immutable Profile supplies physical values, while transport targets, credentials,
callback authority, SDK clients, and lifecycle configuration remain environment-owned Driver inputs.
Changing a Profile binding therefore changes prepared/Driver binding identity, not the source Case,
its Contract, Behavior Fingerprints, or producer provenance rows.

## Go runtime boundary

The corresponding Go architecture is deliberately small:

```text
Case + immutable Profile
        │
        ▼
testpilot.Prepare ──▶ PreparedCase
                    │
                    ▼
             Run(ctx, Driver)
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
 internal execution       Contract verification
        │                       │
        └───────────┬───────────┘
                    ▼
             closed Run + Verdict
```

The Testpilot `.proto` files own the Case protocol. `common/testing/testpilot` owns the
Profile/Driver contract and the two Go calls.
Scheduling, recording, effect ownership, private Slot storage, and Monitor factories are internal.
Static preparation performs no Driver I/O; a Prepared Case snapshots admitted inputs and supports independent
sequential and concurrent Runs.

Temporal server and worker authority do not overlap. The server Driver supplies the authorized
descriptor catalog, transports prepared unary method/request pairs, and returns raw typed responses
and protocol status. Internal execution constructs requests and applies declared response
reads to private Slots and Run Observations. The worker Driver uses Temporal SDK APIs for
workflow, activity, and Nexus-handler execution, owns reservation delivery, and cancels at
activation scope. The composite Driver joins these capabilities without interpreting the Program or
Contract. The composite and its `server` and `worker` packages live under
`common/testing/testpilot/temporal`.

`Prepare` snapshots and resolves Case 1.0 bindings without target I/O. A Prepared Case retains the
unchanged symbolic source and private resolved instruction and role data, and its identity includes
the complete Profile binding fingerprint. `Run` compares that identity, invokes the Driver's no-I/O
`Validate` hook, creates the Contract Monitor, and then calls `Open`. The shared Temporal Driver has
request carriers, worker namespaces, task queues, and Nexus routes solely from those prepared
bindings.

Public static admission exposes `testpilot.PreparationError` with a stable category, bounded input
path, and human-readable detail through `errors.As`. It covers Catalog, Profile, Program, and
Contract rejection, including correlated Contracts; ProtoJSON decoding and runtime failures keep their
own contracts. See the canonical [facade guidance](../common/testing/testpilot/README.md#preparation-diagnostics)
and [public diagnostic type](../common/testing/testpilot/preparation_error.go).

The worker's private activation state composes prepared expression evaluation, outcome validation,
immutable local references, and cumulative work accounting. SDK commands, futures, replay, delivery,
and cancellation remain adapter-owned. See the [Temporal Driver contract](../common/testing/testpilot/temporal/README.md)
and [worker ownership](../common/testing/testpilot/temporal/worker/README.md).

The Executor appends monotonic immutable Run Events. Each event has a unique source identity and
causal references to prior sources. The Evaluator observes the appended copy synchronously and uses
the same prepared Contract for offline evaluation. It checks deadline expiry before every
transition, keeps captures rule-local and Run-local, and records exact supporting event sequences.
Private Slots never become evidence automatically.

Stop prevents new controller dispatch and activation reservation, then cancellation, bounded drain,
and cleanup proceed through owned handles and a fresh cleanup context. Disposition, cleanup status,
and Verdict remain independent; a proved violation is not erased by cleanup failure. After closure,
late completion and Driver diagnostics cannot mutate returned data.

The exploration campaign's Go side is two small packages over that boundary. `tools/umpire/binding`
is the deployment binding `umpire-run` performs, split into what a campaign opens once (the
frontend connection, the provisioned namespace, task queue and Nexus endpoint, the method catalog)
and what each candidate opens for itself (the derived Profile, `Prepare`, one composite Driver with
its own SDK worker); `umpire-run` binds one Case through the same two steps, and `binding.Prepare`
is the candidate-scoped part alone, touching no deployment, which admission and offline replay use.
`tools/umpire/campaign`
is the client of the exploration bridge and the serial path for one candidate: decode the Case the
bridge handed out, bind it (preparation first, so a rejected Case opens no Driver and creates no
Run), run it once, observe its cleanup, and hand the closed Run back to the bridge, which alone
says what it credited. One request is outstanding at a time; the client refuses a second `next`
before `observe` without writing a frame, and a binding or execution failure leaves the candidate
outstanding rather than inventing an observation. The coordinator's state is one `Session` value
(idle, planning, preparing, running, observing, finished) whose every transition consumes the
state it starts from, with the campaign's caps -- candidates, aggregate Case bytes, aggregate Run
Events, one Run's time, the report's bytes -- enforced before the action each bounds; `Drive` runs
that loop to its terminal and never recovers, resumes or persists. The report is a function of the
bridge's and the binder's answers: the same answers give the same report, and a stop changes only
where it ends. `umpire-fuzz run` reports each counterexample by its proposal's digest and path and
writes the bytes only under a `--promotion-root` outside the model; with `--record-root` it also
records each violated Run's Case and recorded Run as they close, through the per-candidate hook
`DriveRecording` offers, retaining nothing in the report.

`tools/umpire/replay` admits one such subject: a Case in its canonical form (the renderer's compact
ProtoJSON, or its persisted re-indentation, decided once in `tools/umpire/internal/casefile`) and a
recorded Run, checked to be the Case's, closed by the Monitor's stop with a violated Verdict, its
supporting sequences naming the Run's events, prepared under the recorded Profile name with the
recorded catalog and bindings (`stale` otherwise), and replayed offline to the recorded Verdict. The
subject's identity is the canonical bytes' SHA-256; its violation key is Contract-relative and read
in Definition IDs (the violated rules, their terminal states, their violating evidence as the
evaluation names it), never the Case or Run identity, sequences, times or the Verdict's accumulated
support, so a rerun of the Case and a reduced candidate can share it.

## Artifact ownership and tests

Semantic owners depend on `Umpire.OutcomeClassification` for neutral classifier and projection
vocabulary and on `Umpire.KnownGap` for carry contracts. Concrete stage classifiers, exhaustive
proofs, the Implementation Link stage not-run marker, Evidence's lossy admission mapping, and
Result Artifact's exact carry mapping stay with their semantic owners. The inventory consumes those
contracts and owns its catalogs; the Temporal inventory tool assembles, validates, and renders them.

Inventory consumers explicitly import `Umpire.Inventory` or a focused inventory module;
the ordinary `Umpire` umbrella does not import it. `make lint-model` rejects direct and transitive
production Umpire paths into the inventory, including facade, helper, external, and test-fixture
bridges, and keeps `Umpire.OutcomeClassification` limited to Lean's `Init` foundation. Dedicated
inventory tests, including Search Known Gap catalog tests, remain valid consumers. See the
[ownership guidance](Umpire/ARCHITECTURE.md#artifact-and-generated-view-boundaries).

`model/INVENTORY.md` is generated by `umpire-inventory`. The retained
Plan Generated Views are generated by `umpire-gen-regression-views`. Case conformance fixtures
are rendered by `umpire-case` and published transactionally by
`umpire-gen-case-runtime-conformance`.

The Case fixture check creates the complete candidate tree in a physical temporary directory,
validates all Cases and expected projections, then recursively diffs it with the checkout.
Promotion is a separate target. Go tests consume checked-in fixtures without invoking Lean or
rewriting data.

`TemporalModelTests` imports the ordinary Temporal model tests. `UmpireTests` imports reusable
Umpire tests, including the scenario-neutral promotion source checks. The experimental test root
was retired with the first-generation Nexus models (fn-86 R6); no Temporal-side compatibility
family remains.

The full regression boundary is:

```sh
make umpire-build-model
make umpire-check-regression
```

It checks owner-managed outputs, active vocabulary, the complete package-local Go suite, the exact
tagged `^TestUmpire` live selector and inherited failure identities, the Case facade corpus, generic
promotion, and the model roots. It intentionally defines no broad generated-Lean drift policy and
adds no GitHub Actions coverage.

## Superseded runtime history

Before fn-64, model docs described a portable plan, caller-specific execution adapter, resident
service, and separate Run Evaluation pipeline. Those runtime interfaces were removed. Historical
design documents mark them explicitly as superseded; they are not current package or command
boundaries.
