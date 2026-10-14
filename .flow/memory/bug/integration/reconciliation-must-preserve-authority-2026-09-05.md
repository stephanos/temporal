---
title: "Reconciliation must preserve authority, IDs, and references"
date: "2026-09-05"
track: bug
category: integration
module: .plans/UMPIRE4_SPEC.md
tags: [umpire, documentation, conformance]
problem_type: integration
symptoms: Current docs reversed runtime ownership and a golden allowed referenced diagnostics to disappear
root_cause: "Vocabulary-only reconciliation did not trace executable ownership, stable IDs, or dynamic reference closure"
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
A documentation reconciliation assigned typed request construction and response projection to the Temporal server Host even though the public Host signature and execution package make the server a transport for already prepared method/request pairs. The same review also found a reused normative ID and a stable projection whose empty dynamic-reference collections could pass vacuously.

## What Didn't Work
Updating related documents and golden fields by vocabulary alone did not prove that ownership statements matched executable interfaces, that replaced rules retained their original IDs, or that excluded dynamic identifiers still preserved their reference relationships.

## Solution
Trace each ownership statement to the public interface and package boundary: the server supplies the authorized descriptor catalog and transports prepared calls, while internal execution constructs requests and projects responses. Retire the old normative rule and add the replacement under the next unused ID. In `tools/umpire/conformance_test.go`, project stable diagnostic kind and code while separately validating unique dynamic IDs and every cleanup-to-Run diagnostic reference.

## Prevention
For documentation cutovers, compare normative IDs against the pre-task document and verify component ownership against actual method inputs and outputs. For closed golden projections, enumerate every excluded dynamic field and test both its structure and its relationships so empty collections cannot make required evidence disappear.
