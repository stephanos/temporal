---
satisfies: [R5, R6]
---

# fn-28-portable-evaluation-contract-and.7 Add the attached disposable-cluster authority adapter
## Description

Add the narrow adapter that lets the resident executor use a caller-owned Temporal SDK client and namespace while retaining strict ownership of Umpire-created workers and run resources.

**Size:** M
**Files:** `tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`
**Touches:** [`tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`]

### Approach
- Accept only the minimum attached-authority capabilities needed by the existing local environment; do not import `tests/testcore` into production packages.
- Treat the cluster and SDK client as borrowed, start fresh run-owned workers/task queues, and stop/delete only resources acquired by Umpire.
- Exercise the adapter with focused fakes before the tagged test supplies the concrete `testcore.NewEnv` implementation.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go` authority and resource accounting.
- `tests/testcore.TestEnv` SDK client, namespace, worker, and cleanup ownership.
- Parent executor interface and existing runner adapter contract.

## Acceptance
- [ ] The exported attached-authority seam is minimal and has both existing loopback/fake and later testcore adapters.
- [ ] Borrowed cluster/client resources are never stopped or reported as Umpire-owned; every Umpire-created worker/resource closes exactly once.
- [ ] Drift, nil/incomplete authority, cancellation, cleanup failure, and reuse tests fail closed.

## Done summary
Finalized the previously implemented attached disposable-cluster authority adapter after auditing implementation commits `7d6d13936629a76f0d3b66ea3a123b71d234711c` and `f0d008def7176dc9a0fbdc12693da8c922dcc307`, task receipt `2670c60025f014d7f9157a6b106cc964fa6c75cf`, plan-sync record `122b02fa1cfe6d5e94e9d8c79cf8d121179208f0`, and the authoritative Codex SHIP review over `372acd4eb40e8dda10473a3b2dc163f30b2b73fc..f0d008def7176dc9a0fbdc12693da8c922dcc307`; the reviewed adapter remains unchanged, and focused local unit, race, vet, lint, and executor integration checks are green.

The finalization-only review inspected the intentionally empty `79cd5d2a52e6be1eb5c45f1b92fa58b9221a783a..HEAD` range and returned scope-only NEEDS_WORK with zero introduced or pre-existing code findings, so it does not supersede the authoritative implementation review; the unrelated user modifications in `config/development.yaml` and `schema/elasticsearch/visibility/index_template_v7.json` remain untouched.

baseline: green (`go test -count=1 -tags test_dep ./tools/umpire/temporal/local/...`)

GATE_CLASSIFY_FULL: unrelated user-owned `config/development.yaml` working-tree modification

stage: impl-review - ran [2026-09-02T02:52:34Z..2026-09-02T02:53:35Z] (scope-only NEEDS_WORK on an empty finalization range; authoritative implementation SHIP retained)
## Evidence
- Commits:
- Tests: baseline: green (go test -count=1 -tags test_dep ./tools/umpire/temporal/local/...), go test -race -count=1 -tags test_dep ./tools/umpire/temporal/local/..., go vet -tags test_dep ./tools/umpire/temporal/local/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=372acd4eb40e8dda10473a3b2dc163f30b2b73fc --config=.github/.golangci.yml ./tools/umpire/temporal/local/... (0 issues), go test -count=1 -tags test_dep ./tools/umpire/executor/..., GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification, NO_RECEIPT: gate receipt was not warrantable while unrelated user-owned config/development.yaml remained dirty, AUTHORITATIVE_REVIEW_SHIP: 372acd4eb40e8dda10473a3b2dc163f30b2b73fc..f0d008def7176dc9a0fbdc12693da8c922dcc307, FINALIZATION_REVIEW_SCOPE_ONLY: 79cd5d2a52e6be1eb5c45f1b92fa58b9221a783a..HEAD returned NEEDS_WORK with 0 introduced and 0 pre-existing findings because the range was empty, INTEGRATION_CONTEXT: tagged TestUmpirePortableCanaryExecutor has no matching test until task .9 and was not treated as task .7 proof
- PRs: