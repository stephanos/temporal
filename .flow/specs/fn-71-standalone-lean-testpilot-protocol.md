# Standalone Lean Testpilot protocol

> HTML render lens (local, ignored): `.flow/artifacts/fn-71-standalone-lean-testpilot-protocol/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Goal & Context
<!-- scope: business -->

A Lean Producer should be able to author an executable Testpilot Case without importing Umpire modeling machinery or Temporal scenarios. Today the Testpilot protobuf is the Go wire authority, but Lean separately maintains protocol-shaped `Umpire.Case` types and a handwritten `Temporal.Testpilot.TestpilotProtoJSON` encoder. The parallel representations have already drifted: the advertised `Umpire.Case.ProtoJSON` emits a retired envelope, and the working encoder performs context checks that the protobuf expresses structurally through distinct Program and Contract expression messages.

Use the user-approved [`Lean-zh/protobuf`](https://github.com/Lean-zh/protobuf) package to derive Lean wire types and ProtoJSON behavior from the existing Testpilot `.proto` import closure. Make those schemas the sole structural source of truth. Keep generated representation details behind a small, developer-friendly Testpilot authoring facade so Nexus3 and other Producers construct schema-valid Cases without writing raw JSON or verbose generated oneof records.

A prototype must prove the package against this repository's pinned Lean toolchain, local `protoc`, Testpilot use of `google.protobuf.Any`, Go decoder, and byte-determinism needs before the migration proceeds. This implements architecture-review finding 2 and the producer-neutrality boundary of SEM-18; it does not establish model-to-Contract semantic correspondence.

This spec is independently executable. fn-73-explicit-environment-binding-for consumes its protocol boundary. The ongoing Nexus3 checked-lowering work is a coordinated consumer, not a hard prerequisite: re-anchor to the current producer declarations before migration and avoid duplicating concurrent work.

## Quick commands

```bash
(cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot)
(cd model && mise exec -- lake exe modelLintTests)
make umpire-check-case-runtime-conformance
mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/...
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

## Architecture & Data Models
<!-- scope: technical -->

Pin `Lean-zh/protobuf` to one reviewed revision compatible with the model's Lean 4.33.1 toolchain. Record and enforce the supported `protoc` input contract in the generation/build path. Do not add another protobuf runtime or extend the existing descriptor-catalog generator to create a competing Testpilot representation.

`Testpilot.Protocol` owns the generated or compile-time-generated Lean declarations for exactly the Testpilot protobuf import closure rooted at `proto/internal/temporal/server/api/testpilot/v1/case.proto`. The checked-in `.proto` files remain the only definitions of messages, enums, oneofs, presence, field numbers, JSON names, bytes, and `Any`. Generation must be deterministic and reproducible; stale generated artifacts must fail an existing or new focused drift check if generated Lean files are checked in. The prototype selects the library's supported generation mode with the smallest reliable build dependency; it must not load the entire Temporal protobuf tree unnecessarily.

`Testpilot.Authoring` is a deep, handwritten convenience module over the generated types. It provides stable, concise smart constructors for Cases, Programs, Contracts, values, paths, expressions, instructions, monitors, Run coordinates, limits, Verdict data, and provenance. Constructors return generated protocol values directly and preserve the protobuf's distinct `ProgramExpression` and `ContractExpression` types. The facade must not duplicate protocol structures, expose raw `Lean.Json` as the normal construction API, or offer a generic expression escape hatch. Nexus3-specific macros and lowering remain beside Nexus3 and call this neutral facade.

`Testpilot.ProtoJSON` is a thin policy wrapper around `Protobuf.Json`, selecting explicit print options and the generated descriptor/type resolver needed for `google.protobuf.Any`. It contains no field-by-field serializer. Project-level canonicality means the selected options and implementation emit deterministic bytes for equal generated messages. Standard ProtoJSON compatibility is supplied by the library and verified by Go interoperability. If library field ordering differs from the retired handwritten output, accept one reviewed fixture migration rather than retaining a second encoder; preserve logical Case identity, list order, values, provenance bytes, Go admission, and execution behavior.

The dependency direction is `Umpire producer -> Testpilot` and `Temporal producer -> Testpilot`. Neither `Testpilot.*` nor anything it imports may import `Umpire.*` or `Temporal.*`. Preserve ordinary `Shared` and standard-library foundations. Register the package/library roots and enforce direct and transitive independence through the existing import-graph checker.

`Testpilot.Case` is the generated Case envelope: version, case identity, generic provenance, Program, and Contract. Generic provenance contains producer identity, producer version, and opaque producer bytes. Umpire continues to own Definition bindings, Behavior Fingerprints, source information, Known Gaps, and their deterministic encoding into `producerData`. `Umpire.Case.Compiler` remains producer assembly with its existing checked-input and unsupported-lowering errors, but lowers into the generated Testpilot protocol through the authoring facade.

Go admission remains authoritative for descriptor resolution, bounds, identity, scope, environment, and executable semantics through `testpilot.DecodeCaseProtoJSON` followed by `testpilot.Prepare(case, profile)`. Generated Lean types guarantee protobuf structure and context ownership; they do not make every structurally valid Case executable.

## API Contracts
<!-- scope: technical -->

The existing Testpilot protobuf remains the wire authority; this spec does not change its version or schema. Generated Lean declarations must represent its full current message graph, including recursive messages, oneofs, proto3 presence, enums, bytes, int64 JSON strings, and `google.protobuf.Any`.

The public authoring API returns generated message values. Every Program expression-bearing constructor accepts only the generated `ProgramExpression`; every Contract expression-bearing constructor accepts only the generated `ContractExpression`. Shared ergonomic combinators may be overloaded or separately named, but their result context must remain statically known. Program Run identity uses the protobuf `run` alternative. Contract Run identity and event coordinates use the protobuf `run_event` alternative. Invalid cross-context references must fail Lean elaboration before serialization.

`Testpilot.ProtoJSON.canonical` accepts the generated Case type and returns either deterministic ProtoJSON text or a typed library/wrapper error. It delegates encoding to `Protobuf.Json` with explicit, centralized print and resolver options. Callers handle every error. No code may silently omit an unsupported rule, expression, value, `Any`, or provenance field.

The one-time migration compares old and new artifacts semantically through strict Go decoding. Any byte diff must be attributable only to the selected ProtoJSON representation, reviewed in managed fixture diffs, and stable on repeated rendering. After cutover, the new bytes are the canonical fixture bytes and staleness checks protect them.

The Umpire provenance encoder retains the exact logical definitions/sources/knownGaps payload, including order and optional subject/detail fields, as opaque bytes. Generic Producers can supply empty or arbitrary bytes without importing Umpire or conforming to that payload schema.

## Edge Cases & Constraints
<!-- scope: technical -->

The prototype is a hard adoption gate. It must exercise the repository's exact Testpilot schema with Lean 4.33.1 and the available `protoc`, rather than a toy message. It covers nested oneofs, recursive Program/Contract expressions, `google.protobuf.Any`, empty/default presence, bytes including non-UTF-8 data, signed and unsigned integer boundaries, enum spelling, and repeated deterministic ProtoJSON rendering. The result must decode with Go's strict Testpilot decoder. If any required behavior is unsupported or unreliable, stop and re-plan; do not fall back silently to handwritten protocol types or a field-by-field JSON codec.

Nested wrong-context references must be rejected just as direct ones. Paths, comparisons, negation, conjunction, and disjunction cannot conceal a Slot or instruction outcome inside a Contract, or an Observation, capture, or event-only coordinate inside a Program. Negative wire fixtures that require schema-invalid or context-invalid data are authored at the raw JSON/Go admission boundary after valid positive serialization; the public Lean authoring types are not weakened for tests.

`Any` serialization must resolve generated Testpilot and required well-known descriptors through an explicit resolver. Unknown type URLs and malformed payloads return errors. Opaque provenance must round-trip byte-for-byte, including empty and non-UTF-8 values.

Preserve all existing comments outside changed code, logical Case IDs, model fingerprints, list ordering, supported execution behavior, and stable Run/Verdict projections. A one-time object-field-order or default-elision fixture change is allowed only with decoded-message equivalence evidence. Do not change the Lean version or add any third-party dependency beyond the explicitly approved pinned protobuf package.

This work adds no execution service, mutable cross-Run state, resource authorization, or crash-recovery protocol. Existing runtime limits and admission continue to bound larger inputs. Generated encoding remains linear in message size; measure build and executable-size regressions during the prototype and record material changes before adoption.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A prototype pins `Lean-zh/protobuf` and proves the exact Testpilot Case schema builds on Lean 4.33.1 with the repository's supported `protoc`, including recursive messages, oneofs, presence, bytes, enums and `google.protobuf.Any`. It emits deterministic ProtoJSON accepted by Go's strict decoder. Errors: unsupported schema/library/toolchain behavior stops the spec for re-planning; no handwritten fallback is introduced. The prototype records build-time and executable-size impact.
- **R2:** The Testpilot `.proto` import closure is the sole structural source of truth for Lean wire declarations, with a deterministic generation/load command and a staleness check for checked-in output. Errors: schema changes cannot leave compiling stale Lean protocol declarations, and generation may not silently broaden to unrelated Temporal schemas.
- **R3:** A first-class `Testpilot` Lean facade exposes the generated protocol, authoring helpers and ProtoJSON wrapper. Executable import-graph checks reject direct and transitive imports from `Testpilot.*` to `Umpire.*` or `Temporal.*`. Errors: unknown/unclassified modules cannot hide a forbidden transitive path.
- **R4:** Developer-friendly authoring helpers construct generated messages directly while preserving distinct Program and Contract expression types. Positive tests cover every reference family and recursive combinator; compile-failure tests reject direct and nested cross-context references. Errors: no coercion, broad handwritten expression union, raw-JSON constructor, or public generated-oneof bypass in the ergonomic facade defeats context safety.
- **R5:** `Testpilot.ProtoJSON.canonical` delegates to `Protobuf.Json` with centralized print and resolver options and contains no handwritten field serializer. Cross-language tests cover both Run identity forms, nested expressions, integer boundaries, bytes, optionals, enums, `Any`, and monitor rules. Errors: unresolved `Any`, invalid values, and serialization failures propagate; nothing is dropped. Repeated equal inputs produce byte-identical output.
- **R6:** Umpire compiler output uses generated Testpilot Cases and generic provenance while preserving the logical definitions, fingerprints, sources, Known Gaps, ordering, optional fields, checked-input rules, and unsupported-lowering diagnostics in opaque `producerData`. Errors: Testpilot and Go do not require or interpret Umpire payload contents.
- **R7:** Current Umpire, Temporal, Nexus3, conformance and synthetic Producers use the authoring facade and library ProtoJSON path. A Testpilot-only synthetic Producer passes Go Decode plus Prepare under a local no-I/O Profile. Managed fixtures receive either byte-identical output or one reviewed semantic-equivalence migration, with unchanged logical Case identity and execution/Verdict behavior. Errors: malformed wire and invalid bounds retain their Go rejection coverage.
- **R8:** Parallel handwritten Testpilot wire types and independent Umpire/Temporal serializers are removed after migration; compatibility exports, if temporarily retained, delegate to generated types and the single library codec. Active documentation names the protobuf schema, generated protocol, authoring facade, opaque provenance, and Go admission owners. Focused Lake, generation-drift, import-graph, fixture, Go `-tags test_dep`, model lint and Go lint gates pass with no new proof placeholders or trust dependencies.

## Boundaries
<!-- scope: business -->

- Checked model-to-Case lowering and the Nexus3 success demonstration remain their owning Nexus3 scope; this spec migrates their protocol construction seam.
- Environment-value binding and retargeting remain fn-73-explicit-environment-binding-for scope.
- Moving the Temporal driver remains fn-72-extract-the-reusable-temporal-testpilot scope.
- Activation-state deepening and public preparation diagnostics remain fn-74-deepen-testpilot-worker-activation scope.
- Target semantic/elaboration separation remains fn-75-separate-lean-target-semantics-from scope.
- Semantic-inventory dependency inversion remains fn-76-make-lean-semantic-inventory-consume scope.
- No new protobuf version, runtime semantics, public Monitor interface, RPC stub generation, external package release, or generated Temporal API catalog redesign.
- No blanket guarantee that every Lean-constructible Case is Go-admissible; generated types provide structural and context safety while Prepare owns semantic admission.

## Decision Context
<!-- scope: both — conditionally substructured -->

The Testpilot protobuf already expresses producer-neutral provenance and distinct expression contexts. Generating Lean declarations from that schema removes an unnecessary second protocol definition and makes schema changes visible at build time. `Lean-zh/protobuf` now supplies the missing Lean code generation, binary runtime, reflection and ProtoJSON support, including `Any`; extending the repository's descriptor catalog generator or maintaining handwritten codecs would duplicate those responsibilities.

Generated APIs are often verbose, so Producers should not depend on generated record/oneof layout throughout authored model code. A small Testpilot authoring module gives stable names and pleasant constructors while returning the generated wire values directly. Feature-specific Nexus3 syntax remains beside Nexus3 and lowers through that neutral seam.

ProtoJSON defines semantic JSON mapping but does not promise preservation of the retired encoder's object-field order. Requiring old bytes would force continued custom serialization. The migration therefore permits one explicit fixture-byte transition after strict decoded-message equivalence and deterministic-output checks; all subsequent fixtures are protected by the normal staleness gate.

Opaque bytes keep Testpilot independent of Umpire's model vocabulary. Runtime semantic validation remains with Prepare because schema-derived types cannot prove descriptor availability, environment compatibility, limits, or complete executable graph validity.

## Early proof point

Task fn-71-standalone-lean-testpilot-protocol.1 is a hard prototype gate. It imports the exact Testpilot schema through the pinned library, constructs representative generated values, serializes them through `Protobuf.Json`, and verifies deterministic Go decoding. Tasks 2-7 must not proceed if the prototype exposes an unsupported required feature or unacceptable build/runtime cost; update this spec from measured evidence instead.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Exact-schema protobuf-library prototype and adoption gate | fn-71-standalone-lean-testpilot-protocol.1 | — |
| R2 | Schema-derived Lean declarations and drift protection | fn-71-standalone-lean-testpilot-protocol.2 | — |
| R3 | Standalone facade and transitive import enforcement | fn-71-standalone-lean-testpilot-protocol.2, fn-71-standalone-lean-testpilot-protocol.3 | — |
| R4 | Developer-friendly context-safe authoring API | fn-71-standalone-lean-testpilot-protocol.2, fn-71-standalone-lean-testpilot-protocol.3 | — |
| R5 | Sole library-backed deterministic ProtoJSON path | fn-71-standalone-lean-testpilot-protocol.1, fn-71-standalone-lean-testpilot-protocol.3, fn-71-standalone-lean-testpilot-protocol.5 | — |
| R6 | Opaque generic provenance and exact Umpire semantics | fn-71-standalone-lean-testpilot-protocol.4 | — |
| R7 | Producer migration and independent Go admission proof | fn-71-standalone-lean-testpilot-protocol.5, fn-71-standalone-lean-testpilot-protocol.6 | — |
| R8 | Retired parallel implementations removed and verified | fn-71-standalone-lean-testpilot-protocol.7 | — |
