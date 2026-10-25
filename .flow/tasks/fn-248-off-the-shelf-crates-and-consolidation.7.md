---
satisfies: [R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.7 Join crate-consolidation lanes and validate current snapshots and guests once

## Description

**Size:** M
**Files:** shared Cargo/lock and conflicting source edits from .1–.6; reviewed current snapshots, necessary guest/generated assets; Makefile and current Model documentation.
**Touches:** [model/**, Makefile, .plans/UMPIRE_MODULES.md]

Dependencies are all six existing fn248 source tasks. This is integration and the original R5 validation, not a seventh feature or an additional full gate. Join the six isolated lanes, preserving shared command/helper/codegen registrations and independent semantic oracles. Resolve Cargo/features and generated WKT paths centrally; verify R18/new package provenance and guest closure boundaries. Retain all original R1–R4 positive/negative obligations.

## Approach

- Merge source lanes and rerun only affected focused controls after conflict resolution. Review text snapshot/UI/expansion and necessary generated asset diffs on joined inputs, reject unexplained semantic changes, then re-accept through the new native insta workflow. Preserve meaningful historical original comparisons and mutation controls.
- Regenerate current guests/pins only where inputs changed, once on joined inputs; record reason and old/new identities/hashes, prove applicable current-input reproduction and committed Temporal guest freshness, and retain all wasm/native/Go ABI and typed-refusal controls. No comparison with historical bytes or old-guest adapter is needed. Keep host-only snapshot tooling out of guest closures.
- Current AGENTS/README define model-validate as the canonical gate. Run one `UMPIRE_APALACHE=require make model-validate`: its single complete TLC/Apalache-required model-check includes clippy/docs/tests/wasm/Go and the existing benchmark follows once. This consumes the existing project pipeline; do not also run a default model-check, a separate model-time or cargo-insta workspace suite. Focused checks remain allowed when a change/failure justifies them.
- Record actual mode/stages/results/current tools and remaining platform limits; do not invent pass receipts. The ownership/pending gate runs inside the existing test pass and covers intentional cross-crate/live/Go readers without claiming those readers executed merely because they are registered. Root owns MILESTONES and lifecycle/review scheduling.

## Quick commands

```bash
UMPIRE_APALACHE=require make model-validate
```

## Acceptance

- [ ] All six lanes are joined with every R1–R4 control retained, shared interfaces/codegen/Cargo features reconciled and affected focused checks passing.
- [ ] Reviewed current text snapshots/UI/expansion diffs are re-accepted once; orphaned/pending/unowned controls, skipped-backend completeness and explicit cross-crate/Go/live ownership controls pass without a second workspace run or new installed CLI prerequisite.
- [ ] Necessary current guest publication records cause/new pins and passes applicable reproducibility, freshness, wasm/native/Go and typed replay/refusal controls without historical byte compatibility or semantic weakening.
- [ ] One canonical required pipeline and existing benchmark pass with actual command, stage/mode/timing/results evidence; all existing gate stages remain and soft timing warnings cannot hide failures.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
