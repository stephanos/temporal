# fn-241.1 review: commit 481017ad95 (work/fn2411-classify) on 22c20a7eae

## Verdict: SHIP

R1 and R2 hold, and I checked them independently rather than trusting the handoff. I found no
blocker or major issue. The minor findings below can be fixed in this task or deferred as follow-ups.

## What I verified (independently)

- **Generated output.** I rebuilt `umpire-temporal-api` from a `git archive` of 22c20a7eae and from
  481017ad95. The base OUT_DIR `services.rs` is byte-identical to the handoff's `services-before.rs`,
  and the commit's is byte-identical to `services-after.rs`. With every `"OutOfScope", "<reason>"`
  masked, before and after are identical, so dispatch, the method table, RPC_BINDINGS order and roles,
  and the bind macro have not changed. 230 changed lines, all OutOfScope reason strings.
- **annotate.** `api_transport.rs` still calls the feature's `classify`. Claimed arms are unchanged,
  and the out-of-scope reason now comes from the same `service::family` the scanner reads, so the
  runtime reason and RPC_BINDINGS agree.
- **Guest.** I diffed `strings` of the before and after wasm. The only data-segment change removes the
  12 old "...activity Model..." reasons and adds the 11 family names. Everything else is code bytes,
  which is expected from the `_ => family(method)` codegen. The committed sha256 is 757fd2e8..., and
  model-check-default.log shows the guest built inside the freshness gate. The slice and gaps logs
  exit 0.
- **Exhaustiveness.** `foundation/service.rs` covers 123 RPCs in descriptor order with no duplicates,
  in 40 arms and 11 families. A new upstream RPC fails in two places: build.rs (`uncovered protobuf
  symbol: <svc>.<rpc>`, before any compile) and E0004 in the `family` match. The scanner rejects a
  `_` arm in a table, and the converted compile control still gets E0004 for both an added RPC and
  an added Model action.
- **Bindings with the wildcard.** If an Action claim arm is removed, that action loses its macro arm,
  and `bind_workflow_action!` fails with E0004 (the environmental pattern covers only `Deadline`).
  I also probed removing an Observed arm (`DescribeActivityExecution`). It compiles, but two tests
  catch it: `the_service_table_classifies_every_rpc_and_the_activity_feature_claims_eight` and
  `realization/tests/annotations.rs::annotation_findings_preserve_server_bytes...`. See finding 2.
- **Diagnostics through the real build path.** `coverage::check` (the build.rs entry point) reports
  `duplicate feature claim: ...`, as the new neutral.rs test shows. The other resolve errors use the
  same `resolve` call, so they surface the same way.
- **Scanner robustness** (throwaway probe test, now removed):
  - Accepted: comments, `pub(crate) const fn`, fn attributes, a renamed parameter, and
    `crate::...::family(method)`.
  - Rejected: arm attributes and guards (with a misleading message, already true at base), a
    misplaced or non-deferring wildcard, and a second table (including one in a nested `mod`).
  - Not scanned: functions in `impl` blocks, as at base.
  - A duplicate arm inside one claim list is refused as a "duplicate feature claim".
  - Only one `service.rs` exists under `model/` (temporal/foundation), and the build tracks it with
    rerun-if-changed.
- **Generic layer.** `umpire/catalog/syntax` and `catalog-build` gain only the generic names
  `family`, `classify`, `RpcRole` and `service.rs`. Service and method enum stay parameters. No
  Temporal knowledge was added, and no dependency was added (`proc_macro2`/`BTreeMap` were
  already in use).
- **Kettle.** It keeps one exhaustive `classify` with no table and resolves through the
  `[] if claims.is_empty()` path. All `umpire-reference-kettle` tests pass, including
  `exhaustiveness` and `descriptor`.
- **Tests.** These pass on 481017ad95: umpire-catalog-syntax, umpire-catalog-build,
  umpire-temporal-annotate-syntax, umpire-temporal-api, umpire-temporal-annotate, umpire-temporal
  and umpire-reference-kettle. I ran them with a private target dir, which I have since deleted.
  The worktree is clean.

## Findings

1. **minor.** A claim list can claim an RPC as `OutOfScope("…")`.
   - Where: `model/umpire/catalog/syntax/src/services.rs:314` (claim arms reuse the complete
     classification path) and `:268-279`.
   - Scenario: a feature writes `WorkflowServiceMethod::RecordActivityTaskHeartbeat =>
     RpcRole::OutOfScope("not in this activity Model")` above its wildcard. The scanner accepts it as
     a claim, which has two effects:
     - It reintroduces a per-feature reason that names a Model. Only the activity-specific api test
       would notice, and a second feature's claims are not covered.
     - It reserves the RPC. A later feature that really binds it gets
       `duplicate feature claim`, even though nobody bound it.
   - This contradicts the spec's "each feature declares only the RPCs it binds, with its own Model
     actions or an observed note".
   - Fix: when `claims` is true, reject `OutOfScope` arms with a named error such as
     `a feature claim must be Action or Observed: <svc>.<rpc>`, and add the case to
     `tables_and_claim_fallbacks_are_declarative`.

2. **minor.** The wildcard means no build-time check that an Observed claim exists, or that a new
   activity-family RPC was considered.
   - Where: `model/temporal/feature/activity/api.rs:473`.
   - Scenario 1: deleting the `DescribeActivityExecution => Observed` arm compiles. The probe showed
     that only the hard-coded eight-claims list and one realization test catch it. A future feature
     with no such list would silently move its Observed RPCs to out of scope in RPC_BINDINGS.
   - Scenario 2: upstream adds, say, `RecordActivityTaskHeartbeatV2`. The build forces only a family
     arm in `service.rs`, and the activity feature's wildcard absorbs the RPC without anyone
     deciding whether the feature should claim it. The old exhaustive `classify` forced that
     decision.
   - The spec accepts this trade-off ("an RPC no feature claims is out of scope"), but nothing
     compensates for it.
   - Fix (generic, cheap): in `umpire_catalog_build::services::generate` or `coverage::check`,
     refuse any RPC whose request or response is mirrored (it is in the dispatch set) but is
     resolved `OutOfScope`. Today the mirrored set is exactly the eight claimed RPCs, so this needs
     no Temporal knowledge and turns both scenarios into build errors. At minimum, have DESIGN.md's
     API-update paragraph tell reviewers to check feature claims when they add a family arm.

3. **minor.** For the activity feature's own unclaimed RPCs, the out-of-scope reason now reads as a
   scope rather than a reason.
   - Where: `model/temporal/foundation/service.rs:37-45`, `:103-131`.
   - Scenario: annotating `RecordActivityTaskHeartbeat`, `PauseActivity`, `ListActivityExecutions`
     and 13 more now returns `out_of_scope: "Activity tasks and executions"`. Before, it returned
     "This activity operation is not represented by the mini activity lifecycle". Anyone reading a
     live or gap report sees a family that names the activity area and no explanation of why it was
     skipped.
   - This is within the spec ("reason derived from its family"), and the handoff lists it.
   - Fix: phrase reasons as families plus a feature-neutral reason, for example
     "Activity tasks and executions; not claimed by this feature", or have `annotate` format
     `"unclaimed <family> RPC"`. Either way, the per-family string stays the single source.

4. **nit.** `defers_to_table` accepts any path that ends in `family`.
   - Where: `model/umpire/catalog/syntax/src/services.rs:251`.
   - Scenario: `_ => RpcRole::OutOfScope(helpers::family(method))`, where `helpers.rs` (not scanned)
     returns different text, compiles and passes the scanner. The runtime `annotate` reason then
     differs from the RPC_BINDINGS reason.
   - Fix: require the path to be `family` or `service::family`, or accept that the table function is
     the only `family` over the method enum and say so in the error text.

5. **nit.** The wildcard error message hard-codes the parameter name `method`, but the check uses the
   real parameter name.
   - Where: `model/umpire/catalog/syntax/src/services.rs:275`.
   - Fix: format the message with `parameter.ident`.

6. **nit.** Some negative tests are missing or only assert `is_err()`.
   - Where: `model/umpire/catalog/syntax/src/services.rs:536-539`.
   - `complete classification duplicates the service table` has no test.
   - `feature claims without a service table` and `duplicate service table` are asserted only with
     `is_err()`. A regression that swapped those messages would pass.
   - `model/temporal/api/tests/build_inputs.rs:21` does not assert that `foundation/service.rs` is
     a tracked input. It is tracked today, but an edit that dropped `service.rs` from the
     `coverage.rs` file list would leave the table untracked until a later touch.
   - Fix: assert the exact messages, add a test for a complete classify beside a table, and assert
     `foundation/service.rs` in `build_inputs.rs`.

7. **nit.** The `ServiceTable` re-export is unused.
   - Where: `model/temporal/annotate/syntax/src/services.rs:4`. Nothing names the type. The api test
     reaches `tables[0].families` without it.
   - Fix: drop it unless a consumer is planned.

8. **nit (forward-looking, out of spec scope).** `resolve` merges every feature's claims into one
   `Classification`, so a single `bind_workflow_action!` match carries every feature's action
   patterns.
   - Where: `model/umpire/catalog/syntax/src/services.rs:85-118` together with
     `catalog-build/src/services.rs::emit_bindings`.
   - Scenario: a second feature's first `Action` claim (with its own action type) puts patterns of
     two types into one match, which is a type error. The new duplicate-claim test uses only an
     Observed `NexusAction` claim, so the collision path the spec's Goal mentions is still untested.
   - Fix: record this as a follow-up for the second feature: per-feature bind macros keyed by claim
     source, or one claim `Classification` per feature.

## Owner-rule check
- No Temporal names entered `model/umpire`.
- Authors write one extra line per feature: `_ => RpcRole::OutOfScope(service::family(method))`.
- Goldens: unchanged apart from reason text. No snapshot or `expected/` files changed. The only
  committed test-text change is the `GetSystemInfo` reason assertion in
  `realization/tests/catalog.rs`.
- No new dependencies.
- Comments are sparse and explain why. The one `#[allow]` carries a reason.
