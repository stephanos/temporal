---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-132-group-the-nexus-and-activity-models-by.7 Close: requirement check, MILESTONES, spec

## Description
**Size:** S
**Touches:** [model/README.md, .plans/UMPIRE_MODULES.md, .plans/ACTIVITY_MODEL_COMPARISON.md, MILESTONES.md, .flow/specs/fn-125-represent-dynamic-configuration-in-the.md, .flow/specs/fn-125-represent-dynamic-configuration-in-the.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.3.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.3.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.6.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.6.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.7.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.7.json, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.8.md, .flow/tasks/fn-125-represent-dynamic-configuration-in-the.8.json, .flow/tasks/fn-119-show-one-go-sdk-workflow-driven-end-to.5.md, .flow/tasks/fn-119-show-one-go-sdk-workflow-driven-end-to.5.json]

**Required investigation:** all task handovers, R1-R5, current generated/model trees, downstream Flow plans and review receipts. Preserve existing spec dependency edges as history after dependencies close; only the overview's present-tense gates lose completed entries.
The conductor owns whole-spec completion review and closure. This task prepares/verifies linked requirement evidence and final docs; it does not close the parent before conductor completion SHIP.

Check R1-R5 against the tree and every task's done evidence, including the source-grouping prerequisite. Re-run the Model-path-specific old-name search across the repository. Verify downstream spec paths/gates and record the last applicable full boundary results without repeating unaffected suites. Leave the parent and its MILESTONES block for conductor-owned completion review/closure.

**Downstream path-maintenance scope:** The finite downstream Flow authoring paths above are writable only to make current Model paths and literal references match the completed kind/form tree. Preserve their deferred status, dependencies, semantics and historical snapshots; remove obsolete source-line ranges instead of inventing replacements. Do not resume any deferred implementation. This aligns the write surface with the existing final-docs/downstream-path acceptance.
## Acceptance
- [ ] Each of R1-R5 is met, with the evidence linked.
- [ ] Final docs and downstream Flow paths match the current tree; MILESTONES keeps completed tasks until the conductor closes the parent.
- [ ] Evidence is ready for whole-spec completion review, with no premature parent closure or claim of merged delivery.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
