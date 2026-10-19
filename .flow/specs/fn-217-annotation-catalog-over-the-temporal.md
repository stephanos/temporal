# Annotation catalog over the Temporal API: kinds, tags, rules, expected errors and gap analysis (model, temporal)

## Conversation Evidence

> user (2026-10-19): "how do we integrate the Temporal proto apis actually into umpire in a way that allows to add metadata? ... annotate a field is 'random data' or a 'timestamp' or a 'uuid' ... validation rules like 'a and b are never set together' ... verify the inputs and outputs; but also draw automatic conclusions from them for the model. eg a #cached could mean it has a ttl or #eventually_consistent ... some ... part of umpire if they generalize (like uuid) and others are specific to temporal"
> user (2026-10-19): "once we annotate anything we'll want to ensure the 'action/input/output/message model' is exhaustive. so if a new field is added, it must be added to that as well"
> user (2026-10-19): "can a field have multiple tags/annotations?" — yes: one kind, any tags, rules on the message.
> user (2026-10-19): "for the validation rules; anything we can borrow that makes this easy? at_least_once is very custom." — JSON Schema's presence algebra, protovalidate's constraint names, AIP field_behavior roles.
> user (2026-10-19): "we need to connect/map the tags/annotations to the expected error from the server. that will allow us to do a gap analysis to find mismatches"

## Goal & Context

fn-216 makes the Model speak the Temporal API (`temporal.api.*` messages in three envelopes). Those messages carry no meaning beyond their types: nothing says a field is a UUID, a timestamp, generated data or a secret; nothing says two fields are never set together; nothing says a read may trail the write. This spec adds that meaning as a typed sidecar catalog, since the API protos are upstream's (`proto/api.binpb`) and cannot carry inline options. The catalog serves four consumers: filling the fields the Model does not decide (`fill`), validating every request sent and response observed (`validate`), feeding the Model's judgment with consequences (a `Visibility` lag, a `Cached` ttl), and, the owner's key addition, mapping each rule to the error the server is expected to return, so a probe can send each violation and report mismatches between the catalog and the server: a gap analysis in both directions.

## Architecture & Data Models

- **Layering (generic in `model/umpire`, Temporal in `model/temporal`; fn-216 R5's gate check applies).** `umpire-annotations` (new crate) defines the vocabulary and the engine with no system names: `Kind` (exactly one per field: `Uuid`, `Identifier`, `Timestamp`, `Duration`, `Counter`, `Random`, `Secret`, `Enum`, `Text`, `Bytes`, with per-kind constraints named after protovalidate's, e.g. `string.uuid`, `duration.gt`, `enum.defined_only`, `repeated.unique`), `Tag` (any number per field, each with a `Consequence` the engine or the Model can act on: `Monotone`, `Immutable`, `Cached { ttl }`, `EventuallyConsistent { polls }`, `IdempotencyKey`, roles `Decided`/`Observed`/`Ignored(reason)` after AIP `field_behavior`), `Rule` on a message (JSON Schema's presence algebra only: `required`, `any_of`, `one_of`, `requires`, `excludes`; value rules belong to kinds), `Expectation` (what the system does on a violation: `Rejects { code, message_pattern, details_type }` or `Accepts` or `Unobservable`), `Catalog` keyed by fully qualified field paths, `validate(msg) -> Vec<Violation>`, `fill(msg, seed)`, and the gap report types. Temporal-specific tags (`Visibility { lag_polls }`, task-queue scoping, ...) are types in `model/temporal` implementing `Tag`; a tag moves up only when its consequence is system-free.
- **Admission rules.** A kind is checked against the prost field type at compile time. A tag declares the kinds it applies to; contradictory pairs are compile errors (`Decided + Ignored`, `Secret + Observed`, `Random + Decided`). A rule naming an `Ignored` field is refused. A tag without a consequence or a validation is not admitted; `Ignored` requires a reason.
- **`annotate!` (in `model/temporal`, macro over the prost-generated API types).** One catalog file per feature, `features/<feature>/api.rs`. Exhaustiveness at four levels, build-failing: fields (the macro emits a struct pattern without `..` per annotated message, so an upstream field addition fails with rustc's "pattern does not mention field"), enum values (`observe` and kind mappings are `match`es without a wildcard; an unknown integer at runtime is a `Refusal`), RPC methods (`build.rs` reads `proto/api.binpb` and requires every method of each named service to be classified `Action(..)`, `Observed` or `OutOfScope(reason)`), actions (every Model action is bound to a method, revealed by an observation, or environmental with a reason). Bumping `api.binpb` therefore fails the build naming each unclassified field, value and method; the catalog diff is the review of the API change.
- **Expected errors and the gap probe.** Every rule and every kind constraint on a request field carries an `Expectation`, normally `Rejects { code: InvalidArgument, message_pattern, details_type }`, restated from the server's own validator (named in the entry, `covers: "validateStartActivityExecutionRequest"`). A Go probe test in Testpilot (next to fn-216.3's) builds, for each expectation, one minimal violating request from a valid filled one, sends it against the in-process server and classifies the outcome: `Confirmed` (code and pattern match), `Mismatch { expected, actual }` (rejected differently), `Accepted` (the server took an input the catalog says it rejects), `Unverifiable` (output-only or unobservable). The report is a golden (`gaps.txt`) per feature; `Accepted` and `Mismatch` fail the probe unless listed in a dated `accepted_gaps` block with a reason.
- **Reverse coverage.** Each feature catalog lists the server validator functions it covers; a Go test parses those packages and fails on a listed function that no longer exists, and lists (as a report, not a failure) validators of the touched RPCs that no catalog entry names, so missing rules are visible.
- **Consumers in fn-216's slice**: `fill` supplies `request_id` (Uuid) and any `Random` input; `validate` runs on every request sent and every response observed in the slice test; `judge` consumes `Visibility`/`Immutable`/`Monotone` consequences when the mini activity catalog carries them.

## API Contracts

- `umpire-annotations`: `Catalog`, `Kind`, `Tag` (trait), `Consequence`, `Rule`, `Expectation`, `validate(&M, &Catalog) -> Vec<Violation>`, `fill(&mut M, &Catalog, &mut Rng)`, `GapReport`.
- `model/temporal`: `annotate! { Message { field: Kind (+ Tag)*, .. , rules: [..], covers: [..] } .. }`; `build.rs` service classification; per-feature `api.rs`.
- Probe: `go test ./common/testing/testpilot/temporal -run Gaps`, golden `features/<feature>/expected/gaps.txt`.

## Edge Cases & Constraints

- Kinds that cannot be generated (`Secret` from outside) are `Decided` or supplied by the test; `fill` refuses a field it cannot fill, naming it.
- A response violation is a server finding (reported, never silently fixed); a request violation before sending is a Model or driver bug and fails the test.
- Redaction: `Secret` fields are redacted from Runs and reports by the engine.
- fn-210 R30 applies to `make model-check`; the probe runs in the Testpilot gate.

## Acceptance Criteria

- **R1:** Exhaustiveness at the four levels is build-failing: a field, enum value or method added to `proto/api.binpb` for an annotated message/service, or an unbound Model action, fails the build naming it. Errors: as stated; pinned by trybuild fixtures and a build-script negative control with a mutated descriptor set.
- **R2:** The layering holds: one kind per field, any tags, rules per message; the admission rules are compile errors with the messages above; kinds are checked against prost types; names follow protovalidate (constraints), JSON Schema (presence operators) and AIP `field_behavior` (roles). Errors: pinned by trybuild fixtures.
- **R3:** Every request-side rule and kind constraint carries an `Expectation` naming the server validator it restates; the probe sends each violation and produces the `gaps.txt` golden with `Confirmed`/`Mismatch`/`Accepted`/`Unverifiable`; `Mismatch` and `Accepted` fail unless listed with a dated reason. Errors: a rule without an expectation is refused at compile time.
- **R4:** Reverse coverage: listed validators are verified to exist; validators of touched RPCs without a catalog entry are reported. No error surface beyond a vanished validator.
- **R5:** The mini activity catalog annotates every message fn-216 touches (`StartActivityExecutionRequest`, `DescribeActivityExecutionResponse`, `PollActivityTaskQueueResponse`, `RespondActivityTask{Completed,Failed}Request`) and the slice test uses `fill` and `validate`; `judge` honours `Visibility` if the catalog declares it. No error surface.
- **R6:** `umpire-annotations` contains no Temporal name (fn-216 R5's gate check covers it); Temporal tags live in `model/temporal`. Errors: the gate check.

## Boundaries

- No inline proto options, no edits to upstream API protos, no CEL, no external validation engine (protovalidate's and garde's vocabularies are borrowed, not their runtimes).
- No annotation of Testpilot's own proto.
- No automatic derivation of Model transitions from annotations; consequences inform `judge`, `fill` and `validate` only.
- Does not start before fn-216.

## Decision Context

2026-10-19, owner: metadata on the Temporal API as a typed sidecar (the protos are upstream's); exhaustive by construction so an API change cannot go unclassified; one kind plus any tags per field; rules borrowed from JSON Schema's presence algebra rather than custom operators (`at_least_one` is `any_of`); every rule mapped to the server's expected error so the probe yields a gap analysis in both directions (catalog vs server, server validators vs catalog). Generic vocabulary in Umpire, Temporal consequences on the Temporal side, decided by whether the consequence is system-free.

2026-10-19, owner, implementation of constraints: the few per-field constraints the catalog needs are implemented natively in `umpire-annotations` (one function per kind constraint over the destructured prost fields), borrowing protovalidate's names and semantics and porting its conformance cases as fixtures where they apply; no CEL and no validation engine is a dependency. Survey that day: the Rust CEL interpreter is healthy (`cel` 0.15.0 on 2026-10-06, ~4.4M downloads, regular releases since the `cel-interpreter` rename in 2025; admissible under fn-210 R18 if ever needed) but unneeded, since the rules are JSON Schema's presence algebra in typed Rust; Buf's own `protovalidate-rust` is a 0.0.1 placeholder (repository not public); community implementations (`prost-protovalidate` 0.6.0, July 2026, prost-reflect based, full conformance claimed, CEL optional; `protovalidate-buffa`, October 2026, generated static impls) read rules only from descriptor options, which upstream's `api.binpb` does not carry, so they would need a descriptor rewrite to serve a sidecar. The owner likes AIP `field_behavior`, JSON Schema's algebra and protovalidate's per-field constraints as the vocabularies.
