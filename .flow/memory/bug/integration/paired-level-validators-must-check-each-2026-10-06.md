---
title: Paired level validators must check each artifact independently
date: "2026-10-06"
track: bug
category: integration
module: model/irgen/src/main/scala/modelir/lint/Structure.scala
tags: [umpire, model, structure-lint, refinement, ownership]
problem_type: integration
symptoms: Malformed Product/System layouts passed when both suffixes were wrong or System omitted refinement
root_cause: Sibling validation was conditional on partial resolution and two-level intent was inferred too narrowly
resolution_type: fix
last_updated: "2026-10-06"
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/channel-catalogs-and-visible-results-2026-09-30, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
The structure lint accepted malformed two-level examples when Product and System machine-name validation short-circuited through a shared conditional, and it did not reject a System level that omitted its Product refinement. Review fixtures with both suffixes wrong and with missing refinement exposed the bypass.

## What Didn't Work
Validating suffixes only after one level happened to resolve, and inferring two-level intent only from a detected refinement, allowed independently invalid or incomplete level files to escape the structural contract.

## Solution
Resolve Product and System machines independently from their level files, validate each required suffix unconditionally, require their common feature prefix, and require the System machine to refine the Product machine whenever both level files exist. Focused malformed-layout fixtures pin the both-wrong and missing-refinement cases.

## Prevention
For structural validators over paired artifacts, create one negative fixture per independently required invariant and one compound-invalid fixture. Do not let successful resolution of one artifact gate validation of its sibling.

## Update 2026-10-06

## Problem
The structure lint accepted malformed two-level layouts through two independent gaps. Product and System name/refinement validation once short-circuited through a sibling, and later root-name refusals did not require level-owned `Phase`, `State`, and `Fact` to be declared in the canonical level files. The ownership migration also left System-only timer and composition types in the feature root.

## What Didn't Work
Checking only forbidden root spellings proves where a few known declarations are not; it does not prove where the required declarations are. Likewise, treating every root type as shared without checking its consumers leaves level-only vocabulary behind. A blanket exemption for `shared` packages would turn the task queue's named `QueueView`/`QueueDetail` exception into a general loophole.

## Solution
Resolve Product and System machines independently, require the System refinement, and when their state or fact types differ require `Phase`, `State`, and `Fact` in each canonical `product/Product.scala` and `system/System.scala`. Reject those names in sibling level files. Exempt only `temporal.shared.taskqueue`, whose cross-level vocabulary is explicitly shared. Move types consumed only by the System level into `system/System.scala`, then prove regenerated IR/Cases equivalent through the exact type-identity ledger.

## Prevention
For paired structural artifacts, test absence, wrong placement, and independent sibling failures rather than only known forbidden names. During ownership migrations, search every remaining root type's consumers and keep it there only when both levels use it. Encode named exceptions as exact identities, not category-wide exclusions.

## Update 2026-10-06

## Problem
Kind/form admission introduced three structural bypasses. Rebasing a form before comparing its original package to its source path concealed a wrong parent kind. An inherited kind Product made a System folder two-level, but a missing canonical System.scala escaped when only an unrefined or derived sibling remained. Machine-only placement checks also admitted unknown or deeper folders containing types alone.

## What Didn't Work
Resolved machine pairs cannot establish whether a required level file exists. Likewise, checking only machine objects misses type-only sources, and comparing paths after a semantic rebase loses the original ownership boundary.

## Solution
Check each original source package before rebasing forms. Require the canonical System file whenever an inherited Product and a System folder establish that level, independently of detected refinement. Validate every declared source against the closed kind/form folder catalog. Keep the existing flat/shared primary, refinement, vocabulary and taskqueue checks unchanged. Focused refusal fixtures demonstrate all three bypasses, including both unrefined and valid derived siblings.

## Prevention
Test the original ownership coordinate, missing canonical files with non-primary siblings, and type-only wrong-folder declarations. Establish diagnostic red before changing the classifier. A malformed Derived expression is not evidence for a missing-file guard; use a valid existing DSL spelling and distinguish that failed probe from the intended reproduction.
