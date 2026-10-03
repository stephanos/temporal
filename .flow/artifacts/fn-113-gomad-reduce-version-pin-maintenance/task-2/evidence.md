# fn-113 task 2 evidence

Host: linux/arm64 with the local development harness on (linux/arm64 added to `version.json`
and the boundary manifest, `syscall.Dup2` shimmed). Everything here is linux/arm64
development evidence, not darwin/arm64 or linux/amd64 evidence. Source: branch `gomad-next-b`
on `d617a8a1c` plus this task's change (commit recorded in the task's Evidence section).
Go: pinned go1.27.1.

## What the command does

`gomadtool adapter-regenerate --module=M --version=V` (dry run) downloads the pinned and the
candidate version into a private module cache, re-applies each rewrite by its existing
exact-occurrence anchor, and prints the upstream diff of every file a rewrite reads, the
proposed anchors, stale compatibility-pack bindings, and an approval digest. With
`--approve-review=DIGEST` it stages the full output set in a scratch copy of the Gomad module
(adapter constants and tests, the `version.json` entry, fixture modules plus `go mod tidy`,
module-local `make generate` outputs), verifies it, and publishes under a lock through a
committed journal under `.toolchain/adapter-regeneration`.

Adapters are now uniformly declared as `rewrittenModule` specs. gRPC and x/net moved off their
custom preparation code onto the same helpers. A `sourceRewrite` can now name a `base` file whose
anchored rewrite replaces `path`, which is how gRPC swaps in its non-Linux implementations.
Prepared source-set pins are `...ByHost` maps. 14 of 15 adapters are regenerable.
`modernc.org/libc` derives its rewrite from parsed syntax and is refused with exit 1.

## Per-platform anchors are pure source hashing

`target.AdapterPreparedSourceSetSHA256` lists the prepared package by directory
(`GO111MODULE=off go list -find`) under the target build environment with `GOOS`/`GOARCH` set,
and hashes the files exactly as capability review does. File selection is build-constraint
evaluation only, so any host computes every platform's pin.
`TestAdapterPreparedSourceSetPinsReproduceOnEveryHost` reproduces **all 30 checked-in pins
(15 adapters x darwin/arm64 + linux/amd64) exactly** on linux/arm64. The libc and pebble
pins differ between platforms, so the platform-specific selection is exercised too.

## Real adapter

The root `go.mod` has moved past no adapted module. `gomadtool pin-impact` on HEAD exits 0
with 0 invalidated adapters (`pin-impact-head.txt`). So no checked-in adapter was regenerated,
and the fixture bump stands in for acceptance. As extra evidence:

- `grpc-v1.84.0-dry-run.txt`: dry run for gRPC v1.84.0, the version task 1's real root-bump
  candidate moves to (2.3 s). All 11 rewritten files are unchanged upstream. The sum,
  inventories, and prepared source set change.
- `grpc-v1.84.0-apply-scratch-copy.txt` / `.diff`: the apply with that digest in a scratch copy
  of `tools/gomad3`, with the harness on, using the default generators and verifiers (14 s, exit 0).
  It published 7 files (adapter, three tests, the descriptor, its generated Go file, and the
  generated upgrade guide) and listed `README.md:522` for review. The diff context shows the
  harness's `linux/arm64` platform entry, which belongs to the copy and not to the change.
- `grpc-v1.84.0-scratch-copy-adapter-tests.txt`: in that regenerated copy, the gRPC adapter
  tests, the gRPC DNS consumer, and the 30-pin reproduction pass against v1.84.0.
- `sentry-v0.49.0-dry-run.txt`: a real upstream change to the rewritten `util.go`. It is
  shown as a diff, and the anchors still match.
- `memory-v1.12.1-blocked.txt`: a real moved anchor. `modernc.org/memory@v1.12.1` changed
  `mmap_unix.go`, so the run exits 1 and writes nothing.

Workload qualification of a regenerated adapter is owed on darwin/arm64 and linux/amd64.
linux/arm64 has no adapter pins, so its qualification cannot run here.

## Tests (linux/arm64, harness on)

| Command | Result |
| --- | --- |
| `go test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/... ./target/...` | pass (1m44s) |
| `go test -tags test_dep -count=1 ./deterministicio/...` | 14 failures, all in BASELINE.md (`adapter replacement digest is invalid` / empty pin on linux/arm64); new tests pass |
| `go test -count=1 -run TestPackageArchitecture .` | pass |
| `make -C tools/gomad3 validate` | pass (24 s) |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` | `test-host.txt`. Every failure is in BASELINE.md except `TestRunHardCrashesAndReapsSimulationNodeProcess`, a runner process test. It passed 3 of 3 times when rerun alone, and this change does not touch runner code. |

Acceptance fixtures:

- Dry run prints the diff, anchors, and digest and writes nothing; the digest is reproducible:
  `TestDryRunPrintsChangedSourceAndAnchorsAndWritesNothing`.
- Apply updates the constants, descriptor, fixture `go.mod`/`go.sum`, and a generated output
  derived from the descriptor together: `TestApplyPublishesConstantsDescriptorTestDataAndGeneratedOutputTogether`.
  A wrong digest writes nothing: `TestApplyWithAWrongDigestWritesNothing`.
- Moved anchor, anchor matching twice, and rewritten file deleted upstream, served from a file
  module proxy: `TestRegenerationThatNeedsAPersonWritesNothing`. The same cases at engine level
  are `TestRegenerateAdapterFailsWithoutWritingWhenAnchorsDoNotMatchOnce` and
  `TestRegenerateAdapterFailsWhenARewrittenFileIsDeletedUpstream`.
- Generation or verification failure in staging: `TestGenerationFailureInStagingPublishesNothing`.
  Checkout changed between staging and publication:
  `TestCheckoutChangedBetweenStagingAndPublicationPublishesNothing`. Competing applies, both a
  held lock and a second apply started while the first stages:
  `TestCompetingApplyOperationsPublishOnce`.
- Interrupted publication is completed by `--recover` or by the next apply, and the result
  equals a clean apply byte for byte: `TestInterruptedPublicationCompletesOnTheNextRun`. An
  uncommitted journal rolls back: `TestUncommittedJournalRollsBack`. A committed journal over
  later edits is left for a person: `TestCommittedJournalOverLaterEditsIsLeftForAPerson`.
- Stale packs: `TestStalePacksNameBindingsOfThePreviousIdentity`. Bumping `modernc.org/memory`
  names the libc-bound `modernc-libc-xsys-*` packs.
- Source edits replace every anchor and only that adapter's literals; memberlist and go-metrics
  share `v0.5.4`: `TestAdapterSourceEditsReplaceEveryAnchor`. A computed anchor fails:
  `TestAdapterSourceEditsRejectAnAnchorThatIsNotALiteral`.

## Harness off (committed state on linux/arm64)

`make -C tools/gomad3 validate` stops at `undefined: syscall.Dup2` in
`runner/internal/execution`, the known harness limitation. It passes only with the harness on.
With the harness off, `go test ./upgrade/adapterregen` and the new `deterministicio` tests
(`TestRegenerateAdapter*`, `TestAdapterSourceEdits*`, the 30-pin reproduction) pass.

## Still owed on qualified platforms

```sh
# darwin/arm64 and linux/amd64, harness absent
go -C tools/gomad3 test -tags test_dep -count=1 ./deterministicio ./upgrade/... ./cmd/gomadtool ./target/...
go -C tools/gomad3 test -count=1 -run TestPackageArchitecture .
make -C tools/gomad3 validate
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
# once an adapted module moves: regenerate it, rebuild .bin/gomad, refresh stale packs, and
# qualify the adapter's workloads on both platforms
```
