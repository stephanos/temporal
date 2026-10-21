# Exhaustive annotation catalogs

Feature authors put ordinary `#[derive(Catalog)]` mirrors in
`temporal/feature/<feature>/api.rs`; reusable messages live in
`temporal/foundation/api.rs`. Upstream protos stay unchanged. Mirror fields follow
proto declaration order, use the exact source names, and name their semantic kind
as a Rust type. The mirrors describe catalogs; they are not transport messages.

```rust,ignore
use crate::foundation::{dc, validators};
use umpire_temporal_annotate::{Catalog, kinds::*};

/// Start an activity with an explicit identifier.
#[derive(Catalog)]
#[api(request = "temporal.api.workflowservice.v1.StartActivityExecutionRequest")]
#[api(covers = validators::activity::validateAndNormalizeStartRequest)]
struct StartActivityExecutionRequest {
    // The actual mirror must include every upstream field, in declaration order.
    #[api(required, idempotency_key, rejects = "request_id")]
    request_id: Uuid,
    #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "activityId")]
    activity_id: Identifier,
}
```

This abbreviated example is intentionally not an exhaustive mirror; use the real
[activity declaration](../feature/activity/api.rs) as a compiling example.
All helper attributes use the single `api` namespace. Rust doc comments become
value-free catalog descriptions and appear beside probe descriptions.

## Kinds, roles and constraints

Scalar kinds include `Text`, `Identifier`, `Uuid`, `Timestamp`, `Duration`,
`RawDuration`, `Counter`, `Flag`, `Number`, `Bytes` and `Secret`.
`Enum<ProtoEnum>` checks defined discriminants; `Message<ProtoMessage>`,
`Vec<K>`, `Map<K, V>` and `OneOf<ProtoOneof>` retain typed nested structure.
Oneofs list their source alternatives with
`#[api(oneof(Result = "result", Failure = "failure"))]`.
A full mirror must classify every alternative.

Request fields default to Decided and response fields to Observed. Secret fields
always default to Decided. Override with `#[api(decided)]` or `#[api(observed)]`;
`Ignored` requires `#[api(ignored = "specific reason")]`. Ignored fields still
participate in coverage, but rules cannot depend on them. Contradictory roles,
incompatible tags and incompatible prost/kind pairs are compile errors.
Tags include `idempotency_key`, `immutable`, `monotone`, `cached = seconds` and
`eventually_consistent = polls`. The generic runtime builder also admits typed
custom tags with concrete consequences.

Constraints are named attributes, such as `min_len`, `max_bytes`, `prefix`,
`gte`, `gt`, `min_items`, `max_items` and `unique`. String lengths count
Unicode scalars; byte bounds count UTF-8 bytes. Dynamic bounds reference typed
`dc::Limit` constants. Factories take `&dc::Limits`, whose defaults match server
limits and can be overridden per run without global state.

A field `#[api(required)]` creates a separate presence rule. Put
`#[api(requires = other)]` or `#[api(conflicts_with = other)]` on the trigger field.
Item-level groups use `#[api(any_of(a, b))]` or `#[api(one_of(a, b))]`.
Request constraints
and rules require both an expectation and nonempty typed validator provenance;
item defaults apply unless a field or rule overrides them. Available expectations
are `accepts`, `unobservable`, `rejects = "pattern"` (InvalidArgument), and
`rejects(NotFound, "pattern")` with a named non-OK `Code`.
Bare `rejects` checks only the default status. Required rules inherit a field's
pattern unless explicitly overridden, and the report labels that inheritance. Patterns are
regular expressions; use `^...$` when exact anchoring matters.

`rejects(NotFound)` and `rejects = any_status` are explicit weaker assertions:
they are labelled in gap output, retain the actual status, and contribute to the
per-feature loose-expectation count. Their server messages are redacted, not
printed as evidence. Descriptions, paths and provenance contain no message values.

## Projections and supplemental intent

`#[api(via = views::projection)]` selects a typed `View<K>`. Setters return
`Result<(), GenerationError>`; invalid encoded messages fail before assignment,
leave the original field unchanged, and report no input contents. Filling,
negative-probe construction and redaction propagate that error.

Use `#[api(also(uuid, accepts))]` for an additional intent on a field without
duplicating a complete mirror. `Mirror::catalog(&limits)` returns its primary
catalog; `Mirror::catalogs(&limits)` includes additional intents. A
`#[api(supplemental)]` mirror may project a subset for a contextual rule.

Request normalization must not hide malformed responses. The item link
`#[api(response_checks = RawRetryPolicy)]` adds a supplemental catalog only when
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
action, observation or an OutOfScope reason. Reverse action bindings remain
exhaustive. A newly added field, enum value, oneof alternative, RPC or Model action
therefore fails until classified; no wildcard silently admits it.
The retained `annotate!` grammar supports reviewed enum inventories and compile
controls for custom-tag/action admission, not a parallel full-message catalog.

Generated `Inspect<dc::Limits>` recursively validates nested messages, repeated
messages, maps and oneofs, and fills only absent generated identifiers. Shared
foundation and feature declarations currently compile in one crate. Splitting
them into separate crates requires a local context marker at the orphan-rule seam;
do not add overlapping inspection implementations.

Validator paths reference committed `foundation/validators.rs` identities. The
Go `TestValidatorCatalogFreshness` control regenerates and compares those
identities from actual AST functions. Normal Rust builds do not regenerate them
or run that Go test. The existing linked-descriptor build step uses
`getproto --linked-api` because the upstream image omits services; it never edits
the upstream image. The normal facade has no parser or descriptor dependency.

After an API update run `make model-check`, classify newly reported symbols and
review all ignore/exclusion reasons. For activity catalog changes also run
`make umpire-check-model-gaps`; inspect changed probe identities and descriptions
before updating its golden. The live gate includes the value-free privacy canary.
