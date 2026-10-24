# fn-243.1 quality audits

Base `c24241ebbe94d59c9b4afcd1f1a44c53c8fc3455`, reviewed head `8c73ef3876b1b3405ee7a2be31892655d2c70adc`. Both axes ran in fresh gpt-6.1-sol high contexts, in parallel with the primary CLI review. Reviewer and writer use the same model family. The reports below are verbatim. The primary implementation review owns its verdict and receipt.

### Correctness axis

## Quality Audit — Correctness axis: fn-243.1

### Summary

- Files changed: 51 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship task-1 checkpoint.
- No correctness findings survived review. Executor integration and program-result judgment remain task-2 obligations.

### Evidence

- **Walked:** Reset publication transactions in `model/temporal/realization/src/live.rs:68`; correlated READ completion in `model/umpire/protocol/runtime/src/runtime.rs:365`; deterministic program enumeration in `model/temporal/feature/activity/temporal.rs:59`.
- **Cited:** native/Wasm coverage, unchanged-Started polling regression, CLI failure controls and shared wire vectors. Existing `.flow/tmp/fn2431-model-check.log` ends with passing Go-host packages; exit 0 was **claimed** by dispatch. No builds rerun.

### Test Budget

- Raw additions: 993 test lines versus 358 implementation lines, including setup/fixtures; no setup-excluded ratio asserted.
- Existing tests changed extensively for authorized Answer retirement; response-byte assertions moved into program-row coverage.

### What's Good

- Program contents remain opaque to Runner and Driver; removed `worker_tasks` wire tag is reserved.
- Read-only audit preserved existing Flow sidecar changes. Reviewer and writer use the same model family.

### Standards axis

## Quality Audit — Standards axis: fn-243.1

### Summary

- Files changed: 51 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis)
- No reportable standards findings in committed diff `c24241eb..8c73ef38`.

### What's Good

- **walked**: `WorkerProgram` retains opaque feature keys/responses; activity-specific schemas stay under `model/temporal`. Generic program extraction uses only host/protocol imports (`model/umpire/hosts/go/runner/program.go:43`).
- **walked**: Activity rows derive from one response policy (`model/temporal/feature/activity/temporal.rs:81`); Driver boundary tests encode the program-decoding restriction (`common/testing/testpilot/dependency_boundary_test.go:99`).

### Evidence

- **executed**: Committed numstat/line-count comparison found no 1,000-line threshold crossings; glossary contains zero terms.
- **cited**: `model/README.md:966` documents the task-2 integration boundary. Existing gate output reviewed in `.flow/tmp/fn2431-model-check.log`; no gates rerun or repository files changed.

Correctness axis: 0 findings; no worst tier.
Standards axis: 0 findings; no worst tier.

# fn-243.2 quality audits

Base `8c82fb4ff3ee4c8f694b9067d7f6652803b50428`, reviewed head `c96f9bacf26f95034ecfa634235e7fbc731b7cc8`. Both axes ran in fresh gpt-6.1-sol high contexts, in parallel with the primary CLI review. Reviewer and writer use the same model family. The reports below are verbatim.

### Correctness axis

## Quality Audit — Correctness axis: fn-243.2

### Summary

- Files changed: 61 · Critical 0 · Should Fix 1 · Consider 1 · Ship: ⚠️ Fix first (axis only).

### Should Fix

- **model/temporal/hosts/go/reference/worker/sdk/sdk.go:70** (Conf 75): Walked: `Follow` ignores `table.Namespace`; the SDK worker inherits the client namespace and can execute namespace B’s matching tasks using namespace A’s program — enforce the program namespace before polling and add a mismatch regression.

### Consider

- **model/temporal/hosts/go/reference/worker/sdk/sdk_test.go:22** (Conf 100): Cited: the replacement conversion test checks only payload data, dropping the retired test’s `proto.Equal` assertion over metadata too — restore complete payload equality.

### Test Gaps

- Namespace mismatch between the SDK client and worker program.

### Test Budget

- Added lines: tests ≤1,983 including fixtures/setup; implementation 1,063. Ratio upper bound 1.87:1.
- Existing callback tests retired; READ continuation assertions changed with the corrected hook contract.

### What's Good

- Walked: reconciliation retains actual acknowledgment evidence and correlates the fresh read before judging cancellation.
- Cited retained logs: model gate and 2,412 live identities passed; broad gates were not rerun.

Suppressed findings: 1 at anchor 25.

### Standards axis

## Quality Audit — Standards axis: fn-243.2

### Summary

- Files changed: 61 · Should Fix 0 · Consider 0 · Blocking: none possible (standards axis)
- No standards findings. Worst tier: none.

### Out-of-axis observations

> Out-of-axis observation: **tests/testpilot_model_slice_umpire_test.go:278** — switching to the existing namespace drops the former `EnvCreate=true` and namespace-deletion assertions for retained provisioning behavior (walked).

### What's Good

- **walked:** Shared table decoding owns matching policy; composition injects executor functions without exposing program decoding to the Driver.
- **executed:** File-size calculation found no 1,000-line crossings. Glossary contains zero terms; root `DESIGN.md` is absent.

Existing gate results were supplied by the conductor; no builds or gates were rerun.

Correctness axis: 2 findings; worst tier Should Fix.
Standards axis: 0 tiered findings; no worst tier. The provisioning observation is routed to correctness for the fix decision.

The conductor accepted the namespace guard fix, full payload equality restoration and live provisioning deletion control. This decision does not change either report or constitute a primary review verdict. Follow-up evidence will record resolution.
