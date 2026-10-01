---
satisfies: [R2, R3, R18, R19, R20]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.21 Run final qualification and retain the finding completion matrix

## Description
Stage 6, R18-R20, plus the evidence links for R2/R3. Run the complete gates once against the finished tree, compare with the baseline, and write the matrix that maps every finding to its requirement and evidence. This task implements nothing; a gap it finds goes back to the owning task.

**Size:** M
**Files:** `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/completion-matrix.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/qualification-evidence.md`, retained command output under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/evidence/`.
**Touches:** [.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/**, .plans/GOMAD_MILESTONES.md]

### Approach
- Baseline: record the pre-fn-109 revision (planning anchored at `d4d800fb47`; use the actual first-task base), toolchain build key, and the current qualification dispositions (`tools/gomad3integration/qualification/*.json`, the core set report) before judging regressions.
- R2/R3: no implementation here. Link the evidence of `fn-108-gomad-reduce-code-size-without-removing.5` (R6), `.6` (R7) and `.7` (equivalence and gate evidence). If either is not done and verified, R2/R3 stay open and this task reports the spec as incomplete.
- Completion matrix: one row per finding F1-F11 and S1-S5 with R-ID, owning task, implementation evidence (files and symbols) and verification evidence (command and result). D1/D2 map to fn-108 once; D3/D4/D5 map to tasks 6, 19 and 20 once, with fn-105.3/.4/.5 closed by reference. No finding may be closed by deferral, and no obligation may have two owners.
- R18 preservation audit: exported-surface diff of `runner`, `artifact`, `target`, `deterministicio`, `record`, `choice`, `world`, `qualification` against the baseline (only the changes in `go-interface-changes.md` may appear); CLI command and flag inventory against `CLI.md`; comment preservation spot-check on moved code (`git diff` for deleted comment lines); fixed-identity canonical projections; no new generic host-I/O grant in the boundary manifest or compatibility packs.
- R19 gates on darwin/arm64, serialized (they share `.toolchain` and qualification output directories): generator validation, the full Gomad test tiers, native/default integration, functional smoke, the core qualification set, and the suites affected by overlay changes. Failures attributable to D12/D14 are recorded with their evidence under those owners, not waived.
- Bounded control cases: run a seed campaign at 10 and at 100 jobs with fixed parallelism and compare policy-state size and payload copies (heap profile or explicit counters) to show no selection-sized state or extra full-payload copy was introduced by the extractions. Record the numbers; an unmeasured claim is not evidence.
- linux/amd64: every gate that needs that host is listed as incomplete with the exact command to run there (CI workflow `.github/workflows/gomad3.yml`). R19 stays partially unmet until that evidence exists; say so plainly.

### Investigation targets
**Required:**
- the spec's "Finding coverage" table and "Acceptance Criteria" R18-R20
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`
- `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml` (gate definitions per platform)
- `tools/gomad3/Makefile` and the root `Makefile:165-235`
- `.plans/GOMAD_MILESTONES.md` sections "Open findings" and "Constraints"

### Quick commands
```bash
cd tools/gomad3
make validate
make test
cd ../..
make gomad3-runner
make gomad3-integration-test
make gomad3-smoke-qualification
make -C tools/gomad3 compatibility-pack-qualification core-qualification-set
tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
make lint-code-fast
flowctl validate --spec fn-109-gomad-deepen-modules-and-tool-interfaces
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] `completion-matrix.md` maps every F1-F11 and S1-S5 to its R-ID, owning task, implementation evidence and verification evidence, with fn-108 R6/R7 linked for R2/R3 and D1-D5 each mapped exactly once.
- [ ] The preservation audit shows no feature removal, no public Go change beyond `go-interface-changes.md`, unchanged CLI inventory, preserved comments and unchanged fixed-identity canonical bytes.
- [ ] Baseline revision and inputs, commands and per-platform results are retained; darwin/arm64 gates (validation, full Gomad tiers, integration, smoke, core qualification, affected suites) pass against unchanged dispositions or each failure is attributed with evidence.
- [ ] 10-job and 100-job control cases are measured and show bounded policy state and no added full-payload copy.
- [ ] linux/amd64 gates, unfinished fn-108 evidence and any D12/D14-owned failure are listed as incomplete acceptance with the command to run; nothing is reported as passing that was not run.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
