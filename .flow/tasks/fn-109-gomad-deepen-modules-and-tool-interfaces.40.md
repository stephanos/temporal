---
satisfies: [R10, R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.40 Preserve adapter command errors through bounded execution

## Description
Implement the bounded compatibility mechanism consumed by task9's omitted adapter source-set listing (R10/R18/R19). Task9 retains helper routing and aggregate qualification. This owner does not duplicate adapter pins/regeneration/publication under fn113.

**Size:** M
**Files:** internal hostexec request/outcome, Unix execution and its tests; existing private gocommand operation and tests.
**Touches:** [tools/gomad3/internal/hostexec/command.go, tools/gomad3/internal/hostexec/command_unix.go, tools/gomad3/internal/hostexec/command_unix_test.go, tools/gomad3/target/internal/gocommand/command.go, tools/gomad3/target/internal/gocommand/command_test.go]

### Approach

- Admit one source/cache writer from the frozen task39 reviewed commit already integrated at 0dc87045ca98912a8942894d63e3af6b575fb2a2. Root owns Flow lifecycle, review, evidence and commits. Read the retained task9 source research and the actual BASE process probe before production edits; revalidate every source/tool binding at HEAD.
- Reuse the existing hostexec execution/capture/process-group owner and gocommand.New injection seam. Add one narrowly opted-in compatibility operation, keeping default Request behavior and existing Structured/Diagnostic error, cancellation, timeout, overflow and capture contracts unchanged. No second subprocess framework, global hook, public target API, shared pin or third-party dependency is admitted.
- Preserve original Start/Wait errors and their concrete objects, ordered unwrap shape, os.PathError operation/path/errno, exec.Error and actual exec.ExitError ProcessState. Reconstructing an ExitError from integers or parsing/stripping formatted Start strings is insufficient. Execution outcomes and infrastructure/cleanup failures must be distinguishable without changing default consumers.
- Pin BASE startup precedence with real subprocess fixtures. Missing bare PATH commands precede canceled/expired context through exec.Error/ErrNotFound; absolute missing commands with those contexts retain the context sentinel. Empty commands retain exec: no command; empty Dir inherits cwd; relative Dir and ./ executable resolution retain their supplied spelling and meaning. Do not move generic validation or an unconditional ctx.Err check ahead of the legacy boundary for the opted-in operation.
- After an acknowledged started marker and stderr emission, BASE cancel and deadline return an actual exec.ExitError with SIGKILL and ExitCode -1, and do not match either context sentinel. Preserve the leader's immediate-kill/Wait behavior before descendant cleanup can deliver TERM. Ordinary SIGTERM remains SIGTERM, ordinary nonzero exits remain raw exit errors, and an already completed outcome must not be overwritten solely because context later becomes canceled. Fresh process/group evidence, not fake flags, must prove termination.
- Keep both output streams bounded through existing captures. Successful structured data requires complete RawBytes. Failed execution or overflow returns no decodable stdout and retains complete bounded stderr when it fits. Infrastructure/cleanup failures win; afterward reject stdout overflow before stderr overflow before projecting a raw command outcome. Cover execution plus overflow, context plus overflow, dual overflow and cleanup plus context/capacity precedence. Never decode a plausible truncated JSON prefix.
- This operation accepts a positive request capacity rather than fixing task9's listing capacity here. Task9 must retain measured admitted-listing sizes before selecting its proposed 4 MiB bound. Reuse the finite 15-minute no-deadline watchdog and disclose it as an added R10 lifetime bound. The new watchdog must remain distinguishable from legacy caller cancellation/deadline; characterize any bound-triggered kill explicitly.
- Add behavioral RED cases before production changes and preserve the BASE controls unchanged for final replay. The probe covers 29 ordinary cases on developmental linux/arm64 stock Go; its empty-source JSON proves neither platform file selection nor pin reproduction, and it exercises no overflow or descendants. Extend controls for both newly bounded streams and real descendant termination. Preserve all existing comments, fixture assertions and protected source outside the admitted surface.
- Review default consumers in toolchain build/patch, gomadtool/conformance, qualification set/soak, upgrade, adapter regeneration and pinimpact. Run focused existing consumer controls where inputs are available; record unavailable exact inputs instead of repeating unchanged missing-toolchain/native-host failures. The declared surface is absent from current generator input lists, but re-anchor that ownership and run required validation.
- Root commits independently reviewed source progress before considering task9 integration. Task9 still depends on task8 and this owner; source progress does not make a dependency-blocked task ready. Task40 completion covers only its shared mechanism, default-consumer preservation and owned verification; it requires no task9 helper integration or task9 completion. Task9/fn113/task21 retain adapter integration, nonempty source-set/pin proof and aggregate R10/R18/R19 qualification without any waiver. This owner's predecessor/matched-first-baseline, full/default/functional/affected-consumer/formal/native Darwin and static both-source-set requirements remain required wherever unproved. Linux execution belongs to fn128 and is nonblocking.

### Investigation targets

**Required:**
- tools/gomad3/internal/hostexec/command.go:17-78
- tools/gomad3/internal/hostexec/command_unix.go:23-119,146-175,294-316
- tools/gomad3/target/internal/gocommand/command.go:70-124
- tools/gomad3/internal/hostexec/command_unix_test.go
- tools/gomad3/target/internal/gocommand/command_test.go
- tools/gomad3/target/internal/capabilityreview/list.go:109-121
- .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/adapter-command-gap-2026-10-05/

### Quick commands

From tools/gomad3 with cached pinned stock Go1.27.1 first on PATH, GOENV=off GOWORK=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOFLAGS= and all Gomad seed variables unset:

```sh
go test -count=1 -tags test_dep ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview
go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$'
/tmp/fn109-lint-tools.ZdNe1t50/golangci-lint-v2.13.0 run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./internal/hostexec ./target/internal/gocommand ./target/internal/capabilityreview
```

Retain actual unfiltered lint baseline/final and every residual; no suppression, filtered gate, rule/pin change or whole-scope subtraction. Add frozen source/tool/config command bindings, logs/exits, focused consumer controls, errortype, formatting and generator checks. Independent source review covers default-consumer stability and every unexecuted path. Formal implementation review and task completion require all required source-owned gates to pass.

### Current source progress

The five-file command compatibility candidate and C1/C2 corrective regressions have fresh source-progress reviews with zero introduced findings. Conductor focused tests pass with 56 test results and no failures/skips; both corrective controls pass 25 repeated runs. Applicable architecture, race, errortype, formatting, validation and consumer controls pass. Actual unfiltered lint remains red with its five inherited findings. Original source-owned acceptance remains open.

Retained evidence and typed blockers are in [task40 progress](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-40/command-compatibility-2026-10-05/progress.md). Task9 retains its original dependencies and helper integration. No aggregate R10 pass or native qualification is claimed. Transferred Linux execution remains with fn128 and is nonblocking.
## Acceptance
- [ ] Real BASE/final controls preserve startup/context/cwd/relative executable ordering, exact errors and actual Start/Wait objects, stderr and ProcessState. After-start cancel/deadline preserve SIGKILL/ExitCode -1 without context-sentinel replacement; completed outcomes retain their original precedence.
- [ ] The opted-in operation shares existing bounded capture and process-group ownership; actual stream overflow and real descendant termination tests pass. Infrastructure/cleanup, stdout overflow, stderr overflow and raw-outcome precedence is verified; failed/truncated stdout cannot be decoded. The finite watchdog is disclosed and tested separately from legacy caller cancellation.
- [ ] Default hostexec and Structured/Diagnostic contracts remain unchanged. Focused existing affected-consumer controls, architecture, errortype, formatting and applicable generator validation pass on frozen sources. Actual unfiltered lint baseline/final and any unavailable inputs/residuals are retained without suppression or pin changes.
- [ ] A fresh independent review finds no actionable introduced defect; root commits source, tests and owned evidence before the helper writer. Task9 retains adapter integration and task21 retains final qualification. No public API, grant, module/pin or approval/publication policy changes.
- [ ] Task40's mechanism-owned R10/R18/R19 contracts and required predecessor/matched-first-baseline, full/default/functional/affected-consumer/formal/native Darwin and static both-source-set verification are proved before its completion. Missing/red source-owned gates keep this owner open. Aggregate adapter integration, nonempty source-set and pin evidence and complete R10 fulfillment remain with task9/fn113/task21; task40 completion requires no task9 integration or completion and claims no aggregate R10 pass. Those owners retain every original requirement. Transferred Linux execution under fn128 is nonblocking.
## Done summary
Blocked:
Task 40 has reviewed command-compatibility source progress with C1/C2 corrected. The frozen evidence and zero-introduced-finding source reviews are retained in .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-40/command-compatibility-2026-10-05/progress.md.

Required source-owned acceptance remains red or unavailable. Actual unfiltered lint still reports four inherited unchecked pipe Close defers and one inherited test-helper sleep. Predecessor/matched-first-baseline, complete original patched/full/default/functional/smoke/affected-native gates, native Darwin qualification and full exact-input adapter regeneration remain unproved. Formal implementation review is withheld until those gates are green. This developmental linux/arm64 host has no patched runtime.

Revive this task when a lawful correction for the inherited lint failures is admitted and the original source-owned baseline/qualification inputs become available on native Darwin. Do not rerun unchanged missing-toolchain or native-host failures. Source progress does not unblock dependency-gated task 9 or waive its helper integration and aggregate requirements.

Transferred Linux execution remains under fn-128 and is nonblocking. No formal SHIP, task completion, actual OS cleanup-fault proof or aggregate R10 pass is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
