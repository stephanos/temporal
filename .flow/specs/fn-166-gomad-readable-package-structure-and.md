# Gomad v3: readable package structure and test layout

## Goal & Context

A reader must understand `tools/gomad3` from its file tree alone. Every non-native package has one clear responsibility and a small exported API over a substantial implementation; test files are organized by behavior, not by the bug, migration or review round that produced them. `backends/native` internals are out of scope except where a misplaced package or test moves under it.

Package doc comments are out of scope (owner decision, 2026-10-12).

Owner decision (2026-10-12): approved the seven-entry top level (`cmd`, `runner`, `backends`, `evidence`, `model`, `internal`, `testdata`) in R7.

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
tools/gomad3/              docs, Makefile; no Go files
  cmd/gomad/               user CLI; internal/cli one file per command
  runner/                  Explore/Resume/Replay/Minimize/Plan/Shard/Merge
    inspect/               Inspect(path) and inspection types
    internal/{campaign, corpus, exploration, minimizer, local}
  backends/                Provider contract and target types (from ./target)
    native/                + cmd/gomadtool, compatibilitypack, sourceinventory
    wasm/                  backend, command, installation, toolchain, qualification, wasi, cluster
  evidence/                record, artifact (+ capture from ./captured), qualification
  model/                   choice, world, clock, network, volume
  internal/                archcheck (+ purity, script policy, architecture tests), canonicaljson,
                           compatibilitypack, iowire, runtimesource, hostfs, hostexec
  testdata/
```
## Acceptance Criteria

- **R1:** The module root contains no Go files. Architecture tests live with `internal/archcheck`/`internal/puritycheck`; native overlay ownership tests live under `backends/native`; the external Runner consumer test lives in `runner`; CI and gate-routing tests live in `tools/gomad3integration`. Stale package names are removed from archcheck tables, tests and `ARCHITECTURE.md`.
- **R2:** Test layout follows one convention: `<file>_test.go` per production file plus at most `integration_test.go` (subprocess or guest-compiling), `characterization_test.go` (frozen output) and `architecture_test.go` per package. One-test and per-bug files are merged into the file covering that behavior; legacy characterization that duplicates current behavioral coverage is retired; no assertion, case or coverage is dropped. Target counts: runner ≤ 25, wasi ≤ 14, cli ≤ 15, gomadtool ≤ 12 test files. Make and CI `-run` selectors keep selecting the same tests.
- **R3:** `runner/internal/exploration/*` is one package; inspection moves to `runner/inspect`, cutting `runner`'s exported identifiers by at least half; `runner/internal/campaign` exports only what `runner` uses. `qualification/workload` merges into `qualification`.
- **R4:** Native-only owners move under `backends/native`: `cmd/gomadtool`, `internal/compatibilitypack` (with `authoring`) and `internal/sourceinventory`; `internal/gomadtool/validation` folds into gomadtool. The root `cmd/` holds only the user CLI and root `internal/` holds only owners used outside the native backend (`hostfs`, `hostexec`, `iowire`, the architecture checker and, until fn-153, `canonicaljson`). Make targets, owner tables and documentation follow (owner decision, 2026-10-12).
- **R5 (identity-rotating, batched with fn-153/fn-154):** the `hostfs`/`hostexec` wrappers are deleted with the WASM allow-list pointing at the internal owners; host filesystem helpers have one implementation; `captured` becomes `artifact/capture`; `backends/wasm/simulation` becomes `backends/wasm/cluster`; `runtimesemantics` becomes `internal/runtimesource`; `runner.SimulationClock` is replaced by `simulation/clock`. Full WASM gates pass on the integrated candidate.
- **R7 (identity-rotating, part of the R5 batch):** the module top level holds at most `cmd`, `runner`, `backends`, `evidence`, `model`, `internal` and `testdata`. `record`, `artifact`, `captured` and `qualification` move under `evidence`; `choice`, `world`, `simulation/clock`, `simulation/network` and `volume` move under `model`; `target` moves into `backends`. Archcheck owner/edge/pure-set tables, generators, Make targets and documentation follow.
- **R6:** Every move is a Git rename where possible; behavior, CLI grammar/output, error precedence and exit statuses are unchanged; architecture, lint, `validate`, portable packages and `test-wasm` pass; no identity changes outside R5.

## Verification

R1–R4 and R6: architecture selection with `GOMAD3_ARCH_ALL_PLATFORMS=1`, `make -C tools/gomad3 validate`, affected lint, portable packages (native-host failures unchanged), `make -C tools/gomad3 test-wasm`, and a check that compiled model and runtime identities are unchanged. R5 and R7: the full WASM gate set from the fn-156 acceptance evidence under the 2026-10-12 gate budgets.
