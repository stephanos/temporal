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
last_updated: "2026-10-20"
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

## Update 2026-10-20

## Problem
The first typed `path!` implementation passed grammar examples but formal review found expression-order, move and inference defects. Generated expressions reversed waypoint evaluation, evaluated exact repetition counts twice and moved owned payloads once per outcome. Qualified Rust structs, nested closure inference, wildcard field validation and generic/helper declarations also had gaps. The suite integration then exposed missing action-first structural closure inference; eagerly choosing the first actor's defining state repaired that case but rejected formerly compiling manual worlds constrained by concrete waypoints, helper arguments or patterns.

## What Didn't Work
Wrapping user expressions repeatedly in token trees preserved node order while changing evaluation and move semantics. A final projected `Pattern<S, ...>` result did not type nested state closures before field lookup. Discarding `_` fields skipped validation. Eager `RegisteredActor<S>` bounds selected the actor owner before caller constraints, and one global receiver heuristic resolved earlier helper-constrained predicates prematurely when a later bare predicate needed context. Autoref dispatch does not detect whether Rust has already inferred an unconstrained state variable.

## Solution
The macro binds preceding builders before later values. `repeat_exact` takes one evaluated count; outcome alternatives share one Rc-backed action domain. Typed construction context carries an existing state type into nested predicates. Qualified struct expressions are preserved; prefixed Option-domain setters validate every authored field, including wildcards, and retain declaration generics without colliding with finalization.

For leading structures, hidden construction pieces retain `Unknown`, deferred `OwnerHint<Actor>` or `Known` state evidence. Concrete state and authored predicate constraints override owner hints. Only an otherwise context-needing receiver predicate resolves the actor association; neutral predicates preserve pending hints, and helper-constrained predicates establish their state before later bare predicates. Nested grouping preserves this evidence without moving or duplicating authored expressions. Receiver/context detection respects lexical bindings. This is bounded syntax-directed construction, not a universal Rust inference fallback; generic helpers may not independently determine a state type.

## Prevention
Keep `path_expression_semantics.rs` source-order, exact-count, owned-outcome, qualified-state and nested-closure controls. Keep `action_inputs_builders.rs` generic/finish controls and unknown-field refusals. Keep `path_leading_patterns.rs`, manual-world constraints and successive/nested helper-to-bare predicates in `path_language_patterns.rs`, plus actual `find`/`cover` consumers in suite `full_patterns.rs`. Test owner-inferred and caller-constrained routes together, including lexical shadowing, zero-repeat domains, later concrete states and later actors.
