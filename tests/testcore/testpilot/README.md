# Temporal Testpilot functional fixtures

This package owns the retained generated functional fixtures in `testdata`, plus their fixture
admission and prepared-Case reuse tests. The fixtures remain canonical ProtoJSON generated from
`Temporal.Testpilot`; `umpire-gen-case-runtime-conformance` continues to write its functional output
to this package.

Cluster provisioning, namespace and Nexus endpoint creation, SDK client ownership, environment
configuration, assertions, and cleanup registration remain under `tests/`. The reusable composite
Driver and its implementation-focused tests live in `common/testing/testpilot/temporal`.

The async Nexus fixture is one canonical Case 1.0 artifact with symbolic resource declarations.
Fixture tests prepare its unchanged bytes against two physical Profiles, confirm distinct binding
identities, and reject missing or inconsistent references before dispatch. The tagged live test adds
two isolated namespaces, queues, and named Nexus routes and verifies both Runs satisfy the same
Contract with correlated history evidence.
