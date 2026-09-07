---
satisfies: [R1, R4, R5]
---
# fn-72-extract-the-reusable-temporal-testpilot.1 Relocate the complete Temporal Driver tree atomically

## Description
Relocate the composite Driver, catalog helpers, server, worker, private delivery implementation, owner READMEs, and focused tests as one atomic package move (R1, R4, R5). The Go `internal` boundary makes a smaller physical split leave a non-compiling intermediate tree.

**Size:** L (atomic mechanical relocation; splitting would violate internal visibility)
**Files:** `tests/testcore/testpilot/{catalog.go,driver.go,driver_test.go,server/**,worker/**,internal/delivery/**}`, `common/testing/temporaltestpilot/**`
**Touches:** [tests/testcore/testpilot/catalog.go, tests/testcore/testpilot/driver.go, tests/testcore/testpilot/driver_test.go, tests/testcore/testpilot/server/**, tests/testcore/testpilot/worker/**, tests/testcore/testpilot/internal/delivery/**, common/testing/temporaltestpilot/**]

### Approach
- Move the existing implementation and owner tests without rewriting behavior; preserve all comments and update only package declarations, imports, and location-bearing README prose.
- Rename only the composite package to `temporal`; keep the `server`, `worker`, and private `internal/delivery` package roles intact.
- Preserve `New`, `Options`, aliases, catalog helpers, interface assertions, server-first Open cleanup, joined Close errors, caller-owned SDK client, transport-only behavior, delivery identities, replay, cancellation, capacity, quarantine, carrier, and completion authority exactly.
- Keep the move atomic so every private-delivery importer remains below the new `internal` parent.

### Investigation targets
**Required** (read before coding):
- `tests/testcore/testpilot/driver.go:25-380` — composite API, lifecycle, and interface assertions to preserve.
- `tests/testcore/testpilot/internal/delivery/ledger.go:1-816` — private ownership and lifecycle core.
- `tests/testcore/testpilot/server/driver.go:1-315` — controller transport owner.
- `tests/testcore/testpilot/worker/driver.go:1-355` — SDK worker owner.
- `tests/testcore/testpilot/worker/sdk_test.go:1-409` — replay and SDK lifecycle regressions.

**Optional** (reference as needed):
- `tests/testcore/testpilot/README.md` — composite ownership overview to move.
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md` — relocation compatibility constraint.

## Acceptance
- [ ] The composite/catalog, server, worker, and private delivery production files exist under `common/testing/temporaltestpilot` and no implementation copy or forwarding package remains at the old path.
- [ ] Implementation-focused tests and server/worker READMEs move with their owners; generated fixtures and fixture-only tests remain under `tests/testcore/testpilot`.
- [ ] Exported signatures, options, aliases, interface assertions, validation categories, cleanup ordering, SDK client ownership, and transport-only behavior are unchanged except the composite import/package name.
- [ ] Existing composite, server, worker, delivery, carrier, replay, cancellation, capacity, quarantine, and completion-capability tests pass with `-tags test_dep` from the shared tree.
- [ ] No comments are dropped during relocation and no new dependency or runtime behavior is introduced.

## Done summary
Relocated the composite Temporal Testpilot Driver, catalog, server, worker, private delivery implementation, focused tests, and owner READMEs to `common/testing/temporaltestpilot`. Retained fixture-only tests and fixtures under `tests/testcore/testpilot`, updating the fixture test to use the shared Catalog helper; the move is uncommitted per user instruction.

baseline: red (`mise exec -- go test -count=1 -tags test_dep ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot` failed before edits because the destination package did not yet exist); focused regression baseline passed; live baseline was stopped at conductor direction.

stage: impl-review - ran (model: gpt-5.6-sol medium, SHIP)

Review: SHIP after correcting the moved root package comment to name `temporal`; receipt `/tmp/impl-review-receipt-fn-72.1-synthetic-correct.json`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: red (mise exec -- go test -count=1 -tags test_dep ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot failed pre-edit: destination package did not exist), TMPDIR=/private/tmp CGO_ENABLED=0 mise exec -- go test -p=1 -count=1 -tags test_dep ./tools/umpire/regression -run 'Test(TestpilotOwnsCaseProtocolAndRuntime|UmpireCIWorkflowRunsSeparatedUnitAndLiveProofs)' (baseline pass), TMPDIR=/private/tmp CGO_ENABLED=0 mise exec -- go test -p=1 -count=1 -tags test_dep ./common/testing/temporaltestpilot/... (pass after review fix), impl-review SHIP: /tmp/impl-review-receipt-fn-72.1-synthetic-correct.json
- PRs:
