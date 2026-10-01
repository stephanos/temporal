---
title: Profile adapter changes leave libc-bound compatibility packs stale
date: "2026-10-01"
track: bug
category: integration
module: tools/gomad3/internal/compatibilitypack
tags: [gomad3, compatibility-pack, profile-digest, adapters]
problem_type: integration
symptoms: every ./tests target analyzes as unsupported with forbidden_import x/sys and unapproved_linkname blockers while make validate is green
root_cause: packs bind the deterministic profile implementation digest exactly and were not rediscovered after adapters were registered
resolution_type: fix
related_to: [bug/integration/go-mod-download-inside-a-target-module-2026-09-28, bug/integration/shard-merge-and-prepared-target-cache-2026-09-29]
---

## Problem
Registering adapters in `tools/gomad3/deterministicio/profile.go` changes the deterministic profile implementation digest on both platforms. The libc-bound compatibility packs (`modernc-libc-xsys-*`) carry that digest in every adapter binding and selection requires equality (`internal/compatibilitypack/v2_selection.go` `matchesPackAdapter`), so stale packs are silently not selected and every `./tests` target analyzes as unsupported. `gomadtool compatibility-pack check` only compares generated artifacts with their requests, so `make validate` stayed green.

## What Didn't Work
The first version of the detection helper collected bindings in a map keyed by activation path / rule import path; the schema allows repeated keys under different module identities, so a current entry could overwrite a stale one and the check would pass.

## Solution
Rediscover each host-platform request with the existing request as draft (`discover` -> `review` -> `generate --approve-review=<digest>` -> `make -C tools/gomad3 validate compatibility-pack-qualification`), confirm with a structural JSON diff that only `profile_implementation_sha256` changed, then rebuild `.bin/gomad` because packs are embedded. `internal/compatibilitypack/profile_binding_test.go` (external test package, wired into `validate-compatibility`) iterates activations and rules directly and fails on any host-platform binding that differs from `deterministicio.Default().Identity()`.

## Prevention
A commit that changes `deterministicAdapters` or `deterministicImplementationVersion` must rediscover the libc-bound packs on darwin/arm64 and on linux/amd64 in the same change; discovery reviews the host platform only, so each platform needs its own host. Production code under `internal/compatibilitypack` and `cmd/gomadtool` may not import `deterministicio` (architecture test), which is why the check is a test.
