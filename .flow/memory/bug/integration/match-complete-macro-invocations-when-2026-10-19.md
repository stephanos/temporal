---
title: Match complete macro invocations when selecting machine outcomes
date: "2026-10-19"
track: bug
category: integration
module: model/umpire/dsl/macros/src/machine.rs
tags: [rust, proc-macros, tokenization]
problem_type: integration
symptoms: A reject field comparison silently changed nonrejecting machine outcomes
root_cause: An identifier and bang token prefix also matched the inequality operator
resolution_type: fix
---

## Problem
The rejection detector matched an identifier followed by bang punctuation. Rust tokenizes `self.reject != 1` with a `!` token, so an ordinary field comparison switched a nonrejecting machine from unit outcomes to typed step outcomes. The formal integration review reproduced the associated-type change.

## What Didn't Work
Recursive token scanning covered literal invocations inside helper macros, but matching only `reject` and `!` also admitted the beginning of `!=`.

## Solution
Commit `e1fae10f8a` requires the full literal invocation shape in `model/umpire/dsl/macros/src/machine.rs`: identifier, bang and a delimited argument group. Recursion still recognizes qualified and nested helper invocations.

## Prevention
The `a_field_comparison_named_reject_keeps_unit_outcomes` control requires `Machine<Outcome = ()>` and checks the emitted state change. It failed with E0271 before the fix and passed afterward. Keep this negative activation control alongside the positive nested-helper control.
