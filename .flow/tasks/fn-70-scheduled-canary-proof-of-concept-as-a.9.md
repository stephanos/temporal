---
satisfies: [R2, R3, R4, R5, R6, R7, R8, R9, R10]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.9 Qualify successive real canary ticks and publish reproducible operator guidance

## Description
Qualify successive real canary ticks and publish reproducible operator guidance.

**Size:** M
**Files:** tests/canary_scheduled_test.go; tools/canary/README.md; common/testing/testpilot/README.md; common/testing/testpilot/temporal/README.md; tests/testcore/testpilot/README.md; Makefile
**Touches:** [tests/canary_scheduled_test.go, tools/canary/README.md, common/testing/testpilot/README.md, common/testing/testpilot/temporal/README.md, tests/testcore/testpilot/README.md, Makefile]

### Approach
- Add TestCanaryScheduledNexusSuccess using actual operator/worker paths and at least two successive real60s Schedule ticks within generous5minute event-driven deadline. Verify completed disposition, satisfied Verdict, successful cleanup, exact Producer correlated support, distinct orchestration/Testpilot identities and immutable bounded artifacts.
- Run existing TestTestpilotAsyncNexusCase and new early two-binding proof against exact post-amendment artifact. Preserve unchanged Contract/source semantics across environments and dependency boundaries; no synthetic-only or manual-trigger replacement for the two ticks.
- Document exact build/owner artifact provisioning/pin update, dedicated local CHASM true-policy settings, transport/callback/binding provisioning, list/apply/worker/pause/resume/status/shutdown/owned cleanup, limits and retention/error behavior. Explain separate45s execution phases,60/75/90s deadlines,5/20s heartbeat,60s target lifetime,10s catch-up and same-cluster freshness limitations; no production readiness claim.
- Wire exact new integration names into existing applicable live gate selection without removing inherited tests or modifying baseline acceptance. Run focused/full required Go and model/fixture gates serially as applicable; compare original raw trust/fixture evidence and report inherited failures precisely.

### Investigation targets
**Required:**
- tests/testpilot_async_nexus_case_test.go:47,221 — real success and correlated support.
- Makefile:1140,1164 — existing live/regression gates.
- common/testing/testpilot/temporal/README.md:3 — shared consumer ownership.
- tests/testcore/testpilot/README.md:3 — owner-managed fixture boundary.

### Quick commands
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^TestCanaryScheduledNexusSuccess$' -timeout 10m > /tmp/fn70-task9-ticks.jsonl`
Then verify both terminal exit 0 and JSON records: each exact named top-level test has Action=run and Action=pass, no matching test/subtest Action=skip or fail, and package pass. Parse with a bounded script; a missing record is failure. This command names a required new test, not a currently existing one.
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^(TestTestpilotAsyncNexusCase|TestCanaryActivitySharedDriverBindings)$' -timeout 10m` (require both exact run/pass records and no skip)
`mise exec -- go test -count=1 -tags test_dep ./tools/canary/... ./common/testing/testpilot/... ./chasm/lib/scheduler`
`mise exec -- make umpire-check-case-runtime-conformance umpire-build-model lint-model`
`mise exec -- make umpire-check-regression`
`mise exec -- make lint-code GOLANGCI_LINT_FIX=false`

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] At least two successive real minute ticks pass with Producer-correlated support and distinct retained identities; named live tests cannot skip.
- [ ] Existing functional consumer and alternate binding pass on identical pinned artifact with shared Driver and unchanged Contract semantics.
- [ ] R1–R8 focused failure/replay/policy/retention coverage remains wired; all applicable gates have explicit terminal outcomes and original trust comparison.
- [ ] R10 runbook reproduces local provisioning through bounded shutdown, documents exact limits/configuration and avoids production/exactly-once claims.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
