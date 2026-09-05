---
title: Validate protobuf descriptor structure before accepting same-name responses
date: "2026-09-05"
track: bug
category: integration
module: tools/umpire/internal/ir/runtime_value.go
tags: [umpire, protobuf, types]
problem_type: integration
symptoms: Same-name divergent Host response descriptors were silently reinterpreted
root_cause: "Message full name identifies a symbol, not its structural schema; repeated fields also have no scalar default"
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04]
---

## Problem
Implementation review found that SnapshotMessage accepted a protobuf response by full message name alone. An independently supplied descriptor can reuse that name with wire-compatible but semantically different fields, allowing the pinned descriptor to reinterpret a crossed response.

## What Didn't Work
Matching FullName and then serializing/decoding into the pinned descriptor validated wire data but did not validate the source schema. Comparing descriptor pointers alone would reject legitimate independently built or generated equivalents.

## Solution
The IR descriptor compatibility helper uses an exact-identity fast path and a bounded structural comparison of fields, oneofs, nested messages, enums, defaults and options. A visited descriptor-pair set terminates recursive schemas. Descriptor traversal and comparisons charge the same runtime work budget used by SnapshotMessage.

## Prevention
Test same-name descriptors with changed field kinds and nested definitions, alongside independently reconstructed recursive descriptors and generated equivalent messages. Exercise exact accepted work and one-less rejection so schema validation cannot bypass execution bounds.

Repeated scalar fields have no protobuf default Value. Skip their default inspection; converting an invalid repeated-bytes default to Bytes panics. Empty and populated repeated-bytes equivalent-message regressions cover this structural-validation edge.
