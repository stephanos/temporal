---
title: Preserve generic paths and prost escaping during macro consolidation
date: "2026-10-25"
track: bug
category: build-errors
module: model/umpire/catalog/syntax
tags: [macros, protobuf, parsing]
problem_type: build-error
symptoms: Generic path options panic and keyword or numeric protobuf names emit invalid Rust
root_cause: Reparsing typed paths as expressions and treating case conversion as complete identifier sanitization
resolution_type: fix
related_to: [bug/build-errors/isolate-scala-cli-author-builds-after-2026-10-16, bug/build-errors/track-schema-inputs-before-reusing-2026-09-07]
---

## Problem
Macro consolidation reparsed generic type paths as expressions and applied case conversion without protobuf identifier escaping. Valid generic context options panicked, and keyword or numeric names generated invalid Rust.

## What Didn't Work
A Path token stream is not always valid expression syntax. Heck case conversion alone does not implement prost identifier sanitization.

## Solution
Construct syn::ExprPath directly from the parsed Path. Reuse the catalog-syntax naming helper across inference, descriptor realization and build generation, retaining prost keyword and numeric-prefix escaping after heck conversion.

## Prevention
Keep generic arguments in all scalar Path-option controls. Compile keyword/numeric/package mirrors and compare naming against actual prost-build output. Task4 review/fix evidence records the red-to-green controls.
