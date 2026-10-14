---
satisfies: [R1]
---
# fn-90-resolve-the-intermittent-live-testpilot.1 Reproduction harness: umpire-repeat command and make target

## Description
Build the reproduction harness (R1): a Go command `umpire-repeat` beside the other Umpire developer
commands, plus a make target that sets up the gate's prerequisites and runs it. It implements the
spec's Harness, Record file and Signature line contracts. It is offline-testable and does not need
the live tests to print signatures yet: fn-90.2 adds those in parallel against the same line format.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-repeat/main.go`, `run.go` (loop, modes, interrupt, fingerprint), `events.go` (`go test -json` reader), `signature.go` (signature line parse, fallback, normalization, hash), `run_test.go`, `signature_test.go`, `testdata/*.jsonl` (canned event streams), `Makefile` (`umpire-repeat` build target and `umpire-repeat-run` target, `.PHONY` line)
**Touches:** [tools/umpire/cmd/umpire-repeat/**, Makefile]

### Approach
- Mirror the command shape of `tools/umpire/cmd/umpire-run` (`main.go` thin, `run.go` with an injectable runner and writers, `run_test.go`), using `tools/umpire/internal/cli` for flag errors and `WriteLine`.
- Build the test binary once with `go test -c -tags 'test_dep integration' -o <tmp> ./tests`. A build failure exits non-zero before iteration 1 and records nothing.
- Run each iteration as `go tool test2json -t -p <pkg> <binary> -test.v=test2json -test.run <SELECT> -test.count=<1|N> -test.timeout=<flag>` with cwd `./tests` (testdata paths are relative to the package dir) and the physical `TMPDIR`, like the gate at `Makefile:800-827`.
- Process mode: N processes with `-test.count=1`. In-process mode: one process with `-test.count=N`; split iterations by the order of each selected top-level test's `run` events, since repeats share a name.
- Detect "no test matched": the package still passes, so fail when no selected top-level test emitted `run`.
- Signature: take the first `TESTPILOT-SIGNATURE <json>` output line of a failing leaf test. Without it, build `{test, assertion}` from the first `OutputType == "error"` line (or the last testify `Error Trace`/`Error:` block, as `tools/testrunner/log.go:362-400` finds it). Neither is readable: `unparsed`. A selected test that started but got no pass/fail event (crash, kill, binary timeout): `process`, with the last output line as the assertion.
- Normalize before hashing: UUIDs, run ids, `-deleted-xxxxx` namespace suffixes, ports, addresses, hex, durations and timestamps; sort list fields; hash the canonical struct (not a map). Strip the `file:line` location from the hashed assertion (tasks .2 and .4 move lines between the baseline and the final loops) but keep it in the record.
- Interrupt: `signal.NotifyContext` with `exec.Cmd.Cancel` sending SIGINT and a `WaitDelay`; the interrupted iteration is not counted or recorded, and the summary prints what completed.
- Fingerprint: before every iteration hash the inputs the tests read at run time: the tracked and untracked contents of `tests/`, `common/testing/testpilot/`, `tools/umpire/`, and the built `model/.lake/build/bin/umpire-explore` and `umpire-replay-bridge`. A change from the loop's starting fingerprint stops the loop like an interrupt and names the changed path. Store the fingerprint in every record line; `summarize` refuses to add files with different fingerprints.
- Run capture: for each iteration set `UMPIRE_REPEAT_RUN_DIR` to a fresh directory beside the record file and list the Run files the tests wrote in that iteration's record line (`runs`).
- Record file: append one JSON line per finished iteration, with the commit (`git rev-parse HEAD`) plus a dirty flag for `tests/`, `common/testing/testpilot/`, `tools/umpire/`. A `summarize <file>...` subcommand prints the summary over one or more files; a malformed line fails and names the file and line. At loop start also record the host load average and the count of other `go test` processes (spec Edge Cases, Shared host).
- The summary prints each rate with its 95% Clopper-Pearson interval, and the rule-of-three bound when the count is zero.
- Make target: `umpire-repeat-run` requires `SELECT`, `COUNT` and `MODE`, optionally takes `RECORD` and `UMPIRE_REPEAT_FLAGS`, and builds `umpire-explore umpire-replay-bridge` first, following `umpire-fuzz-run` at `Makefile:706-722`. It stays out of `umpire-check-regression`.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-run/run.go` and `run_test.go`: command shape and test style to mirror
- `Makefile:800-827`: the gate's prerequisites, flags and physical TMPDIR
- `Makefile:706-722`: make-target convention for a runnable Umpire command
- `tools/testrunner/log.go:122-160,362-400`: existing failure-block and alert parsers to mirror (not import; that package is the CI re-runner)

**Optional:**
- `go help buildjson`, `go doc cmd/test2json`: event and build-event shapes (Go 1.27)

### Key context
- Since Go 1.24, build errors in a `-json` stream arrive as `build-output`/`build-fail` events and a `fail` event carrying `FailedBuild`. Do not set `GODEBUG=gotestjsonbuildtext=1`.
- Non-JSON text can still appear for early fatal errors; treat it as `unparsed`, never drop it.
- `panic: test timed out` should become a signature whose assertion is the timeout line.
## Acceptance
- [ ] Unit tests with canned streams cover: all-pass; one failure with a signature line; failure without a signature line (fallback); unreadable failure (`unparsed`); a started test with no terminal event (`process`); the same assertion at two different line numbers hashes equal; a fingerprint change mid-loop stops it and names the path; `summarize` refuses mixed fingerprints; Run-capture paths are listed per iteration; build failure (exit non-zero, zero iterations); no test matched (exit non-zero); in-process `-count=3` split into 3 iterations; two occurrences differing only in run id, port and namespace suffix hash equal; an interrupted loop records only completed iterations; `summarize` over two record files adds counts; a malformed record line fails naming file and line.
- [ ] `go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...` passes; `make lint-code-fast` is clean.
- [ ] One live smoke: `make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=2 MODE=process` and the same with `MODE=in-process` print two PASS lines and a summary; the output is quoted in the receipt.
## Done summary
Added `umpire-repeat` (tools/umpire/cmd/umpire-repeat) and the `umpire-repeat` / `umpire-repeat-run` make targets. The command builds the live test binary once and runs a selection N times, either one process per iteration or one process with `-test.count=N`. It reads the test2json stream, takes each failing leaf test's signature from its `TESTPILOT-SIGNATURE` line (including the `t.Log`-decorated form), falls back to the first failing assertion, and otherwise reports `unparsed` or `process`. It hashes each signature with iteration-specific values and the source location removed, and appends one record line per finished iteration. The loop stops when the input fingerprint changes, and an interrupt records only the iterations that finished. `summarize` adds up record files that share a fingerprint and prints Clopper-Pearson intervals. Each acceptance case has a canned-stream test in run_test.go or signature_test.go.

Live smoke at c9df050400, run with `make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=2`:
- MODE=process: `1 PASS`, `2 PASS`, then `TestTestpilotNexusPairCase 0/2 rate 0.00% (95% CI 0.00%-84.19%, rule of three <= 150.00%)`, rc=0
- MODE=in-process: the same lines, rc=0 (load 4.24 3.59 3.71, 0 other go test processes)

The tree reads as dirty because other sessions have uncommitted edits under tools/umpire. Loops under .3 need a clean tree.

Follow-up for .3: the SELECT in `umpire-repeat-run` is read with `$(value SELECT)`, so a trailing `$` survives make's expansion.

Gates: unit tests and lint-code-fast are green, and go vet of ./tests is green. `make umpire-check-live-tests` was not run (inconclusive: it is a ~30-minute suite, beyond the foreground bound).

stage: impl-review - ran [claude: NEEDS_WORK (1 P1 t.Log signature decoration, 4 P3) -> fixed in c9df050400 -> SHIP]
## Evidence
- Commits: 5cc298a69b5ef397c96b0089a31f1b11245ee17f, c9df0504000f19bf092de88489a78425d4af6caa
- Tests: go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-repeat/... (green, 16 tests incl. table cases), make lint-code-fast (green), mise exec -- go vet -tags 'test_dep integration' ./tests (green), make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=2 MODE=process (rc=0, 1 PASS, 2 PASS, summary 0/2), make umpire-repeat-run SELECT='^TestTestpilotNexusPairCase$' COUNT=2 MODE=in-process (rc=0, 1 PASS, 2 PASS, summary 0/2), baseline: none (the offline Quick command targets the package this task creates; go vet of ./tests was green), INCONCLUSIVE: make umpire-check-live-tests not run - a ~30 min suite beyond the worker's 600s foreground bound; the diff adds a standalone command and two Makefile targets that no live test imports
- PRs: