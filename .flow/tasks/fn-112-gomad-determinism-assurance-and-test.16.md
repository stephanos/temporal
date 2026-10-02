# fn-112-gomad-determinism-assurance-and-test.16 Keep two retained successes with one outcome signature as distinct artifacts
## Description
Found while gating fn-114.16, with the fake executor only: a seed campaign that keeps two successes with the same outcome signature stores them as one artifact directory, counts both, and publishes a record that `OpenCampaign` rejects. Whether two real seeds can produce the same success signature was not checked. Belongs to R9 (end-to-end CLI tests).

**Size:** M
**Files:** `tools/gomad3/runner/retention.go`, `completion.go`, `tools/gomad3/artifact/store.go`, `tools/gomad3/runner/internal/campaign/open_campaign.go`, their tests
**Touches:** [tools/gomad3/runner/**, tools/gomad3/artifact/**, tools/gomad3/cmd/gomad/internal/cli/**]

### Approach
- Establish first whether it is reachable with real executions: with `--keep-successes=all`, two seeds of a target whose output does not depend on the seed. Use the built CLI. Retain the result either way.
- If reachable, decide the contract from the code and README: a retained success is "an immutable exact-replay artifact" per execution, and its identity includes the seed, so two seeds must not collapse. Either the store key must distinguish them, or retention must count one artifact once and the record must say so. Recorded identities of existing artifacts must not change for campaigns without a collision.
- If it is reachable only through the fake executor, make the fake produce identities the way the real executor does and add a test that pins why real seeds cannot collide.
- The published record must always open through `OpenCampaign`.

## Acceptance
- [ ] Reachability with real executions is shown by a retained CLI run
- [ ] A campaign keeping two successes with equal outcome signatures publishes a record `OpenCampaign` accepts, with counts that match the artifacts on disk, shown by a test
- [ ] Recorded identities for a campaign without a collision equal values retained before the change
- [ ] `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` and `make -C tools/gomad3 validate` pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
