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
Documented the checked Known Gap boundary across the reusable Umpire architecture, model overview,
cross-layer architecture, and local Run Evaluation guide. The docs now distinguish trusted producer
normalization with `KnownGapSet.ofUnordered`, strict external Lean admission with
`KnownGapSet.checkCanonical`, opaque checked semantic consumption and read-only projection at
explicit codec/protocol/semantic-adapter boundaries, and independent Go raw-wire admission plus
response projection/union verification.

All focused and aggregate Lean builds, normalized Go wire tests, the exact regression/generated-view
gate, and model lint passed. `make lint-code` reproduced the approved inherited baseline exactly at
1,379 unrelated findings (errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798,
staticcheck 136, testifylint 4); its sole unrelated `tools/umpire1/monitor_test.go` auto-fix was
restored. The normalized Go gate selected Apple Clang and a physical macOS `TMPDIR` to avoid the
inherited Lean-bundled Clang SDK lookup problem.

Codex review found and verified one documentation correction: `KnownGapSet.toList` is also used at
explicit semantic-adapter boundaries, not only serialization. Its remaining finding concerns a
pre-existing portable-compiler canonicalizer outside this task's declared documentation surface and
has been handed to the parent for spec-completion repair.

stage: impl-review - ran (Codex SHIP after one documentation correction; 0 introduced findings,
1 pre-existing finding handed off) (model: gpt-5.6-sol)
## Evidence
- Commits: ca4dbd909fe7236c6546f1a229c30e7a5971fdf2, 3f91ef3c5ee8b926968fd45105db68c20f2f14ff
- Tests: cd model && mise exec -- lake build Umpire.Planning.Tests.KnownGaps Umpire.Artifact.Tests.Codecs Umpire.Artifact.Tests.Runtime Umpire.Artifact.Tests.Evidence Umpire.Artifact.Tests.Result Temporal.Tool.RunEvaluationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests, CC=/usr/bin/clang TMPDIR=<physical macOS temporary directory> go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/runevaluation, make umpire-check-regression, make lint-model, make lint-code (approved inherited baseline: 1379 findings; errcheck 220, exhaustive 5, forbidigo 211, govet 5, revive 798, staticcheck 136, testifylint 4), git diff --check
- PRs: