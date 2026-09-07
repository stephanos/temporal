# Standalone Lean Testpilot protocol

## Goal & Context
<!-- scope: business -->

A Lean Producer should be able to author an executable Case without importing Umpire modeling machinery or Temporal scenarios. Today `Umpire.Case` owns the reusable data vocabulary, while `Temporal.Testpilot.TestpilotProtoJSON` provides the working wire encoder. The advertised `Umpire.Case.ProtoJSON` encoder instead emits the retired `metadata` envelope, which Go's strict decoder rejects. The working encoder also compensates for one overly broad expression type by rejecting Program-only references in Contracts and Contract-only references in Programs.

Make `Testpilot.*` the independent Lean protocol owner, with a context-safe expression interface and one current canonical ProtoJSON codec. Preserve executable behavior and producer provenance through the migration. A synthetic Producer using only Testpilot must produce a Case admitted by the existing Go facade. This implements architecture-review finding 2 and the producer-neutrality boundary of SEM-18; it does not establish model-to-Contract semantic correspondence.

This spec is independently executable. fn-73-explicit-environment-binding-for consumes its protocol boundary. The ongoing Nexus3 checked-lowering work in fn-68 is a coordinated consumer, not a hard prerequisite: re-anchor to the current producer declarations before implementation and migrate whichever have landed without duplicating or blocking that work.

## Architecture & Data Models
<!-- scope: technical -->

`Testpilot` is a first-class Lean library in the existing model Lake workspace and uses the existing toolchain and dependencies. Its public facade exposes Case, Program, Contract and the supporting runtime data vocabulary currently reachable through `Umpire.Case`, including typed values, paths, roles, Slots, Observations, Run coordinates, monitor state, limits and Verdict data. It does not own model definitions, checked queries, source locations, fingerprints, Known Gap interpretation or producer-specific metadata structures.

The dependency direction is `Umpire producer -> Testpilot` and `Temporal producer -> Testpilot`. Neither `Testpilot.*` nor anything it imports may import `Umpire.*` or `Temporal.*`. Preserve ordinary `Shared` and standard-library foundations. If the codec needs the existing ordered canonical JSON facility, move only the reusable facility into a neutral foundation and retain behavior-preserving access for its existing Umpire users; do not duplicate its renderer or pull the Umpire facade into Testpilot. Register the new library and enforce its direct and transitive independence in the existing import-graph checker and tests.

`Testpilot.Case` pairs exactly one Program and one Contract with version, case identity and generic provenance. Generic provenance contains only producer identity, producer version and opaque producer bytes. Umpire owns the existing Definition bindings, Behavior Fingerprints, source information and Known Gap representation and its deterministic encoding into those bytes. `Umpire.Case.Compiler` remains producer assembly with its existing unsupported-lowering errors; it constructs the generic envelope using that producer-owned encoder. This relocation must not strengthen its assurance claim or pretend it derives monitors from checked Properties.

Expose distinct `ProgramExpression` and `ContractExpression` types. Separate recursive types or an expression family indexed by reference context are both acceptable; recursive combinators must preserve context and every expression-bearing Program/Contract field must require the appropriate type. Program references are Slots, instruction outcomes and the Run identity. Contract references are current-event Observations, rule captures and the existing Run Event coordinates, including Run identity. Both contexts retain literals, paths, presence, equality, ordered comparisons, negation, conjunction and disjunction. A Program Run identity must encode as the current wire `run` reference, whereas the Contract Run identity remains a `runEvent` field. No implicit coercion or public unindexed escape hatch may bypass context separation.

The public codec belongs to `Testpilot.ProtoJSON`; its implementation derives from the current working Temporal encoder. Remove the retired schema implementation. Existing Umpire/Temporal module names may survive only as documented forwarding compatibility accessors to the sole current codec, never as separate serializers. Migrate repository consumers and examples to the intended ownership boundary and remove claims that the retired codec is authoritative.

## API Contracts
<!-- scope: technical -->

The existing Testpilot protobuf remains the wire authority; this spec changes Lean ownership and construction safety, not protobuf version or schema. The complete Case envelope retains `version`, `caseId`, `provenance`, `program` and `contract`. Provenance retains exactly `producerId`, `producerVersion` and base64-encoded `producerData`. No `metadata` field or model-specific field is added to the envelope.

`Testpilot.ProtoJSON.canonical` accepts the generic Case and returns deterministic ProtoJSON. Preserve the current field ordering, list ordering, enum spelling, optional-field behavior, integer/string conventions, floating-point spelling, byte encoding and trailing formatting for unchanged valid inputs. Context errors disappear from this canonical path because they are unconstructible through the typed interface. An `Except` return is needed only if a remaining real serialization failure requires it; callers must handle every retained error and must never drop a failed rule or expression.

The Umpire provenance encoder retains the exact current definitions/sources/knownGaps payload, including order and optional subject/detail fields. Generic Producers can supply empty or arbitrary opaque bytes without depending on, or conforming to, that payload schema. Neither Testpilot Lean nor Go assigns model meaning to those bytes.

Go admission continues through `testpilot.DecodeCaseProtoJSON` followed by `testpilot.Prepare(case, profile)`. The synthetic fixture must pass both, using a valid local test Profile without target I/O. Static descriptor, bounds, identity and scope validation remain Go admission responsibilities: the typed expression split guarantees context ownership, not universal Case validity.

## Edge Cases & Constraints
<!-- scope: technical -->

Nested wrong-context references must be rejected just as direct ones: paths, comparisons and Boolean groups cannot conceal a Slot inside a Contract or an Observation/capture/event-only coordinate inside a Program. Compile-failure tests must also cover instruction outcomes in Contracts and non-Run-ID event fields in Programs. Legitimate Run identity remains available in both contexts with its distinct wire encoding.

Preserve all working valid fixture bytes and stable execution/verdict projections. Existing malformed conformance fixtures still need to test Go rejection. If a malformed fixture intentionally requires a now-unconstructible reference, author its corruption explicitly in the negative test boundary after valid serialization or as raw wire input; do not weaken the public Lean types to support negative fixtures.

Opaque provenance must round-trip byte-for-byte, including empty and non-UTF-8 payloads. Umpire's payload remains deterministic JSON by producer choice, not by Testpilot requirement. Removing obsolete serialization must not delete unrelated comments, change model fingerprints or reinterpret Known Gaps.

Do not change the Lean version, add third-party libraries or broaden optional verification imports. This is pure data construction/serialization; it adds no execution service, mutable cross-Run state, resource authorization or crash-recovery protocol. Existing runtime limits and admission bound execution at larger load. Serialization retains the current order of work in input size; this spec adds no caching or new performance subsystem.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A first-class `Testpilot` Lean facade exposes the standalone protocol, and executable import-graph checks reject direct and transitive imports from `Testpilot.*` to either `Umpire.*` or `Temporal.*`, including paths through neutral-looking bridge modules. Existing legal producer-to-Testpilot and existing model dependency checks pass. Errors: forbidden paths produce the existing structured import diagnostic style; unknown/unclassified modules must not let a forbidden transitive path escape inspection.
- **R2:** Every Program and Contract expression-bearing field uses its context-safe type. Checked Lean examples construct all supported reference families and shared recursive operations; compile-failure regressions reject Contract Slots/outcomes and Program Observations/captures/non-Run-ID event fields, both directly and nested. Errors: invalid context construction fails during elaboration, with no coercion or generic public constructor that accepts it.
- **R3:** The sole public canonical codec emits the current Testpilot wire schema and preserves canonical bytes for unchanged valid fixture inputs. Cross-language coverage exercises both Run identity forms, representative nested expressions, integers, bytes, optional fields and monitor rules. Errors: the codec cannot emit the retired `metadata` envelope, does not silently discard values/rules, and propagates any remaining serialization failure.
- **R4:** A synthetic Lean Producer imports only the Testpilot public API, constructs a bounded Case with non-Umpire provenance, emits it through the public codec and passes Go decode plus Prepare under a local test Profile. Errors: malformed wire input and invalid bounds still fail existing Go admission without target I/O; successful decoding alone is insufficient acceptance.
- **R5:** Umpire producer metadata and its encoding remain producer-owned, with exact preservation of definitions, fingerprints, sources, Known Gaps, ordering and optional fields in `producerData`. Tests also round-trip empty and arbitrary non-UTF-8 producer bytes through the generic codec and Go decoder. Errors: generic protocol admission does not demand Umpire metadata or interpret malformed Umpire payload contents; existing producer-side unsupported-lowering diagnostics remain unchanged.
- **R6:** All current repository Case producers, compiler tests, public import examples, renderer commands and cross-language fixtures use the new protocol boundary without changing their supported execution behavior or stable Run/Verdict projections. Re-anchor any fn-68 consumer changes present at implementation time. Errors: deliberately invalid conformance cases retain their prior rejection expectations through explicit negative-fixture construction; unsupported lowering remains an error rather than omitted semantics.
- **R7:** The stale generic encoder and Temporal-owned encoder no longer contain independent serialization implementations, and active architecture/API documentation names the new owner and opaque provenance boundary. Compatibility exports, if retained, call the same current codec and are covered by equivalent-output tests. Errors: no advertised legacy API may continue successfully emitting the obsolete envelope.
- **R8:** The focused Lean protocol/compiler/producer tests, affected Lake builds, import-graph tests, `make lint-model`, Case-runtime fixture regeneration/staleness checks, affected Go suites with `-tags test_dep`, and `make lint-code` pass. Errors: report failed gates with concrete evidence; environmental or pre-existing failures require baseline evidence and the repository's accepted handling rather than a success claim. No new proof placeholders or unapproved trust dependencies are introduced.

## Boundaries
<!-- scope: business -->

- Checked model-to-Case lowering and the Nexus3 success demonstration remain fn-68 scope.
- Environment-value binding and retargeting remain fn-73-explicit-environment-binding-for scope; this spec introduces no new expression reference for environment values.
- Moving the Temporal driver remains fn-72-extract-the-reusable-temporal-testpilot scope.
- Activation-state deepening and public preparation diagnostics remain fn-74-deepen-testpilot-worker-activation scope.
- Target semantic/elaboration separation remains fn-75-separate-lean-target-semantics-from scope.
- Semantic-inventory dependency inversion remains fn-76-make-lean-semantic-inventory-consume scope.
- No new protocol version, runtime semantics, public Monitor interface, Lean decoder, external package release or generated API schema redesign.
- No blanket guarantee that every Lean-constructible Case is Go-admissible; context ownership is the structural guarantee added here.

## Decision Context
<!-- scope: both — conditionally substructured -->

The Go Case already supports producer-neutral provenance and distinct expression contexts. Aligning Lean with that boundary removes the need for a Temporal serializer to repair generic representation mistakes. A standalone Testpilot facade makes the intended direction enforceable and provides a useful non-model Producer test.

Keeping a broad expression union and validating it during serialization was rejected because it leaves invalid references constructible in the normal API. Creating a second JSON utility or retaining multiple codecs was rejected because it perpetuates format drift. Migrating the reusable ordered JSON facility only as needed is smaller than introducing a new serialization framework.

Opaque bytes keep Testpilot independent of Umpire's model vocabulary while preserving the current producer provenance exactly. Runtime semantic validation remains with Prepare; proving descriptor correctness and complete graph validity in the Lean IR would expand this ownership refactor substantially. The cost is source migration for expression helpers and imports, offset by compile-time detection of context mistakes and one wire implementation. No runtime authority or execution behavior changes are justified by this refactor.
