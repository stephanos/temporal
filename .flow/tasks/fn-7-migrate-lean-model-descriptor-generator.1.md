---
satisfies: [R2, R3]
---
# fn-7-migrate-lean-model-descriptor-generator.1 Extract shared artifact and protobuf path modules

## Description
Relocate the two support packages required by R2 before moving the descriptor command. Keep the Lean API generator buildable and give both deep modules direct regression coverage.

**Size:** M
**Files:** `tools/common/artifactio/artifact.go`, `tools/common/artifactio/artifact_test.go`, `tools/common/protofile/prefix.go`, `tools/common/protofile/prefix_test.go`, `tools/umpire/internal/artifactio/artifact.go`, `tools/umpire/internal/protofile/prefix.go`, `tools/umpire/internal/generate/api/main.go`, `tools/umpire/internal/export/godescriptors/run.go`
**Touches:** [tools/common/artifactio/**, tools/common/protofile/**, tools/umpire/internal/artifactio/**, tools/umpire/internal/protofile/**, tools/umpire/internal/generate/api/main.go, tools/umpire/internal/export/godescriptors/run.go]

### Approach
- Move the existing `Publish`/`Remove` and `NormalizePrefix` implementations without changing their public APIs or existing comments.
- Replace the Umpire-branded artifact temporary prefix with a generic Temporal-tool prefix while preserving same-directory creation, modes, sync, rename, and joined-error behavior.
- Add isolated table-driven tests for invalid artifact paths, create/replace/remove behavior, directory and file modes, separator normalization, trailing slashes, absolute paths, current-directory paths, and parent traversal.
- Add a controlled publication-failure case that leaves the prior destination intact; keep lower-level close/sync/rename error paths structurally unchanged rather than adding a filesystem abstraction solely for fault injection.
- Update the Lean API generator and existing descriptor exporter imports to consume the common packages; do not otherwise refactor their behavior.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/artifactio/artifact.go:10-73` — atomic publication and removal contract
- `tools/umpire/internal/protofile/prefix.go:10-24` — prefix normalization contract
- `tools/umpire/internal/generate/api/main.go:1-60` — artifact helper consumer
- `AGENTS.md:1-76` — Go, comments, testing, and verification conventions

**Optional** (reference as needed):
- `tools/umpire/internal/generate/api/config_test.go` — existing prefix-facing tests
- `tools/umpire/internal/generate/api/main_test.go` — existing artifact publication tests

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile ./tools/umpire/internal/generate/api
```
## Acceptance
- [ ] Common artifact and prefix packages preserve the existing APIs, permissions, validation, sync, and error behavior.
- [ ] Focused direct tests cover successful behavior, invalid paths/prefixes, and a controlled publication failure using `require`.
- [ ] The controlled failure leaves the prior destination intact, while lower-level error paths remain structurally unchanged.
- [ ] The Lean API generator and existing descriptor exporter import the common packages and their focused tests pass.
- [ ] The former helper package directories are removed without altering unrelated Umpire internals or comments.
## Done summary
Moved atomic artifact publication and protobuf-prefix normalization into `tools/common`, updated the Lean API generator and existing descriptor exporter imports, and removed the old internal helper implementations. Added direct `require`-based regression tests covering validation, normalization, permissions, replacement/removal, and controlled failure safety; the artifact implementation is structurally unchanged except for `.temporal-artifact-*`.

TDD: `go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile` first failed on the intentionally missing `Publish`, `Remove`, and `NormalizePrefix` APIs, then passed after the move.

Diff scope:
- `tools/common/artifactio/artifact.go`
- `tools/common/artifactio/artifact_test.go`
- `tools/common/protofile/prefix.go`
- `tools/common/protofile/prefix_test.go`
- `tools/umpire/internal/artifactio/artifact.go` (removed)
- `tools/umpire/internal/protofile/prefix.go` (removed)
- `tools/umpire/internal/generate/api/main.go`
- `tools/umpire/internal/export/godescriptors/run.go`

No files were staged or committed. Task was recorded `done` after conductor verification and receipt recording.

stage: impl-review - failed(tooling: Codex wrapper reviewed the intentionally empty committed range and cannot include the user-owned uncommitted worktree; verdict NEEDS_WORK contained no code findings) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: red (expected: common packages absent before TDD extraction), go test -count=1 -tags test_dep ./tools/common/artifactio ./tools/common/protofile ./tools/umpire/internal/generate/api, go test -count=1 -tags test_dep ./tools/umpire/internal/export/godescriptors, git diff --check
- PRs: