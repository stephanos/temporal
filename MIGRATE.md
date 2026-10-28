# Extract Umpire into temporalio/umpire

Status: resumed by the owner on 2026-10-27 under [fn-258](.flow/specs/fn-258-complete-the-resumed-standalone-umpire.md),
replacing retired [fn-257](.flow/specs/fn-257-extract-the-standalone-umpire.md).
Destination: <https://github.com/temporalio/umpire>. Migration remains unfinished;
remote publication and source cutover require their recorded access and proof gates.
The renewed instruction supersedes the earlier stop. Preserve original inventories,
source candidates and recovery evidence. Task fn-258.1 is reviewed and closed: the
renewed `ac60daaf47` inventory has 1,189 selected files, and export `a92db061` retains
627 mapped commits with 6,634 independently verified projection-neutral prunings.
Standalone `05473210` and consumer `3f5f6135` are current source candidates, including
the reproducible migrated guest. The standalone required-TLC gate passed, with its
full proof retained; timing targets remain missed. Standalone and consumer source
preparation are reviewed and closed. Paired/release gates, candidate transfer
dispositions, publication and cutover remain open;
in-tree authority remains.
After migration, return to the fn-254 priority in [MILESTONES.md](MILESTONES.md).
The connected GitHub app was rechecked on 2026-10-27 and returned HTTP 422 because
`temporalio` SAML SSO policy blocks this connection. Destination contents and
permissions remain unknown; this does not prevent local preparation and validation.

The local history export is verified at `cf8bec4d6`: 1,178 selected files, 581
retained commits and projection-neutral pruning of 6,568 other available commits.
Two fresh clones reproduced the same commit/map, and a committed
[recovery bundle](.flow/artifacts/fn-257-extract-the-standalone-umpire/task2/export-history.bundle)
restores the independent checkout. See [export evidence](.flow/artifacts/fn-257-extract-the-standalone-umpire/task2/README.md).
That original checkpoint is historical. Current portable source and actual consumer
wiring are prepared and reviewed; final paired/release proof and cutover remain pending.
The supplied history stops at 2026-02-05, earlier than all selected Rust
births; earlier ancestry is unavailable and is not claimed recovered.

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
| `model/umpire/author/` | `author/`, preserving the facade, DSL/macros and suite crates |
| `model/umpire/front/`, `model/umpire/ir/` | `front/`, `ir/`, including nested derives |
| `model/umpire/backends/` | `backends/`, preserving native, TLA+, TLC and Apalache boundaries |
| `model/umpire/protocol/` | `protocol/`, including schemas, runtime and stock/control guest sources |
| `model/umpire/catalog/` | `catalog/`, including syntax, macros and build support |
| `model/umpire/dev/` | `dev/`, preserving testing tools and complete shared fixture groups |
| `model/umpire/reference/kettle/` | `reference/kettle/`, preserving Rust reference/guest and its separate Go module |
| `model/umpire/hosts/go/` | `hosts/go/`, including its separate Go module and generated protobuf code |
| `model/umpire/Cargo.toml`, `model/umpire/Cargo.lock` | Existing independent root workspace and lockfile; retain current member/dependency paths |
| `model/umpire/{rust-toolchain,rustfmt,clippy,deny}.toml`, `model/umpire/.cargo/`, `model/umpire/.config/` | Framework-owned tooling and nextest policies at the extracted root |
| `model/umpire/scripts/`, `model/umpire/dev/xtask/`, `model/umpire/dev/guest-driver/` | Existing local bootstraps, native gate, generator provisioning and guest driver; preserve their ownership |
| `model/umpire/README.md`; selected decisions from `model/{DESIGN,GUEST_BUILDS}.md` | Keep the focused framework README and adapt selected design/guest documentation with working links and measurement provenance |
| Existing native `cargo xtask` commands | Small root `Makefile` aliases for `check`, `fmt`, `update`, `time` and `validate`; delegate the existing gate |
| `model/umpire/mise.toml`, `model/umpire/mise.lock` | Existing framework pins, including Java for TLC and nextest; omit Scala tooling |
| `model/umpire/.gitignore`, necessary root ignore rules, root `LICENSE` | Standalone ignores and license/attribution; retain existing copyright notices |
| Root `AGENTS.md` | Write Umpire-specific instructions for the extracted layout and gates; do not copy Temporal server instructions wholesale |

Leave `model/temporal/` (Models/goldens, API generation, realization, guest, tests and
the separate extension Go module), `model/archived/`, its ignored caches, `tools/umpire/`, Testpilot, the Temporal
server, and unrelated `.flow/` history in Temporal. Preserve the decisions needed to
understand the exported interface in Umpire documentation without copying the entire
monorepo tracker. Repair comparison-document links to Scala with source-revision
references; the new repository must not need a sibling Scala archive to build.

The current `model/Cargo.*`, `model/scripts/`, parent tool configuration and combined
README belong to the Temporal consumer. Do not replace the framework's independent
files with these older extraction inputs. The fixed source checkpoint is
`c88fdc5506`; its 1,177 framework files and supplemental license are inventoried in
[task 1's manifest](.flow/artifacts/fn-257-extract-the-standalone-umpire/task1/source-manifest.json).
Later source candidates require explicit new inventories, not changes to this checkpoint.

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
replaces the generic host locally. The generic module exposes `host`, `runner` and
`gen/umpire/.../v1`; Kettle's Go server/tests live in their own module at
`model/umpire/reference/kettle`, replacing the generic module with `../../hosts/go`.
The real server/SDK slice already consumes both host modules. `umpire.local/model/runner`
is source in `model/umpire/hosts/go/runner`, outside generated `gen/umpire`; transport is
`umpire.local/model/host`. Current crate names are `umpire-runtime` at
`umpire/protocol/runtime` and `umpire-fixtures` at `umpire/dev/fixtures`.
Temporal's reference raw/SDK executors stay in `model/temporal/hosts/go/reference/worker`;
server composition in `tools/umpire/temporaldriver` injects Validate/Follow into the
transport Driver. Workers receive Reset's exported program and have no Umpire callback
connection. Generic Answer remains available to other systems; the Shim was superseded.
[Runner and Run](model/DESIGN.md#runner-and-run) records exact replay/publication contracts,
relocation-inclusive counts, and the still-open R40 cancellation proof; extraction must
preserve that boundary rather than claim it closed.
These local references require no publication. The post-extraction setup below preserves
that development property. The local history export has run; consumer cutover and
publication have not.

Rust consumers use Cargo `path` dependencies pointing to the exact Umpire crate
directories. For example, from a consumer manifest at `temporal/model/Cargo.toml`,
`umpire` resolves through `../../umpire/author/umpire`; direct `umpire-dsl` consumers use
`../../umpire/author/dsl`. Compute paths from each actual
manifest, not from the shell's working directory. Share these declarations through
the consumer workspace where appropriate. Keep transitive Umpire crates on the same
local checkout, including the proc-macro crate. Do not introduce git dependencies or
require crates.io publication for the local loop. See [Cargo path dependencies](https://doc.rust-lang.org/cargo/reference/specifying-dependencies.html#specifying-path-dependencies).

Go consumers use the parent `go.work` with `use` entries for `./temporal` and
`./umpire/hosts/go`, plus any actual nested consumer modules. The extracted host's
module identity becomes `github.com/temporalio/umpire/hosts/go`; align its imports,
the schema's `go_package`, and regenerated Go descriptors together. That identity
does not require fetching the module when the workspace resolves it locally. Keep
Kettle's separate module coherent as `github.com/temporalio/umpire/reference/kettle`,
including its schema options and generated imports. Include both framework modules
and Temporal's host extension in the parent workspace. Keep
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
Rust Makefile section and scripts. fn-210 and the fn-247 native-tooling migration
are closed; preserve their evidence rather than reopening them. Record the exact integrated source commit, dirty
paths, host location, tool versions and baseline gate results.

Fresh baseline measurements are waived by the owner's fn-254 amendment; retain
historical source/mode/workload provenance and report unknowns. Current-input final
checks remain required when validation resumes. Preserve every unintegrated candidate
with provisional source ownership, then assign actual destination refs, tracker
records, evidence and cross-repository prerequisites before source removal.

Inspect the destination's branches, contents, license and permissions read-only.
During planning, GitHub returned 404 and `gh` authentication failed; neither proves
that the destination is absent or empty. Obtain working access before importing.
Read-only local inventory and disposable export may proceed while destination access
is unavailable. Its actual state must be established before any destination import
or publication. If it contains commits, prepare a non-destructive integration against its real
default branch; preserve those commits and stop for an owner decision on conflicts.

Local checkpoint done when: the source revision and tracked-file inventory are fixed
and unrelated local changes are preserved. The destination gate remains open until
its actual state is known; a failed probe does not satisfy it.

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

Rust first appears under `model2/framework/` at `038ac1c6ce`; `a5040fa23b`
renames it to `model2/umpire/`. Admit the `model/` Rust paths only on descendants
of either `6264f11760` or `0fc76f7805`: these sibling archive moves are joined by
`e87ee28f2d`. Before them, `model/umpire/` contains Scala. Include historical
`model2/hosts/go/`, intermediate `model/hosts/go/` and the parent native-tool path
before their respective moves. Preserve mixed historical support under a separate
provenance namespace so it cannot overwrite current framework-owned root files.
Account for mapped deletions and additions in move/merge commits; reject collisions
and verify the full projected tree, not just renamed change records.

Done when: the extracted tracked bytes, executable modes and symlink targets match
the approved source map; representative DSL, checker, macro, fixture and host files
retain pre-move history; unrelated Temporal/Scala content is absent from the export.

### 3. Make the workspace self-contained

Apply the destination map. Rebase workspace member/dependency paths, scripts and
Cargo aliases on the new root. Preserve discovery imports from `dev/testing` in
`author/dsl/build.rs`, `backends/native/build.rs` and `dev/fixtures/build.rs`,
the stock guest's reference-fixture inclusion, and the schema paths used by
`protocol/runtime/build.rs`. Keep catalog build/syntax/macros nested under `catalog`,
and shared backend supervision and typed value admission in `backends/tla`; the
Apalache adapter remains independent of the TLC adapter.

Expose `make check`, `make fmt`, `make update`, `make time` and `make validate`
by delegating the framework's existing native commands. Carry every current check, not just Cargo tests:
capability and timing negative controls, protobuf freshness, formatting,
warnings-denied Clippy/rustdoc, native fixtures/UI/property tests, WASI core builds,
wasm Clippy, the real guest-profile artifacts, and CGO-disabled Go parity. Adapt the timing
helper's parent-Makefile call, checker source path and its own negative controls.

Retain framework-local generator provisioning in `scripts/proto-tools.sh`;
its `.bin/` is already independent of Temporal's Makefile. Preserve protoc and
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

- Before extraction: retain the historical source/mode/workload baseline evidence;
  fresh baseline runs are waived. Source/tree/fixture inventories remain required.
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
- Preserve fn-254's current execution targets: warm activity `model-quick` under
  60 seconds, combined `model-check` under 3 minutes and required-TLC
  `model-validate` under 10 minutes. Retain historical 10/60-second timer warnings
  as historical context; do not revive the obsolete 30-second full-gate target.
  Record final standalone costs separately and report misses honestly.
- Compare archived tracked-tree identity and cache inventory before/after Temporal
  cutover. Report baseline failures separately; never relax gates to label migration
  green. Capture exact commands, revisions, results and remaining limitations.

## Pattern Survey

The original survey was recorded on 2026-10-18 before fn-210.17 moved the host.
The references below name current utilities; historical migration evidence is retained
in the linked Flow records and Git history.

### Analogous Features

- `.flow/tasks/fn-210-rust-prototype-kettle-on-a-trait-engine.10.md:7` — The existing mechanical rename moved `model2/framework/` to `model2/umpire/`, including crate names, workspace paths, Make recipes and snapshot references.
- `.flow/specs/fn-210-rust-prototype-kettle-on-a-trait-engine.md:82` — The subsequent relocation moved `model2/` to `model/` while preserving Scala separately under `model/archived/`. These two migrations establish multiple historical source paths.
- `.plans/UMPIRE_MODULES.md:537` — An earlier migration used original-content manifests, independent fixture baselines and explicit consumer dispositions; line 561 separates relocation evidence from cleanup. No reusable repository-history extraction script surfaced in this survey.

### Reusable Utilities

- `Makefile` — `model-check` combines timing and capability controls, protobuf freshness, formatting, linting, documentation, native tests, wasm builds and Go-host parity. Its commands depend on the surrounding Temporal repository layout.
- `model/scripts/proto-go.sh` — The generation/freshness helper resolves the workspace and parent repository, uses the pinned Go protobuf generator, enforces the protoc version and compares generated Go bytes.
- `model/scripts/time-gate.sh` — The timing helper measures the actual parent-repository `model-check` target and the checker's edit/test loop, reporting the 10-second inner-loop and 60-second gate soft budgets.
- `model/umpire/dev/testing/src/discovery.rs` — Fixture discovery validates inputs and expected directories, sorts fixtures deterministically and generates their runner.
- `model/umpire/backends/native/build.rs` — Build scripts reuse shared discovery; `model/umpire/protocol/runtime/build.rs` compiles the shared schemas.

### Convention Anchors

- Workspace ownership: `model/Cargo.toml` declares generic crates beneath `umpire/` and the Temporal extension beneath `temporal/`, with shared package metadata, dependencies and lint policies. The crate map in `model/README.md` describes each role.
- Tool provisioning: `mise.toml` pins protoc and Rust; the Makefile provisions the Go protobuf generator, builds guests and disables CGO for host parity tests.
- Fixture ownership: shared Models live in `umpire/dev/fixtures`, checker-specific fixtures with the native checker and spelling fixtures with the DSL. Preserve snapshot metadata and payloads separately; Go parity tests read the payloads directly.
- Host coupling: the three Go modules under `model/umpire/hosts/go`, `model/umpire/reference/kettle` and `model/temporal/hosts/go` use local replacement paths. Host tests reference guest artifacts and source-owned goldens; preserve those relationships during extraction.
- Protocol identity: shared schemas live under `model/umpire/protocol/proto`; generated packages live under `hosts/go/gen`. Module/import and `go_package` changes must stay in step without changing protobuf wire identities.
- Archive boundary: `model/archived` is frozen and excluded from the Rust workspace and gate.

### Proposed Alignment

Follow the existing ownership boundaries and relocation-evidence conventions, retaining the shared fixture consumers and complete golden groups. The standalone layout must account for workspace support files, parent-owned tooling, host-relative paths and Go identity; preserving history additionally requires the earlier `model2/framework` and `model2/umpire` paths, whose transitions the current task records document.
