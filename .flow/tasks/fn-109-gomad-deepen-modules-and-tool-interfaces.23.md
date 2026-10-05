---
satisfies: [R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.23 Repair module-aware lint routing and supply the nested host gates

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Repair the confirmed qualification-tooling defect returned by fn-109.21. This is
a separate R19 implementation owner, not an edit under task 21's verification
scope or task 20's guidance ownership. The predecessor CLI progress checkpoint
is b43aeb5b15d438eebab65f2ce48eecb19e76e55c. Original R18/R19 and native gates stay
unchanged; this task advances the required lint gate rather than closing them.

**Size:** M
**Touches:** [Makefile, cmd/tools/lintcode/**, tools/gomad3/Makefile, .github/workflows/linters.yml, .github/workflows/gomad3.yml, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-23/**]

### Cause and delivery

Root Makefile's changed-path selector passes every changed filesystem directory
to root-module golangci/errortype. The retained task-21 Linux diagnostic actually
failed with golangci exit 7 and Make exit 2 on nested overlays and conformance
fixtures. Correct package/module ownership before analysis; keep the comparison
revision, pinned tools, lint rules and failure propagation unchanged.

Provide reproducible checks for ordinary Gomad host source and mixedbrain's live
nested module, from their own module directories. Include Gomad's root harness
and direct ./toolchain host package; recursive ./toolchain/... would admit
standard-library overlays. Reuse existing host package scope and pinned tools.
Wire the appropriate nested checks into existing CI rather than relying on a
manual caller to remember them. A local workflow edit is not a CI execution.

Keep ordinary server packages, tools/gomad3sim and tools/gomad3integration in
root lint. Preserve .github/actions/build-docker-images/scripts as live root
tooling. Classify only actual evidence, overlays and known fixtures; unexpected
modules or uncovered source must fail visibly. Reuse existing classifications
where feasible; avoid a generic module/plugin framework or broad path skips.

Exact root fixtures are tools/gomad3sim/testdata/simulation_exploration and
tools/gomad3integration/testdata/tagged, with their existing Runner/capability
and integration gates. tests/mixedbrain is an ordinary nested integration
harness, not an excluded compiler-negative fixture. Gomad's independent
fixture/corpus modules and standard-library overlays retain their existing
qualification owners. Do not blanket-lint compiler-negative fixtures.

### Verification

Baseline a controlled behavioral reproduction of the actual existing selector
before source edits; reuse the unchanged-source real lint failure instead of
rerunning its 147-second environment-independent loader failure. Add executable
regressions first, retain meaningful RED/GREEN outcomes, and test real Git/path
classification and Make dispatch boundaries, not source-text presence.

Cover root+nested+hidden live source, known fixtures/evidence/overlays, unknown
modules and overlay prefix near-misses, tracked/untracked/deleted/renamed paths,
bad comparison revisions, and failure propagation from nested checks. Derive
expected module/package ownership independently of the selector. Isolate test
fixtures and processes; use the cheapest checks that prove these contracts.

Run the new helper tests with -tags test_dep, affected existing Make ownership
tests and make -C tools/gomad3 validate. After freezing source, run the actual
root fast gate and the reproducible nested gates with compatible pinned Linux
binaries and GOLANGCI_LINT_FIX=false. Retain commands, cwd, environment, source
identity, raw terminal results and durations. Remaining current-rule findings
are real failures: report them and their owners, not filtered success. Do not
mass-fix unrelated source, relax config, select a newer baseline or fabricate
native qualification to obtain a green check.

### Constraints and handoff

Read AGENTS.md, tools/gomad3/README.md, MILESTONES, the original fn-109 spec,
the task-21 lint diagnostic and actual Make/CI/config/classification source.
Follow systematic debugging, TDD, code-style and verification skills. Add no
third-party dependency or version upgrade; preserve generated/runtime inputs,
CLI behavior, recorded identities and all existing qualification dispositions.
The existing ^.git lint regex may exclude .github reporting; keep that policy
limitation visible without changing unrelated lint rules.

Root is sole Git/Flow/review owner under the existing source-progress override.
Implement only within declared Touches; ask root with concrete evidence if a
different helper surface is necessary. One checkout writer; delegate independent
read-only scouting where useful. No push, worktree, stash or history rewrite.
Never modify old evidence or gates to manufacture a pass. Formal review runs
only on a green tree; return actual source progress and typed gaps if gates stay
red. Native darwin/arm64 and linux/amd64 qualification remains separately open.
Retain a small handover and evidence under task-23; root commits verified progress
before starting the next implementation task per MILESTONES instruction 5.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Behavioral regressions reproduce the old wrong-owner dispatch and prove the repaired root, Gomad and mixedbrain routes, exact fixture/evidence/overlay dispositions, hidden live tooling, unknown-source rejection and tracked/untracked/deletion/rename behavior.
- [ ] Root fast lint preserves its existing comparison and lint policy; ordinary root packages remain checked and a nested failure propagates as failure.
- [ ] Ordinary Gomad host and mixedbrain lint have reproducible pinned-tool/config targets from their own modules, with no overlay or compiler-negative fixture loading and no silently omitted ordinary host source.
- [ ] Existing CI locally invokes the appropriate nested gate; workflow source is verified without claiming an unexecuted CI result.
- [ ] Helper tests, affected Make ownership tests, generated validation and the actual scoped lint commands have frozen-source terminal evidence. New code is clean; unrelated baseline findings stay visible and leave the affected qualification open.
- [ ] Independent review verifies the source correction; original reports, criteria, qualified workload expectations and native gates remain unchanged. Commit this task's verified source progress and Flow evidence before the next implementation task.


## Done summary
Source progress only; acceptance remains open. Module-aware root/nested lint routing and local CI commands are implemented and independently source-reviewed with no actionable findings. Helper contracts, unfiltered helper lint, errortype, Make ownership and generated validation passed. Actual root/Gomad lint remains red; inherited path-policy repair needs a separate R19 owner. Mixedbrain and the separately retained exact tagged integration batch passed within their documented scopes. Native gates, D5 and original R18/R19 remain open.

The worker handover/evidence are pre-lifecycle snapshots. See task-23/acceptance-open.md for terminal results, source binding and qualification owners. Root commits this verified progress before admitting the next implementation task.

stage: impl-review - skipped(policy: qualification tree red; independent source-progress review is not formal SHIP)
stage: plan-sync - skipped(empty: task not done; no completed wave to project)

## Evidence
- Commits: root source-progress checkpoint carrying this record; not a task completion receipt.
- Tests: task-23/evidence.json and integration-tag-correction/verified/ command receipts; independent-source-review-checks.json.
- Review: task-23/independent-source-review.md, SOURCE PROGRESS COMMIT ready; no formal SHIP.
- Acceptance: task-23/acceptance-open.md; task remains blocked on qualification.
- PRs: none.

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
