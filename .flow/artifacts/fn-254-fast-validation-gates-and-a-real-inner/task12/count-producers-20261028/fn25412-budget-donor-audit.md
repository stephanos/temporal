# Task12 budget donor selective-port audit

Current source `0b66c4f2f8b95546ad59e63ff0834de465a98bdc`. Read-only research against retained `4835a4734a` and `f7ad9c4e58`; no implementation, runtime or review verdict. Canonical task12 anchor was read without Flow calls. Shared count/applicability design remains owned by the separate count-design lane.

The parser/state-machine semantics are useful donors. The historical scheduler wiring is not ready to apply: current scoped receipts, leased supervision, combined graph and native task15 routes have materially changed. Keep one producer count schema and one mise graph.

## Current registry and table drift

Current metadata defines **18 framework +13 Temporal qualified identities**. Framework table leaf names match after qualification. Both donor Temporal tables have obsolete `framework` and miss `temporal-prerequisites`. Their pending entries are all uncalibrated and correctly refuse when enabled. The donor table test derives its expected registry from the table itself; it cannot prove agreement with current source.

A combined invocation must merge the two scoped tables against the actual current mise inventory. A standalone child selects17 stages while retaining the complete18-stage framework registration. Optional skipped standalone receipts and linked child authority need explicit current applicability semantics.

## Concrete defects and required adaptations

### D01 — Duplicate count schema must not be adopted

Donor defines another6field WorkCounts; current canonical producer type also carries linked child authorities and strict receipt validation. Shared applicability/provenance redesign is owned by count_design lane.

Required: Use one canonical producer schema/validated receipt boundary chosen by count_design; adapt bounds/equality to its applicability states, not copied Option fields or fabricated0.

### D02 — Registry and table identity mismatch

Current upstream graph exposes18 framework +13 Temporal scoped identities. Both donors use bare keys; Temporal table retains obsolete framework and misses temporal-prerequisites. Adapter chooses one table and model-only prefix, while combined graph expects both scopes.

Required: Load/qualify the2committed tables, validate exact merged namespace against actual mise listing, distinguish complete registry from selected subset. Keep full18 registered for child17selection. Replace tautological table-keys-as-inventory control with actual registration-derived controls.

### D03 — Retry executable and stage argument are wrong for current scoped graph

retry_command prefers UMPIRE_TEMPORAL_GATE_EXECUTABLE for every stage, including framework stages; passes qualified identity directly after --scheduled-stage and does not rebuild scope-specific task environment/cwd. Native dispatch accepts leaf stage names, not framework/tests.

Required: Resolve recorded scope to correct existing native executable, leaf name, scope, root/cwd and task environment; maintain qualified receipt/budget key. Test both scopes, same-name leaves and raw paths.

### D04 — Retry unleased spawn can reacquire the outer JVM lease

Adapter calls backend_process::run_owned for a public gate retry. Current unleased OwnedProcess preparation strips lease locator; public gate enters supervise while outer invocation still holds admission.

Required: Use existing scheduler leased supervision/owned-command custody contract. Never pass lease env without real descriptor delegation; add held-parent-lease retry completion control. Keep publicfmt outside public JVM admission.

### D05 — Wall retry is delayed until whole DAG completes and dependents have run

admit_stage calls Invocation.first then discards RetryAfterDrain; mise sees success and releases dependents. Adapter loops stages only after first entire graph returns0. This is not immediate slow-stage retry before downstream release.

Required: Decide stage-admission retry/drain handshake within existing scheduler/mise graph. Pause release/new competing starts as needed, drain only owned competitors, retry same leaf once, then resume existing DAG. Do not create a second scheduler or claim end-of-DAG retry is immediate. Preserve correct hard-count failure-before-success semantics.

### D06 — Historical receipt/context hooks do not match current scoped atomic receipts

Current receipts have scope/state/reason/error/count_authority/child_count_receipts; no attempt/context fields. Old work_counts::context/freeze methods are gone. Current scoped leaf directory differs from invocation root, so naïvely calling donor admit_stage there would miss budget.toml and silently bypass it. Current collect also treats unknown metadata/subdirs as errors.

