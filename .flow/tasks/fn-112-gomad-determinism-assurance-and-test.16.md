# fn-112-gomad-determinism-assurance-and-test.16 Keep two retained successes with one outcome signature as distinct artifacts
## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Current-source full Darwin gates, semantic preservation/mapping, real built-CLI evidence and review. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Found while gating fn-114.16, with the fake executor only: a seed campaign that keeps two successes with the same outcome signature stores them as one artifact directory, counts both, and publishes a record that `OpenCampaign` rejects. Whether two real seeds can produce the same success signature was not checked. Belongs to R9 (end-to-end CLI tests).

**Size:** M
**Files:** `tools/gomad3/runner/retention.go`, `completion.go`, `tools/gomad3/artifact/store.go`, `tools/gomad3/runner/internal/campaign/open_campaign.go`, their tests, `tools/gomad3/cmd/gomad/retained_success_e2e_test.go`
**Touches:** [tools/gomad3/runner/**, tools/gomad3/artifact/**, tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/cmd/gomad/retained_success_e2e_test.go]

### Approach
- Establish first whether it is reachable with real executions: with `--keep-successes=all`, two seeds of a target whose output does not depend on the seed. Use the built CLI. Retain the result either way.
- If reachable, decide the contract from the code and README: a retained success is "an immutable exact-replay artifact" per execution, and its identity includes the seed, so two seeds must not collapse. Either the store key must distinguish them, or retention must count one artifact once and the record must say so. Recorded identities of existing artifacts must not change for campaigns without a collision.
- If it is reachable only through the fake executor, make the fake produce identities the way the real executor does and add a test that pins why real seeds cannot collide.
- The published record must always open through `OpenCampaign`.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Reachability with real executions is shown by a retained CLI run
- [ ] A campaign keeping two successes with equal outcome signatures publishes a record `OpenCampaign` accepts, with counts that match the artifacts on disk, shown by a test
- [ ] Recorded identities for a campaign without a collision equal values retained before the change
- [ ] `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` and `make -C tools/gomad3 validate` pass

## Done summary
Real built-CLI executions confirm the collision: seeds1–2 of fmt.Println("same output") retained one outcome-signature path twice and published a campaign inspect rejected for a seed mismatch. Store publication now preserves the signature path for the first success and falls back to full record identity for another successful execution with that signature. Exact-repeat publication is idempotent and failure deduplication is unchanged. No schema or hash projection changed.

Store regression pins the pre-change noncollision signature, path and record hash and retains three distinct seeds. Runner regression checks OpenCampaign, seeds, disk/journal counts and byte totals. The built-CLI regression explores and inspects the two-seed campaign and replays both successes; real-cli-fixed retains reproduced=true evidence for each. The original failing run, red/green tests and handover are retained here.

Native Darwin full test-host passes all45 packages; integrated validate and scoped vet pass. Shared evidence and source bindings are in fn-114/task-13/integrated-test-host-green.log and integrated-source-hashes.json. Independent review of the batch returned SHIP; its receipt is fn-114/task-13/integrated-review.json. Linux remains unverified and root lint cannot load nested-module paths. No implementation commits or pushes; the user owns commits.
## Evidence
- Commits:
- Tests: GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host, make -C tools/gomad3 validate, built CLI explore/inspect/replay of both same-signature successes
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Current-source full Darwin gates, semantic preservation/mapping, real built-CLI evidence and review. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
