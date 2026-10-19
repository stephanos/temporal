# Gomad: qualify the one-box cell and define the per-test support claim

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

`internal/backends/wasm/QUALIFICATION.md` lists four workload cells. The fixture and SQLite cells are qualified on the current candidate; the one-box smoke cell, whose source is the existing Temporal persistence/functional guest owner, is recorded as "unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay pending". The same document says the matrix supplies neither an arbitrary Temporal support claim nor a cross-host guarantee.

Two things are missing on the way to arbitrary functional tests. First, the one Temporal guest that already runs (`TestFrontendSystemInfo` in `tests/gomadfunctional`, prepared by the one-command setup spec) has never been carried through the qualification gates, so there is no qualified Temporal cell at all. Second, there is no defined evidence shape for saying "this test is supported on this candidate". Without it, every later pass from the triage sweep is an unlabeled observation. This spec closes both gaps: it qualifies the one-box cell on the current candidate and defines a per-test support list bound to a candidate, deliberately not a blanket claim.

## Architecture & Data Models

**One-box qualification.** Run the one-box cell through the gates QUALIFICATION.md already requires for a cell: unchanged source preparation, storage/transport artifact qualification, and repeated measured pass and replay on the frozen candidate, with numeric exits, host identity, helper, compiler, module and model identities recorded in the retained checkpoint. The guest is built with `backend.TemporalGuestBuildTags()`; the configuration comes from the one-command setup spec. The qualification owner stays `internal/evidence/qualification` wrapped by `internal/backends/wasm/qualification`, as the document states.

**Per-test support list.** A retained, candidate-bound document whose shape task .2 defines and records, with at least: candidate identity (helper digest, compiler, runtime overlay identity, module identity), one row per supported test (package, function, gates passed with exits, retained artifact reference), and an explicit statement that tests not listed are unsupported. The list is produced by the same qualification owner and lives next to the existing checkpoint records. It is additive to QUALIFICATION.md, which gains a section describing the shape and pointing at the current list.

## API Contracts

- The one-box cell row in QUALIFICATION.md changes from pending to qualified only with the retained checkpoint reference, exits and identities, in the same style as the fixture and SQLite rows.
- The support list is bound to one candidate; a new candidate starts an empty list and re-earns rows. Historical lists keep their identities.
- A test enters the list only through the cell gates; a sweep pass alone never adds a row.

## Edge Cases & Constraints

- If the one-box cell fails a gate on the current candidate, the row stays pending with the failing gate and exit recorded; no partial qualification.
- Capacity and measured-cost limits in QUALIFICATION.md apply unchanged; raising a limit for the one-box cell is a recorded decision in the checkpoint.
- The support list must not be readable as "functional suite supported"; its header and QUALIFICATION.md wording say per-test and candidate-bound.
- No native backend involvement; native execution stays deferred under fn-128/fn-149 and this spec does not mention it beyond that boundary.
- No CI run, PR or push.

## Acceptance Criteria

- **R1:** The one-box cell is qualified on the current candidate: every gate QUALIFICATION.md names for a cell has a recorded numeric exit, identities and host in a retained checkpoint under `.flow/artifacts`, and the QUALIFICATION.md row references it. Errors: a missing gate or exit leaves the row pending.
- **R2:** A per-test support list shape is defined, documented in QUALIFICATION.md, and the first list exists for the current candidate containing exactly the tests that passed the gates (at minimum `TestFrontendSystemInfo`). Errors: a row without gate evidence, or a claim covering unlisted tests, is a finding.

## Boundaries

- No new guest capability, no slim build tag change, no sweep.
- No change to the fixture, SQLite or multi-instance cells.
- No native backend involvement.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): the support claim for arbitrary functional tests is a per-test list bound to a candidate, not a blanket claim. This keeps the existing qualification discipline (identity-bound, gate-backed, no relabeling of historical evidence) and lets the list grow one test at a time as refusals are closed. Rejected: a package-level or suite-level claim (cannot be gate-backed test by test); waiting for the sweep before qualifying anything (leaves no qualified Temporal cell while the sweep runs).

## Early proof point

Task .1 runs the one-box cell's first gate (unchanged source preparation) on the current candidate with the setup-spec configuration. If preparation itself fails, stop and report before running the measured gates.

## Quick commands

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/qualification ./internal/evidence/qualification
# cell gates and their environment are named in internal/backends/wasm/QUALIFICATION.md
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | One-box cell qualified on the current candidate with retained checkpoint. | fn-222-gomad-qualify-the-one-box-cell-and.1 | — |
| R2 | Per-test support list shape defined, documented, first list produced. | fn-222-gomad-qualify-the-one-box-cell-and.2 | — |
