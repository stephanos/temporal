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
The equivalence check is now one gate command for each step (`model/check/Equivalence.scala`,
`Gate.captureBaseline` / `Gate.checkEquivalence`):

- `make umpire-capture-baseline OUT=<dir> [DUMPER=<path>]`
- `make umpire-check-equivalence BASE=<dir> MAP=<file> [STEPS_ONLY=1] [DUMPER=<path>]`

The check fails on every difference the rename map does not explain, and on every map entry that
explains nothing (a rename that matches nothing, or a removal that removes nothing).

- **Step tables (Batch 1d).** `model/framework/StepTables.scala` writes a table for every machine
  and composition the IR files reach. It discovers them through `IrFile.discover` and the restored
  `IrFile.reached()`, which shares one traversal with `subjects()`, so `Built` machines and
  anonymous compositions get tables too. Values use the table pins' spelling: `StepTables.spell`,
  which `MachineTablePins` now calls too. Each machine row records its start, binding order, each
  state's end and refinement image, and one row per state and class, with ordered steps (outcome,
  state, facts, because), disabled pairs and holes included.
  - A composition's table holds its members, its derivation, its syncs and its ends. A small seam
    in `Syncs` records sync statements at run time.
  - Tables are gzipped and deduplicated by digest.
  - The tables depend only on the framework and `model/temporal`, so they run while the IR is stale
    and irgen does not build.
- **Projection (Batch 5b).** Covers every IR declaration, keyed by section and identity and
  qualified by its machine's family. File boundaries therefore do not matter, so a merge of the IR
  files is equivalent under file renames: a simulated fn-160.2 merge of the 7 files into 3 shows 0
  differences under a 4-entry map and 495 without it. It also covers every accepted lint finding
  (as a multiset), every Case by file name, and every manifest entry by Query. Source positions,
  `*Fingerprint` fields and IR `source` root lists are dropped.
- **Interpreter tables (Batch 5b).** Instead of a new projection, the table pins'
  `replay.py --baseline --map` compares the pinned dumper's streams with a captured baseline under
  the same map. Records are canonical, positions and the header's IR file name are dropped, and
  subjects are keyed by family.
- **Rename map.** `old new` renames a whole word everywhere, longest first and all at once.
  Dotted and dashed ids count as words, so an IR file rename also renames its Cases. `key -`
  removes a table and `key/actor.action -` removes one action's rows. With `STEPS_ONLY=1`, an
  entry that only the baseline's IR or Cases mention is deferred to 5b rather than failed.
- **Baseline.** The pre-1d baseline is at
  `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn160/baseline-ae3f36ee51/`. Its
  README records the commit, the command, the sizes, and runtime and memory.
- **Reused:** `IrFile.discover`, MachineTablePins' value spelling, the mutants.py control set
  (mirrored in the tests), replay.py and pins.json (capture replays the pins), and the fn-155
  dumper.
- **Dropped:** the earlier draft's classpath scan, and its rename application per table only.

Proposed MILESTONES.md wording ("Batches 1d and 5b", for the root to apply): "**How 1d is verified
without IR:** before the first 1d edit, `make umpire-capture-baseline` captured the baseline at
`.flow/tmp/fn160/baseline-ae3f36ee51/`. Each 1d change seals with `make umpire-check-equivalence
BASE=<baseline> MAP=<batch map> STEPS_ONLY=1`, plus the Scala build, framework tests and model
tests. Batch 5b's IR equivalence check is `make umpire-check-equivalence BASE=<baseline> MAP=<batch
map> DUMPER=<fn-155 dump>`, which adds the IR, lint, Case and manifest projection and the
interpreter tables."

## Evidence
- Commits: ea5264bc23 (feat: capture and check), 65fceb4993 (fix: native streams keyed by family),
  c2511048e3 (fix: projected IR declarations qualified by family), plus the lint fix (see the
  handoff)
- Tests:
  - `scala-cli test model/check`: 89 passed, of which EquivalenceSuite has 11. They cover a
    one-fact mutation; the mutants.py set (row, result, result-order, fact-order, binding-order,
    explanation, pair, missing pair, missing and extra machine); renames; unused entries; removals;
    projection merges; and family qualification.
  - `scala-cli test model/project.scala model/framework model/temporal`: 160 passed, including
    StepTablesTest (4) and MachineTablePins.
  - `python3 model/testdata/table-pins/test_replay.py`: 8 passed.
- Real-tree controls against the baseline:
  - `make umpire-check-equivalence BASE=<baseline> MAP=/dev/null DUMPER=<dump>`: exit 0 in 422 s
    (tables 60 s, projection 0 s, interpreter 359 s).
  - The same with `STEPS_ONLY=1`: exit 0 in 62 s.
  - One-fact mutation (`nexusProduct.terminate` records `nexusOperationCanceled`): rejected.
  - Changed rule result (`handlerError(false)` → `effects.cancel`): rejected.
  - Fact rename across 7 files, with the map `nexusOperationTimedOut nexusOperationExpired`: passes.
    Without the map it is rejected, and with an extra `noSuchWord other` entry it is rejected as
    unused.
  - Logs: `.worktrees/w-fn1605/.flow/tmp/fn1605/control-*.log`.
- Capture: 128 s, peak RSS about 2.2 GB, step tables at 207 MiB heap.
  - 59 tables (42 distinct, 13,077,242 lines), 9.9 MB.
  - Native replay against the pins: complete, 13,925,786 records.
- Lint:
  - `lint-model-check` passes after the lint fix.
  - `lint-model-models` reports none of this task's files. It fails on inherited findings in other
    files.
  - fmt --check fails on the inherited `irgen/testdata/lifts/RolesCompatibility.scala`.
  - `lint-model-conventions` fails because irgen does not compile (`Realizations.scala` Duration,
    inherited).
- PRs:
