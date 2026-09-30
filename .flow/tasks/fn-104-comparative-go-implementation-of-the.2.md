# fn-104-comparative-go-implementation-of-the.2 T1 Lean dumper and committed parity dumps

## Description
TBD

## Acceptance
- [ ] TBD

## Done summary
`experiments/umpire-go/lean/Dump.lean` reads the production Nexus caller Model through the public names its pins use and writes 21 JSON files to `parity/testdata/lean/`: the tables of nexusProduct, nexusProtocol, the worker's polling machine, handlerWorker and the nexusCaller composition; their Definition IDs; the protocol refinement rows; every Query's outcome and witness; and the 889 exploration targets.

Findings the Go port must match:
- action classes are sorted by key (nexusProtocol has 23, not 22 as the plan said);
- in a state key the last field varies fastest; only enabled rows are listed, states-major;
- reachability is `reachableFrom`'s repeated sweep over rows in table order;
- a refinement row maps each result to a product action key, preferring the same-named action, or to null for a stutter (456 of 1,152);
- composed state keys join member keys with `_` and member actions are `<field>_<key>`.

Checks: counts equal the pins (6/12/25, 192/23/1152, 158 reachable, 96 ends, 316 states and 1,468 rows for the composition); targets equal `CallerExploratoryCoverage.json`; two runs are byte-identical. `dump.sh` writes to a scratch directory and fails without touching committed dumps on a Lean error.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: experiments/umpire-go/lean/dump.sh (twice, diff -r identical)
- PRs: