# fn-108 final evidence: size comparison, equivalence and gates

Recorded 2026-10-01 by task fn-108.7 on darwin/arm64. This task changed no source file. It
measured the tree that tasks fn-108.2 to fn-108.6 left, ran the final gates on it, and wrote this
record. Nothing is staged or committed.

## Result

| Requirement | State | Basis |
| --- | --- | --- |
| R1 size | met | authored production Go fell by 286 code lines and 10691 code bytes; `size-compare.sh` exits 0 |
| R8 preservation | met on darwin/arm64; "Carried items" lists the diagnostic differences and the one widened import allowance | public `go doc` and CLI captures identical to the baseline; protected paths untouched; no comment lost |
| R9 gates | **incomplete** | every darwin/arm64 gate ran and none shows a new failure; no linux/amd64 gate ran (no host), so the both-platform requirement is open |

`make lint-code-fast` exits 2 on this branch, as it did before fn-108. It reports no issue in any
file fn-108 touched because it cannot type-check the nested module. See "Gates".

## Tree and host

| Item | Value |
| --- | --- |
| Baseline revision | `6782b55f49a0317b230e827ea2a63a37d116d502`, which is also `HEAD` |
| Measured tree | `HEAD` plus the uncommitted fn-108.2 to fn-108.6 edits: 24 modified tracked files and 7 untracked Go files under `tools/gomad3` ([final-protected-paths.txt](final-protected-paths.txt)) |
| Diff against the baseline | [final-working-tree.diff](final-working-tree.diff), SHA-256 `47afb287…14513855`, plus the 7 untracked files |
| Platform | Darwin arm64, macOS 26.6.2 (25G83) |
| Host `go` | go1.27.1, from `$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin` first on PATH |
| Patched toolchain | `tools/gomad3/.toolchain/bin/go`, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc` |
| Runner build | `sha256:f8b0a8d42df61c41cfc007ca19a5891a76b0fe50f9c5e8061c9ab5279a93737f` (`gomad doctor`) |

Every gate line in [final-gates/results.txt](final-gates/results.txt) carries a fingerprint of the
three directories taken before and after the gate. All are `a875ec2570434ad6`, so no gate ran on
a different tree and no gate modified the tree. `size-count.sh` printed the same table before and
after the gates.

`shasum -a 256 SHA256SUMS` printed `a64ac192…a1a64327` and `shasum -a 256 -c SHA256SUMS` reported
all 74 files `OK`, so the three scripts and every stored baseline output are the fn-108.1 versions.

## Size (R1)

Commands, from the repository root, counting rule v2 of [baseline.md](baseline.md), scripts unmodified:

```
A=.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing
sh "$A/size-count.sh" > "$A/final-size.txt"
SIZE_COUNT_DETAIL=1 sh "$A/size-count.sh" > "$A/final-size-files.txt"
sh "$A/size-compare.sh" "$A/size-baseline-files.txt" "$A/final-size-files.txt" > "$A/final-size-compare.txt"
```

Outputs: [final-size.txt](final-size.txt), [final-size-files.txt](final-size-files.txt),
[final-size-compare.txt](final-size-compare.txt). `size-compare.sh` exited 0 and printed
`R1 size condition: PASS`.

Totals across `tools/gomad3`, `tools/gomad3sim` and `tools/gomad3integration`:

| Class | Baseline files | Final files | Baseline physical | Final physical | Baseline code | Final code | Code change | Baseline code bytes | Final code bytes | Byte change |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| production-go (authored) | 252 | 254 | 63187 | 62914 | 58624 | 58338 | **-286** | 1970324 | 1959633 | **-10691** |
| overlay-go (runtime overlay) | 35 | 35 | 12852 | 12852 | 11462 | 11462 | 0 | 318423 | 318423 | 0 |
| test-go | 279 | 284 | 45753 | 47854 | 42849 | 44778 | +1929 | 1429606 | 1499935 | +70329 |
| generated-go | 19 | 19 | 4514 | 4514 | 4124 | 4124 | 0 | 213092 | 213092 | 0 |
| protocol-input (templates, patch, JSON, assembly, scripts, make) | 56 | 56 | 14400 | 14400 | 13926 | 13926 | 0 | 663245 | 663245 | 0 |
| other (Markdown, text, go.mod) | 77 | 77 | 7409 | 7413 | 5918 | 5922 | +4 | 419353 | 419583 | +230 |
| all | 718 | 725 | 148115 | 149947 | 136903 | 138550 | +1647 | 5014043 | 5073911 | +59868 |

Production Go by directory:

| Directory | Baseline code | Final code | Change |
| --- | --- | --- | --- |
| `tools/gomad3` | 50984 | 50698 | -286 |
| `tools/gomad3sim` | 7640 | 7640 | 0 |
| `tools/gomad3integration` | 0 | 0 | 0 |

Production physical lines fell by 273 and production code lines by 286. The 13-line difference
is the net change in comment and blank lines. The two new files add 34 (22 comment lines, 12
blank lines). Existing production files lose 21 (the 6 comment lines listed under "Comments and
formatting" and 15 blank lines that went with deleted or merged code).

Production files that changed, in code lines (new helpers, types and files included):

| File | Baseline | Final | Change | Task |
| --- | --- | --- | --- | --- |
| `deterministicio/domain.go` | 108 | 55 | -53 | .2 |
| `deterministicio/memory_adapter.go` | 104 | 47 | -57 | .3 |
| `qualification/set/set.go` | 1043 | 1014 | -29 | .2 |
| `runner/campaign.go` | 76 | 73 | -3 | .2 |
| `runner/choice_exploration_campaign.go` | 599 | 546 | -53 | .5, .6 |
| `runner/completion.go` (new) | 0 | 55 | +55 | .5 |
| `runner/deterministicio.go` | 54 | 47 | -7 | .2 |
| `runner/guidance.go` | 108 | 104 | -4 | .6 |
| `runner/internal/campaign/campaign_journal.go` | 595 | 592 | -3 | .2 |
| `runner/internal/campaign/merge.go` | 633 | 632 | -1 | .2 |
| `runner/internal/minimizer/minimizer.go` | 318 | 302 | -16 | .2 |
| `runner/minimize_operation.go` | 480 | 476 | -4 | .6 |
| `runner/portable_plan.go` | 355 | 350 | -5 | .2 |
| `runner/retention.go` (new) | 0 | 56 | +56 | .6 |
| `runner/runner.go` | 1870 | 1787 | -83 | .5, .6 |
| `runner/simulation_exploration_campaign.go` | 571 | 521 | -50 | .5, .6 |
| `target/capability.go` | 1255 | 1245 | -10 | .2 |
| `upgrade/upgrade.go` | 515 | 496 | -19 | .4 |
| sum | | | **-286** | |

Runtime patch, schema and template deltas: none. `protocol-input` is unchanged in every kind
(12 templates, 1 patch, 37 JSON files, 1 assembly file, 3 scripts, 2 make files), `overlay-go` and
`generated-go` are unchanged, and no file changed class. The residual of `size-compare.sh`, which
adds any growth in those classes to the production change, equals the production change.

Checks the scripts cannot make ([baseline.md](baseline.md), "Comparison rule for the final task"):

1. Scripts and baseline outputs are byte-identical to fn-108.1 (manifest check above).
2. The `generated-go` file list is unchanged.
3. No Go code moved out of the inventory. `git diff --stat 6782b55f4 -- '*.go'` outside the three
   directories is empty and no untracked `.go` file exists outside them.
4. `gofmt -l` over every Go file of the inventory prints nothing.
5. Increases outside production:
   - `test-go` +1929 code lines in 11 files. New files: `runner/completion_characterization_test.go`
     (+422), `runner/completion_test.go` (+141), `runner/retention_characterization_test.go` (+772),
     `runner/retention_test.go` (+128), `upgrade/upgrade_unix_test.go` (+62). Grown files:
     `upgrade/upgrade_test.go` (+241), `deterministicio/memory_adapter_test.go` (+96),
     `runner/internal/campaign/merge_capacity_test.go` (+39), `runner/portable_plan_test.go` (+33),
     `runner/runner_test.go` (+6). One file shrank: `runner/internal/minimizer/minimizer_test.go`
     (-11), the round-trip lines that exercised the removed `Encode`/`Decode`.
   - Markdown +4 code lines: the R5 sentence in `tools/gomad3/ARCHITECTURE.md` on the stronger
     sync and cleanup reporting of `hostfs.Replace`.

## Comments and formatting (R8)

[final-comment-lines.txt](final-comment-lines.txt) lists, for every changed or new Go file, the
comment lines of the baseline file that are absent from the final file. A scratch program built
on `go/scanner` produced it, so text inside string literals does not count as a comment.

- Comment lines in the changed and new Go files: 89 at the baseline, 210 now.
- Removed comment lines: 6. They are three copies of one two-line comment.

| Baseline location | Text | Where it is now |
| --- | --- | --- |
| `runner/runner.go:894-895` | `// A target the watchdog or a cancellation killed wrote no choice trace` / `// to project; the termination is its outcome.` | `runner/completion.go:58-59`, rewrapped |
| `runner/choice_exploration_campaign.go:346-347` | the same sentence | `runner/completion.go:58-59`, identical lines |
| `runner/simulation_exploration_campaign.go:363-364` | the same sentence | `runner/completion.go:58-59`, identical lines |

The acceptance text expects every removed comment line to belong to deleted dead code. That is
not what happened, so it is stated here as measured. The dead code that fn-108.2 deleted carried
no comment. The six lines sat above three copies of the condition
`coverageHasChoice(...) && choiceTraceObserved(...)`. fn-108.5 merged the three copies into
`assessCompletion`, and the comment stands above the one remaining condition. No comment was
removed from code that is still live without its text surviving beside that code.

The `git diff` also removes one line that contains `//go:linkname` in `memory_adapter.go`. It is
part of a string literal (the replacement source text), which moved unchanged into
`memoryRewrites`; fn-108.3 showed the replacement bytes equal.

Formatting was not compressed. `gofmt -l` is empty, and `codebytes`, which is the same for every
layout of one token sequence, fell together with `code`.

## Public surface and protected paths (R8)

`sh api-capture.sh <final-api> <scratch bin>` ran unmodified and exited 0; the capture is
[final-api/](final-api/). `diff -r api-baseline final-api` exits 0 with empty output
([final-api-diff.txt](final-api-diff.txt), 0 bytes). That covers `go doc -all` of the 19 public
packages of the nested module and of `gomad3sim`, and the 39 CLI usage and `--help` captures of
`gomad` and `gomadtool` with their exit statuses and stream routing. The captures are the
darwin/arm64 view. A linux/amd64 capture was not taken and no linux baseline capture exists.

