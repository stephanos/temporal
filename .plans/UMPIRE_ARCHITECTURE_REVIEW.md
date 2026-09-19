**Umpire architecture review — 2026-09-06**

Scope: the current working tree's `model/`, `tools/umpire`, `api/testpilot`,
`common/testing/testpilot`, and the consuming `tests/testcore/testpilot` driver.
The checkout includes ongoing Nexus3 edits; those are experimental authoring work, not evidence
that the established model-to-runtime integration is complete. This review proposes changes;
it does not implement them or create implementation tasks.

**Assessment**

The architecture has a strong execution core and a useful separation of product and implementation
models. It does not yet form a continuous refinement from checked model to executable Case.
The most important work is to complete the semantic connection already planned in Nexus3 and give
Testpilot its own Lean protocol module. Adding more wrappers around the existing interfaces would
not solve either issue.

There are currently two substantial paths:

```text
Feature / System models -> checked Target / Property / Behavior / Query -> planning
                        -> Observation / ImplementationLink / model Property evaluation

Handwritten Temporal.Testpilot Programs and Contracts
    -> Umpire.Case.Compiler assembly
    -> Temporal.Testpilot.TestpilotProtoJSON translation
    -> testpilot/v1 -> Prepare -> PreparedCase.Run -> Temporal driver
```

The second path is valid standalone Case authoring. Its existence does not establish that its
Contracts preserve the Properties in the first path.

**1. High priority: model-to-Case lowering is an unimplemented semantic seam.**

This gap is already explicitly owned by
[`fn-68.2`](../.flow/tasks/fn-68-minimal-nexus3-success-demonstration.2.md), with fixture/live
integration in [`fn-68.3`](../.flow/tasks/fn-68-minimal-nexus3-success-demonstration.3.md).
The planned `Temporal.Feature.Nexus3.Testpilot` adapter consumes the checked Query, its meaning and
selected witness, derives provenance and the success monitor, rejects unsupported semantic forms,
and removes the independent async Case. Those requirements directly address this finding for the
success slice, including separating model-step bounds from execution timeouts. At review follow-up,
the adapter file was absent and the tracked tasks remained `todo`. Treat the recommendations below
as acceptance criteria for that existing work, not a proposal for a competing compiler project.
The initial review should have distinguished this planned remediation from the current code gap.

[`Umpire.Case.Compiler`](../model/Umpire/Case/Compiler.lean) takes a complete `Program` and
`ContractLowering` values containing already-authored monitor rules. `Input` does not accept a
`CheckedProperty`, checked Query, selected model trace, or checked evidence interpretation.
The module explicitly acknowledges that it has no checked-Property-to-lowering producer.
Its checks cannot establish correspondence between a supplied monitor and a model Property.

[`Temporal.Testpilot`](../model/Temporal/Testpilot.lean) imports that assembly module and separately
authors `asyncProgram`, `asyncRule`, and string-valued behavior fingerprints. It does not import the
Nexus Feature/System definitions. A change to the modeled lifecycle or Property therefore need not
change the runtime Contract. Labeling provenance `checked-model` does not establish that connection.

Extract a Temporal-owned lowering module whose inputs are checked model selections and explicit
action/evidence bindings. It should own Action-to-Program realization and Property-to-Contract
translation, return typed unsupported errors, and derive provenance from actual checked inputs.
Keep direct Case construction available for independent non-model Producers.

The proof obligation is more precise than general equivalence: for the supported bounded fragment,
state how accepted runtime evidence corresponds to model traces and how Contract conclusions relate
to model Property evaluation. Model-step bounds and elapsed-time horizons require an explicit
relationship; a compiler cannot silently equate them. Test missing, ambiguous, reordered and
uncorrelated evidence as well as successful traces. Reject unsupported clauses rather than
reimplementing them in drivers.

**2. High priority: the Lean Testpilot protocol has split ownership and a stale public codec.**

[`Umpire.Case.ProtoJSON.canonical`](../model/Umpire/Case/ProtoJSON.lean) emits a `metadata` field.
The actual [`Case` protobuf](../proto/internal/temporal/server/api/testpilot/v1/case.proto) has
`provenance` and opaque `producer_data`; Testpilot's decoder rejects unknown fields.
The documented generic codec consequently does not produce the current wire envelope.

The working path uses
[`Temporal.Testpilot.TestpilotProtoJSON`](../model/Temporal/Testpilot/TestpilotProtoJSON.lean).
It does substantially more than formatting: it adapts the schema and rejects Program expressions
containing Contract-only references, and vice versa. The shared Lean `ValueExpression` allows these
invalid combinations. A lower, supposedly Temporal-specific module repairs an overly broad upper
representation. This directly matches the abstraction leak described in the review request.

Extract `Testpilot.*` in Lean, owning the current Case/Program/Contract vocabulary and one canonical
wire codec. Give Program and Contract expressions distinct types, or index one expression language
by its allowed reference context. Keep model definitions, fingerprints and Known Gap encoding in
the Umpire producer's provenance module. `Umpire -> Testpilot` is then a deliberate producer
dependency; Testpilot imports neither Umpire's modeling machinery nor Temporal scenarios.

Acceptance: a synthetic producer imports only Testpilot, emits a Case accepted by Go, and cannot
construct a Contract Slot reference through the normal typed interface. Cross-language fixtures
must exercise this public codec. Updating the Temporal codec alone must not leave an advertised
generic codec silently stale.

**3. High priority before canary: environment binding is only partially symbolic.**

In [`Temporal.Testpilot`](../model/Temporal/Testpilot.lean), `historyAssignments` and `asyncProgram`
embed namespace `default`; workflow-start requests embed a physical task queue. Meanwhile worker
activation declarations use symbolic task-queue roles, resolved independently by the driver.
[`worker.validCarrierBinding`](../tests/testcore/testpilot/worker/carrier.go) checks that the concrete
request namespace and queue match the configured worker binding.

These checks correctly reject a mismatch. The architectural problem is that selecting a canary
Profile/Driver with another namespace or queue cannot by itself retarget the Case. The caller must
also know which request literals to change, or introduce a request-rewriting workaround.

Define one explicit environment-value binding mechanism, resolved from an immutable authorized
snapshot, that both request construction and worker activation consume. Resolve it during binding
or preparation, before execution. Distinguish bindable resource identities from values whose exact
contents are the behavior under test. If environment-specific Case production is intentional,
make those inputs explicit and derive all uses from them; do not claim byte-identical portability.

Acceptance: the same semantic selection runs against two namespace/queue bindings with no driver
request rewriting and no independently maintained request literals.

**4. Medium priority: reusable Temporal execution belongs outside the test harness.**

[`tests/testcore/testpilot`](../tests/testcore/testpilot/README.md) already implements reusable
transport, SDK registration, reservation delivery, cancellation, completion capabilities and
quarantine. Its production imports use Testpilot, the SDK, transport libraries and its own packages;
the implementation is not intrinsically a functional-test fixture.

Extract this as a reusable Temporal driver module, with `server`, `worker`, and private `delivery`
remaining together. Functional tests and canary should supply environment configuration and
lifecycle policy to that same implementation. Leave cluster setup, test assertions and test
resource provisioning in `tests/`.

The reservation ledger is already a useful deep internal module. Promoting it into another public
library would expose protocol-sensitive details without a demonstrated independent consumer.
Similarly, preserve the server/worker authority split rather than creating separate scenario
drivers for tests and canary.

**5. Medium priority: the driver implementer's interface is much larger than the caller's.**

[`Prepare` and `Run`](../common/testing/testpilot/prepare.go) provide real depth. However,
[`driver.go`](../common/testing/testpilot/driver.go) exposes prepared instruction plans, traversal
order, reference resolvers, work limits and outcome validation. The
[`worker interpreter`](../tests/testcore/testpilot/worker/interpreter.go) must manage activation
values, subtract work charges, evaluate guards and store validated outcomes.

SDK workflow execution needs its own replay-compatible scheduling and futures; those responsibilities
belong in the SDK adapter. Reference ownership, immutable outcome state and budget accounting are
generic Program semantics, and making every adapter assemble them creates drift risk.

Deepen the existing interface with a pure activation-state abstraction that owns reads, guard/input
evaluation, outcome admission and work accounting. The SDK adapter should supply primitive SDK
results and retain SDK futures and cancellation. This can initially be internal to the reusable
driver; expose an adapter-facing helper only when it has a concrete second consumer. Do not move
workflow execution into the controller's goroutine scheduler.

An additional small interface defect: admission returns structured `internal/ir.Error` values
through public methods, but outside callers cannot name that internal type or its categories.
Expose a stable public preparation diagnostic so canary/reporting callers need not parse strings.

**6. Medium priority: Lean semantic modules need narrower internal seams, not new languages.**

The Target/Property/Behavior/Query distinction is correct: model behavior, claims, allowed trace
shape and bounded questions are different responsibilities. `FiniteTable` and `FiniteMachine`
already concentrate enumeration and kernel construction behind checked interfaces. Preserve them,
and use the existing finite-table path to reduce ordinary authoring plumbing before creating
another semantic frontend. Nexus3's syntax should continue elaborating into those same owners.

Two concrete internal dependency improvements are available:

- [`Target.Language`](../model/Umpire/Target/Language.lean) mixes the checked Target representation,
  authoring occurrence diagnostics, canonical projections and elaborator integration. It imports
  `Lean.Elab.Term`, and even Property's language imports the full Target facade. Separate the
  semantic checked Target/kernel interface from elaboration and codec ownership. Ordinary semantic
  consumers should not need the authoring frontend's dependency surface.
- [`Planning.Engine`](../model/Umpire/Planning/Engine.lean) and
  [`Observation.Evaluation.Types`](../model/Umpire/Observation/Evaluation/Types.lean) import
  `SemanticInventory.Types` to describe their own outcomes and Known Gap mappings. Move general
  outcome classification and gap-carry contracts to their semantic owners or a small neutral module;
  let the inventory consume those declarations. This is a dependency-direction improvement, not
  evidence that the inventory currently overrides behavior.

The established Nexus Implementation Link also repeats representation-level list membership and
enumeration proofs. Semantic finite-mapping coverage lemmas would provide more depth than another
wrapper around the existing witness record. Keep the forward-simulation obligation visible.

**Boundaries worth preserving**

| Module | Assessment |
| --- | --- |
| `api/testpilot/v1` | Good standalone wire contract. Opaque producer provenance is appropriate; Testpilot should not become a Lean model registry. |
| `common/testing/testpilot` | Good execution facade: static admission, immutable prepared state, isolated Runs and evaluator ownership. |
| `internal/ir` | Useful shared typed descriptor/expression module for execution and verification; no demonstrated need for another public extraction. |
| `internal/execution` / `internal/verification` | Good authority split through ProgramView and private Monitor interfaces. Keep recorder, Slot storage and monitor selection private. |
| `Temporal.Feature` / `Temporal.System` | Strong semantic separation, connected through an explicit checked Implementation Link. This is the pattern to emulate at model-to-Case lowering. |
| `tools/umpire` | Mostly appropriate producer/generator ownership. Shared artifact publication already exists in `tools/common/artifactio`; do not extract another publisher. |
| `Shared`, generated API/config catalogs | Appropriate independent foundations. Generated structure remains distinct from handwritten behavior. |

Retain Lean's offline Observation/ImplementationLink/Property analysis as a distinct assurance
path until its connection to runtime Contracts is explicitly implemented. Neither that analysis
nor a planning artifact is interchangeable with a Testpilot Verdict.

**Recommended dependency shape**

```text
Shared <- Umpire semantic modeling <- Temporal Feature / System models
                                      |
                              Temporal model-to-Case lowering
                                      |
                           standalone Lean Testpilot IR / codec
                                      |
                             api/testpilot/v1
                                      |
                         common/testing/testpilot
                                      |
                          reusable Temporal driver
                                /           \
                        functional tests    canary
```

The decisive acceptance test is a model Property change flowing through checked lowering into the
Contract, while switching test/canary environment bindings changes no behavioral interpretation.
Complete fn-68's existing Nexus success slice to establish the first connection before expanding
language coverage. Its scoped lowering does not itself resolve the separate protocol/codec,
environment-binding, or reusable-driver findings.

**Validation**

Source inspection covered the producer path, protobuf schema, public runtime facade, private
execution/verification/IR interfaces, Temporal driver composition and Lean semantic imports.
Observed verification results:

- `CGO_ENABLED=0 go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`:
  all four core packages and all four driver packages reported passing. The aggregate command did
  not pass: conformance publisher tests rejected the macOS `/var` temporary-directory symlink;
  remaining generator checks were stopped before completion.
- With `TMPDIR=/private/tmp CGO_ENABLED=0`, the descriptor exporter, Case conformance generator and
  Lean API generator packages reported passing. The remaining broad generator run was stopped;
  this is not a claim that all Umpire tool tests pass.
- The initial default-CGO run encountered missing `stddef.h` headers.
- `cd model && lake exe modelLint` encountered failures creating `.olean` output files during its
  prerequisite compilation and was stopped. The complete import-graph/lint gate is unverified.
  Disk inspection reported 1.9 GiB available and 100% capacity; the exact cause of the file-creation
  failures was not diagnosed.
- All relative links in this report resolve. No live integration, full Go lint, complete Lean lint,
  or cross-language regeneration gate is claimed. No implementation files were changed by this
  review.
