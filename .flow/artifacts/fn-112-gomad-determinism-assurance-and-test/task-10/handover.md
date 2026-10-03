# fn-112 task 10 handover

Host: linux/arm64 (developmental). None of the results below is darwin/arm64 or linux/amd64 evidence.

## Delivered

- `gomadtool soak` (`tools/gomad3/qualification/soak`, `cmd/gomadtool/soak.go`) and `set.SoakQualifyArguments`.
  It runs `gomad qualify --diagnostics` batches without success replay, under two busy host threads. It compares
  each clean batch's evidence digest with its cohort baseline. A cohort is one workload, seed, platform, and
  execution identity, where the identity hashes the runner and toolchain builds, the target, the I/O profile,
  environment, limits, mounts, and choice and diagnostic profiles. The baseline and the cumulative counts live in a
  ledger carried between runs. Overflow, target failure, and infrastructure failure are reported apart from
  divergence. A divergence retains both traces, both evidence records, and `differ.json`/`differ.txt`.
- `tools/gomad3integration/qualification/soak.json`: the four smoke suites plus the guarded-mode
  `frontend-system-info`, seeds 11 and 17, `repeat` 32, minimum 2 and maximum 4 batches (N 64 to 128), a 55m `qualify_timeout`,
  a 120m budget (180-minute job timeout; capped for the fork's runner quota), and 2 load workers. Its `sizing` field records how N is chosen and why. linux/amd64 is listed under
  `informational_platforms` (fn-105 D12).
- Root `make gomad3-soak`; `gomad3.yml` jobs `determinism-soak-darwin` (strict) and `determinism-soak-linux`
  (informational). Both are schedule/dispatch only, with a matrix of workload x seed. Each job restores its ledger
  from the latest retained schedule/dispatch run via `gh run download` and uploads the ledger and the
  report/divergence evidence (retained 90 days).
- Docs: tools/gomad3 README (Contract and Development), SPEC (new `RUNTIME.CHOICES.DIAGNOSTICS`, `RUNTIME.STREAMS`,
  `RUNTIME.TRUST.EXCLUSIONS`, `QUALIFICATION.SOAK`, gomadtool rows `DIAGNOSTIC.DIFF` and `SOAK`, and the closure-mode
  sentence in `TARGET.CAPABILITY`), CLI, ARCHITECTURE, TUTORIAL, AGENTS.md, MILESTONES, and the
  gomad3integration README.

## Local evidence

- The patched toolchain refuses this host (`toolchain-build-refusal.txt`), so real `gomad qualify` cannot run here.
  The soak was exercised end to end against `stand-in-gomad-qualify.go.txt`, which writes real qualification
  reports and real diagnostic traces. It used `developmental-soak.json`, which resolves the real `core.json`
  concurrency workload's qualify arguments.
  - Run 1 and run 2: clean, exit 0. Cumulative 24 repetitions per cohort across 2 ledger runs.
  - Run 3: injected within-batch divergence, exit 1. It retained the traces, and the differ reported
    `first-divergent-ordinal=0 fields=runtime_cheap_rand_draws`.
  - No Gomad determinism bound was measured. The native bound comes from the first retained CI run per platform.
- The developmental run found a real defect: relative work, ledger, and output paths escaped the batch working
  directory. It is fixed, and `TestRunGivesBatchesAbsolutePaths` covers it.
- Review round 1 (NEEDS_WORK) fixes: the bound now counts only clean-batch repetitions (`clean_repetitions`); a
  failed baseline retention is infrastructure and leaves no baseline; `runBatch` is a `soakRun` method; the ledger
  is saved after every batch; soak uploads set `overwrite: true` for re-run attempts.
- Cost cap (conductor follow-up): at most 4 batches within a 120-minute budget and a 180-minute job timeout.
  Matrix jobs keep their own toolchain builds: no gomad3 workflow shares a built toolchain today, and sharing one
  would need a tarred `.toolchain` (symlinks, modes, build-key cache validation at the same absolute path) that has
  no existing pattern to follow.
- `go test -count=1 -tags test_dep ./qualification/soak`: 16 tests pass. A mutation check confirmed coverage: with
  the cross-batch divergence assignment removed, `TestCohortBatchSequenceAAThenBBIsADivergence` and two run tests
  fail.
- With stock Go 1.27.1, `go test -tags test_dep . ./cmd/gomadtool ./qualification/...` passes, except
  `qualification/analysis`. That package fails only because "deterministic I/O requires one of darwin/arm64,
  linux/amd64; host is linux/arm64"; this task did not touch it. `TestPackageArchitecture` and the vocabulary test
  pass.
- `make -C tools/gomad3 validate`: exit 0 (35 s). actionlint v1.7.7 on gomad3.yml: clean, with shellcheck
  unavailable. go vet and gofmt: clean. Scoped golangci-lint: the only findings left in the new files are 4
  unchecked `fmt.Fprint*` calls in `cmd/gomadtool/soak.go`, matching the 131 existing instances in that package.
- fn-111 guide audit (`verify-guides.py` with locally built `gomad`/`gomadtool`): no new errors relative to the
  pre-edit baseline. One error is fixed: `soak` was dispatched but not indexed. The 31 remaining errors predate
  this task; they come from the stale audit script and the clock probe, which cannot build here. SPEC identifiers
  were added under VERIFICATION.CHANGE and none were removed.
- Not runnable here: `make -C tools/gomad3 test-host` and the other toolchain tiers, because the toolchain refuses
  linux/arm64.

## Remaining (native/CI only)

- One completed scheduled or dispatched run of `gomad3.yml` per platform, with its `determinism-soak-*` reports and
  ledgers retained. Its per-cohort counts are the first measured bound. After it, re-check the round sizing from
  `execution_wall_nanos`.
- `make -C tools/gomad3 test-host` on darwin/arm64 and linux/amd64 for the new packages.
