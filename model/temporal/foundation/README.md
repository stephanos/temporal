# Exhaustive annotation catalogs

Feature authors put ordinary `#[derive(Catalog)]` mirrors in
`temporal/feature/<feature>/api.rs`; reusable messages live in
`temporal/foundation/api.rs`. Upstream protos stay unchanged. Mirror fields follow
proto declaration order, use the exact source names, and name their semantic kind
as a Rust type. The mirrors describe catalogs; they are not transport messages.
The realization crate re-exports the generated `temporal::api` namespace and defines
`CatalogContext = foundation::dc::Limits` at its root for the shared derive defaults.
Standalone consumers can select both explicitly with `#[mirror(api = ..., context = ...)]`.

```rust,ignore
use crate::foundation::dc;
use umpire_catalog::{Catalog, kinds::*};

/// Start an activity with an explicit identifier.
#[derive(Catalog)]
#[mirror(request = "temporal.api.workflowservice.v1.StartActivityExecutionRequest")]
#[rule(required(request_id), rejects = "request_id")]
struct StartActivityExecutionRequest {
    // The actual mirror must include every upstream field, in declaration order.
    #[mirror(idempotency_key)]
    #[rule(rejects = "request_id")]
    request_id: Uuid,
    #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "activityId")]
    activity_id: Identifier,
}
```

This abbreviated example is intentionally not an exhaustive mirror; use the real
[activity declaration](../feature/activity/api.rs) as a compiling example.
Exactly two helper namespaces separate server rules from protobuf mapping:
`rule` holds constraints and expectations; `mirror` holds mapping, roles and tags.
Rust doc comments become value-free descriptions beside probe descriptions.

## Kinds, roles and constraints

Scalar kinds include `Text`, `Identifier`, `Uuid`, `Timestamp`, `Duration`,
`RawDuration`, `Counter`, `Flag`, `Number`, `Bytes` and `Secret`.
`Enum<ProtoEnum>` checks defined discriminants; `Message<ProtoMessage>`,
`Vec<K>`, `Map<K, V>` and `OneOf<ProtoOneof>` retain typed nested structure.
Oneofs list their source alternatives with
`#[mirror(oneof(Result = "result", Failure = "failure"))]`.
A full mirror must classify every alternative.

Request fields default to Decided and response fields to Observed. Secret fields
always default to Decided. Override with `#[mirror(decided)]` or `#[mirror(observed)]`;
`Ignored` requires `#[mirror(ignored = "specific reason")]`. Ignored fields still
participate in coverage, but rules cannot depend on them. Contradictory roles,
incompatible tags and incompatible prost/kind pairs are compile errors.
Tags include `idempotency_key`, `immutable`, `monotone`, `cached = seconds` and
`eventually_consistent = polls`. The generic runtime builder also admits typed
custom tags with concrete consequences.

Constraints are named `#[rule]` attributes, such as `min_len`, `max_bytes`, `prefix`,
`gte`, `gt`, `min_items`, `max_items` and `unique`. String lengths count
Unicode scalars; byte bounds count UTF-8 bytes. Dynamic bounds reference typed
`dc::Limit` constants. Factories take `&dc::Limits`, whose defaults match server
limits and can be overridden per run without global state.

Presence belongs on the item, never on fields. The five operators are
`#[rule(required(a, b), rejects = "missing")]`, `any_of(a, b)`, `one_of(a, b)`,
`requires(a, b)` and `excludes(a, b)`. Each request rule carries its own expectation
in the same attribute. Required and any-of probes are attributed to each named
field; an any-of counterexample omits every alternative, since one present
alternative satisfies the rule. Fields carry only value constraints in `#[rule]`.
Request constraints require an expectation. Available expectations are `accepts`,
`unobservable`, `rejects = "pattern"` (InvalidArgument), and
`rejects(NotFound, "pattern")` with a named non-OK `Code`.
Bare `rejects` checks only the default status. Patterns are regular expressions;
use `^...$` when exact anchoring matters.

`rejects(NotFound)` and `rejects = any_status` are explicit weaker assertions:
they are labelled in gap output, retain the actual status, and contribute to the
per-feature loose-expectation count. Their server messages are redacted, not
printed as evidence. Descriptions and paths contain no message values.

## Projections and supplemental intent

