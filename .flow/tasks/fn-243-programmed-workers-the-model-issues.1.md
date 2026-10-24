---
satisfies: [R1]
---
# fn-243-programmed-workers-the-model-issues.1 Program on Reset: protocol, Runtime, Realize, WorkerProgram envelope and the activity rows

## Description
Add `ResetResponse.program` (optional Any) and `PROPOSAL_KIND_READ = 3` to `umpire.model.v1` with wire goldens; `Realize` gains a program hook with a refusing default and `answer` gains a refusing default; the Runtime admits Read proposals (pending with no action, cleared by a correlated final Explained, as Bootstrap is). Define `umpire.temporal.v1.WorkerProgram` (guards, ordered rules of key Any to respond Anys), `ActivityTaskKey { scenario, attempt, activity_type }`, the hold instruction field, and the `umpire-scenario` header name; remove `ObserveActivity.worker_tasks`. The Temporal Realization derives the program from its one reply policy by enumerating scenarios and admitted attempts, stamps the scenario into the Start header instead of the input, proposes Read and Cancel (with a seeded hold), and drops its Answer implementation. Add `umpire program --model --world --out`. Coverage unit test: every scenario and admitted attempt has a row. Executors are not in this task.

Placement: after fn-228.6 and fn-244.2 land; one lane on the Driver.
## Acceptance
- [ ] Deterministic program on Reset for the admitted World; wire goldens pin `ResetResponse.program` and `PROPOSAL_KIND_READ`
- [ ] `umpire program` writes the program file; Runner and Driver never decode it (import graph)
- [ ] Scenario in the `header`; Read proposals; `worker_tasks` gone; Answer dropped from the Temporal Realization
- [ ] Coverage test: every scenario and admitted attempt has a row
## Done summary
Users can export the Model's deterministic worker program with `umpire program --model --world --out` before execution. Reset carries the opaque program, activity scenarios travel in headers, and Read proposals observe worker progress without proposing worker actions. The activity table covers all three scenarios and every admitted attempt; shared wire vectors and source/import boundaries pin the contract.

The full required gate `CI=1 UMPIRE_BACKENDS=require make model-check` exited 0, including native/TLC checks, CI guest freshness, and generic, Kettle and Temporal Go hosts. Focused Go suites, native/Wasm Read regressions, program export and final Go lint passed. The exported program is `.flow/tmp/fn2431-program.binpb` with SHA-256 `518a416709b0ec47f968ff6f97fba0cfa6dc536bce59f233d59906f0497548cd`. The committed guest and the policy's sole changed field, `modelHash`, equal `b6334020f894cc27f694602e5f9980ff8727cbe79d29c2e8bbae493c6525753e`.

baseline: green. The pre-edit runtime/Temporal native suite and focused command/Testpilot/canary Go suite passed. No Quick commands were listed in the task/spec. Red reproductions for program issuance, wire preservation, CLI absence, forbidden live Answer, and unchanged Started reads are retained in `.flow/tmp/fn2431-*.log`.

The first full gate exposed the intentionally retired Answer contexts in the active realization coverage golden. Only that derived golden and the authored macro snapshot were updated; the Model and its 15 frozen compatibility artifacts are unchanged. The conductor independently verified all 15 at `97893090b8`. A 600-second retry stopped during the Kettle host suite with exit 124 and remains INCONCLUSIVE in `.flow/tmp/fn2431-model-check-inconclusive-timeout.log`; the subsequent full run exited 0. An earlier ad-hoc generic Host invocation lacked `UMPIRE_PANIC_NATIVE` and was also inconclusive; the canonical gate supplied and checked those controls.

Task 2 owns raw/SDK executors, unknown-key refusal, hold transport, result/failure content judgment and fresh live parity. At this predecessor boundary, Read never starts the legacy Answer worker and unsupported `TemporalInstruction` execution produces a Driver fault. Existing live slice and mutant controls remain unchanged and were not represented as live execution evidence here.

stage: impl-review - ran [2026-10-24T06:39:30Z..2026-10-24T06:44:25Z] (codex:gpt-6.1-sol:high; SHIP, three clean draws; same family as writer)
stage: quality-correctness - ran (model: gpt-6.1-sol at high; findings: 0; conductor dispatched)
stage: quality-standards - ran (model: gpt-6.1-sol at high; findings: 0; conductor dispatched)
Tier: session (jev-unavailable(no_key))

Primary review receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-243-programmed-workers-the-model-issues.1.json`, copied to `.flow/tmp/fn2431-review-receipt.json`. Raw independent audits are preserved in `.flow/artifacts/fn-243-programmed-workers-the-model-issues/quality-audits.md`. All five isolated implementation lanes were reconciled and removed; recovery patches and source archives remain under `.flow/tmp/fn2431-*-lane-recovery*`.

GATE_SKIPPED:unittest:green-receipt 8c73ef38 - baseline reused from prior post-gate pass

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 97893090b8926190c24f737ce028836e019ce61c, 8c73ef3876b1b3405ee7a2be31892655d2c70adc
- Tests: cd model && mise exec -- cargo test -p umpire-runtime -p umpire-temporal --quiet, go test -tags test_dep ./tools/umpire/cmd/umpire ./common/testing/testpilot/... ./tools/canary/..., CI=1 UMPIRE_BACKENDS=require make model-check, make GOLANGCI_LINT_BASE_REV=c24241ebbe94d59c9b4afcd1f1a44c53c8fc3455 lint-code-fast, go run -tags test_dep ./tools/umpire/cmd/umpire program --model model/temporal/hosts/go/umpire_temporal_guest.wasm --world .flow/tmp/fn2431-world.binpb --out .flow/tmp/fn2431-program.binpb, sha256sum -c .flow/tmp/fn2431-program.sha256, INCONCLUSIVE: timeout 600s env CI=1 UMPIRE_BACKENDS=require make model-check (exit 124; superseded by full exit-0 retry with 1800s bound), INCONCLUSIVE: ad-hoc generic Host suite invoked without UMPIRE_PANIC_NATIVE; canonical model-check supplied the complete fixtures and passed, GATE_SKIPPED:unittest:green-receipt 8c73ef38 - baseline reused from prior post-gate pass
- PRs: