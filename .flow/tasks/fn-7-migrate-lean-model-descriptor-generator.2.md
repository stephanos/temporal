---
satisfies: [R1, R2, R3, R4]
---
# fn-7-migrate-lean-model-descriptor-generator.2 Hard-cutover the Lean model descriptor command

## Description
Move the descriptor implementation and fixtures into the common tool surface, replace the old entrypoint with `genleanmodeldescriptors`, and complete repository integration for R1-R4.

**Size:** M
**Files:** `tools/common/godescriptors/**`, `cmd/tools/genleanmodeldescriptors/main.go`, `cmd/tools/genleanmodeldescriptors/main_test.go`, removed Umpire exporter paths and fixtures, `Makefile`, the current descriptor-generator design, the prior implementation plan, and the approved migration design where needed
**Touches:** [tools/common/godescriptors/**, cmd/tools/genleanmodeldescriptors/**, tools/umpire/cmd/umpire-export-go-descriptors/**, tools/umpire/internal/export/godescriptors/**, tools/umpire/testdata/godescriptors/**, tools/umpire/testdata/godescriptorscompat/**, Makefile, docs/superpowers/specs/2026-08-24-descriptor-driven-lean-generator-design.md, docs/superpowers/specs/2026-08-24-genleanmodeldescriptors-migration-design.md, .turbo/plans/descriptor-driven-lean-generator.md]

### Approach
- Move the exporter tests and fixtures first and verify the new package fails to build without its implementation. Then move the implementation and update fixture import paths to complete the red-green refactor.
- Extend the common-module tests so package-list failures, empty selections, canceled contexts, and a controlled helper failure leave a pre-existing destination unchanged; use a narrow package-local cleanup seam only if needed to assert joined cleanup errors without changing the exported API.
- Consume the common artifact and prefix packages, preserve `Run` behavior and comments, and rename only the flag-set and temporary-helper labels to `genleanmodeldescriptors`.
- Add the thin standard command entrypoint and a subprocess-level test covering missing flags, invalid flags, and positional arguments; assert non-zero exits and the `genleanmodeldescriptors:` stderr prefix. Delete the former entrypoint and exporter package.
- Rename the Makefile command variable and path while retaining its arguments and `UMPIRE_PUBLIC_BINPB` output.
- Update live code, current design documentation, and the prior implementation plan, then search for stale references outside `.flow` migration records and the approved migration design.
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
- [ ] The old command, implementation, fixture paths, variable, and stale live references are absent outside migration records.
- [ ] Freshly generated public descriptors equal the checked-in artifact byte-for-byte.
- [ ] Existing comments are preserved and focused tests, import formatting, and `make lint-code` pass without adding drift or CI machinery.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
