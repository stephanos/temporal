---
satisfies: [R4, R5]
---
# fn-217-annotation-catalog-over-the-temporal.6 Readable api.rs: derive(Catalog) mirror structs replace the annotate! block

## Description
Owner review of `model/temporal/features/activity/api.rs` (2026-10-20): "looks very verbose! this is meant to be read by humans". 627 lines, 259 over 120 columns, longest 925 characters; `api_transport.rs` is another 490 lines restating the catalog. The owner chose an attribute surface over the function-like `annotate!` block, and on 2026-10-21 an independent review (prior art: serde, clap derive, garde, validator, protovalidate, prost-reflect, sqlx/diesel, schemars, tonic-build, kubebuilder markers) was applied as-is; the owner will adjust later. This description is the design to spike and then implement. It supersedes the earlier unit-enum, snake_case-variant, `mirrors =`, `Scope`, `expect!` and line-count decisions.

### Surface: mirror struct, kinds as types, `#[rule]` and `#[mirror]` namespaces (derive)

- `#[derive(Catalog)]` on a plain struct whose field names are the proto field names exactly (including `r#type`), so no renaming rule exists. A derive leaves the item untouched for rustfmt, rustdoc and rust-analyzer; rustc still rejects unknown helper attributes, so the derive registers exactly two helper namespaces (owner, 2026-10-21, asking for a clearer label than `api`): `#[rule(...)]` for what the server enforces (constraints, presence, `rejects`) and `#[mirror(...)]` for how the struct maps to the proto (`request`/`response`, `type`, `via`, `oneof`, `decided`/`observed`, `ignored` reason, `idempotency_key`, `also`, `supplemental`). Ten bare names like `#[required]` would collide across crates. Stacked lines are allowed; most fields carry only a `#[rule]`.
- The **kind is the field type**: `Text`, `Identifier`, `Uuid`, `Duration`, `Timestamp`, `Counter`, `Flag` (bool, new), `Number` (f64, new), `Bytes`, `Secret`, `Enum<T>`, `Message<T>`, `OneOf<T>`, `Map<K, V>` (new, generates the entry catalog; replaces the hand-written `MetadataEntry`), `Ignored` (serde `skip` in type position). Rust-analyzer resolves these as real types; ~80% of fields then need no attribute at all.
- Constraints and expectations are attributes next to the field, protovalidate rule names: `#[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity|Identity")]`, `#[rule(gte = 0, ...)]`, `#[rule(max_len = dc::BLOB_SIZE_ERROR, ...)]`, `#[mirror(ignored = "reason")]` for the rare explained ignore, `#[mirror(idempotency_key)]`, `#[mirror(also(uuid, accepts))]` for a second kind intent on the same field (replaces `api_requests.rs`'s supplemental catalogs; a supplemental second mirror struct is also allowed, marked `#[mirror(supplemental)]` so the build script excludes it from exact coverage and the derive emits no borrowless pattern for it).
- Roles default from the item (`#[mirror(request = "proto.Name")]` is Decided, `#[mirror(response = "…")]` is Observed) and are kind-aware: `Secret` defaults to Decided everywhere (today's `secret_observed` admission error would otherwise fire on every response token). Mark exceptions with `#[mirror(decided)]` / `#[mirror(observed)]`.
- Presence is one algebra at item level, JSON Schema's (owner, 2026-10-21: "I like the rules concept best; let's fold required into that"): `#[rule(required(a, b), rejects = "…")]`, `#[rule(any_of(a, b), …)]`, `#[rule(one_of(a, b), …)]`, `#[rule(requires(a, b), …)]` (if `a` then `b`), `#[rule(excludes(a, b), …)]` (never both), the parent spec's five operators. There is no per-field `required`; a field's own `#[rule]` carries value constraints only. Field names inside a presence rule are checked against the struct by the derive. Each presence rule carries its `rejects` in the same attribute, so the expectation is bound to the rule, not floating on the struct (JSON Schema's `anyOf` error-reporting weakness is the reason). The probe emits one violation per named field of a `required`/`any_of` rule; `gaps.txt` lists them under the rule.
- No `covers` (owner, 2026-10-21: "we're binding it too much to internals of Temporal and Go; we don't want that kind of mapping"). The catalog is black-box: no server function names, no generated `validators.rs`, no `const` aliases for validator paths. The reverse direction is the probe's `Unannotated` outcome (parent R4 as amended); `reverse_coverage.go` and its tests are deleted.
- The prost type path is derived from the proto name (prost's package-to-module and CamelCase rules), removing the 25-type `use` block; `#[mirror(type = path)]` overrides. The generated pattern carries `#[allow(deprecated)]` (upstream deprecated fields).
- Enum catalogs disappear from `api.rs`: `defined_only` uses `<T as prost::Enumeration>::is_valid`; descriptor-vs-prost agreement is prost-build's job; Model-side enum exhaustiveness stays in the Model's own matches (`phase.rs`). No `mirrors =` convention: `Phase` is derived from a pair of enums and `Deadline` maps 2 of 5 `TimeoutType` values, so by-name mirroring would be mostly escape hatches; a hand-written wildcard-free `match` is already the rustc-exhaustive mapping.
- Doc comments on the struct and fields become the catalog's descriptions (schemars precedent) and appear in `gaps.txt`.

### Projections: a `View<K>` trait, not macro inference

Neither a derive nor an attribute macro sees the prost field types. Projection is trait dispatch at type-check time, replacing `types::check`: `trait View<K> { type Out: FieldValue; fn view(&self) -> Self::Out; fn set(&mut self, Self::Out) -> Result<(), GenerationError>; }` implemented for `(Option<ProtoDuration>, Duration)`, `(Option<ProtoDuration>, RawDuration)`, `(Option<ProtoTimestamp>, Timestamp)`, `(Option<T: Message>, Message<T>)` and `Vec<T: Message>` (encoded bytes), `(String, Uuid)` with the lenient normaliser, `(bool, Flag)`, `(f64, Number)`, plus `Box<T>` (for `Failure.cause`). The derive emits `View::<K>::view(&m.field)`; a kind that does not fit the field is "trait not implemented" from rustc. Real setters come for free, which `fill` and the probe's limit-plus-one mutation need (today's setters are no-ops). Custom projections stay explicit as `#[mirror(via = module)]` (serde `with` shape: a module with `view`/`set`) for the three genuine cases: `RetryPolicy.initial_interval`/`maximum_interval` conditional on `maximum_attempts`, `backoff_coefficient`, and `ActivityFailureInfo.activity_type` name extraction.

### Setter signature clarification (2026-10-21)

The conductor authorized fallible `View::set` and projected setter propagation after a real malformed encoded-message mutation path was found. Setters return `Result<(), GenerationError>`; malformed message bytes refuse before replacement, retain the original field, and produce value-free diagnostics. This clarifies annotation mutation admission only; Model and Go panic behavior remain unchanged. Regression evidence must include malformed-byte red/green and the actual generation/probe mutation path.

### Supplemental response linkage clarification (2026-10-21)

The conductor authorized `#[mirror(response_checks = RawRetryPolicy)]` on the full mirror to invoke its supplemental response mirror only for responses. The link must compile-check the same protobuf type and validate only the supplemental catalog, without recursion or another Inspect implementation. This preserves the existing raw RetryPolicy response check alongside request normalization. Tests pin request normalization, malformed raw response rejection, valid response multiplicity, and a wrong mirror/type path compile error.

### Expectations

- `Code` enum (the 16 gRPC status names, protocol names so R6 holds) in `umpire-annotations`; `rejects = "pattern"` defaults to `InvalidArgument` (45 of 45 today); `rejects(NotFound, "…")` otherwise; loose forms `#[rule(rejects)]` (code only) and `rejects = any_status` (any non-OK; not `Any`, which reads as `prost_types::Any`). Patterns are regexes; document anchoring. `gaps.txt` prints `Confirmed (code only)` / `Confirmed (any status)` with the server's actual message and ends with a per-feature count of loose entries; `Accepted` still fires for a loose entry the server took.

### Parameterised limits (dynamic config), simplified

`umpire-annotations` keeps plain integer bounds. `foundation/dc/` defines `pub struct Limit { key: &'static str, default: u64, scope: Scope }` constants (`MAX_ID_LENGTH` = `limit.maxIDLength`, Global, 1000; `BLOB_SIZE_ERROR` = `limit.blobSize.error`, Namespace, 2 MiB) and a `Limits` struct (`Default` = server defaults) that catalog factories take: `catalog::start_activity_execution_request(&dc::Limits::default())`; `max_bytes = dc::MAX_ID_LENGTH` expands to `limits.get(&dc::MAX_ID_LENGTH)` (garde context precedent). Temporal's precedence is per setting kind (`Global`, `Namespace`, `NamespaceID`, `TaskQueue`, `ShardID`, `TaskType`, `Destination` in `setting_gen.go`), not "most specific over dimensions", so the earlier generic `Scope`/`#[scope(..)]`/validate-time resolution is dropped. Feature-specific keys live in `feature/<name>/config.rs` with the same `Limit` type. Deferred: filling `Limits` per Run from the in-process server for one namespace.

### RPC level: generated table, hand-written exhaustive match

`temporal/api/build.rs` already reads the descriptor and generates `protocol.rs`; it additionally generates `services.rs` in tonic-build's shape: a `WorkflowServiceMethod` enum plus a table of `(method, request type path, response type path)`. The feature classifies with an ordinary wildcard-free `match` over that enum (or-patterns group the out-of-scope reasons), which replaces the `service!` grammar and the 115-line block; a new upstream method is "non-exhaustive patterns" naming it. `Action(Worker::Poll)` arms carry the RPC binding, so the reverse `actions bind` table is generated; `Environmental(..)` actions keep one declaration. The request-ID fill is derived from `#[mirror(idempotency_key)]` on a `Uuid`/`Text` field and is the only fill rule.

### Generated transport

`annotate()` dispatch comes from the generated table. The `Inspect` tree is derived from every `Message<_>`, `OneOf<_>`, `Map<_, _>` and `Vec<Message<_>>` field. `OneOf<T>` lists its alternatives in `#[mirror(oneof(Result = "result", Failure = "failure"))]`; exhaustiveness over alternatives is a generated `match` over the prost oneof enum, and the descriptor check sees the alternatives, not the prost field. Hand-written remainder of `api_transport.rs`: `AnnotationResult`, `annotate`, `process`, `findings`, the `Refusal` mapping, the three `via` modules and `uuid` normaliser (about 120 lines). `RetryPolicy`'s response-side raw re-check becomes a supplemental response mirror of `RetryPolicy`.

### Foundation module

`model/temporal/foundation/`: `api.rs` with `common.v1` (`Payload`, `Payloads`, `RetryPolicy`, `WorkflowExecution`, `WorkflowType`, `ActivityType`), all of `failure.v1`, `taskqueue.v1.TaskQueue`, `google.protobuf` `Duration`/`Timestamp`; `dc/` as above; `views.rs`. Nested coverage is workspace-wide, exactly one declaration per type (`descriptor::unique` already exists; `catalog_inputs` walks every `api.rs`). Generated `Inspect` impls for foundation types and feature types coexist only while both are modules of one crate; the README must say so (a feature split into its own crate hits the orphan rule, E0117; the escape is a local marker type parameter). The build script and the derive share one parser (`annotate/syntax`, kubebuilder precedent); two parsers for one grammar is forbidden.

### Checks and acceptance adjustments

- rustfmt owns layout; workspace `max_width = 100` applies (not 120); `cargo fmt --check` clean replaces the earlier 250/80-line targets. Build script checks proto declaration order of fields (full mirrors only), oneof alternatives and service methods, naming the first out-of-place item.
- The 20 trybuild fixtures under `annotate/tests/ui` are rewritten for the derive; E0027 on the generated pattern keeps its wording; budget all `.stderr` regenerations. R2's compile-error admission rules are preserved by the derive.
- Spike `StartActivityExecutionRequest` in this form first and show the owner, then implement: derive crate, `View` trait, `Code`, `dc`, generated `services.rs`, foundation split, rewrite of `api.rs`/`api_transport.rs`, removal of `api_requests.rs`, README, snapshots, `make model-check`.
### Execution and preservation constraints

**Touches:** `Makefile`, `model/umpire/annotations/**`, `model/temporal/annotate/**`, `model/temporal/api/**`, `model/temporal/foundation/**`, `model/temporal/feature/activity/api.rs`, `model/temporal/feature/activity/probe_gaps.rs`, `model/temporal/feature/activity/api_requests.rs`, `model/temporal/feature/activity/api_transport.rs`, `model/temporal/feature/activity/temporal.rs`, `model/temporal/feature/activity/expected/coverage*`, `model/temporal/feature/activity/expected/gaps.txt`, `model/temporal/realization/**`, `model/temporal/testing/**`, `model/temporal/proto/**`, `model/temporal/hosts/go/**`, `model/umpire/tests/**`, `model/README.md`, `model/DESIGN.md`, `model/Cargo.toml`, `model/Cargo.lock`, `tests/testpilot_model_gaps_test.go`
**Quick:** shared-parser/derive unit and trybuild checks, actual View setter tests, descriptor negative controls, rejection of removed validator-mapping syntax, generic black-box mutation/classification controls, native activity fixtures, and `make umpire-check-model-gaps` including the privacy canary.
**Live gate budget clarification (2026-10-21):** The 2,301-case strict and generic corpus exceeded the former aggregate five-minute Go test timeout while continuing to advance (303.786 seconds; the current operation had run one second). The conductor authorizes only `umpire-check-model-gaps` in `Makefile` to use a 20-minute aggregate timeout. Preserve every case, the 30-second per-operation bounds, frame limits, classification and privacy checks; retain the timed-out run as failed evidence. The complete corpus also exceeds the shared testcontext default of 90 seconds at cleanup; pin the same 20-minute aggregate ceiling only at the start of `TestTestpilotModelGaps`, before environment/decorator creation. Preserve shared defaults, standalone controls, per-operation 30-second and cleanup five-second bounds. A focused deadline/extension regression must observe the actual context ceiling without waiting for expiry.

**Full gates:** `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`; scoped Go lint for any changed Go surface.

- Start with one real compiling Start request mirror, compare it with the existing catalog, and send the conductor its source and evidence for presentation before broader migration. The spike is the first proof point, not a replacement for this task's complete acceptance.
- Preserve parent R1 enum-value admission. `phase.rs` has wildcard matches, and prost `is_valid` follows regenerated values, so neither catches newly added declared values. Retain a checked-in, explicitly reviewed enum inventory in `foundation/enum_inventory.rs`, separate from human-facing `api.rs`, using the shared parser and existing descriptor/exhaustive-check machinery. Do not automatically rebuild this admission inventory from a bumped descriptor. Runtime defined-only validation can use prost's inherent `is_valid` through a typed bridge where needed.
- Preserve parent R2's explicit ignored-field reasons. The Start mirror retains all seven existing reasons through `#[mirror(ignored = "reason")]`; bare unexplained ignores must not silently bypass admission.
- Keep the value-free/privacy boundary and its actual live canary passing. Loose expectation diagnostics must not leak private request, response or error contents; record any required safe rendering interpretation in the authoring documentation.
- Do not change Model transitions, phase mappings, panic behavior, ABI, the Umpire Model protocol, Realization judgments or report/rows/states goldens. The annotation-probe envelope may retire validator-mapping fields and add the probe-only inventory, applicability/refusal and predicted-expectation/match metadata needed for the owner's black-box amendment. Reserve removed protobuf field numbers, use fresh numbers for additions and regenerate both consumers. Rust still constructs mutations and classifies outcomes; Go transports and matches declared expectations without exposing private error contents. Admit only bounded, case-correlated metadata and reject malformed or crossed observations. Presence/description/loose-expectation changes, removal of validator mappings, and black-box mutation findings may alter annotation and gap output, with an exact reviewed diff. Existing claims and classifications must not be silently weakened or waived. No new third-party dependency or version; existing workspace dependencies/internal wiring only where necessary.

## Acceptance
- [ ] `api.rs` files use `#[derive(Catalog)]` structs with kinds as field types and the `#[rule]`/`#[mirror]` helper namespaces, presence rules only at item level; no function-like `annotate!` block, no closures in catalogs, no `api_requests.rs`; `cargo fmt --check` is clean and no line exceeds the workspace `max_width`.
- [ ] No `covers` anywhere (owner, 2026-10-21): the `#[rule]` grammar has no `covers`, `foundation/validators.rs` and its generator are deleted, no validator-path `const` aliases remain in any `api.rs`, and `gaps.txt` prints no `[covers: …]`. Gap analysis is mutated requests against the real implementation in both directions: the probe's `Mismatch`/`Accepted` for catalog claims and a new `Unannotated` outcome for rejections no rule predicts, produced by generic per-field mutations (absent, empty, limit plus one, negative, undefined enum value). `reverse_coverage.go`, `reverse_coverage_test.go` and the validator-freshness test are deleted.
- [ ] All parent R1 exhaustiveness levels remain build-failing: fields via the generated borrowless pattern (E0027), enum values via the reviewed admission inventory and exhaustive checks, oneof alternatives via a generated match, RPC methods via a wildcard-free match over the generated `WorkflowServiceMethod`, Model actions via the binding table; each pinned by a trybuild fixture or build-script negative control.
- [ ] Proto declaration order of fields, oneof alternatives and methods is a build error naming the first out-of-place item.
- [ ] `coverage.snap` carries the same messages and classifications as before (field-level presence splits are the only expected diff, listed in the PR); `gaps.txt` shows inherited and loose patterns; `temporal/annotate/README.md` documents the surface, the single-parser rule and the single-crate orphan-rule caveat; `make model-check` passes.
## Done summary
Implemented readable derive(Catalog) mirrors with exactly rule/mirror namespaces,
item-level presence algebra, shared parser/descriptor admission, generated typed
inspection and RPC dispatch, reviewed enum inventories, dynamic limits and fallible
setters. Preserved Model transitions, panic behavior, judgment, ABI and protocol.

Replaced validator-source mappings with independent raw-schema mutations for all
373 request fields (2,238 generic cases). The reviewed report has 79 confirmed
strict cases, 754 actual generic observations and 1,484 explicit local untested
outcomes. All 217 narrowly dated Unannotated dispositions remain visible; no
prefix/future-case waiver exists. Exact golden approval and verbatim correctness/
standards audits are in .flow/artifacts/fn-217-annotation-catalog-over-the-temporal/.

The first formal review delivered NEEDS_WORK with one actionable required-Number
generation defect. Actual seed-3 RED/GREEN drove fix 06caf99b. Renewed default,
required-backend and full live gates pass against the rebuilt guest. The single
resumed review returned SHIP, prior finding fixed, no unaddressed R-IDs. The
reviewer explicitly withdrew the duplicate guard finding against locked Syn 3.
Review session 01a15d09-1a64-76f3-a6a8-6612cfec3574; receipt
/tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.6.json.
Both writer and reviewer are Codex-family; actual reviewer gpt-6.1-sol.

Final live log is complete and uncompromised: 2,317 corpus identities match PASS
subtests, plus 19 boundary controls; privacy canary passes. Actual report equals
the approved golden SHA a3974ea8da0f3c46234770068fe0f163d5e17529b275cea086c2292a124284a0.
Frame sizes 692785/503633/879201 remain under 1 MiB. Timing exits 0: full gate
165.509s misses soft 30s target; inner 1.954s meets soft 10s target. No reduced
corpus, skipped assertion or claimed activity TLC agreement; unsupported activity
source remains explicitly refused while the required backend's admitted controls run.

Foreign fn-233 plans and MILESTONES changes remain preserved outside task commits.
Dirty MILESTONES prevents reusable clean-worktree receipts; passing full commands
are actual execution evidence, not fabricated receipt reuse. Integrated task-owned
worktrees were normally removed with verified recovery archives retained.

stage: impl-review - ran [2026-10-21T05:57..2026-10-21T06:27UTC], NEEDS_WORK then SHIP after one committed fix and renewed gates (model: gpt-6.1-sol)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: QA - skipped(config: pipeline.qa=off)
## Evidence
- Commits: 1c4770b9260eae3cddbdbfd96d1a0b2a4a907a5e, e101cda4322a20bc086069ee19e39fb9cd49fc19, f55725be9b1ea31908870e265ed3a2ae42c89209, edd4829162610c9d840dd50e4cf5ef56b746208c, 43cf9d0e50705493be83ed91d9a1ad0c481a850a, a4ac04c739469f7de89a5ff3d671ead01174012f, 424f6eb7a7b8c160b7270c8ce21b639aa3aedaea, 616d378223cd29b38af4771f11fa126542377406, 8f59fcf4e85cb7fc7cf6f9f519338f5bee4c1885, 870ee1de9ac33b9e87428cac03c8a3f2e7b243b5, 06caf99bcc0614463c4316db7414eb0f184ae71d, 01c90fbce3, 2fc8a58d6b
- Tests: make model-check: session82482 exit0 at06caf99b; default TLC skip explicit, UMPIRE_BACKENDS=require make model-check: session62470 exit0 at06caf99b, real admitted-source TLC controls, make umpire-check-model-gaps: session52071 exit0,563.085s; all2317 corpus+19 boundary controls and privacy; exact approved golden, make model-time: session25337 exit0; gate165.509s soft30MISS; inner1.954s soft10PASS, Focused parser/derive/trybuild/descriptor/setter/native suites, malformed-byte RED/GREEN, required-Number seed3 RED/GREEN, scoped integration Go lint: passed; tracked verification/audit artifacts retain details, flowctl codex impl-review: single resumed session50211 exit0, VERDICT=SHIP at2fc8a58d; receipt round2, prior finding fixed, zero unaddressed R-IDs
- PRs: