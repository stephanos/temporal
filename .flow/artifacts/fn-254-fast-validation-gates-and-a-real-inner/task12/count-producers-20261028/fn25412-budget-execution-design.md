# fn-254.12 budget execution design

Recommendation for the root's next implementation phase, grounded in source HEAD `990aa82b624232d0ce55730456a8b638f42b1aef` plus the current producer changes. Keep published numeric/null counts, add three small evidence records, and decide retry inside the existing native stage callback. Use an invocation-local admission barrier around payloads; mise continues to own every dependency and eligible task. The report proposes APIs, not implemented or validated behavior.

## 1. Canonical evidence API

Leave `WorkCounts`' six `Option<u64>` fields and numeric/null serialization unchanged. Add receipt-level metadata with no duplicate numbers:

```rust
struct CountEvidence {
    version: u8,                    // 1
    tests: MetricEvidence,          // discovered/executed/ignored/skipped atomically
    tlc_launches: MetricEvidence,
    tlc_configurations: MetricEvidence,
}
enum MetricEvidence {
    Observed { authority: AuthorityRef },
    NotApplicable { authority: AuthorityRef, reason: NoWorkReason },
    Unknown { reason: String },
}
struct AuthorityRef { id: String, sha256: String }
enum NoWorkReason {
    NativeScan, CompileOnly, Formatting, ProtobufGeneration,
    GuestBuild, ToolProvisioning, ChildExclusive, SkippedByMode,
}
```

Serialize the evidence with a closed tagged status and deny unknown fields. `AuthorityRef` names bounded invocation-local producer or completed dispatch evidence. It contains no count copy. Use one canonical definition in `gate/work_counts.rs`; budget code imports it and `WorkCounts`.

Keep `validate_receipt(&Value)` accepting historical receipts without `count_evidence`. If evidence is present, validate it strictly. Add `validate_budget_receipt(&Value, &AdmittedContext)` requiring evidence and admitted context, including recursively admitted child receipts. Legacy numeric receipts remain readable but cannot silently gain budget eligibility. Do not change old v1 `tlc-finish0` semantics.

Add `collect_evidenced(directory, selection, completed_dispatch) -> io::Result<ObservedWork>` alongside current `collect(directory) -> WorkCounts`, where `ObservedWork` contains the canonical counts, evidence and linked authorities. Internally share the same parser. Missing evidence defaults to unknown; only the new path has authority to certify N/A or observed zero. The existing empty-directory control still returns unknown.

Validation rules are mechanical. Observed tests require all four numeric counters and the existing sum invariant. Observed TLC fields require numeric values and their producer authority. N/A requires null for its modality, a matched successful native dispatch closure and an allowed typed reason. Unknown remains null and refuses budget admission where applicable. Configurations may be unknown while launches are numeric. A counted event in a modality forbidden by its dispatch contract is a contract error. A table cannot authorize N/A, and changing null into an enum object is explicitly excluded by the root's wire-compatibility decision.

## 2. Exact native dispatch hook

Change the scheduled callback from `FnOnce` to a repeatable native callback with an explicit attempt context:

```rust
pub fn scheduled(
    name: &str,
    run: impl FnMut(&mut StageAttempt) -> io::Result<i32>,
) -> io::Result<i32>;

impl StageAttempt {
    pub fn run(
        &mut self,
        contract: DispatchContract,
        body: impl FnOnce() -> io::Result<i32>,
    ) -> io::Result<i32>;
}
struct DispatchContract {
    tests: TestExpectation,       // RequiredProducer or NoWork(reason)
    tlc: TlcExpectation,          // CompleteObserver or NoWork(reason)
    isolation: PayloadIsolation,  // Shared or ExclusiveChildGraph
}
```

`StageAttempt::run` admits exactly one declared branch before any payload side effect, obtains its payload permit, records a dispatch-open authority, runs the body, and retains the permit through owned cleanup/count finalization. The scheduler closes the dispatch and atomically releases the permit while deciding fast/slow/failure. Calling it twice or executing no declared branch refuses. Branch helpers can execute multiple commands inside that one body, as current timing controls do.

Attach this at framework `Gate::stage` (`gate.rs:171`) and Temporal `run_stage_with_tool` (`model/dev/xtask/src/gate.rs:240`). The actual match arm supplies the contract immediately around the same body/command it currently executes. Do not create another `stage_name -> applicability` table or alter mise metadata.

The resulting branch matrix is:

| Actual branch | Tests | TLC |
| --- | --- | --- |
| Framework timing/capability/protobuf controls and tests | Required existing Rust producers | Complete observer scope |
| Temporal timing/protobuf controls, source-boundary and tests | Required existing Rust producers | Complete observer scope |
| Three Go-host branches | Required Go producer | Complete observer scope with admitted source/capability coverage |
| Framework source-boundary, both retired-name scans | Native scan N/A | Native scan N/A |
| fmt, clippy, docs, wasm compiler checks | Formatting/compile N/A | Same completed native contract N/A |
| protobuf freshness, proto tools, guest build/freshness, Temporal prerequisites | Completed concrete generation/build/tool contract N/A | Same contract N/A, unless actual counted activity contradicts it |
| Standalone own orchestration | ChildExclusive N/A after setup contract closes | ChildExclusive N/A after own setup contract closes |
| Mode-skipped standalone | SkippedByMode N/A | SkippedByMode N/A |

The matrix is design analysis, not a new runtime registry. The contract constructors live in the existing executable match arms. An unfamiliar helper defaults to unknown until its body is covered. The preserved task15 child receipt proves that controls carry actual Rust counts; framework source-boundary has none. Temporal source-boundary actually dispatches selected tests and cannot use the framework scan contract.

## 3. Positive zero for an actual test observer

The missing signal belongs in `work_counts::observe_streamed` (`work_counts.rs:618`) and its Go branch, not in the generic collector or a test-name exemption. Add an owned `TlcObservationScope` around the actual admitted test dispatch:

```rust
TlcObservationScope::begin(test_producer_id, coverage_identity) -> ScopeToken;
ScopeToken::complete(completed_test_authority, owned_completion) -> AuthorityRef;
```

Begin after selecting the actual observer and admitting its exact command/inventory, before dispatch. Complete only after the runner has returned, all owned descendants are drained, the complete nextest/doctest/Go stream validates, and `tests-finish` publishes. The token binds invocation, stage, attempt event set, producer ID, observer protocol version, source/tool/runner identity and backend mode. It must not auto-close successfully in Drop. Missing finish, parse failure, missing/partial close, duplicate/cross-scope token or outstanding TLC attempt leaves unknown or refuses.

`coverage_identity` attests the current admitted source's TLC execution sites use the active process producer; it is not a claim inferred from the absence of events. The Rust adapter already has typed/unclassified process events and trusted source/tool closure. Go tests currently have no Java/TLC command sites in the inspected host/reference packages; a completed Go observer can certify zero only under an explicit admitted checker-free capability/source contract, or future equivalent instrumented coverage. Do not grant Go coverage merely because `go test` returned success. This is the remaining authority that the currently correct Go receipts do not yet contain.

At collection, require complete scope tokens for every actual test producer in a `CompleteObserver` dispatch. With that positive coverage and no TLC attempts, emit observed launch/configuration zero. With attempts, retain the existing real/fixture/unknown aggregation. A scope close cannot erase a v1 finish0, an unmatched attempt or corrupt fixture authority. Scope tokens refer to existing test and TLC producer records, avoiding a second count stream.

## 4. Standalone own work and nested coordination

The current parent does not launch `tlc2.TLC`. `standalone::tool_versions` at line317 executes Java `-version`; `fetch_apalache::install_tlc` provisions and hashes a jar, and `tool_provider::{install,resolve}` run mise install/where. Generic JVM version probing is not TLC startup. Real probes belong to child test receipts. Root accepted this source correction.

`standalone::clean_command` at line164 strips observation scope. Keep child gate receipts independent. Add explicit parent-completed setup dispatch authority for snapshot/controls/tool versions/build/provisioning, then link the already exclusive child stage authority. No parent TLC probe is fabricated. Parent own counts may be N/A once that concrete setup contract completes; own elapsed includes every setup/version/build/provisioning/child step. If a future parent setup launches TLC, it needs a dedicated parent observation scope instead of inherited child attribution.

Use `PayloadIsolation::ExclusiveChildGraph` for the actual standalone branch. It queues an exclusive admission request, drains other outer payloads before entering, and blocks outer starts until the entire existing child graph finishes. The child graph uses its own local barrier and existing lease custody. This is the smallest safe nested design.

Why choose it? Treating standalone as a normal active callback makes a child retry wait for its parent, which is waiting for the child. Independent barriers without exclusivity leave outer Temporal payloads competing with the child's supposedly drained retry. A shared participant tree with suspended ancestor callbacks could preserve overlap but adds ancestry and delegation state. Exclusivity needs one extra permit mode and no new graph. It reduces existing standalone/Temporal overlap and therefore requires explicit calibration; do not claim unchanged wall-time cost.

## 5. Immediate retry and coordinator ownership

Keep one small coordinator per actual graph invocation, under `budget/`, initialized by `scheduler::run` before the existing `supervised` call. Existing callbacks find this immutable admitted context through an invocation-local locator. The outer `JvmLease` remains held and delegated as today. The callback reruns its own `FnMut` body in place; it does not invoke the public gate CLI, acquire the host lease, launch another mise graph, or resolve a retry executable by scope.

Use a stable `File` lock with `try_lock` and bounded polling, already available in the pinned Rust/std and used by `JvmLease`. Hold it only to read/validate/update bounded coordinator state. Never hold it during work, drain waiting or a retry. Replace the state document atomically beneath the stable lock file. Bound participants by selected stage inventory and requests by one retry per stage. Fresh invocation directories mean there is no stale recovery by PID guesses.

Minimal state is:

```rust
struct CoordinatorState {
    phase: Phase,                    // Open, Paused, Exclusive(owner), Aborted(cause)
    active_payloads: BTreeSet<StageId>,
    pending: Vec<Request>,           // ordered while holding the file lock
    consumed_retries: BTreeSet<StageId>,
}
enum Request { Retry(StageId), ExclusiveFirst(StageId) }
```

`StageId` includes invocation and qualified `framework/foo` or `temporal/foo`, plus the attempt where needed. Queued mise callback processes do not enter `active_payloads`. Eligibility still comes entirely from mise. A callback may wait at this admission barrier while retaining its mise slot; the drain predicate ignores those waiting slots.

State transitions are:

1. A declared ordinary branch enters only in Open and atomically registers active, then executes its original body.
2. After cleanup and count/evidence admission, a fast callback atomically removes itself and returns its final result. A hard execution/count/context error removes itself, sets Aborted with its original cause, publishes failure and returns failure. It never requests timing retry.
3. A first successful count-admitted slow callback atomically removes itself, consumes its retry and appends Retry while closing admission. Its callback remains pending, so mise cannot release that stage's dependents.
4. Existing active payloads finish normally. They may return fast or append their own slow requests while removing themselves. No new payload enters. Waiting callbacks are not drain targets.
5. When `active_payloads` is empty, the head owner becomes Exclusive and verifies the frozen semantic input context. It runs the same native branch once, inside the same process and lease lineage.
6. A fast, count-admitted retry publishes attempt2 and its final stage receipt, then hands the barrier to the next pending owner, or reopens admission. A second slow, refused, interrupted, incomparable or failed retry aborts. A failed drain consumes the retry and launches nothing.

For two simultaneous slows, whichever state update acquires the lock first owns the first request. The other finishes its first payload, leaves active and joins pending. Neither waits while counted active. After owner1 passes, owner2 retries before admission reopens. An already completed fast competitor may return to mise; its newly released dependent merely waits at the existing barrier.

Drain uses a finite reviewed policy cap from the authoritative table, not an environment default or an infinite wait. Cancellation, owner death or invalid/partial state aborts; no active entry is cleared solely because a PID appears absent. The outer supervisor already owns/reaps the callback tree. Extend its poll loop (`scheduler.rs:332`) to recognize coordinator abort/cancellation and preserve the original failing stage's code. A queued callback must not manufacture a child exit1 that masks the original37/143; record it as not-run/aborted and let the coordinator's original cause determine the public failure.

## 6. Attempts, exact replay and accounting

Do not mutate process environment to change per-attempt observation scope. Rust2024 environment mutation would introduce an unsafe/global synchronization problem, and current direct producer calls use the process scope. Keep the existing one stage work directory and all unique event files.

At each attempt boundary, capture the exact newly published event set, after owned cleanup, and write a bounded immutable manifest under `budget/attempts/<qualified-stage>/<ordinal>/`. Add `collect_selected` sharing the current parser, using that exact manifest. Validate manifests are disjoint, exhaustive for the final stage event set and digest-matched; an orphan/partial claim refuses. This avoids clearing evidence, reusing claims, setting mutable `UMPIRE_GATE_ATTEMPT`, or counting both passes as one attempt. All work remains invocation-local.

Preserve the current `<scope>/<stage>.json` as the one final stage receipt. Add references to the per-attempt receipts and evidence. Its canonical own `counts` come from collecting all completed own work, including retries; tests sum actual repeated inventories/executions, and TLC configurations deduplicate exact source/config identities across attempts through the existing collector. Per-attempt budget checks use each attempt's canonical `WorkCounts`, not the aggregate as if it were a single run. Cap retries at one, record the aggregate actual work, and retain every linked child invocation if standalone itself retries. Do not claim a sum of per-attempt configuration totals is globally unique.

Each attempt's payload timer includes its execution, owned cleanup and observation finalization. Queue/drain time is recorded separately. Final stage `seconds` remains inclusive callback elapsed, including waiting, both attempts and drain. The outer invocation monotonic timer includes all retry/drain costs once; overlapping callback durations and inclusive standalone child durations are diagnostic and never summed into tier wall time. Failure paths publish attempt and tier reports too.

Capture `BudgetContext` at the earliest native entry, before any Cargo inventory/build can relink the target alias. Pin/open the running controller image there and retain its digest/handle or an admitted immutable image identity. Do not later hash mutable `target/debug/xtask` and mistake a Cargo relink for changed running code. FnMut retry stays in that image. Native child dispatch should use an admitted invocation-local retained executable image, or equivalent image custody, when its mutable alias can change. Preserve role freezing in both mains. Bootstrap a second scoped native executable before graph launch, pin its admitted image, and bind both identities. The existing guest driver locates sibling binaries from its executable path, so placing retained images must preserve the admitted sibling/tool layout or pass those locators explicitly.

Snapshot semantic source content, manifests/locks/task configuration, selected workload/inventory, tools/components/provider/backend modes, admitted table bytes and tier. Recheck before retry. Source bytes and the actual running/admitted tool image matter; artifact directories, receipt nonces, scratch roots and copied-root absolute names are nonsemantic. Rebuilt outputs may change without changing source identity. Compare normalized work/count/selection identities and fixture renderer/input identity, not fresh authority paths or process IDs.

## 7. Normal enforcement and the single table

Load admitted tables automatically for public gate/check/validate/time routes. `Gate::full_inner` (`gate.rs:330`) and combined Temporal `gate::run` (`model/dev/xtask/src/gate.rs:121`) create the admitted context before launching mise. `scheduler::run` receives it explicitly. Caller-provided JSON and opt-in `UMPIRE_GATE_BUDGETS` are not execution authority. No hidden environment optout can admit missing counts or pending bounds.

Merge the two existing authoritative table files using their scope. Validate exact equality against actual mise inventory, all18 framework plus13 Temporal identities in a combined graph. A framework child has complete registry18 and selected subset17. Keep mode-skipped standalone registered with its explicit skipped receipt. Do not derive expected stages from table keys. Direct `gate --stage` can create one selected-stage context against that same complete registry and use the same native callback wrapper; `--list` remains read-only.

Public standalone formatting keeps its existing no-JVM-lease path. Graph fmt is budgeted under its graph invocation. Updating snapshots is a mutating maintenance operation, so do not silently add timing retry there. If budget enforcement is wanted for update, use a separately declared maintenance workload with retry disabled. It cannot masquerade as validation.

Quick already has sequential `execute_with` phases at `quick/mod.rs:229` and test callbacks at line310. Reuse the same attempt/count admission machinery there, with one active payload and a frozen selected-package manifest; do not route it through a new mise graph. Its actual fmt/clippy/test phases consume explicit quick workload profiles in the same table. If these hooks are not included in this implementation, label quick enforcement incomplete; a table tier label alone supplies no coverage. Current fn254 R6 execution targets are60/180/600 seconds for quick/check/validate; old30/120 proposals must not override current spec text. Final broad measurement remains .8.

Start the native tier clock before admission/tool/provider preparation in each public native gate route, so currently in-function bootstrap and setup count. Shell/native compilation before the process starts cannot be reconstructed by `scheduler::run`; retain the existing outer Make/time measurement for end-to-end reporting and state that boundary. Do not report a graph-only timer as the entire Make invocation.

## 8. fn255 bounds without phantom stages or circular admission

Use one additional closed `workloads` section in `model/umpire/dev/gate/budgets.toml`, parsed by the same budget module. Keep `stages` exclusively for actual mise stage identities, so proposed commands do not break exact current31-stage registration. A workload profile names its owning task, consumer kind, mode, selected manifest identity, canonical count maxima/applicability, finite time/output/resource caps and typed workload-specific limits. The six standard metrics remain canonical; Model states, emitted edges, corpus bytes and mutants are different units, not extra test counters.

Use typed consumer views over that table, conceptually `Table::admit_workload(key, kind, manifest) -> AdmittedWorkload`, with closed kinds Generated, DecoderReplay, DecoderFuzz, Mutation, MiriPositive, MiriControl and Coverage. Each view requires its task-specific fields. No second constants table, environment allocation or fallback maxima belongs in a consumer. Direct commands use the existing supervisor and this same authority; .7 later attaches them to actual tiers/stages and updates actual registry entries then.

Distinguish `status = proposed` from `status = admitted`. Store fn255.1's measured generated limits as proposed, explicitly including36states/144edges/depth35/twostarts/290assignments perModel,580 perbatch, defaultoneModel, requiredtwoModel Verified batch and threeshrink candidates. Its measured discovery+twoModel JVMs is a proposal for that exact Verified selection, not a generic launch allowance. Refuted cases need reviewed additional claim-launch bounds before execution. Proposed profiles are parseable planning data and always refuse execution.

There is no circular dependency. Source preparation remains permitted. After committed .12 closure, the owner/conductor reviews a finite task-specific profile, commits its admitted bounds and exact selection before that task's first evidence run, then clears the task's cross-spec block. Time/work bounds can be conservatively reviewed from existing measurements with explicit provenance; they do not need a secretly unbounded calibration run. Reducing selection to fit admitted bounds is allowed. The parser must never turn proposals into admitted maxima automatically. .2-.6 remain incomplete until their own real producers, over-cap controls and task-specific admitted profiles exist.

## 9. Discriminating controls and unresolved observations

Required controls extend current focused tests, without a new full-gate campaign:

- Historical six-field receipts remain readable; missing new evidence refuses budget admission. Explicit N/A with any positive event, unknown relabelled N/A, malformed authority, partial test quartet and cross-scope evidence fail.
- Every current actual branch emits its contract. Derive inventory from real mise metadata and verify18+13, child17selected/18registered. Controls/source-boundary distinctions match retained source receipts.
- Complete observer with no TLC calls yields0; missing open/close/producer, failed parser, fake scope close, active/unmatched TLC attempt and retained finish0 refuse certainty. Go coverage requires its explicit source/capability authority.
- Two first slows race while a fast competitor drains; queued callbacks occupy mise slots but cannot deadlock the drain. Firstslow/secondfast releases its dependent only after attempt2; secondslow/37/143/countunknown aborts before that dependent starts.
- Owner death/cancellation/partial coordinator state/failed drain consumes retry, leaves complete incomplete-work evidence and no orphan process. A second slow request never waits while active. Unrelated invocation coordinators cannot pause each other.
- Parent lease remains held across first/drain/retry; no public CLI reacquisition. Standalone exclusive entry waits/drains correctly; nested child retry cannot wait on an active ancestor or compete with outer Temporal work.
- Same running controller survives replacement of the mutable Cargo alias without false context mismatch. A genuinely changed source/tool/table/mode/selection rejects retry. Paths/nonces change without changing semantic comparison.
- First/retry event manifests retain every producer exactly once. Final counts include both attempts; configuration union and child authority remain correct. Failure reports retain original exit plus both attempts/drain; tier elapsed includes them once.
- Default check/validate enforce without opt-in. `--list`, public fmt under a held lease and maintenance update preserve intended behavior. Quick's real execution seam, if enabled, has an exercised profile rather than a label.
- Proposed fn255 profiles refuse execution; admitted typed profiles reject missing fields, excessive case/corpus/mutant/input/JVM work and phantom stage insertion. No unimplemented command appears as an actual registered stage.

Remaining observable decisions before implementation are the exact admitted drain bound; whether root accepts exclusive standalone's reduced overlap; the Go source/capability coverage authority; the existing role/image locator treatment for retained executables and guest-driver siblings; and whether quick hooks are included now. These are finite API/validation choices, not permission to replace unknowns or defer required normal gate enforcement. Root owns these decisions. No source, Git, Flow, goal, build, runtime, review or remote mutation occurred in this lane.
