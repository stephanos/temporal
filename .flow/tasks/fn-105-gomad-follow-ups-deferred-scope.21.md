---
satisfies: [R21]
---
# fn-105-gomad-follow-ups-deferred-scope.21 D21: investigate host-clock reporting escapes and policy-compatible remedies

## Description
Covered by the 2026-09-30 blanket investigation approval. Existing inventory classifies host-clock escapes including MemStats.LastGC, debug.GCStats, the Prometheus last-GC gauge, FIPS monoTime, execution-tracer snapshots, and Linux syscall.Gettimeofday behind an exact pack gate. LastGC is target-visible reporting state. Assess practical evidence exposure and remedies within the current prohibition on collector/runtime-assembly patches. D11 depends on this investigation, with audit implementation conditional on these findings. Investigation does not authorize collector changes or silently accept the limitation.

## Acceptance
- Retain minimal exposure evidence and inventory paths on the qualified platforms, distinguishing target-visible reporting from runtime control flow and gated operations.
- Establish whether the escapes can affect supported target evidence and whether any are related to D12/D14, with evidence rather than a shared-cause assumption.
- Evaluate remedies that preserve collector behavior and the current deterministic boundary; explicitly identify any option that would require a separate patch-policy decision.
- Record the feasibility result, affected claims, correction owner, and proposed fix or documented limitation in fn-105 for a subsequent decision. Retain any selected fix as explicit open work.
- As the prerequisite for D11, record whether exposure evidence requires a dynamic Linux clock audit, its feasible scope under current restrictions, and the recommended next action. Identify intentionally retained host-clock paths that the proposed fixture must account for. This recommendation does not claim an audit is implemented.
- Do not close D12/D14 from classification of a reporting escape, widen generic syscall access, alter collector policy, or treat an unavailable audit as passing evidence.

## Done summary
Investigated the host-clock escapes; nothing is fixed. Report: docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md (indexed in docs/research/gomad/README.md); milestone Open findings bullet and D11/D21 rows updated; evidence under .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-* and task21-review-rounds.md.

Findings: none of the four inventoried escapes is read by runtime control flow; each hands a host value to the target. A fixture printing LastGC is nondeterministic at stdout with exact choice replay (darwin/arm64, seeds 1/11/17). By source inspection the functional closure neither emits nor consumes the values, and the four escapes are not a cause of D12/D14; that classification closes neither. linux/amd64 was not run. linux cputicks (RDTSC) is an unpinned fifth path and stays open for D12. Open work for the fn-105 owner's decision: document the limitation, correct and widen the inventory, and decide on the proc.go stamp overwrite (needs patch-policy judgement). D11 stays deferred, not implemented; revival triggers and feasible scope are in the report.

Review: 4 rounds, last verdict NEEDS_WORK with one should-fix finding and no blocker; that finding (profile export wording) is applied and unreviewed. Closed on the conductor's instruction.

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden; 4 rounds, 4th authorized by conductor) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: none (investigation task; functional suites, make test and toolchain builds forbidden on the shared host), GOMADSEED=1 direct-seed go run ./cmd/lastgc x2 (darwin/arm64): LastGC/PauseEnd differ, virtual clock equal, gomad qualify --seed 1 --repeat 2 go-run ./cmd/lastgc: qualified=false first-divergence=stdout.full_sha256, gomad qualify --seed 1 --repeat 2 go-run ./cmd/control: qualified=true, gomad qualify --seed {11,17} --repeat 2 --choices --replay-successes --success-limit 4 --success-bytes 64MiB go-run ./cmd/lastgc: choice-replay=exact divergence=stdout.full_sha256, gomad qualify --seed {11,17} --repeat 2 --choices --replay-successes --success-limit 4 --success-bytes 64MiB go-run ./cmd/control: match=true choice-replay=exact, GOOS/GOARCH go list -deps -test -tags disable_grpc_modules,gomad,test_dep ./tests + reader grep (darwin/arm64, linux/amd64 source closures), linux/amd64: not run (host unavailable), review: raw codex bridge, 4 rounds (4th authorized by conductor), last verdict NEEDS_WORK should-fix only; final finding applied and unreviewed
- PRs: