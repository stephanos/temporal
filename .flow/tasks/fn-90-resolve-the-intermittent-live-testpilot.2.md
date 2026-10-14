---
satisfies: [R3]
---
# fn-90-resolve-the-intermittent-live-testpilot.2 Signature lines in the affected live assertions and umpire-run report

## Description
Make every affected live assertion print the spec's Signature line on failure (R3), and make the
umpire-run CLI report the Run's diagnostics so the umpire-run test can build one across the process
boundary. Runs in parallel with fn-90.1; the one-line JSON format is the only thing they share.

**Size:** M
**Files:** `tests/testcore/testpilot/signature.go` and `signature_test.go` (new, pure: signature from Run and Verdict, canonical JSON, line rendering), `tests/testpilot_signature_test.go` (new thin checker and Run capture), `tests/testpilot_nexus_pair_case_test.go`, `tests/testpilot_nexus_caller_case_test.go`, `tests/testpilot_worker_outage_case_test.go`, `tests/testpilot_umpire_run_test.go`, `tools/umpire/cmd/umpire-run/run.go`, `tools/umpire/cmd/umpire-run/run_test.go`
**Touches:** [tests/testcore/testpilot/signature.go, tests/testcore/testpilot/signature_test.go, tests/testpilot_signature_test.go, tests/testpilot_nexus_pair_case_test.go, tests/testpilot_nexus_caller_case_test.go, tests/testpilot_worker_outage_case_test.go, tests/testpilot_umpire_run_test.go, tools/umpire/cmd/umpire-run/**]

### Approach
- Put the pure part in `tests/testcore/testpilot/signature.go` (no build tag, like its neighbours): build the spec's canonical failure signature from a `*testpilotpb.Run`, a `*testpilotpb.Verdict` and the assertion name, render it as one `TESTPILOT-SIGNATURE <json>` line, and compare two signatures field by field (for a later R7 quarantine). Pin it with an offline table test in `signature_test.go` (canonical field order, sorted lists, no ids or messages, equal signatures for runs differing only in ids).
- Add a thin checker in package `tests` (build tags `test_dep && integration`, like its neighbours) that prints that line via `t.Log` just before an assertion fails. Keep it a thin checker (for example `requireRunSatisfied(t, label, run, verdict)`) that owns the disposition, cleanup, Verdict and rule-status checks, so each call site loses lines rather than gaining them. Unresolved rules are those whose status is not SATISFIED; list `rule_id`, `status`, `terminal_state_id`. Diagnostics are `{kind, code}` only, never messages with ids.
- Run capture: when `UMPIRE_REPEAT_RUN_DIR` is set, every affected test writes each closed Run there with `replay.WriteRecordedRun` (as `runRecording` does at `tests/testpilot_live_case_test.go:110-116`), under a unique file name; the umpire-run test passes `--record <dir>/<name>.json`. Unset: nothing written. An unwritable directory fails the test naming the path.
- Apply it at: `tests/testpilot_nexus_pair_case_test.go:45-60`; `requireNexusCallerVerdict` at `tests/testpilot_nexus_caller_case_test.go:213-246` (it serves every caller Query test, including `...CaseRunsFromItsFixtureNameAlone`); both worker-outage tests at `tests/testpilot_worker_outage_case_test.go:39-42` and `:76-79`. Evidence-shape assertions after those (for example `requireCorrelatedNexusPairEvidence`) print the signature with the assertion name, so an ordering mismatch is told apart from a rule status.
- `tools/umpire/cmd/umpire-run/run.go:194` `report()`: add one `diagnostic <kind> <code>` stdout line per Run diagnostic after the rule lines. Exit codes and stderr leak lines stay unchanged (fn-83 contract); extend `run_test.go` to pin the new lines.
- `tests/testpilot_umpire_run_test.go:20-60`: time the command; on any failed check, print a signature built from the CLI's `run`/`verdict`/`rule`/`diagnostic` stdout lines, with the stderr `delete ...` lines as `leaks`. Put the exit status, whether the process was killed by the context, and the elapsed time in the failure message (not the hashed signature).
- No behaviour change on the success path: same assertions, same order.

### Investigation targets
**Required:**
- `tests/testpilot_nexus_pair_case_test.go:26-110`
- `tests/testpilot_nexus_caller_case_test.go:124-260`
- `tests/testpilot_worker_outage_case_test.go:35-85`
- `tests/testpilot_umpire_run_test.go`
- `tools/umpire/cmd/umpire-run/run.go:175-205`

**Optional:**
- `tests/testpilot_live_case_test.go:110-125,247`: `runRecording`, `runEventAt`, `requireRunHasOutcome`
- `tests/testpilot_nexus_control_case_test.go:55-70`: the `UMPIRE_CONTROL_RECORD` pattern

### Key context
- fn-91 renames identifiers in two artifact test files under `tests/testcore/testpilot/`; this task only adds new files there.
- Do not edit `tools/umpire/cmd/umpire-gen-*` (other sessions).
## Acceptance
- [ ] `go test -tags test_dep ./tests/testcore/testpilot -run Signature` passes offline; `go vet -tags 'test_dep integration' ./tests` and `go test -tags test_dep ./tools/umpire/cmd/umpire-run/...` pass; `make lint-code-fast` is clean.
- [ ] A deliberately broken local expectation (not committed), for example requiring VIOLATED in the pair test, prints one `TESTPILOT-SIGNATURE` line whose JSON parses and has the spec's fields in canonical order; its output is quoted in the receipt.
- [ ] `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotNexusPairCase|TestTestpilotNexusCallerAsyncCompletion|TestTestpilotWorkerOutageCase.*|TestTestpilotUmpireRun.*)$'` passes.
- [ ] With `UMPIRE_REPEAT_RUN_DIR` set, one run of the pair test leaves one recorded Run file per closed Run; unset, none.
## Done summary
The affected live Testpilot tests (pair, caller Query, both worker-outage, umpire-run) print one `TESTPILOT-SIGNATURE` line on a failing disposition, Verdict, rule-status or evidence assertion, and capture every closed Run under `UMPIRE_REPEAT_RUN_DIR`. umpire-run reports one `diagnostic <kind> <code>` line per Run diagnostic. The pure signature lives in `tests/testcore/testpilot/signature.go`. A new umpire-repeat test reads that encoder's `t.Log`-decorated line back through the harness parser, field for field, so the two ends of the contract cannot drift apart.

Deliberately broken expectation (VIOLATED required in the pair test, not committed) printed exactly one line, which parses with its keys in canonical order:
`testpilot_signature_test.go:43: TESTPILOT-SIGNATURE {"test":"TestTestpilotNexusPairCase","assertion":"verdict status","run_disposition":"Completed","verdict_status":"Satisfied","unresolved_rules":[],"diagnostics":[],"leaks":[]}`

Follow-up (review P3, outside this task's Touches): `runCapturedCase`/`runCapturedCaseWithBinding`/`runCapturedBoundCase` duplicate `runCase`/`runBoundCase` in `tests/testpilot_run_case_test.go`. Letting `runBoundCase` return the bound live case would remove the copies.

The full live gate `make umpire-check-live-tests` was not run: it takes about 30 minutes, over the 10-minute live-command bound. The affected selection passed; see the evidence.

stage: impl-review - ran [2026-09-27T02:33Z..2026-09-27T02:34:16Z] SHIP (claude:opus:high)
## Evidence
- Commits: 6e368d4378f80f6da5a79799b1193b6e3e11fd50, 82b96aa0d9d6c708883b6696d318d728848a0ac9
- Tests: baseline: green (offline quick commands, pre-edit), go test -count=1 -tags test_dep ./tests/testcore/testpilot -run Signature, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-run/... ./tools/umpire/cmd/umpire-repeat/..., go vet -tags 'test_dep integration' ./tests, make lint-code-fast, go test -v -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotNexusPairCase|TestTestpilotNexusCallerAsyncCompletion|TestTestpilotWorkerOutageCase.*|TestTestpilotUmpireRun.*)$' (PASS, 45s), UMPIRE_REPEAT_RUN_DIR=<dir> go test ... -run '^TestTestpilotNexusPairCase$' (2 recorded Run files for 2 closed Runs; unset run wrote none), make umpire-repeat-run SELECT='^(TestTestpilotNexusPairCase|TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint)$' COUNT=2 MODE=process (2/2 PASS, 3 runs captured per iteration), GATE_NOT_RUN:live-suite:make umpire-check-live-tests - 30 min full live gate exceeds the conductor's 10-minute live-command bound; the affected selection ran green instead
- PRs: