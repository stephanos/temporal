---
satisfies: [R15]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.10 Supply build, cache and adapter locations from one validated installation description

## Description
Stage 3, R15 (S3). The layout of a toolchain installation (`bin/go`, `build-key`, `builds/<key>/...`, `adapters`) is re-derived with `filepath.Join` in target, deterministic I/O and the CLI. Give that knowledge one validated value and make ordinary consumers read locations from it.

**Size:** M
**Files:** `tools/gomad3/toolchain/installation.go` (or a sub-package, see below), `target/target.go`, `target/capability.go`, `target/prepared_cache.go`, `target/internal/build/context.go`, `deterministicio/adapter_registry.go`, `cmd/gomad/internal/cli/{cli.go,doctor.go}`, `architecture_test.go` if an import edge changes, tests.
**Touches:** [tools/gomad3/toolchain/*.go, tools/gomad3/toolchain/installation/**, tools/gomad3/target/**, tools/gomad3/deterministicio/adapter_registry.go, tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/architecture_test.go]

### Approach
- Existing owner: `toolchain.ResolveInstallation` / `Installation{ToolchainRoot, Source, ManifestPath, RepairInstruction}` (`toolchain/installation.go:18-110`); identity read by `target.ReadToolchainIdentity` (`target/target.go:350-400`).
- Derivations to replace: `target/target.go:359` (`bin/go`), `:367` (`build-key`), `:375` (`builds/<key>/bin/go`), `:526`, `:658`; `target/capability.go:225`, `:1070` (`adapters`); `target/prepared_cache.go:91` (`builds/<key>/prepared-targets`); `target/internal/build/context.go:104` (`builds/<key>/target-cache`); `deterministicio/adapter_registry.go:231` (`adapters`). The toolchain builder's own publication paths (`toolchain/build.go:125-158,530,557`) are the implementation that defines the layout and should share the same definitions.
- Import constraint: `target` and `deterministicio` may import only `toolchain/version` from the toolchain owner (`architecture_test.go` `ownerMayImport`, the `imported == modulePath+"/toolchain/version"` rule). Place the description where both can reach it and state the chosen edge explicitly in the test; do not widen the rule to all of `toolchain`.
- Path-stamped identity: the adapter replacement directory is recorded in the target binary's module information (`adapter_registry.go:223-228` comment). Every location must be byte-identical before and after; add a test that pins each derived path for a fixed root and build key.
- Validation moves into the value: malformed `build-key`, missing or non-executable `bin/go`, missing/stale `builds/<key>` and their repair guidance (`"set --toolchain-root or GOMAD3_TOOLCHAIN_DIR to a complete Gomad installation"`, `target.go:362-377`) keep their text.
- Resolution sources to cover: explicit root, `GOMAD3_TOOLCHAIN_DIR`, installation manifest, and the executable-relative `.toolchain` candidates (`installation.go:76-77`).

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/installation.go`, `installation_test.go`
- `tools/gomad3/target/target.go:340-400,520-530,650-665`
- `tools/gomad3/target/prepared_cache.go:80-100`, `target/internal/build/context.go:95-110`
- `tools/gomad3/deterministicio/adapter_registry.go:200-275`
- `tools/gomad3/architecture_test.go:440-466`
**Optional:**
- `tools/gomad3/cmd/gomad/internal/cli/doctor.go:60-110`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./toolchain ./target/... ./deterministicio/... ./cmd/gomad/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/...
make test-builder
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
- [ ] One validated installation description supplies pinned identity and every owned build, cache and adapter location; ordinary consumers no longer join installation-relative paths themselves.
- [ ] A test pins each location for a fixed root and build key, and the stable adapter replacement location is unchanged, so path-stamped target identities do not move.
- [ ] Tests cover each resolution source (explicit, environment, manifest, executable-relative).
- [ ] Malformed manifests, missing or stale builds, invalid roots and identity mismatches fail closed with the existing repair guidance text.
- [ ] The import edge used by `target` and `deterministicio` is explicit in `architecture_test.go` and no broader than needed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
