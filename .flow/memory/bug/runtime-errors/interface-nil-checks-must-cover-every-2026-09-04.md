---
title: Interface nil checks must cover every nil-capable kind
date: "2026-09-04"
track: bug
category: runtime-errors
module: tools/umpire/temporal/nexus/runner.go
tags: [umpire, go, interfaces]
problem_type: runtime-error
symptoms: A typed-nil EnvironmentFactory passed constructor validation and could fail later at runtime
root_cause: The reflection guard checked only pointer kinds
resolution_type: fix
---

## Problem
A constructor accepted typed-nil implementations of `EnvironmentFactory` when the dynamic value used a nil-capable kind other than a pointer. That deferred a deterministic construction failure until runtime.

## What Didn't Work
Checking only `reflect.Pointer` handled the common typed-nil pointer case but assumed every interface implementation had pointer representation.

## Solution
`isNilEnvironmentFactory` now calls `IsNil` for every nil-capable reflection kind, and the constructor test covers pointer, map, slice, function, and channel implementations in `tools/umpire/temporal/nexus/runner_test.go`.

## Prevention
When validating interface values against typed nil, enumerate every nil-capable reflection kind and use a table-driven test with representative implementations.
