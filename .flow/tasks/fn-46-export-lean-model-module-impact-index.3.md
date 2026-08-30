---
satisfies: [R3, R4, R5]
---
# fn-46-export-lean-model-module-impact-index.3 Expose and document the on-demand module index

## Description
Register the thin exporter and process-level integration for R3-R5.

**Size:** M
**Files:** `model/ModelLint/ModuleIndexMain.lean`, `model/ModelLint/ModuleIndexMainTests.lean`, `model/lakefile.toml`, `Makefile`, `model/README.md`, `model/ARCHITECTURE.md`
**Touches:** [model/ModelLint/ModuleIndexMain.lean, model/ModelLint/ModuleIndexMainTests.lean, model/lakefile.toml, Makefile, model/README.md, model/ARCHITECTURE.md]

### Approach
- Add one non-default `[[lean_exe]]` and an injected final-writer seam for success/write-failure tests.
- Buffer complete JSON before one final write; on final-write failure return non-zero while acknowledging the OS may have accepted a prefix.
- Make `umpire-export-model-module-index` use quiet outer/nested Lake commands with no stdout banner; successful stderr is empty.
- Add `umpire-check-model-module-index` to capture stdout/stderr/status separately across warm/cold or stale build, wrong-root, loader/index, and failing-sink cases without a checked snapshot.
- Document exact schema/policy, non-semantic role, exit-code requirement, and stdout-write limitation.

### Investigation targets
**Required** (read before coding):
- `model/lakefile.toml:32-59` — support library and tool executable pattern.
- `Makefile:990-1014,1304-1317` — focused model command and lint style.
- `model/README.md:321-365` — model command/generated-view documentation.
- `model/ARCHITECTURE.md:148-160` — import-policy architecture.
- `model/ModelLint.lean:136-139` — executable exit-code convention.

### Key context
Use `lake -q exe`; both Lake layers and Make must reserve stdout exclusively for the payload.

### Quick commands
`cd model && mise exec -- lake -q build temporal-model-module-index modelLintTests modelLint && mise exec -- lake exe modelLintTests && cd .. && make umpire-check-model-module-index && make lint-model`
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
