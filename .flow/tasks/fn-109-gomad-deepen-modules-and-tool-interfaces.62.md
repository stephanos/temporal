---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.62 Bound periodic-progress fixture startup and observe early completion

## Description
Bounded ordinary-source hang correction for fn-112.10, admitted at cd883200c6791527acf263b151e7cf54d7ccd1e8 under fn109 R18/R19. The root's retained combined-57-59/hang-diagnostic.md and SIGQUIT stack prove runner_test.go:144 blocks on executor.started before creating its deadline or reading buffered completed. Fresh Astra inspection confirms preparationWith validates the fake prepared target, so Linux/arm64 may reject it before Execute. Existing errorPreparer supplies a real portable early-completion control without any policy bypass.

**Touches:** [tools/gomad3/runner/runner_test.go, tools/gomad3/runner/progress_start_test.go]

Repair only startup waiting in TestRunReportsPeriodicProgressWhileTargetIsRunning: bound startup and observe started, completed and deadline. Any completion before Execute, including nil, must fail explicitly rather than hang or pass. After actual startup keep the existing separate one-second periodic-progress deadline, callback/running-state predicates, executor release, completion error assertion and all other test behavior. A test-local wait helper is permitted only to exercise these exact startup alternatives causally; no new production injection/abstraction, fake platform, changed profile validation, skips, changed execution behavior, weakened heartbeat assertions or inflated heartbeat budget.

Retain independent fixed controls for actual start, existing errorPreparer's early error, pre-start nil completion and timeout. Match BASE versus final expectations; a bounded actual BASE hang must be identified as diagnostic observation, not normal coverage. Prove bypassing the completion/deadline alternatives is detected without leaking an unbounded test goroutine. Preserve every non-admitted test byte and disclose the newly bounded startup diagnostic behavior separately from product preservation. Full ordinary Runner may retain unchanged unsupported-host failures; removing a hang is not a green package gate or native qualification.

Use an independent startup timeout derived from the existing OverallTimeout plus TerminateGrace, not the post-start heartbeat allowance. Failure cleanup must release the fake exactly once, registered before launch; context cancellation alone does not unblock it. The existing successful release boundary remains unchanged. A real temporary errorPreparer substitution is a lawful portable causal probe: BASE blocks before observing its sentinel while final produces an ordinary failure containing it without entering Execute. Run probes on private exact source copies, restore the healthy preparer and classify timeout/diagnostic abort separately from normal nonzero test termination. Final source must keep the original preparation and platform validation.

**Quick:** pinned stock Go1.27.1, all tests -tags test_dep -count=1; focused startup controls and the unchanged heartbeat test with a bounded outer test watchdog; vet/format, affected configured lint, check-only make lint-code-fast FIX=false, root final combined full ordinary Runner once after this source correction. Source/tool/raw hashes and actual exits required. All Go/build/test/lint/vet/generator execution waits for root shared-lane grant. Handover/receipts .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-62/ are lifecycle exemptions from product Touches. Root owns lifecycle, gates, independent review and commits; no worker completion/verdict/commit before grant. Parent acceptance remains open wherever unproved; no native fn149/fn128 revival, policy widening, dependencies, merge/push/PR/CI.
## Acceptance
- [ ] Actual startup waits are bounded and early completed error/nil results fail explicitly; no policy bypass, skip or hang-as-success behavior.
- [ ] Existing periodic callback assertion and its independent post-start budget, release/completion semantics and every non-admitted source byte remain intact.
- [ ] Fixed portable controls exercise existing errorPreparer and start/nil/timeout paths; genuine sensitivity evidence distinguishes causal behavior from syntax or source-only proof.
- [ ] Focused/vet/format/configured and canonical lint retain actual source-bound outcomes, no introduced findings; root full ordinary Runner follows integrated correction and inherited failures remain source-owned.
- [ ] Independent source-progress review and separate integrated evidence checkpoint preserve all parent acceptance and native deferrals. Removing the hang alone supplies no formal SHIP/Done or full-package pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
