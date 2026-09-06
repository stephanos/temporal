# Extract Testpilot from Umpire

## Conversation Evidence

> user (turn 1): "tools/umpire/internal/ir and tools/umpire/internal/execution etc define a contract of how to execute \"anything\" and verify it. I think it should be come part of the server and be detached from Umpire itself. makes it more managable and defines a clear boundary with umpire then. dodes that makse sense? suggest a name and location for code"
> user (turn 2): "suggest better names"
> user (turn 3): "isn't this about instructing, remote controlling, pupeteering the server?"
> user (turn 4): "wouldn't we want to land it in common/testing/<name>? and then have a functional test-specific adapter in tests/testcore/<name>? then common/testing can be used for a canary impl too"
> user (turn 5): "use testpilot as name"
> user (turn 6): "yes, let's write a new spec for this change"
> user (turn 7): "update .plans/UMPIRE4_ORDER.md to execute this new spec before fn-68"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 100% [paraphrase] -->

Separate the Case protocol, execution, and verification from Umpire model authoring. Umpire remains responsible for describing behavior and producing Cases; Testpilot becomes the server-owned module that defines the Case protocol, admits a bounded Case, directs an authorized environment, records what happened, and evaluates the Case Contract.

The separation gives functional tests and future production canaries one execution module while allowing each environment to own its remote-control adapter. It also gives Umpire a clear handoff: Umpire produces Testpilot Case data, while protocol ownership, execution state, and environment authority remain outside Umpire.

## Architecture & Data Models
<!-- scope: technical -->
<!-- Architecture & Data Models: 80% [paraphrase], 20% [inferred] -->

The deep module is named Testpilot and lives at `common/testing/testpilot`. It owns the reusable Case representation bindings, descriptor and expression admission, Program scheduling, immutable value flow, event recording, Contract preparation and evaluation, Run closure, and Verdict production. Its public interface remains small while its scheduler, recorder, expression machinery, and evaluator remain internal.

Testpilot also owns the protobuf contract. Proto sources live at `proto/internal/testpilot/v1`, use the protobuf namespace `temporal.server.testpilot.v1`, and generate the Go package at `api/testpilot/v1`. The former Umpire proto source, namespace, generated Go package, helper files, and import paths are removed after consumers and generated artifacts migrate.

An environment implements Testpilot's Driver interface. The functional-test implementation lives at `tests/testcore/testpilot` and owns functional cluster access, namespaces, credentials, SDK workers, RPC channels, and other test-harness mechanics. Server and worker authority remain separate inside that adapter even when composed as one Driver.

A canary implementation may supply another Driver without importing the functional test harness. Testpilot owns no canary policy, credentials, leases, scheduling, publication, or deployment behavior; those remain with the canary caller.

The final dependency direction is Umpire Producer to Testpilot's canonical Case protocol to Testpilot execution to an injected Driver. Testpilot must not import Umpire tooling or either environment adapter, and adapters must not define Program or Contract meaning.

## API Contracts
<!-- scope: technical -->
<!-- API Contracts: 65% [paraphrase], 35% [inferred] -->

- `testpilot.Prepare(case, profile)` performs static admission without Driver access or target I/O and returns an immutable prepared value.
- The prepared value's `Run(ctx, driver)` method verifies Driver identity and authorization, creates fresh per-Run state, executes the admitted Program, records immutable events, evaluates the admitted Contract, and returns the existing Run and Verdict shapes.
- Driver and per-Run Session interfaces expose only authorized effects, reservations, completion, cleanup, and diagnostics required by the admitted Program. They cannot replace the evaluator, synthesize observations, alter the Contract, or reinterpret a Verdict.
- Case, Profile, Program, Contract, Run, Verdict, and value message names, field numbers, enum numbers, cardinalities, defaults, validation semantics, and serialized data remain compatible. Proto file identities, protobuf full names, generated Go import paths, and descriptor identities intentionally move from Umpire to Testpilot.
- Existing checked-in Case data and generated views are regenerated only where the namespace move changes an owned descriptor or derived identity. Every changed artifact must trace to that expected rename; no compatibility translation format or dual protocol is introduced.
- Umpire consumers migrate to Testpilot's interface. Any temporary forwarding facade exists only during the migration and is absent at completion.

## Edge Cases & Constraints
<!-- scope: technical -->
<!-- Edge Cases & Constraints: 45% [paraphrase], 55% [inferred] -->

Preparation must reject malformed, unsupported, over-limit, unauthorized, or internally inconsistent Cases before environment I/O. Run preflight must reject nil, typed-nil, mismatched, or unauthorized Drivers before effects begin. Existing deterministic error categories and failure precedence remain stable.

Cancellation, concurrent independent Runs, recorder atomicity, bounded cleanup, quarantine, replay, incomplete execution, proven-violation precedence, descriptor compatibility, and defensive ownership of mutable protobuf values retain their established behavior. The move must not add retries, instructions, evidence sources, callbacks, or runtime work.

