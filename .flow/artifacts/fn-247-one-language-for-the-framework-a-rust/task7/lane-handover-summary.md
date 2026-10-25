Ported retired-name filtering to native xtask and switched the shell caller to the compiled tool. Removed the byte-only original-artifact verifier, its active manifest and Python call while preserving the independent lifecycle comparison and changed-edge mutation control.

Tier: session (jev-unavailable(no_key)), roleimplementer gpt-6.1-sol medium.
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Status: in_progress; conductor owns integration, review and Flow completion.
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2477-port
Commit: e4899e270f
Baseline: green (old Python policy control; existing xtask tests; two compatibility controls). No Flow CLI was called, per conductor's shared-state freeze.

Controls and evidence:
- `raw_records_exempt_only_author_facade_tokens`: raw NUL path/number/line records, newline-bearing/non-UTF8 paths and lines, trailing EOF, exact exemption boundaries and mixed forbidden/exempt tokens.
- `malformed_records_fail_closed`: missing separators, empty/nondecimal line numbers return 2.
- `posix_ere_recheck_and_grep_failure_propagate`: external POSIX character classes, invalid ERE status 2; no Rust-regex substitution.
- `active_spellings_and_historical_exclusions`: Rust counterpart of the removed Python policy test's active spelling matrix, allowed current names, global historical exclusions and narrower layout-only exclusions.
- `tool_failures_never_report_clean`: git error 7, malformed git stream 2, grep error 9 never report clean.
- `io_and_missing_grep_errors_fail_closed`: output failure and missing grep return 2.
- Compatibility selector executes exactly two fixture tests: `retained_original_exact_lifecycle_projection` and unchanged `changed_original_lifecycle_edge_fails_the_same_comparison`. Both pass. Original snapshots and historical hash artifacts have no diff.
- Initial native corpus: 3 failed with absent-command diagnostics before implementation; final corpus: 6 passed. The full baseline xtask suite passed. Final xtask clippy all-targets passed. Final real repository scan using the built binary returned 0.

Native contract amendment (owner 2026-10-24): Python Unicode whitespace classification is retired in favor of documented byte/ASCII token boundaries. Exact Python traceback/diagnostic bytes and checksum-only original verification are retired. Raw non-UTF8 data is preserved, not lossily decoded. Grep children are waited even after input write failure; successful early grep exit may close its input pipe.

Integration obligations: shell uses `helper=$(bash "$workspace/scripts/xtask.sh")` once before scanning, with UMPIRE_XTASK absolute binary override for focused controls. Lane .5 owns the shared bootstrap file, absent from this lane by agreement. Conductor must join dispatcher/lib/caller edits and run standalone bootstrap, filter, lifecycle and mutation controls against integrated source before marking done. No canonical guest publication or full Model gate was run here.

Logs (workspace-relative): .flow/tmp/fn2477-baseline.log, fn2477-red.log, fn2477-green.log, fn2477-verify.log, fn2477-final2.log, fn2477-shell-final.log. Intermediate clippy style failures are retained in fn2477-verify.log and fn2477-final.log; final2 is green. All commands completed.
