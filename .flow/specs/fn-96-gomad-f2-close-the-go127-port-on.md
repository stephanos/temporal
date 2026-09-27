# Gomad F2: close the go1.27 port on darwin/arm64

## Goal & Context
<!-- scope: business -->

Milestone F2 of `.plans/GOMAD_MILESTONES.md`. The toolchain is ported to go1.27.1 and every tier
passes on linux/amd64. Left open: the macOS `upgrade-dossier` run, the DTrace clock audit, the
approval digest for the go1.26.4-v2 to go1.27.1-v1 boundary diff, and the Temporal corpus
acceptance on darwin/arm64.

## Architecture & Data Models
<!-- scope: technical -->

`make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=<go1.26.4 commit>` publishes
`.toolchain/upgrade-dossier.json`. A non-empty boundary diff is approved by rerunning with
`GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` equal to the reviewed canonical diff digest; CI reads the
same repository variable. The DTrace audit (`make clock-audit`) needs root.

## Edge Cases & Constraints
<!-- scope: technical -->

- The boundary diff is reviewed entry by entry before its digest is recorded; it is never waved
  through.
- If root is unavailable the dossier honestly stays `qualified=false` for the clock audit, and the
  status records that; this is not a failure of the milestone.
- Keep the go1.26.4 descriptor in git history only.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `tools/gomad3/.toolchain/bin/go version` reports `go1.27.1` on darwin/arm64 and
  `make -C tools/gomad3 validate-toolchain` passes.
- **R2:** The upgrade dossier on darwin/arm64 reports every gate passed except any gate that needs
  root, and the boundary diff is either empty or approved by an exact recorded digest. Errors: a
  failed gate is fixed or recorded with its exact output.
- **R3:** The approved boundary diff digest is recorded in the milestone status (and the value the
  `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable needs), with the reviewed entries
  listed.
- **R4:** Whether the DTrace audit ran is recorded; if it ran, its result.
- **R5:** `make gomad3-qualification` on darwin/arm64 reproduces 5 supported and 11 unsupported
  with the same blocker paths as the manifest expectations (plus the entries later milestones
  added). Errors: a changed blocker is investigated and either fixed or the expectation is
  corrected with evidence.

## Boundaries
<!-- scope: business -->

- Setting the GitHub repository variable itself (needs repo admin; record the value instead).
- Any Go version other than go1.27.1.

## Decision Context
<!-- scope: both -->

Linux could not produce the darwin dossier; this host can.
