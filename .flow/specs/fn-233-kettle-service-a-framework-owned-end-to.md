# Kettle service: a framework-owned end-to-end reference system (model, umpire)

## Conversation Evidence

> user (2026-10-21): "let's for a moment think about having a full end to end testable test case like temporal activity but scoped to the framework itself to exercise it fully including realization, annotations and all that. suggest how we would do that best"
> user (2026-10-21): "yes file flow spec"

## Goal & Context

Every end-to-end claim the framework makes today is proven once, against Temporal: the mini activity slice (fn-216) is the only place where a proto API, an annotation catalog (fn-217), a `Realize` binding, a guest, the Go host, a live system, Judge, Answer, Annotate, the gap report and its black-box reverse direction run as one chain. Under `umpire/` the same pieces exist only in isolation: the Kettle fixtures prove the Model and checker, the suite fixtures prove authored runs, `umpire-annotations` tests hand-built catalogs, and the stock and control guests prove the wasm ABI. Nothing generic runs the whole chain, so a generic claim is really a Temporal claim, and the no-Temporal rule (fn-217 R6) is checked by name grep rather than by use.

This spec gives the canonical Kettle Model the system it is missing: a small Kettle service owned by the framework, with its own proto API, a Go reference server with named validators and scoped limits, a `derive(Catalog)` mirror catalog, a `Realize<kettle::World>` binding, a guest, and a live run through the Go host. The service is shaped like the activity API on purpose (client RPCs, a heater that polls and responds, a Describe read, out-of-scope RPCs) so all three envelopes and all four exhaustiveness levels are exercised. Seeded server defects (mutants) make the run a proof: the framework must name each one.

## Architecture & Data Models

- **Reference system under `model/umpire/reference/kettle/`.** The Model is the existing `umpire/tests/fixtures/sample_kettle/model.rs`, included by path; no second Kettle Model. New: `proto/kettle/v1/kettle.proto` (package `umpire.reference.kettle.v1`, `KettleService` with Fill, Heat, Pour, Describe in scope; a heater role with PollHeating and RespondHeating; ListKettles and ResetKettle out of scope), `server/` (Go: gRPC service on a loopback listener, `validators.go` with one named function per rule and distinctive error messages, `limits.go` with a static limit and a per-kettle scoped limit mirroring the `dc` shape, `mutants.go`), `api.rs` (the `#[derive(Catalog)]` mirror structs of fn-217.6), `config.rs` (Kettle limits over the generic scoping), `realization.rs` (`Realize<kettle::World>`), `guest/` (an `export_guest!` crate), `tests/` and `expected/`.
- **Awkward on purpose.** The schema carries one of each annotation feature: enum, oneof, map, nested message, repeated, duration, bytes, a deprecated field, an idempotency key the client may omit (filled by the framework), a static limit, a scoped limit, a cross-field `any_of` rule, and at least one field the catalog leaves `Ignored` with a reason so `gaps.txt` is non-trivial.
- **Three drivers.** (1) Native goldens in `cargo test`: coverage snapshot, `gaps.txt`, exploration report, trybuild refusals. (2) A Go host live run (`umpire/hosts/go/kettle_test.go`): start the reference server in-process, load the guest, run a scripted client and heater; every wire message goes through Annotate, every Describe through Judge, every poll through Answer; the transcript is a golden. (3) Mutants: the server takes a `KETTLE_MUTANT` setting that switches on exactly one seeded defect; the test asserts the framework reports it by name (an `Accepted` or `Mismatch` from an actual server probe, or an `Unexplained` from Judge, different from the same healthy-server control).
- **What moves out of `temporal/`.** The three envelopes (Observation, Callback, Reply) move from `temporal/proto` to `umpire/proto` under a generic package; the descriptor coverage build check moves from `temporal/api/build.rs` to `umpire/`, parameterised by descriptor path. The catalog names no server internals (owner, 2026-10-21), so there is no validator generator or Go source walk to move (`reverse_coverage.go` is deleted by fn-217.6). The Temporal slice consumes the shared pieces and stays green with the explicit envelope identity migration below.

## API Contracts

- `umpire.reference.kettle.v1.KettleService`: Fill, Heat, Pour, Describe, PollHeating, RespondHeating, ListKettles, ResetKettle.
- Go server: `kettle.Serve(listener, Options{Mutant: string}) (stop func())`; mutant names are an enumerated list in `mutants.go` and in the test.
- Rust: `umpire_reference_kettle::{api, config, realization}`; guest crate `umpire-kettle-guest`.
- Goldens: `reference/kettle/expected/{coverage.snap,gaps.txt,report.snap.txt,transcript.snap.txt,mutants.snap.txt}`.

## Edge Cases & Constraints

- The mutant list must include at least: accept an over-long kettle ID (limit), report Boiled before RespondHeating (Judge), silently stop rejecting a field (`Accepted` in `gaps.txt` against the catalog's rejecting prediction), return a wrong status code (Mismatch). Each mutant is a one-line switch, not a fork of the server.
- The server is independent of the Model: written as a plain service from the proto, not from `model.rs`. A shared helper between server and Model is refused.
- Not a second Testpilot: no Case, Program or Contract vocabulary.
- fn-210 R30 applies: the native driver runs in `make model-check`; the Go live run and mutants run in the Go host test target.

## Acceptance Criteria

- **R1:** The envelopes and the descriptor coverage check live under `umpire/` with no Temporal name, and the Temporal slice consumes them unchanged with identical Model and semantic goldens (wire identity migration is specified below). Errors: the no-Temporal gate check; a Temporal golden diff fails the task.
- **R2:** The Kettle service proto and Go reference server exist with named validators, a static and a scoped limit, and a Judge/Answer live run through the Go host with validated selected-start initialization and the RPC projection contract below, whose transcript is a golden. Errors: a transcript diff, invalid selected start, or unexplained healthy observation fails; the server building against `model.rs` is refused in review.
- **R3:** `api.rs` is a `derive(Catalog)` catalog covering every in-scope message with all four exhaustiveness levels build-failing, `config.rs` carries the limits, and Annotate runs on every wire message of the live run; `coverage.snap` and `gaps.txt` are goldens with at least one `Ignored` reason and one `Unverifiable`. Errors: as fn-217 R1 and R3.
- **R4:** Each mutant in the enumerated list has a healthy-versus-mutant witness, an actual server response and a named framework finding in `mutants.snap.txt`; a request-side Annotate violation alone is not detection; a mutant the framework does not detect fails the test, and a new mutant without a test entry fails. Errors: an undetected mutant names the mutant and the driver that should have caught it.
- **R5:** The already-delivered fn-217.6 derive is exercised by a second, framework-owned catalog. Every generic feature missing for Kettle is documented and implemented with both Kettle and Temporal regression controls before this spec closes. No error surface.

## Boundaries

- No change to the Kettle Model or its goldens.
- No new Temporal RPCs, no change to the activity Model or its catalog beyond consuming the moved tooling.
- No remote or paid infrastructure; the Go server is in-process on loopback.
- Does not start before fn-217.7 (rename). Task .3 does not start before fn-217.6's derive exists.

## Implementation contracts resolved by the backlog review

**Selected live start (R2).** Add an additive `Runtime::with_initial(machine, realization, initial) -> Result<Self, Refusal>`. It validates exact membership in `Machine::starts`; an absent start returns State refusal. Store the validated selected initial state and use it for transactional Reset. `Runtime::new` keeps the existing unique-start behavior, including refusal for zero/multiple starts. Explore and Rows still explore the original Machine's complete starts. The Kettle guest selects `{ temperature: 20, heating_on: false, beeping: false }`, and the independent server starts with the same documented initial temperature. Negative controls cover invalid initial states, legacy multi-start refusal, Reset after freeze, and failed response publication leaving state unchanged. All 81 canonical starts and existing Model/golden bytes remain unchanged. This adds no wire method or field.

**RPC projection (R2/R4).** Fill creates the one reference kettle at the documented initial temperature; repeated Fill cannot reset an active run. Heat carries an enabled flag: true commits User::PressButton, false commits User::DepressButton, only after the successful RPC response. Pour is a read-only dispensing acknowledgement in this deliberately bounded service: it changes no modeled state and makes no claim about liquid volume. Fill and Pour match only the current projected state. PollHeating returns a callback describing current heater work; Answer has an empty delivery path and chooses Heater::Heat below 100 while heating, or Heater::Failover at 100 while heating. Answer checks but does not commit that action. RespondHeating returns an applied-action receipt and the full state projection; Judge replays exactly that receipt's single action. Describe projects temperature, heating_on and beeping and admits only the zero-step current match; it cannot infer unobserved Heat/Failover paths. Invalid/disabled RPCs reject without mutation and are tested through annotations, not invented Model rejection edges. ListKettles and ResetKettle are explicitly out of scope.

The healthy transcript pins Fill at 20, Heat(true), Poll/Answer/RespondHeating increments through 100, then Poll/Answer/RespondHeating Failover and Describe with beeping=true/heating_on=false. Also exercise Heat(false) from an enabled heating state and a projection-neutral Pour. The premature-boil witness pauses after Answer but before RespondHeating: a Describe claiming beeping=true is Unexplained, whereas the healthy Describe remains Explained at the unchanged state. Server polling never advances the Model by itself.

**Generic tooling (R1/R3).** Extraction includes the catalog derive and syntax/descriptor checks, inspection/fill runtime traits, descriptor-driven method enum/classification generation, and wire-schema mutation generation currently under temporal/annotate and temporal/api. Shared implementations live under umpire; parameters supply service full names, generated Rust module paths, context type and request roots. Shared code never emits a Temporal crate path or assumes foundation::dc::Limits. Temporal keeps its catalog declarations, dynamic config policy, descriptor acquisition and API-specific roots; Kettle supplies its own. Both consumers use the same implementation. Move the generic portions of Annotate/probe request-response transport with the shared annotation dispatch, retaining Temporal-specific controls outside umpire. Native negative fixtures cover omitted fields/enum variants/methods/actions and a build of Kettle without any temporal crate dependency. The no-Temporal source gate remains mandatory.

**Envelope migration (R1).** Generic Observation, Error, Callback and Reply use package `umpire.realization.v1`; enclosing Any URLs change accordingly. Migrate Rust imports, generated macro paths, Go generated code and consumers, wire identity tests, schema generation/freshness scripts and documentation together. Preserve field numbers, protobuf payload encodings, core method numbers, and all Model/semantic goldens; update only the explicitly changed envelope type-URL wire expectations. Do not claim old Any wrapper bytes remain identical. Preserve fn-227.12's optional Observation.request tag 4 and its absent-request encoding; serialize this migration after that task. Update fn-227's later live consumer paths to the shared envelopes when the move lands. No dual mutable envelope implementation or Temporal-named alias is placed under umpire.

**Mutant oracle (R4).** Every fixed witness runs against a fresh healthy server/guest and a fresh single-mutant server/guest. ID-limit and removed-validator witnesses send the same invalid bytes to the actual server even though Annotate predicts a rejection; the healthy server rejects with the declared code, the mutant accepts, and the probe classifies Accepted. The wrong-status witness pins expected versus actual code as Mismatch. Annotate runs and records its prediction but cannot suppress these deliberate probe sends or itself count as detection. Unexpected server rejection is the distinct Unannotated category; removing rejection is not Unannotated. A no-op mutant must fail the detection test. Unknown mutant names fail setup. Use per-server Options rather than global environment mutation inside parallel tests; the optional KETTLE_MUTANT entrypoint parses into that option. Tests pin mutant-name set equality and structured finding content before comparing the golden. Transport/startup failures are test failures, never mutant findings.

**Build and scheduling.** Task .1 extracts shared tooling and migrates envelopes, after fn-227.12. Task .2 adds selected-start runtime support and the healthy Judge/Answer service. Task .3 supplies the second catalog consumer and probes. Task .4 proves differential mutant detection. The Go reference server and generated gRPC API live inside the existing generic Go host module (module root model/umpire/hosts/go), under its reference/kettle packages, so host tests need no import cycle or server-module dependency. Proto/Rust guest/fixtures remain in the reference tree. Reuse the repository's pinned Go generators and protoc, add reproducible gRPC generation and freshness checks if absent, and declare the direct grpc dependency in that Go module using the version already pinned by the repository. The existing builder supports arbitrary guest packages: add kettle without replacing stock, control or Temporal targets. Native fixture checks and the Go host suite run by make model-check execute the healthy/mutant controls; document and measure the added cost under R30 rather than claiming they are outside that gate. An explicitly selected Kettle test missing its guest must fail, not skip. Loopback listeners use dynamic ports and bounded contexts; cleanup stops server, connections and guest on success and failure.

## Decision Context

2026-10-21, owner: wants an end-to-end test case like the Temporal activity but scoped to the framework. Recommendation adopted: extend the canonical Kettle into a Kettle service rather than invent a second system; a Go reference server because the live run goes through the Go host and an in-process Go server is the simplest loopback (the earlier reason, exercising Go-reading `covers` tooling, fell away when the owner removed `covers` the same day; a Rust server driven natively remains an option); mutants so the run is a proof rather than a smoke test; the envelope and tooling moves as the first task because they measure what is generic today. fn-217.6 has already shipped on Temporal. R5 now exercises Kettle as its second consumer; no historical first-use ordering is required.


Maintainability (plan review): duplication - none identified; structure - extract shared catalog/code-generation implementations to avoid a framework-to-Temporal dependency.

## Requirement coverage

| Req | Tasks |
| --- | --- |
| R1 | fn-233.1 |
| R2 | fn-233.2 |
| R3 | fn-233.1, fn-233.3 |
| R4 | fn-233.4 |
| R5 | fn-233.3 |