`#[mirror(via = views::projection)]` selects a typed `View<K>`. Setters return
`Result<(), GenerationError>`; invalid encoded messages fail before assignment,
leave the original field unchanged, and report no input contents. Filling,
negative-probe construction and redaction propagate that error.

Use `#[mirror(also(uuid, accepts))]` for an additional intent on a field without
duplicating a complete mirror. `Mirror::catalog(&limits)` returns its primary
catalog; `Mirror::catalogs(&limits)` includes additional intents. A
`#[mirror(supplemental)]` mirror may project a subset for a contextual rule.

Request normalization must not hide malformed responses. The item link
`#[mirror(response_checks = RawRetryPolicy)]` adds a supplemental catalog only when
inspecting responses, with a compile-time check that both mirrors use the same
prost type. The primary `Inspect<dc::Limits>` implementation owns recursion;
the supplemental catalog validates once without installing another implementation.
The worker-failure root rule is similarly contextual: transport selects the
declarative `WorkerFailure` catalog only for a worker failure request, while the
ordinary `Failure` mirror still traverses nested and server failures.

## Exhaustiveness and generated glue

The derive and build scanner share one parser. Generated field patterns omit
`..`, and descriptor checking enforces fields and oneof alternatives in source
order. The explicit reviewed `foundation/enum_inventory.rs` is checked against
both descriptors and wildcard-free Rust matches. Runtime enum validity alone is
not an API-update admission check.

The API build generates `WorkflowServiceMethod`, its name/type table and
annotation dispatch from linked service descriptors. An ordinary exhaustive
`classify(WorkflowServiceMethod) -> RpcRole` function binds each RPC to a Model
action, observation or an OutOfScope reason. Reverse action mappings remain
exhaustive. A newly added field, enum value, oneof alternative, RPC or Model action
therefore fails until classified; no wildcard silently admits it.
The retained `annotate!` grammar supports reviewed enum inventories and compile
controls for custom-tag/action admission, not a parallel full-message catalog.

Generated `Inspect<dc::Limits>` recursively validates nested messages, repeated
messages, maps and oneofs, and fills only absent generated identifiers. Shared
foundation and feature declarations currently compile in one crate. Splitting
them into separate crates requires a local context marker at the orphan-rule seam;
do not add overlapping inspection implementations.

Gap analysis is black-box. No catalog attribute names server functions or validator
paths. Alongside strict catalog probes, descriptor-driven typed mutations attempt
each request field independently of its catalog kind, including ignored fields.
Absent, empty, limit-plus-one, negative and undefined-enum mutations record their
applicability explicitly. Recursive shapes have bounded traversal and report
cycle or depth refusals. An actual rejection predicted by no catalog rule is
`Unannotated`, not a successful strict probe or an automatic waiver.

The activity probe's private `feature/activity/probe_gaps.rs` ledger lists exact
reviewed identities with dates and scope reasons. Its known-Unannotated entries
apply only to the reviewed rejection code; they cannot waive an unexpected
acceptance, mismatch, new code or new identity. Rows remain visibly Unannotated.
General dated Accepted/Mismatch dispositions remain separate. A synthesized
missing ancestor or oneof alternative records mutation provenance, not proof
that the named descendant caused the rejection.

A generic rejection is explained when any violated rejection rule matches its
status, message pattern and details type. All predicted rule identities remain
visible; generic `Confirmed` means the outcome is explained, not that every rule
was independently proved. An `accepts` expectation does not override a different
violated rejection rule. If the server accepts while any rejection rule applies,
the generic result is `Accepted`. A rejection with observable predictions but no
match is `Mismatch`; none or only `unobservable` predictions is `Unannotated`.
Strict isolated probes still check each catalog claim separately.

Preparation validates the actual baseline and checks its predicted rule identities
against canonical metadata before dispatch. Divergence refuses without exposing
values. Classification requires the complete indexed prediction set, so missing
or crossed metadata cannot turn a known violation into a successful probe.
An unchanged mutation with no observable violations remains explicitly untested.

The linked-descriptor build step uses `getproto --linked-api` because the upstream
image omits services; it never edits the upstream image. The realization runtime has no parser or descriptor dependency.

After an API update run `make model-check`, classify newly reported symbols and
review all ignore/exclusion reasons. For activity catalog changes also run
`make umpire-check-model-gaps`; inspect changed probe identities and descriptions
before updating its golden. The live gate includes the value-free privacy canary.
