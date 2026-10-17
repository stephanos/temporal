---
satisfies: [R20]
---
# fn-190-models-are-plain-scala-state-as-vars.14 Remove full reachability tables from native checks

## Description
Owner follow-up 2026-10-17: remove the separate eager full-reachability-table path. Use shared on-demand native successor generation for safety, witness/refutation, monitor and progress/fairness checks. Retain visited states and witness/cycle bookkeeping as needed, never a complete adjacency table or optional full-graph backend. Safety checks, witness/refutation searches and monitors stop generating successors when their answer is conclusive; successful finite verification may exhaust reachable states. Progress may exhaust reachable states to preserve cycle/fairness and diagnostic ordering, regenerating successors without storing adjacency. Budget cuts remain unknown unless a counterexample or witness already establishes an answer. Preserve existing claim, channel, fairness and reporting semantics. Native framework only; IR/schema/Go/Temporal holds remain.

## Acceptance
No Graph/full adjacency table API remains. Free find, verification, negative controls and monitors stop at their first decisive result, including initial-state failures without effects. Shared exploration respects budgets and holes. Successful safety exhausts finite reachability and cuts return unknown. Progress/fairness and deterministic witnesses retain existing test results without stored adjacency. Focused instrumentation controls, native compatibility tests and scoped lint pass.


## Done summary
Removed NativeCheck.Graph and its full adjacency table. Shared lazy Search generates native/channel successors on demand, retaining visited states and predecessor paths. Safety checks, free find/refutation searches and direct monitor-product exploration stop when conclusive. verifyAll evaluates safety sections in one pass and retains completion-only vacuity metadata. Progress retains its verdict and diagnostic ordering by exhausting visited states and regenerating successors for region/SCC/fairness/longest/cycle analysis; no adjacency table or alternative full-graph backend remains. Tarjan consumes successor cursors. A consumed search starts fresh for later checks.

Verification: 502 DSL/realize tests pass, including 14 streaming/reuse controls; all eight selected sample suites pass 42 tests, including Bank Leaky, Kettle progress/fairness and cache monitors. Source lint passes 29 tests, scoped formatting and git diff checks pass. Baseline regression run reproduced eager execution before the replacement; instrumentation accounts for existing native guard/effect replay. Fresh independent read-only review found no remaining material issue.

The unrestricted native batch was stopped with exit 143 during the existing expanded Bank successful safety proof in NativeMapsTest; that full proof is not counted as green. No user Bank/Bank.test/NativeMaps/NativeControls edits were staged or committed. IR/schema/Go/Temporal holds remain unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6c16ca3da3
- Tests: mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize --require-tests (502 tests; .flow/tmp/streaming-core-gate.log), mise exec -- scala-cli run --test --server=false model/framework/dsl model/framework/realize model/framework/samples --main-class org.junit.runner.JUnitCore -- framework.dsl.BankTest framework.dsl.KettleTest framework.dsl.ReliableBroadcastTest framework.dsl.TwoLayeredCacheTest framework.dsl.TwoPhaseCommitTest framework.dsl.VotingTest framework.dsl.NativeHistoryTest framework.dsl.NativeVotingTest (42 tests; .flow/tmp/streaming-samples-gate.log), mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite' (29 tests; .flow/tmp/streaming-lint-gate.log), scoped scalafmt check (.flow/tmp/streaming-fmt-check.log), fresh-context read-only review (.flow/tmp/streaming-review.txt)
- PRs: