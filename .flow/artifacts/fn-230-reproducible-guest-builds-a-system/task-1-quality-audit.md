### Correctness axis

## Quality Audit — Correctness axis: fn-230.1 source checkpoint

### Summary

- Files changed: 7 · Critical 0 · Should Fix 0 · Consider 0 · Ship: ✅ Ship source checkpoint only.
- No new correctness defects found. Task/spec completion remains open for actual WASI unwinding and macOS/cross-OS proof.

### Evidence

- **Executed:** focused Python build/check controls, Bash syntax checks, and `git diff --check` passed; working tree stayed clean.
- **Walked:** Cargo failure preserves the output; stale/missing checks return 2 without copying; artifact selection handles custom library names; path remaps include generated target sources.
- **Walked:** parity controls exercise observation and outcome-comparison panics and require trap context, instance invalidation, and Reset rejection.
- Full gates and real-artifact parity were not rerun during this audit.

### Test Budget

- Added tests:implementation ≤129:80 lines including fixture/setup code, below 2:1. No existing tests changed.

### What's Good

- DESIGN.md explicitly distinguishes Linux path-independence evidence from unresolved cross-OS and unwind requirements.
- The builder uses locked Cargo artifact messages and leaves wasm bytes untouched after linking.

### Standards axis

## Quality Audit — Standards axis: fn-230.1 source checkpoint

### Summary

- Files changed: 7 · Should Fix 1 · Consider 0 · Blocking: none possible (standards axis)
- Reviewed `b27d38e..edb0078`; working tree remained clean.

### Should Fix (High priority)

- **model/DESIGN.md:140** (Conf 100): The added section crosses the 1,000-line threshold, growing DESIGN.md from 978 to 1,079 lines (executed) — extract detailed reproduction commands into a focused document, retaining the proof data, limitations and link in DESIGN.md.

### Out-of-axis observations

> Out-of-axis observation: **model/DESIGN.md:160** — cited unresolved requirement: the actual WASI profile uses abort rather than the required caught-panic judgments.
>
> Out-of-axis observation: **model/DESIGN.md:210** — cited unresolved requirement: macOS and cross-OS reproducibility remain unverified.

### What's Good

- Walked: the builder discovers artifacts without system crate names; controls reuse the existing host suite without production changes.
- Executed: Bash syntax and diff whitespace checks passed; no dependency additions appeared in the diff.

Correctness axis: 0 findings; worst tier none.
Standards axis: 1 finding; worst tier Should Fix.

Conductor disposition: fix only the new proof-section layout using a linked focused document; retain proof data and explicit incomplete requirements in DESIGN.md. No existing unrelated section moves, no requirement waiver and no task/spec completion claim.

## Verified documentation correction

Commit `5d5637a2ab76adc4b06253ab0688bef84c7bf3bf` changes only DESIGN.md and the new GUEST_BUILDS.md. The conductor verified a clean lane, clean scoped diff, existing documentation target, DESIGN.md at 998 lines, and unchanged proof-artifact SHA-256 values. This resolves the standards finding without discharging actual WASI unwinding or macOS/cross-OS proof. No task or spec completion is claimed.

## Integrated source checkpoint

The conductor integrated the checkpoint at `183e22fccce1f1badde0cc8ab2eb3ad78355f03e`, after fn-217.2's verified closure. The combined `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/umpire-integrated make model-check` exited 0 (`.flow/tmp/fn2301-integrated-model-check.log`). Focused Python build/fresh/stale/failure controls, Bash syntax checks and diff checks also exited 0. These checks verify integration of the source checkpoint only. Task fn-230.1 remains in progress, with both original proof obligations open.

## Preserved active proof evidence

After integration the worker moved its owned proof files, logs and clean source clones to `.flow/tmp/fn2301-proof-lane/`, preserving tracked older files. `RELOCATION.md` records the exact mapping and retains historical paths as historical evidence. The conductor ran `sha256sum -c fn2301-preserved-sha256.txt` from that directory; all four original final artifacts passed. The worker removed the clean integrated worktree and its local branch without force. Task .1 remains open, and active proof evidence was preserved rather than cleaned as a closed spec.

