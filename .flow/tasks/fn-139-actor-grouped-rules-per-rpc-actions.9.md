---
satisfies: [R10]
---
# fn-139-actor-grouped-rules-per-rpc-actions.9 Enforce rejection RPC codes in conformance

## Description
Runs after the DSL batch's single regeneration and after fn-139.8 has exported the shared rejection-to-gRPC-code table on every Temporal realization.

Make conformance compare each performed Model step's outcome with the Run instruction result correlated to that step. If the Model outcome is `rejected(r)`, the observed gRPC status code must equal the realization table's code for `r`; a mismatch fails conformance and names the expected and observed codes. An accepted step observing an error also fails, and a rejecting step observing OK fails. Compare status codes only, never message text, and use the mapping exported by task 8 rather than a Go-side table.

Trace the existing Case/program coordinates through `tools/umpire/conformance/` to associate completed instruction events with the performed action/class step. Reuse the status code already captured on Run instruction outcomes in `common/testing/testpilot/temporal/`; change capture only if a focused test proves the needed code is absent. Remove any conformance expectation made obsolete by the new generic check. Add table-driven Go tests for an exact rejection match, a mismatched rejection naming both codes, accepted-with-error, and rejected-with-OK.

**Files:** `tools/umpire/conformance/`, its tests and fixtures, and only if proven necessary `common/testing/testpilot/temporal/`.

**Touches:** [tools/umpire/conformance/**, tools/umpire/**, common/testing/testpilot/temporal/**]

## Acceptance
- [ ] Conformance correlates a completed Run instruction outcome to its performed Model step and accepts a rejecting step only when the observed status code equals the realization's exported code for that `Rejection`.
- [ ] A rejection-code mismatch fails the step and names both expected and observed codes. Focused Go tests also cover an exact match, an accepted step observing an error, and a rejecting step observing OK.
- [ ] The implementation reads task 8's realization metadata and contains no independently maintained rejection-to-code table; it compares codes only, never message text.
- [ ] The touched Go unit tests pass, and the batch's full gates and live run exercise the new conformance path before fn-139 closes.


## Done summary
Correlated every performed RPC witness step with its lowered instruction result and enforced the realization-exported rejection-to-gRPC-code mapping. Accepted steps require success; rejected steps require protocol failure with the mapped code; mismatches name expected and observed codes and never compare message text.

The integration audit added separator-insensitive normalization matching Testpilot's recorded `codes.Code.String()` spelling and made the Temporal check explicitly opt-in through rejection metadata, so generic realizations retain their own outcome vocabulary. Focused tests cover exact and mismatched rejection codes, accepted-with-error, rejected-with-OK, repeated performed instructions, and the generic no-metadata path.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d380d2b1ee, a58929556b
- Tests: PASS: go test ./tools/umpire/conformance, PASS: mise exec -- go test -json ./tools/umpire/..., PASS: make umpire-gen-model, PASS: make umpire-check-live-tests (87 identities, empty failure set), PASS: TestTestpilotNexusCallerScheduleToStartTimeout count=10 across HSM and CHASM
- PRs: