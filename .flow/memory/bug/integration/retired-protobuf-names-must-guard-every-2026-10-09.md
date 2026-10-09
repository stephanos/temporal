---
title: Retired protobuf names must guard every top-level declaration kind
date: "2026-10-09"
track: bug
category: integration
module: tools/umpire/ir/schema_test.go
tags: [umpire, protobuf, schema, retirement]
problem_type: integration
symptoms: A retired protobuf message name could be reused by a top-level enum without failing the schema ledger.
root_cause: The retirement guard inspected only added messages instead of the combined package-level declaration namespace.
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/schema-closure-tests-must-compile-2026-10-09, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/transition-table-reads-must-omit-2026-10-06, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem

The descriptor-retirement ledger protected a removed protobuf message name only against later message additions. A top-level enum could reuse the same package-level name without tripping that guard, weakening the promise that the retired declaration stays retired.

## What Didn't Work

Checking `schemaAddedMessages` encoded one declaration kind instead of the package-level namespace that protobuf messages and enums share.

## Solution

Build the current schema declaration union and reject each retired name from its combined top-level message-and-enum names in `tools/umpire/ir/schema_test.go`.

## Prevention

Retirement checks should query the canonical declaration namespace, not one ledger list or one concrete descriptor kind.
