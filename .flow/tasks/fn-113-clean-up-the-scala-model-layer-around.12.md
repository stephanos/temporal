---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.12 Delete the checker branches only typed fixtures reached

## Description
Delete the checker branches only typed fixtures reached. Completes the "Go checker's typed fixture layer" constraint (recorded under R14) after tasks 10 and 11; no reader or lowering semantics change (tables built from IR never set the typed fields).

**Size:** S
**Files:** tools/umpire/model/internal/checker/{claims.go, search.go, refine.go, lower.go, table.go, keyclaims.go} (production), the nine *_support_test.go files (deleted)
**Touches:** [tools/umpire/model/internal/checker/*.go, tools/umpire/model/internal/checker/*_support_test.go]

### Approach
- The list from fn-115.7's decision 1: `PropertyDecl.holds`/`holds2` (`claims.go:14-22`); the typed arms of `searcher.observe`, `readState`/`readStep`, `Refinement.productStep` (`search.go:339-350,453-470,505-515`); `Refinement.mapValue`, `stepOfFn`, `stepOf`, the typed side of `keyLevel` and the typed cases of `checkKeyRefined` (`refine.go:16-45,289`); the typed branch of `asking.accepts`, `Table.alterer()` and the `alter` field (`lower.go:94-104,168-180`, `table.go:72`); `keyAlterer` stays. Delete them and the nine `*_support_test.go` files (1,342 lines) to `.flow/tmp/trash/fn113-12/`.
- Prove deadness before deleting: `grep` that no production code assigns `holds`, `mapValue`, `stepOfFn`, `alter` or `stateValue` from an IR-built table (only the deleted support did). Simplify what remains (`keyLevel` becomes constant, `checkKeyRefined` loses its typed cases); keep diagnostics and error texts unchanged.
- The reader's public aliases (`tools/umpire/model/types.go`) must still build; run the full `./tools/umpire/...` once (goldens included) since the checker is in every consumer's closure.
- Record any branch that stays and why; count checker production lines before and after for the handover (module map's checker row).

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/claims.go:10-25`
- `tools/umpire/model/internal/checker/search.go:339-350,453-470,505-515`
- `tools/umpire/model/internal/checker/refine.go:1-60,285-295`
- `tools/umpire/model/internal/checker/lower.go:85-200`
- `tools/umpire/model/internal/checker/table.go:60-80`
- `tools/umpire/model/types.go` (reader aliases over checker types)
- `.flow/tmp/fn115-7-summary.md` (decision 1), `.plans/UMPIRE_MODULES.md:76-85`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/...; CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn113-12/full-go.jsonl; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] The nine `*_support_test.go` files are gone and the listed production branches are deleted or each remaining one is recorded with its reason; no diagnostic or error text changed.
- [ ] `./tools/umpire/...` passes in full (goldens included) and `lint-code-fast` passes; checker production line counts before and after are in the summary.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
