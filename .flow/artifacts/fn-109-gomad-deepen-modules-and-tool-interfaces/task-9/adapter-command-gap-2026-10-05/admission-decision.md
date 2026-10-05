Conductor-retained copy of the bounded ignored analysis; its stated source base remains historical.

# Task 9 adapter-command admission decision

Decision base is `5c5c025adeabacd9f3217de2b9ab8f7171b0c33c`, branch `gomad`. This is static design analysis. No source, Flow, Git, lifecycle, command/cache test or native execution changed. The research note is `.flow/tmp/adapter-command-gap-research.md`; its earlier research base does not substitute for this decision base. Source references below were read with `git show` at the decision base. Requested design routing is Sol 6.1/high; executed model identity is not exposed here.

## Decision

Amend fn-109.9's command inventory and preservation cases now. Admit no simple `Structured` transport replacement yet. The existing seam cannot preserve this exported helper's complete non-overflow error behavior. A narrow internal outcome extension can recover start/wait error identity and bounded stderr, but cancellation needs a real BASE reproduction and an explicit lifetime/error decision before the source implementation is admitted.

Task 9 owns the R10 omission. Task 39 owns only GOPATH cleanup in the overlapping helper. Run any command correction after task 39's reviewed source-progress commit, then re-anchor its checked cleanup. Task 39 completion and task 21 completion are not prerequisites for source admission. Keep task 21 as the consumer of correction evidence. Original R18/R19 Darwin/full/formal/fixed-identity/static requirements remain required. Transferred Linux execution stays with fn-128.1/.4/.7 and creates no new admission dependency.

## Grounding

- The original parent R10 requires one private command seam, contextual lifetime, output limits and distinct overflow/malformed/capability/timeout/cleanup failures. Parent R18 preserves error precedence and existing public behavior. Task 9 additionally says error text and wrapping are behavior and lists existing wrapping/cleanup examples.
- `target/adapter_source_set.go:31-44` creates a temporary GOPATH, calls `exec.CommandContext`, and captures both streams without bounds. Task 9's inventory omits this command. Its unchanged decode/projection body at `:45-71` defines successful digest bytes and command-before-decode-before-source precedence.
- `target/internal/gocommand/command.go:70-82` drops all result data when execute fails, rejects stdout then stderr overflow, and synthesizes `gocommand.ExitError` for ordinary exits. `:94-115` applies a 15-minute watchdog and substitutes context identities after successful process cleanup.
- `internal/hostexec/command.go:27-39` retains no raw Start/Wait error or ProcessState. `:45-47` rejects an empty directory; `:60-72` uses the smaller configured timeout/caller deadline.
- `internal/hostexec/command_unix.go:59-60` wraps Start errors. `:294-316` discards a nonzero `*exec.ExitError` after extracting its exit/signal fields. `:146-175` sends TERM before KILL. `:88-119` retains cleanup/capture failure precedence before successful result publication.
- `internal/hostexec/output.go:74-98` distinguishes complete RawBytes from bounded head/tail display bytes. Diagnostic bytes cannot serve as complete structured JSON.

## Smallest conditional implementation

If the required reproductions support exact preservation, extend the existing internal mechanism rather than introduce another runner or filesystem framework.

1. Retain the original Start error and original Wait error in a narrowly defined internal outcome. Start failures keep the existing hostexec returned error for ordinary callers, while the adapter-specific projection can recover the exact raw cause without parsing formatted text. Preserve the original nonzero `*exec.ExitError`, including ProcessState, by retaining the object rather than rebuilding it from a status integer. Do not change existing gocommand callers' default error contracts.
2. Add an internal structured projection for this compatibility requirement over the same bounded execution/capture owner. On non-overflow command failure it exposes bounded stderr and the retained raw legacy cause; stdout remains unavailable for decoding. Reject overflow before decoding and document stdout-before-stderr capacity precedence. A runtime cleanup failure still wins over cancellation/capacity errors; caller GOPATH cleanup remains task 39's primary-first composition.
3. Keep `AdapterPreparedSourceSetSHA256`'s public function type. Its public wrapper supplies the existing default runner; a same-package private With entry receives that runner for lawful focused tests. Keep argv, environment override order and the decode/projection body unchanged. Normalize an empty directory only through a BASE-proved equivalent such as `.`; avoid `os.Getwd` or absolute conversion without characterization, because these can add failures or alter relative command lookup.
4. Preserve the caller context and bound capture with a dedicated private listing limit. The proposed 4 MiB per stream is an R10-authorized capacity choice requiring retained listing-size justification, not a historical or measured sufficiency claim. The existing finite watchdog likewise changes the formerly unbounded no-deadline/long-deadline lifetime and must be identified as the R10 lifetime bound.

Potential touches are `target/adapter_source_set.go`, a focused same-package test file, `target/internal/gocommand/command.go` and its tests, plus `internal/hostexec/command.go`, `command_unix.go` and their tests if raw causes must cross that boundary. Only internal mechanism fields/operations are proposed. Other target call sites, source selection/hashing, regeneration, pins, dependency files, process models and public signatures remain outside the proposed implementation.

This is a conditional design, not authority to add fields or modes before root resolves the cancellation decision. Shared hostexec changes require focused existing-consumer review and regressions.

## Required reproduction and decision

Freeze a copy of the original helper from this BASE and task 39's later cleanup-only candidate. Through temporary local executable fixtures, characterize complete wrapped Error text, concrete cause type, ordered unwrap chain, errors.Is/As, stderr and digest for these cases.

- Ordinary nonzero exit with stderr and valid JSON stdout; signal exit; missing executable; non-executable executable; invalid working directory; empty working directory; relative directory and relative executable; empty command if its prior behavior is in the admitted public-input matrix.
- Already-canceled context before Start, cancellation after a child has emitted stderr, expired deadline before Start, and deadline after Start. Retain a started marker and actual process outcome so before/after-Start claims cannot be inferred from scheduler timing. Check child/descendant cleanup separately.

The exact decision concerns cancellation after Start. Legacy `CommandContext` uses its cancellation Kill path and can return a killed-process `*exec.ExitError`; hostexec's TERM-first group cleanup can produce a different signal or successful cooperative exit, and gocommand returns `ctx.Err()` instead. Returning the raw new Wait error alone therefore cannot establish legacy error text/type/unwrap preservation. Joining a context error adds a new unwrap shape. Increasing a timeout or copying capabilityreview's context override does not resolve this conflict.

Choose one of two outcomes using the reproductions. Either an adapter-specific internal execution/projection path preserves the BASE non-overflow cancellation error while providing contextual lifetime and distinct newly introduced watchdog/capacity failures, with existing consumers unchanged; or root obtains an explicit contract decision describing the cancellation/error migration. The current request authorizes no R18 waiver. Until one outcome is concrete and tested, task 9 can record the omitted inventory and design but cannot claim this path corrected or R18 met.

## Proof after source admission

Run BASE characterization before production edits, with all source/cache writers terminal. Then prove exact argv/environment/context/deadline, limit and temporary GOPATH lifetime through `gocommand.New`'s existing recording run seam. Freeze literal source-set digests and listed-error/decode/projection behavior for both supported source selections. Exercise stdout, stderr and dual overflow, valid JSON prefixes, nonzero exit plus overflow, cancellation plus overflow and cleanup plus cancellation/overflow; every failed result has an empty digest and structured data is never decoded after truncation.

Real local process fixtures must prove original exit/start cause preservation, post-Start cancellation/deadline outcomes, bounded capture and descendant termination. Injected flags prove projection/precedence only. Retain the existing gocommand/hostexec process tests and capabilityreview consumers when shared internals change. Genuine GOPATH RemoveAll failures remain task 39's disclosed filesystem proof gap; this command seam does not solve it.

Actual platform pin selection additionally requires the existing `TestAdapterPreparedSourceSetPinsReproduceOnEveryHost` on already available exact inputs with network disabled. A skip supplies no pin evidence; cross-platform file selection supplies static source-set evidence, not native Linux qualification. Root retains all original Darwin/full/formal/affected acceptance and reviews the shared-mechanism diff before committing source progress.
