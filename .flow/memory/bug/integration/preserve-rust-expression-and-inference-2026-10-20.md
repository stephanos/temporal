---
title: Preserve Rust expression and inference contracts in path macros
date: "2026-10-20"
track: bug
category: integration
module: model/umpire/dsl/macros/src/path.rs
tags: [rust, macros, type-inference, expression-semantics]
problem_type: integration
symptoms: "Valid composed paths moved owned payloads repeatedly, reversed evaluation or lost closure inference"
root_cause: Generated token-tree nesting duplicated expressions and deferred state typing; wildcard fields skipped typed validation
resolution_type: fix
related_to: [bug/integration/match-complete-macro-invocations-when-2026-10-19]
---

## Problem
The first typed `path!` implementation passed its grammar examples but formal review found eight interaction defects. Generated expressions reversed waypoint evaluation, evaluated exact repetition counts twice and moved owned payloads once per outcome. Qualified Rust structs, nested closure inference, wildcard field validation and generic/helper declarations also had gaps.

## What Didn't Work
Wrapping user expressions repeatedly in generated token trees preserved node order while changing Rust evaluation and move semantics. Relying on a final projected `Pattern<S, ...>` result did not give nested state closures a concrete parameter type before field lookup. Discarding `_` fields skipped typechecking their authored names.

## Solution
The path macro binds preceding builders before later values. `repeat_exact` takes one evaluated count; `with_outcomes` shares a single Rc-backed action domain. A typed construction context carries the existing builder's state type into nested predicates. The parser preserves ordinary qualified struct expressions and calls a prefixed Option-domain setter for every authored input, including wildcards. Action builders retain declaration generics and keep field setters separate from finalization.

## Prevention
Keep `path_expression_semantics.rs` controls for source-order evaluation, exact-count evaluation, owned outcome alternatives, qualified shorthand/update/parenthesized state expressions and nested closures. Keep `action_inputs_builders.rs` controls for const generics and the `finish` field, plus `path_language_unknown_fields.rs` refusals. Grammar tests should combine legal constructs with owned values and side-effecting Rust expressions.
