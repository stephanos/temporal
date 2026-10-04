Stage 6, R18-R20, plus the evidence links for R2/R3. Run the complete gates once against the finished tree, compare with the baseline, and write the matrix that maps every finding to its requirement and evidence. This task implements nothing; a gap it finds goes back to the owning task.

**Size:** M
**Files:** `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/completion-matrix.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/qualification-evidence.md`, retained command output under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/evidence/`.
**Touches:** [.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/**, MILESTONES.md]

### Approach
- Baseline: record the pre-fn-109 revision (planning anchored at `d4d800fb47`; use the actual first-task base), toolchain build key, and the current qualification dispositions (`tools/gomad3integration/qualification/*.json`, the core set report) before judging regressions.
- R2/R3: no implementation here. Link the evidence of `fn-108-gomad-reduce-code-size-without-removing.5` (R6), `.6` (R7) and `.7` (equivalence and gate evidence). If either is not done and verified, R2/R3 stay open and this task reports the spec as incomplete.
- Completion matrix: one row per finding F1-F11 and S1-S5 with R-ID, owning task, implementation evidence (files and symbols) and verification evidence (command and result). D1/D2 map to fn-108 once; D3/D4/D5 map to tasks 6, 19 and 20 once, with fn-105.3/.4/.5 closed by reference. No finding may be closed by deferral, and no obligation may have two owners.
- R18 preservation audit: exported-surface diff of `runner`, `artifact`, `target`, `deterministicio`, `record`, `choice`, `world`, `qualification` against the baseline (only the changes in `go-interface-changes.md` may appear); CLI command and flag inventory against `CLI.md`; comment preservation spot-check on moved code (`git diff` for deleted comment lines); fixed-identity canonical projections; no new generic host-I/O grant in the boundary manifest or compatibility packs.
- R19 gates on each qualified native platform (`darwin/arm64` and `linux/amd64`), serialized within a platform (they share `.toolchain` and qualification output directories): generator validation, the full Gomad test tiers, native/default integration, functional smoke, the core qualification set, and the suites affected by overlay changes. Failures attributable to D12/D14 are recorded with their evidence under those owners, not waived.
- Bounded control cases: run a seed campaign at 10 and at 100 jobs with fixed parallelism and compare policy-state size and payload copies (heap profile or explicit counters) to show no selection-sized state or extra full-payload copy was introduced by the extractions. Record the numbers; an unmeasured claim is not evidence.
- Measurement preparation is retained in `task-21/bounded-campaign-measurement-source-scout.md`. Use an isolated scratch-only same-package verification harness, retained under this task's artifacts, rather than production hooks or shipped source changes. Reconstruct the actual first-task dirty baseline before comparison; run baseline/current subcases separately after the final source freezes. Distinguish bounded policy/live payload state from selection-derived journal capacities and retained evidence. Profiles need site-specific attribution and normalized per-execution bytes; constructor alias checks alone cannot exclude transient copies. The existing 10/100 active-width/capacity test is a behavioral control, not the missing memory/copy measurements. Any production gap returns to its owning task.
- The actual first-task dirty nested-module baseline has been reconstructed under `task-21/baseline-reconstruction/`. Read `reconstruction.md` and `conductor-verification.md`, then freshly verify its input/source manifests before measurement. The complete 670-file reconstruction independently matches the nested-module Git blobs and executable modes at `38957053f1ce342a8797af1803f5f8f6bb53fcad`; that later equivalent tree corroborates the original baseline without replacing it or identifying the whole historical repository. No measurements or qualification were performed by the reconstruction.
- The environment-bound developmental baseline campaigns are retained in `task-21/bound-baseline-measurement/measurement.md`, with complete pre/post source inventories, explicit pre-launch environment bindings, per-case commands and allocation-site attribution. Read `task-21/conductor-bound-baseline-verification.md` and the independent checkpoint source review before the matched current-tree run. Preserve the historical manifests and their disclosed task-description-only metadata drift. These four baseline campaigns do not complete the current comparison, task admission, R19 or either native platform's gates.
- For either unavailable native platform, list every required gate as incomplete with its exact command (CI workflow `.github/workflows/gomad3.yml`). R19 remains unmet until both platforms have source-bound evidence; the current unsupported host supplies neither. A historical pre-integration result or developmental run cannot close either platform.

### Investigation targets
**Required:**
- the spec's "Finding coverage" table and "Acceptance Criteria" R18-R20
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/documentation-evidence.md`
- `.github/workflows/gomad3.yml`, `.github/workflows/gomad3-smoke.yml` (gate definitions per platform)
- `tools/gomad3/Makefile` and the root `Makefile:165-235`
- `MILESTONES.md` sections "Open findings" and "Constraints"

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
- Follow `MILESTONES.md`: commit this task's verified progress separately, including its implementation, tests, documentation and Flow records. Root is the sole committer; preserve unrelated changes. Record unavailable native gates as incomplete and keep acceptance open. This supersedes older user-only commit instructions. Push, stash, worktree creation and history rewrites require separate authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Recheck the actual execution host before gates. This development session is `linux/arm64`, with the patched toolchain absent; neither native `darwin/arm64` nor native `linux/amd64` qualification is available here. Cross-platform source/type/vet checks and stock-host tests are developmental evidence only. Keep each required native gate incomplete until a source-bound result exists on its qualified platform.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
