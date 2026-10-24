---
title: Facade admission must normalize DSL prefixes through grouped imports
date: "2026-10-24"
track: bug
category: integration
module: model/umpire/author/dsl/macros/src/roots.rs
tags: [rust, facade, macro, admission]
problem_type: integration
symptoms: Grouped facade imports compiled natively but refused embedded model admission
root_cause: Only an immediate facade-to-dsl UseTree::Path was normalized; grouped prefix branches were missed
resolution_type: fix
related_to: [bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09, bug/integration/match-complete-macro-invocations-when-2026-10-19, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/preserve-arithmetic-evaluation-roots-2026-10-23, bug/integration/preserve-rust-expression-and-inference-2026-10-20, bug/integration/preserve-scalar-bound-identities-when-2026-10-22, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/prove-result-shape-before-propagating-2026-10-21, bug/integration/realized-controls-constrain-admission-2026-09-30]
---

## Problem
A Model importing DSL names with `use author_kit::{dsl::{Action, State, check}}` compiled natively, but the facade adapter left the grouped `dsl` prefix in its cloned admission AST. The embedded model refused `umpire_dsl::dsl::check` as a possible macro rebinding and omitted interpreter support.

## What Didn't Work
The initial adapter removed the DSL segment only when it immediately followed the resolved facade root as a path node. Top-level facade imports and flat `author_kit::dsl::{…}` imports passed, so those tests missed nested use groups.

## Solution
Admission::facade_use_tree in model/umpire/author/dsl/macros/src/roots.rs recursively visits groups at the facade prefix and removes their immediate DSL segment. It leaves unrelated nested prefixes and direct-DSL imports intact. The facade-only consumer now uses a mixed group and checks successful embedded admission; a table covers flat, grouped, nested and preserved-prefix forms.

## Prevention
Exercise grouped imports in the real single-dependency author consumer and assert embedded admission alongside native compilation. Preserve the red-before-fix diagnostic and direct-DSL negative normalization control when changing facade path adaptation.
