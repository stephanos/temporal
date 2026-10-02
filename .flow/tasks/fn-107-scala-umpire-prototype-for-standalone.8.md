---
satisfies: [R9]
---
# fn-107-scala-umpire-prototype-for-standalone.8 Demonstrate Quint transition agreement and one P monitor

Touches: [model/scalav2/backends/**, model/scalav2/README.md]

## Description
Build narrow adapters from the admitted finite IR to Quint and one P event monitor. Gate each supported backend on semantic agreement.

**Size:** M
**Files:** proposed Quint adapter/runner, P monitor adapter, agreement/replay fixtures, backend README.

### Approach
- Verify primary tool documentation and available pinned toolchains during implementation. Do not silently substitute sampling for exhaustive transition agreement.
- Encode components as finite state, preserving queue order/fault choices, initial states, public results, monitor state, and progress assumptions in the selected slice.
- Enumerate reachable transition agreement with Go and replay external witnesses through ordinary IR admission/evaluation.
- Export one passive authored monitor to P and compare bounded accepted/rejected event traces. State coverage and exclude P module-refinement claims.

### Investigation targets
**Required:** model/scalav2/SEMANTICS.md; proto/internal/temporal/server/api/modelir/v1/ir.proto; model/scalav2/goir/parity_test.go; model/go/umpire/table.go; model/go/umpire/search.go.
**Optional:** .plans/lean/UMPIRE_OUTSIDE_THE_BOX.md:237.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/...`; document backend-specific agreement commands after verifying the tools.

## Acceptance
- [ ] Quint and Go agree on the entire reachable transition relation and selected properties of the declared finite specimens.
- [ ] External counterexamples replay and rejected/unsupported witnesses remain diagnostic errors.
- [ ] The P monitor and authored Go monitor agree on supported bounded traces, including negative controls.
- [ ] Receipts distinguish transition agreement, checker coverage, and unsupported module-refinement functionality.

## Done summary
Quint 0.33.0 now evaluates every machine of the four lifted IR files and agrees with `goir` on all of it, and P 3.1.0 agrees with Go's `terminalFinality` monitor on every event trace of both activity admission designs within five steps. Compositions, refinements, Queries and progress claims are not exported, and Apalache does not take the Nexus close module. Nothing is committed and the task stays `in_progress`.

Workspace: `/private/tmp/claude-501/-Users-stephan-Workspace-temporal-umpire/3649feab-fa2b-4588-b529-974aeb4c1742/scratchpad/w8`. Logs are under `.flow/tmp/fn-107/task8-logs/` in that copy.

### Files

Changed:
- `model/scalav2/README.md`: four hunks (diagram line, run line, layout row, one Results paragraph). `task8-logs/README.diff` is the diff against the main checkout's file.

Added, all under `model/scalav2/backends/`:
- `slice.go` (608 lines): `Slice`, `Receipt`, Go's reading of a machine, its monitor product and its Properties, `Replay`.
- `quint.go` (1041): IR to Quint, the dump module and the check module.
- `itf.go` (393): reading ITF values back by the IR's types.
- `agreement.go` (377): comparing a dump with Go.
- `checked.go` (186): confirming monitor verdicts and counterexamples through `goir.Check`.
- `verify.go` (178): the Apalache run and its verdict's comparison.
- `p.go` (838): IR to P, the traces, comparing P's reports.
- `tool.go` (61), `run.sh` (83, executable), `README.md` (167).
- Tests: `quint_test.go` (628), `p_test.go` (207), `verify_test.go` (159), `encode_test.go` (178), `tools_test.go` (44).

The package imports `goir`'s and `umpire`'s public API only. No file outside the Touches changed.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | Quint and Go agree on the entire reachable transition relation and selected properties | pass for all 29 machines; the 8 compositions are not exported (gap 1) | `TestQuintAgreesWithGo`, `TestQuintDisagreesOnAnotherModel`, `TestQuintReadsPropertiesAboutAnAction`, `TestAgreementReadsAFaithfulDump`, `TestAgreementRejectsATamperedDump` (13 cases), `TestGoMonitorVerdictsAreTheSpecimens` |
| 2 | External counterexamples replay; rejected and unsupported witnesses stay errors | pass | `TestExternalWitnessesReplayOrAreErrors`, `TestAWitnessOfAnotherMonitorIsRejected`, `TestVerifiedVerdictsAreHeldToGo`, `TestQuintVerifyAgreesWithGo`, `TestCheckerAnswersAreFoldedIn`, `TestADumpThatIsNoDumpIsAnError`, `TestQuintExportRejectsWhatItDoesNotTranslate` (5 cases), `TestQuintStopsWhereTheModelHasNoValue` |
| 3 | The P monitor and the authored Go monitor agree on supported bounded traces, with negative controls | pass | `TestPMonitorAgreesWithGo`, `TestPDisagreesOnAnotherMonitor`, `TestPTracesCarryGoVerdicts`, `TestPAgreementReadsTheCheckersReports`, `TestPExportRejectsWhatItDoesNotTranslate` |
| 4 | Receipts distinguish transition agreement, checker coverage and unsupported module refinement | pass | `TestQuintExportListsWhatItLeavesOut`, the kind maps in `TestPMonitorAgreesWithGo` and `TestPAgreementReadsTheCheckersReports`, `TestQuintVerifyDoesNotTakeTheNexusModule` |

### What the final run compared

`backends/run.sh` printed 131 receipts: 66 agreed, 31 covered, 33 unsupported, 1 not run (`agreement-final-wZXPUV/receipts.txt`).

- **Transitions (Quint):** 29 machines, 1,666 reachable states, 24,805 state and class pairs: 5,838 enabled with 6,337 results and 18,967 disabled. Results compare outcome, state, facts in order and the explanation. Starts compare in order; reachable states, ends and classes as sets.
- **Monitors (Quint):** 11 machines, 2,378 steps over 946 product states. Seven machines violate a monitor. Quint's counterexample of each violated monitor replays through a fresh `goir.Build` and through `goir.Check`.
- **Properties (Quint):** 149 Properties, 45,819 readings, 13,437 of them of a step the Property is about.
- **Apalache:** `currentAdmission` and `staleAdmission`, by `atMostOneActiveAttempt` and `terminalFinality`. No violation of either in any run of `currentAdmission` up to 4 steps. A two-step counterexample of each for `staleAdmission`, which is Go's shortest.
- **P:** `staleAdmission` 1,171 traces (996 accepted, 175 rejected), `currentAdmission` 483, all accepted.

### How the Quint agreement is exhaustive

The module computes the reachable set and every row in pure definitions, and its one state variable holds the result. `quint run` takes one sample of one step to write that as an ITF trace, so nothing is sampled. Go gives the module one number, how many rounds of successors to take, and the dump says whether a step still leaves the set. `TestAgreementRejectsATamperedDump/an_open_frontier` fails the comparison on `closed: false`.

Three mutants of the exported Model (a step's condition inverted, a Property negated, the monitor's notion of over narrowed) each make real Quint disagree with the unmutated Go reading. Two mutants of the P monitor make real P disagree.

### Test-first record

- `red-01-no-package.log`: `quint_test.go` written before any implementation; the build fails on `undefined: Slice`.
- The first real Quint run agreed on all four slices (`quint-01-first-agreement.log`). I treated that as suspect and added the mutant controls before trusting it.
- `red-02-pair-left-out.log`, `red-03-product-claims-pair-left-out.log`: a pair missing from the dump read as agreement. Tests written first, both red, then fixed.
- Not test-first: the P exporter, the Apalache path and the Property comparison were written with their tests. Their tests passed on first run against the unmutated Models.
- Mutation over the comparison code, foreground, each file restored after its mutant (`mutants.py`): 16 mutants, 7 survived the first run (`mutants-01.log`), 0 after five added tests (`mutants-02.log`).
- One wrong expectation of mine: I pinned the shortest `terminalFinality` rejection at 4 steps. The feature Model's is 2 (a schedule-to-close timeout, then the stale start). The specimen's A4 path is one of the rejected traces at step 4, which the test now pins.

### Decisions that differ from the task text

1. **Compositions are listed `unsupported`.** Their members are exported as machines. See gap 1.
2. **Go's monitor product is evaluated here with `goir.Interpreter`,** about 60 lines that mirror `goir`'s private `binding.monitor`. `goir.Check` confirms it: a verify of an always-true Property over every path from each start, and one over the classes of each counterexample, on a clone of the Model with those Queries added.
3. **A counterexample of the evaluator's product is a shortest path Go extracts from Quint's dumped graph.** No Quint checker produced it. The four Apalache counterexamples are the checker's own.
4. **Queries and progress claims are not exported.** Quint is given the Properties and monitors, read on every step. Each slice has one `query-agreement` receipt and each progress claim one `progress-agreement` receipt, all `unsupported`.
5. **P gets one monitor, translated function by function,** on two machines, traces of at most five steps, steps without facts. Only the first 16 rejected traces also run alone against the monitor's own assertion; all 175 run in `tcAgreement`.
6. **`run.sh --install` installs .NET and P.** Without it the script fails and names the command.
7. **A value no `match` case accepts and a call outside a precondition are written as `List().head()`,** which stops Quint's evaluator with `QNT505`. The run fails; no row is produced.

### Findings

1. **`p check -tc` selects by prefix and runs nothing for a name that matches nothing, with exit 0.** `RunP` requires the output to name exactly the test case asked for.
2. **`quint verify` starts an Apalache server and writes `_apalache-out` into its working directory.** The checks run in a temporary directory on port 38822 and stop the server on that port.
3. **Apalache 0.62.1 fails on the Nexus close check module** in its InlinePass: `Recursive substitution took more than 100000 iterations`. The first attempt, with the whole export, failed on a 20 MB JSON limit; check modules now hold one machine.
4. **No exported machine declares a Property about an action.** That translation is tested on a derived Model (`TestQuintReadsPropertiesAboutAnAction`).
5. **`make lint-code` exits 3 with `parallel golangci-lint is running`** when another session lints. I retried until it ran.

### Gaps outside the Touches (not changed)

1. **`goir` has no public reading of a composition's table.** `binding.subject` and `compositionSubject` are private and `Report` exposes no tables. A composed export has nothing public to be compared with.
2. **`goir` exposes no bound monitor.** `Machine.Monitors` is the declarations only, which is why decision 2 exists.
3. **`model/quint/quint.sh` is reused as the pinned Quint.** If that experiment is removed, `run.sh` needs its own one-line wrapper.

### Left undone

1. No Apalache or TLC result for the nine Nexus machines.
2. No P export of `atMostOneActiveAttempt` or any Nexus monitor: their types have enum cases that carry values.
3. Evidence lines, Definition IDs and fingerprints are not exported.
4. P traces longer than five steps.

### Side effects outside the repository

- `/tmp/umpire-backend-tools`: .NET SDK 8.0.425, P 3.1.0, the install script and two install logs.
- `~/.quint/apalache-dist-0.62.1`, downloaded by `quint verify`. NuGet's and npm's caches hold P's and Quint's packages.
- No `lake` or `lean` was called. I ran `model/quint/quint.sh` only, never `model/quint/run.sh`.
- Before the no-recursive-delete rule arrived I removed one stray `_apalache-out` directory and one kept output directory inside the copy with `rm -rf`. None since; `run.sh` and the tests delete no directory.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, before any edit) | 0 |
| same, final (`backends` 41.7 s with its 8 tool tests skipped, `goir` 37.5 s) | 0 |
| `model/scalav2/backends/run.sh --out <dir>` (170 s, no skip) | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/backends/...' GOLANGCI_LINT_FIX=false` (0 issues, after 32 fixed) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |

`go test ./model/scalav2/...` alone was not run separately; the first command covers it. `make umpire-check-scala` and `make lint-scala` were not run: the task changes no Scala, no IR and no lifter input. No `flowctl gate` receipt was attempted: the tree is dirty and the owner commits.


### Review round 1

Codex returned NEEDS_WORK with one introduced P1: the eight compositions were all `unsupported`. It is valid. Quint now evaluates every composition `goir` builds and agrees with it; the task stays `in_progress` and nothing is committed.

**What changed for the reader of the receipts.** Six of the eight compositions are compared exactly as machines are: `standaloneActivity`, `currentOverQueue`, `staleOverQueue`, `currentOverMatching`, `staleOverMatching` and `currentOverLossyMatching`. They add 886 reachable states, 17,701 state and class pairs and all 18 composition Properties (6,953 readings). The other two, `currentOverForgetful` and `currentOverVolatile`, have no composed table in Go: `goir` rejects the replacement in each (the two violating providers), so there is nothing to compare an export with. Their receipts are `unsupported` with `goir`'s rejection quoted as the reason, and neither declares a Property.

**The `goir` addition (widened Touches).** `Realizer.Composition(name)` in `goir/compose.go` returns a `Composed`: the declaration, the composed `*umpire.Table`, `State` and `Step` decoders, and the composition's Properties as `BoundProperty`. It reads `binding.subject` and `binding.propertyReads`, the code `Check` uses; there is no second construction. `claims.go` changed in one place: `Bound`'s inline Property binding became the shared helper `boundProperty`. `Check` and every existing goir test are unchanged. `r1-goir.diff` is the diff, 61 changed lines.

`goir/composed_test.go` (new, 3 tests) holds the reading to `Check`:
- every composition receipt's Definition ID, Behavior Fingerprint and row count equal the reading's table, on `activity` and `activity-system`, and every Query witness over a composition replays on it;
- a `refinement-rejected` composition has no reading, with the same `RefinementError` kind;
- states and steps decode to the composition's state record with outcome and facts as strings, and `atMostOneActive` read through the reading fails exactly on the last step of `Check`'s own counterexample for `staleOverQueue`;
- a composition past `Scope.Compose.States` is a `*umpire.ComposeLimitError`, and a machine's name or an unknown name is an error.

**The Quint side.** `backends/composed.go` (new, 397 lines) writes a composition from the IR declaration over the member machines the module already holds: own classes, sync pairs, the product of members' results with the first member's outcome and both members' facts, the product of starts, `ends`, the same fixed-point reach, and the Properties. A claim reads composed outcomes and facts as strings and Quint builds none, so the module spells each member's outcomes and facts out, one literal per value.

**Receipts for the review's points.**
- Replacement: the three compositions whose replacement holds are exported as composed tables, and each has a `module-refinement` receipt saying the replacement is `goir`'s verdict and no refinement is exported.
- Unknown pairs: a composition with one is refused. No test reaches that branch, because a member with a hole row is refused first, at the machine (`TestQuintExportRejectsWhatItDoesNotTranslate/a_value_no_case_matches` on the activity system, whose compositions use that member).
- Ceilings: `OpenWithin(m, scope)` binds within a scope. A composition past it has a `resource-limit` receipt and no text in the module (`TestACompositionPastTheCeilingIsAResourceLimit`).
- Apalache was not run on compositions. No receipt claims it was.

**Final run:** 146 receipts: 78 agreed, 37 covered, 30 unsupported, 1 not run (`r1-agreement-*/receipts.txt`). 35 subjects (29 machines, 6 compositions), 2,552 reachable states, 42,506 pairs (9,748 enabled with 10,292 results, 32,758 disabled), 2,378 monitor product steps, 52,772 Property readings of 167 Properties. The 30 unsupported are 9 machine refinements, 3 replacements, 2 rejected compositions, 10 progress claims, 4 Query receipts and P's 2 module-refinement receipts.

**Red, then green.**
- `r1-red-01-goir-composed.log`: the goir tests fail to build on `r.Composition undefined`.
- `r1-red-02-compositions.log`: the backend tests fail to build on `x.Compositions undefined`.
- First real Quint run failed to parse: a class variant and a spelling function shared the name `c0_o0` (`r1-quint-01.log`). Fixed by naming variants `C<j>_…`; `r1-quint-02.log` is green, all six compositions agreed on the first evaluation.
- Real-Quint mutants: a step's condition inverted makes `currentOverMatching` disagree on transitions, and a composition's Property negated makes `staleOverQueue` disagree on Properties.
- `TestAgreementRejectsATamperedComposition` (5 cases) and five comparison mutants (`r1-mutants-01.log`): 3 killed, 2 survive. One swaps the order of a sync's two input suffixes, which no slice can show because no sync has inputs on both sides. The other removes the unknown-pair refusal, which is unreachable as said above.

**Decisions.**
1. `Composition` takes a name. A `ClaimKey` would add a family that the name already determines.
2. A composition's Property about one composed class (`when_class`) is refused by the exporter. No slice declares one.
3. `TestCompositionsAreExported` pins the six names and the 18 Properties.

**Files this round.**
- Changed: `model/scalav2/goir/claims.go`, `model/scalav2/goir/compose.go` (before-copies under `.flow/tmp/fn-107/task8-before/` in the copy, byte-equal to the main checkout's current files), `model/scalav2/README.md`, and in `model/scalav2/backends/`: `slice.go`, `quint.go`, `itf.go`, `agreement.go`, `quint_test.go`, `p_test.go`, `encode_test.go`, `README.md`.
- Added: `model/scalav2/goir/composed_test.go`, `model/scalav2/backends/composed.go`.

**Gates.** The machine was under load from other sessions; times are 3 to 5 times the first round's.

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (after the sync, before any edit) | 0 |
| same, final (`backends` 298 s, `goir` 206 s, `goir/conformance` 15 s, `goir/testpilot` 270 s) | 0 |
| `model/scalav2/backends/run.sh --out <dir>` (587 s, no skip) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/backends/... ./model/scalav2/goir' GOLANGCI_LINT_FIX=false` (0 issues, after 4 fixed) | 0 |

No recursive delete ran this round, and nothing called `lake` or `lean`. `SEMANTICS.md`'s "What goir implements" does not mention the new reading; it is outside the Touches.

Review round 2 (conductor): SHIP with two P3 notes. One is fixed by the conductor (the backends README names both comparison paths, `goir.Build` for machines and `Realizer.Composition` for compositions). The other stands as a recorded limit: no selected sync has inputs on both sides, so the suffix-order decoder in `composed.go` has no test that could tell a swap; add a two-input sync fixture before using the exporter on a broader slice.

The task was implemented in a directory copy; its files (`model/scalav2/backends/**`, `goir/claims.go`, `goir/compose.go`, `goir/composed_test.go`, four hunks of `model/scalav2/README.md`) were merged into the main checkout by path while task 19 was running there. On the merged tree `go vet` passes, the goir composition and bound-claim tests pass, `go test ./model/scalav2/backends/` passes, and the scoped lint was run. `model/scalav2/backends/run.sh` (the full Quint and P agreement, about 10 minutes) was run by the implementer in the copy, not again by the conductor after the merge.

Tools live outside the repository: .NET 8.0.425 and P 3.1.0 under `/tmp/umpire-backend-tools`, Apalache 0.62.1 under `~/.quint`, Quint 0.33.0 through `model/quint/quint.sh`.

The work is uncommitted; the owner makes the commits. The review outputs are under `.flow/tmp/fn-107/task8/`.

stage: implement - ran (worker subagent in a directory copy, session model claude-opus-5-5; one fix round)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f807-d105-79f1-8837-5ba4ec8412c5; round 1 NEEDS_WORK with one P1 (compositions not exported), fixed; round 2 SHIP with two P3)
stage: wave-join - ran (files copied by path into the main checkout; README hunks applied as a patch)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0 before any edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), model/scalav2/backends/run.sh --out <dir> (rc=0; 131 receipts: 66 agreed, 31 covered, 33 unsupported, 1 not-run; Quint 0.33.0, Apalache 0.62.1, P 3.1.0), GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/backends/...' GOLANGCI_LINT_FIX=false (rc=0), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0), review round 1 baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0 after the workspace sync, before any edit), review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), review round 1: model/scalav2/backends/run.sh --out <dir> (rc=0; 146 receipts: 78 agreed, 37 covered, 30 unsupported, 1 not-run; 29 machines and 6 compositions), review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0), review round 1: GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/scalav2/backends/... ./model/scalav2/goir' GOLANGCI_LINT_FIX=false (rc=0), conductor, merged tree: go vet -tags test_dep ./model/scalav2/goir/ ./model/scalav2/backends/ (rc 0), conductor, merged tree: go test -tags test_dep -short -run 'Compos|Bound|Realizer' ./model/scalav2/goir/ (rc 0), conductor, merged tree: go test -tags test_dep ./model/scalav2/backends/ (rc 0), implementer, in the copy: model/scalav2/backends/run.sh (rc 0; 146 receipts: 78 agreed, 37 covered, 30 unsupported, 1 not run), codex impl-review rounds: .flow/tmp/fn-107/task8/t8-r1.md, t8-r2.md (final SHIP)
- PRs: