# fn-113 task 4 evidence

Host: **linux/arm64 with the local development harness on** (linux/arm64 added to
`version.json` and the boundary manifest, `syscall.Dup2` shimmed). Everything here is
linux/arm64 development evidence, not darwin/arm64 or linux/amd64 evidence. Go: pinned
go1.27.1. Source: branch `gomad-next-b`; the walk ran on `7fd67d5aa`, the review fixes are
`d17d74611`, and the documentation commit follows it (recorded in the task's Evidence section).

## Review fixes for task 3 (`d17d74611`)

- `compatibility-pack refresh` now judges the packs in the refreshed root's `packs/`
  (`compatibility.LoadPackDirectory`, passed as `pinimpact.Spec.Packs`) instead of
  `compatibility.LoadPacks()`. Before, a `--compatibility-root` refresh without
  `GOMAD3_COMPATIBILITY_PACKS`, or with it naming another directory, selected nothing and
  exited 0. `TestRunCompatibilityPackRefreshStopsAtApprovalAndResumes` now runs with the
  variable unset, set to the root's packs, and set elsewhere; the `unset` and `elsewhere`
  cases fail without the fix. `TestLoadPackDirectoryReadsOnlyThatDirectory` covers the loader.
- `compatibility-pack check` (and so `make validate`) requires the repository root's
  `working-directories.json`, and every entry must name a directory with a `go.mod`
  (`authoring.CheckWorkingDirectories`). A downstream `--compatibility-root` may still omit
  the table. Tests: `TestCheckWorkingDirectoriesRequiresTheTableAndAModulePerEntry`,
  `TestRunCompatibilityPackCheckRequiresTheRepositoryWorkingDirectoryTable`.
- Refresh of the unmodified tree still exits 0 with `0 requests selected` (5.8 s). The walk's
  refresh rerun with the fix applied printed the same four results (`walk/13-*`).

## Documentation

- `tools/gomad3/README.md`: dependency bump procedure after the Go upgrade section, a pointer
  from the adapter anchors to `adapter-regenerate`, and the review-fix behavior in the pack
  section.
- `tools/gomad3/CLI.md`: step 9 "Bump a dependency" (order of the three commands, the
  `--baseline-ref` caveat, what still needs a person), and the `compatibility-pack` index row
  now names refresh. `pin-impact` and `adapter-regenerate` rows were already present.
- `tools/gomad3/SPEC.md`: `[MAINTENANCE.DEPENDENCY]`, `[COMMAND.GOMADTOOL.PIN.IMPACT]`,
  `[COMMAND.GOMADTOOL.ADAPTER.REGENERATE]`, and refresh in `[COMMAND.GOMADTOOL.COMPATIBILITY.PACK]`.
- `tools/gomad3/ARCHITECTURE.md`: maintenance gates paragraph on how the three commands reuse
  the build's pin owners.
- Upgrade guide: "Dependency bumps" section in `renderUpgradeGuide`
  (`toolchain/version/descriptor.go`), regenerated with `make generate`.
- `.plans/GOMAD_NEXT.md` COMPAT-8 and `MILESTONES.md` (fn-113 row, maintenance-cost rows).

Doc checks (`check-docs.py`, run from the repository root against a `gomadtool` built from the
working tree): 78 `gomadtool` flags in the seven edited documents are accepted by their
command's `-h`; the CLI.md index lists all 17 `gomadtool` commands and nothing else; 103
relative links and fragments resolve. A planted bad flag and fragment were reported (negative
check). `git diff --check` is clean. fn-111's own scripts were not rerun: they bind
darwin/arm64 help text and fixed file hashes.

## Measured bump (R5)

Representative bump: the root module moves `google.golang.org/grpc` v1.83.2 -> v1.84.0 (an
adapted module, 11 rewritten files) and `github.com/klauspost/compress` v1.18.5 -> v1.18.6
(activates 4 packs: 2 darwin/arm64, 2 linux/amd64), the same candidate task 1 measured. It ran
in a scratch `git clone --shared` of the worktree with the harness applied, so nothing here
was committed. Outputs: `walk/`; timings: `walk/timings.txt`.

| # | Step | Kind | Result here | Time |
| --- | --- | --- | --- | --- |
| 1 | `go get github.com/klauspost/compress@v1.18.6 google.golang.org/grpc@v1.84.0` | command | go.mod/go.sum changed | 0.4 s (module cache warm) |
| 2 | `go -C tools/gomad3 run ./cmd/gomadtool pin-impact --root=.` | command | exit 1: gRPC adapter and 30 rules of the 4 packs invalidated | 8.2 s |
| 3 | `adapter-regenerate --module=google.golang.org/grpc --version=v1.84.0` | command | exit 0: 0 of 11 rewritten files changed upstream; digest printed | 2.1 s |
| - | same with `--approve-review=... --stage-only` (optional) | command | 7 files listed, nothing written | 13.7 s |
| 4 | same with `--approve-review=...` | command | exit 0: 7 files published; `README.md:522` listed | 13.1 s |
| 5 | edit `README.md:522` to v1.84.0 | hand edit | | |
| 6 | `compatibility-pack refresh --root=.` | command, once per platform (2) | exit 1: the 4 requests `not-evaluable` on linux/arm64 | 8.2 s |
| 7 | `compatibility-pack generate --request=... --approve-review=...` | command, per request (4) | **not run**: no darwin/arm64 or linux/amd64 host | |

Shared final gates, the same before and after: `make gomad3` (65.8 s here, including a fresh
toolchain build in the clone), `make -C tools/gomad3 validate compatibility-pack-qualification`
on each platform (validate: exit 0, 8.9 s; qualification not run here, no linux/arm64 packs),
and requalification of the gRPC workloads on both platforms (owed). The gRPC adapter tests and
the 30-pin reproduction pass in the clone against v1.84.0 (3.6 s; a first attempt failed only
because the clone had no `.toolchain` yet).

Manual steps (one command invocation or one hand edit, spec R5), excluding the shared final
gates and the person's review time:

| | Before (task 1 baseline) | After (walked) |
| --- | --- | --- |
| bump | 1 | 1 |
| report | none exists | 1 (`pin-impact`) |
| gRPC adapter, this bump (0 of 11 rewritten files changed upstream) | 2 hand edits + `make generate` + about 3 `go test` iterations (original inventory, replacement inventory, prepared source set) + 1 on the other platform = 7 | 2 commands + 1 hand edit = 3 |
| gRPC adapter, worst case (all 11 rewritten files change upstream) | 2 hand edits + `make generate` + about 2 + 2 x 11 = 24 `go test` iterations + 1 on the other platform = 28, plus anchor repair | 2 commands + 1 hand edit = 3, plus anchor repair |
| 4 packs | 4 x (discover, review, generate) = 12 | 2 refresh + 4 generate = 6 |
| **total, this bump** | **about 20 (2 hand edits)** | **11 (1 hand edit)** |
| **total, worst case** | **about 41 (2 hand edits)** | **11 (1 hand edit)** |

The walked bump is the fair comparison: no rewritten gRPC file changed upstream, so the old
procedure would have stopped at about 3 digest mismatches (task 1's "about 5 to 10 commands"
per adapter), not 2 x 11. The 41-step figure applies only when every rewritten file changes.

The baseline's "2 hand edits" is a lower bound: the walked apply also rewrote three test files
that name the version (`adapter_registry_test.go`, `grpc_adapter_test.go`,
`requirements_test.go`), which the old procedure would edit by hand too, plus the README line.
Task 1 recorded step counts, not times, so there is no measured baseline time; one gRPC adapter
test iteration took 3.6 s here, so the 25 baseline iterations alone are about 1.5 minutes of
test time before any hand editing in the worst case, and about 4 runs (15 s) for this bump. The after path's tool time on this host is about 32 s for
steps 2 to 6 (45 s with the optional stage-only run).

Two findings from the walk:

- After the bump is committed, `pin-impact` and `refresh` against the default `HEAD` report
  nothing: the packs pinned to klauspost v1.18.5 are then `not selected` (`walk/10-*`,
  `walk/11-*`). With `--baseline-ref` set to the revision before the bump, refresh selects
  the same 4 requests (`walk/12-*`). The docs now say to run both before committing or to pass
  `--baseline-ref`.
- Adapter regeneration does not cover the hand-maintained prose reference in `README.md`; the
  apply lists it, and it stays one hand edit.

## Gates (linux/arm64, harness on)

| Command | Result |
| --- | --- |
| `python3 .flow/artifacts/.../task-4/check-docs.py GOMADTOOL` | exit 0 (78 flags, 17 commands, 103 links) |
| `go test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/...` (review fixes) | pass (13 s) |
| `go test -count=1 -run TestPackageArchitecture .` | pass |
| `go test -count=1 ./toolchain/version/` | pass |
| `go run ./cmd/gomadtool version-generate --check` | exit 0 |
| `make -C tools/gomad3 validate` | exit 0 (review fixes: 14 s; docs: 23 s) |
| `go run ./cmd/gomadtool pin-impact` on the unmodified tree | exit 0 |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` | exit 2 in 237 s; 20 failures, every one in the harness baseline (`TestRunTimesOutAndRemovesTermIgnoringProcessGroup`, flaky, passed); `test-host.txt` |

With the harness off, `make validate` stops at `undefined: syscall.Dup2`, the known harness
limitation (tasks 1 to 3 recorded the same).

## R6: not met

R6 needs `make -C tools/gomad3 validate` and `test`, compatibility-pack qualification, and the
core set on darwin/arm64 and linux/amd64. None of these ran on a qualified platform. Owed, on
each of darwin/arm64 and linux/amd64 with no harness:

```sh
make -C tools/gomad3 validate
make -C tools/gomad3 test
make -C tools/gomad3 compatibility-pack-qualification   # darwin: 8 requests, linux/amd64: 3
make -C tools/gomad3 core-qualification
GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host
go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/... ./qualification/analysis ./deterministicio
go -C tools/gomad3 test -count=1 -run TestPackageArchitecture .
go -C tools/gomad3 run ./cmd/gomadtool pin-impact                              # expect exit 0
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh --root=.      # expect exit 0
```

To finish the measured walk on qualified hosts: apply the same bump uncommitted, run
`compatibility-pack refresh --root=.` on each platform, review and approve its two requests
with the printed `generate` commands, run `make gomad3` and
`make -C tools/gomad3 validate compatibility-pack-qualification`, and requalify the gRPC
workloads.

## Review fixes for task 4 (`gomad: bump-procedure review fixes`)

Host: linux/arm64 with the development harness on; outputs in `review-fixes/`. The scratch runs
used a `git clone --shared` of the worktree with the uncommitted fix and harness applied.

1. **(HIGH) Staged adapter regeneration failed at the pack check.** `d17d74611` made
   `compatibility-pack check` require a `go.mod` in every mapped working directory; the
   staged copy holds only `tools/gomad3`, so the seven entries mapped to the repository root
   resolved to the temp directory and every apply or `--stage-only` failed. Fix:
   `check --staged-copy` (`authoring.CheckStagedCopy`, `CheckWorkingDirectories(root, false)`)
   still requires the table to map exactly the requests but skips the `go.mod` check;
   `DefaultVerifiers` passes it, and `make validate` keeps the strict check.
   Tests: `TestDefaultPackCheckPassesInAStagedCopyOfTheModule` (new; runs the real default
   verifier in a `copyCheckout` stage of this module, and asserts the check without
   `--staged-copy` fails there with "holds no go.mod"),
   `TestCheckWorkingDirectoriesRequiresTheTableAndAModulePerEntry` (extended). Real run of the
   walked gRPC v1.84.0 `--stage-only`: exit 1 with the old verifier
   (`01-stage-only-before-fix.txt`), exit 0 listing the same 7 files with the fix
   (`02-stage-only-with-fix.txt`, 10 s).
2. **(MEDIUM) A committed bump reported all clear.** `pin-impact` now reports a pack rule
   invalidated, not "not selected", when neither side selects the pack but the candidate
   requires every one of its activation modules at other versions (a stranded pack), whatever
   the baseline. Refresh inherits it. Test: `TestCommittedBumpKeepsTheStrandedPackInvalidated`
   (fails without the fix). The unmodified tree still reports 0 invalidated (exit 0). After
   committing the klauspost v1.18.6 bump in the scratch clone, `pin-impact` exits 1 with the
   30 rules of the 4 packs invalidated (`03-*`), and `refresh --root=.` selects the same 4
   requests and exits 1 (`04-*`). A module the bump removes is still reported stale, and its
   pack `unselected`, only against a baseline that requires it; CLI.md, README, and the
   generated upgrade guide now say so instead of "pass `--baseline-ref` after committing".
3. **(MEDIUM) Step count.** The table above now counts the walked bump fairly (about 20
   before, 0 of 11 rewritten files changed) and keeps about 41 as the worst case;
   MILESTONES's fn-113 row states both.
4. **(LOW)** CLI.md "Bump a dependency", the README procedure, and the generated upgrade guide
   name `pin-impact --module=DIR` for a module other than the repository root module.

Gates after the fixes (linux/arm64, harness on): `go test -tags test_dep -count=1
./cmd/gomadtool ./upgrade/... ./internal/compatibilitypack/... ./toolchain/version/` pass
(13 s); `TestPackageArchitecture` pass; `make -C tools/gomad3 validate` exit 0 (5 s);
`check-docs.py` exit 0 (78 flags, 17 commands, 103 links); `git diff --check` clean.
The darwin/arm64 and linux/amd64 commands listed under R6 remain owed.
