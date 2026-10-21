# Task .4 verification and discovery record

Base: `1a92390e19d8a3a82b1c4ebb0bb16943d991cd6b`. The pre-edit canonical model and live-slice baselines passed; the new gap test did not exist and was not counted as a passing baseline. Full logs and command exit receipts are retained in `.flow/tmp/fn2174-baseline.md` and the task handover.

## Live discovery and disposition

The first actual live report ran 62 contextual probes: 60 Confirmed and two Mismatch. It is retained unchanged as `.flow/tmp/fn2174-live-first-immutable.txt`. Unknown Start ID policy enum values were initially declared Accepts, but `chasm/lib/activity/handler.go` rejects them through the reuse/conflict policy maps. The previously cited normalizer does not perform that downstream check. Only these two catalog expectations and covers references were corrected to the actual handler rejection; no independent Model or server behavior changed, and no accepted-gap waiver was added.

The direct initial golden was reviewed explicitly after missing-golden and deliberately stale-golden negative controls. Ordinary tests never update it. Final report: 62 Confirmed, zero Mismatch/Accepted/Unverifiable; 10 declared Accepts and 52 Rejects. Reverse coverage reports 73 uncovered and three unresolved seams, not whole-program coverage. Unresolved calls are callback-validator Validate, visibility-manager ValidateCustomSearchAttributes and duration CheckValid. An unavoidable same-path UUID Accepts companion accompanies the oversized request-ID rejection; unrelated companions remain refused.

Both actual oversized probes materialize and revalidate the decoded protobuf exceeding 2 MiB. Rust sends an exact stateless literal/repeat assembly with integrity metadata; Go performs bounded generic expansion and the actual RPC. The 1 MiB individual ABI and 16 MiB live bounds remain unchanged.

## Quality audits

Fresh correctness and standards auditors ran concurrently. Verbatim reports are `.flow/tmp/fn2174-quality-correctness.md` and `fn2174-quality-standards.md`; separate dispositions are `fn2174-quality-dispositions.md`. Correctness found one Should Fix; standards found two Should Fix and one Consider, all introduced, no preexisting findings. All three Should Fix items were addressed: empty source-owned dated-gap registry with invalid/stale identity controls, extracted Rust probe module, and split additive probe schema. The constructor-consolidation Consider was deferred to preserve the established rendering/output-only API while requiring checked classification for live probes. No real waiver was authorized or added.

Scope exceptions and unrelated external planning exclusions are recorded separately in `task-4-scope-exception.md`. In particular, native fixture ownership retains unknown-file, native-consumer and collision failures; registration of the live golden supplies no native execution/comparison evidence.

## Post-audit verification

- Canonical `make model-check` passed, original terminal handle 73972: 280 passing Rust cases across 94 result groups and both Go-host packages. Log: `.flow/tmp/fn2174-post-audit-model.log`.
- `UMPIRE_BACKENDS=require make model-check` passed, original terminal handle 1736: 280 Rust cases, actual generic TLC controls and agreement, both Go-host packages. Log: `.flow/tmp/fn2174-required-model.log`. Pinned TLC 1.7.4 jar SHA-256: `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`. The Temporal activity Model remains source-lift refused; this is not Temporal TLC execution evidence.
- Actual dedicated gap gate passed, original terminal handle 24242: all 62 contextual probes, valid baselines, logger privacy/failure canary and reviewed golden comparison. Log: `.flow/tmp/fn2174-post-audit-gaps.log`.
- Scoped Go lint passed, original terminal handle 87429: zero issues. Log: `.flow/tmp/fn2174-post-audit-go-lint.log`.
- Refreshed original live slice passed, terminal handle 68885: six scenarios and six annotation controls. Log: `.flow/tmp/fn2174-post-audit-slice.log`.
- Timing command exited zero, original terminal handle 19688, but the whole-gate soft budget was missed: 54.401s versus 30s. Inner edit/test loop met its soft target at 3.391s versus 10s. Log: `.flow/tmp/fn2174-final-time.log`. This is not a whole-gate timing-budget pass.

Timing investigation: the measured Temporal Go-host step dominates at 46.085s. The final task .3 measurement was 34.560s for that step and 43.685s overall, so the current gate is 10.716s slower overall. Existing parity tests create many isolated guests; `model.New` recompiles the module on every call. The rebuilt guest is 18,520,031 bytes. Increased compilation work from the additive probe implementation is a plausible contributor, not a separately profiled causal proof. No controls, timings, test budgets or host isolation were weakened to hide the miss. A shared compilation-cache redesign is outside this task.

Gate receipts cannot be admitted while externally owned MILESTONES planning remains dirty. No receipt bypass or green-receipt reuse is claimed. Source commits and formal review exclude those unrelated planning files.
