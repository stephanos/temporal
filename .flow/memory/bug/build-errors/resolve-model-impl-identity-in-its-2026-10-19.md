---
title: Resolve Model impl identity in its original lexical and trait context
date: "2026-10-19"
track: bug
category: build-errors
module: model/umpire/dsl/macros/src/machine.rs
tags: [rust, proc-macros, readonly, type-identity]
problem_type: build-error
symptoms: Qualified and local mutable Model impls bypassed the readonly signature rule
root_cause: The macro matched top-level type spelling instead of compiler-resolved identity and lexical scope
resolution_type: fix
related_to: [bug/build-errors/match-enum-values-rather-than-2026-10-18]
---

## Problem

The machine macro accepted mutable Model methods outside on/effect when their impl used a qualified path or appeared in a helper module. Review found the bypass in fn-210.20. Function-local aliases, const-local impls and local trait projections exposed the same gap during the fix.

## What Didn't Work

Matching Model or self::Model at module scope checked spelling rather than Rust type identity. Hoisting checks lost local aliases and projections. Copying a trait impl's constraints into an unrelated generated trait also lost inherited Self::Assoc context and rejected valid helper types.

## Solution

Commit dc56fc7199096944292522ec47137c97723f66f8 adds compiler-resolved readonly declaration checks in model/umpire/dsl/macros/src/machine.rs. The traversal inserts inert const declarations in the original lexical scope and preserves original statements and method/impl headers. Generated guard traits retain the original trait context and generic arguments. The owner explicitly approved lexical declaration insertion. Qualified, nested, aliased, local and projected Model methods refuse while unrelated generic, shadowed and projected helpers compile.

## Prevention

Pair each forbidden Model form with an unrelated-helper positive control. Include function/const-local aliases, trait projections, inherited associated-type bounds and lifetime/type/const generic parameters. Verify generated fixture registries point at the checkout under test when sharing a Cargo target directory. All 41 refusal fixtures and the integrated make model-check passed at 4b7924fb105e15e53b72cd17e0a8ea95153b52c7; the same primary reviewer marked the finding fixed and returned SHIP.
