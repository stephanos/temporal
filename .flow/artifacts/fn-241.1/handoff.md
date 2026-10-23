# fn-241.1 handoff (lane C, branch work/fn2411-classify, worktree .worktrees/fn2411-classify)

## Commit
- 481017ad95 refactor(temporal): one WorkflowService table classifies every RPC by family (on 22c20a7eae)

## Design
- `model/temporal/foundation/service.rs` (new, outside feature/): `pub enum RpcRole<A>` (moved from
  the activity feature, generic over the feature's action) and `pub const fn family(method:
  WorkflowServiceMethod) -> &'static str`, one exhaustive match over all 123 RPCs in descriptor
  order. Same-family neighbours are grouped with or-patterns into 40 arms and 11 families:
  Namespace administration; Workflow executions; Activity tasks and executions; Cluster and service
  information; Task queue administration; Worker infrastructure; Schedules; Worker deployments and
  versioning; Batch operations; Nexus tasks and operations; Workflow rules.
- `feature/activity/api.rs`: `classify` holds only the eight claimed arms plus
  `_ => RpcRole::OutOfScope(service::family(method))`. `annotate` is unchanged.
- Generic syntax (`umpire/catalog/syntax/src/services.rs`, takes no Temporal names): it now reads
  `family` tables and `classify` claim lists, i.e. a match whose last arm is a wildcard that defers
  exactly to `family(method)`. `Declarations::resolve` merges them in table order. A complete
  exhaustive `classify` with no wildcard still works the old way, so Kettle is unchanged.
  Named diagnostics:
  - `duplicate feature claim: <svc>.<rpc>`
  - `unclassified RPC: <svc>.<rpc> is claimed but absent from the service table`
  - `duplicate service table`
  - `feature claims without a service table`
  - `complete classification duplicates the service table`

  An RPC missing from the table fails descriptor admission with `uncovered protobuf symbol:
  <svc>.<rpc>` and fails the compiled exhaustive match with E0004.
- `umpire/catalog-build/src/coverage.rs`: it now also discovers `service.rs`, merges declarations
  across files and resolves them once.
- `temporal/annotate/syntax/src/services.rs`: adds `declarations` and `resolve` wrappers.
- `model/DESIGN.md`: updated the RPC methods row and the tracked-inputs sentence.

## Acceptance
- [x] One service table outside feature/ classifies every RPC and owns exhaustiveness. Evidence:
  api `tests/services.rs::the_service_table_classifies_every_rpc_and_the_activity_feature_claims_eight`
  runs descriptor::check against the real linked descriptor using the real table and claims.
  `service_table_and_bindings_refuse_new_rpc_and_model_action` shows the table match gives E0004
  for an added RPC and the bind macro gives E0004 for an added Model action. These are the
  converted original controls.
- [x] The activity feature declares only its eight RPCs (asserted in that test, in order). No
  out-of-scope reason contains "Model" (asserted).
- [x] A duplicate claim or an unclassified RPC fails the build with a named diagnostic. Evidence:
  `duplicate_claims_and_unclassified_rpcs_are_named` (real sources plus a second-feature claim; a
  claim the table lacks; an unclaimed RPC dropped from the table is refused by descriptor
  admission); catalog-build `neutral.rs::coverage_build_resolves_feature_claims_against_one_service_table`
  (the build entry point `coverage::check` with two api.rs files); syntax unit tests.
- [x] Generated dispatch and bindings are unchanged. The generated `services.rs` before and after
  is identical once OutOfScope reason strings are masked (`.flow/tmp/fn-241.1/services-{before,after}.rs`).
  RPC_BINDINGS order, the bind macro and the dispatch macro are byte-identical.
- [x] Guest rebuilt and committed; slice and gaps pass.

## Gates (all exit 0; logs in .flow/tmp/fn-241.1/)
- Focused cargo tests for umpire-catalog-syntax, umpire-catalog-build, umpire-temporal-annotate-syntax,
  umpire-temporal-api, umpire-temporal-annotate, umpire-temporal, umpire-temporal-tests and
  umpire-reference-kettle: pass (focused-tests*.log; the first run's one failure was the reason-text
  assertion fixed below)
- cargo fmt --check plus clippy -D warnings on the touched crates: clean
- make model-check (default): exit 0 (model-check-default.log)
- UMPIRE_BACKENDS=require make model-check: exit 0 (model-check-require.log)
- make umpire-check-model-slice: exit 0 (model-slice.log)
- make umpire-check-model-gaps: exit 0, 510 s (model-gaps.log)

## Changed reason text and golden files
- model/temporal/feature/activity/api.rs and model/temporal/foundation/service.rs: out-of-scope
  reasons are now family names.
- Generated OUT_DIR services.rs RPC_BINDINGS OutOfScope targets (not committed).
- model/temporal/realization/tests/catalog.rs: the GetSystemInfo assertion changed from
  contains("capabilit") to == "Cluster and service information".
- model/temporal/hosts/go/umpire_temporal_guest.wasm: rebuilt, sha256 d80dd567... -> 757fd2e8...
- No snapshot, expected/ or behavioural golden changed.

## Decisions (made on the owner's behalf)
- The table is a plain string-returning `family` match, not a Family enum, so the scanner reads
  one function. The reason is the family name itself, so it stays accurate both in the global
  RPC_BINDINGS and in each feature's annotate result.
- The table keeps descriptor order because descriptor admission enforces order, so a family
  recurs between groups (match_same_arms is allowed, with a reason).
- The claim function keeps the name `classify`, with a wildcard fallback. The scanner accepts that
  wildcard only in this exact deferring form.
- The table lives in `foundation/`, which is already shared by the realization crate and by the
  annotate crate's tests. Catalog discovery adds the generic filename `service.rs`.
- Finer per-RPC wording (e.g. "SDK namespace discovery...") is folded into its family.

## Open items
- The disk notes in the brief mention the overlay; it was at 13 GiB free during the focused tests
  and later recovered to 37 GiB. The host mount never dropped below 29 GiB.
- I'm finished with the shared target dir (.worktrees/fn2385-interpreted/model/target); lane B can
  resume.

## Review follow-up (review.md: SHIP with minor findings)

Commit: 427231772b fix(catalog): feature claims bind or observe, and annotated RPCs must be claimed

- Minor 1: a claim list that also has an `OutOfScope` arm is refused with
  `a feature claim must be Action or Observed: <svc>.<rpc>`. Test:
  syntax `claims_bind_or_observe_and_name_their_wildcard_parameter`.
- Minor 2 (generic, in `umpire_catalog_build::services::generate`): generation refuses an RPC whose
  request or response is mirrored but resolves out of scope, with
  `annotated RPC is out of scope: <svc>.<rpc> has mirrored messages but no Action or Observed classification`.
  Today the mirrored RPC set is exactly the eight claims, and Kettle builds unchanged. Tests:
  - neutral `an_annotated_rpc_must_be_claimed`
  - api `a_dropped_claim_or_an_unclaimed_annotated_rpc_fails_generation`, which covers a dropped
    `DescribeActivityExecution` Observed claim, and a newly mirrored `RecordActivityTaskHeartbeatRequest`
    left unclaimed, both against the real sources and descriptor.
- Nits:
  - The wildcard accepts only `family(<param>)` or `service::family(<param>)`; `helpers::family` and
    `a::service::family` are refused.
  - The wildcard diagnostic uses the real parameter name.
  - All five resolve errors are asserted verbatim, including
    `complete classification duplicates the service table`.
  - `build_inputs.rs` asserts that `foundation/service.rs` is a tracked input.
  - The `ServiceTable` re-export is removed.
- Left as is, per the conductor: minor 3 (reason wording) and the future multi-feature bind-macro
  collision (nit 8).

Gates, all exit 0, run with the private target `/home/agent/.cache/codex-build/fn2411-target`
(now deleted):
- focused tests on the eight crates: 50 test results, all ok (focused-tests-fix.log)
- `cargo fmt --check` and clippy `-D warnings` on the touched crates: clean
- `make model-check`, default: exit 0 (model-check-default-2.log)

Guest bytes unchanged: only build-time code and tests changed, the worktree is clean after the
gate, and its guest-freshness step passed. So slice and gaps were not rerun; the earlier runs on
481017ad95 still stand.

I did not use the shared target for this round.
