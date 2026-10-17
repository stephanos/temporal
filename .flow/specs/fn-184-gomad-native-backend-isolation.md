# Gomad: native backend isolation

## Conversation Evidence

> user: "Decide on native isolation and file it as a spec: yes to isolate native backend"

## Goal & Context

Make the 2026-10-14 native hibernation real in code: the default Gomad host gates (build, vet, lint, `go test ./...`, archcheck inventory, the WASM and host-tools CI jobs) stop compiling, testing and linting the native backend, while the native backend stays in the tree, buildable and testable through one explicit selection. Nothing is deleted and no native ledger changes. [paraphrase]

Measured on 2026-10-16, the native backend is about 55k production and 46k test Go lines (36.3k/41.2k under `internal/backends/native` plus the 18.5k-line runtime overlay) against 17k/33k for the WASM Go side and 58k/53k shared. Native is hibernated by owner decision, yet every WASM and shared change still pays for it in compile, lint, test and reading time. The coupling is already narrow: no package under `internal/backends/wasm` imports native (`layout_test.go` enforces it), and the only non-native importers are eleven files in `cmd/gomad/internal/cli` (launcher, application, analyze, compare_support, doctor, qualify_set, working_directory and their tests).

## Architecture & Data Models

Two mechanisms are candidates; task .1 decides with a measured prototype.

- **Nested Go module.** `internal/backends/native/go.mod` with module path `go.temporal.io/server/tools/gomad/internal/backends/native`. The directory does not move, so overlay readers and archcheck paths keep working; `./...` from the parent no longer enters it; the parent requires it through a local `replace` and only tagged CLI files import it. Native packages may still import shared `internal/` packages because Go's internal rule is import-path based.
- **File-level build constraint.** A `gomad_native` constraint on every native source and test file, with the CLI seam under the same tag. Keeps one module but touches about 436 files and needs a lint rule to keep the tag present.

Whichever wins, the CLI gets one registration seam: a `native_enabled.go`/`native_disabled.go` pair that registers the native provider, capabilities, private modes (supervisor, coordinator, target bootstrap) and native-only commands. The default build refuses a native installation header with an explicit "native backend not built in" error and never falls back to WASM.

Shared code that reads native sources stays as is: the volume projection generator reads `overlay/src/internal/gomadfs` and `gomadchoicewire`, the network generator reads `gomadio`, `model/cluster` links to `internal/gomadfs` symbols, and archcheck ownership tables and exclusions name native paths. These are file-path and linkname relations, not Go imports, so neither mechanism breaks them. The runtime overlay stays under `internal/backends/native/toolchain/runtime/overlay`.

## API Contracts

- `gomad --installation FILE COMMAND` grammar, defaults, exit statuses, WASM command set and recorded formats are unchanged. A default binary given a native installation header exits with the existing unsupported-configuration status and a message naming the selection that enables native.
- Native-only commands (`analyze`, `compare-support`, `qualify-set`, the native parts of `doctor`, native working-directory validation and the private supervisor/coordinator/bootstrap modes) exist only in a native-enabled build. `doctor` in a default build reports native as "not built in", not as "unavailable host".
- `make -C tools/gomad` default targets (`test-wasm`, `lint-code`, `validate`, `test-harness`, `world-test`) operate on the default set. Native targets (`toolchain`, `test-host`, `test-toolchain`, `overlay-test`, `test-simulation`, `core-qualification`, `upgrade-dossier`, `compatibility-pack-qualification`, `test-builder`, `test-live-capability`, `test-runtime`, `test-upstream`) set the selection themselves and keep their names. `gomadtool` generators used by `validate` keep running; if they live inside the isolated set they are invoked with the selection.
- `internal/archcheck` keeps inventorying native sources for the both-source-set static checks only when the selection is on; the default run checks the default set and must not report native packages as uncovered.

## Edge Cases & Constraints

- Feature preservation: this is freezing, not removal. Every native source, test, fixture, generator and ledger stays; fn-128 and fn-149 deferred owners and their revival triggers are untouched. No native execution pass is recorded or implied.
- Identity: the choice identity generator binds native runtime glue and the patch; model identities bind shared clock sources. Neither mechanism may rotate a retained identity silently. If a path or go.mod change rotates one, record the rotation and requalify the affected WASM gates on the frozen candidate.
- `go.sum` and dependency pins: a nested module needs its own go.mod/go.sum carrying the same pinned versions; `pin-impact` and version-pin validation must treat it as part of the module set.
- CI: edit `gomad.yml` so `wasm-linux` and `host-tools-linux` run on the default set and the native jobs pass the selection. No CI run, PR or push is authorized by this spec.
- Coordination: fn-183 edits `cluster/provider.go` and WASM installation; fn-180/181 edit the CLI replay path. Sequence the CLI seam after their in-flight CLI edits land or in the same batch to avoid file collisions.

## Acceptance Criteria

- **R1:** The default host set excludes every package under `internal/backends/native`: `go build ./...`, `go vet ./...`, `make lint-code`, `make test-wasm`, `make test-harness`, `make world-test` and the archcheck default inventory on a clean checkout compile and test no native package. Errors: a native import reachable from the default set is a build or archcheck failure, never a warning.
- **R2:** One CLI seam registers the native provider, capabilities, private modes and native-only commands only under the selection. Default builds refuse native installations with the existing unsupported-configuration status and a message naming the selection; WASM grammar, defaults, exit statuses, recorded formats and diagnostics are byte-for-byte preserved on the existing CLI fixtures. Errors: no fallback to WASM and no silent command disappearance without a refusal.
- **R3:** The native backend builds and its full existing test set (`make test-host`, `test-toolchain`, `overlay-test`, `test-simulation`, generators, `validate`) runs unchanged under the selection, with every source, fixture, ledger and deferred owner retained. Errors: a native test, generator or ledger that no longer runs under the selection blocks completion; native execution remains deferred and no pass is claimed.
- **R4:** Shared consumers of native sources (projection generators, linkname glue, archcheck tables, version-pin and pin-impact inventories) still validate on both the default set and the selected set, and retained choice, model, helper and runtime identities are unchanged or their rotation is recorded with the requalified WASM gates. Errors: a stale generated projection or unexplained identity change is a finding.
- **R5:** A retained before/after measurement on the same host records default-set package count, production and test lines, and wall time for build, vet, lint and the default test targets, plus the same for the selected set; README, ARCHITECTURE and MILESTONES describe the selection, the refusal and the preserved native gates. Errors: missing numbers or a documentation claim of native qualification is a finding. **Transferred (owner decision 2026-10-17):** the wall times for build, vet, lint and the default test targets, for the default and selected sets, move to fn-185 R6. Package and line counts are recorded at close in `.flow/artifacts/fn184-closeout-2026-10-17.md`.

## Boundaries

- No deletion, relocation or rewrite of native code, overlay, fixtures or ledgers; no change to fn-128/fn-149 scope.
- No change to WASM semantics, evidence formats, model identities or the Rust helper.
- No CI run, PR or push; workflow files are edited as source only.
- No broad package reorganization beyond the CLI seam and the chosen isolation mechanism.

## Decision Context

Owner decision (2026-10-16): isolate the native backend. The 2026-10-14 hibernation deferred native acceptance but left native code in every gate; isolation converts that decision into lower maintenance cost per WASM task. The feature-preservation constraint is honored because the code stays and remains buildable and testable through the selection.

## Early proof point

Task .1 prototypes the chosen mechanism on a throwaway branch and shows `go build ./...` plus `go vet ./...` of the default set succeeding with zero native packages in `go list -deps`, and the native set building under the selection. If neither mechanism achieves that without touching shared identities, stop and report before the seam work.

## Quick commands

```sh
cd tools/gomad && go list -deps ./cmd/gomad | grep internal/backends/native   # expect empty on the default set
cd tools/gomad && go build ./... && go vet ./... && make test-harness world-test
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/... -run 'Layout|Architecture'
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Default host set compiles and tests no native package; a reachable native import is a build or archcheck failure. | fn-184-gomad-native-backend-isolation.1, fn-184-gomad-native-backend-isolation.3 | — |
| R2 | One CLI seam registers native only under the selection; default builds refuse native installations with the existing status; WASM grammar and outputs preserved. | fn-184-gomad-native-backend-isolation.2 | — |
| R3 | Native builds and its full existing test set runs unchanged under the selection; everything retained; native execution stays deferred. | fn-184-gomad-native-backend-isolation.3 | — |
| R4 | Shared consumers of native sources validate on both sets; identities unchanged or rotation recorded with requalified WASM gates. | fn-184-gomad-native-backend-isolation.3 | — |
| R5 | Before/after measurement on the same host; README, ARCHITECTURE and MILESTONES updated without claiming native qualification. | fn-184-gomad-native-backend-isolation.1, fn-184-gomad-native-backend-isolation.4 | Wall times transferred to fn-185 R6 by the owner decision of 2026-10-17; package and line counts recorded at close. |
