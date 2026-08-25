---
satisfies: [R1, R2, R3, R4]
---
# fn-7-migrate-lean-model-descriptor-generator.2 Hard-cutover the Lean model descriptor command

## Description
Move the descriptor implementation and fixtures into the common tool surface, replace the old entrypoint with `genleanmodeldescriptors`, and complete repository integration for R1-R4.

**Size:** M
**Files:** `tools/common/godescriptors/**`, `cmd/tools/genleanmodeldescriptors/main.go`, `cmd/tools/genleanmodeldescriptors/main_test.go`, removed Umpire exporter paths and fixtures, `Makefile`, the current descriptor-generator design, the prior implementation plans, `.gitignore`, and the approved migration design where needed
**Touches:** [tools/common/godescriptors/**, cmd/tools/genleanmodeldescriptors/**, tools/umpire/cmd/umpire-export-go-descriptors/**, tools/umpire/internal/export/godescriptors/**, tools/umpire/testdata/godescriptors/**, tools/umpire/testdata/godescriptorscompat/**, Makefile, docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md, docs/superpowers/specs/2026-08-24-genleanmodeldescriptors-migration-design.md, .turbo/plans/descriptor-driven-lean-generator.md, .turbo/plans/simplify-lean-api-output.md, .turbo/plans/lean-declaration-plan.md, .gitignore]

### Approach
- Move the exporter tests and fixtures first and verify the new package fails to build without its implementation. Then move the implementation and update fixture import paths to complete the red-green refactor.
- Extend the common-module tests so package-list failures, empty selections, canceled contexts, and a controlled helper failure leave a pre-existing destination unchanged; use a narrow package-local cleanup seam only if needed to assert joined cleanup errors without changing the exported API.
- Consume the common artifact and prefix packages, preserve `Run` behavior and comments, and rename only the flag-set and temporary-helper labels to `genleanmodeldescriptors`.
- Add the thin standard command entrypoint and a subprocess-level test covering missing flags, invalid flags, and positional arguments; assert non-zero exits and the `genleanmodeldescriptors:` stderr prefix. Delete the former entrypoint and exporter package.
- Rename the Makefile command variable and path while retaining its arguments and `UMPIRE_PUBLIC_BINPB` output.
- Add a narrow ignore exception so the relocated testdata fixtures remain trackable.
- Update live code, current design documentation, and the related prior implementation plans, then search for stale references outside `.flow` migration records and the approved migration design.
- Generate a temporary public descriptor set with the new command and compare it byte-for-byte with the checked-in artifact; do not rewrite the checked-in file when unchanged.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/cmd/umpire-export-go-descriptors/main.go:1-18` — thin entrypoint pattern and old diagnostic
- `tools/umpire/internal/export/godescriptors/run.go:37-180` — behavior-preserving implementation move
- `tools/umpire/internal/export/godescriptors/run_test.go:14-73` — focused behavior and fixture paths
- `Makefile:118-124` and `Makefile:493-503` — command variable and sole integration caller
- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md:182-214` — current command contract to revise

**Optional** (reference as needed):
- `cmd/tools/gendynamicconfig/main.go` — neighboring `gen*` command layout
- `.turbo/plans/descriptor-driven-lean-generator.md:159-199` — obsolete implementation references

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/common/godescriptors ./cmd/tools/genleanmodeldescriptors
go build -tags test_dep ./cmd/tools/genleanmodeldescriptors
```
## Acceptance
- [ ] The relocated tests demonstrate red before implementation and green after the move.
- [ ] Common-module failure tests leave pre-existing output intact and preserve cleanup-error joining where exercised.
- [ ] Subprocess tests verify non-zero exits and the new stderr prefix for missing flags, invalid flags, and positional arguments.
- [ ] The new command and Make integration preserve flags, deterministic output, transitive imports, and failure behavior with the new diagnostic name.
- [ ] The relocated fixtures are visible to Git, and the old command, implementation, fixture paths, variable, and stale live references are absent outside migration records.
- [ ] Freshly generated public descriptors equal the checked-in artifact byte-for-byte.
- [ ] Existing comments are preserved and focused tests, import formatting, and `make lint-code` pass without adding drift or CI machinery.
## Done summary
Hard-cut over the registered Go descriptor generator to `cmd/tools/genleanmodeldescriptors` and `tools/common/godescriptors`, moved its fixtures, switched the Make integration, removed the old Umpire command/exporter/fixture paths, and updated the current design plus prior implementation plan. The common module retains `Run(context.Context, []string) error`, deterministic transitive descriptor collection, atomic publication, and cleanup-error joining; focused failure tests now prove package-list, empty-selection, cancellation, and controlled helper failures leave an existing output unchanged.

TDD evidence:
- Baseline Quick commands were red because neither destination package existed yet.
- After moving the exporter tests and fixtures first, `go test -count=1 -tags test_dep ./tools/common/godescriptors` failed with undefined `Run` and `listDescriptorPackages` (expected RED).
- After adding all failure-safety cases but before the implementation move, the same test also failed with undefined `removeDescriptorHelper` and `exportDescriptors` (expected RED).
- After moving the implementation and renaming its flag-set and temporary-helper labels, the common-module test passed (GREEN).
- Before adding the new entrypoint, `go test -count=1 -tags test_dep ./cmd/tools/genleanmodeldescriptors` failed because the package had no non-test Go files (expected RED).
- After adding the new thin entrypoint and deleting the old one, the command test passed (GREEN), including subprocess checks for missing flags, an invalid flag, and a positional argument.

Verification:
- `go test -count=1 -tags test_dep ./tools/common/godescriptors ./cmd/tools/genleanmodeldescriptors` passed.
- `go build -tags test_dep ./cmd/tools/genleanmodeldescriptors` passed; the generated root binary was removed afterward.
- Fresh generation to `/tmp/tmp.urwycBHPHE/umpire-public.binpb` passed and `cmp` against `proto/umpire-public.binpb` returned 0; the checked-in artifact was not rewritten.
- Focused golangci-lint on both new packages passed with 0 issues.
- `git diff --check` passed.
- The stale-reference scan outside `.flow/**` and the approved migration design returned no matches.
- A narrow `.gitignore` exception keeps the relocated `tools/common/godescriptors/testdata` fixtures visible to Git, and the six empty legacy directories were removed.
- Exact helper-path references in the descriptor, Lean declaration, and Lean API simplification implementation plans now point at `tools/common/artifactio`.
- Conductor verification reran the combined artifact, prefix, descriptor, API-generator, and command tests; the temp-output build, descriptor byte comparison, focused lint, and diff hygiene all passed.
- `make lint-code` did not reach green: the first run failed because a Go build temp directory exhausted `/tmp`; the retry completed analysis but reported 1,811 pre-existing branch-wide findings, with no findings naming either new package. Its automatic unrelated fixes were reversed exactly and verified clean.

Diff scope:
- `.turbo/plans/descriptor-driven-lean-generator.md`
- `.turbo/plans/lean-declaration-plan.md`
- `.turbo/plans/simplify-lean-api-output.md`
- `.gitignore`
- `Makefile`
- `cmd/tools/genleanmodeldescriptors/main.go`
- `cmd/tools/genleanmodeldescriptors/main_test.go`
- `docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md`
- `tools/common/godescriptors/run.go`
- `tools/common/godescriptors/run_test.go`
- `tools/common/godescriptors/testdata/godescriptors/descriptors.pb.go`
- `tools/common/godescriptors/testdata/godescriptorsbroken/descriptors.pb.go`
- `tools/common/godescriptors/testdata/godescriptorscompat/descriptors.pb.go`
- `tools/umpire/cmd/umpire-export-go-descriptors/main.go` (removed)
- `tools/umpire/internal/export/godescriptors/run.go` (removed)
- `tools/umpire/internal/export/godescriptors/run_test.go` (removed)
- `tools/umpire/testdata/godescriptors/descriptors.pb.go` (removed)
- `tools/umpire/testdata/godescriptorscompat/descriptors.pb.go` (removed)

No files were staged or committed. The task was recorded `done` after conductor verification and receipt recording.

stage: impl-review - failed(tooling: Codex wrapper accepts only committed base..HEAD ranges and cannot inspect the user-owned uncommitted worktree; no temporary commit was created) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: TDD RED: go test -count=1 -tags test_dep ./tools/common/godescriptors (undefined Run/listDescriptorPackages before implementation move), TDD RED: go test -count=1 -tags test_dep ./cmd/tools/genleanmodeldescriptors (no non-test Go files before entrypoint move), go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile ./tools/common/godescriptors ./tools/umpire/internal/generate/api ./cmd/tools/genleanmodeldescriptors, go build -tags test_dep -o <temp>/genleanmodeldescriptors ./cmd/tools/genleanmodeldescriptors, go run -tags test_dep ./cmd/tools/genleanmodeldescriptors --package-pattern go.temporal.io/api/... --file-prefix temporal/api/ --output <temp>/umpire-public.binpb && cmp <temp>/umpire-public.binpb proto/umpire-public.binpb, .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --config=.github/.golangci.yml ./tools/common/artifactio ./tools/common/protofile ./tools/common/godescriptors ./cmd/tools/genleanmodeldescriptors, git diff --check, stale-reference and removed-path scan (no matches or legacy directories outside .flow migration records), git ls-files --others --exclude-standard tools/common/godescriptors/testdata (all three fixtures visible), make lint-code (not green: first run /tmp no-space; retry reported 1,811 inherited branch-wide findings and no new-package findings)
- PRs: