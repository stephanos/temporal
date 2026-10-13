# Gomad v3: readable package structure and test layout

## Goal & Context

A reader must understand the final `tools/gomad` tree from its file layout alone. Every non-native package has one clear responsibility and a small exported API over a substantial implementation; test files are organized by behavior, not by the bug, migration or review round that produced them. `backends/native` internals are out of scope except where a misplaced package or test moves under it. The tree remains at `tools/gomad3` until the final atomic rename in R11.

Package doc comments are out of scope (owner decision, 2026-10-12).

Owner decision (2026-10-12): approved grouping the shared packages; refined the same day to place them under `internal/` with `cmd` as the only public entry point (R7).

Evidence: independent structure review on 2026-10-12 (summary below). Counts are production / test files.

- Module root: 0 / 10 — architecture rules, native overlay ownership checks, the external Runner consumer test and CI routing tests have no production owner there.
- Shallow wrappers: `hostfs` (7 lines) and `hostexec` forward to `internal/hostfs` and `internal/hostexec` only to satisfy the WASM import allow-list. Host filesystem helpers are implemented three times (`artifact` rename/link-count files, `captured` hardlink files, `internal/hostfs`).
- Fragmented owners: four `runner/internal/exploration/*` packages for one frontier concern; single-file `qualification/workload`, `internal/gomadtool/validation`, `internal/sourceinventory` (native-only) and `runtimesemantics`.
- Wide APIs: `runner` exports 158 identifiers for about ten entry points, mostly inspection DTOs; `runner/internal/campaign` exports 170 for one importer.
- Overloaded names: "simulation" names the clock/network models, the WASM cluster provider, the scenario frontier and `tools/gomad3sim`; `captured` does not name mounts.
- Test sprawl: `runner` 36 / 53 (one 2,857-line `runner_test.go`, six legacy characterization files, eleven one-test files); `backends/wasm/wasi` 14 / 28 (seven `helper_*` files for one 229-line file; guest-compiling tests mixed with unit tests); `cmd/gomad/internal/cli` 15 / 24; `cmd/gomadtool` 12 / 19.
- Stale references: `internal/archcheck` owner/edge tables, `runner_consumer_test.go` and `ARCHITECTURE.md` still name `runner/internal/execution`, `internal/preparation` and `target/internal/gocommand`.

## Constraint: identity-bound sources

`backends/wasm/wasi`, `backends/wasm/simulation`, `world`, `volume`, `backends/wasm/toolchain`, `simulation/clock`, `simulation/network`, `internal/canonicaljson`, `choice/interrupted.go` and `runner/simulation_broker.go` embed and hash their own sources, including import lines. Renaming those files or any package they import (`hostfs`, `hostexec`, `captured`, `runner`, `choice`, `backends`, `volume`, `world`, `simulation/network`, `internal/iowire`, `runtimesemantics`) rotates model/runtime identities and requires a full WASM requalification. Such moves are confined to R5 and batched with the fn-153/fn-154 requalification.

## Target tree

```text
tools/gomad/               docs, Makefile; no Go files
  cmd/gomad/               the only public entry point (CLI); internal/cli one file per command
  internal/
    runner/                Explore/Resume/Replay/Minimize/Plan/Shard/Merge; inspect/ and internal/{campaign, corpus, exploration, minimizer, local}
    backends/              Provider contract and target types; native/ (+ gomadtool, compatibilitypack, sourceinventory), wasm/ (backend, command, installation, toolchain, qualification, wasi, cluster)
    evidence/              record, artifact (+ capture), qualification
    model/                 choice, world, clock, network, volume, cluster (from tools/gomad3sim), iowire
    host/                  fs and exec (one implementation each)
    archcheck/             structure and purity checks, script policy, architecture tests
  testdata/
```
## Acceptance Criteria

