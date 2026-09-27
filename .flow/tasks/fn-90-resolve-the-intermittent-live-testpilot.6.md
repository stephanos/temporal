---
satisfies: [R6, R7]
---
# fn-90-resolve-the-intermittent-live-testpilot.6 Resolve or close the async-Nexus INCONCLUSIVE Run

## Description
Resolve (3), the async-Nexus Run that ends INCONCLUSIVE or INCOMPLETE, at the path fn-90.3's
signatures name, or close it as not reproduced (R6; R7 only if the cause is external).

**Size:** S if not reproduced; M for one path's fix. If signatures show both (3a) and (3b), fix the more frequent one here and create a follow-up task for the other with `flowctl task create --spec fn-90-resolve-the-intermittent-live-testpilot` (dep on this task; fn-90.7 then depends on it too).
**Files:** (3a): `model/Temporal/Case/Realization/Nexus.lean` (declared `timeoutMilliseconds`), regenerated fixtures. (3b): `model/Temporal/Feature/Nexus/Caller/Model.lean` or the Program realization in `model/Temporal/Case/Realization/Nexus.lean`, regenerated fixtures. Quarantine: `tests/testcore/testpilot/quarantine.go` and `quarantine_test.go` (created by fn-90.5 if it needed one; otherwise here).
**Touches:** [tests/testcore/testpilot/quarantine.go, tests/testcore/testpilot/quarantine_test.go, model/Temporal/Case/Realization/Nexus.lean, model/Temporal/Feature/Nexus/Caller/**, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/case-runtime-conformance/**, tests/testpilot_nexus_caller_case_test.go]

### Approach
- Not reproduced in fn-90.3 on every (3) identity: close with that evidence. No quarantine.
- Path (3a), Run INCOMPLETE from an exhausted instruction timeout: the bound is Case-owned. Correct the declared value in the Nexus realization (`model/Temporal/Case/Realization/Nexus.lean:333-370,442-464`; `respond-async`, `finish-workflow` and the other 5000 ms bindings) to cover the p99 fn-90.3 extracted, with a stated margin. Regenerate with `make umpire-gen-case-runtime-conformance`; check with `make umpire-check-case-runtime-conformance`. Never override it in Go or the test.
- Path (3b), Run COMPLETED with an unresolved rule on the completion-before-start path: first confirm on both switch values that the server records the synthesized started event with the same shape (CHASM `chasm/lib/nexusoperation/operation.go:268-298`; hsm `service/history/hsm/nexusoperations/completion.go:122-155`). Then pick one:
  - the async Model admits the synthesized started event as product behaviour (upstream temporal#6821), or
  - the Program orders the completion after the start is observed (`await-completion-authority` at `Nexus.lean:442` gains an observed-start precondition).
  Record which, and why, in the spec's Decision Context via `flowctl spec set-plan`. Prefer the Model option when the Case's claim is about the server's async protocol, the Program option when the claim is about the handler's normal path.
- Before any Lean edit: `flowctl show fn-88-veil-concrete-checker-as-the-umpire`. If fn-88 is open, block this task naming fn-88 and the change (spec Edge Cases, fn-88 overlap) rather than landing uncoordinated.
- Error cases to keep proving: the worker-outage Case still closes as its Contract says; the forged-completion control (`tests/testpilot_nexus_control_case_test.go:30`) still yields VIOLATED; `CheckSwitchAgreement` still fails on a hsm/chasm divergence.
- Cause outside the repository: upstream issue plus an R7 quarantine for exactly that signature in `tests/testcore/testpilot/quarantine.go` / `quarantine_test.go` (add an entry if fn-90.5 created them, else create them as fn-90.5 describes).
- Afterwards rerun the fn-90.3 loops 3 to 6 at their counts.

### Investigation targets
**Required:**
- fn-90.3 receipt (signatures and latencies for (3))
- fn-90.3 Run-capture latency figures (p99 for the bound)
- `common/testing/testpilot/internal/execution/recorder.go:319-329,377-400` (3a)
- `common/testing/testpilot/internal/verification/correlated.go:645-670` (3b)
- `model/Temporal/Case/Realization/Nexus.lean:330-470`
- `model/Temporal/Feature/Nexus/Caller/Model.lean`

**Optional:**
- `tests/testpilot_nexus_caller_case_test.go:124-210` (switch-value runs)

### Key context
- SEM-16: no retry in Testpilot, the Driver or the evaluator. QLF-05: no evaluator weakening.
- Do not edit `tools/umpire/cmd/umpire-gen-*` or `model/Temporal/API/**` (other sessions).
## Acceptance
- [ ] Either "not reproduced on the successor" per identity, or the named path fixed at its source with regenerated fixtures and the Decision Context updated for (3b).
- [ ] After a fix: loops 3 to 6 from fn-90.3 show zero failures at full count; controls and switch agreement still pass.
- [ ] Any quarantine matches the spec's contract and its offline unit test passes; otherwise "no quarantine needed".
- [ ] `make lint-code-fast` clean; `make umpire-check-case-runtime-conformance` clean if fixtures changed; `make lint-model` (with `LEAN_NUM_THREADS=1`) clean if Lean changed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
