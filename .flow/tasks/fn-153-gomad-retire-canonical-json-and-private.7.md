---
satisfies: [R1, R2]
---
# fn-153-gomad-retire-canonical-json-and-private.7 Migrate qualification evidence and shared report publication

## Description
Implements R1/R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** qualification/qualification.go, analysis/analysis.go, comparison/comparison.go, soak/{ledger,soak}.go and focused tests (4 JSON callers plus 2 private publishers).
**Touches:** [tools/gomad3/qualification/qualification*.go, tools/gomad3/qualification/*_test.go, tools/gomad3/qualification/analysis/**, tools/gomad3/qualification/comparison/**, tools/gomad3/qualification/soak/ledger*.go, tools/gomad3/qualification/soak/soak*.go]

### Approach

- Replace ordinary report/evidence/cohort encodes and strict parses, keeping complete identities, clone behavior, status/classification and bounded evidence handling.
- Move WriteQualificationReport staging/rename/directory durability through the shared handle; keep private 0700 root, 0600 file, temp-derived unique name and nonempty returned path after publication errors.
- Replace soak's private whole-file writer with shared replacement; retain filenames, indentation/final newline, requested modes and ledger semantics while adopting shared sync/cleanup behavior.
- Behavior pin: frozen semantic report/evidence/cohort comparisons and identity-input mutations; exact report-path/classification table across pre/post-rename failures, close/cleanup joins, cancellation and permission failures.
- Migrate diagnostics_test.go and other surviving root qualification test dependencies here; their diagnostic and classification checks remain owned by this report/evidence task.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/qualification/qualification.go:224`
- `tools/gomad3/qualification/qualification_test.go:184`
- `tools/gomad3/qualification/analysis/analysis.go:112`
- `tools/gomad3/qualification/comparison/comparison.go:240`
- `tools/gomad3/qualification/soak/ledger.go:138`
- `tools/gomad3/qualification/soak/soak.go:711`
- `tools/gomad3/qualification/soak/soak_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./qualification ./qualification/analysis ./qualification/comparison ./qualification/soak

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] Qualification and soak retain no private atomic publisher; private report modes/names and post-publication path/error classification match the frozen behavior pin.
- [ ] Report/evidence/cohort identities bind complete inputs, semantic clones preserve nil/pointer ownership and report corruption/staleness remains detectable.
- [ ] Focused report/ledger parsing and publication negatives cover malformed/unknown/duplicate/trailing input, invalid strings, bounds, primary/cleanup error precedence and published-but-not-durable outcomes.
## Done summary
Migrated qualification, native capability-analysis/comparison, and soak evidence to complete deterministic stdlib identities with strict typed decoding, bounded reads, detached nested evidence, and invalid-string refusal. Routed qualification reports through disposition-aware hostfs staging and soak JSON through shared durable replacement without changing public filenames, modes, or formatting. Nil-preserving blocker clones now keep returned evidence reproducible from its recorded identity. Focused, race, hostfs/world, and architecture gates pass; the implementation re-review is SHIP. The full native analysis path remains host-inconclusive on linux/arm64, and aggregate validation reaches only fn153.17's known stale compatibility approval.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1d24472a13020a7238919b7ebb1761c92d690059, 8743c3ad00a8a2af2ef44ef658bab256bf46770c, 24017f0edc81184582693daa348d07dae3322d96
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./qualification ./backends/native/qualification/comparison ./backends/native/qualification/soak: PASS, go -C tools/gomad3 test -tags test_dep -count=1 -run '^TestBuildRejectsInvalidOriginalIdentityStrings$' ./backends/native/qualification/analysis: PASS, go -C tools/gomad3 test -tags test_dep -race -count=1 ./qualification ./backends/native/qualification/comparison ./backends/native/qualification/soak: PASS, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./world: PASS, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck: PASS, make -C tools/gomad3 validate: EXPECTED FN153.17-OWNED DRIFT (compatibility-pack request golang-x-sys-v047-darwin-arm64 approval is stale), full native analysis suite: INCONCLUSIVE on linux/arm64 because the deterministic native profile/toolchain supports darwin/arm64 and linux/amd64, flowctl codex impl-review re-review: SHIP, prior blocker nil/empty identity finding fixed and no new findings
- PRs:
## Revival trigger (2026-10-13)

Revived after fn153.1, fn153.2 and fn153.3 completed with SHIP implementation reviews. Qualification report and soak publication now consume the integrated hostfs staging and record identity seams; general qualification ownership is disjoint from the concurrent compatibility and Runner migrations. The forced local claim is required only because Flow chain admission still sees the completed fn152 parent on an unpushed remote branch; pushing remains outside the requested scope.
