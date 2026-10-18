# Testpilot schema in the framework, Temporal-agnostic

## Conversation Evidence

> user (2026-10-18): "I think we'll need to make TestPilot Temporal-agnostic; and part of the framework. (I believe PLang does sth similar). It would be a conformance descriptor that the tools/testpilot would have to interpret for temporal (and the cases would contain temporal-specifics like RPCs from model/temporal anyway)."
> user (2026-10-18): "is it possible to move the testpilot IR into model/framework/proto without too much effort?"

## Goal & Context

The Testpilot Case schema (eleven proto files: Case, Program, Contract, correlated clauses, expressions, evidence policies, events, sources, values, instructions, Run) is already generic except for seven instruction fields typed with Temporal API messages. It lives under the server's internal protos and is generated for Go by the server's protoc pipeline and for Scala by the held gate. This spec makes the schema a framework asset: the files move under the framework's proto directory, the seven Temporal-typed fields go behind a driver seam so the framework proto depends only on `cel/expr` and well-known types, the Go runtime splits into a generic interpreter and a Temporal driver, and a new generic conformance-descriptor file joins the set for fn-204. The P pattern is the model: the harness interprets a shipped spec artifact; only the driver knows the target.

## Architecture & Data Models

- Framework proto directory holds the Testpilot files and, after fn-201, the IR file. One protoc invocation generates Go into the existing output package and Scala into the API jar.
- Driver seam for the seven fields (activity attempt failure, heartbeat payloads, workflow command, Nexus start response and handler error, a payload, a failure): the owner chooses between a `google.protobuf.Any` driver payload with a registry, and a generic `oneof` extended by a Temporal wrapper message. The Temporal messages move to a Temporal proto file under `model/temporal`.
- Go runtime split: `common/testing/testpilot` generic packages (internal, evaluation, contract, recordedrun, campaign, replay) read only the framework schema; the `temporal` package and the Driver read the Temporal extension.
- Package rename to `umpire.testpilot.v1` with a matching Go package path; all Go imports updated mechanically.
- Conformance descriptor proto: rows keyed by state and class with ordered results (target, outcome, facts, choice), interned states, per-result claim bitsets, monitor automata (states, next table, read set, violated set), progress and end marks, fairness classes, and evidence roles (correlation keys, ordinals, closing events) declared abstractly.

## API Contracts

- The protocol comment check runs over the new paths and still requires leading comments on every message.
- Generic instruction messages carry the driver payload; the Temporal extension file defines the seven concrete messages; the registry maps extension type URLs to decoders in Go and Scala.
- `ConformanceDescriptor` is a new message in the framework schema with the fields above; fn-204 defines its interpretation.
- The Scala API jar is generated from the framework path; the IR jar is unaffected until fn-201.

## Edge Cases & Constraints

- Existing Case fixtures must decode unchanged after the seam when the seam is `oneof`-based; with `Any` they are regenerated once by fn-203 and that regeneration is noted in both specs.
- The server's protoc pipeline globs the internal proto root; the framework path is added to the proto path and directory list rather than copied.
- Go work is held under the current framework-only override; this spec is independent of the DSL and runs when the owner releases Go work.

## Acceptance Criteria

- **R1:** The eleven Testpilot protos live under the framework proto directory, Go and Scala bindings generate from there, and the protocol comment check passes over the new paths. Errors: a binding generated from the old path fails the build naming the moved file.
- **R2:** The framework proto depends only on `cel/expr` and well-known types; the seven Temporal-typed fields are behind the chosen driver seam with the Temporal messages in a Temporal proto file. Errors: an unregistered extension type URL fails decoding with the URL.
- **R3:** The Go runtime splits into generic packages that import no Temporal API type and a Temporal driver package that does; a boundary test enforces the split. No error surface beyond the test.
- **R4:** The package is renamed to `umpire.testpilot.v1` with the Go package path updated everywhere; `go build ./...` and the live tests pass. No error surface.
- **R5:** `ConformanceDescriptor` exists in the framework schema with rows, interned states, claim bitsets, monitor automata, progress and end marks, fairness classes and abstract evidence roles, each field commented. No error surface beyond R1's comment check.

## Boundaries

- No descriptor emission or interpretation (fn-203, fn-204).
- No change to the Driver's Temporal behaviour.
- No removal of `tools/umpire` packages (fn-208).

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

Leaving the schema under the server's internal protos keeps Testpilot a Temporal component by location and forces every Scala consumer through the held Go generation route. Moving it is an afternoon of path changes; the seam for seven fields is the only design work, and it is what makes a Temporal-agnostic interpreter possible. `Any` gives the cleanest boundary at the cost of a registry on both sides; the `oneof` wrapper is smaller but keeps a Temporal name in the generic file. The owner picks; the spec implements one. The descriptor message is defined here, before fn-204 interprets it, so the schema move happens once. No dependency on the DSL or Temporal-Model phases; held only by the Go-work override. Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).

2026-10-18, owner decision (R2, R4): the driver seam is google.protobuf.Any with a type-URL registry on both sides; the Temporal messages live in a Temporal proto under model/temporal; the package is renamed to umpire.testpilot.v1 in the same change so one fixture regeneration, shared with fn-203, covers both.

2026-10-18, deterministic simulation (owner): the generic/driver split (R3) must admit the deterministic Go runtime as a driver: the adapter interface is act-and-observe over abstract evidence roles (correlation keys, ordinals, closing events), with no assumption that observations arrive as RPC responses or that time is wall-clock. Bridge frames stay pure data with no callbacks, so a fork()'d Go process can resume against a snapshot.

2026-10-18, the simulation is WebAssembly on wasmtime (owner): the generic adapter interface of R3 must be expressible both as the NDJSON bridge (live Testpilot runtime) and as an in-host interface (wasmtime host functions or a component-model world) for the simulation. Define the interface once as data types in the framework proto and derive both transports from it; no transport-specific types in the generic packages.

2026-10-18, target boundary (owner): the generic Go packages of R3 are the host loop and the driver interface, not a runtime with a Program interpreter. On the fn-209 go branch, Program sequencing, Contract evaluation, evidence policies and assessment leave Go; the driver interface has three implementations (real Temporal, simulated Temporal in wasm, recorded Run), and the read-until predicate on an instruction is the only execution-control logic a driver evaluates. The schema keeps Program as data the Model side consumes; nothing in the generic Go packages interprets it.

2026-10-18, snapshot and restore (owner): the generic adapter interface carries `snapshot() -> id` and `restore(id)` on the oracle side, over the Model's immutable state, claim and monitor values; a host that can snapshot its own store may ignore them, a host driving a sidecar must use them. This is what lets a native-image oracle serve the deterministic simulation and keeps the interface identical across the wasm and native hostings.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

