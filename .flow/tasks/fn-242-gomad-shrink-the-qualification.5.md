---
satisfies: [R3]
---
# fn-242-gomad-shrink-the-qualification.5 Parity-lane disposition: keep, fold or retire each lane; retire the Go baseline

## Description
For every lane under tools/gomad/testdata/parity/ (about 100 Python files) and scripts/behavioral-parity and all-verb-parity: record keep (golden regression driven from outside), fold (into Rust tests) or retire (with the out-of-tree Go baseline at /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a). Carry out each disposition; update the Make parity target; note in fn-232 that R13 is satisfied by the kept lanes. Retired lanes must show where their checks live now. Touches: tools/gomad/testdata/parity/**, tools/gomad/scripts/**, tools/gomad/Makefile, .flow/artifacts/fn242-parity-dispositions.md.

## Acceptance
- [ ] Disposition table covers every lane file and both scripts
- [ ] Kept lanes run from the Make target against the Rust binary only; folded lanes have Rust tests; retired lanes cite replacement coverage
- [ ] Go baseline dependency removed from the kept lanes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
