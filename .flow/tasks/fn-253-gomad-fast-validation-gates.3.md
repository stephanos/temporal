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
Blocked:
# fn-242 parked by owner

The owner requested: "can we move past fn-242 for now and leave that be?"

Stop fn-242 source changes, preparation cohorts, callbacks, proof reconciliation, reviews and task closure. Resume only on explicit owner direction. Preserve the worktrees, commits, admissions and evidence; parking supplies no completion credit.

Ordinary Rust lint and test suites under fn-253 may exercise existing cycle modules. This does not resume fn-242 acceptance work or authorize its standalone preparation/proof campaigns.

The last frozen source is `7e713d3d1a4adee10e85917a72c102a72e470dc1`. Its prepare-only run completed all 15 admissions, then refused configuration because setup supplied empty `live_corpus.families`. There is no genuine configuration or executed B10 callback proof from this run. Preserve `/home/agent/wt/evidence/fn242-b11-workload-reference-oracle-producers/genuine-prepare-7e713-run.json` and `genuine-prepare-7e713-root` unchanged.

The process-field source checkpoints `01902d7970` and `46e77aa9a7` and B10 landing `9469156b9d` remain unaccepted work. Existing passing checks retain only their original scope.

The overall MILESTONES goal continues through independent fn-253 work: inner loop, nextest, standalone entrypoints, gate visibility and evidence-store work that does not resume the parked cycle integration. fn-253.2/.3 still wait for cutover; downstream dependency gates are not waived. No full qualification attempt has been used: 0 of 2.
## Evidence
- Commits:
- Tests:
- PRs:
