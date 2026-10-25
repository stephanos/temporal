# Retire the Scala-era build and Go tooling; keep model/archived as reading material

## Conversation Evidence

> user (2026-10-25): "are there more simplifications in the whole build/setup/test scripting space we can make?"
> user (2026-10-25): "we want to keep model/archived to learn from it"

## Goal & Context

The Scala Models live in `model/archived` as reference material. The tooling that built them, and the Go tooling that read their IR, still cost build and CI time and confuse readers:

- **Go reader in `tools/umpire`, about 43k lines:** `ir` (8.9k), `interp` (3.1k), `check` (12.0k), `realization` (3.3k), `export` (8.5k) and `internal` (7.4k). Nothing outside these packages imports them. Their README says the IR they read "is no longer checked in, so they keep only their own test fixtures." CI (`.github/workflows/umpire.yml`) still runs `go test ./tools/umpire/...` on every push, and `umpire-install-backends` / `umpire-check-backends` pin and run Quint for `export`.
- **Scala Make targets, about 290 lines of the root `Makefile`:** the `check-framework` wrapper (lines 1–8 and its `endif` at 1268) and the archived Scala block at 541–817 (`MODEL_*`, scala-cli, scalafix, scalapb jars, `lift-model`, `lint-model-*`, `umpire-capture-baseline`, `umpire-check-equivalence`, ...). The block's own comment says they "are unmaintained and are not expected to build." Its `$(shell find … | awk …)` feature discovery runs every time make parses the file.
- **Pin:** `scala-cli` in `mise.toml`. Java stays for TLC.

Keep: `model/archived` (owner), `tools/umpire/temporaldriver` (used by `tools/canary`), `tools/umpire/cmd/umpire` if it imports only kept packages, the live Testpilot targets and the canary.

## Acceptance Criteria

- **R1:** `tools/umpire/{ir,interp,check,realization,export,internal}` are deleted, along with `umpire-install-backends`, `umpire-check-backends`, the Quint pin and helper script, and their CI steps. `go build ./...` and `go vet ./tools/... ./common/testing/testpilot/...` pass. The CI step runs only the packages that remain, and `make umpire-check-live-tests` and `make canary-build` still pass.
- **R2:** The `check-framework` wrapper and the archived Scala block are gone from the root `Makefile`, and so are the `scala-cli` pin and every Make variable or target that only served them. Make still parses with no `MODEL_*` Scala references (`rg -n 'scala-cli|MODEL_JAR|MODEL_ROOT|scalafix|scalapb' Makefile mise.toml` is empty). `umpire-clean-scratch` keeps its `.flow/tmp` scratch half if agents still use it; its scala-cli half goes.
- **R3:** `model/archived/README.md` says the tree is read-only reference material that no longer builds, and names the commit that removed its tooling. `AGENTS.md`, `model/README.md`, `.plans/UMPIRE_MODULES.md` and `tools/umpire/README.md` describe only what remains.
- **R4:** `make model-check` behaves as before. This spec does not touch the Rust model section (lines 1270+, owned by fn-254).

## Boundaries

- Nothing under `model/archived` is deleted or edited, apart from its README.
- No changes to the Rust gate (`model-*` targets after line 1270), `model/scripts` or xtask; fn-254 owns those.
- Adding the Rust `model-check` to CI is fn-254.10's job, not this spec's.

## Decision Context

2026-10-25, owner: keep `model/archived` to learn from it. Its build tooling and the Go IR reader go, since neither runs on anything current. The Quint export depended on the Scala IR; future Quint parity starts from the Rust front end (fn-238), so the export is no foundation for it.

Ordering: independent of fn-254's lanes, but both edit the root `Makefile`. The changed regions (lines 1–8, 541–817 and the `endif` at 1268) don't overlap fn-254's model section. Land serially with the conductor, rebasing onto whichever lands first.

The retained Go production/test dependency closure was checked before implementation and contains none of the six removed trees. Text searches explicitly exempt protected archived material, historical specifications and synthetic forbidden-import fixtures; live imports/build hooks/current documentation must be clean. The parse-time probe uses the existing `canary-build` target (`help` does not exist).

## Quick commands

```bash
mise exec -- go build ./... && mise exec -- go vet ./tools/... ./common/testing/testpilot/...
make -n model-check >/dev/null && make canary-build
```
