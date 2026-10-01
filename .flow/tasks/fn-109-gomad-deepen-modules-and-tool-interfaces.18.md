---
satisfies: [R12]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.18 Select backend-specific filesystem handle and mapping implementations at creation

## Description
Stage 5, second half of R12: the filesystem handle and mapping family, using the pattern proven on network handles in task 17. `Handle` and `Mapping` carry a `processHandle` next to local in-memory state, and about sixteen handle operations branch to `processHandle*` functions.

**External coordination:** overlay edit; same fn-110 and toolchain-rebuild rules as the simulation-time task.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/{fs.go,process_volume.go,volume.go,runtime.go}`, new per-backend files, `fs_test.go`, `toolchain/version/version.json` for new overlay files.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/**, tools/gomad3/toolchain/runtime/overlay/src/os/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3sim/*_test.go]

### Approach
- Current shape: `Handle.processHandle` (`fs.go:87`) and `Mapping.processHandle` (`:107`); `processFilesystem = &FS{process: true}` (`process_volume.go:25`). Handle dispatch to the process backend at `fs.go:802`, `:826` (read, read-at), `:855`, `:907` (write, write-at), `:957` (truncate), `:1005` (chmod), `:1031` (chtimes), `:1054` (chdir), `:1079` (seek), `:1107` (stat), `:1129` (readdir), `:1174` (close), `:1195` (sync), `:1209` (map); process side in `process_volume.go:95-195`.
- Target: open/create and map select a private implementation once; `Handle` and `Mapping` keep their exported operations and the patched `os` adapter and libc adapter callers (`internal/gomadio/libc.go`) do not change. Use the typed volume commands from task 14 inside the process implementation.
- Keep three things explicit rather than abstracted away: mapping capabilities (writable mapping behaviour and which backend supports it), mount immutability for read-only mounts, and capacity accounting. In-process restart must not claim fresh globals or hard cleanup that only process nodes provide.
- Volume semantics stay in `volume.go` / `simulation_volume.go` (persisted and volatile views, sync, crash selection); do not copy them into handle implementations.
- Preserve stale-incarnation rejection, partial read/write with error, deadline-free blocking behaviour, deterministic timestamps and directory order, and validation-before-mutation on replay.
- Reuse task 17's shared-operation test shape: one table run against standalone, in-process and process backends.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs.go:60-130,780-1230`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/process_volume.go` (272 lines)
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/fs_test.go`
- `tools/gomad3sim/volume_toolchain_test.go`, `tools/gomad3/runner/internal/execution/io_filesystem_toolchain_test.go`
- task 17's network implementation (pattern to mirror)
**Optional:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/libc.go` (mapping callers)

### Quick commands
```bash
cd tools/gomad3
make generate && make validate
.toolchain/bin/go test -count=1 -tags test_dep internal/gomadfs internal/gomadio
make toolchain && make overlay-test
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Filesystem|Libc|Sqlite'
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Volume|Process|Backend'
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Filesystem handle and mapping creation selects one private backend implementation; handle operations no longer branch on `processHandle`.
- [ ] Mapping capabilities, read-only mount immutability and capacity accounting remain explicit per backend; volume semantics are shared, not duplicated.
- [ ] Shared operation tests pass for standalone, in-process and process backends; backend-specific tests retain hard-isolation and mapping distinctions.
- [ ] Partial I/O, capacity, stale incarnation, close and replay divergence behave as before, with validation before mutation.
- [ ] Overlay inventories validate, the toolchain rebuilds, `make overlay-test` and the gomad3sim volume/process tests pass on darwin/arm64; linux/amd64 is recorded as incomplete.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
