# Task 40 correctness source-progress review

Assessment: NEEDS_FIX. Two Important introduced findings prevent the source-progress commit on this frozen candidate. This assessment covers correctness of the five admitted files only. It supplies no formal implementation SHIP, task-completion, or merge-ready verdict.

## Scope and evidence

BASE and HEAD were both `29e12b5e473de0aa4e7b3df4db8626d421ed8891`. The five working-tree file hashes matched `logs/source-freeze.sha256` during review. I read repository instructions, `flowctl usage`, task 40 through `flowctl cat`, the five-file diff and surrounding execution/capture code, retained BASE probe source/results documentation, and the completed handover and logs. I ran no tests, cache commands, source edits, commits, or Flow writes. Findings below come from inspection of the candidate and pinned Go1.27.1 standard-library source, not executed reproductions.

The dispatch requested writer and reviewer `gpt-6.1-sol` at high effort, from the same model family. Execution metadata proving the actual model was not supplied, so this report records the requested routing only.

Retained executed evidence includes passing `final-fixed-focused.log` and `final-fixed-architecture.log`, plus the handover's recorded exit 0 for errortype, validation, formatting, and applicable focused consumer controls. Final unfiltered lint remains exit 1 with five inherited findings. The unchanged public-helper probe retains 29 matching ordinary cases and supplies preservation evidence for that helper; it does not exercise task 9 integration. None of the two edge cases below has an executed reproduction in the evidence reviewed.

## Strengths

- The opt-in request and result reuse the shared capture and process-group owner. Default request validation, effective timeout, Structured, and Diagnostic behavior remain on their existing paths.
- Ordinary startup and exit failures retain concrete error objects. The directory preflight at `tools/gomad3/internal/hostexec/command_unix.go:72` checks lookup/empty-name/context precedence, keeps the actual `os.Stat` PathError, and applies upstream's `Op = "chdir"` assignment. The pinned `os/exec_posix.go:25` confirms that Setpgid suppresses the upstream preflight. There is no reconstructed ExitError or formatted-string parsing.
- The normal cancellation path kills and waits for the leader before descendant TERM cleanup. Real retained tests inspect SIGKILL, exit code -1, PID, descendant death, stream overflow, and stderr preservation. Compatibility returns stdout only after successful outcome and complete-stream checks, with stdout overflow preceding stderr overflow.

## Critical introduced findings

None.

## Important introduced findings

### C1. Non-exit Wait failures are demoted from infrastructure errors

Location: `tools/gomad3/internal/hostexec/command_unix.go:162`, with projection consequences at `tools/gomad3/target/internal/gocommand/command.go:110`.

Preserve mode assigns every `waitErr` to CommandError and unconditionally clears `classificationErr`. That bypasses the existing distinction between an unsuccessful process exit and an I/O failure. A supported concrete case is an opted-in Request with a Stdin reader that returns a sentinel read error, while the child reads to EOF, prints output, and exits 0. Go1.27.1 `os/exec/exec.go:563` copies the reader and `:962` returns its error from Wait when the process otherwise succeeds. The new branch returns nil infrastructure error and a successful Termination/ExitCode, with the I/O failure stored as CommandError. A projection therefore treats this infrastructure failure below stream overflow.

The current Compatibility request has nil Stdin, which bounds the immediate adapter exposure. The shared opted-in hostexec Request still accepts Stdin, and task 40 explicitly requires infrastructure failures to remain distinguishable from command outcomes. This is an introduced classification defect in that owner, not an assertion that the existing adapter normally supplies stdin.

Fix: preserve actual `*exec.ExitError` objects as command outcomes, but return non-exit Wait failures through Run's infrastructure-error return while retaining the original cause object. Add a real failing-reader control, including output overflow, to establish that infrastructure still wins.

### C2. A late watchdog can relabel an already completed nonzero or signal outcome

Location: `tools/gomad3/internal/hostexec/command_unix.go:125`, with the wrapper at `tools/gomad3/target/internal/gocommand/command.go:119`.

The timeout branch sets WatchdogTimeout, but clears it only when `waitErr == nil`. If the process has already completed with exit 7 or SIGTERM and the timer wins the select, Process.Kill returns `os.ErrProcessDone`. Receiving the original nonnil Wait result then leaves WatchdogTimeout true, and Compatibility returns WatchdogError instead of the completed raw outcome. The watchdog did not induce that termination.

This has a bounded concrete control without fabricated result flags. Supply a Stdin reader whose copy remains blocked briefly after a real child exits 7, then release it after the configured watchdog fires. Pinned Go1.27.1 `os/exec/exec.go:948` records ProcessState before `:962` awaits copying, so the leader is done while the hostexec waits channel remains pending. Kill observes ErrProcessDone; the eventual real Wait error remains exit 7. The same logical ordering can occur when a ready waits result races the timer in the select. This review establishes that ordering from source, without claiming it was executed.

Fix: treat ErrProcessDone from the leader kill as an already completed outcome and clear the watchdog classification even when Wait returns a nonzero/signal error. Retain the actual Wait object and the ordinary descendant cleanup. Add the real bounded control above; retain the existing genuine watchdog SIGKILL control.

## Minor introduced findings

None.

## Pre-existing observations and remaining acceptance

The four ignored pipe Close defers and the historical test-helper sleep remain the five unfiltered lint findings; they are not introduced findings in this review. Shared cleanup helpers already contain more complex fallback Wait ownership, so this report does not attribute all historical cleanup concerns to task 40. Directory mutation and cancellation during startup remain timing-sensitive; the fixed preflight proves the specified ordinary ordering and object shape, not universal race equivalence.

Original full/default/functional, predecessor/matched-baseline, formal review, affected inputs, and native Darwin requirements remain open as disclosed in the handover. Native Linux remains deferred under fn-128. These open gates do not add to the introduced finding count, and fixing C1/C2 will still require a new frozen source binding and bounded re-review before a source-progress commit.
