# fn-112 task 16: two retained successes with one outcome signature

Host: linux/arm64 with the local development harness on (adds linux/arm64 to
the toolchain and boundary manifests, shims `syscall.Dup2`). This is
development evidence only, not darwin/arm64 or linux/amd64 evidence.
Base revision: bce040728 (gomad-next-b).

## Reachability with real executions (before the change)

Fixture `tools/gomad3/cmd/gomad/testdata/seedfree` prints the same line for
every seed. Built CLI (`make runner` -> `.bin/gomad`), run from `tools/gomad3`:

```sh
env -u GOROOT -u GOMADSEED -u GOFLAGS GOTOOLCHAIN=local GOWORK=off CGO_ENABLED=0 \
  .bin/gomad explore --json --artifacts $SCRATCH/artifacts --seeds 0,1 --parallel 1 \
  --execution-timeout 60s --overall-timeout 5m --keep-successes all \
  --success-limit 3 --success-bytes 256MiB go-run ./cmd/gomad/testdata/seedfree
.bin/gomad inspect --json $SCRATCH/artifacts/v1/campaign-*
```

- explore: exit 0, `retained_successes: 2`, both `artifact` events name one
  directory `successes/sha256-0d2e6feda49add0c252c1012cbaa6839`
  (`collision-before-explore.json`).
- One directory on disk, holding seed 0's manifest; seed 1's evidence is gone.
- inspect: exit 2, "validate published success artifact 2: retained success
  artifact does not match its campaign execution" (`collision-before-inspect.txt`).

So the defect is reachable with real seeds: the success store was keyed by
the outcome (failure) signature, whose projection excludes the seed and
`GOMADSEED`, and the store reused an existing artifact of equal signature.

## Contract chosen

A retained success is an exact-replay artifact per execution (README:
"Each retained success is an immutable exact-replay artifact"). New
`artifact.StoreKeyExecution`, used by the seed, choice-exploration and
simulation-exploration success stores:

- The first artifact of a signature keeps the old directory
  (`sha256-<first 32 hex of the signature>`), so campaigns without a
  collision record the same artifact references as before.
- An existing artifact is reused only for the same execution (campaign ID,
  selection ordinal, seed), as a resumed campaign that publishes it again.
- Another execution of that signature is stored under
  `sha256-<DomainHash("gomad3-execution-artifact-v1", campaign_id, ordinal, seed, signature)>`.

Failure stores are unchanged (distinct failures are still one per signature).

## After the change

- Same seedfree command: exit 0, two directories
  (`sha256-0d2e6feda49add0c252c1012cbaa6839`,
  `sha256-d8050f517fb3a44d31e52b3eafca18c4c85322f7e7b4d884085de55fbfb4610c`),
  inspect exit 0 with `retained_successes: 2`, `gomad replay` exit 0 for each
  (`collision-after-*.json`).
- No-collision campaign (`testdata/campaign`, seeds 0-2, keep all; exit 1
  because seed 2 fails by design): `identities-before.json` and
  `identities-after.json` (from `identities.py`) agree on directory names,
  seeds, outcome signatures and target SHA-256. Full manifests also agree
  except fields that vary on every run: `campaign_id`, `created_at`, `host`,
  `limits.overall_timeout_nanos` (remaining overall time), `runner.runner_build`
  (the changed binary) and therefore `record_hash`. `record_hash` is not
  reproducible between two runs of the same binary because it binds the
  remaining overall timeout.

## Tests

- `artifact`: `TestPublishKeepsEachExecutionOfOneOutcomeSignature`.
- `runner`: `TestRunKeepsTwoSuccessesOfOneOutcomeSignatureApart` (fake
  executor, same result for both seeds; `OpenCampaign` accepts; 2 retained,
  2 directories, summed stored bytes equal `retained_success_bytes`). Red
  before the change: both `SuccessArtifacts` named one directory.
- `cmd/gomad`: `TestCLIExploreKeepsSuccessesOfOneOutcomeSignatureApart`
  (built CLI, seedfree fixture, inspect and replay of both artifacts).

## Gates (linux/arm64 dev harness)

See `gates.md`.

## Owed

darwin/arm64 and linux/amd64: `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`,
`make -C tools/gomad3 validate`, and the CLI reproduction above.
