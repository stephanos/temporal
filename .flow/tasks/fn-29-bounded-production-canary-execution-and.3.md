---
satisfies: [R3, R10]
---

# fn-29-bounded-production-canary-execution-and.3 Acquire authority from the protected workflow and preflight the exact scope

## Description
Add `tools/canary/authority`: read the credentials from the environment only (`UMPIRE_CANARY_TLS_CERT`, `UMPIRE_CANARY_TLS_KEY`, `UMPIRE_CANARY_API_KEY`, and the coordinates `UMPIRE_CANARY_GRPC`, `UMPIRE_CANARY_NAMESPACE`, `UMPIRE_CANARY_TASK_QUEUE`, `UMPIRE_CANARY_HANDLER_QUEUE`, `UMPIRE_CANARY_ENDPOINT`), require at least one credential (a TLS certificate and key, or an API key) and build a `Transport` value (gRPC transport and per-RPC credentials for the Driver's server endpoint and the SDK client's dial options) -- the untagged binary's only transport source, which never yields plaintext -- and a `Redactor` that removes every credential and raw coordinate from any text before it is written. Add `tools/canary/preflight`: before any mutation, require `GITHUB_EVENT_NAME=workflow_dispatch`, `GITHUB_REPOSITORY` the policy's repository, `GITHUB_REF` the policy's trusted ref and `GITHUB_WORKFLOW_REF` exactly `<repository>/<workflow path>@<ref>`; the policy configured (a coordinate still `unconfigured` refuses as `policy-unconfigured`) and the SHA-256 of each coordinate equal to the policy's digest; the namespace to exist (`DescribeNamespace`); no Nexus endpoint read, which needs cluster admin: the credential is a namespace writer on the canary namespace only, and the Run's own public observation of the handler's reply proves the route; the pinned Case and catalog as .2's `casebinding` checks them, whose `PreparedCase` preflight returns for the controller to run. It returns a `Scope` holding only digests, the invocation ID and the `PreparedCase`, or a named refusal. A test harness hook supplies the environment; nothing reads a file or flag for a credential.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/authority/ ./tools/canary/preflight/`

**Files:** `tools/canary/authority/**`, `tools/canary/preflight/**`
**Touches:** `tools/canary/authority/**`, `tools/canary/preflight/**`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Any ref, event, workflow, coordinate digest, namespace, Case or catalog mismatch performs no mutation and creates no Run or receipt; each refusal is named. The endpoint's route is not read (it needs cluster admin): it is an operator precondition the Run itself proves, a repointed endpoint yielding a Run that is not accepted.
- [x] Credentials and raw coordinates never reach a receipt, provenance document, summary, progress line or log: the Redactor is applied to every written text and a test plants each credential and coordinate and finds none; recorded Runs, which hold whole history events, are never uploaded (.5, .9).
- [x] Preflight proves the exact canary scope and claims nothing about the rest of production.

## Done summary
`tools/canary/authority` reads `UMPIRE_CANARY_*` coordinates and the credential (a TLS pair, an API key, or both) only through an injected lookup, refuses a half pair, an unreadable pair or no credential by variable name, and builds a TLS-only `Transport` for the Driver's endpoint (with the namespace header) and the SDK client. Its `Redactor` removes every credential and coordinate, including PEM body lines and the target's host, through `Redact`, a bounded line-buffered `Writer`, and an SDK `Logger` at warning level; a test plants each value in every form and finds none. `tools/canary/preflight.Check` refuses by name (`workflow-context`, `policy-unconfigured`, `coordinate-mismatch`, `case-mismatch`, `namespace-missing`, `namespace-unavailable`), deciding every connection-free check first and then making one `DescribeNamespace` read through an interface with no mutating method; every refusal detail is redacted. It returns a `Scope` of the invocation ID, the coordinate digests and the prepared Case. Implementation review: SHIP in one round; its two P3 notes and four FYIs applied.
## Evidence
- Commits: 9d7a3ea300ad6db015b1a2d06f425eaf4eb6a7e0, 6aa00cc1a9a7928dfe9ff9b3ef2c9c7e47d04661
- Tests: go test -count=1 -tags test_dep ./tools/canary/..., GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: