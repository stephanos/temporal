---
satisfies: [R12]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.18 Select backend-specific filesystem handle and mapping implementations at creation

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 5, second half of R12: the filesystem handle and mapping family, using the pattern proven on network handles in task 17. `Handle` and `Mapping` carry a `processHandle` next to local in-memory state, and about sixteen handle operations branch to `processHandle*` functions.

**External coordination:** overlay edit; same fn-110 and toolchain-rebuild rules as the simulation-time task.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/{fs.go,process_volume.go,volume.go,runtime.go}`, new per-backend files, `fs_test.go`, `toolchain/version/version.json` for new overlay files.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/**, tools/gomad3/toolchain/runtime/overlay/src/os/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3sim/*_test.go, tools/gomad3/filesystem_handles_ownership_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/runner/internal/execution/*_test.go, tools/gomad3/Makefile]

### Approach
- Current shape: `Handle.processHandle` (`fs.go:87`) and `Mapping.processHandle` (`:107`); `processFilesystem = &FS{process: true}` (`process_volume.go:25`). Handle dispatch to the process backend at `fs.go:802`, `:826` (read, read-at), `:855`, `:907` (write, write-at), `:957` (truncate), `:1005` (chmod), `:1031` (chtimes), `:1054` (chdir), `:1079` (seek), `:1107` (stat), `:1129` (readdir), `:1174` (close), `:1195` (sync), `:1209` (map); process side in `process_volume.go:95-195`.
- Target: open/create and map select a private implementation once; `Handle` and `Mapping` keep their exported operations and the patched `os` adapter and libc adapter callers (`internal/gomadio/libc.go`) do not change. Use the typed volume commands from task 14 inside the process implementation.
- Keep three things explicit rather than abstracted away: mapping capabilities (writable mapping behaviour and which backend supports it), mount immutability for read-only mounts, and capacity accounting. In-process restart must not claim fresh globals or hard cleanup that only process nodes provide.
- Volume semantics stay in `volume.go` / `simulation_volume.go` (persisted and volatile views, sync, crash selection); do not copy them into handle implementations.
- Preserve stale-incarnation rejection, partial read/write with error, deadline-free blocking behaviour, deterministic timestamps and directory order, and validation-before-mutation on replay.
- Reuse task 17's reviewed shared-operation test shape: one table run against standalone, in-process and actual process backends, with explicit existing differences rather than uniform expectations. Standalone and in-process share a local filesystem representation; do not duplicate its model to invent a third representation. Include process cases in Runner's root-integration selector; the existing oneNodeVolumeSpec parity helper is in-process only. A nested-module architectural ownership test may analyze production Handle/Mapping ASTs, retaining an expected old-source optional-state failure before migration and passing final implementation. Behavioral tests exercise real operations, not source-text assertions.
- Connect each new process case to the canonical `make test-simulation` filter used by CI and extend task 17's gate-selection regression. Direct-root skips and names excluded by that filter do not deliver process coverage. Preserve the separately selected forward-clock regression and strict-delay watchdog exclusion.

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
- Follow MILESTONES verification instruction 5: commit each verified task separately, including implementation, tests, documentation and Flow records. Keep source-owned unavailable Darwin gates incomplete and acceptance open; transferred Linux gates remain open under fn-128; preserve unrelated changes and push only when authorized.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual development host is `linux/arm64`, which cannot qualify the complete patched toolchain. Keep the required `darwin/arm64` gates open here until their exact native evidence passes. Linux gates remain deferred under fn-128.1/.4/.7.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Filesystem handle and mapping creation selects one private backend implementation; handle operations no longer branch on `processHandle`.
- [ ] Mapping capabilities, read-only mount immutability and capacity accounting remain explicit per backend; volume semantics are shared, not duplicated.
- [ ] Shared operation tests pass for standalone, in-process and process backends; backend-specific tests retain hard-isolation and mapping distinctions.
- [ ] Partial I/O, capacity, stale incarnation, close and replay divergence behave as before, with validation before mutation.
- [ ] Overlay inventories validate, the toolchain rebuilds, `make overlay-test` and the gomad3sim volume/process tests pass on darwin/arm64; linux/amd64 is recorded as incomplete.

## Done summary
Blocked:
# Task 18 acceptance awaits qualified native gates

The filesystem-owner source candidate is frozen, independently reviewed and
passes feasible conductor checks; see handover.md, evidence.json,
source-audit.md and conductor-verification.md. This is not task completion or
R12 acceptance.

Actual host is linux/arm64 and the pinned patched executable is absent.
Pre-edit canonical overlay, Runner and root commands exit 127. The unchanged
task-17 native-toolchain.log records builder exit 2: complete mode supports only
darwin/arm64 and linux/amd64. Task-16 linter-platform.json remains applicable to
the incompatible Mach-O ARM64 linter. Neither unsupported command was retried.

Patched rebuild, native overlay/filesystem/process tests, actual volume IPC,
native virtual time, isolation/replay, full host/test-host and qualification on
both supported platforms remain open. Scratch checks are developmental; native
pipe and root process fixtures have compiled/linked but have not executed.
Preserve D12, resolved D14 and the existing strict-delay watchdog disposition.

Keep Flow acceptance blocked until actual native commands qualify the
integrated source. MILESTONES item 4 permits sequential source advancement after
review, not completion or an acceptance waiver. User owns commits; commits [].

Blocked:
# Task 18 acceptance awaits qualified native gates

The fourteen-file filesystem handle/mapping source candidate is independently
reviewed and checkpointed after committed task 17. See checkpoint-report.md,
checkpoint-verification.json and conductor-checkpoint.md for source-bound
stock-host checks. MILESTONES verification instruction 5 permits committing
verified progress; it does not waive acceptance.

Actual host is linux/arm64. The patched executable remains absent. Original
overlay, Runner and root commands exit 127; the unchanged task-17 builder
receipt exits 2 because complete mode supports only darwin/arm64 and
linux/amd64. The task-16 Mach-O linter receipt also remains applicable.
These unchanged environment failures were not retried.

Both-platform rebuild, native overlay/filesystem/process tests, actual volume
IPC, native virtual time, hard isolation/replay, root gomad3sim and full
host/test-host qualification remain open. Pipe and root-process fixtures have
compiled/linked but have not executed. Developmental shim checks supply no
native acceptance. Preserve D12, resolved D14 and the strict-delay watchdog
disposition. Keep task 18 and R12 open until required native gates pass.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
