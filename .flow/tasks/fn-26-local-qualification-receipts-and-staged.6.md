---
satisfies: [R1, R4, R5, R7, R8]
---

# fn-26-local-qualification-receipts-and-staged.6 Prove assessment isolation live and document the contract

## Description
Close the Profile (every load rejection), subject, receipt, reason, multiplicity, cap, Known Gap, publication and output matrices: a second, test-only Profile (.2's JSON fixture, never declared in Lean or embedded) assesses the same subject to a different receipt without touching the first; the same subject and Profile publish byte-identical receipts twice; an existing final name that is a symlink, FIFO, directory, oversized or different file is a conflict left untouched; every crossed (a correlated-only Case included), stale, open, inconsistent, incompatible and oversized subject rejects before assessment, at N and N+1 of each cap; an inconclusive Verdict is `incomplete` and a violated one `rejected`, as is a violated, stopped Run whose cleanup failed (with `cleanup-unclosed` among its reasons); a regenerated Case with the same IDs against an older record is `crossed`. Live proof in the Testpilot suite: record the caller Model's `asyncCompletion` Case with `umpire-run --record` against the test cluster, assess it with `umpire-assess run` twice under `local-ephemeral` (accepted, one receipt, the second `already-published`), and assess the negative control's pinned record (`rejected`, its Verdict violated) -- no Driver or Run is created by either assessment. Document the exact environment-scoped claim (including that `local-ephemeral-cluster` trust is asserted by the Profile, not checked against the recorded identity), that publication needs hard links and fails closed (exit 3) on a filesystem without them, the lack of self-authentication, the separation between the Case Runtime's Verdict and offline Claim Assessment, and the retained exclusions in `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md` and `.plans/UMPIRE4_SPEC.md`.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/...; go test -count=1 -tags "test_dep integration" ./tests/ -run '^TestTestpilotAssess'; make lint-code-fast`

**Size:** M
**Files:** `tools/umpire/evaluation/**`, `tests/testpilot_assess_test.go`, `model/README.md`, `.plans/UMPIRE4_COMPONENTS.md`, `.plans/UMPIRE4_SPEC.md`

### Re-plan note (2026-09-23)
Rewritten on fn-85 and fn-22 (the recorded subject, the canonical Case form, the shared command-edge helpers) and revised by plan review round one; the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Crossed, N/N+1, multiple-Profile, idempotency, cleanup, evidence, identity, and output cases fail at the intended boundary.
- [x] Focused and aggregate Lean/Go/regression, formatting, and lint gates pass with `-tags test_dep` for Go tests.
- [x] Existing comments remain accurate and docs exclude CI, remote, canary, production, release, and implicit authority.


## Done summary
Live proof in `tests/testpilot_assess_test.go`: the caller Model's asyncCompletion Case is recorded with `umpire-run --record` against the test cluster and assessed twice by `umpire-assess run` under `local-ephemeral` (accepted, one receipt, the second `already-published`, the recorded Run byte-identical after), and the negative control's pinned record is rejected (`verdict-violated`, `monitor-stopped`); neither assessment is given an address. `tools/umpire/evaluation/caps_test.go` admits at N and rejects at N+1 for the Case bytes, the recorded Run bytes and the Run events (the receipt cap's N/N+1 is in the receipt tests). The remaining matrices (every Profile load rejection, subject rejection classes, reasons, multiplicity, the test-only Profile's distinct receipt, publication conflicts, output statuses) are the tests tasks .2 to .5 delivered. `model/README.md`, `.plans/UMPIRE4_SPEC.md` (Claim Assessment concept, QLF-03 amendment) and `.plans/UMPIRE4_COMPONENTS.md` state the environment-scoped claim, the asserted trust, the lack of self-authentication, the hard-link requirement, the Verdict/assessment separation and the retained exclusions. `make umpire-check-live-tests` passes with 32 identities. Implementation review: SHIP in one round, its two P3 notes applied.
## Evidence
- Commits: ba56f66ecf1b2f5005084c66143d0028286a7a4b, 8bb9eeb37ddb543ff451c55921a590e99820cdb8
- Tests: go test -count=1 -tags test_dep ./tools/umpire/..., make umpire-check-live-tests (32 passing identities), make umpire-check-retired-vocabulary umpire-check-evaluation-profiles, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: