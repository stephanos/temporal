---
title: Paired level validators must check each artifact independently
date: "2026-10-06"
track: bug
category: integration
module: model/irgen/src/main/scala/modelir/lint/Structure.scala
tags: [umpire, model, structure-lint, refinement]
problem_type: integration
symptoms: Malformed Product/System layouts passed when both suffixes were wrong or System omitted refinement
root_cause: Sibling validation was conditional on partial resolution and two-level intent was inferred too narrowly
resolution_type: fix
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
