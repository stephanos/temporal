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
- `.plans/UMPIRE4_DIRECTION.md` sections 1 and 6 — reasoning to cite

### Key context
- The CLAUDE.md cold-build note ("~12 minutes") is loaded by the harness but the file is deleted in the working tree; update it only if restored.
- Docs describe the landed state; do not restate spec rationale.

## Acceptance
- [ ] Model docs updated as listed; `make umpire-check-inventory` and `make umpire-check-plan-index` pass
- [ ] `UMPIRE4_ORDER.md` records the adopt (or defer) result and the R1 receipt
- [ ] Rollback drill diff stat in task evidence showing only the permitted set; build and regression pass on the scratch branch
- [ ] Defer mode: absence test present and passing; every other task closed as not applicable citing the receipt

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
