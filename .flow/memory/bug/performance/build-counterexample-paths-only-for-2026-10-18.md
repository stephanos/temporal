---
title: Build counterexample paths only for failing predicates
date: "2026-10-18"
track: bug
category: performance
module: model2/framework/checker/src/explore.rs
tags: [checker, paths, complexity]
problem_type: performance
symptoms: Holding predicates caused quadratic predecessor traversal on deep graphs
root_cause: Paths were constructed eagerly before the predicate verdict was recorded
resolution_type: fix
---

## Problem
The checker rebuilt a complete predecessor trace after every holding invariant and postcondition. On an N-state chain, successful checks cloned states along all prefixes and performed quadratic work.

## What Didn't Work
Passing an already constructed Path into record made the allocation happen before record knew whether a counterexample was needed.

## Solution
record receives a lazy path builder and invokes it only for Verdict::Fails or a caught predicate panic. Holding predicates and holes retain their verdicts without reconstructing traces.

## Prevention
The deep_chain golden explores 1,001 states with a counting Clone implementation and bounds clone work to four clones per reached state. It reported excessive work before the fix and linear work afterward, without timing thresholds.
