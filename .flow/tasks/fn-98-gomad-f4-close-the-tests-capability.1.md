---
satisfies: [R2, R3]
---
# fn-98-gomad-f4-close-the-tests-capability.1 Regenerate the darwin compatibility packs and confirm darwin adapter source-set pins

## Description
Run discover/review/generate for the darwin packs on this host against the current profile digest; confirm or correct the fx, SDK, otel adapter darwin prepared source-set pins.

## Acceptance
- `make validate` and `compatibility-pack-qualification` pass on darwin

## Done summary
Observed on darwin/arm64 that every compatibility pack is current against the profile digest and that the fx, SDK, and otel darwin prepared source-set pins match what the darwin capability review computes (mutation probe per pin); nothing needed correcting. `make -C tools/gomad3 validate compatibility-pack-qualification` passes (six requests), and the observation is recorded in `MILESTONES.md` F4 status.

baseline: green (make -C tools/gomad3 validate compatibility-pack-qualification)

stage: impl-review - ran [codex fan-out NEEDS_WORK (unrecorded observation) .. re-review SHIP]

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4e2c5659e69562834eb8b545db1d08234482ce78, cc18bba34cc2dcf0c9bb5021e8642cdd419e143c
- Tests: make -C tools/gomad3 validate compatibility-pack-qualification (baseline green, verify green; darwin/arm64), go test -tags test_dep -count=1 -run TestRewrittenModule ./deterministicio (darwin/arm64, pass), mutation probe: each darwin pin zeroed in turn -> TestRewrittenModulePreparedPackageSourceSetIdentity reported fx d8b6580641c5..., sdk/internal 45cd84114a3b..., otel/sdk/resource 796855abd6e0..., equal to committed pins; files restored, gomad doctor: fx, temporal sdk, otel sdk adapters ok
- PRs: