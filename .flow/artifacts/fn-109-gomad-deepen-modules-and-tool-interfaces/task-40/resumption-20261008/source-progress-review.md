# Task40 readiness source-progress review

SOURCE-PROGRESS PASS. No actionable introduced correctness, preservation or evidence finding was identified in the admitted test-readiness repair. This verdict supports committing verified source progress. Task40 acceptance remains open and this is no formal implementation SHIP or Done verdict.

The reviewer used a fresh context. Requested reviewer was `gpt-6.1-sol/high`, from the same GPT family as the writer. Actual model execution telemetry was unavailable. The reviewer read AGENTS.md, the MILESTONES delivery and verification workflow, tools/gomad3/README.md, the authoritative task40 file, admission.md, conductor-preparation-20261008/findings.md and the final handover.md/evidence.json. Review consisted of source reasoning and read-only Git, shell and Node inspection. No Go, build, test, lint, generator, qualification or lifecycle command ran during review. Only this review file was written; no source, acceptance, Git/index or other artifact was modified.

## Reviewed candidate

- HEAD and comparison base were `4ce2d847afd8762728f30d153d725fd1c073ecb0`.
- Final source manifest was `source-1f4c3da2976f69a2ac13ea2ec3218b5594e139fe95cd18de54eae823abc509a1.json`. All 1,043 entries matched the current checkout independently.
- Changed file `tools/gomad3/target/internal/gocommand/command_test.go` had SHA-256 `84364a221242b142729ec1c31d17a8a199ab0df52f39ec7ce6861d6ebe3e4a63`.
- Tool manifest `tools-73c86870edf2cdafb4bebf6fc17c6ca3c0ba258bfc8c4316eeafc63f18ad4c55.json` matched all 13 actual tool inputs independently.
- Reviewed handover SHA-256 was `9c84b3796c4b22b70eba282cf4957da177fcd64d85ea9d9be06eadd1f0694c97`; evidence.json SHA-256 was `30503a9cc7c7b23ff72eb4b517a5ea8e05b55f47e5db6f0bd2248b7a78b21023`.

Independent comparison against the base found 17 original function chunks, 15 byte-identical and only the two expressly admitted fixtures changed. The three additions were commandAcknowledgement, waitCommandAcknowledgement and TestCommandAcknowledgementRequiresCompletePID. Original comments and processAlive were unchanged. The four protected hostexec/gocommand mechanism and test files were byte-identical to base. The Gomad source diff contained only the admitted command_test.go file. Production consumers and pins retain their exact checked source.

## Readiness and termination reasoning

commandAcknowledgement requires a complete newline-terminated positive integer. Empty files, partial writes, read failures and malformed/nonpositive values return not-ready; none establish process death. processAlive still treats parse failure as alive and accepts only ESRCH as death. The stopped ticker waiter checks readiness repeatedly and exits on its readiness context. Its buffered acknowledgement channel cannot block the monitor when the command returns early.

Both fixtures publish the marker after starting their descendant. The compatibility fixture publishes it after both stream writes, then validates complete stdout/stderr byte totals after capture and cleanup. Cancellation uses WithCancel and occurs after acknowledgement. An unsuccessful five-second readiness monitor cancels the caller and fails the explicit prerequisite. The deadline branch retains an actual ten-second context deadline, a five-second readiness limit and an explicit DeadlineExceeded assertion. The one-second watchdog branch has no competing caller deadline and additionally requires a nil caller error and Cancelled false. These mechanisms retain bounded failure exits without replacing physical evidence with flags.

All original actual ExitError identity/PID, SIGKILL, ExitCode -1, raw-error projection, bounded stdout/stderr, process-group cleanup and ESRCH assertions remain. Structured additionally checks the retained marker against the acknowledged PID. There is no new production cancellation, cleanup, timeout or error-projection behavior.

## Retained execution evidence

All 18 packet receipt hashes, referenced raw stdout/stderr hashes, source/tool/control manifest hashes, exact commands, exits, timestamps and named pass/fail/skip counts matched independently. Each receipt reports unchanged source, tool and control inputs, a completed foreground spawnSync handle, no signal and no runner error. Earlier controls.json versions differ from the final configuration only as disclosed by their content-addressed metadata snapshots, whose bytes match the retained historical hashes. Outer session terminal status is retained worker/conductor bookkeeping, not an independent re-execution by this reviewer. Every reviewer tool call also returned terminal.

| Retained check | Verified observation |
| --- | --- |
| Deterministic acknowledgement RED | Exit 1; one failure for an empty file accepted as ready by the pre-fix predicate. |
| Pre-repair watchdog diagnostic | Exit 0; two named passes; elapsed 1.126351709s, caller nil, Cancelled false, WatchdogTimeout true, actual WatchdogError wrapping raw ExitError, complete 11/10-byte streams. |
| Focused readiness GREEN | Exit 0; eight named passes, including actual deadline and watchdog branches. |
| Complete hostexec/gocommand command gate | Exit 0; 92 named passes, zero failures/skips. |
| Ten readiness/cancellation repetitions | Exit 0; 60 named passes, zero failures/skips; deadline/watchdog excluded explicitly. |
| Sequential architecture replacement | Exit 0; 14 named passes, zero failures/skips. |
| Sequential generated validation replacement | Exit 0; check-only make validate. |
| Task40 scoped lint and actual make lint-code-fast | Exit 0; zero issues each. |
| Task9 scope and integrated unfiltered lint | Exit 1 with two inherited build-context ST1005; exit 2 with 24 residuals, comprising 20 errcheck, one forbidigo and three staticcheck. |
| Formatting, standalone errortype and preservation | Exit 0 each. |
| Linux amd64 / Darwin arm64 source inventories | Exit 0; eight packages each, command_test.go included and no listed package error. |

The original task9 default receipt and raw logs retain 86 named passes, five failures and no skips. They show an empty descendant PID, empty startup streams in both cancel-overflow branches and raw killed-process projection in the watchdog case. The pre-repair diagnostic identifies the winner of that fresh diagnostic run only. The winner of the historical watchdog RED remains unproved; no production watchdog defect or fix is established.

Architecture handle 19724 and generated-validation handle 52472 overlapped during `2026-10-09T02:16:13.767Z` to `02:16:18.003Z`. The packet preserves these as supplementary evidence. The sequential architecture replacement ran `02:16:53.285Z` to `02:17:26.449Z`; generated validation in serial driver 15923 ran `02:17:58.546Z` to `02:18:01.168Z`. The required replacement receipts bind the same final source and avoid the disclosed overlap.

Both reused task9 upgrade diagnostics retain exit 0 and 31 named passes with no failures/skips. Their raw output hashes match, and comparison of source manifests finds only this test file changed from their source `7303aa7e32e0a28b895dd97156a08a6f6b2c5f2c5dd0cb5eff3c952401cc9f5a`. They supply same-production controls. The prior upgrade publication ENOENT remains unexplained, and neither diagnostic proves a publication fix or full current-candidate qualification.

## Remaining acceptance

Genuine first deferred OS pipe-close failures and simultaneous cleanup failures remain unexecuted. The reachability research explains the admitted surface's limitation without discharging that proof obligation. This admission provides no waiver, close-substitution seam or natural OS-fault claim.

Broader unfiltered lint remains red. Original predecessor/matched-first-baseline preservation and complete source-owned full/default/functional/affected-consumer/formal requirements remain open wherever unproved. The two go list inventories establish source selection only; they supply no complete both-platform type-check or native execution proof. This reviewer supplies no full native test-host, patched-runtime, functional, adapter qualification or soak result.

The actual host is Linux aarch64 using stock Go1.27.1 and remains developmental. Supported native linux/amd64 and darwin/arm64 qualification remain deferred and unverified under fn128/fn149. Task9, fn113 and task21 retain adapter integration, nonempty source-set/pin and aggregate R10/R18/R19 obligations. Root retains source-progress commit and lifecycle ownership. No PR, push, CI, publication, acceptance-policy change or native-owner revival follows from this review.
