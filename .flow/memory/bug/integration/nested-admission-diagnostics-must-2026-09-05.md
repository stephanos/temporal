---
title: Nested admission diagnostics must retain exact source coordinates
date: "2026-09-05"
track: bug
category: integration
module: model/Umpire/Property/Check.lean
tags: [umpire, property, diagnostics, provenance]
problem_type: integration
symptoms: Nested Property failures with distinct coordinates produced indistinguishable diagnostics
root_cause: The new nested checker reused a legacy path-only error field and container-level duplicate attribution
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Nested guarded Property diagnostics retained only a path even though every authored declaration carried a complete `SourceLocation`. Errors on different lines were indistinguishable, and duplicate IDs were attributed to the containing declaration rather than the second offending declaration.

## What Didn't Work
Routing every nested validation failure through the legacy path-only `PropertyError.sourcePath` field preserved the old error shape but silently discarded line, column, and provenance at the new nested authoring boundary.

## Solution
Add optional full source provenance to `PropertyError`, populate it for nested guarded declarations, and identify duplicates from the second `(id, source)` pair. Keep the field absent for legacy validation paths so their canonical encodings remain stable. Regression tests use distinct coordinates for malformed, duplicate, and reference failures.

## Prevention
Whenever a checked nested authoring node has its own source coordinate, test two failures with the same path and different line or column values, and assert both the offending declaration and related IDs in the canonical diagnostic.
