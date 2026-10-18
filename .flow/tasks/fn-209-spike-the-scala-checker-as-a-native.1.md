---
satisfies: [R1]
---
# fn-209-spike-the-scala-checker-as-a-native.1 Can the claim-section rewriter become a compile-time macro or plugin phase?

## Description
Replace the load-time bytecode rewriting of claim sections with a compile-time mechanism on one sample (macro or a reify-plugin phase), or record with evidence why it cannot be done. No production change; the proof lives in `.flow/tmp` and the design note.

**Files:** `model/framework/checker/NativeClaimSections.scala` (read), the existing framework compiler/plugin sources (read), `model/framework/design/native-checker-spike.md` (write).
**Touches:** [model/framework/design/native-checker-spike.md, .flow/tmp/fn209/**]

**Admission, 2026-10-18:** the owner explicitly requested a parallel fn-209 subagent while fn-195.20 continues. This admits this isolated feasibility spike ahead of its previous Phase 4 placement, not the remaining Phase 4 series. The design note follows the already committed `model/design` to `model/framework/design` relocation. Prototype sources and build products remain task-owned scratch, with no production claim-rewriter replacement. Use a separate worktree and fresh complete-recipe build workspaces; coordinate the shared compiler/toolchain lane with the conductor. The fn-195.20 matrix was terminal GREEN before this lane acquired compilation authority.

**Quick commands:** the original `mise exec -- scala-cli test --server=false model/framework/checker model/framework/samples --require-tests` on the prototype branch remains the stated comparison target. The owner-removed repository `model/build/ir-scalapb.jar` must not be regenerated or restored. Use a fresh scratch source/library baseline with the retained immutable dependency, strict native options and positive executed sample/checker counts; record why this is not a canonical repository gate. Any blocker must be evidenced, not inferred from missing tooling alone.

Keep Temporal excluded entirely. This admission permits fn-209-specific scratch compile-time mechanism probes, not a full export-module compile or the still-unapproved held-suite metadata compilation. No Go, backend, schema generation, Case production, server operation or production build target in this task. Later fn-209 target tasks require their own dependency-ready admission. No fn-203/fn-204 recut before the final spike decision.
## Acceptance
- Design note section with the working replacement or the recorded blocker.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
