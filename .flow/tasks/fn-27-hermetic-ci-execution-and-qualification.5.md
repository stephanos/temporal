---
satisfies: [R2, R3, R4, R5, R6]
---
# fn-27-hermetic-ci-execution-and-qualification.5 Compose CI provenance, execution, conformance, and qualification

## Description
Implement R2/R3/R4/R5/R6 behind one deep end-to-end controller with no second authority.

**Size:** M
**Files:** `tools/umpire/ci/**`, `tools/umpire/qualification/**`, `tools/umpire/temporal/nexus/**`
**Touches:** [tools/umpire/ci/**, tools/umpire/qualification/**, tools/umpire/temporal/nexus/**]

### Approach

- Build a package-private-injectable, production-fixed two-stage CI collector with the exact variable/material/toolchain allowlist, deadlines, bounds, workflow/profile digest checks, pre/post tracked-tree checks, a read-only input strictly below the workspace, output strictly below runner-temp and disjoint from the workspace/input, symlink/retained-identity checks, and derived run-id checks.
- Add `QualifyCI` as an offline fn-26 policy specialization over an admitted CI six-member set, strict pilot evidence, exact v2 profile, and checked provenance; retain the local path and reason table unchanged.
- Expose one orchestration API that performs admission/pilot/provenance/profile before IO, then runtime, conformance, qualification, v3 construction, and a single final publish.
- Preserve valid failed/incomplete stage evidence and all independent status dimensions; return structured phase-tagged tooling failures when a stage cannot construct its artifact.
- Wire SIGINT/SIGTERM through fresh isolation/cleanup contexts, finalize postflight status after cleanup, revalidate output containment under the publication lock, never auto-retry, and retain authoritative publication/reporting booleans.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — checked runtime API and cleanup semantics
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — pure conformance API and errors
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — pilot/policy/status/receipt construction
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.2.md` — strict source closure
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.4.md` — offline qualification controller
- `.flow/tasks/fn-18-versioned-umpire-artifact-boundary.10.md` — sole publisher semantics

### Acceptance

- [ ] Every malformed or mismatched preflight/root input performs no runtime IO; package consumers cannot substitute provenance/profile/checker/authority.
- [ ] One bounded valid run traverses runtime, canonical conformance, offline policy, v3 admission, and exactly one final publication.
- [ ] Valid runtime/semantic/qualification non-success is published with all dimensions intact and status 2; tooling failures use the exact phase/boolean contract.
- [ ] Postflight tree/resource/authority outcomes follow the exact reason table; containment changes, cancellation, cleanup uncertainty, publication conflict, and broken reporting never leak authority or trigger an automatic rerun.

## Acceptance
- [ ] R2/R3/R4/R5/R6 end-to-end controller and isolation invariants are complete.
- [ ] Independent fakes cover every stage, status precedence, cancellation, and publication ambiguity.
- [ ] Existing runtime/conformance/qualification comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
