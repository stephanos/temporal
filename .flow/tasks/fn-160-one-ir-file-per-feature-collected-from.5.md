---
satisfies: [R5]
---
# fn-160-one-ir-file-per-feature-collected-from.5 Reusable equivalence check: capture a baseline, compare under a rename map

## Description
Turn the meaning-preserving equivalence check into a reusable gate command, so Batch 5b's single regeneration (and every later batch that keeps meaning but changes identities) is sealed by running a command instead of building a projection again. Each spec has rebuilt it so far: `.flow/tmp/fn-126/` (baseline scripts, still present), fn-132.6's `prove.py`, and fn-155.1's `project.py` with its identity map, step-table dump and negative controls. The fn-155 and fn-132.6 scratch directories are gone; their method is described in `.flow/tasks/fn-155-name-the-standalone-activitys-repeated.1.md` and the fn-155 spec (R6, R7).

**Size:** M
**Files:** a new gate entry point in `model/check` (or a Go command under `tools/umpire/cmd`; pick by where the IR and Case readers already live), a Makefile target, its tests and fixtures, `model/README.md` (how to seal a batch), and `MILESTONES.md`'s Batches text (the seal becomes the command)
**Touches:** [model/check/**, tools/umpire/**, Makefile, model/README.md, MILESTONES.md]

### Approach
- **Capture:** a command that snapshots a baseline from a clean tree (IR files, lowered Cases, manifest, lint acceptances, and every machine's step table, including the interpreter-built tables fn-155 used) into a directory outside the tree, with its commit id.
- **Compare:** a command that projects the baseline and the current tree after normalization (source positions, and the identity mapping), and reports every remaining difference. The rename map is a checked-in or scratch file listing old → new ids: machines, Properties, Queries, realizations, effects/functions, IR files, Case names. Unmapped differences fail. So do map entries that match nothing, and step-table differences.
- **Negative controls:** tests showing that a one-fact model mutation and a changed rule result are rejected, as fn-155.1 did.
- **Reuse:** start from fn-126's scripts and fn-155.1's described projection rather than designing from scratch. Keep it fast enough for a focused check (record runtime and memory).
- **Interface:** `make umpire-capture-baseline OUT=…` and `make umpire-check-equivalence BASE=… MAP=…` (names are a suggestion).

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. The step-table half lands first in 1d (baseline captured before any 1d edit). The IR and Case projection half lands in 5b.

### Quick commands
```sh
make umpire-capture-baseline OUT=/tmp/base
make umpire-check-equivalence BASE=/tmp/base MAP=/dev/null   # identical tree: passes
```

## Acceptance
- [ ] One command captures a baseline (IR, Cases, manifest, lint acceptances, step tables) and one compares the current tree against it under a rename map, failing on unmapped differences, unused map entries and step-table changes.
- [ ] Negative controls (a model mutation, a changed rule result) are rejected in tests; an unchanged tree passes with an empty map.
- [ ] The README and MILESTONES.md describe sealing a batch with these commands.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
