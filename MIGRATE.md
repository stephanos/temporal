# Gomad repository migration

## Outcome and decisions

Move active Gomad development to the existing private repository
`https://github.com/temporalio/gomad`. Unarchive it, keep it private, make
Gomad the root project, preserve the entire available Git history and durable
Flow records, and retain patched Temporal as a pinned source dependency.

The user asked for this execution plan and will give it to an agent with access.
Writing this plan does not perform the migration or certify either open spec.

Decisions for the executor:

- Preserve commit IDs and merge topology. Add relocation commits; do not filter,
  squash, rebase, or rewrite old history.
- Preserve the destination repository's existing refs and history as well.
- Use a pinned Git submodule at `third_party/temporal` initially. An ordinary
  Go dependency alone does not provide all the source trees and modified tests.
- Keep all Gomad code, including the deferred native backend, in the root project.
- Retain patched Temporal in the public `https://github.com/stephanos/temporal`
  repository on branch `gomad`. Move the standalone Gomad pieces out of its
  current tree; retain required Temporal-side patches, tests and integration
  seams there. Future Temporal dependency updates continue on that branch.
- Point `third_party/temporal` at `https://github.com/stephanos/temporal.git`,
  with an explicit pinned commit from branch `gomad`. Do not create a
  `dependencies/temporal` branch in the private Gomad repository or a new
  Temporal repository. The dependency must not contain the Gomad parent gitlink.
- Publish standalone Gomad code, Flow state, recovery snapshots and continuation
  branches only to private `temporalio/gomad`. Publish only separately reviewed
  Temporal-owned changes and extraction cleanup to the public Temporal branch.
  Check public commit ancestry as well as its final tree for private material.
- Use a new, collision-free branch for the standalone root. Prefer `gomad` if
  unoccupied; otherwise use a fresh `standalone-gomad-<date>` name. Make the
  verified branch the destination default. Do not overwrite an existing branch
  merely to obtain the name `main`.
- Do not start GitHub Actions, profiling, native qualification, or benchmark runs
  as an incidental effect of publishing.

A submodule records a separate repository and a pinned commit. Its tree is
independent of the parent tree. See the [Git submodule documentation](https://git-scm.com/docs/gitsubmodules).

## Scope and exclusions

This plan covers local preservation, repository layout, source dependencies,
module paths, operational paths, developer tooling, identities, private import,
verification and handoff.

It does not change model semantics, wire/evidence formats, qualification budgets,
or the scope of fn-187/fn-189. Neither spec becomes done because files moved.
Native execution stays deferred under fn-128/fn-149. Fn-185 profiling stays
outside this migration. Existing public exposure cannot be undone by a private
destination.

Do not delete the source checkout, dirty worktrees, source branches, retained
evidence or the old public repository. GitHub issues, PRs, releases, permissions,
secrets, branch protections and Actions settings are not transported by Git
history. Inventory and preserve them; changing their ownership is separate from
copying commits.

## Current execution checkpoint

Revalidate these observations before acting. They describe 2026-10-18 state.

Later local continuation implemented the two adapters after their intended RED
failures. The complete ordinary Rust suite now passes201tests with two ignored;
six-mode Clippy passes. Both originals remain. Expected compiled model identity
was rebound after reviewing the measurement ownership move, and all identity
regressions pass in both checkouts. The later coverage exporter interruption has
a narrowly reviewed diagnostic disposition; fresh paired source phases and64
tooling tests are ready. The primary canonical full WASM gate subsequently passed;
a later Cargo hardlink-publication refusal required a reviewed build-handoff fix
and a new full-gate/source freeze. Runtime/workload qualification remains
incomplete; consult the current artifact for live handles. The detailed original
checkpoint below remains historical; read
`.flow/artifacts/rust-hygiene-2026-10-18.md` first and coordinate its source freeze
before migration. Neither original deletion nor final qualification is complete.

| Item | Observed value |
| --- | --- |
| Source checkout | `/Users/stephan/Workspace/skunkworks/gomad/temporal` |
| Source branch/HEAD | `gomad`, `bcb72b2bc7bce461dfcb38344be7c2b54153ad81` |
| Current source remote | `origin = github.com:stephanos/temporal` |
| Integrated upstream Temporal base | `48a1e5ff025e55b3d4d0f44247423d4c279ff636` |
| Preserved upstream merge | `0077477965a5b766b95a667247a9f562c100941b` |
| R3 working lane | `/home/agent/wt/wasm-fn187-functional` |
| R3 base HEAD | `95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc` |
| R3 staged patch SHA256 | `0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299` |
| Toolchains | stock Go1.27.1; Rust1.99.1; matching LLVM23.1.1 |
| Mise location | `tools/gomad/mise.toml`, moving to root `mise.toml` |

The primary and multiple worktrees contain uncommitted work. A clone or bundle
alone will lose that work. The R3 index contains an earlier accepted staged
batch plus later unstaged edits. Preserve that distinction.

Fn-187 tasks .1/.2 are done; .3 is in progress; .4/.5 remain todo. Fn-189 task .1
is in progress. Query Flow again rather than trusting this table.

The latest primary full gate is terminal, exit2. It ran for10m39s, with the WASI
package reporting441.120s. Conformance, projections and architecture passed.
The reported failure is `TestImplementationIdentityBindsCompiledModelDigest`:

- Observed `sha256:9b087b968ee0054b2886b670934e37f0e4546939442d61fde69606ac056c4c99`.
- Expected `sha256:109cabe810787e615569ecac2ca9ca51baa71e430adce05dccd817d44fe53a60`.
- Log: `/home/agent/wt/evidence/fn189/rust199-full-gate-2026-10-18/test-wasm.log`.

Do not blindly replace that expected digest. Review all bound source changes,
establish the intended identity rotation, preserve historical receipts, and
rerun identity and affected semantic gates.

R3 is intentionally not green at this checkpoint. Two new negative-control
tests have no-op adapter stubs in `cluster/transport_input_corpus.rs` and
`storage_transport/vector_corpus.rs`. The new cluster JSON also exceeds the
existing EOF loader's one-file roster. Their RED execution has not run.
Both original inline tests remain. The earlier199-pass Rust result predates
these additions. Do not mistake a clean build, prior receipt or copied task
status for completion.

Current evidence and detailed continuation context:

- `.flow/artifacts/rust-hygiene-2026-10-18.md`.
- `.flow/artifacts/fn187-lightweight-correctness-2026-10-18.md`.
- `.flow/artifacts/rust-crate-reuse-review-2026-10-18.md`.
- R3 `.flow/tmp/fn187-r3/` contains essential migration/conservation tooling.
- `/home/agent/wt/evidence/fn187/r3-current/` contains frozen conservation evidence.
- `/home/agent/wt/evidence/fn189/rust199-full-gate-2026-10-18/` contains the
  failed full gate, source manifests and logs. Its5422-entry qualification
  manifest SHA256 is
  `af86a0566188561cdfacd426343517c31b0070be603b240ab4e459ad1e3c5b4c`.

No runtime or workload qualification followed that failed gate. The old
primary session68363 is terminal. A later queued adapter-test orchestrator was
cancelled before execution. Recheck the process table and agent ownership;
do not use old handles as proof that a process remains alive.

Twenty worktrees remain. `fn184` and `fn184-base` were removed after completed
spec verification; their helper binaries and recovery record are retained under
`/home/agent/wt/evidence/fn184/retired-worktrees-2026-10-18/`.

## Expected destination layout

```text
gomad/
  cmd/
  internal/
    backends/
      native/              # separate Go module, feature-preserved
      wasm/
  testdata/
  third_party/
    temporal/              # stephanos/temporal, pinned gomad-branch commit
  .flow/
  .github/workflows/
  .plans/                  # retained design material
  AGENTS.md
  README.md
  SPEC.md
  ARCHITECTURE.md
  MILESTONES.md
  MIGRATE.md
  mise.toml
  go.mod
  go.sum
  Makefile
```

Keep `go.temporal.io/server` as Temporal's module name. Use
`github.com/temporalio/gomad` for Gomad and
`github.com/temporalio/gomad/internal/backends/native` for its child module.
A GitHub repository rename alone does not update Go import paths.

## Implementation steps

### 1. Establish access and a stable handoff boundary

1. Read this file, root `AGENTS.md`, current Gomad README, WASM README,
   `MILESTONES.md`, fn-187 and fn-189 through Flow.
2. Verify access to the exact destination with an authenticated GitHub account.
   Confirm full repository identity, private visibility, archive state,
   administrator permission, existing default branch and ref inventory.
   Separately verify access to `stephanos/temporal`, its public visibility and
   existing `gomad` branch tip. These repositories keep distinct owners, remotes
   and publication inventories.
3. The originating session could not access GitHub. API returned401, HTTPS Git
   had no credentials, and SSH host verification failed. Supply working
   credentials and verify SSH host keys through trusted GitHub documentation;
   do not disable host verification or print tokens.
4. Coordinate a short source-write freeze with all active agents. Wait for or
   safely stop owned processes. Do not kill unrelated sessions.
5. Create an isolated durable migration checkout under `/home/agent/wt/`.
   Keep the source checkout and its worktrees unchanged. Use no shared clone
   alternates or hardlinks that leave the migration dependent on source storage.
6. Keep work tracking in Flow. Record a migration owner there if one does not
   exist; this document is an execution design, not a second status tracker.

### 2. Preserve history, uncommitted work and external evidence

1. Inventory all refs and worktrees, including detached HEADs, local branches,
   remote-tracking refs, tags, notes, stashes and relevant reflog-only work.
   Assign retention refs to otherwise unreferenced wanted commits in the
   isolated backup repository. Do not silently discard a detached worker tip.
2. Save an exact source-ref-to-object-ID manifest and a verified full Git bundle.
   Validate the bundle and repository object connectivity. A bundle of
   `--all` only preserves reachable objects; preserve wanted reflog tips first.
   If LFS exists, preserve its objects separately.
3. For every dirty worktree, retain its base commit, branch, staged binary diff,
   unstaged binary diff, untracked files, file modes, symlinks, and hashes.
   Preserve ignored files that are required sources, fixture inputs, receipts,
   measurement results or reconstruction tools.
4. Save the index separately. Reconstruct the R3 staged and unstaged layers on
   its exact base in a recovery checkout and verify the recorded staged hash.
   Do not sweep all worktrees into one commit or apply every worker over primary.
5. Snapshot authorized uncommitted primary changes into explicit private recovery
   commits on isolated branches. Keep unfinished R3 and other work on their own
   recovery branches. Preserve original refs and original snapshots as well.
   A relocation of primary alone is insufficient to hand off the unfinished work.
6. Inventory and hash evidence outside Git, particularly `/home/agent/wt/evidence/`
   and the ignored `.flow/tmp/fn187-r3/` tooling. Retain required raw objects,
   profiles, scripts, selectors, fixtures and manifests with their provenance.
   Put portable authored recovery material under a reviewed
   `.flow/artifacts/migration/` location. Store oversized immutable raw evidence
   in approved private artifact storage or a transferred recovery directory,
   with a location/hash manifest.
7. Secret-scan all content intended for upload, including historical Git objects
   and recovery archives. Inventory large blobs across the full history before
   pushing. If GitHub rejects historical objects, stop for a preservation decision;
   do not silently rewrite history or omit refs. GitHub has
   [large-file restrictions](https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github).
8. Keep original evidence bytes unchanged. Add a path/provenance map for relocated
   files. Do not mass-rewrite historical SHAs, absolute paths or frozen receipts
   to pretend they describe the new candidate.

### 3. Preserve Flow state beyond the tracked directory

1. Copy the complete tracked `.flow/` tree, including specs, tasks, memory,
   artifacts, configuration, schemas and supporting files. Preserve relevant
   untracked authored records too. Keep historical non-Gomad records unless
   the owner separately approves pruning.
2. Determine the actual authoritative state location. Here `flowctl.py`
   resolves `FLOW_STATE_DIR`, otherwise `<git-common-dir>/flow-state`.
   Copying `.flow/` alone does not preserve current task statuses and evidence.
3. Export effective spec/task state through Flow and save a private, versioned
   migration snapshot of durable runtime task JSON. Copy/reconstruct durable
   state in the destination's state store using the supported CLI or a
   documented, validated store migration. Inspect the installed CLI's usage
   rather than inventing an import command.
4. Preserve status, completion evidence, actor/claim history and receipt references.
   Do not restore live locks, stale PID handles or active leases as runnable
   ownership. Reconcile claims through the supported lifecycle after the freeze.
5. Capture a before/after effective-state comparison for all specs/tasks.
   Fn-187/fn-189 must retain their actual unfinished states. Preserve native
   deferrals and completed tasks; no mass reset to todo or mass completion.
6. Document bootstrap for a fresh clone. A user must be able to restore effective
   task state from a sanitized durable migration export without access to the
   old checkout's `.git`. Never publish credentials or machine lock files.
7. Inspect any additional state store under `<git-common-dir>/flow-next`.
   It was absent in the originating checkout, but the executor must recheck.

### 4. Retain patched Temporal on stephanos/temporal:gomad

1. Freeze the selected server tree from the preserved source candidate. Keep
   all required Temporal-side Gomad seams, `gomad_wasm_slim` providers/refusals,
   `tests/gomadfunctional`, persistence/testcore changes and upstream merge history.
   Do not replace it with pristine upstream `main`.
2. Prepare the Temporal-only continuation in a separate isolated checkout based
   on the fetched public `stephanos/temporal` branch `gomad`. Keep that existing
   branch's history and advance it only by fast-forward. Transfer selected
   Temporal-side changes from the preserved source through a reviewed path/diff
   inventory. Do not base it on, merge or push private recovery commits merely
   because the final tree will remove Gomad; their ancestors would become public.
   Retain the selected public upstream merge ancestry explicitly where required.
3. Once secured privately, remove the standalone `tools/gomad` copy, project Flow
   records and Gomad-only root docs/tooling/workflows from the public branch's
   current tree. Retain upstream Temporal support files, required server tests,
   `tests/gomadfunctional` and the actual server-side Gomad seams. Split shared
   root files by ownership rather than deleting them wholesale. Existing public
   commits and their old Gomad contents remain historical and are not rewritten.
4. Preserve Temporal's own existing dashboard submodule configuration where
   required. That configuration belongs inside the Temporal dependency, not at
   Gomad's new root. Confirm recursive submodule checkout terminates.
5. Keep the dependency tree free of the new Gomad parent gitlink. A dependency
   pointing to a Gomad-root commit would create a recursive checkout structure.
6. Record `https://github.com/stephanos/temporal.git`, branch `gomad`, as the
   canonical dependency remote and update lane. The private repository owns
   standalone Gomad; the public fork owns patched Temporal. Review the complete
   public commit range, tree and reachable new objects for private Gomad/Flow,
   recovery, credentials and machine-local evidence before publication.
7. Keep preparation local until step9's publication and Actions safeguards pass.
   Step9 publishes and verifies the public Temporal `gomad` commit before the
   private Gomad branch containing its gitlink. The pin must be fetchable from
   that public branch without private recovery refs or local object storage.
8. Future Temporal updates land on public `stephanos/temporal:gomad`. Gomad then
   updates its explicit gitlink after verification. Ordinary clone/submodule
   initialization must use the recorded pin, never silently follow branch HEAD.

### 5. Relocate Gomad and reconcile root-owned integration

1. Branch the standalone candidate from the preserved primary snapshot so the
   complete original history remains its ancestry. Do not start with an orphan
   branch, subtree-filtered history or a fresh `git init`.
2. Move the contents of `tools/gomad/` to repository root in a forward commit.
   Remove the old Temporal-root tree from this branch's current tree only after
   it is preserved on the dependency branch and in the backup. Use reviewed
   explicit inventories; do not run broad destructive commands against a workspace.
3. Preserve all Gomad sources, fixtures, native modules, Rust crate/lockfile,
   version/protocol generators, manifests and schemas. Move `mise.toml` with the
   module. Preserve native hibernation, not deletion.
4. Reconcile root-file collisions intentionally. Gomad's README and Makefile
   become canonical; carry the applicable Temporal-root Gomad targets into the
   standalone Makefile before replacing the old root files.
5. Retain `AGENTS.md`, `.github/copilot-instructions.md`, relevant `.plans/`,
   `MILESTONES.md`, current contracts, migration plan and evidence indexes.
   Update active instructions for the new roots. Keep historical records intact.
6. Carry required root integration tooling, especially `cmd/tools/lintcode`
   behavior and its Gomad-specific tests, into a suitable standalone command
   such as `cmd/lintcode`. Bring its actual dependencies and lint configuration;
   do not replace it with a weaker lint approximation.
7. Preserve copyright, license and notice files for both projects and copied
   tooling. Keep separate provenance where necessary.
8. Add `third_party/temporal` as a real gitlink to the prepared Temporal-only
   commit. For prepublication builds, initialize its local checkout from a
   separate local dependency clone, using only a command-scoped local-file
   transport allowance if Git requires one. Set the tracked `.gitmodules` URL
   to `https://github.com/stephanos/temporal.git` before committing; record
   `branch = gomad` as the update lane while retaining the gitlink's exact pin.
   Never commit a workstation path. Publish the public dependency commit in
   step9 before the private parent branch. Do not
   vendor another full Temporal working tree into the Gomad tree.
9. Retain a reviewed path map from each extracted file to its new owner. Verify
   file contents and modes, accounting only for deliberate path/import updates.
10. Migrate the unfinished R3 lane to a corresponding root-layout continuation
    branch. Replay its exact preserved layers, adapt the same module/path changes,
    and compare the change set semantically against the old R3 snapshot.
    Keep the old-layout recovery ref and staged-patch proof. Do not label its
    no-op adapters complete or integrate unreviewed R3 code into canonical root.

### 6. Repair Go module boundaries and import ownership

Apply this mapping in current code and build configuration:

| Module | New declaration | Local replacements |
| --- | --- | --- |
| Gomad root | `github.com/temporalio/gomad` | `go.temporal.io/server => ./third_party/temporal`; native child => `./internal/backends/native` |
| Native child | `github.com/temporalio/gomad/internal/backends/native` | Gomad root => `../../..`; Temporal => `../../../third_party/temporal` |
| Temporal dependency | `go.temporal.io/server` | Preserve its independent module contract; do not add a reverse Gomad requirement |

1. Rewrite current Gomad imports from `go.temporal.io/server/tools/gomad`
   to `github.com/temporalio/gomad`, including the child module, generators,
   generated output sources, fixtures and test tooling where applicable.
2. Update the generator templates and regenerate outputs. Never hand-edit a
   generated file while leaving its template or provenance stale.
3. Retain exact existing dependency versions. Run module resolution and inspect
   graph changes; do not combine migration with broad dependency upgrades.
4. Check Go `internal` visibility after changing module names. Identify any
   Temporal guest imports of server-internal packages that relied on the old
   module prefix. Stage those guests under the pinned Temporal module when
   required; do not bypass Go's visibility rules.
5. Recheck dependency direction. A source survey found no reverse Gomad Go imports
   outside `tools/gomad` at this checkpoint, but all migrated fixtures/tooling
   still need validation.
6. Preserve default/native isolation, including default refusal of native
   installations and both-source-set static checks. Do not add a broad `go.work`
   to mask incorrect module replacements.
7. Test the missing-submodule case. Core commands that do not require Temporal
   should remain independent where the existing contract permits; commands
   requiring its source must issue a useful initialization instruction.

### 7. Separate Gomad roots from Temporal source/build roots

1. Replace live `repo/tools/gomad` assumptions with explicit Gomad-root ownership.
   Start with `cmd/wasmmeasure` path construction,
   `internal/measurement/measurement.go` reproduction paths,
   `internal/measure/producers.go`, Makefiles and root-discovery helpers.
2. Resolve the Temporal source root independently, defaulting to the initialized
   pinned `third_party/temporal`. Bind its actual commit and dirty/source identity
   in preparation and qualification evidence. Do not infer it as Gomad's parent.
3. Reuse `BuildOverlay(stockRoot, semanticsRoot, outputRoot)` and
   `temporalStageFixture` with their already separate root parameters.
   Keep `internal/model/runtimesource` and native projection inputs owned by Gomad.
4. Update Temporal guest staging in WASI tests, qualification and measurement.
   Reuse `temporalMaterializeModule` and `temporalCompareModuleGraphs` to build
   from a private materialized dependency copy, with inventories and graph
   comparisons preserved. Do not write generated guest files into the pinned
   submodule or silently create a second active `tools/gomad` there.
5. Preserve virtual guest paths such as `TEMPORAL_ROOT=/workspace`. Host checkout
   relocation is not permission to change guest filesystem or protocol semantics.
6. Update compatibility-pack scanners, qualification manifests, source inventories,
   archcheck module classifications, scripts, CLI flags/help and rerun commands.
   Audit path strings passed indirectly through helpers, not only import lines.
7. Search active sources for old module prefixes, `tools/gomad`,
   `tests/gomadfunctional`, `repo/../..`, and original absolute workspace paths.
   Classify each remaining hit as intentional historical data, dependency-owned
   Temporal path, or a bug. Retain an exceptions inventory; do not globally rewrite
   historical evidence.

### 8. Move developer tooling and identity-bound data

1. Keep Rust1.99.1 with rustfmt/Clippy through root `mise.toml` and crate
   `rust-toolchain.toml`. Preserve strict six-feature Clippy and reasoned scoped
   exceptions. Keep Go1.27.1 and existing Rust dependency locks.
2. Move applicable workflows and root lint/Make integration. Update checkout
   roots, `go-version-file`, cache keys, artifact paths and changed-path selectors.
   Configure the public Temporal submodule checkout and private Gomad checkout
   without embedding credentials in files.
3. Update model/helper/runtime/projection identity inputs and admission metadata
   deliberately. Source moves, import changes and compiler changes can rotate
   identities. Capture a reviewed old-to-new identity table.
4. Resolve the current compiled-model digest failure through a source-bound
   identity audit. Keep mutation/missing-source checks and exact digest assertions.
   Do not relax guards to make relocation pass.
5. Preserve every historical receipt under its original compiler, source and
   identity. Rebind current selectors, source hashes, line references and tooling
   only after their current code is frozen.
6. Preserve fn-187 fixtures and assertion mappings. The four pending original-input
   JSON/wire files must survive; they are not proof that the adapters work.
   Keep ignored benchmark roots ignored and resource budgets unchanged.

### 9. Import refs into the private destination safely

1. Back up destination refs and repository metadata before unarchiving or pushing.
   Fetch its existing objects into the isolated backup and preserve them.
2. Unarchive the existing private repository using administrator access. Verify
   it remains private. GitHub makes archived repository content and settings
   read-only until unarchived; see [archival documentation](https://docs.github.com/en/repositories/archiving-a-github-repository/archiving-repositories).
3. Prevent incidental Actions execution before either repository receives a
   migration push. Snapshot each repository's current Actions settings and use
   the authorized administration path to make the publication window safe.
   Verify the settings. If either publication cannot be made safe, stop before
   its push. Do not restore automatic runs until the owner authorizes them.
4. Add the destination as a separate remote in the isolated migration repository.
   Do not repoint the original shared repository's `origin` as a first step.
5. Create an explicit source-ref/object/destination-ref map. Preserve source local
   branches, remote-tracking tips, wanted detached/reflog tips, tags and notes
   under collision-free archival namespaces where names conflict. Preserve
   annotated tag objects, not merely their peeled commits. Do not upload local
   replacement refs or transient merge state as active Git behavior.
6. Push explicit refspecs. Verify remote tips after each bounded batch. Keep every
   destination ref that predated migration. Do not use `git push --mirror`,
   `--prune`, `--force` or branch deletion to make the import fit.
   GitHub's [duplication guide](https://docs.github.com/en/repositories/creating-and-managing-repositories/duplicating-a-repository)
   documents mirroring, but its overwrite-oriented workflow is unsuitable for
   preserving this already existing destination's refs.
7. Retain the complete available source graph, including Temporal ancestors,
   merges and task branches. Entire-history preservation implies this history
   remains large and includes old repository layouts.
8. Recheck the public `stephanos/temporal:gomad` tip against the recorded starting
   tip. If it advanced, stop and reconcile locally; never force it back. Publish
   only the reviewed Temporal-only continuation by normal fast-forward and
   verify that exact commit is fetchable from the public branch. Then publish
   the standalone gitlink branch and all Gomad continuation/recovery branches
   privately to `temporalio/gomad`. Record both repositories' resulting tips.
9. Fetch into a fresh verifier repository and prove the expected source objects
   are reachable through the imported refs. Compare the exact ref/object map,
   check representative merge parents and preserved SHAs, and run Git integrity
   checks. A successful push or matching commit count alone is insufficient.
10. Set the destination default branch to the verified standalone branch without
    overwriting an unrelated branch. Retain the former default branch unchanged.
    Document the canonical clone URL, default branch and submodule initialization.
11. The public Temporal remote receives only its reviewed server continuation
    and extraction cleanup. It receives no standalone Gomad updates, private
    snapshots, Flow/recovery archives or continuation branches. Keep the
    originating checkout available until the owner confirms the private clone
    and handoff work. Do not resume standalone Gomad development in that old
    mixed-layout checkout or push its old local `gomad` branch by mistake.

## Verification

Run checks in the isolated migrated checkout, with only one compiler/runtime
owner and serial resource-sensitive execution. Adapt commands to the new root
without narrowing selectors, timeouts or budgets.

### Preservation and portability

- Source and destination private backups verify; source and destination ref maps
  account for every retained ref and wanted detached tip.
- Every original source commit remains reachable with its original SHA and
  parents. No missing large/LFS object is concealed.
- The destination remains private and unarchived; old destination refs survive.
- A fresh private clone with submodules initialized retrieves the exact Temporal
  pin and terminates recursion. It requires no files from the original workspace.
- The parent gitlink and documented pin agree with the verified public
  `stephanos/temporal:gomad` cutover commit. The public branch has only the
  reviewed Temporal-owned continuation; private recovery ancestors are absent.
- Both the Gomad and Temporal checkouts are clean after verification; guest
  preparation writes only into designated output/materialization directories.
- Effective Flow statuses/evidence match the export, including fn-187/fn-189.
- Recovery branches reproduce primary and R3 changes. R3's original staged patch
  is verifiable in its old-layout recovery checkout, and a documented mapping
  explains its root-layout continuation.
- Relocated evidence paths resolve through the recorded mapping. Historical
  artifacts still identify their actual original inputs.

### Build, generator and boundary checks

From the new Gomad root, after inspecting the migrated target definitions:

```sh
mise exec -- rustc --version
mise exec -- cargo fmt --version
mise exec -- cargo clippy --version
go build ./...
go vet ./...
make validate
make -C internal/backends/wasm fmt-check
make -C internal/backends/wasm lint-rust
make -C internal/backends/wasm test-helper
make test-wasm GOMAD_STOCK_GO=/absolute/path/to/stock/go1.27.1
```

Use the repository's required `test_dep` tag for Go tests. Preserve the actual
default lint driver and baseline policy, including errortype. Run portable
native source/generator/static checks through explicit module selection; do not
revive native runtime qualification. Verify that default build/vet/lint/test
inventory excludes the native child and that native selection still builds.

The architecture tests, generated volume/network drift tests, World protocol
generator checks and both-source-set static checks must remain effective.
Exercise all six Rust lint modes, ordinary Rust tests, supported diagnostic
helper corpus modes and the shared Rust/Go corpus with regeneration disabled.
Verify no unexpected skips or filters. A regeneration check must retain the
existing repeatability/byte-equality contract.

### Execution, identity and qualification

- Run focused implementation identity tests after the reviewed identity update.
  Prove changing or omitting bound sources still changes/refuses identities.
- Exercise CLI help/doctor, a small WASM guest, retained artifact inspection,
  fresh exact replay, installed execution and missing-dependency refusal.
- Build fresh source manifests using the destination's actual absolute paths.
  Do not reuse old source manifests as current qualification inputs.
- Run `qualify-runtime` with all32 seeds plus its aggregate.
- Run `qualify-workloads` with all10 required workload/profile/cell combinations
  plus aggregate, including actual durable SQLite and Temporal inputs.
- Retain all required installed scenario, recovery, inspection, network/replay,
  selected SQLite crash/catalogue and other fn-189 gates from its existing
  ledgers. The two Make targets alone are not the full fn-189 completion audit.
- Keep resource/capacity/RSS budgets and seed/repetition requirements intact.
  Report failures and unsupported checks directly; do not relabel them passes.
- For fn-187, finish both adapter RED/GREEN runs, assertion replacement and
  before/after coverage conservation across required configurations before
  deleting originals. R4 restructuring and R5 counts/wall-time/docs remain.
  Migration does not satisfy those requirements.

### Cutover acceptance

Cutover requires a fresh private clone with working build/test/identity boundaries,
complete history/ref preservation, portable Flow/recovery state, retrievable
evidence and a documented canonical dependency pin. Any remaining spec failures
must remain explicit unfinished work in Flow and the final handoff.

Never claim all fn-187/fn-189 gates pass unless their complete existing acceptance
requirements have current-candidate evidence. Do not substitute migration success
for their completion.

## Failure handling and rollback

- Authentication, privacy or admin mismatch: stop remote changes; retain the
  local plan, backups and migration branches.
- Missing history, secret-scanning block, rejected historical blob or unavailable
  evidence: stop before default-branch cutover. Preserve the failed attempt.
- Build, fixture, identity or coverage regression: keep the candidate on its
  migration branch and repair the cause. Preserve source and budgets.
- Existing destination branch collision: choose another new name. Never force
  update or delete the existing branch.
- If cutover must be reversed, restore the saved default-branch setting to its
  preserved old branch. Retain imported refs and migration evidence for audit.
  Repository deletion and destructive cleanup require separate authorization.
- Keep Actions disabled until owner-approved verification/enablement.
- Do not remove old worktrees or backups merely because the first push worked.

## Deliverables for the executing agent

1. Private standalone Gomad repository with preserved history and destination refs.
2. Patched Temporal retained on public `stephanos/temporal:gomad`, with standalone
   pieces removed from its current tree and an exact remotely retrievable pin.
3. Root-layout sources, modules, tooling, docs and CI definitions.
4. Portable `.flow/` plus effective lifecycle state and a validated restore path.
5. Recovery snapshots and a migrated continuation branch for unfinished R3 work.
6. Source/ref/path/identity/evidence maps with hashes and fresh-clone verification.
7. Current verification results with failures and deferrals preserved.
8. Handoff naming canonical URLs/branches, Temporal pin, Flow statuses,
   remaining fn-187/fn-189 work, Actions state and retained worktree/backup reasons.

## Context files

Read these source-layout anchors before modifying their relocated equivalents:

- `AGENTS.md`, `.github/copilot-instructions.md`, `MILESTONES.md`.
- `tools/gomad/docs/README.md`, `tools/gomad/docs/SPEC.md`, `tools/gomad/docs/ARCHITECTURE.md`.
- `tools/gomad/go.mod` and `internal/backends/native/go.mod`.
- Root `Makefile`, `tools/gomad/Makefile`, WASM `Makefile`.
- `.github/workflows/gomad.yml`, `gomad-smoke.yml`, root lint tooling/config.
- `tools/gomad/internal/backends/wasm/wasi/identity.go`, `identity_test.go`.
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go` guest staging.
- `tools/gomad/internal/backends/wasm/cmd/wasmmeasure/main.go`.
- `tools/gomad/internal/backends/wasm/internal/measure/producers.go`.
- `tools/gomad/internal/backends/wasm/internal/measurement/measurement.go`.
- `tools/gomad/internal/backends/wasm/qualification/` admission/freeze manifests.
- `tools/gomad/internal/model/runtimesource/sources.go`, projection generators.
- `tools/gomad/internal/archcheck/inventory.go`.
- Fn-187/fn-189 specs, task descriptions and the checkpoint artifacts listed above.

The following survey was produced read-only during plan preparation. Paths and
line numbers refer to the source layout and must be rechecked at execution time.

## Pattern Survey

### Analogous Features

- `tools/gomad/docs/README.md:13` — Existing extraction separates the shared/WASM module from the native child module while retaining source paths and backend contracts.
- `cmd/tools/lintcode/rename_policy_test.go:51` — Existing rename tests preserve historical versus newly introduced findings across module moves; migration-sensitive lint behavior already has coverage.
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:2782` — Temporal qualification captures a dependency, materializes a private copy, verifies unchanged source inventory, and compares effective module graphs.
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:2441` — Temporal frontend guests compile `./tests/gomadfunctional`; other guests compile Temporal persistence packages and `./tests`.
- No existing full repository extraction preserving every historical SHA was identified.

### Reusable Utilities

- `tools/gomad/internal/model/runtimesource/sources.go:13` — `Read` loads canonical runtime semantic sources; `Verify` at line 27 checks overlay copies byte-for-byte. These shared sources already have a Gomad owner.
- `tools/gomad/internal/backends/wasm/toolchain/overlay.go:52` — `BuildOverlay(stockRoot, semanticsRoot, outputRoot)` accepts separate explicit roots and returns overlay identity.
- `tools/gomad/internal/backends/wasm/internal/measure/producers.go:799` — `temporalStageFixture` accepts independent source and destination anchors. Existing call sites often pass the same Temporal repository root, although the helper interface supports separated ownership.
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:3154` — `temporalMaterializeModule` and `temporalCompareModuleGraphs` at line 3170 retain and verify private dependency materialization.
- `tools/gomad/internal/backends/native/sourceinventory/inventory.go:34` — `Digest` hashes a bounded regular-file inventory and rejects symbolic links.
- `tools/gomad/internal/archcheck/inventory.go:185` — Architecture inventory detects nested modules and rejects unclassified module boundaries.

### Convention Anchors

- **Module boundaries:** `tools/gomad/go.mod:1` and `tools/gomad/internal/backends/native/go.mod:1` declare separate modules with reciprocal local replacements and distinct relative replacements for Temporal. The current module namespace also determines Go `internal` import accessibility.
- **Canonical source ownership:** `tools/gomad/docs/README.md:13` assigns runtime/model sources to Gomad; `internal/model/network/adapter.go:20` and `internal/model/volume/runtime.go:5` declare generation beside the owning model.
- **Repository versus module roots:** `tools/gomad/internal/backends/wasm/cmd/wasmmeasure/main.go:70`, `:103`, and `:192` construct `repo/tools/gomad` paths. `internal/measurement/measurement.go:96` records that location in reproduction metadata. These are operational assumptions, beyond prose links.
- **Guest staging:** `tools/gomad/internal/backends/wasm/wasi/integration_test.go:2492` stages Gomad-owned fixtures into Temporal `.tmp/wasm-backend-temporal`; line 2508 discovers dependencies from the Temporal build root. Line 2573 sets guest `TEMPORAL_ROOT=/workspace`, a virtual guest namespace distinct from host checkout placement.
- **Root integration ownership:** Root `Makefile:170`, `:190`, `:510`, and `:513` dispatch Gomad builds, qualification, and module-aware lint; `.github/workflows/gomad-smoke.yml:15` and `:106` route paths and select the nested Go version. Root `cmd/tools/lintcode` contains Gomad-specific lint tests and behavior.
- **Current versus historical guidance:** `AGENTS.md:109` identifies root `MILESTONES.md` as delivery order and Gomad `SPEC.md`/`ARCHITECTURE.md` as current contracts. `.flow/artifacts` contains historical references to earlier layouts.
- **Durable lifecycle state outside `.flow`:** `/home/agent/.codex/scripts/flowctl.py:936` resolves `FLOW_STATE_DIR`, otherwise `<git-common-dir>/flow-state`, before a nongit fallback. This checkout has `temporal/.git/flow-state/tasks` and `locks`; copying `.flow` alone excludes that runtime state. The implementation also references `<git-common-dir>/flow-next/pilot-strikes.json`, but that directory is absent here.
- **In-flight work preservation:** `AGENTS.md:123` requires durable worktrees under `/home/agent/wt` and forbids removal of another worker’s active or dirty worktree.

### Proposed Alignment

Follow the existing Gomad ownership and explicit-root interfaces while treating Temporal as the separate guest source/build root. This leaves the migration responsible for repository-root consumers, nested module/import boundaries, identity-bound inputs, root integration tooling, and shared Flow runtime state; the existing rename precedent does not establish preservation of the complete Git graph.
