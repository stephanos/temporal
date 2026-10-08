---
satisfies: [R11]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.16 Implement the simulation progress lifecycle owner and remove caller-side accounting

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 5, second half of R11. Implement the design selected in `simulation-progress-design.md` so one private module owns admission, forwarding, delivered-but-unconsumed work, abandonment and participant removal. Callers stop sequencing several accounting calls and the parallel response-barrier state machine disappears.

**Size:** M (if the migration exceeds one session, migrate the coordinator-owned paths first and report the remaining call sites; do not leave two accounting schemes active for the same operation)
**Files:** `tools/gomad3/runner/internal/execution/{simulation_time.go,simulation_unix.go,simulation_model.go,process_unix.go}`, a new private lifecycle file, tests.
**Touches:** [tools/gomad3/runner/internal/execution/**]

### Approach
- Re-read the task 15 design record and characterization tests first. Preserve all valid-behavior test bodies and assertions unchanged; adapt only migration-sensitive fixture wiring. Strengthen the two explicitly named `Preexisting` defect pins into rejection-before-mutation regressions and retain their failures against the old production source, as detailed in Acceptance and the design record.
- Replace the eight external-work methods (`simulation_time.go:299-434`) with the selected lifecycle interface. Callers to migrate: 24 sites in `simulation_unix.go` (`:173-557`) and 3 in `process_unix.go` (`:101`, `:216`, `:220`), plus the `delivered` / `arrived` / `discarded` callbacks wired into `newSimulationModelTransport` (`simulation_model.go:45`).
- Remove `simulationResponseBarrier`, `coordinator.responses` and their helpers (`simulation_unix.go:34-49,151-200,480-560`): the lifecycle owner holds that state. A wrapper that forwards to the old methods one-to-one does not meet R11.
- Unchanged owners: blocking transport and cancellation (`simulationModelTransport`), process lifetime (`process_unix.go`, `supervisor_unix.go`), domain mutation (the model handlers), native timers (runtime). Time-wire encoding from the generated codec stays as delivered by the simulation-time task.
- Invariants to keep: acknowledgement before admission, atomic transfer between participants, rejection of unknown acknowledgement, abandoned response and stale incarnation before any progress or model mutation, committed-versus-uncommitted classification on death, and more than one concurrent blocking operation per participant.
- Process conformance is required in addition to state-machine tests: detached model agreement cannot prove cross-process timing.

### Investigation targets
**Required:**
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`
- `tools/gomad3/runner/internal/execution/simulation_time.go:167-508`
- `tools/gomad3/runner/internal/execution/simulation_unix.go` (655 lines)
- `tools/gomad3/runner/internal/execution/process_unix.go:90-225`
- `tools/gomad3/runner/internal/execution/simulation_model.go`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Simulation'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'Simulation'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,integration ./runner/internal/execution -run 'TestRootProcessSimulationUsesRunnerTransport'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/...
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
```

### Constraints
- Commit each verified task separately under MILESTONES item 5, including its implementation, tests, documentation and Flow records. Preserve unrelated changes; keep unavailable native gates and acceptance open. Do not push without authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual development host is `linux/arm64`, not the inherited `darwin/arm64` assumption. Required native `darwin/arm64` and `linux/amd64` process/runtime gates remain incomplete; stock-host source checks do not qualify them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] One private lifecycle owner handles admission, forwarding, delivered-but-unconsumed work, abandonment and participant removal; callers no longer sequence multiple accounting updates.
- [ ] The response-barrier map and its helpers are removed, with no second bookkeeping structure replacing them outside the lifecycle owner.
- [ ] Task 15's valid-behavior characterization test bodies and assertions pass unchanged, including concurrent blocking operations and cancellation with a late committed response. Only migration-sensitive wiring in `simulation_progress_fixture_test.go` may adapt to the new interface. The two explicitly named `Preexisting` defect pins are strengthened to rejection-before-mutation regressions, with retained failures against the old production source; preserving their confirmed incorrect behavior does not satisfy R11.
- [ ] Unknown acknowledgement, abandoned response and stale incarnation fail before invalid progress or model mutation.
- [ ] Process simulation conformance (`TestRootProcessSimulationUsesRunnerTransport` and the gomad3sim toolchain tests) passes on darwin/arm64 with `-race` clean for the execution package; linux/amd64 is recorded as incomplete.
## Done summary
Blocked:
# Task 16 acceptance blocked after reviewed source implementation

Task 16's atomic lifecycle source is integrated in the current worktree and reviewed by two fresh read-only same-family audits with no source findings. Independent stock Go 1.27.1 linux/arm64 focused, expanded race, 100-repeat, scoped vet, architecture, preservation comparison, make validate and formatting/diff checks passed. Exact hashes and commands are retained in evidence.json, handover.md and conductor-verification.md; commits are empty because the user owns commits.

The five required patched-toolchain Quick commands exit 127 because `.toolchain/bin/go` is absent. The current linux/arm64 development host is neither supported native qualification platform. The exact existing linter is Mach-O CpuArm64 and exits 2 before analysis. Required patched whole-host/Runner, root process transport, gomad3sim toolchain, native timing/isolation and both-platform qualifications remain incomplete. The historical stock broad Simulation child-exit-49 baseline remains attributed to the unchanged unavailable environment, not waived or retried without a cause change.

Formal implementation review is deferred while required gates are unavailable; source audits do not establish SHIP, task completion or R11 closure. D12, resolved D14, the model-delay watchdog and all preservation expectations retain their existing dispositions. Source implementation may advance to task 17 under MILESTONES item 4 after this reviewed predecessor, while native acceptance stays open.

Blocked:
# Task 16 native acceptance remains open after source checkpoint

The exact reviewed atomic lifecycle implementation follows the committed task-15
characterizations in a separate verified-progress commit under MILESTONES item 5.
See conductor-checkpoint.md, checkpoint-report.md, checkpoint-verification.json
and the retained source-audit.md. Nine actual source deltas retain the unchanged
transport and valid test bodies; the two strengthened negatives have original
meaningful old-source RED and corrected GREEN evidence.

All 52 focused source tests and nine inherited architecture checks passed on
stock linux/arm64. The five prescribed patched Quick commands, real root-process
transport and gomad3sim conformance, whole Runner/host and both native platform
gates remain incomplete. The existing Mach-O linter is unavailable here. These
stock results do not prove native timing, hard isolation or exact replay. The
historical broad stock child-exit-49 failures were not retried or weakened.

Keep task 16 acceptance and R11 open. No Flow completion or push is included.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
