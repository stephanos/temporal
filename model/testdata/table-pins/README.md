# Complete interpreter table pins

`pins.json` seals the seven activity/Nexus artifacts from pre-change commit
`41065b6070490b71e6ab73a1ddb06640b8b45bb5`. The inputs were independently
materialized with `git show` and checked against fn156 task 1's sealed managed-tree
hashes. The inherited fn155 dumper and validator were checked against their
sealed proof manifest. No candidate output was used to establish these pins.

The 61 machine/composition table occurrences contain 13,925,786 records. Each
uncompressed JSONL hash includes the full native declaration and catalogs,
starts/ends, binding order, classes, monitor holes, explicit disabled pairs,
enabled rows, ordered alternatives, outcomes, facts, and explanations. Compositions
use the inherited interpreter's unchanged `check.DefaultScope`; their complete
reachable catalogs are pinned too. Whole streams stay in scratch storage.

`MachineTablePins` independently discovers compiled exports, checks their complete
declaration closure against lifted IR and this manifest, and compares native
state/outcome/fact/start/end/class catalogs and binding order to Scala values.

Run the comparator's focused regression tests with:

```sh
python3 model/testdata/table-pins/test_replay.py
```

Complete replay requires the existing, sealed fn155 native dumper (no new Go
runtime API). Supply its location explicitly; an absent or changed binary is
inconclusive, never a pass. Choose a new output directory on every observation:

```sh
python3 model/testdata/table-pins/replay.py --ir model/ir \
  --dumper /path/to/fn155/tables/dump --out /fresh/scratch/native \
  --receipt /scratch/replay-receipt.json
python3 model/testdata/table-pins/mutants.py --native /scratch/baseline-native \
  --out /fresh/scratch/mutants
```

The replay takes the shared `/tmp/umpire-heavy-gates.lock`. It fails on missing or
extra artifacts/subjects, incomplete dumps, catalog/row/result/order changes,
changed interpreter scope, and input mutation during observation. The mutant
probe uses a real independently captured ActivityProduct table, changing one
named row, result, alternative order, fact order, binding order, explanation or
pair, plus missing/extra subjects.

Pins intentionally retain source positions. Fresh generation already differed
from tracked pre-change IR before compiler/lint migration; fn156 task 1's
`wildcard/checked-in-baseline-drift.json` records that inherited difference.
Neither fresh originals nor a changed candidate may replace this baseline.

## Finite-domain test inventory

Universal state/class/input assertions enumerate `Finite[T].values` or existing
`classesOf` catalogs. Original named scenario assertions remain alongside these
products. Compile-error, declaration-shape and invalid-token tests are structural
checks, not claims over a finite state domain.

| Assertion family | Complete domain |
| --- | --- |
| Effects lowering/refusal/rule blocks | Snapshot/Job/Task state catalogs (9 each) |
| Inputs named/default/lowered calls | 3/4-input classes (8/16); Counted 6 × 5-input 48; Counted 6 × 6-input 144 |
| Phased projections/composition | Snapshot 9; all before/after pairs (81) |
| Choices guards | Boolean × Boolean (4) |
| Rules switch/stuck/loose/spread/projection/rejection | Lamp 9 × every applicable Knob 2 or complete Lamp catalog |
| Closable properties | Snapshot 3 × Snapshot 3 × Answer 2 (18) |
| Pausable properties | Phase 5 × Phase 5 (25), for record and derived machine |
| Retries settlement | Retryable 2 × Snapshot 48 × Snapshot 48 (4,608); policy counts all 48; optional controls 2 × 2 |
| Deadline firing | Snapshot 160 × Snapshot 160 (25,600) |
| Deadline settlement | Retryable 2 × Snapshot 160 × Snapshot 160 × empty/every-singleton Fact (204,800) |
| Deadline optional controls | RetryState 3 × RetryState 3 (9) |
| Activity dispatch mapping | All ActivityProtocol states, restricted only by the asserted scheduled precondition |
| Activity retry timeout settlement | All ActivityProtocol states, restricted only by the asserted running-phase/armed-timer precondition |
| Existing activity/Nexus row regressions and new generic laws | Every declared finite state and action class under each assertion's explicit applicability precondition |
| Native table replay | Every state/class pair of every exported machine/composition, including disabled and hole dispositions |

Fact lists are not a finite type. Deadline's fact-presence assertions explicitly
check the empty list and each singleton of the entire three-fact domain; this is
not a finite-domain exception. No large finite domain was sampled or exempted.
The initial independent native capture and validation took approximately 131
seconds in total; the fresh tracked-IR replay was approximately 80 seconds.
Per-artifact timing and commands are recorded in the task handover, not asserted
as portable performance budgets.