Functional and canary Drivers cannot import each other. A Driver may perform only effects authorized by the prepared Profile and Program, and all Contract evidence must still enter through Testpilot's recorder. A Driver failure cannot manufacture success or erase an already committed violation.

The existing older common testing Umpire framework is not merged into Testpilot. Historical implementations and open downstream specifications remain unchanged except where their active Case protocol or Case Runtime dependency must point to Testpilot.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Testpilot is the single owner of the Case protobuf contract, generic Case admission, Program execution, recording, and Contract evaluation under the server's shared testing layer; it has no dependency on Umpire tooling, the functional test harness, or a canary implementation. Proto sources use the top-level internal Testpilot namespace rather than the Temporal-server API nesting. Errors: forbidden dependency directions or a second protocol, execution, or evaluation owner fail the migration. [user]
- **R2:** The public sequence is `Prepare(case, profile)` followed by `prepared.Run(ctx, driver)`, with immutable prepared state and fresh Run-local state. Preparation performs no target I/O; Run validates Driver identity and authorization before effects. Errors: malformed or over-limit Case/Profile input and nil, typed-nil, mismatched, or unauthorized Drivers fail before target effects. [inferred]
- **R3:** Driver is the only environment-control seam. It exposes the existing bounded effect, reservation, completion, cleanup, and diagnostic capabilities without authority to change Program or Contract semantics, recorded evidence, or Verdict interpretation. Errors: undeclared effects, observations, retries, or evaluator substitution are rejected or structurally impossible. [paraphrase]
- **R4:** The functional test adapter is owned by the functional test-core layer and preserves the separate server and worker authorities while passing the existing focused, conformance, and live Case Runtime tests through Testpilot. Errors: crossed authority, lost cleanup, changed failure identity, or dependence from Testpilot back into the adapter fails verification. [user]
- **R5:** A non-functional caller can implement a separate Driver using only Testpilot's public interface, proving that a future canary does not need the functional test harness. Errors: canary policy, credentials, leases, publication, or deployment behavior entering Testpilot violates the seam. [user]
- **R6:** Protobuf message and enum numbers, field shapes, canonical Case data, admission outcomes, authorized effects, Run events, Verdicts, diagnostics, cancellation and cleanup precedence, concurrency behavior, and bounded 10x-load characteristics remain unchanged across the migration corpus. Proto file names, protobuf full names, generated import paths, and descriptor-derived identities change only as required by the Testpilot namespace migration and are exhaustively accounted for. Errors: any unexplained byte, identity, ordering, error-category, resource-bound, descriptor, or generated-artifact difference fails completion. [inferred]
- **R7:** All active Case protocol and runtime consumers, generators, fixtures, descriptor catalogs, and architecture rules name Testpilot as the owner; the former Umpire proto package and runtime implementation are removed without aliases or a permanent forwarding layer, and fn-68 depends on this migration before execution. Existing comments are preserved. Errors: stale active imports, proto names, generated files, fixtures, or documentation; parallel protocol/runtime paths; an unaccounted consumer; or fn-68 executing first prevents completion. [user]

## Boundaries
<!-- scope: business -->

- No new Program instruction, Contract rule, evidence source, retry behavior, or runtime capability. [inferred]
- No Case message/field/enum redesign or wire-version change beyond the approved proto source, namespace, descriptor, and generated-package rename. [user]
- No change to Umpire's Lean authoring, planning, Query, Producer, or canonical Case semantics. [paraphrase]
- No production canary implementation, policy, credentials, leases, reconciliation, publication, or deployment work. [paraphrase]
- No redesign or consolidation of the older common testing Umpire framework. [inferred]

## Decision Context
<!-- scope: both -->
<!-- Decision Context: 85% [paraphrase], 15% [inferred] -->

- Testpilot describes the module's role as the pilot that owns the executable Case protocol, directs a system under test, and observes the resulting Run; Umpire remains the behavior author and Producer. [user]
- The shared server testing layer owns reusable execution and verification, while the functional test-core layer owns the concrete functional-cluster Driver. [user]
- Environment adapters remain caller-owned so the same Testpilot module can later support a canary without depending on functional-suite infrastructure. [user]
- This is a fresh follow-up to the completed Case Runtime rather than a rewrite of its behavioral contract; that implementation and corpus provide the compatibility baseline. [inferred]
- The migration precedes fn-68 so the first Nexus3 success demonstration exercises the final ownership seam. [user]
- The Testpilot proto sources use a short top-level internal path rather than the existing nested Temporal-server API path; generated Go code remains under the server API output tree with a Testpilot package identity. [user]

## Requirement coverage

| Requirement | Planned task |
| --- | --- |
| R1 | TBD — populate via `/flow-next:plan` |
| R2 | TBD — populate via `/flow-next:plan` |
| R3 | TBD — populate via `/flow-next:plan` |
| R4 | TBD — populate via `/flow-next:plan` |
| R5 | TBD — populate via `/flow-next:plan` |
| R6 | TBD — populate via `/flow-next:plan` |
| R7 | TBD — populate via `/flow-next:plan` |
