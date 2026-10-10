---
satisfies: [R8]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.8 Report Draft capture confinement from an isolated baseline

## Description
Deliver R8's capture-checking finding independently from the source-changing lane. Wait for none of tasks .1-.6. Work in a private disposable checkout/compiler project and publish no shared tracked source or docs; integration task .7 publishes the report.

Execution-order amendment authorized by the owner on 2026-10-10: fn-156 implementation may proceed ahead of fn-155 closure in parallel with fn-146 while fn-157 is deferred. For this independent report, freeze the current integrated local baseline containing committed fn-155.1–.5 rather than waiting for fn-155.6. This changes only start order. Pin Draft/compiler source hashes and report the outstanding fn-155 baseline reconciliation; task .7 must compare those hashes against the eventually closed predecessor and rerun affected experiments before final publication if inputs differ. Do not inherit fn-155.6's missing provenance or gate evidence as passing.

**Size:** M
**Files:** private scratch copies of `model/framework/Syntax.scala` and compiler directives; `.flow/tmp/fn156/capture/report.md` and linked diagnostic examples.
**Touches:** [.flow/tmp/fn156/capture/**]

### Report handover for independent review

The report-only worker has finished all owned commands. Review the actual report, examples and receipts, not an empty source diff: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/agent/fn156-capture/.flow/tmp/fn156/capture/report.md`, its linked `index.md`, and `task8/handover-summary.md` plus `handover-evidence.json`. This frozen worktree remains at `557ecca8d0a5e5588fb847dc49c8a7384fae54e8` with no tracked implementation changes. The conductor independently reran the artifact verifier: all 30 original probe receipts and input/diagnostic hashes pass, with 14 accepted, 16 rejected and no inconclusive observations. Original worker verification/handover evidence is preserved separately under the conductor's `.flow/tmp/fn156/capture-review/`.

Report acceptance does not adopt capture checking, publish the report, close .7 or waive fn-155 baseline reconciliation. Assess whether the restricted extracted API experiments support the report's bounded conclusions, its required escape/valid-effect coverage, and its explicit full-framework, unchecked-library and lifter limitations. Production compiler/source lane changes are not part of this report.

### Approach

- Record the immutable integrated baseline SHA and compiler version, and create a private disposable worktree/project with isolated compiler caches and outputs. Read copies of shared source; never switch, edit or compile against the concurrently changing integration checkout. Lock any genuine production-sized command via actual flock/fcntl ownership of `/tmp/umpire-heavy-gates.lock`; small isolated probes need not monopolize it.
- Use project research tier gpt-6-astra high for report-only investigation. Re-fetch official pinned-release capture-checking docs. Time-box the entire spike to two elapsed days. Compiler incompatibility and experimental-feature restrictions are legitimate report outcomes, not a reason to change the toolchain silently.
- Exercise valid effects and Draft escapes through return values, retained closures, object fields and nested effects. Record acceptance/rejection diagnostics and minimal examples for each, required API/type changes, soundness limitations and adoption cost. No merged capture source change or adoption.
- Defer a validation stuck beyond one hour across attempts unless it blocks all other available work; preserve logs, unmet experiment coverage and revisit conditions. Deliver the report with available evidence within the overall two-day limit, explicitly identifying any unanswered confinement cases.
- Hand `.flow/tmp/fn156/capture/report.md` plus source/toolchain hashes to .7. Source lane changes affect neither this scratch checkout nor the report; .7 documents the finding's pinned baseline and assesses changed Draft/compiler inputs before publishing.

### Investigation targets

**Required:**
- `model/framework/Syntax.scala:125` - Draft context at the committed baseline.
- `model/framework/Syntax.scala:171` - effect scope.
- `model/project.scala` and `model/irgen/project.scala` - baseline compiler versions/options.
- `.plans/SCALA.md` - prior investigation context, read-only here.

### Quick commands

Run disposable actual-compiler capture compiles with private inputs, recorded start/end timestamps and complete diagnostics. Keep report and examples in the capture-only scratch subtree. Do not merge disposable source changes or touch the null-trial report section.
## Acceptance
- [ ] Report states whether the pinned compiler can prove Draft confinement, with valid-use and all specified escape evidence, limitations and adoption cost; unsupported or deferred cases are explicit.
- [ ] Baseline SHA, compiler version, actual diagnostics and start/end timestamps demonstrate the two-day elapsed bound; report handoff is reproducible.
- [ ] Disposable capture experiments make no shared tracked source/docs change, use isolated inputs/caches and retain any stuck-validation unanswered cases without passing credit.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
