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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
