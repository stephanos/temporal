---
title: Preserve generic declaration context in generated adapters
date: "2026-10-21"
track: bug
category: build-errors
module: model/umpire/dsl/macros/src/machine/declared.rs
tags: [rust, proc-macros, generics]
problem_type: build-error
symptoms: Valid generic Machines failed in generated carrier and readonly declarations
root_cause: "Copied generic bounds, defaults and Self retained tokens but changed declaration context"
resolution_type: fix
related_to: [bug/build-errors/match-enum-values-rather-than-2026-10-18, bug/build-errors/resolve-model-impl-identity-in-its-2026-10-19]
---

## Problem

Valid native generic machines failed compilation during fn-227.1 implementation review. The generated action carrier assumed Actor supplied Debug, merged actor defaults into an illegal parameter order, and copied declaration Self bounds into different types. Readonly checks also copied parent-local Model bounds into child scopes.

## What Didn't Work

Passing generic derive-only tests did not prove generated Machine and helper declarations were well formed. Copying complete Generics values preserved tokens while changing their type or lexical context. Actor supplies Clone, Eq, Hash and Display, but it does not supply Debug.

## Solution

Commit f0adb5afdb659614cf418a7f420106a85718ba06 gives the action carrier separate conditional Debug bounds, removes defaults from synthesized combined parameters and function/impl contexts, and resolves Self to the originating type before merging bounds. Declared-world readonly probes specialize at the Model lexical root; guards remain at each original declaration and reference that probe. The absent-World branch retains its old expansion bytes and diagnostics.

## Prevention

Compile and execute real generic Machine controls with payloads absent from state fields, defaulted actors using different Model parameter subsets, disjoint Model/World Self-bound settings and parent-local bounds with child mutable helpers. Pair the helper positive with a forbidden child mutable-Model control. generic_bounds and generic_readonly cover these cases. All four formal findings were fixed in the resumed gpt-6.1-sol/high review, which returned SHIP; its Cargo rebuild was read-only blocked, so worker-run Quick/full gates remain the execution evidence.
