---
title: Preserve scalar bound identities when naming shared IR expressions
date: "2026-10-22"
track: bug
category: integration
module: model/umpire/front/src/body/vector.rs
tags: [rust, ir, admission, compatibility]
problem_type: integration
symptoms: Saved vector length guards refused after replacing substitutions with named locals
root_cause: Boolean-only memoized facts discarded scalar length identities required by comparisons
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/default-empty-extensions-must-preserve-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09, bug/integration/match-complete-macro-invocations-when-2026-10-19, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/preserve-rust-expression-and-inference-2026-10-20, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/prove-result-shape-before-propagating-2026-10-21, bug/integration/realized-controls-constrain-admission-2026-09-30]
---

## Problem
Naming shared IR expressions caused a previously admitted bounded vector update to refuse when the authored guard compared a saved length. The first implementation cached Boolean guard results but discarded scalar identity. Review exposed the direct alias; a source-pinned pre-task probe also established acceptance of conditional expressions whose arms carry the same vector length.

## What Didn't Work
Recursively expanding local definitions preserved shape but repeated work exponentially for shared Boolean DAGs. Caching only whether each definition proved an upper bound avoided expansion while losing length identities needed by subsequent comparisons.

## Solution
The vector validator computes abstract facts once in dependency order: unknown, the original vector, its length, finite scalar, or upper-bound guard. Comparisons consume those facts; conditional sequence and length values retain identity only when both arms refer to the original vector. This also preserves length guards after an unrelated conditional leaves that vector unchanged. Sequence checks separately memoize local references and their limited state. See model/umpire/front/src/body/vector.rs.

## Prevention
Keep positive direct, chained and same-origin conditional length controls beside stale-length, cross-field, mixed-arm and unknown-arm refusal controls. Exercise repeated shared Boolean locals at a size that times out with recursive expansion. Run actual required-TLC native/state/verdict agreement for admitted variants, rather than checking admission alone.
