---
satisfies: [R15]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.10 Supply build, cache and adapter locations from one validated installation description

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

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


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] One validated installation description supplies pinned identity and every owned build, cache and adapter location; ordinary consumers no longer join installation-relative paths themselves.
- [ ] A test pins each location for a fixed root and build key, and the stable adapter replacement location is unchanged, so path-stamped target identities do not move.
- [ ] Tests cover each resolution source (explicit, environment, manifest, executable-relative).
- [ ] Malformed manifests, missing or stale builds, invalid roots and identity mismatches fail closed with the existing repair guidance text.
- [ ] The import edge used by `target` and `deterministicio` is explicit in `architecture_test.go` and no broader than needed.

## Done summary
Blocked:
Blocked: R15 is implemented and reviewed (SHIP). Only native darwin/arm64 and linux/amd64 gates remain, and they belong to task 21.

Done (commits 5825bff79f and 7bece3e999 on gomad-fn109, base 331b75bb6): the new leaf package `toolchain/installation` provides Layout, Build and Description. A Description is the validated installation and supplies the pinned build key plus the build, target-cache, prepared-target and adapter locations. The builder publishes through it. Target preparation, linked review, module-cache queries and the deterministic I/O adapter cache now read locations from it. Validation and repair-guidance text are unchanged. Locations are byte-identical for an absolute root, and the adapter replacement path is pinned for absolute and relative roots. Tests cover explicit, environment, manifest and executable-relative resolution. The architecture edge allows exactly toolchain/version and toolchain/installation for target and deterministicio.

Local evidence (linux/arm64, developmental only; a local, uncommitted shim added linux/arm64 to supported_platforms so a patched toolchain could build; details in task-10/local-evidence.json):
- Characterization tests passed before the edit and after it.
- Quick commands `.toolchain/bin/go test -count=1 -tags test_dep ./toolchain ./target/... ./deterministicio/... ./cmd/gomad/... .`: exit 1 at baseline (245s) and after the change (217s), with the identical set of 15 shim-induced failures.
- Architecture tests: exit 0. `make validate`: exit 0, both with the shim and on the committed tree without it. go vet and gofmt: clean.
- runner/qualification: failures match base sources except two process-timing tests, which pass on rerun.
- `make test-builder`: exit 2, from the two linux/arm64 inventory tests that also fail at baseline.
- Full `make test-host`: exit 2 (661s, 42 ok, 8 failing packages). The failures are shim platform classes or load-related; isolated reruns pass and the preparation failures reproduce on base.
- golangci-lint: not run, because the repository binary is a darwin build.

Remaining native gates: darwin/arm64 and linux/amd64 full `make -C tools/gomad3 test-host` plus the task Quick commands and `make test-builder`, and scoped golangci-lint, on qualified hosts (task 21).
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
