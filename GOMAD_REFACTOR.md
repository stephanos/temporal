# GoMaD refactoring roadmap

## Scope

This roadmap covers the runtime-simulation implementation imported from gomad
and now located in [`tools/gomad`](tools/gomad). It addresses file and package
structure, naming, dependency ownership, test organization, and the boundary
between translated code and the simulator runtime.

The previous AST-rewrite implementation in `tools/gomad_old` is outside the
design scope. Its tests may be used as a one-time source of missing scenarios,
but its package structure is not a target architecture for the new GoMaD.

The imported tree is a nested Go module with 27 loadable packages. Its
production import graph is acyclic and its main package boundaries are broadly
sound. The primary structural debt is concentrated in oversized files,
scattered Go-version policy, a mixed product identity, and duplicated protocol
and logging concepts. The roadmap therefore favors splitting files and
deepening existing modules before creating more packages.

## Priority and effort

| Level  | Meaning                                                                             |
| ------ | ----------------------------------------------------------------------------------- |
| **P0** | Blocks a coherent GoMaD adoption path or makes compatibility results misleading.    |
| **P1** | High-value maintenance or correctness work that should precede feature expansion.   |
| **P2** | Structural improvement that becomes valuable after the P0/P1 boundaries are stable. |
| **P3** | Optional cleanup with limited architectural impact.                                 |

| Effort | Meaning                                                                          |
| ------ | -------------------------------------------------------------------------------- |
| **S**  | Localized deletion, rename, documentation, or helper extraction.                 |
| **M**  | Multi-file refactor within an existing package or a bounded caller migration.    |
| **L**  | Cross-package or translated-ABI change requiring regeneration and broad testing. |

## Recommended decisions

### 1. Keep the engine isolated behind a GoMaD-facing boundary

Keep the nested module so simulator, translator, SQLite, and testing
dependencies remain isolated from most of Temporal's main module. Use GoMaD
names consistently across the nested module, command, and repository
integration.

The immediate target should be:

- `tools/gomad_old` remains the legacy implementation;
- the repository-facing command and build-tag path select the runtime simulator;
- the user-facing executable is `gomad`;
- Gomad-prefixed internal names are either intentionally documented as upstream
  engine names or renamed together in a later, atomic fork decision.

Do not leave two commands claiming to be GoMaD or perform a piecemeal module,
package, cache, environment-variable, and directive rename.

### 2. Make standard-library compatibility version-neutral at the package level

Rename [`internal/hooks/go123`](tools/gomad/internal/hooks/go123) to a stable
boundary such as `internal/stdlib/hooks`. Put release-specific policy in files
such as `policy_go126.go` and keep truly version-specific adaptations in
`go126.go` files. Supporting a new Go release should change one explicit
compatibility policy rather than package paths throughout generated and
translated output.

### 3. Split files before splitting core packages

`gomadruntime`, `internal/simulation`, and `internal/translate` are tightly
coupled internally. First expose their existing responsibilities as focused
files and narrow helpers. Introduce new package boundaries only after those
dependencies are visible and can be moved without cycles.

### 4. Treat the translated runtime surface as an ABI

Translated external packages must import runtime support, but they should not
implicitly depend on the whole scheduler implementation. After the runtime
file split, define a narrow `runtimeabi` surface and move scheduler internals
behind it incrementally. This is a long-term boundary, not part of the initial
rename or cleanup.

### 5. Preserve upstream-diffable compatibility sources

Keep copied `reflect`, `testing`, race, channel, semaphore, and generated
syscall files in source-compatible boundaries. Add an exact Go tag/commit and
source-path manifest; several headers currently point to mutable `master`.
Do not split copied standard-library files merely because they are large.

## Roadmap summary

