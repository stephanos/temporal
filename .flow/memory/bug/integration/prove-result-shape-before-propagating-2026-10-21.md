---
title: Prove result shape before propagating generated-enum coverage
date: "2026-10-21"
track: bug
category: integration
module: model/temporal/annotate/syntax/src/realize.rs
tags: [rust, proc-macros, type-identity, exhaustiveness]
problem_type: integration
symptoms: Generated enum wildcards escaped through method results and rest patterns
root_cause: Receiver shape and written pattern positions were mistaken for proved result type and tuple arity
resolution_type: fix
related_to: [bug/integration/match-complete-macro-invocations-when-2026-10-19, bug/integration/preserve-rust-expression-and-inference-2026-10-20]
---

## Problem
The realization macro's generated-enum wildcard admission inferred unchanged type coverage from method names such as unwrap and map_err. Formal review showed that custom methods, transformed error types and removed wrapper layers could silently hide a generated enum. Rest patterns also skipped tracked payloads or misaligned tuple suffixes.

## What Didn't Work
Propagating receiver coverage through familiar syntax was not proof of result identity. Treating a rest as an uninteresting pattern ignored the omitted fields, and indexing suffixes by their written position lost their actual tuple position.

## Solution
Unproved method and try-expression results are Unknown; explicit cast targets and typed bindings establish coverage through the shared type analysis. Tuple and variant paths retain proved arity, so rest matching checks the omitted interval and correctly aligns suffixes. The implementation remains bounded source analysis and does not rewrite expressions.

## Prevention
Keep focused refusal controls for map_err introducing an enum error, custom unwrap, tuple unwrap, try-wrapper removal, explicit reference casts, Some(..), whole-tuple rests and shifted suffixes. Retain positive scalar casts, non-enum rests and byte-policy fallbacks. The actual grouped RED and ordinary GREEN evidence is recorded in task-1-verification.md and fn2311-formal-r1-*.log.
