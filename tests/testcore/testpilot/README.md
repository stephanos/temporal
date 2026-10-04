# Temporal Testpilot functional fixtures

This package owns the functional fixtures in `testdata`, plus their fixture admission and
prepared-Case reuse tests. None is written by hand. `testdata/generated` holds the Cases lowered
from the Scala model that the functional tests pin, with the manifest naming each one's Query:
`make umpire-gen-fixtures` publishes the tree and `make umpire-check-fixtures` checks it, and every
Case in it is byte for byte the one under `model/cases`, in compact canonical ProtoJSON. The Cases
beside that tree are retained as they were rendered, indented for review, because no Scala Model
declares their Query yet: the Nexus pair, System Info, workflow start, the worker outage and the
synthetic payload fixture.

`TestTestpilotGeneratedCases` runs every lowered Case under `model/cases` as
`TestTestpilotGeneratedCases/<model>-<query>`, the depth-2 name the functional job shards and the
salt optimizer times; `GeneratedCaseName` derives it as the Case file's stem. `generated_names_test.go`
holds the name contract: every name is unique and free of `/` and whitespace, and the sorted set
equals `testdata/generated-case-names.txt`, which a lowered Query added, renamed or removed rewrites
in the same change with `UMPIRE_CASE_NAME_GOLDENS=write`. A Case whose Program schedules a
workflow Nexus operation runs once per value of the Nexus implementation switch (`switch.go`): `hsm`
and `chasm` each set the six keys the upstream Nexus workflow suite sets, the CHASM rollout percent
(0 or 100) included. Every other Case, a standalone Nexus operation's included, runs once. A
cluster's settings name each key once: a key two sources give different values refuses the Case,
naming both.

Cluster provisioning, namespace and Nexus endpoint creation, SDK client ownership, environment
configuration, assertions, and cleanup registration remain under `tests/`. The reusable composite
Driver and its implementation-focused tests live in `common/testing/testpilot/temporal`.

The caller Model (`model/temporal/nexuscaller`) lowers one fixture per Query,
`generated/nexus-caller-<query>-case.json`: sync success, async reply then succeeded
callback, async reply then failed callback, a non-retryable handler error, a retryable handler error
then success after one backoff, a schedule-to-start timeout with the handler's worker stopped, and a
start-to-close timeout after an asynchronous reply. Each is one canonical Case 1.0 artifact with
symbolic resource references. The canary's pinned Case is not here: `make canary-gen-case` publishes
it under `tools/canary/casebinding/testdata`. Fixture tests prepare the async-completion
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
`runCapturedCase` is the single-shot wrapper that loads a fixture by name, binds it, runs once,
captures the closed Run when `UMPIRE_REPEAT_RUN_DIR` is set, and fails the test on a Run error;
`runCapturedCaseWithBinding` does the same over a binding the caller chose. Tests that run
concurrently or deliberately omit a resource call `bindCase` directly. Both are test helpers outside the public facade, so MOD-12's
`Prepare` then `Run` sequence is unchanged.

The worker-outage fixture (`workerOutageTests-survived`) is the fault Case, one of the retained
fixtures: its controller stops the SDK worker of its own activation queue before starting
the workflow, resumes it after, and waits for the workflow the resumed worker completes. Its
Contract carries the outage-order rule over the two fault actions --
bounded liveness with a `rule_events` deadline, so the outage window is counted in what the Run
recorded, never on the host's clock -- beside the correlated capability confirming the path's
steps from the completed event, so the Run proves the queued task survived the outage.
`worker_outage_artifact_test.go` prepares its unchanged bytes and pins that bound offline; the
tagged live tests run it, and run it beside a plain Nexus Case on a *different* queue, because a
pooled peer worker on the same physical queue would keep polling through the outage. The
system-info fixture (`systemInfoTests-answered`) is the unary Case, also retained: one
`GetSystemInfo` call, no workflow, and the instruction's completion as its evidence.

A generated fixture is named by its Model and Query: the Case ID is
`temporal.case.scala.<model>.<query>` and the file is `generated/<model>-<query>-case.json`, where
`<model>` is the IR file's name without `.json`. The Queries this tree pins are listed under the
`functional` kind in `tools/umpire/cmd/umpire-gen-cases/main.go`, and `generated/manifest.json` records
each one's standing and expected assessment. A Query is added by adding it to that list and running
`make umpire-gen-fixtures`. The retained fixtures keep the names they were rendered under,
`<set>-<query>-case.json` with the Case ID `temporal.case.<set>.<query>`. A Case whose path realizes
a class with a declared example carries an abstraction claim row in its provenance naming the
action, the field, the class and the example it ran.

Admission is checked once for every fixture rather than once per Case: `fixture_table_test.go`
enumerates `testdata/*-case.json` and `testdata/generated/*-case.json`, decodes each strictly, pins its identity, and -- where
`DeriveProfile` can read the Case's Profile -- prepares it over unchanged bytes and rejects a
mutated role. The per-Case tests beside it keep what that table cannot say: the outage Deadline, the
typed tenfold load, Run isolation, and the checked Provenance the async Nexus Case carries.
