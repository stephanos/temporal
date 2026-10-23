---
satisfies: [R1, R2]
---
# fn-242-length-arithmetic-parity-across.1 len() arithmetic parity and TLC evaluation-error parsing

## Description
After fn-238.5 is integrated. See the spec. Decide refusal in the front (recommended) versus bounded emission; make the usize_lengths probe refused or agreeing across compiled, interpreted and TLC; parse TLC evaluation errors into a typed located backend error with a control. Goldens unchanged unless a fixture used the refused form (list each).

## Acceptance
- [ ] len() arithmetic agrees across compiled, interpreted and TLC, or is refused at its span
- [ ] TLC overflow/evaluation errors surface as typed located backend errors, with a control
- [ ] Default and required model gates pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
