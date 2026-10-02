# fn-108.4 evidence: upgrade dossier publication through hostfs.Replace

Platform darwin/arm64, patched go1.27.1 toolchain (`tools/gomad3/.toolchain/bin/go`), base revision
`6782b55f49a0317b230e827ea2a63a37d116d502`. Nothing staged or committed. linux/amd64: not run (no host).

## Change

- `tools/gomad3/upgrade/upgrade.go`: `publish` keeps `json.MarshalIndent(dossier, "", "  ")` plus `'\n'` and
  calls `hostfs.Replace(path, contents, 0o644)`, wrapped as `publish upgrade dossier: %w`. The private
  MkdirAll/CreateTemp/Chmod/Write/Close/Rename sequence is gone.
- `tools/gomad3/architecture_test.go`: `ownerMayImport` gains the single edge `upgrade` -> `hostfs`.
  Without it `TestPackageArchitecture` fails with "owner upgrade package .../upgrade imports forbidden
  owner hostfs package .../internal/hostfs" (checked, then restored).
- `tools/gomad3/ARCHITECTURE.md`: one sentence in the upgrade-dossier paragraph on the stronger
  sync/cleanup reporting.
- Tests: `upgrade/upgrade_test.go` (three tests, two helpers) and new `upgrade/upgrade_unix_test.go`.

## Behaviour differences (all intended by the task)

- Error text: `create upgrade dossier directory|create upgrade dossier|chmod upgrade dossier|write upgrade
  dossier|close upgrade dossier: ...` become `publish upgrade dossier: <hostfs stage>: ...`. The wrapped
  OS error is unchanged (`errors.Is` on `os.ErrPermission`, `ENOTDIR`, `EFBIG` holds before and after).
- New reported failures: file sync, directory open/sync/close, temp-file removal, and a close error
  after a failed chmod/write. A directory-sync failure is returned after the rename, so the complete
  new dossier is already in place.
- Temp-file prefix `.upgrade-dossier-*` becomes `.safefile-*`. No other match for the old prefix in the
  repository; CI uploads the exact path `tools/gomad3/.toolchain/upgrade-dossier.json`.
- Unchanged: payload bytes, trailing newline, mode 0o644, destination, parent creation (0o755),
  rename-over-existing, publication before the gate-failure return, dossier classification.

## Characterization tests

Written before the switch; passed against the old publisher, then unchanged against the new one.

| Test | Pins |
| --- | --- |
| `TestPublishWritesFixedDossierBytes` | golden literal bytes (two-space indent, HTML escaping, trailing newline), mode 0o644, nested parent creation, replacement of a longer stale 0o600 file, no leftover dot-file |
| `TestRunReplacesExistingDossierAfterFailedGate` | failed gate still publishes over a prior dossier; bytes equal `MarshalIndent` + newline; mode; no temp entry |
| `TestRunReportsPublicationFailureAndKeepsPriorDossier` | read-only directory (create fails, `os.ErrPermission`), file-blocked parent (`ENOTDIR`), destination is a non-empty directory (rename fails); error precedes the gate failure; prior bytes intact; no temp entry |
| `TestRunKeepsPriorDossierWhenWriteFails` (unix) | a child test process with `RLIMIT_FSIZE` 16 and `SIGXFSZ` ignored makes the write fail with `EFBIG` after a real partial write; prior dossier intact; no temp entry |

Mutation check: a publisher body of `os.WriteFile(path, contents, 0o644)` fails all four tests.

Not injected: close, file-sync, directory-sync and cleanup failures. A real filesystem cannot produce
them without a seam; the upgrade package has none and `internal/hostfs` is outside this task's
Touches. They return through the same single `hostfs.Replace` error path the injected failures use.
Follow-up for the hostfs owner: fault-injection tests for those stages in `internal/hostfs`.

## Commands and results (all exit 0)

Run from `tools/gomad3` with `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off`:

- baseline before any edit: `.toolchain/bin/go test -tags test_dep ./upgrade/... ./internal/hostfs ./cmd/gomadtool` and `.toolchain/bin/go test -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges|TestUpgradeOrchestrationIsAboveToolchain'`
- after: `.toolchain/bin/go test -count=1 -tags test_dep ./upgrade/... ./internal/hostfs ./cmd/gomadtool` (upgrade: 13 top-level tests pass, none skipped)
- after: `.toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges|TestUpgradeOrchestrationIsAboveToolchain' -v` (3 pass)
- `gofmt -l upgrade architecture_test.go` (no output), `.toolchain/bin/go vet -tags test_dep ./upgrade/... .`
- `GOOS=linux GOARCH=amd64 .toolchain/bin/go vet -tags test_dep ./upgrade/` (compile check only, not a linux test run)
- `make -C tools/gomad3 validate` (run once, while fn-108.2's uncommitted edits were in the tree)

Not run: the full `make -C tools/gomad3 test` (conductor instruction: fn-108.2 edits other packages
concurrently; `flowctl gate classify` reports FULL because of the uncommitted `.plans/GOMAD_MILESTONES.md`
from earlier tasks; no gate receipt written). linux/amd64 gates: not run (no host).

## Size and API

`git diff HEAD --numstat -- tools/gomad3/upgrade tools/gomad3/architecture_test.go tools/gomad3/ARCHITECTURE.md`:

| File | Added | Removed | Class |
| --- | --- | --- | --- |
| `tools/gomad3/upgrade/upgrade.go` | 2 | 21 | production-go |
| `tools/gomad3/upgrade/upgrade_test.go` | 248 | 0 | test-go |
| `tools/gomad3/upgrade/upgrade_unix_test.go` (untracked) | 75 | 0 | test-go |
| `tools/gomad3/architecture_test.go` | 1 | 1 | test-go |
| `tools/gomad3/ARCHITECTURE.md` | 5 | 1 | other |

`size-count.sh` rule v2 for `upgrade/upgrade.go`: 546 -> 527 physical lines, 515 -> 496 code lines,
15949 -> 15428 code bytes (production Go: -19 code lines).

Public API: `go doc -all go.temporal.io/server/tools/gomad3/upgrade` under the `api-capture.sh`
environment is byte-identical to `api-baseline/go-doc/upgrade.txt` (empty diff).

## Review

`task4-review.md`: raw codex bridge, gpt-5.6-sol at high, one round, SHIP with no findings. The reviewed
diff is `task4-working-tree.diff`.
