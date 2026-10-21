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
Implemented catalog-derived, decoded and locally verified request witnesses, actual live-server probes, value-free classification with an empty validated dated-gap registry, bounded stateless transport recipes, and package-scoped reverse validator coverage. The reviewed initial gaps golden records 62 Confirmed (52 Rejects/10 intentional Accepts), zero Mismatch/Accepted/Unverifiable, 73 uncovered and three unresolved seams.

Tier: session (jev-unavailable(no_key))
stage: impl-review - ran [2026-10-21T00:17:55Z..2026-10-21T00:22:12Z] - SHIP (three-lens codex:gpt-6.1-sol:high; no findings)

Writer and CLI reviewers are the same GPT/Codex family, disclosed rather than claimed cross-family independent. Six in-host child lanes were reconciled; all agents and owned command handles are terminal. No CLI implementation bridge ran; host execution metadata did not expose an actual writer model identifier.

Source range: `1a92390e19d8a3a82b1c4ebb0bb16943d991cd6b..3dc12ffc6e84892f9fbb8f1cc548ea2e3dc9d0cf`. Formal receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-217-annotation-catalog-over-the-temporal.4.json`; fanout RID `a63efcb8166d42818cd8f21b41f66ac3`. All three sidecars under `.flow/review-fanout/<RID>/` were read before empty-plan finalization; the parent spec ledger retains the SHIP provenance. Read-only reviewers could not run fresh Go builds; they stated this limitation and used retained live logs. One reviewer also ran 11 cached Rust controls. No formal fix loop or memory-capture transition occurred.

Baseline: green canonical model gate and existing live slice before edits; no green baseline claimed for the then-absent new gap test. See `.flow/tmp/fn2174-baseline.md`. New APIs/witness route, policy-catalog correction, dated dispositions, stale golden, external artifact ownership and schema dependency controls have retained RED/GREEN evidence.

Post-audit verification at the exact frozen source:

- `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check`: exit 0, 280 Rust cases/94 result groups plus both Go-host packages; original handle 73972, `.flow/tmp/fn2174-post-audit-model.log`.
- `UMPIRE_BACKENDS=require CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check`: exit 0, 280 Rust cases, actual generic TLC controls/agreement, both Go-host packages; original 1736, `.flow/tmp/fn2174-required-model.log`. Pinned jar SHA-256 `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`. The Temporal activity Model remains source-lift refused; no Temporal TLC execution claim.
- `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-gaps`: exit 0, all 62 actual contextual probes and valid baselines, private logger failure/cleanup canary, direct reviewed golden comparison; original 24242, `.flow/tmp/fn2174-post-audit-gaps.log`.
- `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-slice`: exit 0, all six original scenarios and six annotation controls; original 68885, `.flow/tmp/fn2174-post-audit-slice.log`.
- `GOLANGCI_LINT_BASE_REV=1a92390e19d8a3a82b1c4ebb0bb16943d991cd6b make lint-code-fast`: exit 0, zero issues; original 87429, `.flow/tmp/fn2174-post-audit-go-lint.log`.
- `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-time`: command exit 0, whole-gate soft-target MISS 54.401s/30s; inner-loop PASS 3.391s/10s; original 19688, `.flow/tmp/fn2174-final-time.log`. Temporal Go-host step 46.085s dominates; prior task .3 was 34.560s/43.685s step/total. Cold guest compilation per isolated parity host is a plausible contributor, not a profiled causal proof. No thresholds or isolation controls were changed.

The first immutable report (`.flow/tmp/fn2174-live-first-immutable.txt`) discovered two genuine catalog inaccuracies: unknown Start ID policy values were declared Accepts but the actual activity handler rejects them. Exact downstream map checks support correcting those two expectations/provenance, not changing the independent Model/server or inventing waivers. Native red/green policy controls and live confirmation preserve the discovery. The actual >2 MiB decoded payload/reason probes retain the 1 MiB individual ABI and 16 MiB live bound. An unavoidable same-path UUID Accepts companion for the oversized ID is reported explicitly; unrelated companion violations fail generation.

Reverse coverage verifies actual package/receiver/function identities and rejects vanished listed targets or copied-source aliases. Its three unresolved seams are callback-validator Validate, visibility-manager ValidateCustomSearchAttributes and duration CheckValid; 73 uncovered validators are honest findings, not whole-program coverage or gate failures.

Quality audit reports retained verbatim: `.flow/tmp/fn2174-quality-correctness.md` (0 Critical/1 Should Fix/0 Consider) and `.flow/tmp/fn2174-quality-standards.md` (0 blocking/2 Should Fix/1 Consider); all introduced, no preexisting. All three Should Fix findings were fixed and regated. `.flow/tmp/fn2174-quality-dispositions.md` explains the deferred constructor-consolidation Consider: preserve established rendering/output-only constructor semantics while using checked admission exclusively for live probes.

Versioned context: `.flow/artifacts/fn-217-annotation-catalog-over-the-temporal/task-4-verification.md` and `task-4-scope-exception.md`. Explicit conductor exceptions cover only minimal `model/umpire/testing/**` ExternalConsumer registration/controls, generator input in `model/scripts/proto-temporal-go.sh`, and probe schema freshness dependency/assertions in `model/temporal/api/build.rs` and `api/tests/build_inputs.rs`. The original full gate failed on the unowned live golden; exact external registration fixed ownership without pretending native execution/comparison, weakening unknown-file checks, or permitting external-only fixtures.

Full classification was required; model/gaps/slice green receipts were refused with `NO_RECEIPT: worktree dirty outside the ignore set (MILESTONES.md) - receipt not warrantable`. Actual terminal logs supply evidence; no receipt bypass or skipped-gate claim. Source content stayed unchanged through review and post-review range verification.

Five externally owned planning files are deliberately preserved dirty and excluded by approved staging boundary: `MILESTONES.md`, `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.6.{md,json}`, and `.flow/tasks/fn-217-annotation-catalog-over-the-temporal.7.{md,json}`. No clean-tree claim, remote write, planning-task implementation, Model/server edit, protocol reservation change, accepted-gap waiver, ordinary golden update, PlanSync or tracker bridge occurred.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3dc12ffc6e84892f9fbb8f1cc548ea2e3dc9d0cf
- Tests: CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check, UMPIRE_BACKENDS=require CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-gaps, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make umpire-check-model-slice, GOLANGCI_LINT_BASE_REV=1a92390e19d8a3a82b1c4ebb0bb16943d991cd6b make lint-code-fast, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-time
- PRs: