# Temporal Testpilot functional fixtures

This package owns the retained generated functional fixtures in `testdata`, plus their fixture
admission and prepared-Case reuse tests. The fixtures remain canonical ProtoJSON generated from
`Temporal.Testpilot`, stored indented for review -- two spaces and one trailing newline -- so a Case
change reads as a line diff; `umpire-gen-case-runtime-conformance` continues to write its functional
output to this package and is the only writer of that form.

Cluster provisioning, namespace and Nexus endpoint creation, SDK client ownership, environment
configuration, assertions, and cleanup registration remain under `tests/`. The reusable composite
Driver and its implementation-focused tests live in `common/testing/testpilot/temporal`.

The async Nexus fixture is one canonical Case 1.0 artifact with symbolic resource declarations.
Fixture tests prepare its unchanged bytes against two physical Profiles, confirm distinct binding
identities, and reject missing or inconsistent references before dispatch. The tagged live test adds
two isolated namespaces, queues, and named Nexus routes and verifies both Runs satisfy the same
Contract with correlated history evidence.

`derive_profile_test.go` holds the derivation oracle: the hand-written `AsyncNexusProfile`,
`TypedNexusProfile` and `TypedUnaryProfile` are compared field for field against
`temporal.DeriveProfile` over the same fixture bytes, including the identity the binding supplies.
Live tests no longer hand-write a Profile at all. `bindCase` under `tests/` takes a decoded Case and
an explicit `CaseBinding` (identity, namespace, task queue, Nexus endpoint, and whether this test
creates the endpoint), derives the Profile, provisions, prepares, and returns the bound Case;
`runCase` is the single-shot wrapper that loads a fixture by name, binds it, runs once, and fails
the test on a Run error. Tests that vary bindings, run concurrently, or deliberately omit a
resource call `bindCase` directly. Both are test helpers outside the public facade, so MOD-12's
`Prepare` then `Run` sequence is unchanged.

The worker-outage fixture is the fault Case: its controller stops the SDK worker of its own
activation queue before starting the workflow, resumes it after, and reads the closing history event
back. Its Contract carries the checked-in `rule_events` deadline -- the outage window is counted in
what the Run recorded, never on the host's clock -- and a safety rule over the completed workflow,
so the Run proves the queued task survived the outage. `worker_outage_artifact_test.go` prepares its
unchanged bytes and pins that bound offline; the tagged live tests run it, and run it beside a plain
Nexus Case on a *different* queue, because a pooled peer worker on the same physical queue would
keep polling through the outage.

Admission is checked once for every fixture rather than once per Case: `fixture_table_test.go`
enumerates `testdata/*-case.json`, decodes each strictly, pins its identity, and -- where
`DeriveProfile` can read the Case's Profile -- prepares it over unchanged bytes and rejects a
mutated role. The per-Case tests beside it keep what that table cannot say: the outage Deadline, the
typed tenfold load, Run isolation, and the checked Provenance the async Nexus Case carries.
