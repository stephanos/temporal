# Task 40 standards and plan-fidelity source review

Assessment: SOURCE_PROGRESS_COMMIT_ONLY

Introduced actionable findings: 0. Worst introduced severity: none.

This fresh read-only review covers the five admitted task40 files against BASE `29e12b5e473de0aa4e7b3df4db8626d421ed8891`. HEAD equals BASE on branch `gomad`; the source progress is an uncommitted diff. All five files match `logs/source-freeze.sha256`. The review axis is standards and plan fidelity. This assessment supplies no formal SHIP verdict, task completion, or merge-readiness claim.

Requested reviewer routing is `gpt-6.1-sol` at high, from the same model family as the requested writer. Execution metadata did not expose an actual model identifier to this review; the requested selector does not prove executed identity.

I read AGENTS.md, flowctl usage, task40 and the relevant parent specification through flowctl, MILESTONES.md, the Gomad README, the retained task9 research/admission/probe documentation, the complete admitted diff, surrounding command/capture code, default consumer construction, generator input owners, and the worker handover and retained logs. I ran only read-only inspection, source-hash verification and scoped `git diff --check`; I ran no tests, cache-mutating commands, source edits, commits, or Flow lifecycle writes.

## Strengths

- Scope and ownership fit the admitted task. `internal/hostexec/command.go:25` adds one opt-in request field and `:40` retains the original command cause. `target/internal/gocommand/command.go:96` owns its private compatibility projection through the existing New seam. The change reuses the existing captures, pipes, process-group cleanup and Wait owner. It adds no subprocess framework, global hook, public target API, new policy grant, dependency, pin, generated source, or adapter-publication change.
- Default behavior stays isolated in source. Existing callers omit PreserveCommandError. The unchanged Structured, Diagnostic, execute and exitError bodies keep their existing cancellation, timeout, synthetic exit and capture contracts. The false branch retains command/Dir validation, effectiveTimeout, exec.Command, wrapped Start errors, existing group cleanup and Wait classification. The real opt-out control at `internal/hostexec/command_unix_test.go:21` checks that the added CommandError field remains nil for a default request.
- The consumer inventory covers toolchain build/patch, gomadtool and conformance, qualification set/soak, upgrade, adapter regeneration and pinimpact. Their existing Request literals leave the flag false and retain their own result/error projection. Existing target preparation/listing callers still select Structured or Diagnostic. The sole production flag assignment is the new Compatibility operation. The handover's focused consumer logs record applicable checks separately from the accidental broader conformance selection that required the absent patched toolchain.
- Capture and error projection follow the specified order at `target/internal/gocommand/command.go:106`. Infrastructure errors win, then stdout overflow, stderr overflow, watchdog and raw command outcome. Complete bounded stderr survives when available; stdout is assigned only after those rejection paths. Positive capacity comes from the request, without fixing task9's proposed 4 MiB listing bound. The 15-minute default is tested against the independent literal at `command_test.go:248` and disclosed as an added lifetime bound in task40 and the handover. The separate real one-second watchdog case retains its typed wrapper and original termination cause.
- Original command causes cross the owner boundary directly. Start failures retain the returned object at `internal/hostexec/command_unix.go:84`, and Wait failures retain waitErr at `:163`. There is no rebuilt ExitError or formatted-error parsing. The after-start branch kills and waits for the leader before shared descendant TERM cleanup. Tests check the actual ExitError PID, SIGKILL, ExitCode -1, stderr, descendant liveness and GroupGone. Ordinary SIGTERM and late cancellation have separate controls.
- The directory preflight at `internal/hostexec/command_unix.go:72` is a justified compatibility adjustment. Pinned Go1.27.1 `src/os/exec_posix.go:25` performs Stat only when SysProcAttr is nil; Setpgid suppresses it. The opt-in code calls the actual Stat, retains that PathError object, assigns the same chdir operation and respects prior Cmd.Err, empty-command and canceled-context precedence. Its one new comment explains this otherwise non-obvious upstream interaction. The expanded invalid-directory failure is retained honestly as RED before the correction.
- Both test diffs are additive, with 19 and 239 added lines and zero deleted lines. Existing comments, assertions and test bodies remain intact. New expectations use independent literals for errno, operations, stream names, diagnostics, signals and the default timeout. The pointer comparisons check projection of actual owner results. The injected cleanup test is explicitly scoped to seam precedence rather than presented as genuine OS cleanup-fault proof.
- Generator ownership is preserved. The protocol generator's explicit choice/live-capability inputs at `internal/gomadtool/generation/protocol/protocol.go:564` and `:594` exclude both admitted owners; version/boundary generators consume their existing descriptors, manifests and overlay/patch sources. No new hash or generated artifact update is justified by this diff. Retained make-validate output checks those boundaries and existing pack/profile/manifest freshness.

## Important introduced issues

None found on this axis.

## Minor introduced issues

None found on this axis.

## Preexisting observations and evidence limits

The final unfiltered lint remains red with the baseline's four unchecked pipe-Close defers (`internal/hostexec/command_unix.go:47`, `:48`, `:53`, `:54`) and old test-helper sleep (`command_unix_test.go:213`). The intermediate two introduced QF1008 findings are fixed in the frozen source. These five inherited residuals are not introduced standards findings and do not make the original lint gate pass.

The retained focused/architecture/errortype/format/validation and affected-consumer evidence supports source progress on developmental linux/arm64. It does not replace the original predecessor/matched-baseline, full/default/functional/affected-native, formal, or native Darwin requirements. Broader exact-input adapter regeneration remains unproved. No actual infrastructure/pipe/group cleanup fault was injected; the cleanup seam proves projection only. The completion control cancels after real Run returns and does not establish every simultaneous Wait/cancellation/watchdog race. Deeper execution races belong to the correctness review axis.

The public helper remains unchanged. Its 29-case probe preserves the existing helper controls and supplies no task9 integration, nonempty source selection, pin reproduction, or admitted listing-size proof. Task9 retains its task8 dependency and helper integration; fn113/task21 retain pins, publication and aggregate qualification. Root retains Flow, review, evidence and commits. Native Linux qualification remains with fn128 and nonblocking. The source-progress assessment waives none of task40's original owned requirements.
