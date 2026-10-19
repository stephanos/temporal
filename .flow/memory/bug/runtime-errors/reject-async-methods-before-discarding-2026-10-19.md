---
title: Reject async methods before discarding generated call results
date: "2026-10-19"
track: bug
category: runtime-errors
module: model/umpire/dsl/macros/src/machine.rs
tags: [rust, proc-macros, async, handlers]
problem_type: runtime-error
symptoms: Async handler bodies never ran and actions silently appeared disabled
root_cause: "ReturnType::Default also accepts async fn, whose invocation returns a future"
resolution_type: fix
---

## Problem
The changed-state dispatch called a registered handler and discarded its result. An `async fn` with no written return type passed registration, but dispatch dropped its future without executing the body. The action appeared disabled and its postconditions never ran.

## What Didn't Work
Checking only `syn::ReturnType::Default` did not establish a synchronous unit-returning method. Rust emitted an unused-future warning, so ordinary compilation still succeeded. The old Option-returning dispatch had rejected this case through a type mismatch.

## Solution
Registration rejects `method.sig.asyncness` before checking the return type in `model/umpire/dsl/macros/src/machine.rs`. The `machine_handler_async` trybuild fixture first compiled successfully against the defective producer, then pinned the synchronous-handler refusal after commit e007ac10dab8040a922f1e1111c040e8e4361f3d. Focused tests, the full native/wasm/Go gate and the resumed primary review passed.

## Prevention
Keep the async compile-fail control whenever changing generated invocation or signature validation. Apply the same synchronous-method check to unit-returning claim methods, because discarding an async claim future could falsely verify it. An omitted syntactic return type alone does not establish the invocation's result type.
