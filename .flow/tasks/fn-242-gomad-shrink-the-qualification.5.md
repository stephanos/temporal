---
satisfies: [R3]
---
# fn-242-gomad-shrink-the-qualification.5 Dispose parity lanes and preserve required Go baseline inputs

**Touches:** .flow/artifacts/fn242-handoff-*, .flow/artifacts/fn242-obligation-inventory*, .flow/artifacts/fn242-parity-dispositions.md, .flow/artifacts/fn242-size.md, .github/workflows/gomad.yml, tools/gomad/ARCHITECTURE.md, tools/gomad/CLI.md, tools/gomad/Makefile, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/determinism/inventory.json, tools/gomad/rust/driver/**, tools/gomad/rust/gomad/src/**, tools/gomad/rust/gomad/tests/**, tools/gomad/rust/wasmhost/src/**, tools/gomad/rust/wasmhost/tests/**, tools/gomad/rust/workload_tools/**, tools/gomad/scripts/**, tools/gomad/testdata/parity/**

## Description
For every lane under tools/gomad/testdata/parity/ (about100 Python files) and scripts/behavioral-parity and all-verb-parity, record keep (golden regression driven from outside), fold (into Rust tests) or retire with explicit replacement coverage. Carry out each disposition and update the Make parity target. Inventory all pending R4/public18 consumers before removing any dependency. Preserve the independently usable frozen Go executable/helper/source/compiler/tool pins and raw proofs at /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a and other receiving-owned roots until fn-246.6/.7 authenticate the original comparisons and release the inputs. A retained qualified reference lane may keep its baseline dependency while that acceptance is pending. Disposition paperwork and golden-only replacement do not themselves credit fn-232 R13 or fn185 R4. Record this pending ownership explicitly; do not complete source owners here. Touches: tools/gomad/testdata/parity/**, tools/gomad/scripts/**, tools/gomad/Makefile, possible folded Rust test destinations under tools/gomad/rust/{gomad,wasmhost}/src/** and tests/**, .flow/artifacts/fn242-parity-dispositions.md. The disposition inventory records exact Rust destinations before writes. This task runs after .2/.3/.4/.6/.9/.10 so folded tests cannot race their implementation surfaces; read-only disposition analysis may prepare earlier.

## Acceptance
- [ ] Disposition table covers every lane file and both scripts, records exact Rust fold destinations, and performs writes only after the overlapping implementation tasks finish
- [ ] Kept regression lanes run from the Make target; folded lanes have Rust tests; retired lanes cite replacement coverage without losing any pending fresh-reference comparison
- [ ] Frozen Go executable/helper/source/compiler/tools and raw proof inputs remain independently usable until actual fn-246 R4/public18 receiving; consumer inventory names the owner and release condition
- [ ] Record fn-232 R13 and fn185 R4 as pending actual proof; dispositions alone produce no qualification or behavioral-equivalence pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


## Owner-authorized batch B16 (parity-cutover-handoff)

This task is the execution leader for fn-242-gomad-shrink-the-qualification.5, fn-242-gomad-shrink-the-qualification.7, fn-242-gomad-shrink-the-qualification.8. Before implementation or review, read every member task and `.flow/artifacts/fn242-execution-batches.json`. Implement every member's original acceptance and exact proof ownership in the manifest's internal order. Reserve the full Touches union and wait for every external dependency. Use shared genuine fixtures, one independent whole-batch review (3 draw(s)), and representative high-risk source mutants. Member task receipts follow actual batch SHIP in topological order with per-member evidence; no internal review/worker barrier or acceptance waiver.
