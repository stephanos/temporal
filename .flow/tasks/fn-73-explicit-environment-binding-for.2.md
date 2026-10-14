---
satisfies: [R1, R2, R4, R7]
---
# fn-73-explicit-environment-binding-for.2 Resolve and fingerprint immutable environment snapshots

## Description
Implement the scenario-neutral Profile snapshot, Case-version admission and deterministic Prepare-time resolver (R1, R2, R4). This is the early proof point: one symbolic source must yield immutable prepared values and distinct binding identities without target I/O.

**Size:** M
**Files:** `common/testing/testpilot/profile.go`, `common/testing/testpilot/prepare.go`, `common/testing/testpilot/internal/execution/{prepare,dataflow,program,request}.go`, focused tests under the same packages
**Touches:** [common/testing/testpilot/profile.go, common/testing/testpilot/prepare.go, common/testing/testpilot/internal/execution/**, common/testing/testpilot/*_test.go]

### Approach
- Deep-copy and validate the complete Profile binding collection once, then compute a domain-separated, length-delimited SHA-256 fingerprint over sorted pairs.
- Admit exactly literal-only Case 1.0 and closed binding-bearing Case 1.1; reject binding fields in 1.0 and meaningless literal-only 1.1 Programs.
- Extend declaration binding and direct request-assignment preparation; resolve environment references only after a singular text destination is known, retaining symbolic metadata while compiling the private runtime value.
- Keep the general expression compiler exhaustive so every nested, guard, payload and Contract placement rejects automatically.
- Store copied resolved role metadata on the private PreparedProgram and preserve the original symbolic source Snapshot.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/profile.go:60-107` — ProfileSpec snapshot authority
- `common/testing/testpilot/internal/execution/prepare.go:56-83,232-335` — version and role admission
- `common/testing/testpilot/internal/execution/dataflow.go:455-490` — destination-aware assignment binding
- `common/testing/testpilot/internal/ir/expression.go:234-262` — exhaustive unsupported-expression rejection
- `common/testing/testpilot/internal/execution/program.go:71-83,156` — immutable program storage and ceilings

**Optional** (reference as needed):
- `common/testing/testpilot/internal/execution/request.go:31-41` — runtime request construction
- `common/testing/testpilot/internal/ir/write.go:18-48` — final materialized-size enforcement

### Key context
The fingerprint covers every validated Profile binding, including unused entries, while Behavior Fingerprints and producer provenance remain untouched. Final request size still needs the existing runtime check because Run-derived values are unavailable during Prepare.

## Acceptance
- [ ] Profile bindings are fully validated and deep-copied, including unused extras; malformed IDs/UTF-8, empty or duplicate entries, collection and cumulative-byte overflow reject.
- [ ] Fixed fingerprint vectors cover reordering, delimiter-containing values, caller mutation and unused-binding changes, with lowercase hexadecimal output.
- [ ] Case 1.0 rejects every binding field and preserves all existing literal fixtures; Case 1.1 requires a nonempty closed graph and rejects unused definitions, undeclared references and literal-only Programs.
- [ ] Environment references are accepted only as direct singular-text request assignments and become private resolved execution inputs with preserved symbolic identity.
- [ ] Resolved role values and symbolic source snapshots remain immutable across caller mutation and concurrent preparations/Runs.
- [ ] Focused `go test -count=1 -tags test_dep` suites for `common/testing/testpilot/...` pass without target I/O.

## Done summary
Implemented immutable Profile environment snapshots, canonical domain-separated binding fingerprints, exact Case 1.0/1.1 admission, direct singular-text environment resolution, and private resolved role metadata. Added malformed/limit/version/closure, fixed-vector, mutation, concurrency, and request-materialization coverage. Independent implementation review returned SHIP after fixing negative-minor admission; public Driver identity propagation remains intentionally assigned to task 3. Plan sync was skipped because `planSync.enabled` is false.
## Evidence
- Commits:
- Tests: TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/..., TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=10 -tags test_dep ./common/testing/testpilot/... (focused concurrency/fingerprint selection), git diff --check, implementation review SHIP: /tmp/impl-review-receipt-fn-73-explicit-environment-binding-for.2.json
- PRs: