# WASM user stop

The owner narrowed the goal to completing fn-189 and then stopping, then explicitly asked to stop all in-flight tasks safely. The goal is paused. No implementation, integration, review or qualification should resume without a new instruction. Fn-189 remains incomplete; fn-187 and fn-185 are parked and must not be integrated as prerequisites for the narrowed goal.

Every worker reported its owned builds, tests, helpers and descendants stopped and reaped. Interrupted commands retain their nonzero exits; none is a pass. No commit, push, PR, CI action, revert or deletion occurred.

## Fn-189 state

Primary checkout remains `gomad` at `bad3c401b5`. Its uncommitted changes are the task description/state update, milestone status rows and the one-line Rust cluster fixture correction (4,000 loop iterations, unchanged 20,000 fuel limit). The corrected full Rust suite passed 420 tests; 20 detached artifact replays passed. Runtime qualification completed only 18 of 32 seeds before the source manifest correctly refused the changed fixture. The full WASM suite and lint were interrupted and are not passing final gates.

The five mapped pack source reconciliations remain isolated and unreviewed in `/home/agent/wt/wasm-milestones-pins`. Source-only review, generation, pack check, mapped refresh and focused source controls passed there. Lint was stopped; independent review and integration remain pending. Historical alternative-version packs were preserved. Raw global pin-impact still reports historical rows; the mapped workload reconciliation is the relevant existing source gate.

Qualification handover: [fn189-linux-qualification-2026-10-17.md](fn189-linux-qualification-2026-10-17.md). Pack handover: `/home/agent/wt/wasm-milestones-pins/.flow/artifacts/fn189-pack-source-reconciliation-2026-10-17.md`.

If resumed, work only on fn-189: review and integrate its pin repair, retain the fixture correction, freeze that candidate, finish the required source/runtime/workload/resource gates and independent acceptance review, then stop. The earlier proposal to combine fn-187 acceptance with fn-189 is superseded by the narrowed scope. Native execution stays deferred. Commits remain with the owner.

## Preserved worktrees

No worktree was removed. Dirty candidate trees retain uncommitted work; inherited trees remain untouched; clean baselines retain evidence inputs.

| Path under `/home/agent/wt/` | Reason kept |
| --- | --- |
| `fn184`, `fn184-base`, `fn185`, `fn187`, `fn189` | Inherited source/evidence baselines and owner work |
| `fn189-cluster-baseline` | Clean historical cluster comparator at `51604889c9` |
| `wasm-milestones-pins` | Uncommitted fn-189 source-only pack repair and evidence |
| `wasm-milestones-fn187` | Uncommitted corpus/module and measurement extraction candidate, parked |
| `wasm-fn187-baseline` | Baseline with the same test-only fuel fixture correction, parked |
| `wasm-fn187-coverage` | Interrupted coverage proof and uncommitted corpus work, parked |
| `wasm-fn187-measure` | Frozen measurement extraction and assertion-preservation evidence, parked |
| `wasm-fn187-model-corpus` | Partial model corpus expansion; latest changes require verification, parked |
| `wasm-fn187-network-corpus` | Partial network corpus expansion; latest Rust run interrupted, parked |
| `wasm-fn187-volume-corpus` | Partial volume corpus expansion; latest generation requires verification, parked |
| `wasm-milestones-fn185` | Uncommitted profiling merge and provisional fn-187 dependency overlay, parked |

Fn-187 handover lives in its candidate's `.flow/artifacts/fn187-milestones-2026-10-17.md`; fn-185 handover lives in its candidate's `.flow/artifacts/fn185-milestones-2026-10-17.md`. Their overlays and partial evidence are not accepted primary source.

Raw logs, staged inputs and build caches remain under primary `.flow/tmp/fn189-linux/`, the pin candidate's `.flow/tmp/fn189-pins/`, worktree `.toolchain/` directories, `/home/agent/wt/evidence/fn187/current-coverage/`, `/home/agent/.cache/wasm-milestones-fn185-target/` and `/home/agent/.cache/codex-build/fn187-network-corpus/`. They are retained for inspection/resumption; interrupted results require verification before use.
