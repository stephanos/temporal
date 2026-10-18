---
title: Match enum values rather than references in empty derives
date: "2026-10-18"
track: bug
category: build-errors
module: model2/umpire/dsl/macros/src/action.rs
tags: [rust, proc-macros, empty-enums]
problem_type: build-error
symptoms: Empty Action enums fail generated Clone and Spell with E0004
root_cause: A reference to an empty enum is considered inhabited for exhaustiveness
resolution_type: fix
---

## Problem

An empty enum accepted by `derive(Action)` failed to compile. Generated `Clone` and `Spell` implementations matched `self`, and Rust rejected the empty matches with E0004 because references are considered inhabited.

## What Didn't Work

Populated action fixtures compiled with `match self { ... }` and missed the empty catalog boundary.

## Solution

The two implementations in `model2/umpire/dsl/macros/src/action.rs` match `*self`. Commit 333cc0dd967996dd9065937c7df8016a0baa7a2e reproduces the compiler error; 59b5c5953e4017622910317255cce5d9b54b89e8 fixes it. The `empty_actions` fixture compiles the generated traits and snapshots the empty catalog.

## Prevention

Compile empty and populated enums through the real derive. Expansion snapshots alone do not establish that generated Rust type-checks.
