---
title: Track schema inputs before reusing generated Lean modules
date: "2026-09-07"
track: bug
category: build-errors
module: model/lakefile.lean
tags: [lean, protobuf, lake, codegen]
problem_type: build-error
symptoms: A protobuf schema changed while Lake reused stale generated Lean declarations
root_cause: The external schema closure was not part of a traced Lake dependency that invalidated the generated protocol module
resolution_type: fix
---

## Problem

The generated Testpilot Lean protocol was elaborated directly from its protobuf closure, but Lake could reuse an existing `Protocol.olean` after a schema changed because external schema files were absent from the module trace.

## What Didn't Work

Adding bare `input_file` targets as library dependencies waited for those jobs without propagating their traces into the module artifact. Lean `include_str` likewise reads a file during elaboration without declaring a Lake dependency.

## Solution

`model/lakefile.lean` collects typed input jobs for the exact eight-schema closure into a traced stamp target. When that target changes, it invalidates only `Testpilot/Protocol.olean`, forcing the existing `#load_proto_file` elaboration to regenerate declarations. A real schema mutation proved that both the stamp and protocol module rebuild.

## Prevention

Keep a mutation test that changes one closure schema and requires the subsequent Lake build output to include `Built Testpilot.Protocol`; a warm unchanged build should leave the module cached.
