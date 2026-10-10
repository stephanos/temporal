---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.67 Preserve busy host workloads while correcting spin lint

## Description
Correct the two configured spinning-loop findings while preserving intentional host CPU activity. Follow [the bounded admission](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-67/admission.md).

**Touches:** [tools/gomad3/internal/gomadtool/conformance/runtime_repeatability.go, tools/gomad3/runner/internal/execution/process_test.go, tools/gomad3/internal/gomadtool/conformance/cpu_load_lifecycle_test.go, tools/gomad3/runner/internal/execution/unresponsive_supervisor_lifecycle_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-67/**]

Mirror the established soak atomic stop lifecycle in the host-load helper; use unconditional atomic increment in the intentionally unresponsive supervisor fixture. Preserve busy CPU activity, locked-thread/start/join/error behavior, guard, original parent assertions and production execution. No blocking/yield/sleep/clock substitution, lint exception, runtime/native fixture or error-literal change.

Configured lint is the meaningful failing check. Baseline and successor bounded subprocess lifecycle controls prove preservation, not a fabricated behavioral RED. Verify the real helper stays alive without protocol output until killed/reaped and that zero/two load workers stop repeatedly/concurrently. Run test_dep focused/end-to-end, package architecture, validation, affected vet/errortype, formatting, configured unfiltered lint and repository fast lint. Root serializes execution lanes and owns original-base full-block comparison against 52, both-source-set checks, integrated review and lifecycle. Native qualification remains deferred and cannot be inferred from ordinary portable source controls.

## Acceptance
- [ ] Baseline configured affected lint retains both complete SA5004/SA5002 blocks. Successor removes exactly those two without a new finding; original-base aggregate results and integrated errortype disposition stay explicit.
- [ ] Host load retains locked OS threads, startup acknowledgement/count/deadline/error, busy polling, cleanup-before-join and idempotent stop. Its zero/two-worker repeated/concurrent lifecycle controls pass in externally bounded subprocesses.
- [ ] The actual supervisor helper retains guard and unconditional CPU work without wraparound exit, voluntary yield, I/O or protocol response. Direct bounded liveness/kill/reap/no-output preservation and the unchanged parent timeout test pass.
- [ ] Only the four named product/test paths change. All existing assertions, comments and unrelated behavior remain; no lint-policy, literal-format, runtime, native fixture, production execution or public contract change.
- [ ] Current architecture/generated validation, affected vet/errortype, both-source-set static checks, formatting and required repository fast lint have bound evidence or exact unaffected-input reconciliation. Required red source acceptance remains open, with no native qualification or aggregate-green claim.
- [ ] Root integrates/reviews the candidate and retains the next frozen original-base block comparison and coverage boundaries. Formal completion follows all remaining owned source gates; fn-128/fn-149 obligations stay deferred and unverified.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
