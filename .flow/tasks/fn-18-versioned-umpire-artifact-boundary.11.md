---
satisfies: [R8]
---
# fn-18-versioned-umpire-artifact-boundary.11 Expose Artifact checks and synchronize persistence documentation

## Description
Expose the retained admission and set-check surfaces and document the v2-only transport boundary.


**Size:** M
**Files:** `Makefile`, Artifact facades/commands, active model docs, and `.plans/UMPIRE4_*.md`
**Touches:** [Makefile, model/Umpire/Artifact.lean, tools/umpire/cmd/umpire-artifact/**, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_*.md]

### Approach
- Wire vertical Lean modules through the public facade with comments preserved and no aliases.
- Add thin check commands for one Artifact and one complete set; publication remains an explicit API, not an implicit check side effect.
- Document the exact v2 baseline, retained families, canonical bytes, checksums/fingerprints, Limits, Known Gaps, atomic visibility, and deferred boundaries.
- Reconcile active Umpire4 order/current-contract prose with the authorized pre-release pretty-v2
  correction and update fn-19's stale input contract from `umpire-experiment/v3` to the sole
  supported `umpire-experiment/v2` before fn-19 can start; do not change any other fn-19 behavior.

### Investigation targets
**Required:** tasks `.1`–`.10`, the root Make Umpire section, and active model/Umpire4 docs.

## Acceptance
- [ ] Root checks fail closed with exact stream/exit behavior and never mutate checked inputs.
- [ ] Facades and docs use Definition ID, Behavior Fingerprint, Artifact Checksum, Limit, Known Gap, Observation Evaluation, Evidence Link, Implementation Link, Run Evaluation, and Result consistently.
- [ ] No pre-v2 support, migration, coverage/replay/verification-receipt family, management platform, CI workflow, model-local Makefile, or Umpire3 path is introduced.
- [ ] `.plans/UMPIRE4_ORDER.md`, fn-37 supersession notes, and fn-19 agree that current executable
  input is deterministic-pretty `umpire-experiment/v2` with the fn-18 checksum formula.

### Quick command

`mise exec -- make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