`git diff --stat 6782b55f4` and `git status --short --untracked-files=all` are both empty for
`tools/gomad3/internal/compatibilitypack`, `tools/gomad3/qualification/*.json`,
`tools/gomad3integration/qualification`, every `schema/` directory, every `*.json`, `*.tmpl` and
`*.patch` file under `tools/`, `tools/gomad3/toolchain/runtime`, `tools/gomad3sim` and
`tools/gomad3integration` ([final-protected-paths.txt](final-protected-paths.txt)). Compatibility
packs, qualification manifests, schemas, the runtime patch and the overlay are untouched.
`make -C tools/gomad3 validate` passed, so every generated file is current and no code moved into
a generator input.

## Gates (R9), darwin/arm64

Each gate ran once on the final tree through `run-gate.sh`, which keeps the combined output as
`final-gates/<name>.log` and one line with exit status, UTC times and load in
[final-gates/results.txt](final-gates/results.txt). No log contains `(cached)`.

| Gate | Command | Result | Seconds | Baseline disposition | Disposition now |
| --- | --- | --- | --- | --- | --- |
| validate | `make -C tools/gomad3 validate` | exit 0; generators `-check`, patch, script, compatibility packs, `TestHostPacksBindCurrentProfile`, qualification manifest | 2 | exit 0 (fn-108.1) | unchanged |
| test | `GOFLAGS=-count=1 make -C tools/gomad3 test` | exit 0; `gomad3 all black-box tiers passed`: harness (3 packages ok), toolchain, interception, host (45 packages ok, 1 without tests, includes `architecture_test.go`), overlay (4 ok), world under `-race` (3 ok), builder, live-capability, runtime, upstream | 1197 | harness, host and world exit 0 (fn-108.1). The whole target exit 0 in fn-105.14 on the sources that became the baseline revision (`fn105-d14-make-test.log`) | unchanged |
| gomad3sim | `go test -count=1 -tags test_dep ./tools/gomad3sim/...` | exit 0, 1 package ok | 2 | exit 0 (fn-108.1) | unchanged |
| integration | `make gomad3-integration-test` | exit 0, 1 package ok | 13 | exit 0 (fn-108.1) | unchanged |
| smoke qualification | `make gomad3-smoke-qualification` | exit 0; `expectations-met=true supported=4 unsupported=0 failed=0 infrastructure-errors=0 completed=4/4`; seed 11; 4 replayed, 0 diverged | 343 | 4/4 in fn-105.14 (`fn105-d14-smoke-qualification.log`) | unchanged |
| core qualification | `make -C tools/gomad3 core-qualification` | exit 0; 9 compatibility-pack requests qualified; `expectations-met=true supported=7 unsupported=0 failed=0 infrastructure-errors=0 completed=7/7`; seed 17; 7 replayed, 0 diverged | 344 | none at the baseline revision. fn-108.3 ran it on its mid-spec tree with the same counts | meets the manifest expectations |
| Temporal representative set | `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` | exit 0; 9 compatibility-pack requests qualified; `expectations-met=true supported=28 unsupported=0 failed=0 infrastructure-errors=0 completed=28/28`; seeds 11 and 17; 56 seed-runs replayed with `choice_replay_exact`, 0 diverged, 0 timed out | 2361 | none. fn-105.14 states the representative set was not rerun on toolchain key `8d28bd44` | meets the manifest expectations (28 of 28 `qualified` on darwin/arm64) |
| lint | `make lint-code-fast` | exit 2. See below | 1 and 162 | exit 2 in fn-105.14 (`fn105-d14-lint.log`) | pre-existing failure; no report in an fn-108 file |

Supporting runs on the same tree:

| Check | Command | Result |
| --- | --- | --- |
| Build | `make gomad3` | exit 0, Runner rebuilt from the final tree |
| Format | `gofmt -l` over the inventory's Go files | no output |
| Vet | `.toolchain/bin/go vet -tags test_dep` over the `test-host` package set | exit 0, no output |
| Vet, cross type-check | the same with `GOOS=linux GOARCH=amd64` | exit 0. This is a compile check on darwin. It is not a linux gate |
| Per-test listing | the five `go test` recipes of the baseline listing with `-count=1 -json` added | all exit 0 |

**Per-test dispositions.** [final-gates/test-dispositions-darwin-arm64.tsv](final-gates/test-dispositions-darwin-arm64.tsv)
has 1163 top-level tests: 1142 pass, 21 skip, 0 fail. All 1140 baseline tests are present with
the result they had at the baseline (1119 pass, 21 skip). The 23 added tests all pass: 3 in
`deterministicio`, 15 in `runner`, 1 in `runner/internal/campaign`, 4 in `upgrade`. The listing
equals the one fn-108.6 took before it renamed three test helpers, so that rename changed no
test result. These runs are the full-tier verification of the renamed helpers that fn-108.6
left open.

**Lint.** The first run stopped after 1 second with
`GOLANGCI_LINT_BASE_REV=main is not a known commit`. The checkout has no local `main` branch, so
that run observed nothing and is inconclusive
([final-gates/lint-code-fast.log](final-gates/lint-code-fast.log)). The rerun
`make lint-code-fast GOLANGCI_LINT_BASE_REV=stephanos/main` (merge base `951c5516e9`) exited 2
([log](final-gates/lint-code-fast-base-stephanos-main.log)). golangci-lint 2.13.0 printed
`0 issues.` and exited 7 because it logged type-check errors of the form
`main module (go.temporal.io/server) does not contain package go.temporal.io/server/tools/gomad3/...`
for packages of the nested module. The `errortype` vet step after it did not run.

Classification against fn-105.14's run of the same command before fn-108:

- The type-check errors for the nested module are in both runs. They are pre-existing: the root
  module's linter cannot load `tools/gomad3`.
- fn-105.14 also had three `staticcheck` SA1019 reports in `tests/timeskipping_propagation_test.go`
  and `tests/versioning_3_query_test.go`. This run does not show them. Its base revision differs
  and why they are absent was not investigated. fn-108 touches no file under `tests/`.
- Reports in files fn-108 touched: none in either run. Every Go file fn-108 touched is in the
  nested module, which this linter does not analyse, so the lint gate gives no signal for them.
  `gofmt` and `go vet` above are the checks that cover those files.

**Host load.** The machine was not quiet. Another session ran unrelated Go and JVM test suites
(`umpire`) during these gates, and the 1-minute load average ranged from 4 to 32 on 8 cores. No
gate reported a watchdog timeout or an infrastructure error, so no rerun was needed.

**Qualification reports.** `final-gates/` holds the smoke and core set reports, a row listing per
workload and seed for all three sets (`*-qualification-rows.tsv`), and the top-level counters of
the Temporal report (`temporal-qualification-set.summary.json`). The full Temporal report is
13.8 MB and is kept outside git at `tools/gomad3/.toolchain/fn-110/baseline/temporal-set-report.json`
(SHA-256 `2bbb4659…4cc290d7`), next to copies of the smoke and core reports and `identity.json`,
for fn-110 to use as its "before" outcomes.

The `evidence_sha256` of a seed covers the Runner build identity. The seven core digests differ
from the ones fn-108.3 recorded because fn-108.5 and fn-108.6 rebuilt the Runner; classification,
replay results, closure digests and trace sizes are equal, and three per-seed artifact byte
totals differ by 2 to 3 bytes (not investigated). The four smoke digests equal the ones from fn-108.6's run,
which used the same Runner build. Neither comparison says anything about the baseline revision,
because no qualification report from the baseline Runner build exists.

## Gates not run: linux/amd64

No linux/amd64 host was available. None of the following ran on that platform, and R9's
"full Gomad gates on both supported platforms" is incomplete until CI or a Linux host runs them.

| Gate | Command on a linux/amd64 host with go1.27.1 first on PATH | State |
| --- | --- | --- |
| validate | `make -C tools/gomad3 validate` | not run (no linux/amd64 host). Expected to fail in `TestHostPacksBindCurrentProfile` on the stale pack `modernc-libc-xsys-v047-linux-amd64`, a finding that predates fn-108 |
| test | `make -C tools/gomad3 test` | not run (no linux/amd64 host) |
| gomad3sim | `go test -count=1 -tags test_dep ./tools/gomad3sim/...` | not run (no linux/amd64 host) |
| integration | `make gomad3-integration-test` | not run (no linux/amd64 host) |
| smoke qualification | `make gomad3-smoke-qualification` | not run (no linux/amd64 host) |
| core qualification | `make -C tools/gomad3 core-qualification` | not run (no linux/amd64 host) |
| Temporal representative set | `make gomad3-qualification GOMAD3_QUALIFICATION_PRUNE=1` | not run (no linux/amd64 host). D12 makes the F5 and F6 suites `intermittent` there |
| public surface | `sh .flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/api-capture.sh OUTDIR BINDIR` | not run (no linux/amd64 host); no linux baseline capture exists to compare with |
| memory adapter pin | covered by `make -C tools/gomad3 test` (`deterministicio`, `memoryPreparedSourceSetSHA256` for linux/amd64) | not run (no linux/amd64 host) |

The toolchain builder downloads the Go source archive from go.dev, so the host needs that access.

## Fixed-input regression evidence

The characterization tests drive `Explore` and `Minimize` through fake executors with fixed
identities, so their projections do not depend on the Runner build. The `-json` run of the
`test-host` recipe on the final tree logged the same projection lines the tasks captured before
their extractions:

| Capture | Lines | Compared with | Result |
| --- | --- | --- | --- |
| [final-gates/retention-projections-final.txt](final-gates/retention-projections-final.txt) | 31 | `task6-projections-before.txt`, taken on the pre-extraction production files | `cmp` equal, SHA-256 `c9448c30…89756f6` |
| [final-gates/completion-record-inputs-final.txt](final-gates/completion-record-inputs-final.txt) | 3 | `task5-canonical-record-inputs-before.txt` with its log prefix removed | `cmp` equal |

Paths and the tests that exercise them (all in `tools/gomad3/runner` unless a path is given):

| Path | Characterization written before the extraction | Existing tests that pass before and after |
| --- | --- | --- |
| Ordinary seed | `TestCompletionFaultsKeepReasonPrecedenceAndEvidence` (16 faults, seed column), `TestCompletionProjectsWorldCoverageAndChoicesForEveryStrategy`, `TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy`, `TestRetentionCapacityExhaustionFailsVisiblyForEveryStrategy`, `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` | `TestRunCancellationIsAHostFailure` and the seed cases of `runner_test.go` |
| Guided | `TestGuidedAdmissionReplaysBeforeTheCorpusAdvances` (exact, diverged, unverified and failed replay, incomplete transcript) | `TestRunGuidesFromImmutableCorpusAndKeepsUnguidedSeeds`, `TestRunGuidesFromReplayVerifiedChoiceCoverage`, `TestRunResumesGuidedBatchWithoutReselectingSeeds`; a guided campaign through the built CLI in fn-108.6 |
| Choice exploration | the choice cases of the three completion tests and of the retention tests; `TestExplorationCancellationIsAHostFailure` | `TestRunChoiceExplorationResumeRerunsTheWholeIncompleteRound`; a choice-exploration campaign through the built CLI in fn-108.6 |
| Simulation exploration | the simulation cases of the same tests | `TestRunSimulationExplorationExecutesRootAndEveryScenarioRank`, `TestRunSimulationExplorationRetainsExactDeduplicatedSimulationFailure` |
| Retention | the four retention characterization tests; direct tests in `retention_test.go` | `TestRunResumeRestoresSeenChoiceFeaturesBeforeNovelRetention`, `TestRunResumeRejectsTamperedRetainedSuccessArtifact` |
| Interruption and resume | `TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState` (incomplete transcript, unusable store, cancellation; pins the interrupted state per strategy, resumes, pins the result) | `TestRunResumesVerifiedBatchAndSkipsCompletedOrdinals`, `TestRunSimulationExplorationResumePreservesCommittedCandidates`, the journal resume tests in `runner/internal/campaign`, `TestRunRecover*` and `TestRunResume*` in `cmd/gomad/internal/cli` |
| Minimization | none | `TestMinimizePublishesLinkedExactScheduleAndFaultReduction` and `TestMinimizeRejectsSimulationArtifactWithoutExactChoiceTape` in `minimize_operation_test.go`; `TestExecutionArtifactInputCarriesTheCapturedEvidence` for the shared input constructor |

Gaps:

- **Minimization** has no before-and-after projection. Its evidence is the two unchanged tests
  of `minimize_operation_test.go`, which pass at the baseline and now, and the direct test of
  `executionArtifactInput`. The only fn-108 change on that path replaces a literal
  `artifact.ArtifactInput` with that constructor plus the `Simulation` field
  (`minimize_operation.go`, -4 code lines).
- **Simulation exploration** has no run through the built CLI in fn-108. Its fixed-input evidence
  is unit-level, through the fake executors.
- **`recover`** is covered by existing tests only. fn-108 added no characterization for it and
  changed no code in `runner/internal/campaign` beyond the removed unused wrapper and the merged
  canonical check.
- The Temporal, smoke and core qualification sets exercise the ordinary seed path with replay
  through the real Runner. They do not exercise guidance, exploration or minimization.

## Carried items from the tasks

Behaviour that differs from the baseline, all in diagnostics or unreachable code:

- fn-108.2: `OpenMergedCampaign` returns `errors.New("merged campaign record is invalid")` where
  it returned `errors.Join(errors.New(<same text>), err)` with a nil `err`. Message and
  `errors.Is`/`errors.As` results are the same.
- fn-108.3: eleven diagnostic strings of the memory adapter read `modernc.org/memory` where they
  read `modernc memory`, four of them now name `mmap_unix.go`, and the shared owner adds a
  regular-file check with its own message. Check order and replacement bytes are unchanged.
- fn-108.4: dossier publication errors read `publish upgrade dossier: <hostfs stage>: ...`, the
  temporary file prefix is `.safefile-*`, and sync, directory and cleanup failures are now
  reported. Payload bytes, mode, path and publication after a failed gate are unchanged.
- fn-108.5: a failure of `SummarizeSemanticProbes(nil)` would now preserve the partial instead of
  returning at once on the seed path. No input reaches that branch.

Assertions and test names:

- `architecture_test.go` allows one more import edge, `upgrade` to `hostfs`. R5 requires that
  edge. No architecture assertion was removed.
- `TestStateStopsAtAttemptBudgetAndRoundTrips` kept its name after its round-trip lines went with
  the removed `Encode`/`Decode`. The name overstates what the test covers. Renaming it would
  change the key of the baseline listing, so it is left for the minimizer owner.

Open follow-ups, recorded for their owners and not changed here:

| Item | Owner |
| --- | --- |
| Close, file-sync, directory-sync and cleanup failures of `hostfs.Replace` are not fault-injected; that needs a seam in `internal/hostfs` | hostfs owner |
| `recordedWorldForMinimization` stays separate from `assessWorld` because its diagnostics differ | Runner owner |
| In the seed strategy an ordinal that completed beside a failed retention is still published and journaled, so the journal can hold ordinal N+1 ahead of the resumed N. The baseline behaves the same and a characterization case pins it | Runner campaign owner |
| `tools/gomad3/ARCHITECTURE.md` does not name `runner/completion.go` and `runner/retention.go` as owners | fn-109 R9 (documentation reconciliation) |
| The Quick command `go build ./...` in the fn-108 task files fails on the runtime overlay packages before and after fn-108 | fn-108 task text |
| `make lint-code-fast` needs a `main` ref in this checkout and cannot analyse the nested module | repository tooling |

One side effect of this task: a `gomad doctor` call made from `tools/gomad3` to read the Runner
build created the empty directory `tools/gomad3/.gomad/artifacts` during the `-json` run of the
`test-host` recipe. Both empty directories were removed with `rmdir` afterwards and
`size-count.sh` reports the same inventory as before.

## Pre-existing findings

fn-108 changes none of the dispositions in [baseline.md](baseline.md): D12 (linux/amd64 replay
divergence, fn-105.12, not reproducible here), the stale linux pack
`modernc-libc-xsys-v047-linux-amd64`, and the host-clock findings of D11 and D21. D14 stays fixed
on darwin/arm64: `functional-signal-chasm` is `qualified` on seeds 11 and 17 with exact choice
replay in the Temporal set above.

## D1 and D2 evidence for fn-105

| Obligation | Delivered by | Evidence |
| --- | --- | --- |
| fn-105.1 (D1, fn-102 R2) | fn-108.5, R6 | [task5-evidence.md](task5-evidence.md), [task5.diff](task5.diff), [task5-review.md](task5-review.md), `task5-canonical-record-inputs-before.txt` and `-after.txt`, `task5-mutation-red.txt` |
| fn-105.2 (D2, fn-102 R3) | fn-108.6, R7 | [task6-evidence.md](task6-evidence.md), [task6.diff](task6.diff), [task6-review.md](task6-review.md), `task6-projections-before.txt` and `-after.txt`, `task6-mutation-red.txt`, `task6-test-dispositions-darwin-arm64.tsv` |
| Both, on the final tree | fn-108.7 | this file, `final-gates/retention-projections-final.txt`, `final-gates/completion-record-inputs-final.txt`, `final-gates/test-dispositions-darwin-arm64.tsv` |

## Files written by this task

| Path | Content |
| --- | --- |
| `final.md` | this record |
| `final-size.txt`, `final-size-files.txt`, `final-size-compare.txt` | outputs of the unmodified size scripts |
| `final-working-tree.diff` | `git diff 6782b55f4 -- tools/gomad3 tools/gomad3sim tools/gomad3integration` |
| `final-comment-lines.txt` | removed and added comment lines per changed Go file |
| `final-api/`, `final-api-diff.txt` | public-surface capture and its empty diff against `api-baseline/` |
| `final-protected-paths.txt` | diff and status of the protected paths and of the three directories |
| `final-gates/results.txt`, `final-gates/*.log` | exit status, times, load and output of every gate |
| `final-gates/test-dispositions-darwin-arm64.tsv` | per-test results of the five `go test` gates |
| `final-gates/*-qualification-rows.tsv`, `*-set.json`, `temporal-qualification-set.summary.json` | qualification outcomes |
| `final-gates/retention-projections-final.txt`, `final-gates/completion-record-inputs-final.txt` | fixed-input projections logged on the final tree |
| `task7-milestones.diff` | this task's hunks in `.plans/GOMAD_MILESTONES.md` against the pre-edit copy |
| `task7-review.md` | review record of this task, written when the review closes |
| `tools/gomad3/.toolchain/fn-110/baseline/` (ignored by git) | `identity.json` and the three full set reports for fn-110 |
