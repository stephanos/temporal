---
title: Required numeric generation must sample present candidates
date: "2026-10-21"
track: bug
category: integration
module: model/umpire/annotations/src/value.rs
tags: [umpire, annotations, generation, presence]
problem_type: test-failure
symptoms: Required Number fill with seed 3 refused satisfiable default constraints
root_cause: Default generate_present sampled absent zero before checking presence
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/preserve-explicit-replay-bindings-when-2026-10-13, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/retired-protobuf-names-must-guard-every-2026-10-09, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/schema-closure-tests-must-compile-2026-10-09, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/transition-table-reads-must-omit-2026-10-06, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
The new floating-point FieldValue implementation inherited the default generate_present adapter, which samples an ordinary value and then refuses if it is absent. With seed3, required Number fill sampled zero twice and returned Invalid although nonzero values satisfied every constraint. Formal integration review found the defect; the focused real fill regression reproduced it.

## What Didn't Work
Sampling first and checking presence afterward can confuse an unlucky default-valued candidate with impossible constraints. Repeating a bounded random draw does not establish satisfiability and still permits deterministic failure.

## Solution
The shared private generate_number helper in model/umpire/annotations/src/value.rs filters absent zero from candidates before present-value sampling. Ordinary generation keeps its previous candidate order, filtering and RNG selection. Required fractional intervals remain constructible, while a truly zero-only constraint refuses present generation. String, bytes and integral implementations already override present generation; bool always generates true.

## Prevention
Every new scalar FieldValue needs a required-field fill regression using a seed that selects its absent default through ordinary generation. Test satisfiable constrained intervals and a genuinely default-only impossible presence case. Construct the legal present candidate set before sampling instead of adding retries.
