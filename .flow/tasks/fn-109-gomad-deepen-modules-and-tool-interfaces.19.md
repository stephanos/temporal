---
satisfies: [R8]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.19 Enforce package coverage, host-effect and public-signature rules in the architecture checks (fulfils fn-105.4 D4)

## Description
Stage 6, R8 (F7). The architecture test lists a fixed set of roots and ignores standard-library and external imports, so a new top-level package or a host effect in a pure module goes unnoticed. This task is the single implementation owner of fn-105.4 (D4, origin brief `.flow/tasks/fn-102-gomad-architecture-consolidate.5.md`); close fn-105.4 by reference afterwards. It runs after the refactors so the rules describe the final owners.

**Size:** M
**Files:** `tools/gomad3/architecture_test.go`, negative fixtures under `tools/gomad3/testdata/architecture/` (new), possibly a small private checker package if the test file would otherwise hold the whole checker.
**Touches:** [tools/gomad3/architecture_test.go, tools/gomad3/testdata/**, tools/gomad3/internal/gomadtool/**]

### Approach
- Gaps: `listHostPackages` (`architecture_test.go:371-396`) passes thirteen explicit patterns to `go list`; `TestPackageArchitecture` skips every import outside the module (`:35-37`); several checks only assert file presence or absence (`TestCleanupRemovesSupersededFiles` `:58`, and similar).
- Discovery: enumerate every host package of the module (`./...`) for both qualified source sets (`GOOS=darwin GOARCH=arm64` and `GOOS=linux GOARCH=amd64`), with an explicit, justified exclusion list for the runtime overlay (`toolchain/runtime/overlay`), `testdata` fixtures and the conformance fixture module. A package with no owner fails; an exclusion that matches nothing fails too.
- Host-effect rules are targeted, not a standard-library allowlist: name the pure modules and what they may not reach (`world`: no `os`, `net`, `os/exec`, `time.Now`, goroutine start; `runner/internal/campaign/controller.go` and the exploration frontiers: no host effects; the capability evaluator from task 11; the lifecycle owner from task 16; `record`). Mixed packages need file-level rules (`runner/internal/campaign` holds the pure controller beside effectful journals). The rule must catch an effect imported through the standard library or a dependency, on either platform's file set.
- Public-signature rule: no exported function, method, field or interface of a public package may mention a type from an `internal/` package the consumer cannot import (the R5 defect). Extend the existing export walker (`packageExports` `:332`, `TestPublicPackagesDoNotExportTypeAliases` `:80`).
- Negative fixtures prove each rule rejects: an ownerless new root, a forbidden owner edge, a forbidden host effect in a pure file, and an inaccessible type in a public signature. Run the checker against the fixture and assert the specific failure; a fixture that merely exists proves nothing.
- Keep current ownership and edge checks (`packageOwner` `:398`, `ownerMayImport` `:440`, `moduleMayImport` `:468`, `TestExactModuleEdges` `:263`) including the owners added by tasks 7, 10 and 11. Valid platform-specific files (`*_unix.go`, `*_other.go`, `*_darwin.go`) stay accepted.

### Investigation targets
**Required:**
- `tools/gomad3/architecture_test.go` (whole file)
- `.flow/tasks/fn-102-gomad-architecture-consolidate.5.md`
- `tools/gomad3/ARCHITECTURE.md` sections "System boundary" and "World" (documented purity)
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md` (public surface after R5/R13)
**Optional:**
- `tools/gomad3/internal/gomadtool/validation/` (existing Go-owned content checks)

### Quick commands
```bash
cd tools/gomad3
GOWORK=off go test -count=1 -tags test_dep .
GOWORK=off GOOS=linux GOARCH=amd64 go vet -tags test_dep ./...
GOWORK=off go vet -tags test_dep ./...
make validate
cd ../.. && flowctl show fn-105-gomad-follow-ups-deferred-scope.4
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Package discovery covers every host package on both qualified source sets with explicit, justified exclusions; an ownerless package or a stale exclusion fails.
- [ ] Targeted host-effect rules cover the named pure modules, including file-level rules for mixed packages, and catch effects reached through the standard library or a dependency.
- [ ] A public-signature rule rejects exported declarations that mention inaccessible internal types.
- [ ] Negative fixtures demonstrate rejection of an ownerless new root, a forbidden import edge, a forbidden host effect and an inaccessible public type, each asserting the specific failure.
- [ ] Existing ownership checks still pass; valid platform files and the explicit overlay/fixture exclusions remain accepted; no check relies on filename presence alone.
- [ ] fn-105.4 is closed by reference to this task (one owner).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
