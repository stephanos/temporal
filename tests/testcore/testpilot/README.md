# Temporal Testpilot functional fixtures

This package owns the retained generated functional fixtures in `testdata`, plus their fixture
admission and prepared-Case reuse tests. Every fixture is canonical ProtoJSON produced from a Model
file's `case … realizes` block under `model/Temporal/Feature` and rendered by `umpire-case`; none is
written by hand. It is stored indented for review -- two spaces and one trailing newline -- so a Case
change reads as a line diff; `umpire-gen-case-runtime-conformance` continues to write its functional
output to this package and is the only writer of that form.

Cluster provisioning, namespace and Nexus endpoint creation, SDK client ownership, environment
configuration, assertions, and cleanup registration remain under `tests/`. The reusable composite
Driver and its implementation-focused tests live in `common/testing/testpilot/temporal`.

The caller Model's functional set (`model/Temporal/Feature/Nexus/Caller/Model.lean`) produces one
fixture per Query, `nexusCallerTests-<query>-case.json`: sync success, async reply then succeeded
callback, async reply then failed callback, a non-retryable handler error, a retryable handler error
then success after one backoff, a schedule-to-start timeout with the handler's worker stopped, and a
start-to-close timeout after an asynchronous reply. Each is one canonical Case 1.0 artifact with
symbolic resource references; the Model's canary set produces no fixture, and its exploratory set's
coverage targets are a golden under `model/`. Fixture tests prepare the async-completion
fixture's unchanged bytes against two physical Profiles, confirm distinct binding identities, and
reject missing or inconsistent references before dispatch. The tagged live tests run each Query once
per value of the implementation switch, under two isolated namespaces, queues and named Nexus routes
each, and verify every Run satisfies the same Contract with correlated history evidence.

`derive_profile_test.go` holds the derivation oracle: the hand-written `NexusCallerProfile` and
`WorkflowStartProfile` are compared field for field against `temporal.DeriveProfile` over the same
fixture bytes, including the identity the binding supplies; `NexusPairProfile` prepares the pair
Case offline with two handler reservations.
Live tests no longer hand-write a Profile at all. `bindCase` under `tests/` takes a decoded Case and
an explicit `CaseBinding` (identity, namespace, task queue, Nexus endpoint, and whether this test
creates the endpoint), derives the Profile, provisions, prepares, and returns the bound Case;
`runCase` is the single-shot wrapper that loads a fixture by name, binds it, runs once, and fails
the test on a Run error. Tests that vary bindings, run concurrently, or deliberately omit a
resource call `bindCase` directly. Both are test helpers outside the public facade, so MOD-12's
`Prepare` then `Run` sequence is unchanged.

The worker-outage fixture (`workerOutageTests-survived`) is the fault Case, produced from the
worker-outage Model: its controller stops the SDK worker of its own activation queue before starting
the workflow, resumes it after, and waits for the workflow the resumed worker completes. Its
Contract carries the outage-order rule the Producer derives from the Model's two fault actions --
bounded liveness with a `rule_events` deadline, so the outage window is counted in what the Run
recorded, never on the host's clock -- beside the correlated capability confirming the Model's
steps from the completed event, so the Run proves the queued task survived the outage.
`worker_outage_artifact_test.go` prepares its unchanged bytes and pins that bound offline; the
tagged live tests run it, and run it beside a plain Nexus Case on a *different* queue, because a
pooled peer worker on the same physical queue would keep polling through the outage. The
system-info fixture (`systemInfoTests-answered`) is the unary Case, produced from the system-info
Model: one `GetSystemInfo` call, no workflow, and the instruction's completion as its evidence.

A functional set's Cases are named by what they are: a Model file's `set` lists its `find` Queries
and a `case` block over the set realizes each of them, with the Case ID `temporal.case.<set>.<query>`
and the fixture `<set>-<query>-case.json`. `umpire-case --list` enumerates every registered Case --
each set's Queries and the Cases that register their values explicitly -- sorted by Case ID, and the
generator renders exactly that list, so a Query added to a set is a fixture the moment the generator
runs. A Case whose path realizes a class with an `examples:` line carries an abstraction claim row in
its provenance naming the action, the field, the class and the example it ran.

Admission is checked once for every fixture rather than once per Case: `fixture_table_test.go`
enumerates `testdata/*-case.json`, decodes each strictly, pins its identity, and -- where
`DeriveProfile` can read the Case's Profile -- prepares it over unchanged bytes and rejects a
mutated role. The per-Case tests beside it keep what that table cannot say: the outage Deadline, the
typed tenfold load, Run isolation, and the checked Provenance the async Nexus Case carries.
