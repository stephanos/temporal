---
title: Encode fixture folder names before generating Rust identifiers
date: "2026-10-18"
track: bug
category: integration
module: model2/umpire/testing/src/discovery.rs
tags: [fixtures, codegen, identifiers]
problem_type: integration
symptoms: Hyphenated fixture folder emitted invalid Rust
root_cause: Directory names were interpolated into module identifiers without encoding
resolution_type: fix
related_to: [bug/integration/schema-closure-tests-must-compile-2026-10-09]
---

## Problem
Fixture discovery generated Rust module identifiers directly from fixture directory names. A valid folder named `kettle-cut` emitted invalid Rust, while replacing punctuation with underscores would collide with `kettle_cut`.

## What Didn't Work
The initial shared-fixture move replaced numeric module names with literal folder names so `kettle_cut` could import the canonical Kettle fixture. Existing snake-case folders passed, hiding the regression for other accepted names.

## Solution
`model2/umpire/testing/src/discovery.rs` keeps lowercase ASCII letters and digits and encodes every other UTF-8 byte as an underscore plus two hexadecimal digits. Encoding underscores too preserves distinct names, while the `fixture_` prefix admits leading digits and reserved words.

## Prevention
`compiles_distinct_non_identifier_folder_names` generates fixture modules for hyphens, underscores, escape-like names, uppercase, digits and Unicode, then compiles the generated Rust with rustc. It failed on the unencoded implementation and passed after the fix.
