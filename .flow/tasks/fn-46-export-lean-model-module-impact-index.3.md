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
- Add exporter-only root preflight using a narrow adapter over pinned `Lake.loadWorkspaceRoot`: explicit current-directory root config, declared temporal-model package and root-owned modelLint/modelLintTests/exporter module targets, canonical directory equality, no dependency resolution/toolchain update/ambient CLI renaming. Capture configuration logs; preserve the shared lint loader's old discovery behavior and reinitialize metadata search paths as required.
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
`cd model && mise exec -- lake -q build temporal-model-module-index modelLintTests modelLint && mise exec -- lake exe modelLintTests && cd .. && make umpire-check-model-module-index && make lint-model`

Run the newly registered process suite explicitly and require terminal exit 0 with actual named warm/cold/stale/wrong-root/final-write cases executed. Run nonfixing aggregate Go lint with the established exact inherited-set comparison; new findings, missing exits and resource kills fail. Preserve all comments/unrelated work; no staging/commits/pushes, default-build/CI expansion, generated API drift gate, new dependency or cancellation work.
## Acceptance
- [ ] Warm and cold/stale success paths emit exactly one parseable v1 JSON document plus LF and empty stderr.
- [ ] Loader/index/serialization/wrong-root failures emit empty stdout and non-zero; a failing final writer returns non-zero and may leave only an explicitly documented truncated prefix.
- [ ] Lake and Make export surfaces are quiet, opt-in, and create no repository artifact; the check target captures streams/status independently.
- [ ] Tests cover terminal LF, exact bytes, wrong root, child chatter suppression/replay, loader/index failures, Make path, and injected write failure.
- [ ] Documentation is exact and focused checks plus `make lint-model` pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
