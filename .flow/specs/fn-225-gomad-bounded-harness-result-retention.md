# Gomad bounded harness result retention

The owner approved changing the harness to replace completed results rather than accumulate every attempt. Qualification currently retains duplicate prepared WASM modules, compiled caches, workload directories and replay artifacts, exhausting local disk. Enforce retention in code rather than agent instructions.

## Acceptance Criteria

- **R1:** Repeated runs for the same explicit output root and test/configuration publish a complete latest result without mixing files from different attempts. Stage a run separately, publish only once complete, and preserve the previous complete result on interruption or publication failure. Keep identity-bound paths valid for retained receipts.
- **R2:** Automatically reclaim superseded disposable outputs after their last consumer releases them. Keep bounded failed-run diagnostics and explicitly retained baseline/replay inputs. Do not delete active, pinned, unrelated or historical evidence implicitly. Concurrent runs must not delete or overwrite one another's inputs.
- **R3:** Exercise the real harness retention path with lightweight filesystem tests: repeated success has bounded disk growth; failed/interrupted replacement preserves the previous result; overlapping runs and pinned inputs are safe; unrelated files and symlink targets cannot be removed. Retain existing replay, admission and error contracts. Document defaults and explicit keep behavior briefly.

## Scope and design

Implement one small integrated change in the existing WASM measurement/qualification harness and its shared output lifecycle. Inspect existing publication and receipt owners before choosing the narrowest shared seam. Separate disposable execution scratch from published evidence. Prefer stable latest-result discovery with immutable retained run paths where authenticated receipts require them; replace the published result as a unit, not individual files. Avoid a general artifact service, new dependencies, duplicated capture machinery or a new CLI command unless an existing entry point cannot express retention safely. Record any necessary compatibility decision in this spec before implementation.

Existing external evidence is cleaned separately only after its consumers finish. Do not mutate the frozen upstream-sync candidate or its active inputs. Implement in the isolated worktree, integrate after fn-189 qualification and review. Tests and lint should cover the changed harness owner; rerun expensive WASM gates only for a concrete affected contract.

Owner decision, 2026-10-19. Prioritize fn-225 immediately after fn-189, before
fn-187 and the remaining milestones. The current run stops after fn-225 and
cleanup of released artifacts/worktrees.

## Verification

Use test-first filesystem/integration tests with -tags test_dep, affected package tests, affected lint, and source/diff checks. Show repeated-run storage remains bounded without performing another multi-gigabyte qualification solely for this feature.

## Compatibility decision

The shared measurement runner stages each producer's explicit output root under
`.gomad-measure/<producer>/run-*`. These paths never move. Authenticated reports
and admission receipts continue naming the original bytes. `latest.json` in that
producer directory atomically selects a completed run after producer cleanup.
Published rerun metadata keeps the original requested output root, so reruns
replace results in that same store instead of nesting stores inside prior runs.
Internal namespace samples retain their parent's authenticated index paths.
The default retains the latest success and one recent failed/interrupted run;
active runs, direct managed-input pins and explicit `--keep` runs are exempt.
Retained consumers hold hardlinks to managed input run tokens. Collection removes
those links and revisits released inputs, preserving transitive receipt paths
without retaining every helper/module forever. Cross-filesystem pin creation
refuses before executing the consumer. Historical/unmanaged evidence is untouched.

| Requirement | Task |
| --- | --- |
| R1 | fn-225-gomad-bounded-harness-result-retention.1 |
| R2 | fn-225-gomad-bounded-harness-result-retention.1 |
| R3 | fn-225-gomad-bounded-harness-result-retention.1 |
