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
R1: the front refuses `+` and `*` on a length or index operand at its source span (comparison, indexing, `-`, `/` and `%` stay admitted and agree across compiled Rust, the interpreted checker and TLC, since operands stay within TLC's 32-bit integers and a negative difference fails all three alike). The only Model using the refused form was the fn-238.5 probe, now the refused control `lengths.rs:40:20`; no golden changed. R2: TLC evaluation errors (overflow in an invariant, in Next, in an action property, in a multi-line invariant, an out-of-domain tuple) surface as a typed backend error naming the property and state; real violations stay Refuted; a control drives TLC into "Overflow when computing 3*1000000000".

Review: fresh-context round (Opus, same family) SHIP. Known limits recorded, not fixed (owner asked to wrap up): literal-only sums beside a length are refused with a length-specific message; a failed Assert inside a `<>` property stays an untyped TLC execution error; TLC's location-named failing properties are not cross-checked against the run's property; the tool-error control asserts loosely; two fixture comparisons use hard-coded TLC counts; no refusal tests in guards, lets or helpers (probed working).

Gates: worktree default and required `make model-check` rc 0. Merged by 9760f49950. `UMPIRE_BACKENDS=require make model-time` rc 0 at 9760f49950 in a clean worktree from a cold cache: gate 969.974 s against the 60 s soft target, inner loop 9.535 s against 10 s, start load 12-20.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7bd76093f9, 9760f49950
- Tests: UMPIRE_BACKENDS=require make model-time, make model-check
- PRs: