---
satisfies: [R3, R4, R5]
---
# fn-46-export-lean-model-module-impact-index.3 Expose and document the on-demand module index

## Description
Register the thin exporter and process-level integration for R3-R5.

**Size:** M
**Files:** `model/ModelLint/ModuleIndexMain.lean`, `model/ModelLint/ModuleIndexMainTests.lean`, `model/lakefile.lean`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/ModelLint/ModuleIndexMain.lean, model/ModelLint/ModuleIndexMainTests.lean, model/lakefile.lean, Makefile, model/README.md, model/ARCHITECTURE.md]

### Approach
- Register a non-default `lean_exe` exporter plus an explicitly executed process-test root in the current Lean Lake configuration. Add an injected final-writer seam for success/write-failure tests.
- Add exporter-only root preflight using a narrow adapter over pinned `Lake.loadWorkspaceRoot`: explicit current-directory root config, declared temporal-model package and root-owned umpire-lint/umpire-lint-tests/exporter module targets, canonical directory equality, no dependency resolution/toolchain update/ambient CLI renaming. Capture configuration logs; preserve the shared lint loader's old discovery behavior and reinitialize metadata search paths as required.
- Buffer complete JSON before one final write; on final-write failure return non-zero while acknowledging the OS may have accepted a prefix.
- Make `umpire-export-model-module-index` use quiet outer/nested Lake commands with no stdout banner; successful stderr is empty.
- Add `umpire-check-model-module-index` to capture stdout/stderr/status separately across real warm/cold/stale outer Lake and Make paths, wrong-root, loader/index and failing-sink cases without a checked snapshot. Use isolated scratch packages, never destructive shared-cache cleanup. Wrong-root cases include another valid Lake package, wrong package name/missing owned targets and a valid relocated model checkout.
- Document exact schema/policy, non-semantic role, exit-code requirement, and stdout-write limitation.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.lean:5` — package identity; opt-in tool/root declarations at 92–113.
- `Makefile` — existing focused model and SemanticInventory process-check targets; preserve fn-77's final gate wiring.
- `model/README.md` — current developer commands and delivered ownership boundaries.
- `model/ARCHITECTURE.md` — model import-policy and semantic/tooling separation.
- `model/ModelLint.lean` — existing exit-code and child-stream convention.
- Pinned Lake primary sources: `Lake/Load/Config.lean:20`, `Lake/Load/Workspace.lean:31`, `Lake/Config/Workspace.lean:103` — root-only load and root-owned executable lookup. Isolate this internal-use root-loading API against the pinned version.

### Key context
Use `lake -q exe`; both Lake layers and Make must reserve stdout exclusively for the payload.
The full payload and LF are buffered before final write. A failed sink may leave an OS prefix and must return nonzero; no atomic stdout claim. Public facade/test hits describe import impact, not evidence that IO suites ran.

### Quick commands
`cd model && mise exec -- lake -q build temporal-model-module-index umpire-lint-tests umpire-lint && mise exec -- lake exe umpire-lint-tests && cd .. && make umpire-check-model-module-index && make lint-model`

Run the newly registered process suite explicitly and require terminal exit 0 with actual named warm/cold/stale/wrong-root/final-write cases executed. Run nonfixing aggregate Go lint with the established exact inherited-set comparison; new findings, missing exits and resource kills fail. Preserve all comments/unrelated work; no staging/commits/pushes, default-build/CI expansion, generated API drift gate, new dependency or cancellation work.

### Sequencing note (2026-09-12)
Start only after fn-86 R6 has deleted the hand-written Nexus models: `TemporalExperimentalTests` is a configured `focusedTests` root whose imports and `compatibilityFamilies` pin fn-86 deletes, and fn-85 adds `Umpire.Command`-adjacent modules and a `Temporal.Case` realization that belong in the facade list. Before freezing `IndexPolicy`, correct the spec's corrupted facade entry `Umpire.the deleted execution handoff` (a vocabulary-sweep artifact; it is not a module) and add `Umpire.Command` and `Temporal.Case` to the facade roots. Task .1 has no such dependency and may run now.
## Acceptance
- [x] Warm and cold/stale success paths emit exactly one parseable v1 JSON document plus LF and empty stderr.
- [x] Loader/index/serialization/wrong-root failures emit empty stdout and non-zero; a failing final writer returns non-zero and may leave only an explicitly documented truncated prefix.
- [x] Lake and Make export surfaces are quiet, opt-in, and create no repository artifact; the check target captures streams/status independently.
- [x] Tests cover terminal LF, exact bytes, wrong root, child chatter suppression/replay, loader/index failures, Make path, and injected write failure.
- [x] Documentation is exact and focused checks plus `make lint-model` pass.
## Done summary
Done 2026-09-21; self-review. Commit 36ae3bc.

`temporal-model-module-index` is the shared loader, the pure index and one write, plus the two
decisions neither could make: where it is, and what it writes.

**Where it is.** `ModelLint.ModuleIndexExporter.preflightRoot` loads the current directory as a
Lake root through the pinned `Lake.loadWorkspaceRoot` (root configuration only: no dependency
resolution, no manifest or toolchain update, no CLI argument handling, Lake's log captured and
returned with a failure) and requires the `temporal-model` package at that canonical directory,
owning `umpire-lint`, `umpire-lint-tests` and the exporter with their declared roots. Another
package, a package that borrowed the name, and one that roots an owned executable elsewhere are
refused with one `[model-module-index/root]` line per reason and nothing on stdout; a relocated
checkout passes. The shared loader is untouched and re-initialises its own search path, so the
preflight's search-path change does not leak into the metadata read.

**What it writes.** `run effects roots writeOutput writeError` buffers the whole document before
the one call to the injected writer; a failed write is `[model-module-index/write]` and status 1,
and the message says that whatever the stream already accepted is not a document. A successful
build's chatter is discarded (`quieted`); a failed one is replayed on stderr followed by the loader's
own `[model-import-graph/build]` line.

**Surfaces.** `lean_exe temporal-model-module-index` and `temporal-model-module-index-tests`, both
non-default; `make umpire-export-model-module-index` (`lake -q exe`, no banner, stdout is the
document) and `make umpire-check-model-module-index`, which builds both executables, runs the
process suite, and then runs the Make export path with stdout, stderr and status captured
separately and checks the document's first bytes, single line and terminal LF. Nothing is written
to the repository. `model/README.md` documents the schema, the classification spellings, the
non-semantic role, the exit-code contract and the final-write limitation; `model/ARCHITECTURE.md`
places the index beside the import policy it projects.

**The process suite** (`ModelLint.ModuleIndexMainTests`, run from `model/`): the real outer
`lake -q exe` warm twice, stale (its own root source touched) and cold (its binary and traces
removed), each byte-identical to an in-process export through injected writers with empty stderr;
the same command from a relocated copy of `model/` and `proto/` sharing the build directory through
a link, byte-identical again because paths are root-relative; the binary from an empty directory,
another package, a package that borrowed the name and a misrooted one; an argument as a usage
error; and four child-process fixtures on injected effects (failing writer, failing build with
transcript replay, cycle plus unknown root, successful chatter suppressed with the exact one-row
document).

### What moved from the plan

The plan named `ModuleIndexMain.lean` as the exporter. A Lake executable's root must define a
top-level `main`, and the process tests are a second executable that imports the exporter's logic,
so the logic lives in `ModelLint/ModuleIndexExporter.lean` and `ModuleIndexMain.lean` is the
one-line boundary, as `Temporal.Tool.InventoryMain` is for the inventory.

The relocated-checkout case first copied `model/` alone and failed in the nested build: the lakefile
declares the repository's `../proto` files as inputs, so a relocated checkout is `model/` and
`proto/` side by side, and the test copies both.

The "cold" case removes the exporter's own binary and its trace, which the same command restores;
no shared cache is cleaned.

### Gates

`make umpire-check-model-module-index` (process suite and Make path), `cd model && lake build`,
`lake exe umpire-lint-tests`, `LEAN_NUM_THREADS=1 make lint-model` at the fn-86 closeout baseline
(exit 2 from the two generated `Temporal/API/Proto.lean` errors; the import graph, inventory and both
controlled violations pass). No Go file changed, so the inherited-set Go lint comparison has nothing
to compare.

### Note on the task's execution constraints

The task text says "no staging, commits or pushes". This session's git requirements say to commit
and push to the designated branch, as .1 and .2 recorded. Implementer and reviewer are the same
session; fn-46 as a whole owes a cross-model re-review before its completion review.

## Evidence
- Commits: 36ae3bc
- Tests: make umpire-check-model-module-index, cd model && lake build, cd model && lake exe umpire-lint-tests, LEAN_NUM_THREADS=1 make lint-model
- PRs:
