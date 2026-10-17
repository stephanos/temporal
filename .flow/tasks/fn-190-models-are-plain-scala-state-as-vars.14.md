---
satisfies: [R20]
---
# fn-190-models-are-plain-scala-state-as-vars.14 Remove full reachability tables from native checks

## Description
Owner follow-up 2026-10-17: remove the separate eager full-reachability-table path. Use shared on-demand native successor generation for safety, witness/refutation, monitor and progress/fairness checks. Retain visited states and witness/cycle bookkeeping as needed, never a complete adjacency table or optional full-graph backend. Safety checks, witness/refutation searches and monitors stop generating successors when their answer is conclusive; successful finite verification may exhaust reachable states. Progress may exhaust reachable states to preserve cycle/fairness and diagnostic ordering, regenerating successors without storing adjacency. Budget cuts remain unknown unless a counterexample or witness already establishes an answer. Preserve existing claim, channel, fairness and reporting semantics. Native framework only; IR/schema/Go/Temporal holds remain.

## Acceptance
No Graph/full adjacency table API remains. Free find, verification, negative controls and monitors stop at their first decisive result, including initial-state failures without effects. Shared exploration respects budgets and holes. Successful safety exhausts finite reachability and cuts return unknown. Progress/fairness and deterministic witnesses retain existing test results without stored adjacency. Focused instrumentation controls, native compatibility tests and scoped lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
