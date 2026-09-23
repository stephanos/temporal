---
satisfies: [R2, R3, R4]
---

# fn-26-local-qualification-receipts-and-staged.2 Admit exact Case Runtime qualification subjects and load the Profiles

## Description
Add `tools/umpire/evaluation/admission.go`: `Admit(caseBytes, recordedRun []byte) (*Subject, error)` over the canonical Case (`casefile.Canonical`, then `testpilot.DecodeCaseProtoJSON`) and fn-22's recorded Run (`replay.DecodeRecordedRun`), rejecting by reason class before any assessment: `noncanonical`, `malformed`, `incompatible` (a Case whose format version is not 1.0), `oversized` (a Case over 4 MiB, a recorded Run over 16 MiB or 65,536 events, named constants), `crossed` (the Run's `case_id` or `program_id` is not the Case's; the Verdict's rule IDs are not exactly the Contract's `rules` and `correlated.rules` together, each once), `open` (no terminal disposition, no cleanup outcome, no Verdict), `inconsistent` (a supporting sequence naming no event or an event twice; a Verdict status that disagrees with its rules or the disposition: violated when any rule is violated and then `STOPPED_BY_MONITOR`, satisfied only when every rule is satisfied on a `COMPLETED` Run), and `stale` (the recorded catalog fingerprint is not `NewWorkflowServiceCatalog().Identity()`). Reading the canonical Case and the recorded Run, the Case and Program crossing and the supporting-sequence check are exported from `tools/umpire/replay` and called by both admissions, each mapping to its own reasons; the crossing tests include a correlated-only Case. The `Subject` carries the Case identity (SHA-256 of the canonical bytes), the Case, Program and Contract IDs, the recorded `DriverIdentity`, the Run's ID, disposition, cleanup and Verdict, and the Case's Known Gaps as its provenance declares them. It never prepares, runs, replays or reads an event's payload. Add `tools/umpire/evaluation/profile.go`: the rendered Profiles embedded with `//go:embed testdata/profiles/*.json`, `LoadProfile(name)` selecting only by an exact name from that set and decoding strictly, and the Profile identity as the SHA-256 of its embedded bytes.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/replay/`

**Size:** M
**Files:** `tools/umpire/evaluation/admission.go`, `tools/umpire/evaluation/admission_test.go`, `tools/umpire/evaluation/profile.go`, `tools/umpire/evaluation/profile_test.go`, `tools/umpire/replay/subject.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Missing, extra, crossed, stale, incompatible, open, or noncanonical subjects reject.
- [ ] Run disposition, Verdict, cleanup, Driver identity, evidence, and trust remain independent.
- [ ] Admission cannot prepare or execute a Case.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