| ID     | Priority | Effort | Depends on     | Outcome                                                                             |
| ------ | -------- | ------ | -------------- | ----------------------------------------------------------------------------------- |
| RF-001 | P0       | S      | None           | Remove the Bolt and etcd examples and their exclusive dependency graph.             |
| RF-002 | P0       | S/M    | None           | Fix or delete the misleading cross-architecture test harness.                       |
| RF-003 | P0       | M      | None           | Wire the repository GoMaD command and tests to the runtime implementation.          |
| RF-004 | P0       | M      | RF-003         | Establish and document the GoMaD-facing identity boundary.                          |
| RF-101 | P0       | L      | RF-004         | Replace the stale `go123` package path with a stable stdlib boundary.               |
| RF-102 | P1       | M      | RF-101         | Consolidate hook, linkname, assembly, skip, and target-version policy.              |
| RF-103 | P1       | S      | RF-101         | Pin copied standard-library source provenance to an exact Go revision.              |
| RF-201 | P1       | M      | RF-004         | Split Linux syscall implementation by subsystem.                                    |
| RF-202 | P1       | M      | RF-004         | Split scheduler/runtime implementation by responsibility.                           |
| RF-203 | P1       | M      | None           | Split filesystem state, namespace, I/O, crash, and mmap responsibilities.           |
| RF-204 | P1       | M      | RF-101         | Turn translation orchestration and rewrite ordering into explicit stages.           |
| RF-205 | P2       | S/M    | RF-004         | Split CLI subcommands and centralize Go command construction.                       |
| RF-206 | P2       | M      | RF-201         | Split network packets, listeners, buffers, and streams.                             |
| RF-301 | P2       | L      | RF-202, RF-204 | Introduce a narrow translated-code runtime ABI.                                     |
| RF-302 | P2       | M      | RF-004         | Replace the `internal/gomadtool` grab bag with owned modules.                       |
| RF-303 | P2       | M      | None           | Consolidate trace parsing around one typed event schema.                            |
| RF-304 | P2       | M      | RF-202         | Share the metatest subprocess protocol and implement runner cleanup.                |
| RF-401 | P1       | S      | None           | Remove inert tests, obsolete commented implementations, and private dead symbols.   |
| RF-402 | P1       | M      | RF-201–RF-204  | Split behavioral tests by capability and failure mode.                              |
| RF-403 | P0       | M/L    | RF-003         | Add coverage gates for translator, runtime, syscall, crash, and determinism claims. |

## Phase 0: dependency and CI hygiene

### RF-001: remove heavyweight compatibility examples

Status: approved for implementation after review of this roadmap.

Delete:

- `tools/gomad/examples/bolt`;
- `tools/gomad/examples/etcd`;
- the corresponding entries in `tools/gomad/README.md`.

Remove from the nested module:

- `go.etcd.io/bbolt`;
- `go.etcd.io/etcd/client/pkg/v3`;
- `go.etcd.io/etcd/client/v3`;
- `go.etcd.io/etcd/pkg/v3`;
- `go.etcd.io/etcd/raft/v3`;
- `go.etcd.io/etcd/server/v3`;
- transitive requirements retained only by those modules.

Run `go mod tidy` rather than manually guessing the transitive dependency set.
Retain `go.uber.org/zap`: it is also used by the behavior logging tests. Retain
the small examples in `tools/gomad/examples`; they exercise the public API
without dominating the module graph.

Verification:

- no source or documentation reference remains to the deleted examples;
- `go mod why` reports no path to bbolt or etcd modules;
- all nested-module packages load with `-tags=test_dep`;
- core and behavior tests do not depend on the removed examples.

### RF-002: repair the architecture test claim

DONE

## Phase 1: adoption and identity

### RF-003: connect the new engine to the repository entry point

The repository command at `cmd/tools/gomad` and the `gomad`-tagged tests still
select the previous AST implementation. Define one adoption path that builds
and invokes the nested runtime implementation, then rename legacy entry points
to `gomad_old` where they must remain temporarily.

The boundary should own:

- command-line compatibility required by Temporal CI;
- propagation of seeds and simulator arguments;
- test-package selection and build tags;
- translated artifact and cache locations;
- clear errors when the nested tool has not been built or prepared.

### RF-004: make naming intentional

Use `gomad` consistently for user-facing command names and Temporal integration.
Document internal Gomad names as upstream engine names until an explicit fork
decision justifies an atomic rename. If a full rename is chosen later, change
the module path, root package, runtime imports, command, cache directory,
environment variables, directives, generated files, scripts, and goldens in
one change.

Avoid an intermediate state where directory names say GoMaD while command and
runtime diagnostics still ambiguously refer to a different product.

### Public API cleanup

Before the new implementation becomes the repository default:

- internalize `Machine.GetInodeInfo` and its duplicated `InodeInfo`; only an
  internal disk test currently uses them;
- delete or redesign `SetSometimesCrashOnSync`; it has no callers and currently
  ignores its Boolean input;
- decide whether `nemesis` is a supported public policy package or an internal
  scenario helper before expanding it.

Exit criteria for Phase 1:

- one documented GoMaD command reaches the runtime simulator;
- the old and new implementations cannot be selected accidentally under the
  same command or build-tag identity;
- public APIs do not expose test-only filesystem internals or ignored controls.

## Phase 2: Go-version compatibility boundary

### RF-101: replace `go123`

Move the hook package to a version-neutral path and regenerate architecture
proxies. Rename `hooksGo123`, `keepAsmPackagesGo123`, skipped-package tables,
accepted-linkname tables, and related generated selectors as one migration.
Translated caches must be invalidated because the import path changes.

### RF-102: centralize compatibility policy

Represent target-version behavior with one policy value containing:

- packages skipped or translated;
- packages retaining assembly;
- selector hooks and linknames;
- accepted no-body symbols;
- runtime and standard-library replacement import paths;
- target Go release and supported architectures.

Keep syntax or API differences in release-named files. A Go upgrade should be
reviewable as a policy diff plus focused compatibility implementations.

### RF-103: record copied-source provenance

Add a manifest containing the exact Go release/tag, commit, source path, local
destination, and intentional divergence for copied files. Preserve their file
boundaries to keep comparison with Go sources mechanical.

Exit criteria for Phase 2:

- no `go123` path or identifier remains;
- one manifest describes the supported Go release and copied sources;
- generated amd64 and arm64 proxies match the centralized policy;
- compatibility tests cover every locally implemented Go 1.26 hook family.

## Phase 3: split structural hotspots

These are file-level splits first. Preserve behavior, comments, package names,
and generated interfaces during the moves.

### RF-201: Linux syscall adapter

Split the 1,823-line `internal/simulation/os_linux.go` into focused files:

- `linux_files.go`;
- `linux_network.go`;
- `linux_poll.go`;
- `linux_process.go`;
- `linux_memory.go`;
- `syscall_trace.go`.

After the split, define a narrow machine-resource and crash-callback interface.
Only then evaluate moving the adapter into an `internal/simulation/linux`
package.

### RF-202: simulator runtime

Split `gomadruntime/runtime.go` into:

- `errors.go`;
- `machine.go`;
- `scheduler.go`;
- `run.go`;
- `goroutine.go`;
- `runtime_api.go`.

Extract the repeated scheduler bootstrap used by normal runs, shared-global
initialization, and benchmarking. Keep the package intact during this phase to
avoid scheduler/channel/timer/race cycles.

### RF-203: filesystem

Split `internal/simulation/fs/filesystem.go` into state, namespace operations,
I/O, crash persistence, and mmap files. Keep `chunkedfile.go` and
`pendingops.go` separate; they already provide deep, independently testable
modules.

Move the map-reflection adapter from the tail of `gomadruntime/map.go` to
`reflect_map.go` without changing packages.

### RF-204: translator pipeline

Rename `internal/translate/main.go` to `pipeline.go` and separate:

- package loading and graph planning;
- hashing and cache lookup;
- ordered rewrite passes;
- translated-source emission;
- dependency manifest construction.

Turn the implicit `preApply` ordering contract into named phases. Split
`types.go` into type-expression rendering and implicit-conversion analysis, and
deduplicate alias/named rendering. Extract shared generic/named container
analysis used by map and channel rewriting.

### RF-205 and RF-206: secondary hotspots

- split CLI subcommands into `runTranslate`, `runTest`, `runBuildTests`,
  `runViewer`, and `runDebug` files;
- centralize translated `go test` flag construction across CLI and self-tests;
- split network packet, listener, buffer, and stream responsibilities;
- split syscall generator parsing, model, proxy generation, dispatch generation,
  and syscall-number generation;
- keep generated outputs checked in.

Exit criteria for Phase 3:

- no behavior changes or package import changes are mixed with file moves;
- every split package passes its focused unit and behavior tests;
- translator pass ordering is explicit and tested;
- generated output is unchanged except where a prior approved rename requires it.

## Phase 4: deepen ownership boundaries

### RF-301: translated runtime ABI

Define the operations translated code is allowed to call and expose them from a
narrow package. Move scheduler, machine lifecycle, logging, and test-runner
implementation behind internal packages only when doing so does not create
cycles. Treat ABI changes as cache-version changes and add translator/runtime
compatibility tests.

### RF-302: dismantle `internal/gomadtool`

Move responsibilities to owners instead of creating another generic utility
package:

- translated output layout and `BuildConfig` to an `internal/layout` module;
- `FindGoMod` and test-module construction to `internal/modutil`;
- precompiled test-binary management to `metatesting` or an internal test-binary
  module;
- subprocess failure policy out of helpers that currently accept `*testing.T`.

### RF-303: unify trace events

Create one typed event schema and JSONL decoder. Keep terminal pretty-printing,
viewer handlers, and metatesting assertions as separate consumers. Do not merge
the viewer and terminal renderer packages simply to reduce package count.

### RF-304: share the metatest protocol

Move duplicated run request/result structures into a small internal protocol
package. Implement runner cleanup through `t.Cleanup` so global runner entries
and child processes do not live for the duration of the test process.

Exit criteria for Phase 4:

- translated code depends on an explicit, tested ABI;
- layout, module, test-binary, trace, and protocol concepts each have one owner;
- package extraction does not introduce cycles or make public APIs broader.

## Phase 5: tests and dead-code cleanup

### RF-401: remove inert artifacts

Verified private candidates with no callers include:

- `wrappedChan` in `internal/reflect/value.go`;
- `(*fnv64).Sum` in `gomadruntime/fnv64.go`;
- `logInitialized` in `internal/simulation/userspace.go`;
- `Stack.Endpoint` in `internal/simulation/network/stack.go`;
- `NewEmptyFilesystem` and the no-op `Filesystem.Release`.

Delete or restore intentionally skipped/comment-only tests:

- `internal/tests/race/testdata/io_test.go`;
- `internal/tests/race/testdata/os_test.go`, which imports a removed bridge and
  is permanently tagged `sim && skip`;
- `internal/tests/behavior/nemesis_test.go`;
- `internal/tests/behavior/nemesis_meta_test.go`.

Remove abandoned commented-out implementations in a dedicated cleanup while
preserving explanatory, compatibility, and provenance comments. The largest
blocks are in `net_test.go` and `disk_crash_test.go`.

The ignored `.gomad` directory is generated cache state, not source. It may be
deleted locally when reclaiming space but should not become a tracked cleanup
change.

### RF-402: organize behavioral tests by contract

After removing obsolete blocks, split large suites by modeled capability:

- disk I/O, namespace, mmap, sync, and crash durability;
- TCP, HTTP, gRPC, listener lifecycle, partition, and restart behavior;
- map operations, generics, iteration, and reflection;
- scheduler lifecycle, panic, deadlock, and crash behavior.

Use native differential tests where the same contract should match ordinary Go
or Linux. Borrow scenarios from `tools/gomad_old` only when they test a runtime
claim that the new implementation also makes; port the scenario to the new
public API rather than importing old test helpers.

### RF-403: make coverage claims executable

Maintain a compatibility matrix covering:

- translator syntax and type-system transformations;
- channel, map, select, panic, semaphore, and timer semantics;
- package-global and machine isolation;
- syscall support and explicit unsupported behavior;
- network partition, crash, restart-generation, and half-open connection cases;
- filesystem persistence and crash-state exploration;
- deterministic replay checksums and trace equivalence;
- race-detector integration;
- amd64 and arm64 generation and execution.

Every supported claim should point to a focused test. Unsupported or approximate
semantics should point to a documented limitation rather than silently falling
through to native behavior.

Exit criteria for Phase 5:

- no permanently skipped or comment-only test file remains without an owner;
- behavioral suites are discoverable by capability;
- the compatibility matrix links every supported boundary to executable tests;
- full normal and race-enabled simulator acceptance suites pass.

## Rename inventory

After the identity and stdlib decisions are settled, apply these local clarity
renames independently of any product-wide rebrand:

| Current                        | Recommended                            |
| ------------------------------ | -------------------------------------- |
| `internal/translate/main.go`   | `pipeline.go`                          |
| `internal/translate/go.go`     | `goroutine_rewrite.go`                 |
| `internal/translate/tests.go`  | `test_rewrite.go`                      |
| `internal/translate/cache.go`  | `cache_codec.go` or `package_cache.go` |
| `internal/testing/missing.go`  | `unsupported.go`                       |
| `internal/reflect/no.go`       | `unsupported_linknames.go`             |
| `internal/simulation/gomad.go` | `control.go`                           |
| `internal/gomadlog/main.go`    | `event.go` or `record.go`              |
| `internal/gomadviewer/main.go` | `server.go`                            |
| `getDecriptor`                 | `getDescriptor`                        |
| `ErrPaniced` / `parkPaniced`   | `ErrPanicked` / `parkPanicked`         |

Retain a deprecated alias for an exported typo when translated artifacts or
external consumers may still reference it.

## Consolidate, but do not merge packages

Consolidate these duplicated concepts behind one owner:

- stdlib hook and translation policy;
- map/channel generic container analysis;
- scheduler bootstrap;
- CLI/self-test Go command construction;
- trace event parsing;
- metatest subprocess protocol.

Keep these package boundaries:

- `internal/simulation/fs`, `network`, and `syscallabi`;
- `internal/coro` and `internal/race`;
- `internal/translate/cache`;
- `metatesting` and `nemesis` outside the root API;
- viewer and terminal trace rendering as separate consumers;
- small source-package-oriented hook files;
- checked-in generated syscall, protobuf, and stringer outputs.

## Ordering and dependencies

The recommended sequence is:

1. remove heavyweight examples and repair false CI claims;
2. wire the runtime implementation to the repository GoMaD entry point;
3. settle the user-facing/upstream identity boundary;
4. replace the stale Go-version compatibility package and record provenance;
5. split large files without changing packages;
6. introduce runtime ABI and ownership packages where the file splits expose a
   stable interface;
7. reorganize tests and make compatibility coverage executable.

Do not combine steps 2–4 into one mechanical rename. Each changes translated
paths, generated code, cache identity, or repository integration and needs its
own acceptance evidence.

## Risk controls

- **Hard cutover:** keep mechanical renames separate from semantic ports and do
  not retain compatibility aliases for the previous package or command names.
- **Generated code:** regenerate whenever hook paths, syscall interfaces, or
  `//go:generate` inputs change; review generated diffs separately.
- **Translated caches:** version or invalidate caches when imports, runtime ABI,
  or rewrite policy changes.
- **Package cycles:** begin with same-package file splits and prove narrow
  interfaces before package extraction.
- **Behavior drift:** use differential and replay tests for refactors affecting
  maps, channels, scheduling, filesystem, or networking.
- **Performance:** retain scheduler and trace benchmarks when extracting
  interfaces; avoid adding per-operation allocations to runtime ABI adapters.
- **Scalability:** bound crash-state exploration and runner lifetime before
  increasing workload size or CI parallelism.
- **Security:** syscall trace payloads can contain application or storage data;
  make capture explicit and treat emitted traces as sensitive artifacts.

## Verification gates

Run focused checks after each roadmap item, followed by the nested-module
acceptance sequence at phase boundaries:

```text
go mod tidy
go list -tags=test_dep ./...
go build -tags=test_dep -o .gomad/gomadtool ./cmd/gomad
go test -ldflags=-checklinkname=0 -tags=linkname,test_dep ./gomadruntime
.gomad/gomadtool prepare-selftest
.gomad/gomadtool test ./internal/tests/behavior ./nemesis
.gomad/gomadtool test -race ./internal/tests/behavior ./nemesis
```

Use the repository's `make lint-code` gate after changes connect the nested tool
to Temporal packages. Architecture-specific hook or syscall changes additionally
require genuine amd64 and arm64 verification.
