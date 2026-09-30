---
satisfies: [R13, R21]
---
# fn-88-veil-concrete-checker-as-the-umpire.7 Model docs, rollback drill, and defer-branch closeout

## Description
Update the model documentation for the landed backend, run the rollback drill against the closed permitted diff, record the result in the delivery order and plan index, and in either defer mode write the absence test and record the defer instead. The GOV-02 amendment drafts are task .8, not here.

**Size:** M
**Files:** `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `model/README.md`, `model/AUTHORING.md`, `.plans/UMPIRE4_ORDER.md`, `.plans/index.json`, `experiments/umpire-dsl/VEIL_RESULTS.md` (link the landed state), `model/Umpire/Search/Tests/Absence.lean` (defer mode only)
**Touches:** [model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/README.md, model/AUTHORING.md, .plans/UMPIRE4_ORDER.md, .plans/index.json, experiments/umpire-dsl/VEIL_RESULTS.md, model/Umpire/Search/Tests/Absence.lean]

### Approach
- Model docs: `model/ARCHITECTURE.md` dependency diagram and imports prose (line 26 and 58 today) gain Product, Selection, the adapter, and the third external requirement; `model/Umpire/ARCHITECTURE.md` module table rows and the lifecycle section gain selection and the replay gate; `model/README.md` build section records the R1 cold-build figure and the manifest pin check; `model/AUTHORING.md` §8 explains `search` as paths on `reference` and states on `veil`.
- Rollback drill (R21) on a scratch branch: remove the `require` and manifest entry, delete `Umpire/Search/Backend/Veil.lean` and `Tests/BackendVeil.lean`, reduce `Umpire/Search/Selection.lean` to always `reference`, delete the differential test's veil arm and the veil pins, remove the lint rule and its controlled violation, and let the R18 goldens flip back. Confirm the build and `make umpire-check-regression` pass and that the `git diff --stat` touches nothing outside that set; paste the stat into task evidence; do not merge.
- Order and index: record the adopt result and the R1 receipt in `.plans/UMPIRE4_ORDER.md`; keep the `.plans/index.json` flowSpecs entry in sync with Flow (`make umpire-check-plan-index`).
- Defer mode: write `model/Umpire/Search/Tests/Absence.lean` proving no `Veil.*` import, no Lake requirement, and no Veil Make target exist; record the defer and the `FiniteTable` to TLA+ exporter follow-up in `UMPIRE4_ORDER.md`; confirm every other task closed itself as not applicable with the receipt identity.

### Investigation targets
**Required:**
- `model/ARCHITECTURE.md:20-60`, `model/Umpire/ARCHITECTURE.md:20-45, 97-165`, `model/README.md:471-490`, `model/AUTHORING.md:612-686`
- `tools/planindex/check.go` — what the index check validates

**Optional:**
- `.plans/lean/UMPIRE4_DIRECTION.md` sections 1 and 6 — reasoning to cite

### Key context
- The CLAUDE.md cold-build note ("~12 minutes") is loaded by the harness but the file is deleted in the working tree; update it only if restored.
- Docs describe the landed state; do not restate spec rationale.

### Carried from fn-88.5 (2026-09-27)
- The adapter is pure (`bfsStep` bounded by `Limits.search`) and deduplicates by exact product state. Correct every GOV-02 draft fn-88.8 wrote (UMPIRE4_SPEC.md, SPEC_MODEL_ARCH, DSL, SPEC_COMPS, COMPONENTS) and AUTHORING.md: drop the `IO`-during-elaboration and 64-bit-hash trust statements; a `veil` absence answer rests on the adapter theorems and the differential test.
- R20: measure one cold CI run once .6 makes CI build Veil (push to the fork and read the run, or record why not); add a `.lake` cache to `.github/workflows/umpire.yml` if it does not fit the 30/40-minute job timeouts. Pin Node for CI if the runner's default is not enough for Veil's widget build.
- Rerun `LEAN_NUM_THREADS=1 make lint-model` on a quiet host; its last `lake lint` step was inconclusive at .5.

### Carried from fn-88.10 (2026-09-28)
- `model/ARCHITECTURE.md` lines ~208 and ~212 still say Replay candidates are named by the "Plan checksum"; they are now named by `witnessKey` (the checksum with `explored` cleared).
- R20: the Umpire workflow runs on pushes to `stephanos/umpire`. The conductor pushed 089f9ef966 to the fork branch `stephanos/umpire`, which started run 36377425097 (`gh run view 36377425097 -R stephanos/temporal --json jobs`); read each Lean job's duration against its 30/40-minute timeout from that run (it is the first with Veil in the build and a cold `.lake`), and record it. If a job exceeded or came near its timeout, add the `.lake` cache step and Node pin to `.github/workflows/umpire.yml` (update `tools/umpire/regression/ci_workflow_test.go`'s pinned steps); the conductor will push again to measure.

## Acceptance
- [ ] Model docs updated as listed; `make umpire-check-inventory` and `make umpire-check-plan-index` pass
- [ ] `UMPIRE4_ORDER.md` records the adopt (or defer) result and the R1 receipt
- [ ] Rollback drill diff stat in task evidence showing only the permitted set; build and regression pass on the scratch branch
- [ ] Defer mode: absence test present and passing; every other task closed as not applicable citing the receipt

## Done summary
The model docs and the GOV-02 drafts now describe the `veil` backend as built. The rollback drill passed inside the permitted set. `UMPIRE4_ORDER.md` records fn-88 as done and awaiting its completion review. CI run 36377425097 failed on a pre-existing missing input, not on Veil, and a CI step now generates it.

- **Model docs (4f188c2455):**
  - `model/ARCHITECTURE.md`: the diagram and import prose gain Search.Product, the adapter, Selection, Veil as the third Lake requirement, `umpire-check-veil-pin`, and the isolation rule. Replay Cases are now named by `witnessKey`, not the Plan checksum (the fn-88.10 carry).
  - `model/Umpire/ARCHITECTURE.md`: module rows for the three modules; a lifecycle paragraph on selection and the replay gate.
  - `model/README.md`: the Veil pin check, the Node prerequisite, and the cold-build figures.
  - `model/AUTHORING.md` §8: `search` counts paths on `reference` and product states on `veil`, and the trust basis of each backend.
  - `VEIL_RESULTS.md`: a section on the landed state.
- **Trust statements (fn-88.5 carry):** the drafts in UMPIRE4_SPEC, SPEC_MODEL_ARCH, DSL, SPEC_COMPS and COMPONENTS no longer say the checker runs in `IO` during elaboration or rests on a 64-bit hash. A `veil` absence answer now rests on the pure `bfsStep` driver, exact product-state dedup, the adapter theorems, and the differential test. The fn-88 status cells there say landed.
- **R21 rollback drill:** run on a scratch clone and not merged. It touched 23 files, all in the permitted set:
  - the `require` and its manifest entries;
  - the adapter, and `BackendVeil` with its registration;
  - Selection, reduced to `reference` only;
  - the veil pins: Replay selection, the R9 Caller and Pair counts, the R15 and R9 fixtures, and the R19 fallback checks;
  - the veil-pin Make target and its CI pin;
  - the lint rule and its controlled violation;
  - one golden flipped back: `CallerRetryPlanningReceipt.json` (veil to reference).

  The build and `make umpire-check-regression` were green. One deviation: the differential's `.veil` call sites stayed. They now run `reference` through the reduced Selection, and their pinned lines were re-pinned to "reference default". The diff stat is in the evidence.
- **R20:** not closed. Run 36377425097 failed in both jobs on the missing git-ignored `proto/api.binpb`, which the build has read since 990f534a0a; the 2026-09-23 runs failed on it too. Before stopping, the portability job built Veil's closure, npm widget included, in 84 s with the runner's own Node. c3dc9eb68c adds `make proto/api.binpb` to both jobs and `ci_workflow_test.go` pins it. Whether the whole cold build fits the timeouts is unmeasured: the conductor must push again. No `.lake` cache was added and Node is not pinned.
- **Order (c646beb00c):** fn-88 is recorded as all tasks done and awaiting its completion review, with the R1 and R22 receipts, the drill result, the CI finding, and the lint baseline. `index.json` needed no change, and plan-index is valid.
- `LEAN_NUM_THREADS=1 make lint-model` passed on a quiet host with Veil required (the fn-88.5 carry).
- Defer mode: not applicable (R22 adopt), so R13's absence test was not written.

Follow-ups:
- Push to measure R20. If a job comes near its timeout, add the `.lake` cache.
- `.plans/lean/UMPIRE4_DIRECTION.md:137` still says 4.33.1 (the fn-88.12 carry, outside Touches).
- getproto writes descriptor sets in nondeterministic order.

stage: impl-review - ran [2026-09-28..2026-09-28] (codex fan-out, three draws SHIP, zero findings; forced full review, no triage)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4f188c24552e251726e275cd34d98ec803f04426, c3dc9eb68c1c8dfd2250e4629cf96f081f3ab0cb, c646beb00ce7bd5fa026391da5d7e2d0e13aa782
- Tests: baseline: green via receipt (GATE_SKIPPED:goldens:green-receipt d157f623, GATE_SKIPPED:regression:green-receipt d157f623 - baseline reused from prior post-gate pass), LEAN_NUM_THREADS=1 make lint-model (baseline, quiet host, rc=0, 4949 s; post-edit diff touches no Lean source, so not rerun), CC=/usr/bin/clang go test -count=1 -tags test_dep ./tools/umpire/regression/ -run TestUmpireCIWorkflow (green; red first with the workflow step reverted), make umpire-check-plan-index (valid), make umpire-check-inventory (rc=0), make umpire-check-regression (rc=0, 1968 s, 45 live identities, includes umpire-check-goldens), R21 rollback drill on a scratch clone of cc14df4460 (branch fn-88.7-rollback-drill, deleted after): make umpire-build-model (614 jobs, green), make umpire-check-goldens flips only CallerRetryPlanningReceipt.json (veil->reference), make umpire-check-regression (rc=0, 1555 s, 45 live identities; two earlier attempts failed on no space left on device during the live build and are INCONCLUSIVE, not red), R21 drill diff stat: Makefile                                           |  41 +--;model/ModelLint/ImportGraph.lean                   |  67 +----;model/ModelLint/ImportGraphTests.lean              |  47 +---;.../Fixtures/CallerRetryPlanningReceipt.json       |   2 +-;model/Temporal/Feature/Nexus/Caller/Tests.lean     |  20 --;model/Temporal/Feature/Nexus/Pair/Tests.lean       |  15 --;model/Temporal/Tool/ExplorationBridgeTests.lean    |  37 ---;model/Temporal/Tool/ReplayBridgeTests.lean         |  22 --;model/TemporalModelTests/SearchDifferential.lean   |  70 ++---;.../SearchDifferential/CallerCampaign1.lean        |   4 +-;.../SearchDifferential/CallerCampaign2.lean        |   4 +-;.../SearchDifferential/CallerCampaign3.lean        |   4 +-;.../SearchDifferential/CallerCampaign4.lean        |   2 +-;model/Umpire/Exploration/Tests/Classed.lean        |  37 ---;model/Umpire/Search/Backend/Veil.lean              | 295 ---------------------;model/Umpire/Search/Selection.lean                 |  90 +------;model/Umpire/Search/Tests.lean                     |   1 -;model/Umpire/Search/Tests/BackendVeil.lean         | 185 -------------;model/Umpire/Search/Tests/Differential.lean        | 108 ++------;model/Umpire/Search/Tests/Replay.lean              |  84 ++----;model/lake-manifest.json                           |  82 +-----;model/lakefile.lean                                |   7 -;tools/umpire/regression/ci_workflow_test.go        |   4 -;23 files changed, 105 insertions(+), 1123 deletions(-), R20: CI run 36377425097 failed in both jobs on missing proto/api.binpb (pre-existing since 990f534a0a, 2026-09-19); portability built the Veil closure incl. npm widget in 84 s; fix c3dc9eb68c; whole cold-build fit unmeasured pending the next push
- PRs: