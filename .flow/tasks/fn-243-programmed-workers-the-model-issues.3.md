---
satisfies: [R4]
---
# fn-243-programmed-workers-the-model-issues.3 A raw executor in another language follows the program file

## Description
After .2. A raw executor in one language other than Go follows the same program file with no connection to Umpire, if a gRPC stack for that language is available offline (checked 2026-10-24: the sandbox has the Go SDK in its module cache only; no Python SDK or gRPC). If none is available, record what is missing and how the Go raw executor's tests would be reused, and close the task as recorded rather than proven.
## Acceptance
- [x] Non-Go raw executor follows the program file and passes the complete and fail scenarios, or the gap is recorded with what is missing
## R4 evidence

Recorded, not proven. The refreshed offline inventory, missing language runtimes/bindings and
Go raw/live CLI reuse points are recorded in
[the R4 gap record](../artifacts/fn-243-programmed-workers-the-model-issues/non-go-offline-gap.md).
Protoc 29.5 is available through mise; a usable non-Go gRPC stack was not found in the checked
locations. No non-Go execution passed.

## Done summary
Recorded R4's conditional offline gap with the missing non-Go gRPC runtimes/bindings and concrete Go raw/table/issued-program/live CLI reuse points. No usable non-Go stack was found in the checked locations; protoc 29.5 is available through mise. R4 is recorded, not proven; no non-Go Complete or Fail execution passed.

Durable evidence: `.flow/artifacts/fn-243-programmed-workers-the-model-issues/non-go-offline-gap.md`. Refreshed inventory output: `.flow/tmp/fn2433-inventory.log`. Scope is two documentation files plus Flow lifecycle/review metadata. The Model, guest, program, original counterexamples, deferred cancellation obligation and open parent specs remain unchanged.

Tier: session (jev-unavailable(no_key)). No actual implementer model is inferred from the dispatch. The actual reviewer metadata records gpt-6.1-sol/high, the writer family; this is not a cross-family review.

Baseline: green, actual `cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal` exit 0, 12+7 tests, zero filtered, `.flow/tmp/fn2433-baseline.log`. No honorable receipt existed for that exact command. No baseline handoff was used. Inventory commands, link/source-reference validation, guest SHA256 comparison, Flow spec validation and `git diff --check` passed. The correct guest SHA256 remains b4a4f57f1a64525ae568bd56a73eff85b0b253aeae44d9e4de3e2d88f546cf2a; the program export hash is attributed to the task .2 retained artifact, not regenerated here. Two guessed CLI test paths did not exist during discovery; the actual harness was located via rg and read. Those failed lookups supplied no verification credit.

GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched)

The spec defines one unittest Quick command and no smoke gate. Final Verify classified tier-B, so no broad Model/live gate was rerun. The correctness reviewer attempted Cargo but hit its read-only build-lock restriction; only the worker's actual 19-test baseline receives execution credit.

stage: impl-review - ran (codex:gpt-6.1-sol:high, first-round correctness/contracts/integration draws all SHIP, zero introduced/pre-existing findings, merged receipt finalized)

Primary receipt: `/tmp/impl-review-receipt-8f37faba39e2-fn-243-programmed-workers-the-model-issues.3.json`. Raw review sidecars: `.flow/review-fanout/fa10ba216af440c69b11b81d745a6575/`. Review scope base 8295401a5dab0d90c2b5661566e74a39e8264cde, implementation commit 20f62a7123. Task .3 alone is closed; fn-243 and fn-228 remain open and fn-243.5 remains blocked.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 20f62a712338ffdf51eb5a6c38ddafe29af6bf30
- Tests: cd model && mise exec -- cargo test -p umpire-temporal --test program_history --test proposal, GATE_SKIPPED:unittest:docs-only - cumulative diff classified tier-B (no executable paths touched), Offline inventory: Python importlib resolution, Node require.resolve, mise protoc, Cargo/pip/npm/Maven/Gradle cache inspection (.flow/tmp/fn2433-inventory.log), python3 task link/source-reference validation and guest SHA256 comparison, flowctl validate --spec fn-243-programmed-workers-the-model-issues --json, git diff --check
- PRs: