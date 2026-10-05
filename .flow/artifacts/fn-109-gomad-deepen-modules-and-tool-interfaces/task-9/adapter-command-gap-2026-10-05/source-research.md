Conductor-retained copy of the bounded ignored analysis; its stated source base remains historical.

# Adapter source-set command gap

Research base is `9f66d05bc8`. This note records static inspection only. No Go command, source/cache-mutating test, native test, download, lifecycle mutation, staging or commit ran. `git check-ignore` confirms this path is ignored. The stable helper, gocommand files, go-command tests and capabilityreview listing files have no diff against that base. The active fn-109.38 writer's capability implementation and prepared_cache.go were outside this investigation.

## Recommendation and admission

R10 owns the omitted command at `tools/gomad3/target/adapter_source_set.go:37`. It still calls `exec.CommandContext` and captures both streams in unbounded `bytes.Buffer`s. Flow's current `fn-109.9` status is `todo`, although its historical acceptance boxes and done summary claim every target listing uses the seam. Its Approach inventory omits this helper. Those historical receipts do not qualify the omitted path.

Use the existing `target/internal/gocommand.Runner.Structured` operation. Keep the exported `AdapterPreparedSourceSetSHA256` signature and add only a same-package private `adapterPreparedSourceSetSHA256With(..., runner gocommand.Runner)` entry for focused tests, matching `target.go:370` and `capabilityreview/list.go:87`. No new public injection API or general command framework is needed. The public wrapper supplies `gocommand.Default()`.

Root should amend task 9's inventory and preservation cases, then admit this as one bounded R10 source writer after the current writer's reviewed source-progress commit. Do not append it to fn-109.38. Task 39 explicitly owns only this helper's deferred GOPATH removal, and its source admission follows task 38. Serialize that overlapping file and re-anchor whichever change runs second. Task 39 is not authority to change command transport. Fn-113.2 retains regeneration approval, source pins, publication and qualification. Task 21 consumes the R10 preservation evidence; its completion is not a prerequisite for admitting this source repair. Native Linux evidence remains with fn-128.1/.4/.7, while Darwin and static coverage of both source sets remain with fn-109.

The straightforward transport replacement is small, but it does not establish exact error compatibility. The API hurdles below must be characterized and resolved in the admitted scope before claiming R18 preservation.

## Exact request and validation contract

The command request must retain these values from `adapter_source_set.go:31-44`.

| Item | Existing value |
| --- | --- |
| Command | `[]string{goCommand, "list", "-e", "-find", "-json", "."}` |
| Directory | `packageDirectory` |
| Environment | `append(targetbuild.Environment(), "GO111MODULE=off", "GOPATH="+gopath, "GOOS="+goos, "GOARCH="+goarch)` in that order |
| Temporary lifetime | `os.MkdirTemp("", "gomad3-source-set-gopath-")` before the command; deferred removal after all parsing and source hashing |
| Context | The original caller's `ctx`, without detaching or replacing it |

`target/internal/build/context.go:88-100` preserves ambient entries except the reserved build/runtime settings and appends the fixed cgo, experiment, Go environment and timezone settings. GO111MODULE/GOPATH/GOOS/GOARCH may already exist in the inherited slice. Retain the final override order and the command's existing duplicate-key semantics. Do not use `capabilityreview.ListWith` as this helper's listing operation. Its `list -deps -json -mod=readonly`, module mode and stream-of-packages parser are different (`capabilityreview/list.go:95-123`).

Give Structured a fixed positive private output limit. Existing nearby structured/diagnostic commands use 4 MiB and Go environment uses 64 KiB (`target.go:48-51`). A dedicated prepared-package-listing constant of 4 MiB is a small proposed bound, subject to the admitted owner checking retained listing sizes. It is a newly enforced capacity, not a historical limit or a measured sufficiency claim. Do not introduce configuration or reuse a module-download constant solely because its number matches. Test both streams and document overflow as the R10-authorized new rejection.

With no explicit Request.Timeout, the seam already applies its 15-minute watchdog and 100 ms termination grace (`gocommand/command.go:11-12,94-115`). `hostexec.effectiveTimeout` uses the smaller caller deadline (`internal/hostexec/command.go:60-72`). This introduces a finite bound for contexts without a deadline and caps longer deadlines; record that explicitly rather than claiming the old unbounded timing is identical.

Keep the existing post-command body intact (`adapter_source_set.go:45-71`). Its order is command failure, one complete `json.Unmarshal`, listed Error, missing Dir/Name, Go-source projection, foreign-source projection, source-set digest. `json.Unmarshal` rejects trailing objects/junk. The only tolerated listed error is the existing suffix `" expects import "+strconv.Quote(importPath)` (`:52`). It is a suffix match, not a general import-error exemption; preserve its current acceptance precisely, including escaped import paths and the ignored prefix. Listed ImportPath is replaced by the supplied importPath before projection. An exit failure always wins over any valid JSON or accepted import-comment error in stdout.

Unchanged ordinary error templates are `create source-set GOPATH: %w`, `list prepared package %s for %s/%s: %w: %s` with trimmed stderr, `decode prepared package %s listing: %w`, `list prepared package %s for %s/%s: %s`, and `prepared package listing has no directory or name`. Source/file errors follow the existing projection path. Preserve all source enumeration, normalization and hashing calls; this request does not authorize new pins.

## Existing mechanism and API hurdles

`gocommand/command.go:70-82` first executes, then rejects stdout overflow, then stderr overflow, then returns complete RawBytes and any exit error. This ensures neither a valid JSON prefix nor a truncated tail reaches decoding. Host capture drains both streams and maintains complete hashes while bounding retained bytes (`internal/hostexec/output.go:31-48,51-98`). Overflow does not immediately terminate a producer; the contextual/watchdog lifetime still bounds it.

The existing API is sufficient for normal success, malformed JSON, ordinary exit status text, and fail-closed overflow. It cannot preserve every old public error exactly by simply replacing lines 37-44.

1. `Structured` returns a zero result whenever execute fails (`:71-74`). Cancellation, watchdog and transport failures therefore discard stderr. Previously this helper included all captured stderr on a failed Run. Preserving bounded stderr for a cancellation with output requires a narrow change to the existing seam's error-result contract or a separately accepted behavior change. Any seam change must keep stdout unavailable on execution errors and preserve execute-error precedence over overflow. Do not switch to Diagnostic and then decode its bytes as if they were complete.
2. `gocommand.ExitError` retains ordinary `exit status N` / `signal: ...` text (`:58-68`) but is not `*exec.ExitError`. `hostexec.Result` carries no original error or ProcessState (`internal/hostexec/command.go:27-39`), so exact errors.As compatibility cannot be reconstructed through the current Runner. A public function returning error cannot be assumed to have no external type-sensitive consumers. Repository production consumers simply propagate the error, but that is not external API proof.
3. `hostexec` wraps command-start errors with `start command %q: %w` (`command_unix.go:59-60`). The old helper wrapped the direct exec Run error. Startup failures will acquire different text and another unwrap level. Preserve or explicitly admit this change; do not strip a formatted string to simulate the old error.
4. The seam returns caller cancellation/deadline identity after successful host cleanup (`gocommand/command.go:104-115`). A running `exec.CommandContext` commonly previously returned a killed-process `*exec.ExitError`; a context canceled before start returns the context error. Characterize both on BASE. R10's contextual contract calls for cancellation/timeout to remain distinct; do not claim simultaneous exact legacy cancellation text/type and ctx.Err identity without a specific design.
5. The old `exec.Cmd.Dir == ""` inherited the caller's cwd. `hostexec.validateRequest` rejects empty Dir (`command.go:45-47`). Preserve this public input behavior with a minimal directory normalization if admitted, or characterize/document its rejection. Current production callers supply a prepared directory, so this is a public edge rather than the normal path. Nonempty relative directories and relative command paths also need a regression; avoid gratuitous absolute-path conversion.
6. Host cleanup errors precede context and overflow because execute returns its run error first. Preserve that seam order. Do not copy capabilityreview's extra ctx.Err override blindly. Its `ListWith` has a different error contract (`list.go:113-121`). Task 39 later composes GOPATH-removal errors primary-first and clears a digest on cleanup failure; keep that independent lifetime correction in its owner.

The bounded options are to admit the documented R10 lifetime/exit-error migration for this helper and test it, or to admit a narrow richer error result inside the existing gocommand/hostexec seam for exact compatibility. The latter reaches shared consumers and needs a larger focused review. Neither option is authorized by this read-only research. There is no source-set hashing or module-policy reason to add a new command abstraction.

## Portable proof strategy

Add focused same-package tests against the private With entry. Inject `gocommand.New` with a recording hostexec function, as `target/go_command_test.go:107-127` and `target/installation_test.go:35` already do. This is an existing lawful command seam, separate from task 39's absent filesystem fault seam.

The injected function should observe the original context/deadline, exact argv and directory, environment order and overrides, a positive fixed limit, and the temporary GOPATH while it exists. After each return, verify the GOPATH is absent on ordinary success/failure. That proves actual ordinary cleanup, not genuine RemoveAll failure execution.

Use literal source files and captured/literal JSON listings, with baseline-recorded literal digests, for both darwin/arm64 and linux/amd64 cases. Cover Go and foreign files, source-order input variation if BASE permits it, overridden listed ImportPath, accepted exact quoted import-comment suffix, a different import path, other listed errors, malformed/trailing JSON, missing Dir and missing Name, and source-read failure. Freeze all referenced projection/hash sources only after fn-109.38 is committed. A fake listing proves digest/projection preservation; it does not prove actual Go platform file selection.

For each stdout/stderr overflow case, return Truncated=true with a syntactically valid prefix or plausible complete JSON and assert an errors.As OverflowError, empty digest and no decode/source-access error. Include both streams overflowing to retain stdout-first order, a nonzero exit plus overflow, and cancellation plus overflow. Inject cancellation/deadline/cleanup faults to prove the adapter's wrapping and precedence. A fake cancellation proves propagation only.

Keep existing process tests as real mechanism evidence. They use ordinary local shell processes and need no patched runtime or module download on supported Unix hosts.

| Existing test | Location and proof |
| --- | --- |
| TestStructuredRejectsOverflowBeforeReturningData | `target/internal/gocommand/command_test.go:18`, actual output overflow |
| TestDiagnosticBoundsLongOutputAndKeepsFullHashes | same file `:27`, bounded diagnostic capture and complete hash |
| TestStructuredCancellationRemovesDescendant | same file `:42`, cancellation and descendant termination |
| TestDiagnosticRetainsOutputOnWatchdogAndPropagatesCleanupFailure | same file `:73`, injected watchdog/cleanup distinction |
| TestListRejectsOverflowBeforeDecodingValidPrefix | `target/internal/capabilityreview/list_test.go:30`, caller overflow guard |
| TestListDistinguishesMalformedListingAndInvalidInput | same file `:39`, distinct listing errors |
| TestListPreservesDeadline | same file `:53`, actual sleeping command bounded by context |
| TestRunTimesOutAndRemovesTermIgnoringDescendant | `internal/hostexec/command_unix_test.go:83`, watchdog and TERM-resistant descendant removal |
| TestRunClassifiesContextCancellation | same file `:107`, cancellation and GroupGone |
| TestRunBoundsEachOutputStream | same file `:121`, exact per-stream retention/counts |

Add a helper integration regression with a temporary executable fixture returning malformed JSON or an exit failure to prove the public wrapper reaches the default seam. For bounded output, existing gocommand tests plus the helper's request-limit/overflow tests compose direct evidence; a helper-level real over-limit fixture is useful if its fixed limit can be exercised cheaply. Do not call a synthetic Truncated flag a measurement of memory use. Reuse the existing descendant test; if testing helper cancellation end-to-end, retain a real descendant PID and verify it is gone. A process that only returns a context sentinel is no termination proof.

Proposed focused commands, only after root freezes the source and admits testing, are `go test -count=1 -tags test_dep ./target/internal/gocommand ./target/internal/capabilityreview ./internal/hostexec`, followed by a narrowly named new target adapter-source-set test selection. Use the already available pinned stock Go with GOTOOLCHAIN=local and network disabled. The real public preparation/cache test `target/go_command_test.go:19` compiles and mutates caches; do not run it concurrently with another source/cache writer. Native/full/affected gates remain the owning task's separate obligations.

## Real pin evidence and blockers

`deterministicio/adapter_regenerate_test.go:40` already defines `TestAdapterPreparedSourceSetPinsReproduceOnEveryHost`. It checks every adapter's two pinned platform digests (`:64-79`). Its helper probes `.toolchain/bin/go` and skips when unavailable (`:20-29`); a skip establishes no pin evidence. The test calls `downloadPinnedModule` (`:55`) and prepares replacements, so it is neither a pure test nor safe to run blindly under this task's no-download constraint. Run later only with all exact inputs already cached and network disabled, retain the subtest/output count, and record missing-input failures without fetching anything. Cross-platform source selection on one host is static source-set evidence, not native qualification of the other platform.

Production callers at BASE are `deterministicio/adapter_regenerate.go:247,361` and `deterministicio/libc_regenerate.go:74,181`. The first pair creates proposed pins and verifies prepared pins, and both propagate this helper's error. They show why success digest bytes, import-comment handling and exact platform selection must stay stable. Those files require no edit for the wrapper/seam integration.

Current blockers to a completed repair are source admission/serialization, an explicit resolution of the error-result/type/cancellation compatibility gap, a justified fixed output bound, and execution evidence against a frozen candidate. This investigation ran no test and claims no native, cancellation, memory-bound or pin qualification result.
