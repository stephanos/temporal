---
title: CI job timeout must exceed a gate's Go -timeout or output is lost
date: "2026-09-26"
track: bug
category: build-errors
module: .github/workflows/umpire.yml
tags: [ci, timeout, live-tests]
problem_type: build-error
symptoms: CI job killed before go test timeout fires; no live output in log
root_cause: job timeout-minutes shorter than go test -timeout on a buffered gate
resolution_type: fix
---

## Problem
Adding an explicit `-timeout 30m` to the live Testpilot gate's `go test` left the CI job that runs it (umpire.yml `portability`) at `timeout-minutes: 15`. GitHub would kill the job before Go's timeout fired, and since the gate buffers live output to a temp file and prints it only after `go test` exits, a hang would leave no output in the CI log.

## Solution
Raised the job limit to 40 minutes and pinned it in `tools/umpire/regression/ci_workflow_test.go` with a comment tying it to the Go timeout.

## Prevention
When changing a Go test `-timeout` in a Makefile gate, check every CI job that runs that target: the job's `timeout-minutes` must exceed the Go timeout, or the diagnostic dump is lost.
