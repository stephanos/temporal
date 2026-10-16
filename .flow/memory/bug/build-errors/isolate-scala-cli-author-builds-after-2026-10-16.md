---
title: Isolate Scala CLI author builds after local macro jar changes
date: "2026-10-16"
track: bug
category: build-errors
module: model/framework/README.md
tags: [scala-cli, macros, cache]
problem_type: build-error
symptoms: Newly packaged public author jar calls an obsolete macro-generated method signature
root_cause: Same-path dependency jar replacement reused cached author macro expansions
resolution_type: fix
---

## Problem
A fresh Scala CLI package command rebuilt the native jar but reused author bytecode containing an old StateField.make macro expansion. The public client failed with NoSuchMethodError despite native and author package commands both returning zero.

## What Didn't Work
Overwriting a dependency jar at the same local path and using package -f did not establish fresh author compilation. javap showed an eight-argument invocation in the author jar against a new nine-argument native method.

## Solution
Use the supported --workspace option with a newly allocated directory for each complete separate-library recipe, applying it to native, author, runtime and refusal stages. Preserve the failed attempt and rerun the entire recipe. The refreshed author bytecode used the new signature and all five native public-client families plus channel passed. The framework README records this freshness requirement.

## Prevention
After changing inline or macro implementation dependencies, isolate the compiler workspace for separate-library verification. A package success or newly written jar alone is insufficient freshness evidence. Do not add compatibility overloads to hide stale expansions.
