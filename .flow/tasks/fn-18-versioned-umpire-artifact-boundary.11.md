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
Exposed the retained Artifact v2 boundary through the public Lean facade and thin read-only root checks for one document or one complete set. Both commands fail closed with exact exit/stream behavior, reject noncanonical bytes and unsafe filesystem inputs without mutation, and never publish as a side effect; explicit atomic PublishSet/LoadSet remain the only persistence API.

Synchronized the active model and Umpire4 persistence documentation, fn-37 supersession notes, and only fn-19's stale executable input contract around the sole deterministic two-space pretty v2 representation, exact checksum preimages, retained families, closure stages, Limits/Known Gaps, atomic visibility, and deferred boundaries. The review fix bounds set traversal before reads, preserves unsupported-format precedence, and removes contradictory duplicate Artifact API prose.

stage: impl-review - ran (model: gpt-5.6-sol)
## Evidence
- Commits: 538f5024a34d447dd41c8d86725f7780721f72ea, 73e349d716150b7d097f36f517c3293fa89fde4a
- Tests: TDD RED: command package absent; focused check/check-set tests failed to compile until exit and run surfaces were implemented, review RED: stale-manifest plus v3 member returned artifact-checksum; oversized unexpected file was read before rejection; both focused regressions GREEN after 73e349d71, mise exec -- make umpire-check-artifact FAMILY=umpire-experiment/v2 ARTIFACT=tools/umpire/artifact/testdata/switch-experiment-v2.json (pass, silent), mise exec -- make umpire-check-artifact-set SET=tools/umpire/artifact/testdata/valid-run-evaluation-set (pass, silent), exact command stream/exit/mutation suite for all six families, compact rejection, unsupported precedence, unexpected/oversized/symlink inputs, usage, success, and no publication (pass), mise exec -- go test -count=1 ./tools/common/artifactio ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-artifact (pass), mise exec -- go vet ./tools/common/artifactio ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-artifact (pass), mise exec -- go test -count=1 -race ./tools/common/artifactio ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/umpire/cmd/umpire-artifact (pass), scoped golangci-lint from 19ad3f1c4 over artifactio/artifact/artifactv2/command (0 issues), cd model && mise exec -- lake build Umpire.Artifact UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect umpire-gen-tests (pass, 229 jobs), mise exec -- make umpire-build-model (pass, 182 jobs), mise exec -- make lint-model (pass), mise exec -- make umpire-check-regression (pass: generated views, active vocabulary, 226-job Lean build), checked-in set member cmp parity, exact one terminal LF, gofmt, and git diff --check (pass), diagnostic-only broad ./tools/umpire/... probe: unrelated dynamic-config root scan fails on inherited case-insensitive duplicate Tools/tools and Temporal/temporal aliases at identical inodes; task diff touches neither subsystem and all declared/relevant Go packages pass serially, flowctl codex impl-review fn-18-versioned-umpire-artifact-boundary.11 --base 19ad3f1c4eeef393b5aa8dc5ae72bd7a56b68ef8 --receipt /tmp/impl-review-receipt-fn-18-versioned-umpire-artifact-boundary.11.json (SHIP; all 3 prior findings fixed; introduced=0; unaddressed R-IDs=[]; R8 met)
- PRs: