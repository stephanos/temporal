---
satisfies: [R3, R4, R7]
---
# fn-8-umpire-temporal-dynamic-config.3 Render and safely publish the generated Lean catalog

## Description
Turn the canonical projection into the complete generated Lean catalog and satisfy R4. Encapsulate coherent managed-artifact publication behind a small, independently tested common-tool interface.

**Size:** M
**Files:** `cmd/tools/genleandynamicconfig/render.go`, `cmd/tools/genleandynamicconfig/publish.go`, `cmd/tools/genleandynamicconfig/render_test.go`, `cmd/tools/genleandynamicconfig/publish_test.go`, `tools/common/artifactio/set.go`, `tools/common/artifactio/set_test.go`
**Touches:** [cmd/tools/genleandynamicconfig/**, tools/common/artifactio/set.go, tools/common/artifactio/set_test.go]

### Approach
- Render exactly the facade, structural types, and settings/fixture modules from the sorted projection. Follow the existing generated Lean naming, header, escaping, and import conventions without placing handwritten semantics in the output.
- Keep the public artifact-set API deep and small: validate the complete managed path set, realpath/symlink containment, same-filesystem staging, single-writer locking, prior-tree backup, handled-error rollback, and interrupted-publication recovery internally. Do not refactor existing API generation unless required to keep the helper generic.
- Validate the complete candidate artifact set and Lean syntax/elaboration before replacing any retained destination; authored siblings outside the managed facade/directory must be untouchable.
- Make unchanged generation byte-identical. Explicitly sort declarations, constraints, structured fields, fixtures, imports, and diagnostics rather than relying on Go iteration order.
- Test invalid Lean, unexpected/missing paths, symlink and traversal attempts, concurrent invocation, injected publication failures, prior-tree preservation, recovery markers/backups, cleanup, and repeated identical output.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/internal/generate/api/main.go:50-176` — complete three-artifact validation/publication pattern
- `tools/umpire/internal/generate/api/main_test.go:100-232` — managed-path and failure-preservation tests
- `tools/common/artifactio/artifact.go:10-40` — existing atomic single-file primitive
- `tools/umpire/internal/generate/api/render.go:8-21` — generated Lean renderer conventions
- `model/Temporal/API.lean:1-7` — public generated facade shape

**Optional** (reference as needed):
- `model/Temporal/API/Types.lean:1-6` — child-module import/namespace conventions
- `model/lakefile.toml:1-16` — Lean project/toolchain boundary

### Key context
A portable rename cannot atomically replace the facade and child directory together. The contract is validated staging plus recoverable serialized publication: handled failures restore the old coherent set, and a later run repairs detected interruption before reporting success.

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/common/artifactio ./cmd/tools/genleandynamicconfig
cd model && mise exec -- lake build
```

## Acceptance
- [ ] Rendering emits exactly three coherent generated modules with complete declarations/fixtures and byte-identical repeated output.
- [ ] Candidate path-set and Lean validation finish before publication, and authored siblings are never managed or removed.
- [ ] The artifact-set module rejects unsafe/symlinked paths and concurrent writers, stages on the target filesystem, rolls back handled failures, and recovers interrupted publication before success.
- [ ] Tests cover invalid candidates, unexpected paths, nondeterministic inputs, publication failures, interruption recovery, and preservation of the previous coherent tree.
- [ ] Focused Go and candidate Lean verification pass without adding a drift-check target or CI behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
