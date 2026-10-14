---
title: Portable carriers must keep the checked namespace and type rules
date: "2026-09-09"
track: bug
category: integration
module: model/Testpilot/Scoped.lean
tags: [umpire, portable, captures, admission, types]
problem_type: integration
symptoms: Two scoped clauses could alias one capture stream; cross-kind comparisons evaluated instead of rejecting
root_cause: A new portable carrier copied the model's shapes without its identifier-namespace and operand-type admission rules
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Extending the portable scoped capability with keyed captures and a correlation reproduced the
checked model's shapes but not its two namespace/type guarantees. `ScopedCaptureRef` names a
retained occurrence by `capture_id` + `ordinal` alone, and the runtime store keyed it the same way,
so two clauses declaring one id silently aliased the same occurrence stream. Comparisons were
evaluated with raw protobuf value equality, so a text field compared against a natural literal
evaluated as an ordinary false instead of rejecting.

## What Didn't Work
Deduplicating capture ids per clause. It mirrored where the declaration lives, not where the
reference resolves: the reference carries no clause identity, so per-clause uniqueness cannot make
it unambiguous.

## Solution
Make the portable admission enforce what the checked path already enforces. Capture identities are
one namespace across the whole capability (`checkProperty` already requires that across a
declaration), and every operand carries the declared scalar kind of the field or capture it reads,
with both sides of a comparison required to agree. A field two projection rules declare at
different kinds has no single type, so reading it rejects. Implemented symmetrically in
`model/Testpilot/Scoped.lean` (`retainedKind`, `operand`) and
`common/testing/testpilot/internal/verification/scoped_prepare.go` (`validOperand`,
`validCorrelation`).

## Prevention
When a portable/offline carrier is added for a checked model construct, enumerate the model's
admission rules for that construct first and write the rejection matrix as tests on both sides
before the happy path. An identifier that resolves without its declaring scope in the wire format
is the tell: its uniqueness rule belongs to the whole artifact, not the declaration site.