Required: Add minimal attempt/context budget metadata while preserving every current validation/atomic claim/write/link rule. Pass explicit invocation root separately from scoped receipt dir. Extend only dedicated metadata allowlist, validate retry subtree, never relax duplicate/partial/child authority checks.

### D07 — Default opt-in is preparation, not required normal-gate enforcement

enabled(None or0) returnsfalse; budgets need caller UMPIRE_GATE_BUDGET_INPUTS and optional override. No complete source/tool inventory producer is established; comments acknowledge arbitrary JSON cannot prove completeness.

Required: Final required gates must derive admitted table/context/inventory normally, fail unknown counts/pending stages, and avoid a user env bypass. Derive semantic source/tool/mode/tier closure from existing admitted roots/tool providers, not trust incomplete caller lists. Preserve --list and independent publicfmt behavior.

### D08 — Retry failure loses public status and combined failed-attempt summary

Adapter converts any nonzero rerun status to ioError (top dispatcher becomes1) rather than preserving37/143. budget-attempts/report are published only after every retry succeeds, so second-slow/interrupted paths leave raw records but no combined tier report.

Required: Persist each original/retry receipt plus outcome and elapsed contribution before admission returns; publish failure totals as well as success totals. Preserve actual failed child code/signal and observation error context; never turn refusal into timing warning.

### D09 — Donor count fixtures violate current test-count invariants

controls::counts(v) sets discovered=executed=ignored=skipped=v; for nonzero v discovered differs from executed+ignored+skipped. Module accepts these only because it bypasses current producer validation.

Required: Adapt all fixtures to valid canonical counts/applicability and validate through current receipt boundary. Retain overrun/unknown precedence tests; do not weaken producer invariant checks to accept donor synthetic data.

### D10 — Adapter covers only scheduler graph routes

Budget wrapper only surrounds scheduler::run. Existing quick phases use quick::execute_with and publicupdate/directstage execution do not traverse full scheduler run. Tier keyquick in JSON does not instrument actualquick.

Required: Explicitly settle per-entry enforcement coverage and reuse existing quick/native phase seams where required. Do not claim all current stages or quick tier are enforced solely from the full-graph wrapper; do not invent a parallel graph.

## Port sequence

1. Wait for count_design decision; import canonical WorkCounts/applicability/child authority validation.
2. Register exact qualified18framework/13Temporal tables from current mise graph; derive fullregistry/selectedsubset separately. Replace obsolete Temporalframework withtemporal-prerequisites; include skip/standalonechild identity semantics.
3. Admit complete source/tool/backendmode/tier/table context automatically for ordinary gate routes before selected work; keep declaration/hash checks but attach trusted closure producer.
4. Integrate attempt/context with current scoped atomic scheduled receipt publisher and strict collector, preserving counts/state/reason/errors and child raw receipt authority.
5. Resolve immediate wall-only retry/drain sequencing before dependents release using existing stageadmission and ownedsupervision; correct scope-to-native route/cwd and inheritedlease.
6. Persist/report first+second attempts even on failure; use one outer monotonic invocation wallclock for tier total; inclusive standalonechild per-stage sums remain diagnostics and child budgets validate under own linked authority.
7. Fill actual max bounds only from admitted optimized producers and explicit mode/feature/source pins, after unknown modalities resolved. Pending scaffold may remain research material but cannot pass admission. Preserve deferred fulltiming/task8 obligations.
8. Keep public fmt bypassing JVM admission; graph framework/fmt/temporal/fmt remain registered budget stages under their parent invocation. Add integration controls before required enforcement.

## Semantics worth retaining

Count/unknown/execution refusal precedes wall-time consideration. Exactly125% is accepted; only a successful count-admitted first slow attempt may retry. Retry is consumed before draining, even if drain or execution fails. Second slow/incomparable/interrupted work refuses. Keep reviewed table reasons, exact table-byte identity, complete registry equality and immutable invocation/source/tool/tier identity.

The donor per-stage attempt sum is diagnostic; its outer monotonic clock avoids summing overlapping and inclusive child elapsed twice. Preserve that distinction. Extend durable attempt/tier reporting to failure paths; never add nested child stage totals to the already-inclusive standalone parent. Child authority stays independently validated, and fresh invocation paths/timestamps are not semantic count identity.

