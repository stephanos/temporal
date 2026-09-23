---
satisfies: [R1, R4, R5, R7, R8]
---

# fn-26-local-qualification-receipts-and-staged.6 Prove assessment isolation live and document the contract

## Description
Close the Profile (every load rejection), subject, receipt, reason, multiplicity, cap, Known Gap, publication and output matrices: a second, test-only Profile (.2's JSON fixture, never declared in Lean or embedded) assesses the same subject to a different receipt without touching the first; the same subject and Profile publish byte-identical receipts twice; an existing final name that is a symlink, FIFO, directory, oversized or different file is a conflict left untouched; every crossed (a correlated-only Case included), stale, open, inconsistent, incompatible and oversized subject rejects before assessment, at N and N+1 of each cap; an inconclusive Verdict is `incomplete` and a violated one `rejected`, as is a violated, stopped Run whose cleanup failed (with `cleanup-unclosed` among its reasons); a regenerated Case with the same IDs against an older record is `crossed`. Live proof in the Testpilot suite: record the caller Model's `asyncCompletion` Case with `umpire-run --record` against the test cluster, assess it with `umpire-assess run` twice under `local-ephemeral` (accepted, one receipt, the second `already-published`), and assess the negative control's pinned record (`rejected`, its Verdict violated) -- no Driver or Run is created by either assessment. Document the exact environment-scoped claim (including that `local-ephemeral-cluster` trust is asserted by the Profile, not checked against the recorded identity), the lack of self-authentication, the separation between the Case Runtime's Verdict and offline Claim Assessment, and the retained exclusions in `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md` and `.plans/UMPIRE4_SPEC.md`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/...; go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotAssess'; make lint-code-fast`

**Size:** M
**Files:** `tools/umpire/evaluation/**`, `tests/testpilot_assess_test.go`, `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_SPEC.md`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Crossed, N/N+1, multiple-Profile, idempotency, cleanup, evidence, identity, and output cases fail at the intended boundary.
- [ ] Focused and aggregate Lean/Go/regression, formatting, and lint gates pass with `-tags test_dep` for Go tests.
- [ ] Existing comments remain accurate and docs exclude CI, remote, canary, production, release, and implicit authority.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
