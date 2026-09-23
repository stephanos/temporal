---
satisfies: [R3, R10]
---

# fn-29-bounded-production-canary-execution-and.3 Acquire authority from the protected workflow and preflight the exact scope

## Description
Add `tools/canary/authority`: read the credentials from the environment only (`UMPIRE_CANARY_TLS_CERT`, `UMPIRE_CANARY_TLS_KEY`, `UMPIRE_CANARY_API_KEY`, and the coordinates `UMPIRE_CANARY_GRPC`, `UMPIRE_CANARY_HTTP`, `UMPIRE_CANARY_NAMESPACE`, `UMPIRE_CANARY_TASK_QUEUE`, `UMPIRE_CANARY_HANDLER_QUEUE`, `UMPIRE_CANARY_ENDPOINT`), build gRPC transport and per-RPC credentials for the Driver's server endpoint and the SDK client, and a `Redactor` that removes every credential and raw coordinate from any text before it is written. Add `tools/canary/preflight`: before any mutation, require `GITHUB_EVENT_NAME=workflow_dispatch`, `GITHUB_REF` the policy's trusted ref and `GITHUB_WORKFLOW_REF` the policy's workflow on it; the SHA-256 of each coordinate equal to the policy's digest; the namespace to exist (`DescribeNamespace`); no Nexus endpoint read, which needs cluster admin: the credential is a namespace writer on the canary namespace only, and the Run's own public observation of the handler's reply proves the route; the pinned Case and catalog as .2's `casebinding` checks them, whose `PreparedCase` preflight returns for the controller to run. It returns a `Scope` holding only digests, the invocation ID and the `PreparedCase`, or a named refusal. A test harness hook supplies the environment; nothing reads a file or flag for a credential.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/authority/ ./tools/canary/preflight/`

**Files:** `tools/canary/authority/**`, `tools/canary/preflight/**`
**Touches:** `tools/canary/authority/**`, `tools/canary/preflight/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Any ref, event, workflow, coordinate digest, namespace, queue, endpoint route, Case or catalog mismatch performs no mutation and creates no Run or receipt; each refusal is named.
- [ ] Credentials and raw coordinates never reach a receipt, provenance document, summary, progress line or log: the Redactor is applied to every written text and a test plants each credential and coordinate and finds none; recorded Runs, which hold whole history events, are never uploaded (.5, .9).
- [ ] Preflight proves the exact canary scope and claims nothing about the rest of production.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
