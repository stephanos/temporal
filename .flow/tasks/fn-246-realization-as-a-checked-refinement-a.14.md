---
satisfies: [R6]
---
# fn-246-realization-as-a-checked-refinement-a.14 Heartbeat task key and raw/SDK continuation capability

## Description
Move fn-237.4's already-approved narrow heartbeat key/codec/raw-and-SDK continuation capability earlier for R6. This task builds that single executor contract before .5 derives strategies. fn-237.4 retains admission, actual worker journal/evidence/gates and every original mandatory live proof.

**Size:** M
**Files:** feature-owned activity task key/response codec and generated wire bindings, worker program admission, raw/SDK continuation and token-scoped completion controls.
**Touches:** [model/temporal/proto/umpire/temporal/v1/**, model/temporal/api/**, model/temporal/hosts/go/umpire/temporal/v1/**, model/temporal/hosts/go/reference/worker/**, model/temporal/hosts/go/*test.go, model/temporal/realization/tests/program.rs, model/temporal/realization/tests/program_history.rs, model/temporal/realization/tests/catalog.rs, model/temporal/feature/activity/claims.rs, common/testing/testpilot/temporal/act/**, tools/umpire/temporaldriver/**, model/umpire/hosts/go/runner/session.go, model/umpire/hosts/go/runner/*test.go, model/temporal/hosts/go/umpire_temporal_guest.wasm, model/temporal/feature/activity/temporal.rs]

### Approach

All source coordinates are provisional planning anchors. Reanchor final fn-237.3 fields, emitted graph, accessors and fn-247 xtask/build owners after their committed closures before implementation. The current fn-237.3 source checkpoint and fn-247 plan do not satisfy those gates.

1. Establish the earliest SDK proof before schema/binding/strategy integration relies on repeated-heartbeat capability. Add an actual pinned SDK worker/client transport control for repeated-false, false-to-true and after-true delivery, throttle/coalescing, real terminal RPC selection and local cancellation/Close at each wait. Drive actual RPC completions and associated SDK cancellation; a RecordHeartbeat API invocation is insufficient. Test any supported detached-context hypothesis while preserving SDK activity values and Run-owned cancellation/join, with no escaping background work. Source inspection alone proves no delivery capability. If a required choice fails, retain the exact per-executor conflict and block integration rather than use raw sends or refusal credit. No probe/build runs during planning.
2. Reanchor fn-237.4's original bounded continuation contract and current executor ownership. Move the key/codec/executor capability with the approved finite-history extension; root/plan-sync updates the later task's consumed prerequisite so no duplicate implementation or live-proof deletion results. Retain terminal-only semantic controls while changing the current feature schema; no old wire/hash/readability pin is required.
3. Add a bounded actual-observable-history key in the current format, retaining exact observed heartbeat bits/order and the original tri-state max-one-heartbeat semantic subset. Admit heartbeat/canceled responses through current codecs and migrate WorkerProgram/core bindings coherently if layouts change. Regenerate through existing descriptor/protobuf owners; old key/response layouts and old executors are not compatibility targets.
4. Preserve one response per exact row. Only actual correlated heartbeat completion advances the history; each admitted prefix may choose a next heartbeat or terminal within declared finite depth. Refuse duplicates/conflicting prefixes, missing continuations, exhausted depth/work, unsupported fields and unrepresentable per-executor programs without silently truncating finite policies. Never put hidden Model state/budgets, wall-clock inference, Model callbacks or a general sequence interpreter in Go. The original max-one-heartbeat witness remains a mandatory later subset with provenance.
5. Raw retains the exact task/token/attempt across every actual completion. Use only the supported SDK path actually established in step 1. Require token-scoped actual shared-transport completion and associated SDK cancellation delivery for true before canceled result. Verify actual terminal RPC selection after notification. Any supported context/lifetime adaptation must retain SDK values and executor-owned local shutdown/Close/join. Record per-executor representability; a genuinely missing required choice blocks R6. No raw RPC surrogate or refusal credit.
6. Reuse the existing transport seam or deepen it minimally for actual completion notification. The per-Driver mechanical capture owner remains act, with temporaldriver composing its callbacks into the nested worker bridge. Preserve Evidence() through the existing generic Runner Driver.Close fault path, matching Exchange.Close; a mechanical fault-only payload may use temporal.proto without adding ActivityEvidence, a journal, cuts or gates. Snapshot presence alone must not fault ordinary successful terminal work or expected empty-poll shutdown. Independent capture defaults are4096 invocations and1GiB encoded snapshots, preserving the pinned SDK128MiB message allowance; validate reservations/overrides before effects and fault explicitly without truncation or positive credit. Claim every newly consumed heartbeat request/response field, reusing existing kinds and independently justified constraints. If an additional mechanical bridge is necessary, coordinate exact ownership with the conductor before editing the later journal/gate surface. Transport failure/malformed/missing response, context cancellation and Close do not fabricate a bit or advance a key. Wake/join blocked waits and retain partial actual evidence through existing fault owners.
7. Supply server-free program/native/mock-transport tests now. Mandatory actual live cooperative witnesses remain fn-237.4; this task must not claim them from mocks. Existing terminal live rows remain in .5/.6's validation.

**Dispatch exclusion:** Production Rust key/program integration consumes .7’s capability/filler migration and is serialized with its active feature/activity/temporal.rs and program-test ownership. SDK-only proof is already integrated. Serialize with .2 on realization/tests/program.rs; the parent overlap guard applies even when the task DAG otherwise permits parallel work. Preserve existing dependencies.

### Investigation targets

**Required:**

- `.flow/tasks/fn-237-activity-timeouts-heartbeat-and.4.md` sections Bounded per-task executor continuation and Mandatory actual worker evidence path for the moved/retained boundary.
- `model/temporal/proto/umpire/temporal/v1/activity.proto` and `temporal.proto` for feature key and unchanged program envelope.
- `model/temporal/hosts/go/reference/worker/program.go:45-72` for current row admission/matching.
- `model/temporal/hosts/go/reference/worker/raw/raw.go`, `sdk/sdk.go` and pinned SDK v1.48.0 `internal/{internal_activity.go:397,internal_task_handlers.go:2182-2330,2586,2624}` for task ownership, native throttling, cancellation and actual final RPC.
- `common/testing/testpilot/temporal/act/act.go:282` for existing shared transport; later journal/gates stay fn-237.4 owned.
- `model/temporal/hosts/go/program_test.go` and `model/temporal/realization/tests/program.rs` for current terminal-response semantics, program round-trip and executor parity.

### Verification

Run existing Temporal protobuf generation/check scripts, Rust program codec tests and `go test -tags test_dep ./...` from model/temporal/hosts/go. Use the actual pinned SDK worker/client with mock transport to test exact repeated-false, false-to-true, after-true/canceled-context, throttle/coalescing and terminal-RPC behavior; counting API invocations is insufficient. Also run focused act/temporaldriver tests from the repository root and generic runner tests from model/umpire/hosts/go, retaining actual Driver.Close fault evidence controls. Regenerate the local Temporal guest when current codec/program tests require it; the conductor verifies/promotes the joined guest. Changed-Go fmt/lint and scoped Rust fmt/strict clippy are required. Full fresh guest/live gates belong .6; original actual cooperative proof stays later.

## Acceptance
- [ ] Terminal-only response semantics and initial unknown/false/true behavior remain tested in the current format. Current key/codec/program bindings regenerate coherently and pass freshness; old readability or unchanged envelope/ABI is unnecessary.
- [ ] Before dependent binding/strategy integration, the actual pinned SDK transport proof establishes repeated-false, false-to-true, after-true behavior, throttle/coalescing, real terminal RPC selection and Run-owned shutdown/join. Detached-context support remains a hypothesis until this control passes using supported APIs; a missing required SDK choice blocks integration without a raw surrogate.
- [ ] Exact bounded observable-history prefixes advance only from actual correlated heartbeat completion with retained bit order, one response per exact row and one terminal/no second poll on the same task/token/attempt. Raw and actual pinned SDK mock-transport paths distinguish repeated-false, false-to-true, after-true cancellation/throttle and real terminal RPCs; original one-heartbeat subset remains exact.
- [ ] Missing/duplicate/reordered/ambiguous row or response, conflicting history prefix, exhausted depth/work, crossed token/attempt, unsupported fields/program, transport error, delayed/coalesced SDK notification and cancellation/Close at every blocked wait refuse without fabricated evidence, false bit, truncated completeness or terminal success; all waiters join. Per-executor representability facts and every missing required choice remain explicit R6 blockers.
- [ ] fn-237.4 consumes the moved capability while retaining timeout/cooperative admission, journal/evidence/gates and every original raw/SDK live witness. Focused protobuf/Rust/Go/lint evidence is retained; mocks are never presented as live proof.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