## Normal enforcement and fmt admission

Default `UMPIRE_GATE_BUDGETS` off and caller-supplied input JSON are explicitly unvalidated preparation. Final ordinary required gates must obtain a complete admitted context automatically and enforce actual bounds, with no hiddenoptout. Do not pass candidatepending tables, use estimates as actualcounts or replace unknown with0. Scope quick/update/direct native routes explicitly: the donor graph adapter alone does not instrument them.

Public `fmt` currently bypasses the public JVM-supervision branch in both dispatchers. Keep that property and its held-host-lease control. Framework/Temporal fmt inside a full graph remain ordinary registered stages under the parent invocation; adding budget support must not force a standalone formatting command to wait for the JVM lease.

## Controls to retain and extend

- `hard_count_refusal_never_requests_timing_retry`
- `missing_counts_and_missing_identity_are_not_zero`
- `unknown_omitted_and_duplicate_registry_entries_refuse`
- `tolerance_boundary_is_inclusive_and_fast_stages_cannot_retry`
- `second_fast_retains_first_slow_and_runs_after_drain`
- `second_slow_is_a_failure_and_cannot_retry_again`
- `interrupted_or_failed_retry_cannot_be_relabelled_or_repeated`
- `failed_drain_prevents_rerun_and_consumes_retry`
- `crossed_invocations_and_changed_work_are_incomparable`

Donor synthetic counts need correction to satisfy actual producer invariants before these tests are reused. Replace the table-self-inventory test and opt-in-bypass assumptions with current graph/normal-entry integration controls.

- Current scoped registry exactly18+13, childselected17/fullregistry18, unknown/omitted/duplicate stage; correcttwo-table merge and currenttemporal-prerequisite.
- Both scoped native retry routes preserve leafname/cwd/context/backendflags/rawOSpaths; framework retry never goes toTemporalentry.
- Retry while parenthostleaseheld finishes via same leasedcustody; cancellation kills/reaps owned retry descendants; no inherited descriptor spoofing.
- Hardcount overrun/unknown/inapplicable-corruption/backend37/signal143 refuses before mise dependents; no retry.
- First wallslow blocks dependency release, ownedcompetitorsdrain, same leaf retried once immediately; interrupteddrain cannot resume/admit.
- Firstslow/secondfast reports bothattempts; secondslow/refusal/interruption persists both raw/combined attempts and actualpubliccode; tierclockincludesdrain/retry once.
- Scope/context/attempt tampering, partial budget metadata, duplicate retryclaims, unlinkedchild receipts and unexpectedsubdirs fail without weakening currentcollector.
- Shared child authority validated separately; parentinclusive elapsed/owncounts never add child totals; fresh child invocationpaths do not masquerade as changedsemanticwork.
- Normal publiccheck/validate (and scopechosen quick/update/direct routes) enforce without optinenv; --list staysread-only; publicfmt underheldhostlease remainsfast.
- Frozen source/tool/lock/taskconfig/provider modes/table/tier identity changed betweenattempts refuses; atomically replaced controller/runtimebinary handling follows existingfrozen identity contract.

## Unsettled design boundaries

- Immediate drain protocol: Current supervisor drop drains invocation by cancellation/owned completion, not a per-stage pause barrier. Choose native stageadmission handshake that blocks new competing leaves without replacing mise DAG or killing/reschedulingalreadycompleted work.
- Shared count/applicability provenance: Canonical schema and proof for0/notapplicable/unknown; donorOptionfields are not that design.
- Tier elapsed start/exclusions: Donor starts clock inside graph adapter after earlier bootstrap/provider/listing work. Choose/document outer invocation start and existing caller-preparation exclusions explicitly; include owned retries/drain once and no inclusivechilddoublecharge.
- Non-graph native routes: Quick/update/directstages use existing differentexecution seams; decide stated enforcement coverage from canonical task/spec, not justtierlabels.

Exact source/donor symbol references and hashes, all31 current stage identities, table diffs,10 concrete candidate defects and selectiveport steps are in `fn25412-budget-donor-audit.json`. No source/Git/Flow/runtime operation occurred. Migration remains deferred; no external checkout or alternategraph was used.
