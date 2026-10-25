---
title: Registry admission and opaque metadata must preserve semantics
date: "2026-10-25"
track: bug
category: integration
module: model/umpire/front
tags: [umpire, registry, macros, admission]
problem_type: integration
symptoms: Empty-prefix Reach refused every candidate and metadata tokens changed inferred outcomes
root_cause: Registry dispatch omitted root evidence and the semantic scanner traversed opaque registration tokens
resolution_type: fix
related_to: []
---

## Problem
The synthetic typed registry rejected empty paths before checking authored evidence, so Runtime Reach could not admit a root. Separately, reject!/stay! tokens inside opaque variant metadata affected inferred Model semantics.

## Solution
Use the same typed predicates for root and variant admission; require agreement and preserve refusals/conflicts. Strip only registered variant metadata from a cloned semantic-scan input, while forwarding the original tokens unchanged. Genuine handler macros remain visible.

## Prevention
Exercise successful and conflicting empty-prefix admission through Runtime Reach, retained cursors and malformed evidence. Pair opaque-metadata controls with genuine executable reject!/stay! controls and native/lifted outcome agreement. Both fixes passed the original reviewer's re-review; see fn-246 task11 final-review and review-fixes-5036795dc1 artifacts.
