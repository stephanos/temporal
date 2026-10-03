# Shard generated Cases per Case in CI

## Goal & Context
<!-- scope: business -->

The generated Cases are the Model's product: every lowered Query is one Case file, and `TestTestpilotGeneratedCases` runs each against an in-process cluster. Today that test runs only in the Umpire workflow's live gate, unsharded, under one 30-minute budget, and it is shaped as two subtests named after the Nexus implementation switch (`hsm`, `chasm`), each running every Case, including the eight standalone activity Cases that have no Nexus in them.

The functional CI job shards tests across five runners by hashing a test's name with a salt, and a daily optimizer re-balances the salt from the recorded durations of depth-2 test names. So the unit CI can balance is the depth-2 name, and the unit must be stable across runs. For the generated Cases the natural unit is the Case: one `<model>-<query>` per lowered Query, with the Nexus implementation switch below it only where the Case exercises Nexus.

This spec gives each generated Case its own depth-2 subtest and its own dedicated cluster, runs the HSM and CHASM values only for Cases that bind a Nexus endpoint, keeps the per-Case agreement check between the two values meaningful, puts the generated Cases into the sharded functional job while the Umpire live gate keeps running them, and pins the derived names so a renamed Query fails loudly instead of silently moving a shard unit. It changes no Case byte and no schema.

## Architecture & Data Models
<!-- scope: technical -->

**The shard unit is the depth-2 test name, by construction.** The functional sharding check hashes the first two segments of the running test's name (`Top/Sub`), not the full name, so a cluster constructed in a deeper subtest lands in the shard of its depth-2 ancestor. That is the unit the salt optimizer already aggregates; the check and the optimizer agree for every functional test, not only this one. A depth-1 name hashes as itself, as today. The check is also callable from a subtest that constructs no cluster, so a depth-2 unit whose clusters live in its children can skip as a whole when it is not in the running shard. The depth-2 rule is stated in two places, the optimizer's aggregation level and the check's prefix; each site's comment names the other.

**One subtest per lowered Case, named after it.** `TestTestpilotGeneratedCases/<name>` where `<name>` is derived from the manifest entry: the Case file's stem (`<model>-<query>`, the file name without `-case.json`). The derivation is one exported function over a manifest entry, used by the functional test and by the name-contract test. Each Case subtest constructs its own dedicated in-memory cluster; nothing is shared across Cases.

**The Nexus implementation switch is per Case, derived from what the Case binds.** A Case that declares an endpoint role with a resource binding (the condition the live tests already use to decide whether to create a Nexus endpoint) runs under `<name>/hsm` and `<name>/chasm`, each with its own cluster constructed under that value's settings, CHASM for standalone activity enabled in both. A Case that binds no endpoint runs once, directly under `<name>`, with CHASM enabled and no switch subtest. The switch is declared by the functional tests; no Model, IR or Case declares it, and no schema is added for it.

**Agreement across the switch.** Every Case subtest runs the sharding check first, so a Case outside the running shard skips as a whole before any value subtest starts. For a Nexus Case in the shard, each value subtest constructs its cluster (the check agrees through the prefix), runs, and hands its Verdict up; after both, the Case subtest runs the existing agreement check over the Verdicts collected and fails on a divergence. A value whose preparation the Profile rejects fails inside that value's subtest, naming the value, rather than skipping: one implementation running and the other not is a finding, not an agreement. The Case subtest counts nothing itself, so a retry of one value by the functional test runner, or a developer's `-run` on one value, still passes or fails on that value alone.

**Build tags.** The generated-Case test and the helpers it needs build under `test_dep` alone; the hand-written Testpilot suites keep `test_dep && integration`. The sharded functional job builds `./tests` with `test_dep`, so it picks up the generated Cases and nothing else from Testpilot; the Umpire live gate builds with both tags and the selector `^TestTestpilot`, so it keeps running them too.

**Name contract.** A test with no cluster pins the derived names: for every lowered manifest entry the name equals the Case file's stem, names are unique, and none carries a `/` or whitespace (a `/` would add a depth, whitespace is rewritten by the test runner). The sorted list is compared with a committed golden rewritten through the repository's goldens convention (an environment variable set to `write`).

## Edge Cases & Constraints
<!-- scope: technical -->

- **No Case byte changes.** This spec runs after fn-112 task 10 closes the structural Case freeze and must not change any Case, manifest, IR or fixture byte; the Case check target proves it. The tasks state this cross-spec gate because flowctl records dependencies within one spec only.
- **Shard placement moves for some tests.** Hashing the depth-2 prefix changes the shard only of tests that construct a cluster at depth 3 or deeper; the daily optimizer re-balances the salt within a day. The salt file is not edited by hand here.
- **A rename is a shard-unit rename.** The optimizer has no duration for the new name until it has run once; the name golden makes the rename a reviewed change rather than a silent one.
- **Preparation rejected under one value.** Today a Case the Profile rejects as unsupported is skipped. Under the new shape a Nexus value subtest whose preparation is rejected fails, naming the value; a non-Nexus Case keeps today's skip, since no agreement depends on it. The Case subtest never counts Verdicts: an out-of-shard Case has already skipped as a whole, and a single value re-run by the test runner's per-leaf retry (anchored to the leaf name, so the sibling value is filtered out) is judged on its own.
- **Live gate headroom.** `make umpire-check-live-tests` runs the whole `^TestTestpilot` set unsharded under a fixed 30-minute `go test` timeout; the generated test's cluster count there rises from 2 to 24. R7's measurement reports the gate's wall-clock so the headroom is known before the shape is kept.
- **Cost.** Sixteen lowered Cases today: eight standalone activity, eight Nexus. The old shape constructed two clusters and ran 128 Runs; the new shape constructs 24 clusters (eight activity, sixteen Nexus) and runs 96 Runs (two bindings, two rounds each, as today), spread over five shards and retried by the functional test runner on failure. The done summary records the measured wall-clock before and after.
- **Persistence.** The generated-Case clusters use in-memory SQLite whatever persistence the functional job's matrix selects, so the test's behavior does not vary across the job's database variants; it runs in each of them as any other dedicated-cluster test does.
- **Concurrent edits.** fn-119 tasks 4 and 6 also edit the generated-Case test (a per-Case log line, a Case selector for the example). Whichever lands second rebases; neither changes the name shape this spec fixes.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Every lowered manifest Case runs as `TestTestpilotGeneratedCases/<name>`, `<name>` derived from the manifest entry as the Case file's stem, with its own dedicated cluster constructed inside that subtest. Errors: a manifest entry that is not lowered produces no subtest; a manifest that cannot be read fails the test before any cluster is constructed.
- **R2:** A Case that binds a Nexus endpoint runs under `<name>/hsm` and `<name>/chasm`, each with its own cluster under that value's settings, and the Case subtest fails with the existing agreement check's divergence message when the two Verdicts differ; a Case that binds no endpoint runs once under `<name>` with CHASM enabled and no switch subtest. Errors: a divergence names the switch, both values and both Verdicts; a Nexus value whose preparation is rejected fails that value's subtest naming the value, and the agreement check runs over the Verdicts collected; a non-Nexus Case rejected at preparation skips as today.
- **R3:** The functional sharding check hashes the depth-2 prefix of the test name and is callable from a subtest that constructs no cluster; every Case subtest calls it first, so with `TEST_TOTAL_SHARDS` and `TEST_SHARD_INDEX` set each `<name>` unit runs in exactly one shard, skips as a whole elsewhere, and both of its switch subtests run in that shard; a unit test proves a depth-3 subtest is placed with its depth-2 ancestor. Errors: malformed shard variables fail as today; a depth-1 name hashes as itself, unchanged; the sharded run of the generated test is green in every shard.
- **R4:** A name-contract test with no cluster asserts, for every lowered manifest entry, that the derived name equals the Case file's stem, that names are unique and carry no `/` or whitespace, and that the sorted list equals a committed golden; a renamed, added or removed lowered Case fails it until the golden is rewritten through the goldens convention. Errors: a manifest that cannot be decoded fails the test; an entry that is not lowered is absent from the golden.
- **R5:** The generated-Case test and the helpers it needs build under `test_dep` alone, so `make functional-test-coverage` runs it sharded, and `make umpire-check-live-tests` still runs it under `test_dep integration` with the `^TestTestpilot` selector. Errors: the hand-written Testpilot suites keep `test_dep && integration` and do not enter the functional job; a `test_dep`-only build of `./tests` compiles.
- **R6:** The comment on the Nexus implementation switch name states that the functional tests declare the switch and its value names, and that no Model, IR or Case declares it or depends on it (no error surface beyond the comment).
- **R7:** No Case, manifest, IR or fixture byte changes; the done summary cites the Case check target, the before and after wall-clock of the generated-Case test unsharded and per shard with five shards locally, and the cluster and Run counts. Errors: a changed byte stops the task.

## Boundaries
<!-- scope: business -->

- No schema change: no `repeat:` or switch declaration in a Model, IR or Case, and no IR field for the switch.
- No sharding of the hand-written Testpilot suites; they stay in the Umpire live gate only.
- No behavior change to the salt optimizer (a cross-reference comment at its aggregation level is allowed), and no change to the salt file, the shard count or the functional job's database matrix.
- No removal or weakening of `make umpire-check-live-tests`; it remains the Umpire gate.
- No change to how a Case is run, bound, assessed or recorded; the per-Case log line and example selector belong to fn-119.
- No generated API drift verification or new CI workflow (declined concept; the sharded run uses the existing functional job).

## Decision Context
<!-- scope: both — conditionally substructured -->

The owner's direction on 2026-10-03: the HSM/CHASM distinction is not a top-level concern but a per-Query one; the shard unit is `TestTestpilotGeneratedCases/<model>-<query>`, with the switch below it only for Cases that exercise the Nexus implementation, derived from what the Case binds rather than from a schema change.

**Depth-2 prefix in the sharding check, not two clusters per Case.** The sharding check runs at cluster construction, and a Nexus Case needs one cluster per switch value. Constructing both at depth 2 would satisfy the shard unit without touching the shared check, at the cost of two in-process clusters alive at once per Nexus Case and an unresolved question of two dedicated clusters in one test. Hashing the depth-2 prefix instead makes the check and the optimizer agree by construction for every functional test, and the generated test constructs one cluster at a time. Rejected a per-test shard key option as more surface for the same effect.

**One cluster per Case, not one per shard.** A cluster constructed above the Case subtests would hash the top-level name and make the whole test one unit; per-Case clusters are what sharding needs, and they also isolate Cases from each other, which fn-90's history-interleaving flake argues for.

**Drop `integration` on the generated test, not `TEST_TAG=integration` on the job.** Setting the tag on the functional job would pull all seventeen Testpilot files into every database variant of the sharded job, including the per-Query Nexus suites that duplicate the generated Nexus Cases and the suites with fn-90's flake history, and `TEST_TAG` is a Makefile-wide knob that also reaches the XDC and NDC jobs. Moving the few helpers the generated test uses into a `test_dep`-only file is the smaller change, and the live gate's tag set still includes it.

**Nexus by binding.** The endpoint-binding condition already decides whether a live test creates a Nexus endpoint; using it for the switch keeps Case bytes untouched and needs no declaration anywhere. The comment claiming the realization declares the switch via `repeat:` dates from the Lean era; no Scala Model, IR or Case declares it.

**A golden for the names.** The names are derived, so they cannot drift from the manifest; what can happen silently is a Query rename moving a shard unit and losing its duration history. The golden turns that into a reviewed diff.

**Cross-spec gate stated in tasks, no spec dependency.** The gate is fn-112 task 10 (structural Case freeze), not the fn-112 spec; a spec dependency would wait for fn-112 tasks 11 and 12 too. The same pattern as fn-118's entry gates.

**Declined ledger.** `.flow/memory/declined/generated-api-drift-verification.md` declines generated API drift verification and new CI coverage. This spec adds neither: it moves an existing test into the existing sharded job at the owner's direction, and the name golden checks test names, not generated API output. The ledger's prior requests record this.

**Runtime.** Cluster construction rises from 2 to 24 while Runs fall from 128 to 96; the cost is the price of a per-Case unit and is spread over five shards. The measurement in R7 decides whether the shape needs a second look.

## Quick commands

```bash
go test -count=1 -tags test_dep ./tests/testcore/testpilot/... ./tests/testcore/ -run 'Shard|GeneratedCaseName'
# Until task 3 moves the helpers the generated test builds only with both tags; after it, `-tags test_dep` alone also works.
TEST_TOTAL_SHARDS=5 TEST_SHARD_INDEX=0 go test -count=1 -v -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases'
make umpire-check-cases && make umpire-check-live-tests
```

## Early proof point

Task fn-121-shard-generated-cases-per-case-in-ci.1 validates the core approach (per-Case dedicated clusters with the switch below the Case, placed by the depth-2 prefix, pass unsharded and under five local shards with the agreement check intact). If it fails, re-evaluate whether the sharding check can hash the depth-2 prefix for every functional test before continuing with fn-121-shard-generated-cases-per-case-in-ci.2+.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Every lowered manifest Case runs as `TestTestpilotGeneratedCases/<name>`, `<name>` derived from the manifest entry as the Case file's stem, with its own dedicated cluster constructed inside that subtest. Errors: a manifest entry that is not lowered produces no subtest; a manifest that cannot be read fails the test before any cluster is constructed. | fn-121-shard-generated-cases-per-case-in-ci.1 | — |
| R2 | A Case that binds a Nexus endpoint runs under `<name>/hsm` and `<name>/chasm`, each with its own cluster under that value's settings, and the Case subtest fails with the existing agreement check's divergence message when the two Verdicts differ; a Case that binds no endpoint runs once under `<name>` with CHASM enabled and no switch subtest. Errors: a divergence names the switch, both values and both Verdicts; a Nexus value whose preparation is rejected fails that value's subtest naming the value, and the agreement check runs over the Verdicts collected; a non-Nexus Case rejected at preparation skips as today. | fn-121-shard-generated-cases-per-case-in-ci.1 | — |
| R3 | The functional sharding check hashes the depth-2 prefix of the test name and is callable from a subtest that constructs no cluster; every Case subtest calls it first, so with `TEST_TOTAL_SHARDS` and `TEST_SHARD_INDEX` set each `<name>` unit runs in exactly one shard, skips as a whole elsewhere, and both of its switch subtests run in that shard; a unit test proves a depth-3 subtest is placed with its depth-2 ancestor. Errors: malformed shard variables fail as today; a depth-1 name hashes as itself, unchanged; the sharded run of the generated test is green in every shard. | fn-121-shard-generated-cases-per-case-in-ci.1 | — |
| R4 | A name-contract test with no cluster asserts, for every lowered manifest entry, that the derived name equals the Case file's stem, that names are unique and carry no `/` or whitespace, and that the sorted list equals a committed golden; a renamed, added or removed lowered Case fails it until the golden is rewritten through the goldens convention. Errors: a manifest that cannot be decoded fails the test; an entry that is not lowered is absent from the golden. | fn-121-shard-generated-cases-per-case-in-ci.2 | — |
| R5 | The generated-Case test and the helpers it needs build under `test_dep` alone, so `make functional-test-coverage` runs it sharded, and `make umpire-check-live-tests` still runs it under `test_dep integration` with the `^TestTestpilot` selector. Errors: the hand-written Testpilot suites keep `test_dep && integration` and do not enter the functional job; a `test_dep`-only build of `./tests` compiles. | fn-121-shard-generated-cases-per-case-in-ci.3 | — |
| R6 | The comment on the Nexus implementation switch name states that the functional tests declare the switch and its value names, and that no Model, IR or Case declares it or depends on it (no error surface beyond the comment). | fn-121-shard-generated-cases-per-case-in-ci.1 | — |
| R7 | No Case, manifest, IR or fixture byte changes; the done summary cites the Case check target, the before and after wall-clock of the generated-Case test unsharded and per shard with five shards locally, and the cluster and Run counts. Errors: a changed byte stops the task. | fn-121-shard-generated-cases-per-case-in-ci.3 | — |
