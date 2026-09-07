---
satisfies: [R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.1 Remove Nexus knowledge from the server transport

## Description
Remove Nexus feature knowledge from `common/testing/testpilot/temporal/server`. That package must remain a generic controller-side transport: it supplies the authorized descriptor catalog, invokes prepared unary RPC method/request pairs, and returns raw typed responses plus protocol status.

Delete the Nexus-named server file and API surface, including Nexus-specific completion types and methods. Express any required callback or opaque-capability operation through generic transport contracts. The Lean model owns Nexus semantics; `temporal/worker` may use the Go SDK's Nexus APIs where SDK execution requires them, but the generic Testpilot server transport must not contain Nexus identifiers, imports, branching, or feature-specific tests.

Update composite Temporal Driver wiring, ownership documentation, and boundary regression coverage while preserving the existing admitted Case behavior and authorization checks.

## Acceptance
- [ ] `common/testing/testpilot/temporal/server` contains no Nexus-named files, exported APIs, internal identifiers, imports, branches, or feature-specific tests.
- [ ] The server package exposes only generic descriptor-catalog and prepared unary-RPC transport behavior, including raw typed responses and protocol status.
- [ ] Any unavoidable Go SDK Nexus mechanics live in `common/testing/testpilot/temporal/worker`; Nexus semantics and lowering remain owned by the Lean model.
- [ ] The composite Temporal Driver uses a generic opaque-capability or callback transport contract without teaching Testpilot server code what Nexus means.
- [ ] Focused server, SDK worker, composite Driver, dependency-boundary, and existing Case integration tests pass with `-tags test_dep`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
