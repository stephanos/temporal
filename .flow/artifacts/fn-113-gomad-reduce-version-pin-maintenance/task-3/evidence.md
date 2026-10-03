# fn-113 task 3 evidence

Host: linux/arm64 with the local development harness on (linux/arm64 added to `version.json`
and the boundary manifest, `syscall.Dup2` shimmed). Everything here is linux/arm64
development evidence, not darwin/arm64 or linux/amd64 evidence. Source: branch `gomad-next-b`
on `56148912d` plus this task's change (commit recorded in the task's Evidence section).
Go: pinned go1.27.1.

## What changed

- `gomadtool compatibility-pack refresh` (`cmd/gomadtool/compatibility_pack_refresh.go`,
  engine `internal/compatibilitypack/authoring/refresh.go`). It runs the pin impact report once
  per mapped working directory (working tree = candidate, `--baseline-ref`, default `HEAD` =
  baseline). It selects requests with invalidated or unknown pack-rule pins, requests without
  an approval, and host-platform requests bound to another deterministic I/O profile, which is
  the fn-105.26 audit. Each selected request is discovered in memory in its own directory. A
  request is current only when its stored approval equals the review digest of the fresh
  evidence. Otherwise the fresh evidence is written with the approval cleared, and only when
  it differs from what is stored. Reports and packs are then regenerated, and refresh prints
  the digest and the exact `generate --approve-review` command. Other-platform requests are
  reported `not-evaluable` and left untouched. Packs whose directory no longer requires their
  modules are reported `unselected`. Exit status: 0 nothing to do, 1 action needed, 2 invalid
  input, 3 infrastructure failure.
- `internal/compatibilitypack/working-directories.json`: one table mapping every request to its
  module directory, relative to the pack root. It is read by `refresh` and by the new
  `compatibility-pack qualify --all`, which the `compatibility-pack-qualification` Makefile
  target now runs. It qualifies the requests that name the host platform, as the per-platform
  Makefile lists did. `check` rejects a request with no entry, an entry with no request, and
  an unclean or absolute path. All of these are invalid input.
- `generate` (and so refresh) removes packs and reports it no longer renders. Before this
  change, a request whose approval was cleared left a stale pack that `check` rejected.
- No policy widening. Discovery, the review digest (`ApprovalSHA256`), pack validation
  (including the `os/exec`, `os/signal`, `os/user`, `plugin`, and `runtime/cgo` ban), and
  `generate --approve-review` are unchanged. Refresh never approves. Newly observed facts
  still arrive denied.
- The reviewer is `qualification/analysis.ReviewCompatibilityTarget`, the same preparation and
  review that `discover` uses. The host profile comes from
  `analysis.HostDeterministicProfile`. The developer owner may not import `target` or
  `deterministicio`, so `authoring.CapabilityReview` is a type alias.

## Stale variant removed: `modernc-libc-xsys-v041`

`pack-selection-audit.sh` runs `gomadtool pin-impact` for every Go module tracked in the
repository. Each module is the baseline and an empty module is the candidate, so every pack
rule the module selects is reported `stale`, and that lists exactly the packs each module
selects. Selection works by exact module identity in the module graph, and MVS does not
depend on GOOS/GOARCH. A pack that no module graph selects can therefore be selected on no
platform, and this audit is valid for darwin/arm64 and linux/amd64 packs although it ran on
linux/arm64.

- `pack-selection-before-removal.txt`: 18 modules (root, `tests/mixedbrain`, `tools/gomad3`,
  the qualification corpus, and every adapter, conformance, and compatibility fixture).
  `modernc-libc-xsys-v041` (activation `golang.org/x/sys@v0.41.0`, libc v1.72.3, memory
  v1.11.0) is selected only by `internal/compatibilitypack/testdata/v041`. Every other pack is
  selected by at least one module other than its own qualification fixture. Each one is
  selected by the root module or the corpus, including `golang-x-sys-v047-darwin-arm64`,
  whose `testdata/xsys` fixture is also used elsewhere. So v041 is the only unselected variant.
- `testdata/v041` existed only to qualify that pack: fn-95.3 authored it in 7ea97c052 when the
  darwin qualification list named a fixture that was missing. Nothing else references the
  fixture, its module path, or the `modernc-libc-xsys-v041-fixture` workload. No
  qualification manifest names that workload.
- Removed together: request, report, pack, fixture module, and the darwin qualification entry
  (now the table). `generation.json` and `packs_generated_test.go` were regenerated. The two
  selection tests that used v041 as their example pack now use `modernc-libc-xsys-v047`,
  selected alone, because other packs also activate on x/sys v0.47.0. The evidence test now
  expects that pack's six rules.
- Every remaining pack, request, and report is byte-identical to `HEAD`. The regenerated
  `generation.json` differs only by the three removed outputs and the
  `packs_generated_test.go` hash, which differs only by the v041 mutation lines.
- `pack-selection-after-removal.txt`: the same audit after removal differs only by the two
  v041 lines.

## Commands and results (linux/arm64, harness on unless stated)

| Command | Result |
| --- | --- |
| `go test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/... ./qualification/analysis` | pass, except `TestPrepareCapabilityReviewSupportsTemporalBackoffWithGRPCAdapter` (BASELINE) |
| `go test -count=1 -run TestPackageArchitecture .` | pass |
| `make -C tools/gomad3 validate` | pass (6 s) |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` | exit 2 in 328 s; every failure is in BASELINE.md (`test-host.txt`) |
| `gomadtool compatibility-pack refresh --root=.` on the unmodified tree | exit 0, `0 requests selected` (17 s) |
| refresh after `go get github.com/klauspost/compress@v1.18.6` in the root module | exit 1 in 6 s. It names the 4 packs task 1's report named, all `not-evaluable` here; no pack file written (`real-bump-refresh.txt`) |
| refresh with the real reviewer, linux/arm64 dev request in a scratch pack root, `x/sys` v0.47.0 -> v0.48.0 | `awaiting-approval` with the fresh digest. The rerun reports the same digest. The older approval is refused. After approval: rerun exit 0, `qualify --all` and `check` pass (`linux-arm64-dev-refresh.txt`) |

Harness off (committed state): `make validate` stops at `undefined: syscall.Dup2` in
`runner/internal/execution`, which is the known harness limitation. It also means `gomadtool`
and `cmd/gomadtool` tests do not build on linux/arm64. For validate-compatibility semantics I
ran `go test ./internal/compatibilitypack/...` (pass; `TestHostPacksBindCurrentProfile`
skips, because linux/arm64 has no profile). I also ran a throwaway test, not committed, that
calls `authoring.Check` on the real root and `compatibility.LoadPacks`. Both pass: 11
embedded packs validate under strict pack validation (8 darwin/arm64, 3 linux/amd64), the
table maps every request, and v041 is gone. The profile binding of darwin/arm64 and
linux/amd64 packs can only be checked on those hosts. No remaining pack changed, so their
bindings are as committed.

Acceptance tests (`internal/compatibilitypack/authoring/refresh_test.go` unless named):

- Two modules, each discovered in its own directory at its own candidate version (v1.1.0 and
  v1.2.0), stopping with one digest per request:
  `TestRefreshDiscoversEachRequestInItsModuleAndStopsAtApproval`. The CLI version through the
  real pin impact report against a Git baseline, approval, and rerun is
  `cmd/gomadtool TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes`. An unmapped
  request is invalid input (exit 2 there, `InputError` in
  `TestRefreshRejectsAnUnmappedRequestAsInvalidInput`).
- Approve one of two refreshed requests and rerun: the approved request stays and is
  untouched, and only the other is reported:
  `TestRefreshKeepsAnApprovedRequestAndReportsOnlyTheOther`.
- An approval of older evidence is never current, including a request file that carries the
  fresh evidence with the older approval: `TestRefreshNeverTreatsAnApprovalOfOlderEvidenceAsCurrent`.
- Other-platform requests are reported, not reviewed, and their request, report, and pack
  bytes are unchanged: `TestRefreshReportsOtherPlatformRequestsAndLeavesThemUntouched`.
- Profile-drift selection: `TestRefreshSelectsAHostRequestBoundToAnotherProfile`. A failing
  review is reported without a rewrite: `TestRefreshReportsAReviewFailureAndContinues`. The
  checked-in table maps every request to a directory with a `go.mod`:
  `TestCheckedInWorkingDirectoriesMapEveryRequest`.

## Still owed on qualified platforms

```sh
# darwin/arm64 and linux/amd64, harness absent
make -C tools/gomad3 validate compatibility-pack-qualification   # darwin: 8 requests, linux/amd64: 3
go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./internal/compatibilitypack/... ./qualification/analysis
go -C tools/gomad3 test -count=1 -run TestPackageArchitecture .
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh --root=.   # expect exit 0 on an unmodified checkout
```

On a real bump, each platform's host runs `refresh`, reviews the reports, approves its own
platform's requests, and runs `compatibility-pack-qualification`.