- **R1:** The module root contains no Go files. Architecture tests live with `internal/archcheck`/`internal/puritycheck`; native overlay ownership tests live under `backends/native`; the external Runner consumer test lives in `runner`; CI and gate-routing tests live with `internal/archcheck`. Stale package names are removed from archcheck tables, tests and `ARCHITECTURE.md`.
- **R2:** Test layout follows one convention: `<file>_test.go` per production file plus at most `integration_test.go` (subprocess or guest-compiling), `characterization_test.go` (frozen output) and `architecture_test.go` per package. One-test and per-bug files are merged into the file covering that behavior; legacy characterization that duplicates current behavioral coverage is retired; no assertion, case or coverage is dropped. Target counts: runner ≤ 25, wasi ≤ 14, cli ≤ 15, gomadtool ≤ 12 test files. Make and CI `-run` selectors keep selecting the same tests.
- **R3:** `runner/internal/exploration/*` is one package; inspection moves to `runner/inspect`, cutting `runner`'s exported identifiers by at least half; `runner/internal/campaign` exports only what `runner` uses. `qualification/workload` merges into `qualification`.
- **R4:** Native-only owners move under `backends/native`: `cmd/gomadtool`, `internal/compatibilitypack` (with `authoring`) and `internal/sourceinventory`; `internal/gomadtool/validation` folds into gomadtool. The root `cmd/` holds only the user CLI and root `internal/` holds only owners used outside the native backend (`hostfs`, `hostexec`, `iowire`, the architecture checker and, until fn-153, `canonicaljson`). Make targets, owner tables and documentation follow (owner decision, 2026-10-12).
- **R5 (identity-rotating, batched with fn-153/fn-154):** the `hostfs`/`hostexec` wrappers are deleted with the WASM allow-list pointing at the internal owners; host filesystem helpers have one implementation; `captured` becomes `artifact/capture`; `backends/wasm/simulation` becomes `backends/wasm/cluster`; `runtimesemantics` becomes `internal/runtimesource`; `runner.SimulationClock` is replaced by `simulation/clock`. Full WASM gates pass on the integrated candidate.
- **R7 (identity-rotating, part of the R5 batch):** the module is a tool, not a Go library. Its top level holds only `cmd`, `internal` and `testdata`; `internal/` holds `runner`, `backends`, `evidence`, `model`, `host` and `archcheck`. `record`, `artifact`, `captured` and `qualification` move under `internal/evidence`; `choice`, `world`, `simulation/clock`, `simulation/network`, `volume` and `iowire` under `internal/model`; `target` into `internal/backends`; `hostfs` and `hostexec` become `internal/host`; `puritycheck` merges into `internal/archcheck`. The library-contract checks (external Runner consumer test and fixture module, public-signature and type-alias rules) are deleted. Archcheck owner/edge/pure-set tables, generators, Make targets and documentation follow (owner decision, 2026-10-12, replacing the earlier seven-entry public layout).
- **R8 (identity-rotating, part of the R5 batch):** dead and prototype-era paths are deleted: the Rust helper `diagnostic-profile` feature (`wasmhost/src/diagnostic.rs` and its gated sites and tests), WASM network v1 (`wasi/network.go`, the v1 `Config.Network` field, Rust `NetworkModel`/`Model::net`, and v1-only tests), and the legacy choice-trace v1 decoder (`choice/legacy_v1.go` and its branches), which is refused by name like v2. NetworkV2 and broker networking are unchanged (owner decision, 2026-10-12).
- **R9 (owner-approved 2026-10-12):** `tools/gomad3integration` is removed: its qualification manifests, `testdata/tagged` fixture and manifest checks move under `backends/native/qualification` (their only owner), and the CI and gate-routing tests planned for it in R1 live with `internal/archcheck` instead.
- **R10 (owner-approved 2026-10-12; identity-rotating, part of the R5 batch):** `tools/gomad3sim` moves into the gomad3 module as `internal/model/cluster` after its single `go.temporal.io/server/common/collection` use is replaced by the standard library. Its flat package splits into the public cluster API (cluster, nodes, faults, scenarios, oracles) and internal runtime-domain wire codecs and history records. The gomad3 module then needs the server module only for Temporal guest fixtures.
- **R11 (owner-approved 2026-10-13; final identity rotation):** after R5/R7/R8/R10 settle the final source tree, rename `tools/gomad3` to `tools/gomad` and the nested module/import path from `go.temporal.io/server/tools/gomad3` to `go.temporal.io/server/tools/gomad` in one atomic change. Workspace, build, generator, architecture, CI, fixture, documentation and identity references follow. The `gomad` CLI name and behavior stay unchanged; no `gomad3` compatibility tree or import shim remains.
- **R6:** Every move is a Git rename where possible; behavior, CLI grammar/output, error precedence and exit statuses are unchanged; architecture, lint, `validate`, portable packages and `test-wasm` pass; no identity changes outside R5 and R11.

## Verification

R1–R4 and R6: architecture selection with `GOMAD3_ARCH_ALL_PLATFORMS=1`, `make -C tools/gomad3 validate`, affected lint, portable packages (native-host failures unchanged), `make -C tools/gomad3 test-wasm`, and a check that compiled model and runtime identities are unchanged. R5 and R7: the full WASM gate set from the fn-156 acceptance evidence under the 2026-10-12 gate budgets. R11 repeats those final gates from `tools/gomad`, proves the old directory and module/import path are absent outside intentional historical records, and records the resulting identity rotation.
