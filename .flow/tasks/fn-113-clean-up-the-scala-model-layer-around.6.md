---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.6 Audit: map every munit test to a Go test over the IR

## Description
Audit: map every munit test to a Go test over the IR. Implements R14; the committed audit is what allows tasks 7 and 8 to delete tests.

**Size:** M
**Files:** .plans/umpire-scala-evaluator-audit.md (new); new or extended Go tests under tools/umpire/model (fixtures from `model/ir/*.json` and `model/lifter/testdata/lifts/expected/*.json`)
**Touches:** [.plans/umpire-scala-evaluator-audit.md, tools/umpire/model/*_test.go (new or extended tests only), tools/umpire/model/testdata/** (new fixture inputs only if a claim needs one)]

### Approach
- The suites, at planning time: `model/umpire/test/Declarations.test.scala` (16 tests, lines 175-415), `model/temporal/test/NexusCallerPins.test.scala` (9, lines 24-300), `model/temporal/test/StandaloneActivityPins.test.scala` (6, lines 26-164), `model/temporal/test/NexusKernel.test.scala` (2). That is 33 tests; the spec says 32. Audit all 33 and note the count.
- For each test: suite, line, the claim in one sentence, and one outcome: A, an existing Go test over the IR that asserts the same claim (name and package); B, a new Go assertion this task adds (fixtures built from IR through `Load`/`Check`/`Build`, never a typed constructor; `require`); C, a recorded reason the claim no longer applies (for example "the native table refuses a channel, a hole or a monitor, which only the IR interpreter reads"; "coverage targets have no IR form, task 7 records the authored facts"; "the kernel dispatch existed for the proof lemmas, task 13 removes it"; "a native step-function test stays, since step functions remain executable Scala").
- Leads: channels, `interpret_test.go` (`TestChannelCatalogs`, `TestChannelSendAndDelivery`, `TestUnorderedSendKeepsCatalogOrder`, `TestASendToAFullChannelLeavesTheDomain`); stutters and visible projections, `checking_test.go:527 TestRefinementControls`, `interpret_test.go:278`, `internal/checker/refinement_test.go`; "names what a refined machine sees must refine one", `admission_test.go:500`; replacement, `checking_test.go:468`; holes, `interpret_test.go:210`, `bound_test.go:171`; monitors, `checking_test.go:230-242`, `interpret_test.go:300`; table sizes, starts, reachable, stuck, `parity_test.go:24` and the semantics snapshots under `testdata/migration/semantics`; Query answers and explored counts, the golden Query answers; the canary silent-step rule and set checks (`check(...)`), look for a Go admission counterpart in `tools/umpire/lower` and `tools/canary/assessment`, else outcome C with the facts task 7 records.
- The audit file is outside `model/`, but keep it free of the retired front end's vocabulary; name the Scala line of each test so tasks 7 and 8 delete by reference. No Scala file changes in this task.

### Investigation targets
**Required**:
- `model/umpire/test/Declarations.test.scala:1-60,175-415`
- `model/temporal/test/NexusCallerPins.test.scala:24-300`
- `model/temporal/test/StandaloneActivityPins.test.scala:26-164`
- `model/temporal/test/NexusKernel.test.scala:34-54`
- `tools/umpire/model/interpret_test.go:109-300`, `tools/umpire/model/checking_test.go:230-600`, `tools/umpire/model/admission_test.go:479-520`, `tools/umpire/model/parity_test.go:24`, `tools/umpire/model/fixtures_test.go` (how tests load IR fixtures)
- `.plans/umpire-migration-claims.json` (the precedent for a claim inventory)

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '<new tests>' ./tools/umpire/model; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `.plans/umpire-scala-evaluator-audit.md` lists all 33 munit tests with suite, line, claim and one of the three outcomes; every A names an existing Go test over the IR that asserts the same claim, every B names the Go test this task added, every C states why the claim no longer applies.
- [ ] The added Go tests build their fixtures from IR, pass, and `lint-code-fast` passes; no Scala file changed.
- [ ] The summary notes the 33 versus 32 count for the conductor.


## Done summary
# fn-113.6 handover: audit of the munit tests against Go tests over the IR

### Status

- The audit is written: `.plans/umpire-scala-evaluator-audit.md` covers all 35 munit tests. The totals are A 6, B 20 and C 9; three of the B outcomes are provisional (see Review).
- **The move is done** (conductor go, 2026-10-03). `tools/umpire/model/{declarations,nexus,activity}_pins_test.go` (package `model`) now hold the 19 new tests (20 B claims). The scratch package was moved to `.flow/tmp/trash/fn113-6/fn113audit/` and the scratch dump test to `.flow/tmp/trash/fn113-6/dump_test.go.txt`. `gofmt` realigned one map literal in `nexus_pins_test.go` after the helper rename; nothing else changed against `.flow/tmp/fn113-6/staged/`.
- The tests were re-run against the IR the ScalaPB printer rewrote (`model/ir/*.json` and `model/lifter/testdata/lifts/expected/*.json`, timestamped 10-02 18:23). All 19 pass. The hand-written ProtoJSON for `nexusCaller`, `terminalHolds` and `stoppedWorkerRepliesNothing` still merges and checks with every Scala pin value: it refers only to declaration and function names, which the printer did not change.
- Lint scoped to the package passes with 0 issues.
- No Scala file changed, nothing was staged or committed, and nothing under `tools/umpire/model/internal/checker` or `tools/umpire/lower` was touched.

### The test count: 35, not 33 (task) or 32 (spec)

`model/umpire/test/Declarations.test.scala` has 18 tests, not 16: the tests at lines 367 and 382 open with a multi-line `test(`. The full count is 18 in Declarations, 9 in NexusCallerPins, 6 in StandaloneActivityPins and 2 in NexusKernel. The spec's "Decisions under the owner's delegation, Counts" says 33. It should say 35.

### Findings the conductor should know

1. **Three Nexus caller declarations are not lifted into any IR file.** They are the composition `nexusCaller` (Model.scala:205), `terminalHolds` (Claims.scala:245) and `stoppedWorkerRepliesNothing` with its property and scenario (Claims.scala:319-342). `model/gate/Roots.scala` lists only the machines, `functionalQueries` and the realization for `nexus-caller.json`, so once the native evaluator is removed nothing checks these claims. `nexus_pins_test.go` covers them for now by appending ProtoJSON declarations (`nexusCallerClaims`) to the lifted file, written in the shape the lifter gives the activity Model's equivalents. Every Scala pin value holds in Go unchanged: 316 states, 158 of them stopped, 1,468 rows, 144 reply rows, `stoppedWorkerRepliesNothing` verified and exercised, `terminalHolds` verified, and the free search within four exploring 111 states. **Open item (in the audit):** the conductor schedules adding `Claims$package$.terminalHolds` and `Claims$package$.stoppedWorkerRepliesNothing` as roots of `nexus-caller.json` and then drops the fragment. Under R13 this adds receipts to the frozen golden, so it needs the owner's agreement and a decision on whether to re-capture the golden or project it. It is out of scope here.
2. **Go repeats one Scala refusal word for word.** A product Property on an action the protocol lacks (`timesOutOnProtocol`) is a `DeclarationError` in Go with the same text as the Scala refusal.
3. **Go also refuses the monitor-on-a-composition-member Query** (Declarations:401), as unsupported and for a reason of its own. The audit counts it as A.
4. **The canary silent-step rule (NexusCallerPins:212) has a Go consequence that does not depend on the sets.** Lowering turns `backoff` into a capability Known Gap in `model/cases/nexus-caller-retry-case.json`, pinned by lower's `TestMigrationGoldens`, and the canary's Evaluation Profile blocks capability gaps. The audit records the claim as C, because the sets have no IR form.
5. **`StandaloneActivityPins.test.scala:164` stays.** It is a pure step-function test (C), so tasks 7 and 8 must not delete it.
6. **`/usr/bin/clang` does not exist in this sandbox.** The suggested `CC=/usr/bin/clang` fails cgo with "C compiler not found". Every run here dropped `CC` and used the default `gcc`.

### Checks run (logs under `.flow/tmp/fn113-6/`)

| Command | Result | Log |
| --- | --- | --- |
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run 'TestDump' ./tools/umpire/fn113audit/ -v` (scratch, deleted) | PASS. A fact-finding dump. | `dump.log` |
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/fn113audit/ -v` (scratch, before the move) | PASS, 19/19 | `scratch-tests.log` |
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run '^(<the 16 cited A tests>)$' ./tools/umpire/model/ -v` | PASS, 16/16 | `cited-a-tests.log` |
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run '^(<the 19 new tests by name>)$' ./tools/umpire/model/ -v` (after the move, against the ScalaPB-printed IR; `free -g`: 11 GB available) | PASS, 19/19 | `moved-tests.log` |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model` | First run: 1 gci finding (map alignment in `nexus_pins_test.go`), fixed with `gofmt -w`. Re-run: 0 issues. | `lint.log` |

Not run: the model gate (forbidden), `TestMigrationGoldens` (this task changes no input), the whole `./tools/umpire/...` suite (memory), and `make lint-code-fast`, which the conductor replaced with the scoped lint because other agents' files trip it.

### Review (NEEDS_WORK, applied)

1. **Blocking.** In the audit, `NexusCallerPins.test.scala:156` (its `terminalHolds` part), `:257` and `:300` are now **B (provisional)**, because they check a hand-written IR copy of `nexusCaller`, `terminalHolds` and `stoppedWorkerRepliesNothing`. "What tasks 7 and 8 may delete" gained an open item: task 8 deletes those three tests only after the three declarations are gate roots of `nexus-caller.json`. The conductor schedules that lift, which adds receipts to the frozen goldens (R13, under the owner's delegation). `:190` and `:198` stay plain B, because their Queries are test-local.
2. In `nexus_pins_test.go`, the fragment's comment now says "written in the lifter's shape with declaration positions only".
3. `nexus_pins_test.go` now reuses `load(t)` (diagnostics_test.go) and `machines(t)` (parity_test.go). `nexusCallerModel` is gone, and `TestNexusProtocolTable` takes both machines from one built map.
4. `Declarations.test.scala:401` is reclassified A→C (the native search's refusal goes with the search in task 8). `TestUnsupportedDeclarationsAreListedAndNeverCounted` is kept as a side note. The totals are now A 6, B 20 and C 9.
5. The `NexusCallerPins:300` and `StandaloneActivityPins:146` rows now each have one outcome: "B (set part: C, per :212)". The activity row records that the canary silent-step half goes with the sets.
6. In `declarations_pins_test.go`, `inbox`'s comment says it builds only the probe expression by hand, and that the Model comes from the IR.

After the changes, `free -g` showed 11 GB available. The 19 tests pass by name (`review-tests.log`), and the scoped lint `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model` reports 0 issues (`review-lint.log`).

### R25 (libraries)

No generic machinery was written, and no library was weighed. The only helper of note is `withDeclarations`, six lines that merge ProtoJSON with `protojson` and `proto.Merge`, both already dependencies.

### Shared-document changes for the conductor

- Spec `fn-113-...md`: in "Decisions under the owner's delegation, Counts", change "33 munit tests" to 35. R14 and the spec's "32 munit tests" text could say 35 as well.
- `MILESTONES.md` and the migration manifest: record that the R14 audit is `.plans/umpire-scala-evaluator-audit.md`, and that the three unlifted Nexus caller declarations are covered by appended fixtures until they become gate roots (finding 1).
- `.plans/UMPIRE_MODULES.md`: no change.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 63ab0f12d64ac85596a47803a9c6707c343416a0
- Tests: GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run '^(<19 new audit tests by name>)$' ./tools/umpire/model/ -v -> PASS 19/19 after review fixes (log .flow/tmp/fn113-6/review-tests.log), GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run '^(<16 cited outcome-A tests>)$' ./tools/umpire/model/ -v -> PASS 16/16 (log .flow/tmp/fn113-6/cited-a-tests.log), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model -> 0 issues (log .flow/tmp/fn113-6/review-lint.log), independent review (claude-opus-5-5, fresh context): NEEDS_WORK; all six findings applied by the implementer, 19 tests and scoped lint pass after
- PRs: