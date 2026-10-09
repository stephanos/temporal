---
title: Schema closure tests must compile imported generated declarations
date: "2026-10-09"
track: bug
category: integration
module: model/check/test/Gate.test.scala
tags: [umpire, protobuf, scalapb, codegen]
problem_type: integration
symptoms: Stubbed gate tests never compiled an imported ScalaPB declaration
root_cause: The fixture logged commands and touched a jar instead of exercising real code generation
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/transition-table-reads-must-omit-2026-10-06, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem

The schema-closure tests asserted the generated command line with stand-in `protoc` and `scala-cli` scripts, but never proved that a declaration in an imported proto was generated and compiled by the real toolchain. A broken multi-file ScalaPB invocation could therefore satisfy the unit tests.

## What Didn't Work

The synthetic gate repository logged arguments and touched the expected jar. That covered closure ordering, hashing, and invalidation, but the fake jar could not establish that ScalaPB emitted imported declarations or that Scala accepted their generated types.

## Solution

`model/check/test/Gate.test.scala` now creates a real two-file proto closure, runs it through the production `Gate.generateIr` path with pinned `protoc`, ScalaPB, and Scala, then compiles a probe that constructs both the root and imported generated classes.

## Prevention

When an acceptance criterion requires code generation across file boundaries, keep the fast stubbed behavior tests but add one focused real-toolchain fixture that consumes a generated declaration from the imported file.
