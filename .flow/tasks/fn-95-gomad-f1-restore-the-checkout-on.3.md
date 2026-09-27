---
satisfies: [R3, R4]
---
# fn-95-gomad-f1-restore-the-checkout-on.3 Pass gomad3-integration-test and the core qualification set on darwin/arm64

## Description
Run `make gomad3-integration-test` and `make -C tools/gomad3 compatibility-pack-qualification core-qualification-set`; verify 5/5 supported with exact choice replay.

## Acceptance
- integration test passes
- core set: selected 5, supported 5, unsupported 0, all choice_replay_exact

## Done summary
`make gomad3-integration-test` passes on darwin/arm64 with both tests running. The core qualification set also passes: selected 5, supported 5, unsupported 0, failed 0, infrastructure errors 0, and all five workloads have `choice_replay_exact`: concurrency-state-invariant, filesystem-transaction, loopback-tcp-roundtrip, modernc-libc-boundary and sqlite-transaction.

`compatibility-pack-qualification` failed at first on darwin. The darwin request list names `internal/compatibilitypack/testdata/v041`, but that fixture was never committed, so it was missing. Two repairs, both in 7ea97c052e:
- Authored the fixture as module `gomad3.compatibility.v041`, which requires modernc libc v1.72.3 and resolves to x/sys v0.41.0. Its test is `TestLibcCompatibilityClosure`, a libc file round trip.
- Rediscovered, reviewed and regenerated the `modernc-libc-xsys-v041` pack. Its libc and memory adapter identities were stale, the same drift task .2 fixed for v047. Allowed capabilities, linkname directives and platform scope did not change. New review sha is `sha256:e93c386eae937f3b91cf28da8549527db8cbc724ec96061e99ab2dd7f7e34eed`.

All four darwin packs now qualify: reflect2-go126, v041, v047 and v047-isatty-v021.

For task .4: `.plans/GOMAD_MILESTONES.md` says "Nothing references a `compatibilitypack/testdata/v041` fixture any more". That is wrong on darwin, because the Makefile's non-linux branch still qualifies it. The fixture now exists.

stage: impl-review - ran (codex 3-draw fan-out, all SHIP, no findings)
## Evidence
- Commits: 7ea97c052e34efe15109890968c7940ef421d1c5
- Tests: baseline: green via handoff (verified at 45e6788d by fn-95-gomad-f1-restore-the-checkout-on.2), make gomad3-integration-test (pass at HEAD; 2 tests, 0 skips), make -C tools/gomad3 compatibility-pack-qualification core-qualification-set (pass: 4/4 darwin packs qualified; core set selected=5 supported=5 unsupported=0 failed=0 infrastructure-errors=0, choice_replay_exact=true for all 5), make -C tools/gomad3 validate (compatibility packs are current), go test -tags test_dep ./internal/compatibilitypack/... ./cmd/gomadtool/... (tools/gomad3, pass)
- PRs: