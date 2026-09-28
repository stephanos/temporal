---
title: go/build cross-platform MatchFile keeps host arch feature ToolTags
date: "2026-09-28"
track: bug
category: runtime-errors
module: tools/gomad3/qualification/set/manifestgen
tags: [go-build, build-tags, gomad3, qualification]
problem_type: runtime-error
symptoms: feature-tagged test files silently omitted when listing tests for another GOARCH
root_cause: build.Default.ToolTags carries host arch feature tags; overriding GOOS/GOARCH does not reset them
resolution_type: fix
---

## Problem
Listing a package's tests for another platform by copying `go/build.Default` and overriding `GOOS`/`GOARCH` kept the host's architecture feature tags in `ToolTags` (e.g. `arm64.v8.0` on darwin/arm64). A test file constrained on `amd64.v1` was then excluded on both platforms, so the cross-platform consistency check and the staleness check both passed while the test was silently missing.

## What Didn't Work
Setting only `GOOS`/`GOARCH` on a copy of `build.Default`; the `go test -list` parity fixture had no feature-tagged file, so it could not catch this.

## Solution
In `tools/gomad3/qualification/set/manifestgen/manifestgen.go` `listPlatformTests`, drop `build.Default.ToolTags` entries prefixed with the host GOARCH and append the target's baseline feature tag (`amd64.v1`, `arm64.v8.0`); refuse unknown architectures. Also the reviewer flagged that the set validator's `^Test[A-Za-z0-9_]+$` rejected valid Go tests (`Test`, `TestÉclair`); it now admits `^Test[\p{L}\p{Nd}_]*$` and workload IDs encode non-ASCII runes as `u%04x`.

## Prevention
Any cross-platform `go/build.Context` must reset `ToolTags`, not just GOOS/GOARCH. Keep a fixture file with an `//go:build amd64.v1` constraint in platform-enumeration tests.
