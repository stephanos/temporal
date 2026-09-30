---
satisfies: [R2]
---
# fn-95-gomad-f1-restore-the-checkout-on.2 Pass make -C tools/gomad3 test on darwin/arm64

## Description
Run the full gate tier by tier (test-harness, test-toolchain, intercept-test, test-host, overlay-test, world-test, test-builder, test-live-capability, test-runtime, test-upstream). Fix darwin regressions introduced by the linux port (platform-pinned fakes, darwin source-set pins, fixtures). Record pre-existing unrelated failures precisely.

## Acceptance
- every tier passes on darwin/arm64, or a failure is recorded with exact output and justification in the milestone status

## Done summary
`make -C tools/gomad3 test` now passes all ten tiers on darwin/arm64. Harness, toolchain, intercept, overlay, world, builder, live-capability and upstream passed unchanged. The host tier needed five repairs, all in commit 45e6788d97:
- Re-pinned the darwin source sets for the libc, memory, x/net and gRPC adapters, and regenerated the darwin v047 and isatty-v021 libc packs. Only the adapter identities changed.
- Split the libc_adapter fixture's fstat call into per-platform files, because darwin's libc has no `Tstat`.
- Fixed the exec-provenance Go version check. Binaries built with `GOEXPERIMENT=nogreenteagc` are stamped `go1.27.1-X:nogreenteagc`, so the check failed on every platform.
- Replaced the runner fake preparer's go1.26.4 darwin pin with the profile's target contract. This also fixes the test that hung for 10 minutes on Linux. It exposed a stale seed-environment order assertion, which is now fixed.
- Re-pinned the boundary test manifest's `os` package fingerprint.

Host note (not a Gomad defect): cgo on this Mac resolves `clang` to mise's lean4 clang, which cannot find `stddef.h`. With that clang, the runtime tier's `clock-cgo-build` fixture fails. The runtime and upstream tiers and the final full run used `PATH=$GOROOT/bin:/usr/bin:$PATH`, which puts Xcode clang first. No code was changed for this.

stage: impl-review - ran (codex 3-draw fan-out, all SHIP, no findings)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 45e6788d97c429f1ab1e94abf54c12085ee32927
- Tests: make -C tools/gomad3 test (darwin/arm64, all ten tiers, suite_rc=0), baseline: red (make -C tools/gomad3 test-host failed pre-edit: runner fake-preparer go1.26.4 pin + timeout, stale darwin adapter source-set pins, exec provenance GOEXPERIMENT version suffix, stale boundary test os fingerprint)
- PRs: