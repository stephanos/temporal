# Task 40 source progress

Task fn-109-gomad-deepen-modules-and-tool-interfaces.40 remains in_progress. The private Compatibility operation now preserves real command errors and bounded diagnostics over the shared hostexec capture and process-group owner. Root owns review, commits, evidence archiving and lifecycle.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer gpt-6.1-sol/high retained.
stage: impl-review - skipped(policy: host-deferred; root owns review and original owned gates are red or unproved)

Base commit is 29e12b5e473de0aa4e7b3df4db8626d421ed8891. HEAD still equals that commit. This lane created no commits. The five admitted source/test files carry uncommitted source progress. Root's MILESTONES.md edit and unrelated .turbo notes remain intact.

## Mechanism and compatibility

Request.PreserveCommandError opts into legacy startup/context precedence. Result.CommandError separates the original Start/Wait cause from infrastructure failures returned by Run. Default requests leave that field nil. Structured, Diagnostic, execute and exitError are unchanged. Compatibility projects infrastructure errors first, then stdout overflow, stderr overflow, the finite watchdog, and the original command error. It returns complete RawBytes on successful structured output and returns no stdout on failure or overflow. Complete stderr within capacity survives raw failures and stdout overflow.

Caller cancellation uses exec.CommandContext with its original lookup/start error ordering. After startup, Run kills and waits for the leader before shared descendant termination can send SIGTERM. Actual exec.ExitError objects retain their ProcessState, PID, SIGKILL 9 and ExitCode -1. Ordinary SIGTERM remains signal 15. Completed command errors survive late context cancellation. The Compatibility operation accepts any positive addressable output capacity. Timeout zero uses the existing 15-minute finite watchdog. WatchdogError unwraps the actual termination cause and does not match caller cancellation/deadline sentinels. A one-second real-process control exercises watchdog termination separately; a seam test verifies the 15-minute default request.

The process-group configuration exposed an upstream difference. Go1.27.1 os.startProcess performs a directory os.Stat preflight only with nil SysProcAttr. Setpgid suppresses it, changing an invalid directory from chdir PathError to fork/exec PathError. The opt-in branch mirrors that preflight after Cmd.Err, nonempty command and caller-context checks. It retains the original os.Stat PathError object and applies the same upstream Op assignment. Other startup errors come directly from command.Start. The test initially failed on the unexpected fork/exec operation, and passed after this adjustment. Relevant pinned source is src/os/exec_posix.go lines 25-37 and src/os/exec/exec.go lines 667-727. Hashes are in logs/tool-config.sha256. No formatted error parsing, stripped startup prefix, or reconstructed ExitError exists. Concurrent directory mutation between preflight and Start retains the upstream race limitation; these controls do not establish race-complete equivalence.

## Test evidence

All commands ran from tools/gomad3 with cached stock Go1.27.1 linux/arm64 on PATH, GOENV=off, GOWORK=off, GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, GOFLAGS empty, and GOMADSEED/GOMAD3_CHILD_SEED/GOMAD3_SEED unset. Each suite used a foreground timeout 600 call with output captured once. No command remains running.

| Observation | Exit | Seconds | Retained log under logs/ |
| --- | --- | --- | --- |
| Baseline focused Quick command | 0 | 1 | baseline-focused.log |
| Baseline architecture Quick command | 0 | 1 | baseline-architecture.log |
| Baseline unfiltered lint Quick command | 1 | 1 | baseline-lint.log |
| Initial behavioral RED on existing Structured path | 1 | unmeasured | red.log |
| First opt-in focused pass | 0 | unmeasured | green-first.log |
| Expanded invalid-Dir regression | 1 | unmeasured | green-expanded.log |
| Directory compatibility correction | 0 | unmeasured | green-fixed-dir.log |
| Final controls before promoted-Pid cleanup | 0 | unmeasured | final-focused.log |
| Final architecture before promoted-Pid cleanup | 0 | unmeasured | final-architecture.log |
| Intermediate unfiltered lint | 1 | unmeasured | final-lint.log |
| Intermediate errortype | 0 | unmeasured | final-errortype.log |
| Frozen final focused Quick command | 0 | 4 | final-fixed-focused.log |
| Frozen final architecture Quick command | 0 | 2 | final-fixed-architecture.log |
| Frozen final unfiltered lint Quick command | 1 | 0 | final-fixed-lint.log |
| Frozen final errortype | 0 | 0 | final-fixed-errortype.log |
| Complete make validate | 0 | 5 | validation.log |
| Default toolchain focused controls | 0 | 1 | consumer-toolchain.log |
| Initial conformance selection | 1 | 0 | consumer-conformance.log |
| Applicable conformance default-consumer controls | 0 | 0 | consumer-conformance-focused.log |
| Qualification set/soak | 0 | 2 | consumer-set-soak.log |
| Upgrade focused controls | 0 | 0 | consumer-upgrade.log |
| Pinimpact GoResolver controls | 0 | 0 | consumer-pinimpact.log |
| Adapterregen input-independent controls | 0 | 1 | consumer-adapterregen.log |
| Unchanged public helper final probe | 0 | 2 | public-helper-controls.jsonl and .err |
| BASE/final normalized public helper comparison | 0 | unmeasured | public-helper-control-diff.log |

The initial RED used two unchanged real-process assertions against the existing Structured path, then switched only their operation call to Compatibility after implementation. Exit 7 was a synthetic gocommand.ExitError; missing bare PATH lookup was preempted by working-directory validation. These are behavioral REDs, not missing-symbol compile failures. The expanded test's directory failure was also retained, despite the historical green-expanded filename. The new cases added later are contract controls; only the two initial cases and directory case have retained RED observations.

Real-process tests cover raw exit/stderr, actual PathError operation/path/errno, empty command, missing bare PATH versus absolute command under canceled/expired contexts, invalid directory with those contexts, inherited/dot/relative cwd, ./ executable resolution, actual stdout/stderr/dual overflow, cancel with stdout and stderr overflow, acknowledged after-start cancel/deadline, finite watchdog, and TERM-ignoring descendants. Assertions verify leader ProcessState and descendant death. The completed SIGTERM control cancels context only after actual Run returns and keeps the original error object. Cleanup plus cancellation/watchdog/capacity precedence uses the existing New seam; it does not claim an injected flag proves OS cleanup failure. The default hostexec raw-error opt-out has its own real exit control.

The final lint result is red with exactly the original four unchecked pipe Close defers and existing test helper time.Sleep. An intermediate lint result additionally found two introduced QF1008 selectors. Those were corrected with the promoted Pid method; final assertions retain real ProcessState PID checks. No existing assertion or comment was removed. Both test diffs have zero deleted lines, retained in logs/tests-additive.diff. No suppression, rule/pin change or filtered lint gate ran.

The initial conformance selection accidentally matched TestRuntimeOwnedControlProbe, which needs absent .toolchain/bin/go. That actual failure remains recorded. A separate focused consumer-control selection names applicable Run/Resolve cases and passed. It supplies focused regression evidence and does not turn the missing-toolchain conformance gate green. The 14 selected toolchain and six selected upgrade tests exist in source; no observed suite was an empty selection.

The public helper final probe used the unchanged archived probe.go. Its 29 observations match the archived conductor BASE after removing volatile paths/PIDs and retaining case name, digest, error-chain types, errno, signal/status, context identities and lifetime predicates. All 18 observed children and their GOPATHs were gone. This exercises the still-unmigrated public helper and proves preserved default helper behavior. Its empty-source JSON supplies no platform selection, pin, largest-listing size or task9 integration proof. The opt-in mechanism tests supply separate bounded-stream and descendant evidence.

## Ownership and bindings

The read-only consumer survey found default hostexec usage in toolchain/build.go and patch.go, conformance/driver.go, qualification/set and soak, upgrade, adapterregen and pinimpact. Those callers remain unchanged. The hostexec owner has no new cross-owner import; target to hostexec is already permitted. Neither admitted package is in current Makefile generator inputs or protocol implementation input lists. make validate checked version, protocol, boundary, compiler fixtures, patch/overlay and script ownership, compatibility packs/profile, and qualification-manifest freshness without regenerating output.

logs/source-freeze.sha256 binds the five final files and was rechecked successfully. logs/base-source.sha256 binds each file at the base commit. logs/tool-config.sha256 binds pinned go/gofmt, lint and errortype binaries, .github/.golangci.yml, nested go.mod/go.sum and the relevant pinned standard-library sources. Tool versions are retained in go-version.log, lint-version.log and errortype-version.log. git diff --check passed; protected config/module/pin/public-helper paths have zero diff. No source writer or cache writer ran concurrently with this lane. A read-only research scout supplied consumer/generator paths; it ran no commands that mutate caches or files.

## Required acceptance still unproved

Task completion remains unavailable. Actual lint is red. This lane did not run the complete original patched-toolchain/full/default/functional/smoke/affected-native acceptance, predecessor full-gate baseline, or native Darwin qualification because the host is linux/arm64 and the patched toolchain is absent. Independent source review and formal implementation review remain root-owned and unperformed by this lane. Existing native and missing-toolchain failures were not reclassified as passes. Static both-source-set architecture evidence is covered by the architecture Quick command; broader original qualification remains required. Full adapter regeneration requires exact pinned cached Sentry inputs, which were not proved available; only input-independent controls ran. No full 15-minute elapsed watchdog run or forced real infrastructure-cleanup failure was performed. The one-second termination test and injected projection precedence have their stated scope.

Task9 retains helper integration, task8 dependency, measured listing capacity selection, nonempty source-set selection and pin proof. fn113 and task21 retain their original publication and aggregate qualification. Transferred native Linux qualification remains owned by fn128 and nonblocking. No original requirement has been waived.

Defect route:
- prior fixes: frozen admitted task39 source and current git history read; no other source writer admitted. Remote PR/issue ownership unchecked because root owns admission and this lane did no external tracker operations. Memory search surfaced existing module-download/cache cautions.
- diagnosis: initial real Structured-path RED confirmed synthetic exit conversion and early generic validation; expanded real test confirmed Setpgid changed upstream directory preflight.
- introduced by: not bisected; this lane adds the explicitly admitted mechanism and worktrees/history changes are forbidden by dispatch.
- base: initial behavioral RED at 29e12b5e473de0aa4e7b3df4db8626d421ed8891; head: frozen final focused controls pass, unchanged public helper 29-case comparison passes.
- live: no live app surface; real subprocess fixtures exercised the library boundary.
