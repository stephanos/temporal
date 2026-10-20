# Declarative Realizations: the #[realization] attribute surface and the coverage golden (model, temporal)

## Conversation Evidence

> user (2026-10-20): "can/should we use more elegant tricks from Rust to make temporal.rs even nicer? it should also be obcious to the reader which parts of an RPC are not covered/mapped by the model ie not supported. and I was thinking about whether want to use #[] metdata more heavily to map mdoel <> system with less code. but I onder if we want to break the boundary between them to do that."
> assistant: ordinary Rust only, with `observe` made declarative; coverage visible by construction through a per-RPC golden of mapped, ignored and refused fields measured against fn-217's catalog, with exhaustive enum matches and no wildcard arms; attributes yes, on the Realization side only, since the boundary is the direction of reference: the Realization names both, the Model never names Temporal. No `#[field("info.status")]` on Model state. Order: fn-229 helpers first, this after fn-217.
> user: "yes" (file it with that dependency)

## Goal & Context

After fn-229 the activity Realization is three trait methods over shared helpers. What remains hand-written is the dispatch (which RPC goes to which handler), the unpacking, the refusals for anything unlisted, and the knowledge of which fields of each message the Model actually reads. None of that should be prose a reader infers from wildcard arms. This spec makes a Realization a set of annotated handler functions, with everything else generated, and makes coverage a golden: per RPC, the fields the Model maps, the fields it ignores with a reason, the fields that refuse, and the RPCs it does not handle at all. The Model file is untouched and still names nothing from Temporal.

## Architecture & Data Models

- **Attribute surface** (a proc macro in the Temporal realization crate, not in `umpire-dsl`, since it names Temporal services): `#[realization(<model module>)]` on an `impl Realization` block containing handlers:
  - `#[observe(<Service>::<Method>, reads(<field paths>), ignores(<field path> = "<reason>", ..))]` on `fn name(<response type> or Result<<response type>, Code>) -> Result<Projection<..> | Step<Actor> | Phase-like, Refusal>`; the return type selects the generated projection: a state-valued return becomes `Current` when equal to the Model's current projection and a one-step candidate otherwise; a `Step` carries its action and expected outcome.
  - `#[answer(<task type>, reads(..), ignores(..))]` on `fn name(task) -> Result<Reply<Actor>, Refusal>` where `Reply { delivery, action, request }`.
  - `#[propose]` on `fn name(state, seed) -> Proposal` once fn-228.2 exists; absent until then.
- **Generated code**: the `Realize` impl with dispatch on the RPC method path, envelope decoding and unpacking through the fn-229 helpers, `Unsupported` refusal for any unlisted method, and the quiescent and pending refusals.
- **Coverage golden** `expected/realization.snap.txt` per Model: a table per handled RPC with every field of its message from the fn-217 catalog classified as `mapped` (in `reads`), `ignored` (in `ignores`, with the reason) or `unsupported` (neither: present in the request means refusal), followed by the list of catalog RPCs for the Model's actions with no handler. The macro checks `reads` and `ignores` against the catalog at build time: an unknown path is a compile error; a catalog field in neither list is `unsupported` in the golden, never silently accepted.
- **Exhaustive enum matches**: handlers that match on generated enums list every variant; the macro refuses a wildcard arm in such an enum match with a located error, so an upstream enum addition is a compile error in the Realization. Ordinary byte/payload-policy fallback arms remain valid. Enum conversions in helpers need their own exhaustive matches and controls.
- **Boundary**: the Model module is not annotated and stays as fn-216 wrote it; `umpire-dsl` and `umpire-wasm` gain nothing Temporal-specific; the existing boundary scan covers the new crate.

## API Contracts

- Attributes `#[realization]`, `#[observe]`, `#[answer]`, `#[propose]` as above, in the Temporal realization crate's macro sub-crate.
- `Step<A> { action: A, expected: Option<Outcome> }` and `Reply<A> { delivery: Vec<A>, action: A, request: Any }` as the handler return vocabulary, in the realization crate.
- Golden `realization.snap.txt` with the exact three classifications and the unhandled-RPC list; its format is the contract and is pinned by the activity Model's golden.
- The generated `Realize` impl has the same behaviour as the hand-written one: all activity goldens, wire goldens, the slice and the fn-218 cover tests unchanged.
- A state-valued return needs an explicit Realization-owned projection/action mapping. The macro must not infer an action from a phase name, add a `Poll` transition when Describe reports Started, or alter the existing empty-candidate behavior for unexplained phases. Authors may use the already admitted `Projection` return where the mapping is not a single step. State-valued conversion controls must exercise Current and the declared one-step mapping.
- Coverage distinguishes observed responses, incoming callbacks and emitted requests. The Complete/Fail requests are partial SDK replies; outgoing fields are classified as produced or ignored within their request context, never mislabeled as handler reads. The report retains the three mapped/ignored/unsupported classifications and names each context.
- Unsupported-field refusal checks decoded protobuf presence, including optional messages and oneofs. It cannot use an opaque ignored field's metadata presence, which is always false. Proto3 scalar defaults carry no wire presence after decoding; that limit is explicit. Opaque ignored parents stop nested path traversal and retain their reason.
- Compile-time path checks reuse fn-217's admitted declarations and protobuf graph, including RPC input/output edges, raw field names, containers and oneof alternatives. Runtime catalog factories alone are not compile-time metadata. Catalog inputs must be tracked, and parser/API dependencies must preserve the existing acyclic build boundary.

## Edge Cases & Constraints

- A `reads` path the handler body never touches is not detected; the golden states what the author declares, and the review reads the handler. Recorded as the known limit.
- Two handlers on one RPC is a compile error; a handler returning a type the macro cannot classify is a located error naming the accepted return types.
- `ignores` requires a reason string; an empty reason is refused.
- The macro expansion is pinned by an insta snapshot of the activity Realization's expansion, formatted by prettyplease, as the DSL macros are.
- R30 budgets hold; one new proc-macro crate under `model/temporal`.

## Acceptance Criteria

- **R1:** The activity Realization is rewritten as `#[realization(activity)]` handlers with `#[observe]` and `#[answer]`, and the generated impl reproduces every activity golden, wire golden, the dedicated slice and the fn-218 cover tests byte-identically. Errors: duplicate handler, unclassifiable return type, wildcard enum arm and unknown field path are compile errors pinned by trybuild.
- **R2:** `realization.snap.txt` for the activity Model lists every catalog field of the five fn-217 messages as mapped, ignored with reason, or unsupported, plus the unhandled RPCs for the Model's actions; the golden is stable and reviewed as the coverage statement. Errors: a catalog change that adds a field fails the golden until classified.
- **R3:** The Model module is unchanged and the boundary scan passes; `umpire-dsl` and `umpire-wasm` carry no Temporal name. Errors: the gate check.
- **R4:** The expansion snapshot and a DESIGN.md section "Declarative Realizations" record the surface, the return-type rules and the golden format, with the before and after line counts of the activity Realization. No error surface beyond the gate.

## Boundaries

- No annotation of Model state or actions with system field paths; projection authoring stays on the Realization side.
- No generic (non-Temporal) attribute macro in `umpire-dsl`; a second system would add its own realization macro over the same `Realize` trait.
- No `#[propose]` implementation before fn-228.2; the attribute is reserved.
- No change to the oracle protocol, envelopes or ABI.

## Decision Context

2026-10-20, owner: asked whether heavier attribute metadata to map Model and system would break the boundary, and asked for filing. Decision: attributes on the Realization only, which keeps the direction of reference one way; coverage becomes a golden measured against fn-217's catalog so unsupported parts of an RPC are visible by construction rather than inferred from wildcard arms. Rejected: `#[field(..)]` on Model state (puts the system into the oracle TLC checks); trait-object or generic dispatch tricks (shorter, less readable); a hand-maintained coverage list (would drift from the catalog). Dependencies: fn-229 (the helpers are the macro's expansion target) and fn-217 (the catalog is the field universe). Placement after both; before fn-228.2 if possible so Propose lands on the attribute form.

## Resolved via Codebase

- At `54503f3799`, `model/temporal/features/activity/temporal.rs` explains Describe by comparing `state.phase`, proposing only RespondCompleted or RespondFailed when it differs, and returning empty candidates for other phases. Its payload-policy byte match needs a fallback. The existing enum conversion lives in `features/activity/phase.rs`; handler-body scanning alone cannot guarantee exhaustiveness inside helpers. Enum refusal controls must target generated enum matches while preserving payload-policy fallback behavior.
- `model/temporal/realization/src/support.rs` supplies Observed, callback, method, reply and ApplicationFailure. `model/umpire/wasm/src/runtime.rs` owns state freezing, outcomes, transition choice and transactional publication. Generated Realize dispatch must preserve their refusal ordering and responsibilities.
- `model/temporal/annotate/syntax/src/lib.rs`, `tree.rs` and `descriptor.rs` expose the admitted source parser and coverage checks without generated API imports. API generation depends on that syntax crate. Importing the generated API into it would create a build cycle.
- `model/umpire/annotations/src/catalog.rs` makes opaque IgnoredField presence false. This is deliberate for annotation validation and cannot prove an unsupported protobuf field is absent.
- `model/temporal/testing/fixtures.rs` registers the existing Temporal consumers. The shared fixture runner refuses unowned expected files, so realization.snap.txt needs a Temporal-owned consumer.

## Resolved via Project Docs

- The owner-approved R1 and API contract require byte-identical activity behavior. The Realization alone owns system-field paths and action projections. These clarifications preserve that requirement and every existing R-ID.
- The active milestone goal explicitly delegates recommendations to the agent. This refinement asked zero questions, changed only Architecture & Data Models, API Contracts and these two evidence sections, and leaves fn-217/fn-229 dependencies and both task statuses unchanged. fn-217.3's final catalog is still pending; implementation must recheck the completed catalog rather than treating the earlier five-message examples as the entire transport universe.
