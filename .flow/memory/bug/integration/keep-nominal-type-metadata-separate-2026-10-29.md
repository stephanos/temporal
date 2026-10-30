---
title: Keep nominal type metadata separate from emitted identifiers
date: "2026-10-29"
track: bug
category: integration
module: model/umpire/backends/tla/src/values.rs
tags: [tla, nominal-types, schema, frontend]
problem_type: integration
symptoms: Imported bounded types passed native and interpreted checks but TLA schema refused their qualified identities
root_cause: Bounded nominal metadata reused emitted-language identifier validation and lacked a bound Model schema regression
resolution_type: fix
related_to: [bug/integration/retired-protobuf-names-must-guard-every-2026-10-09]
---

## Problem
Imported const-generic bounded types passed native and interpreted checks but failed the TLA schema before model checking. Their qualified nominal identities were validated as emitted TLA identifiers.

## What Didn't Work
Admission tests and a small native/interpreter comparison did not exercise the bound Model's backend schema. Deferring the whole countdown fixture to the next task left this integration boundary unchecked.

## Solution
`model/umpire/backends/tla/src/values.rs::descriptor_names` now accepts qualified nominal type paths as metadata while retaining old simple names and strict emitted-identifier checks. The consolidated DSL countdown regression binds a real World, emits the admitted source and validates its schema.

## Prevention
When extending nominal type identities, check native export/decode and the actual bound emission/schema path before claiming backend compatibility. Keep full exploration and mutants separate when planned, but verify that the next checker can consume the produced IR.
