---
satisfies: [R2, R3, R4]
---

# fn-26-local-qualification-receipts-and-staged.2 Admit exact Case Runtime qualification subjects and load the Profiles

## Description
Add `tools/umpire/evaluation/admission.go`: `Admit(caseBytes, recordedRun []byte, catalog string) (*Subject, error)` (`catalog` the fingerprint to compare against: the command passes `NewWorkflowServiceCatalog().Identity()`, the goldens a fixed one) over the canonical Case (`casefile.Canonical`, then `testpilot.DecodeCaseProtoJSON`) and fn-22's recorded Run (`replay.DecodeRecordedRun`), rejecting by reason class before any assessment: `noncanonical` (a Case not in canonical form, or a recorded Run whose bytes are not `EncodeRecordedRun`'s re-encoding of what they decode to), `malformed`, `incompatible` (a Case whose format version is not 1.0), `oversized` (a Case over 4 MiB, a recorded Run over 16 MiB or 65,536 events, named constants), `crossed` (the Run's `case_id` or `program_id` is not the Case's; the Verdict's rule IDs are not exactly the Contract's `rules` and `correlated.rules` together, each once), `open` (an empty Run ID, no events, no terminal disposition, no cleanup outcome, no Verdict), `inconsistent` (a supporting sequence naming no event or an event twice; a rule of the closed Verdict still `PENDING` or `UNSPECIFIED`; an `UNSPECIFIED` Verdict status; a Verdict status that disagrees with its rules or the disposition either way: violated exactly when some rule is violated, `STOPPED_BY_MONITOR` exactly when violated, satisfied exactly when every rule is satisfied on a `COMPLETED` Run, inconclusive otherwise), and `stale` (the recorded catalog fingerprint is not `catalog`). The recorded-Run codec, reading the canonical Case, the Case and Program crossing, the supporting-sequence check and the violated-form rule (`ViolatedForm`, from `replay/form.go`, which `replay/rerun.go` also calls) move to a leaf package, `tools/umpire/internal/recordedrun`, whose codec also rejects a duplicate or case-folded outer key, which `tools/umpire/replay` (keeping its names as aliases) and `tools/umpire/evaluation` both import, each mapping to its own reasons; the crossing tests include a correlated-only Case. The `Subject` carries the Case identity (SHA-256 of the canonical bytes), the Run identity (SHA-256 of the canonical recorded-Run bytes), the Case, Program and Contract IDs, the recorded `DriverIdentity`, the Run's ID, disposition, cleanup and Verdict, and the Case's Known Gaps as its provenance declares them. It never prepares, runs, replays or reads an event's payload. Add `tools/umpire/evaluation/profile.go`: the rendered Profiles embedded with `//go:embed testdata/profiles/*.json`, `LoadProfile(name)` selecting only by an exact name from that set, decoding strictly and validating as Lean declares -- an unknown condition, decision or Known Gap kind, an empty table, an empty or duplicate reason name, a repeated condition, `known-gap-blocking` without a blocking kind or blocking kinds without it all reject, so `Assess` only ever receives a valid Profile -- and the Profile identity as `sha256:<hex>` of its embedded bytes, which a Go test matches against the identity pinned in Lean. An unexported `parseProfile(bytes)` does the decoding and validation `LoadProfile` uses, so the test-only Profile fixture (`testdata/test-profiles/`, outside the embedded pattern) and every load rejection are exercised here.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/evaluation/ ./tools/umpire/replay/`

**Size:** M
**Files:** `tools/umpire/evaluation/admission.go`, `tools/umpire/evaluation/admission_test.go`, `tools/umpire/evaluation/profile.go`, `tools/umpire/evaluation/profile_test.go`, `tools/umpire/internal/recordedrun/**`, `tools/umpire/replay/recorded.go`, `tools/umpire/replay/subject.go`, `tools/umpire/replay/form.go`, `tools/umpire/replay/rerun.go`, `tools/umpire/evaluation/testdata/test-profiles/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Missing, extra, crossed (a correlated-only Case included), stale, incompatible, open (an empty Run ID or no events included), inconsistent (each direction of the Verdict agreement and an `UNSPECIFIED` status included), or noncanonical (a re-spaced recorded Run and a duplicate or case-folded outer key included) subjects reject.
- [ ] The Subject's Run identity is the SHA-256 of the canonical recorded bytes; replay admission still passes its tests over the shared leaf package.
- [ ] A Profile with an unknown condition, decision or Known Gap kind, an empty table, a duplicate reason or condition, or a contradictory Known Gap policy fails to load; the embedded Profile's identity matches the one pinned in Lean.
- [ ] Admission cannot prepare or execute a Case, and qualification never imports the package that holds the replay bridge.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
