---
title: go mod download inside a target module rewrites its go.sum and bypasses sum chec
date: "2026-09-28"
track: bug
category: integration
module: tools/gomad3/target
tags: [go-modules, gomad3, adapters, clean-checkout]
problem_type: integration
symptoms: missing adapter sum silently accepted; analysis modifies target go.sum
root_cause: "go mod download run with the target module as cwd, before sum validation"
resolution_type: fix
---

## Problem
To make adapter preparation work on a clean module cache, `PrepareTargetBuildAdapters` began downloading each pinned adapter module with `go mod download module@version`. The first version ran that command with the target module as its working directory, and ran it before checking the target's `go.sum`. Inside a module, `go mod download` writes missing sums into that module's `go.sum`. A target missing an adapter's sum was therefore repaired silently and then accepted, and `gomad analyze` modified the checkout it was only supposed to read.

## What Didn't Work
Running the download "in the target so it respects the target's go.mod" felt natural. The unit test for the download passed because it ran in a directory with no `go.mod`, so the go.sum side effect never showed up.

## Solution
Check the sums first with `requireAdapterSums` (tools/gomad3/deterministicio/adapter_registry.go). Then have `target.DownloadModule` run in a fresh empty temp directory, outside every module, and check the reported `Sum` against the pinned sum (tools/gomad3/target/target.go). The regression test `TestPrepareTargetBuildAdaptersRejectsMissingSumBeforeDownloading` confirms that the missing-sum case is still rejected and that `go.mod` and `go.sum` stay byte-identical.

## Prevention
Treat any `go mod download`, `go get`, or `go list -m` call made on behalf of a target as a possible writer of the target's `go.mod`/`go.sum`. Run it outside the module unless the task is to edit it, and keep existing validation ahead of any step that fetches or repairs. Tests for such helpers need a fixture that has a `go.mod`.
