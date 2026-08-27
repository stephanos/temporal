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

### Investigation targets
**Required:** tasks `.1`–`.10`, the root Make Umpire section, and active model/Umpire4 docs.

## Acceptance
- [ ] Root checks fail closed with exact stream/exit behavior and never mutate checked inputs.
- [ ] Facades and docs use Definition ID, Behavior Fingerprint, Artifact Checksum, Limit, Known Gap, Observation Evaluation, Evidence Link, Implementation Link, Run Evaluation, and Result consistently.
- [ ] No pre-v2 support, migration, coverage/replay/verification-receipt family, management platform, CI workflow, model-local Makefile, or Umpire3 path is introduced.

### Quick command

`mise exec -- make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
