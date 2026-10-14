---
satisfies:
  - R1
  - R2
  - R3
  - R5
  - R6
  - R7
---

# fn-73-explicit-environment-binding-for.8 Collapse symbolic bindings into Case 1.0 and retire the legacy Driver resource mode

## Description
Amend fn-73 and its normative Umpire/Testpilot documentation so exact Case 1.0 is the only admitted and generated format. Migrate every resource-bearing Lean producer, generated fixture, Go fixture, and test helper to symbolic bindings. Remove the shared Temporal Driver's legacy physical-resource mode and its namespace, task-queue, and Nexus-endpoint fallback configuration; Profile-owned symbolic bindings become the sole source of physical resource names. Transport connections, credentials, SDK clients, callback authority, HTTP clients, and lifecycle settings remain explicit Driver inputs.

Case 1.0 may have an empty environment only when the Program uses no physical resource. Every resource-bearing role and request field must use a declared symbolic binding, and preparation must reject incomplete, unused, crossed, or literal physical-resource configuration before Driver I/O. Retain a focused negative compatibility test proving Case 1.1 is unsupported; retain no legacy literal-resource path.

## Acceptance
- The fn-73 spec, UMPIRE4 normative documents, package documentation, and roadmap describe exact Case 1.0 as the sole supported format and contain no requirement to preserve a legacy Driver mode.
- All checked Lean producers, canonical ProtoJSON fixtures, conformance fixtures, and Go test constructors emit exact Case 1.0. Fixture generation and staleness checks pass.
- Prepare accepts exact Case 1.0 only and rejects Case 1.1 and every other version before Driver validation, Open, worker registration, or target effects.
- A resource-free Case 1.0 Program may declare no environment bindings. Any Program using namespaces, task queues, named Nexus endpoints, or resource-bearing request fields declares a complete closed symbolic binding graph and obtains physical values only from the immutable Profile snapshot.
- The shared Temporal Driver exposes no legacy namespace, task-queue, or Nexus-endpoint resource mode, fallback, or mixed-mode branch. Existing transport and lifecycle inputs remain explicit.
- Focused Testpilot and Temporal Driver suites, Lean authoring/codec checks, deterministic fixture checks, the two-environment live Nexus3 selector, make lint-model, and make lint-code pass. The protected workflow remains unchanged.


## Done summary
Collapsed symbolic environment binding into exact Case 1.0 as a deliberate breaking semantic change. Resource-free Programs may omit environment declarations; every physical Temporal resource is now declared and resolved through the immutable Profile binding snapshot. Removed the shared Temporal Driver's legacy namespace, task-queue and named Nexus-endpoint options and all fallback/mixed-mode branches. Migrated generic fixtures, worker tests, the checked Nexus3 and synthetic Lean producers, generated artifacts, and active architecture documentation. Reviews found and verified fixes for controller-only and cleanup Temporal RPC validation so binding checks always run before Open while SDK worker registration remains conditional. Case 1.1 is retained only as an explicit unsupported-version regression. No dependency, proof axiom, workflow, or generic Nexus-operation change was introduced. Full `make lint-code` retains the known 1,279 inherited findings; focused Testpilot lint reports zero issues.
## Evidence
- Commits: b03db676fe01203af056bd7e3cf105a35735082f
- Tests: TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot, TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./common/testing/testpilot/internal/execution ./common/testing/testpilot ./common/testing/testpilot/temporal/worker ./common/testing/testpilot/temporal, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-case-runtime-conformance, TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 mise exec -- go test -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase$', make lint-model, TMPDIR=/private/tmp CGO_ENABLED=0 GOFLAGS=-p=1 .bin/golangci-lint-v2.13.1 run --build-tags 'disable_grpc_modules,,test_dep,' --timeout 10m --fix=false --config=.github/.golangci.yml ./common/testing/testpilot/..., make lint-code, git diff --check, git diff --cached --check
- PRs:
