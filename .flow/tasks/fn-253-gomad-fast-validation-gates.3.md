---
satisfies: [R4]
---
# fn-253-gomad-fast-validation-gates.3 Content-keyed stage reuse for development cycles

## Description
R4: content-keyed reuse of stage receipts for development cycles.

The reuse key is a hash over the stage's full input closure:
- its leaf argv and environment
- the toolchain digests
- the frozen files the stage reads (from the stage table, falling back to the whole inventory when undeclared)
- the receipts of the stages it depends on

A matching green receipt is copied in and marked `reused`, with the original cycle ID.

Reuse is off by default. It is refused in fn-246 final qualification and in any cycle that claims closure; the cycle's verdict records how many stages were reused. fn-178's `--reuse` keyed on git pathspecs, and this replaces that approach because a pathspec misses inputs it doesn't name.

Ordering: after the fn-242 cycle verb cutover, and after fn-253.2, so the cost classes show which stages are worth reusing.

## Acceptance
- For each stage family, a mutation of one closure input (a source file, toolchain, config, or upstream receipt) invalidates reuse (e2e mutation fixtures). A mutation outside the closure keeps the reuse.
- A closure-claiming cycle and a fn-246 qualification refuse `--reuse` with a typed error.
- A development cycle with no source change reuses every reusable stage, and its wall time is recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
