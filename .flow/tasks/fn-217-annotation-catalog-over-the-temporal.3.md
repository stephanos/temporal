---
satisfies: [R3, R5]
---
# fn-217-annotation-catalog-over-the-temporal.3 Mini activity catalog with expected errors; fill and validate in the slice test

## Description
Implement the approved complete typed activity catalog in
model/temporal/features/activity/api.rs using .2's macro. Keep all semantic
rules, generation and validation in Rust. Every request constraint/presence rule
has an expectation tied to the actual qualified server validator. Replace/move
the .2 WorkflowService bootstrap classifications rather than duplicate them.

Validate the actual SDK transport, not only reconstructed callback/reply envelopes.
The current slice rebuilds PollActivityTaskQueueResponse from activity.Info, and
its model Complete/Fail replies are partial requests converted back to SDK results.
The SDK adds token, namespace, identity and deployment fields afterward. These
partial envelopes are not proof that actual wire requests/responses were checked.

Use a slice-owned SDK client with a unary transport interceptor (the existing
TestEnv.SdkClient has no dial-option hook), and route explicit Describe/Cancel
through that client's WorkflowService too. Mechanically marshal actual protobufs,
call a thin typed guest/Go-host route for Rust fill/validate, apply filled request
bytes before transport, and validate successful responses without repair. Record
RPC errors separately; an allocated reply on error is not a successful response.
Accumulate failures from background goroutines and assert them after worker/client
shutdown, so retries and worker logs cannot swallow annotation failures. Never
call testing.FailNow from a worker/poller goroutine.

Cover StartActivityExecution, DescribeActivityExecution, RequestCancelActivityExecution,
PollActivityTaskQueue, RespondActivityTaskCompleted/Failed and PollActivityExecution
(used by handle.Get), including cancellation reporting if emitted. Record actual
transport coverage in the live test. Account explicitly in Rust for conditional
SDK infrastructure RPCs (GetSystemInfo, DescribeNamespace, ShutdownWorker,
RecordWorkerHeartbeat, Nexus control traffic). No silent Go bypass/allow-list:
unknown traffic refuses, and any intentional OutOfScope classification has a
concrete catalog reason and appears honestly in coverage evidence. Do not claim
opaque or out-of-scope contents were semantically validated.

Annotate the consumed nested fields/oneofs in ActivityExecutionInfo, Payload(s),
Failure/ApplicationFailureInfo and ActivityExecutionOutcome, plus meaningful Start
options. Explicitly ignored parent subtrees retain a reason and limited scope;
do not hide fields actually read by the Model/SDK behind an ignored aggregate.
Empty successful worker polls are valid. Keep synthetic callback/partial reply
validation appropriate to that layer; full wire rules apply after SDK completion.

Preserve externally supplied namespace, activity/queue/run IDs and Model-decided
inputs. Rust fills absent request IDs; already filled requests remain stable
across retries. Go supplies bytes/seed/direction/method, not duplicate rules.
Response violations are server findings and are reported/refused, never filled.
Keep annotation dispatch independent of Judge's frozen state, and preserve Model
state, transactional allocation and existing ABI exports. Reserve Method 6 for
fn-228 Propose; any additive typed protocol operation uses numbers above 6 and
retains existing field/enum numbers. A canceled poll context must not poison the
Wasm host during cleanup validation; use a bounded independent validation context.

Restate actual validators, not assumptions inferred from kinds. Start/Cancel
request_id is length-checked and accepts non-UUID text: UUID intent can honestly
use Expectation::Accepts (which the engine classifies Confirmed on success), not
an invented rejection or fake accepted-gap waiver. Optional Describe/Poll run IDs
really are UUID-checked. Go len(string) is a byte count: never represent a byte
limit as Unicode string.max_len. Extend native generic constraints narrowly when
needed, with matching macro admission and real tests. Timeout zero/nil values and
normalization, retry MaximumAttempts=1 bypass, and successful RPCs that record a
payload-size failure need the actual server expectation, not assumed strictness.

Audit full-envelope diagnostics: task tokens and other Secret contents must not
enter logs, violations or reports. Redaction is for copies/reports, not deletion
of the valid token before transport. Keep independent Models and existing Model,
lift, coverage and wire goldens unchanged. Task .4 owns live gap/reverse coverage.

**Touches:** `model/temporal/features/activity/**`, `model/temporal/guest/**`, `model/temporal/realization/**`, `model/temporal/api/**`, `model/temporal/annotate/**`, `model/temporal/proto/**`, `model/temporal/hosts/go/**`, `model/umpire/annotations/**`, `model/umpire/wasm/**`, `model/umpire/proto/**`, `model/umpire/hosts/go/**`, `tests/testpilot_model_slice_test.go`, `tests/testpilot_model_annotations_test.go`, `model/Cargo.toml`, `model/Cargo.lock`
**Quick:** focused Rust catalog/route/refusal tests and Go transport controls; `make umpire-check-model-slice`.
**Full gates:** `make model-check`; `make model-check UMPIRE_BACKENDS=require`; `make model-time` with actual timing.
Use a lane-specific Cargo target. Reuse existing transport/protobuf helpers and
admitted dependencies. API-independent Model code must gain no Temporal import.

## Acceptance
- [ ] Real typed message/enum/action/service catalogs cover all required activity messages and actual observed slice traffic; nested consumed fields are not hidden.
- [ ] Every declared request constraint/rule has a faithful qualified-validator expectation; compile-time admission remains enforced.
- [ ] Actual SDK RPC attempts pass through Rust fill/validate before transport and successful-response validation afterward; no duplicate Go rule table or synthetic-only substitute.
- [ ] Coverage proves which RPCs were observed; unknown RPCs and annotation failures cannot be silently bypassed by retries/background goroutines.
- [ ] Missing IDs are filled, supplied values and retry IDs remain stable, empty polls work, response violations are retained without repair, and secrets stay out of diagnostics.
- [ ] Native/Wasm/Go controls prove invalid requests are not sent, invalid responses are reported, and annotation calls preserve existing Model state/freeze/transaction contracts.
- [ ] All six existing live scenarios, default/required-TLC gates and timing execute successfully with unchanged independent Model and existing goldens.

## Done summary
# Task .3 completion

Implemented 39 typed activity message catalogs, ten exhaustive enum catalogs and complete WorkflowService classification. Actual SDK attempts pass through Rust request filling/validation and successful-response validation. Background failures survive retries/shutdown; supplied values and retry IDs remain stable, empty polls work, and response bytes are never repaired. Secret diagnostics remain value-free.

Final worker source: 9baf1f772e79f4332fb20906fc5b7af0432224f4, all seven source commits integrated into umpire. Final focused verification passed 27 native controls, three compile-fail fixtures, five API/descriptor controls and API/realization Clippy. Final live verification passed all six original scenarios and six annotation controls, with actual coverage of all seven required activity RPCs. Infrastructure scope reasons and failed cancellation observations remain explicit.

Final make model-check and make model-check UMPIRE_BACKENDS=require passed. Required TLC executed all twelve admitted generic fixture consumers and semantic controls; the unchanged Temporal activity Model retains its explicit source-lift refusal. The conductor independently ran the integrated full model gate on final source (session 65352, exit 0). Existing independent Model, lift, coverage, wire and macro goldens remain unchanged.

Actual same-source make model-time passed as a command: 43.685s warm gate versus 30s soft target and 10.616s edit/test versus 10s, 54.499s total. Both soft targets were missed, not waived or claimed passed. Bounded attribution identified Temporal Go-host startup (34.560s) and native tests (6.379s); historical supplemental trybuild profiling is retained without claiming an isolated current compile measurement. The separate stale missing-directory build-input defect was fixed with RED/GREEN and absent-directory warm-reuse controls; every test remains enabled.

Two-axis quality reports and focused resolutions are preserved separately in task-3-quality-audit.md and task-3-failure-context-finding.md. The layout, supplemental admission, SDK failure coverage, failure-name context and request-enum expectation findings are resolved. Formal first-round fan-out retained one P2; the single resumed review returned SHIP with zero surviving findings at reviewed root 7073840836e87e49ebe29c89e7d2c22bf58a0dd7. Full replies are retained in task-3-implementation-review-round-{1,2}.md. Writer and reviewers are the same model family.

Task .4 still owns live gap probes, reverse coverage and the gap golden. No accepted-gap waiver, Model fitting, remote write or PR was introduced. Final handover and refreshed TLC proof/control transcripts remain under .flow/tmp.

stage: wave - ran (isolated implementation lane with four disjoint implementation subagents; integration, review and closure conductor-owned)
stage: impl-review - ran [2026-10-20T22:29:03Z..2026-10-20T23:18:11Z] (model: gpt-6.1-sol at high)
Tracker sync: n/a (bridge inactive)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7e5eb61c036f8299a1c7b4d16424a7f054104ddd, 56aa9bd2252e0ae62a535e8606400d49e73a6f43, 55a27ddd024a20ea94c40f0209eeb7d6408794a4, 5c9d19cca65f6920ceb8fd988d107f4eb866618f, 1691aa14ce849d628a752bafcec34212248ef0c3, a23540b5446e834df6fc1e478aae093d991fb38f, 9baf1f772e79f4332fb20906fc5b7af0432224f4, 7073840836e87e49ebe29c89e7d2c22bf58a0dd7
- Tests: cd model && mise exec -- cargo test -p umpire-temporal --test catalog --test annotations --test realization --test supplemental_admission, cd model && mise exec -- cargo test -p umpire-temporal-api --test build_inputs --test descriptor, cd model && mise exec -- cargo clippy -p umpire-temporal -p umpire-temporal-api --all-targets --all-features -- -D warnings, make umpire-check-model-slice, make model-check, make model-check UMPIRE_BACKENDS=require, make model-time UMPIRE_INNER_CRATE=umpire-temporal UMPIRE_INNER_SOURCE=temporal/features/activity/api.rs, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check
- PRs: