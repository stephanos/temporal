# fn-114 task 5: gate evidence (darwin/arm64, 2026-10-02)

Host: darwin/arm64, stock go1.27.1 first on PATH
(`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin`).
Base commit `32b4cf30ff3450d0dd38eefd4fb6814cb0a8e18c`. Linux/amd64 was not
run: no native host in this session.

## Toolchain identity

| | build key |
| --- | --- |
| before (base commit) | `56e4a2f0c5514d43b9a0682d030964588dd3d5a58978ba0b989459bb556843a3` |
| after (this task) | `245141dc288328269b6fc164034c6d74b87c525e5d3c4756d3b82a3fcde40d52` |

The choice implementation identity changed with the overlay edit;
`protocol-generate` refreshed `choice/internal/wire/wire_generated.go`,
`overlay/src/internal/gomadchoicewire/wire_generated.go`, and the live-capability
producer/guard identities in `target/internal/livecap/protocol_generated.go` and
`overlay/src/cmd/internal/gomadcap/protocol_generated.go`. Retained artifacts are
not rewritten.

## Baseline (before any edit)

| command | exit | elapsed | note |
| --- | ---: | --- | --- |
| `make -C tools/gomad3 validate test-toolchain overlay-test` | 0 | 1:08 | |
| `make -C tools/gomad3 test-runtime` | 2 | 7:49 | `go run` fixture build failed on a vanished entry in the host go-build cache (`internal/bytealg`); environment, not source |
| `make -C tools/gomad3 test-runtime` (rerun) | 0 | 7:37 | |
| `go -C tools/gomad3 test -tags test_dep ./runner/... ./artifact/... ./target/... ./cmd/gomad/...` | 1 | 2:28 | 13 pre-existing failures, listed below |
| focused rerun of two of those failures | 1 | 1.2s | deterministic, not load |

Pre-existing (baseline: red) failures, all helper-target tests outside this
task's files: `runner` `TestReplayExecutesMatchingWorldPlanThroughChildTransport`,
`TestReplayRejectsFirstDivergentWorldTransitionBeforeTargetMutation` ("I/O
transcript terminal is absent"); `runner/internal/execution`
`TestRunTransportsCompleteChoiceTrace`, `TestRunReturnsValidatedOverflowChoiceTrace`,
`TestRunReplaysCompleteChoiceTape`, `TestRunReplaysLogicalChoiceAcrossPhysicalRunQueueOrder`,
`TestRunReplaysSelectPermutationAcrossSeededPhysicalOrder`,
`TestRunPreservesChoicePrefixRNGPosition`, `TestRunForcesCanonicalRankAtFinalPrefixDecision`,
`TestRunTargetInheritsChoiceTapeReadOnly`, `TestRunRejectsExhaustedChoiceTapeBeforeTargetMarker`,
`TestRunRejectsChoiceMetadataMismatchBeforeTargetMarker`, `TestRunRejectsUnconsumedChoiceTape`
("choice trace unterminated").

## Red first

The updated C2 characterization run against the unmodified toolchain
(`56e4a2f0`) fails on identity instability; see `red-first.txt`:

```
timer callback identities: seed 4 led B with identity 8db55ff1…, earlier runs with ecea36d5…
```

## After the edit

| command | exit | elapsed |
| --- | ---: | --- |
| `make -C tools/gomad3 toolchain` (rebuild) | 0 | 2:17 |
| `make -C tools/gomad3 validate test-toolchain overlay-test` | 0 | 0:34 |
| `make -C tools/gomad3 test-runtime` | 0 | 7:47 |
| `tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep -count=1 -run TestRuntimeSearchFixtures ./internal/gomadtool/conformance` (retained to `search-reproduction.json`) | 0 | 0:11 |
| `.toolchain/bin/go test -tags test_dep -count=1 -run 'TestPatchedRuntimeGoroutineCreationsAreReviewed\|TestGoroutineCreationInventoryRejectsSeededSite\|TestPatchedRuntimeHostClockReferencesAreReviewed' ./toolchain` | 0 | 2.4s |
| `go -C tools/gomad3 test -tags test_dep -count=1 ./choice/... ./internal/gomadtool/... ./cmd/gomadtool` | 0 | 0:21 |
| `go -C tools/gomad3 test -tags test_dep ./runner/... ./artifact/... ./target/... ./cmd/gomad/...` | 1 | 2:33 |
| `go -C tools/gomad3 test -tags test_dep -count=1 -run TestDiagnosticsOffPreservesExistingCanonicalIdentities ./runner` (after the golden refresh) | 0 | 0.8s |
| `gofmt -l` on changed packages, `go vet -tags test_dep` on `./toolchain ./toolchain/version ./internal/gomadtool/... ./choice/... ./target/internal/livecap/...` | 0 | |

The Runner suite's failure set after the edit is the same 13 pre-existing
tests plus, before the golden refresh, `TestDiagnosticsOffPreservesExistingCanonicalIdentities/choices`.
That golden (`runner/testdata/diagnostic-identity-choices.json`) pins the artifact
manifest's choice implementation identity; the refresh changed only
`implementation_sha256` (three places) and the digests derived from it
(`tape_sha256`, `failure_signature`, `record_hash`, `portable_plan_sha256`).
The `plain` golden is unchanged.

## What `search-reproduction.json` shows (build key `245141dc`)

- 32 seeds of the task-2 fixture: both firing orders occur (`A B`, `B A`); every
  run decides between the same two-goroutine alternative set; A always leads
  with one identity and B with another; the two make up the set.
- All 16 one-decision alternative prefixes of the seed-6 parent execute with
  exit 0, keep the same identities, and include prefixes that swap the callbacks.
- Cross-seed experiment (seed-6 plan under seed 16): exit 0, transcript `B A`,
  the same as the seed-6 parent. Recorded as evidence, not a requirement.
- `timer-creator-identity`, modes none/stopped/pending x seeds 1-8: one
  alternative set, consistent leaders.
- `timer-reset-identity`, seeds 1-8: one alternative set, two distinct callback
  identities of one reset timer.
- E3 select shapes: the same seven frontiers exhaust after 196/68/68/68/196/68/68
  executions as in task 2.
