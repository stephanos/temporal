---
satisfies: [R6]
---
# fn-48-canonicalize-known-gaps-as-a-checked-set.5 Document and verify the KnownGapSet boundary

## Description
Update the public model guidance and run the complete Lean/Go compatibility gates (R6).

**Size:** S
**Files:** `model/Umpire/ARCHITECTURE.md`, `model/README.md`, `model/ARCHITECTURE.md`, `tools/umpire/runevaluation/README.md`
**Touches:** [model/Umpire/ARCHITECTURE.md, model/README.md, model/ARCHITECTURE.md, tools/umpire/runevaluation/README.md]

### Approach
- Document producer normalization, strict Lean admission, checked semantic consumption, and independent Go wire admission/verification.
- Preserve existing comments except where raw-list ownership becomes factually stale.
- Run focused suites first, then aggregate model including experimental integration, Go wire tests, exact regression, and lint gates.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/ARCHITECTURE.md:320-330,420-455` — Planning and artifact contracts
- `model/README.md:200-215` — portable Artifact overview
- `model/ARCHITECTURE.md:320-335` — cross-layer Known Gap description
- `tools/umpire/runevaluation/README.md` — Go checker trust boundary
## Acceptance
- [ ] Public docs distinguish the checked Lean semantic collection from strict independent Go wire admission/verification.
- [ ] Existing comments remain present unless their ownership statement changed.
- [ ] Focused and aggregate Lean builds including `TemporalExperimentalTests`, Go artifactv2/runevaluation tests with `-tags test_dep`, exact regression, `make lint-model`, and `make lint-code` pass.
- [ ] No generated fixture, artifact byte, checksum, or fingerprint drift remains.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
