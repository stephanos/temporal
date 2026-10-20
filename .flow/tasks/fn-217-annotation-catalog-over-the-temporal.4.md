---
satisfies: [R3, R4]
---
# fn-217-annotation-catalog-over-the-temporal.4 Gap probe and reverse coverage: gaps.txt golden

## Description
For every expectation from the Rust activity catalog, build a minimal violating
request from a valid filled baseline, send it to the actual in-process activity
server, and derive Confirmed/Mismatch/Accepted/Unverifiable from real results.
Use .3's typed catalog/guest route; extend typed Rust probe generation narrowly
where needed. No duplicate Go annotation/mutation rules or hand-built outcomes.
Go performs transport and prerequisite setup only. Verify generated cases really
violate their declared constraint, and fail rather than silently omit a case that
cannot be generated. Unverifiable is for declared unobservable/output-only rules,
not a substitute for missing request-probe implementation.

Compare the committed gaps.txt under model/temporal/features/activity/expected;
ordinary tests never update goldens. Mismatch and Accepted fail unless the Rust
catalog has a valid dated accepted-gap reason. Intentional Expectation::Accepts
plus successful RPC is Confirmed, not an unacknowledged gap. In particular, do not
invent UUID rejection for Start/Cancel request IDs to make the report uniform.
Check code, anchored message pattern and requested detail type. Match raw server
messages only in memory; pass code/match boolean/detail type to the value-free
report, never raw messages or request/callback envelopes containing secrets.

Use valid real namespace/queue/type/timeouts before mutating one constraint.
RespondComplete/Failed identity/failure probes require a real live polled task
token because token decoding and namespace lookup happen first. Empty token
bytes are not a reliable malformed-protobuf control. Retry validation probes
need a retry-enabled baseline: MaximumAttempts=1 bypasses remaining retry checks.
Oversize completion/failure payloads can record a failure while returning RPC
success; cancellation after completion is state-dependent, not a shape rule.
Preserve these distinctions rather than fitting expected errors to probe setup.

Reverse coverage indexes real ast.FuncDecl definitions by package and receiver/
function identity. Verify every listed covers target exists. Trace touched RPC
entrypoints and relevant local/imported validator helpers; names beginning
validate alone miss exported helpers and inline Respond checks. Report uncovered
or unresolved validator seams honestly without failing merely for missing catalog
coverage; a vanished explicitly listed target fails. Relevant roots include CHASM
Start/Describe/PollActivityExecution/Cancel and frontend worker Poll/Respond RPCs.

Preserve existing Model/goldens/ABI and Method 6 reservation. Any additional typed
probe protocol arm keeps prior field/enum numbers. Makefile changes stay in the
Testpilot section. Existing slice scenarios remain controls; cleanup must not
swallow assertion failures or leak workers/tasks.

**Touches:** `tests/testpilot_model_gaps_test.go`, `tests/testpilot_model_slice_test.go`, `tests/testpilot_model_annotations_test.go`, `model/temporal/features/activity/api.rs`, `model/temporal/features/activity/expected/**`, `model/temporal/testing/**`, `model/temporal/annotate/**`, `model/temporal/realization/**`, `model/temporal/guest/**`, `model/temporal/proto/**`, `model/temporal/hosts/go/**`, `model/umpire/annotations/**`, `model/umpire/wasm/**`, `model/umpire/proto/**`, `model/umpire/hosts/go/**`, `Makefile`
**Quick:** actual `TestTestpilotModelGaps` with a freshly built guest and the existing environment; focused typed mutation/refusal/coverage controls.
**Full gates:** `make model-check`; dedicated gap and model-slice targets; timing if a new gate step is added.
Use a unique Cargo target and existing in-process harness. No server-validator
changes, external dependencies, metadata-only report fixture or golden updates
that mask a mismatch.

## Acceptance
- [ ] Every catalog request expectation produces an actual live probe result and stable golden; no missing case is disguised as Unverifiable.
- [ ] Wrong code/pattern/detail type, accepted expected rejection and stale golden fail; intentional expected acceptance is Confirmed.
- [ ] Minimal probes isolate setup/state from the declared request constraint, including live-token and retry-enabled prerequisites.
- [ ] A listed validator removed/renamed in a negative control fails reverse coverage; uncovered actual validators are reported honestly.
- [ ] The probe consumes Rust catalog/mutation metadata through the typed route, without a duplicate Go rule list, synthetic results or leaked secrets.
- [ ] Full model gate and actual dedicated gap/live-slice commands pass with original Model and wire contracts preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
