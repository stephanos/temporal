---
satisfies: [R1, R2, R5, R6, R7]
---
# fn-73-explicit-environment-binding-for.5 Migrate the checked Nexus3 Case and owned fixture

## Description
Move only the checked Nexus3 success producer to symbolic Case 1.1 and regenerate its owner-managed functional fixture (R1, R2, R5, R6). Preserve model selection, Contract meaning, Behavior Fingerprints and producer provenance while removing physical resource literals.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus3/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`, `tests/testcore/testpilot/{artifact_test.go,testdata/async-nexus-case.json}`, a small shared test fixture helper if needed
**Touches:** [model/Temporal/Feature/Nexus3/Testpilot.lean, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/artifact_test.go, tests/testcore/testpilot/testdata/async-nexus-case.json, tests/testcore/testpilot/*.go]

### Approach
- Declare stable symbolic IDs for worker namespace, task queue and named Nexus endpoint through the authored Testpilot facade.
- Replace only namespace/task-queue assignments and role resources; leave workflow/request IDs, types, service/operation names, payloads, limits, Properties and Contract predicates behavioral.
- Advance this producer to 1.1 and keep the other functional and generic conformance fixtures byte-identical at 1.0.
- Regenerate through the transactional fixture owner and compare the migrated semantic baseline separately from the expected one-time Program/Case byte change.
- Centralize reusable binding IDs/Profile construction for artifact and live tests in a small test-facing helper rather than duplicating physical configuration.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus3/Testpilot.lean:74-107,240-290` — hard-coded resources and checked producer/version
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:229-324` — transactional functional fixture ownership
- `tests/testcore/testpilot/artifact_test.go:176-206` — fixture admission Profile
- `tests/testcore/testpilot/testdata/async-nexus-case.json` — sole functional fixture that moves to 1.1
- `model/Temporal/Tool/Testpilot.lean:21-35` — generator routing

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus3/Nexus.md:2-12` — behavioral semantics that remain unchanged
- `tests/testcore/testpilot/protobuf_lean_authoring_test.go` — cross-language fixture assertions

### Key context
The exact new 1.1 fixture must be reused across environments in task 6. The whole Case bytes change once during migration; the Contract bytes, definition bindings, Behavior Fingerprints and provenance meaning do not.

## Acceptance
- [ ] Nexus3 declares environment definitions and bound worker/queue/endpoint roles through authored helpers and emits Case 1.1.
- [ ] Namespace and task-queue request carriers contain direct EnvironmentRefs; the Producer contains no physical namespace, queue or endpoint names.
- [ ] The checked selection, definition bindings, Behavior Fingerprints, Contract bytes and producer provenance meaning match the pre-migration semantic baseline.
- [ ] Only `async-nexus-case.json` changes semantically/version-wise; all other managed 1.0 fixtures remain byte-identical.
- [ ] The regenerated fixture prepares successfully with two distinct valid Profile snapshots and rejects missing/inconsistent bindings before dispatch.
- [ ] The transactional fixture check, relevant Lean Nexus3/Testpilot builds, and focused Go tests with `-tags test_dep` pass.

## Done summary
Migrated the checked Nexus3 producer to symbolic Case 1.1 with stable authored namespace, task-queue and Nexus-endpoint binding IDs and direct request EnvironmentRefs. Regenerated only the owner-managed async fixture, centralized reusable Profile construction, and added two-Profile plus missing/inconsistent pre-dispatch coverage. Contract/provenance JSON stayed identical and every other managed fixture stayed byte-identical. Repository checks, Lean builds, focused Go tests, and independent implementation review all passed; review verdict SHIP. Plan sync was skipped because `planSync.enabled` is false.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp make umpire-gen-case-runtime-conformance, TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 make umpire-check-case-runtime-conformance umpire-check-testpilot-protocol umpire-check-testpilot-authoring, (cd model && mise exec -- lake build Testpilot TestpilotTests Temporal.Feature.Nexus3.Tests temporal-testpilot), TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./common/testing/temporaltestpilot/... ./tests/testcore/testpilot/..., gofmt -d on task-changed Go files, git diff --check, implementation review SHIP: /tmp/impl-review-receipt-fn-73-explicit-environment-binding-for.5.json, semantic comparison: Contract/provenance equal; non-async managed fixtures byte-identical
- PRs: