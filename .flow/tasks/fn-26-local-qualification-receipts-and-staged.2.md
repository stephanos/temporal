---
satisfies: [R2, R3, R4]
---

# fn-26-local-qualification-receipts-and-staged.2 Admit exact Case Runtime qualification subjects

## Description
Add `tools/umpire/evaluation/admission.go`: `Admit(caseBytes, recordedRun []byte, catalog *testpilot.Catalog) (*Subject, error)` over the canonical Case (`casefile.Canonical`, then `testpilot.DecodeCaseProtoJSON`) and fn-22's recorded Run (`replay.DecodeRecordedRun`), rejecting by reason class before any assessment: noncanonical, malformed, crossed (the Run's Case, Program or Contract ID is not the Case's; a rule the Verdict names is not the Contract's), open (no terminal disposition, no cleanup outcome, no Verdict), inconsistent (a supporting sequence naming no event or an event twice), stale (the recorded catalog fingerprint is not `catalog.Identity()`), and oversized (beyond the Profile-independent byte and event caps). The `Subject` carries the Case identity (SHA-256 of the canonical bytes), the IDs, the recorded `DriverIdentity`, the Run's ID, disposition, cleanup and Verdict, and the Case's Known Gaps as its provenance declares them. It never prepares, runs, replays or reads an event's payload.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ -run Admit`

**Size:** M
**Files:** `tools/umpire/evaluation/admission.go`, `tools/umpire/evaluation/admission_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers); the spec's **Re-plan** section states the contracts.

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
