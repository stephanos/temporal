# Extract Umpire into temporalio/umpire

Status: draft execution plan. Destination: <https://github.com/temporalio/umpire>.
This document authorizes no migration, remote creation, push, release or source deletion.

## Outcome and decisions

Umpire becomes a standalone, system-independent Rust workspace with a Go wasm host.
Preserve its relevant Git history, including the earlier Rust paths. Temporal consumes
the local checkout during development: an uncommitted Umpire change must be usable
without a commit, push, package publication or binary download.

The Go host now lives at `model/umpire/hosts/go`, following the owner-approved move
from `model/hosts/go` (fn-210.17). Re-read the current tree when executing extraction.
The target repository puts the contents of
`model/umpire/` at its root, rather than retaining a redundant `model/umpire/` wrapper.

Keep checker behavior, macro authoring, fixtures, protobuf messages and ABI unchanged.
The application exports remain `umpire_buffer_alloc`, `umpire_buffer_release` and
`umpire_call`. The stock guest contains the trait Kettle; the reusable `umpire-runtime`
library is export-free and separate guests select their Model and realization.
Extraction does not make a guest a generic loader for arbitrary Models. Umpire ships no
prebuilt wasm. Temporal commits only its reproducible guest beside its Go host module,
exposes it through `temporal.Guest()` and freshness-checks it in `make model-check`.
Model authors refresh it with `make umpire-guest`; ordinary Temporal Go tests need no Rust.
Generic stock/control guests stay source-built and uncommitted; no Git LFS is used for wasm.

## Source and destination map

Paths in the first column are relative to this Temporal repository. Destination paths
are relative to the new Umpire repository.

| Source | Destination and treatment |
| --- | --- |
| `model/umpire/{dsl,checker,testing,fixtures,runtime,proto}/` | Same directories at repository root; preserve complete fixture groups |
| `model/umpire/hosts/go/` | `hosts/go/`, including its separate Go module and generated protobuf code |
| `model/Cargo.toml`, `model/Cargo.lock` | Root workspace files containing only generic members/dependencies; remove their `umpire/` prefix and retain a separate Temporal consumer workspace |
| `model/{rust-toolchain,rustfmt,clippy,deny}.toml`, `model/.cargo/` | Root tooling configuration, preserving existing policies |
| `model/scripts/` | Generic generation/capability/timing scripts become `scripts/`; Temporal generation and its boundary policy remain in Temporal |
| `model/README.md`, `model/DESIGN.md` | Root documentation with working local links and preserved measurement provenance |
| Rust targets in root `Makefile` | Small standalone root `Makefile`, including generator provisioning |
| Rust, Go and protoc pins in root `mise.toml` | Standalone `mise.toml`; omit Scala/JVM tools |
| `model/.gitignore`, necessary root ignore rules, root `LICENSE` | Standalone ignores and license/attribution; retain existing copyright notices |
| Root `AGENTS.md` | Write Umpire-specific instructions for the extracted layout and gates; do not copy Temporal server instructions wholesale |

Leave `model/temporal/` (Models/goldens, API generation, realization, guest, tests and
the separate extension Go module), `model/archived/`, its ignored caches, `tools/umpire/`, Testpilot, the Temporal
server, and unrelated `.flow/` history in Temporal. Preserve the decisions needed to
understand the exported interface in Umpire documentation without copying the entire
monorepo tracker. Repair comparison-document links to Scala with source-revision
references; the new repository must not need a sibling Scala archive to build.

## Local development contract

Use sibling checkouts under a developer-chosen parent:

```text
<dev-root>/
  temporal/
  umpire/
  go.work
```

Today Cargo workspace `path` dependencies link the in-tree generic and Temporal
crates. Root `go.mod` requires/replaces `umpire.local/model` and `umpire.local/temporal`
with `model/umpire/hosts/go` and `model/temporal/hosts/go`; the extension module also
replaces the generic host locally. The real server/SDK slice already consumes both.
These local references require no publication. The post-extraction setup below preserves
that development property; extraction and publication have not run.

Rust consumers use Cargo `path` dependencies pointing to the exact Umpire crate
directories. For example, from a consumer manifest at `temporal/model/Cargo.toml`,
`umpire-dsl` resolves through `../../umpire/dsl`; compute paths from each actual
manifest, not from the shell's working directory. Share these declarations through
the consumer workspace where appropriate. Keep transitive Umpire crates on the same
local checkout, including the proc-macro crate. Do not introduce git dependencies or
require crates.io publication for the local loop. See [Cargo path dependencies](https://doc.rust-lang.org/cargo/reference/specifying-dependencies.html#specifying-path-dependencies).

Go consumers use the parent `go.work` with `use` entries for `./temporal` and
`./umpire/hosts/go`, plus any actual nested consumer modules. The extracted host's
module identity becomes `github.com/temporalio/umpire/hosts/go`; align its imports,
the schema's `go_package`, and regenerated Go descriptors together. That identity
does not require fetching the module when the workspace resolves it locally. Keep
the parent workspace outside either repository; scripts may create it only with
the owner's consent when one already exists, preserving unrelated entries. Verify
the selected workspace with `go env GOWORK`. See [Go multi-module workspaces](https://go.dev/doc/tutorial/workspaces).

CI reproduces the sibling-checkout layout at explicit source revisions and constructs
its own workspace. An Umpire-only job tests the host module independently; a paired
job tests the actual Temporal consumers. Record both revisions in the job output.
If there are no Rust or Go Temporal consumers yet, prove local resolution with small
disposable consumers instead of inventing server integration. Missing local checkouts
produce an actionable setup error, not a silent fallback to a remote or cached copy.

## Execution

### 1. Freeze a source checkpoint and inspect the destination

Read the current `AGENTS.md`, `model/README.md`, `model/DESIGN.md`, workspace manifests,
Rust Makefile section and scripts. Inspect Flow's current fn-210 state; finish its
remaining prototype work before extraction unless the owner explicitly transfers
that work to the new repository. Record the exact integrated source commit, dirty
paths, host location, tool versions and baseline gate results.

Inspect the destination's branches, contents, license and permissions read-only.
During planning, GitHub returned 404 and `gh` authentication failed; neither proves
that the destination is absent or empty. Obtain working access before importing.
If it contains commits, propose a non-destructive integration against its real
default branch; preserve those commits and stop for an owner decision on conflicts.

Done when: the source revision and tracked-file inventory are fixed, unrelated local
changes are preserved, and the destination's actual state is known.

### 2. Extract history in a disposable clone

Create an isolated clone and perform any history filtering there, never in this
checkout or its shared `.git` directory. Use a reviewed path-and-revision mapping
with [git-filter-repo](https://github.com/newren/git-filter-repo/blob/main/Documentation/git-filter-repo.txt).
Include the Rust lineage through `model2/framework/`, `model2/umpire/` and
`model/umpire/`, plus both host locations and the required supporting files.
Preserve relevant authorship, dates and commit messages; record the old-to-new
commit map. Rewritten commit IDs are expected; identical IDs are not the criterion.

Git path filtering does not discover renames for this task. More importantly,
pre-migration `model/README.md` and other `model/` paths belonged to Scala. Identify
the revisions at which those paths became Rust support files and filter accordingly.
A broad historical `model/` export is incorrect. Produce a complete extracted source
checkpoint first, then make portable-layout adaptations as separate changes.

Done when: the extracted tracked bytes, executable modes and symlink targets match
the approved source map; representative DSL, checker, macro, fixture and host files
retain pre-move history; unrelated Temporal/Scala content is absent from the export.

### 3. Make the workspace self-contained

Apply the destination map. Rebase workspace member/dependency paths, scripts and
Cargo aliases on the new root. Preserve sibling-relative discovery imports in
`dsl/build.rs`, `checker/build.rs` and `tests/build.rs`, the wasm fixture inclusion,
and the schema path used by `wasm/build.rs`.

Extract only the Rust gate recipes, exposing `make check`, `make fmt`, `make update`
and `make time` at the new root. Carry every current check, not just Cargo tests:
capability and timing negative controls, protobuf freshness, formatting,
warnings-denied Clippy/rustdoc, native fixtures/UI/property tests, WASI core builds,
wasm Clippy, the real guest-profile artifacts, and CGO-disabled Go parity. Adapt the timing
helper's parent-Makefile call, checker source path and its own negative controls.

Provision the pinned Go protobuf generator inside the new repository, replacing
`proto-go.sh`'s dependency on Temporal's `.bin/` and Makefile. Preserve protoc and
generator version checks. Keep generated Rust output in Cargo's build output;
regenerate the checked-in Go code rather than editing generated files manually.
Write focused standalone documentation and a CI workflow that runs the same gate.

Done when: a clean Umpire-only checkout builds and tests without Temporal, Scala,
personal absolute paths, pre-existing `.bin/` tools or a shared Cargo target cache.

### 4. Relocate and identify the Go host

Keep its separate Go module at `hosts/go/`. Update the module/import prefix and
`go_package` together, preserving protobuf package names, field numbers, method IDs
and the byte ABI. Account for generated descriptor changes caused by `go_package`.
Repair the real artifact lookup and the trait fixture report/rows paths from the
new host directory. Keep `UMPIRE_WASM` and `CARGO_TARGET_DIR` overrides working.

Done when: the real guest passes export/import checks, ownership/refusal/trap
controls and report/row parity from both the standalone gate and the host directory.
The host imports neither Temporal server packages nor Testpilot.

### 5. Wire and prove the local consumer loop

Inventory actual Temporal callers before changing dependencies. Replace their local
framework source references with the Cargo paths and Go workspace described above.
Retain Temporal-only Models and adapters in Temporal. Forward any retained Temporal
`model-*` commands to the sibling Umpire checkout explicitly, or replace those
commands with documented consumer gates; preserve the owner's expected local loop.

Prove resolution, not just successful compilation: Cargo metadata must name the
local crate manifests, and Go module inspection must show the local host directory.
In disposable verification copies, make one uncommitted Umpire Rust change and one
uncommitted Go host change that cause recognizable consumer compile/test failures.
Observe those failures from Temporal, restore the exact bytes, and observe green
again. No commit, push or registry operation may occur between either pair of runs.

Use the same freshly built local wasm artifact for paired host tests. Rebuilding
after Rust changes remains necessary; consuming a stale downloaded guest would not
prove that the local Rust changes are in use.

Done when: both language consumers demonstrably pick up unpushed local changes,
and documented setup works from a fresh pair of checkouts.

### 6. Cut over only after verification

Present the source map, extracted-history map, target diff, baseline comparison and
standalone/paired test evidence. Obtain the normal authorization for remote writes;
publish through a non-destructive branch/PR path appropriate to the destination.
Preserve both repositories' existing history and branch protections.

Remove the duplicate generic source from Temporal only after the independently
verified destination and local-consumer path are available and that removal is
approved. Update Temporal's current docs, agent instructions and milestone/task
pointers. Keep the Scala archive and caches intact, and retain the recorded source
checkpoint so the extraction can be compared or reversed without rewriting history.

Done when: each repository has one clear source owner, both local and CI consumers
resolve the intended checkout/revision, and no active build command depends on the
removed in-tree implementation. Destination publication and Temporal cleanup are
separate completion claims from preparing this plan.

## Verification evidence

- Before extraction: the actual source `make model-check` and warm `make model-time`.
- After extraction: standalone `make check` and `make time`, plus
  `mise exec -- cargo test --workspace --release --locked`.
- Run the standalone gate from a fresh temporary clone with a fresh target directory;
  dependency downloads may occur during setup, but no Temporal checkout is available.
- Regenerate protobuf output and require a clean tracked tree afterward. Review the
  module-path metadata delta separately from the unchanged wire contract.
- Verify every fixture input, snapshot metadata file, payload and
  expected output. No semantic golden refresh is part of extraction.
- Record `cargo metadata` manifest paths, `go env GOWORK` and `go list -m -json`
  resolution, then the two local-change red/green consumer proofs.
- Preserve the current warm soft targets: 10 seconds for the inner loop and 30 seconds
  for the full gate. Record minor noise and investigate material overruns.
- Compare archived tracked-tree identity and cache inventory before/after Temporal
  cutover. Report baseline failures separately; never relax gates to label migration
  green. Capture exact commands, revisions, results and remaining limitations.

## Pattern Survey

Recorded on 2026-10-18 before fn-210.17 moved the host. Paths and line references
below retain that survey's historical source evidence; current host paths are in
the source and destination map above.

### Analogous Features

- `.flow/tasks/fn-210-rust-prototype-kettle-on-a-trait-engine.10.md:7` — The existing mechanical rename moved `model2/framework/` to `model2/umpire/`, including crate names, workspace paths, Make recipes and snapshot references.
- `.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md:82` — The subsequent relocation moved `model2/` to `model/` while preserving Scala separately under `model/archived/`. These two migrations establish multiple historical source paths.
- `.plans/UMPIRE_MODULES.md:537` — An earlier migration used original-content manifests, independent fixture baselines and explicit consumer dispositions; line 561 separates relocation evidence from cleanup. No reusable repository-history extraction script surfaced in this survey.

### Reusable Utilities

- `Makefile:1424` — `model-check` combines timing and capability controls, protobuf freshness, formatting, linting, documentation, native tests, wasm builds and Go-host parity. Its commands depend on the surrounding Temporal repository layout.
- `model/scripts/proto-go.sh:3` — The generation/freshness helper resolves the workspace and parent repository, uses parent `.bin/protoc-gen-go-v1.36.6`, enforces protoc 29.5, and compares generated Go bytes.
- `model/scripts/time-gate.sh:27` — The timing helper measures the actual parent-repository `model-check` target and the checker’s edit/test loop, with 3-second and 30-second budgets.
- `model/umpire/testing/src/discovery.rs:10` — `discover` validates fixture inputs and expected directories, then sorts fixtures deterministically. `generate` at line 43 builds their runner.
- `model/umpire/checker/build.rs:5` — Build scripts include the shared discovery source by relative path; the wasm build script separately compiles the sibling protocol (`model/umpire/runtime/build.rs:4`).

### Convention Anchors

- Workspace ownership: `model/Cargo.toml:5` defines six members beneath `umpire/`; shared package metadata, dependencies and lint policies live above that subtree. `model/README.md:20` also identifies external formatting, Clippy, toolchain and dependency-policy files.
- Tool provisioning: `mise.toml:3` pins protoc and line 6 pins Rust. `Makefile:306` provisions the Go protobuf generator, while `Makefile:1436` supplies the wasm artifact and disables CGO for host tests.
- Fixture ownership: `model/README.md:77` assigns shared Models to `umpire/fixtures`, checker-specific fixtures to the checker, and spelling fixtures to the DSL.
- Golden integrity: `model/README.md` defines each expected output as snapshot metadata and its payload. Go parity tests read the payloads directly; there are no fixture symlink aliases.
- Host coupling: `model/hosts/go/host_test.go:578` defaults to `../../target`; lines 604 and 621 read `../../umpire/fixtures` goldens. Moving the host changes these relative relationships.
- Go identity: `model/hosts/go/go.mod:1`, `model/umpire/proto/umpire/oracle/v1/oracle.proto:4` and `model/hosts/go/host.go:14` share `umpire.local/oracle`; generated descriptors and imports therefore participate in any module-path change.
- Archive boundary: `model/README.md:7` explicitly freezes Scala and excludes it from the Rust workspace and gate.

### Proposed Alignment

Follow the existing ownership boundaries and relocation-evidence conventions, retaining the shared fixture consumers and complete golden groups. The standalone layout must account for workspace support files, parent-owned tooling, host-relative paths and Go identity; preserving history additionally requires the earlier `model2/framework` and `model2/umpire` paths, whose transitions the current task records document.
