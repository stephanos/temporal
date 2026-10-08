---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.48 Preserve patch regeneration cleanup and publication

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Correct the seven unchecked cleanup sites in `toolchain/patch_regenerate.go`, anchored at `101b14f882195422c31f35afc259cb25050b31e3`, under the original R18/R19 lint and preservation gates. The [primary-source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-101b14f882.md) supplies the bounded recommendation.

Task47 remains the dependency. Its integrated reviewed source permits source admission under MILESTONES item3 without completing its blocked acceptance. Task21 consumes this correction through a direct dependency and implements nothing. Fn110 retains patch minimization and native regeneration qualification ownership; task10 retains installation ownership. Preserve every original dependency and requirement.

**Touches:** [tools/gomad3/toolchain/patch_regenerate.go, tools/gomad3/toolchain/patch_cleanup_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-48/**]

### Approach

- Check existing deferred work RemoveAll, candidate VERSION Close, copy input Close, temporary Remove and the three early temporary Close calls at their existing lifetimes. Mirror surrounding conditional cleanup handling, retaining the exact primary error when cleanup succeeds, the sole cleanup error directly and primary-first joins only on actual failure. No new production test seam, shared framework, rollback or changed validation.
- Preserve Close-before-primary-formatting in all three early branches, the copy output Close/error expression, deferred input lifetime and cleanup LIFO order. Suppress missing temporary pathname only after successful Rename; retain pre-publication missing-path failure. Do not change Sync/Close/validation/git-check/Rename/directory-Sync order, error callback provenance, canonical context or descriptor pins.
- Write tests before production edits. Use existing synthetic regeneration fixtures and caller-owned context seams for genuine public RegeneratePatch filesystem cleanup failures. Assert old output stays fixed on cancellation/failure; otherwise completed publication is not rolled back. Test healthy bytes/modes/work/temp absence and primary-first error identities. Explicitly disclose unexecuted Close and post-publication/multiple-cleanup faults rather than substituting synthetic sentinel errors.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

- [ ] All seven admitted sites check genuine cleanup failures at the existing lifetimes, preserving exact primary identity on nil cleanup and primary-first ordering otherwise. Existing comments, source pins, patch bytes and publication semantics remain fixed.
- [ ] At least one public RegeneratePatch filesystem-cleanup regression meaningfully fails against exact BASE before implementation and passes after it, with real OS cause, preserved primary identity and unchanged previous output. Healthy/error controls pass BASE and final. No existing test assertion is weakened.
- [ ] Focused patch/source consumer selection, vet/direct errortype, architecture/purity, check-only validation and changed-line fast lint run on frozen source. Unfiltered scoped/full lint disclose the actual delta from273 findings and any changed residuals; full exit and errortype reachability remain explicit.
- [ ] A fresh independent source/evidence review and separate verified-progress commit feed task21. Original native Darwin full-host/builder/full/default/functional/affected-consumer, matched-first-baseline, bounded10/100, formal and predecessor acceptance remain open wherever unproved. Linux stays deferred and unverified under fn128. Source progress grants neither SHIP nor completed qualification.

## Quick commands

Use pinned stockGo1.27.1 and the existing offline dependency cache, always `-tags test_dep`, in one serialized Go/cache lane. Baseline before source edits.

```bash
go -C tools/gomad3 test -tags test_dep -count=1 ./toolchain -run 'Test(Validate|Materialize|Regenerate|PatchCleanup|PinnedArchive|PinnedContext|Ensure|Extract|SourceArchive|SourceCleanup|CopyWithContext|CopyExactly)'
go -C tools/gomad3 vet -tags test_dep ./toolchain
go -C tools/gomad3 test -tags test_dep -count=1 . -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$'
make -C tools/gomad3 validate
make lint-code-fast GOLANGCI_LINT_BASE_REV=101b14f882195422c31f35afc259cb25050b31e3 GOLANGCI_LINT_FIX=false
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
```

Retain configured scoped toolchain lint and `GOFLAGS=-tags=test_dep errortype -test=true ./toolchain`. The native Darwin host/builder/full and each original affected gate remain required under their owners; portable controls do not qualify them.

## Done summary
Blocked:
Seven patch-regeneration cleanup omissions now report real failures at the original lifetimes while preserving primary identity, Close-before-format ordering and completed publication. Public tests expose exact-BASE ENOENT, ENOTEMPTY and EACCES omissions; all six final cases pass. The post-publication permission case retains literal patch bytes and 0644 mode, returns a direct *os.PathError, and succeeds after restoring owned permissions and retrying. Both permission cases executed without a skip on UID 1000; their UID 0 skip remains explicit.

Fresh independent source-progress review found no Critical/Important issue. Its Minor evidence/type-assertion refinement is resolved. Root final exact-BASE overlay has four meaningful failing subtests and two passing controls, while final source passes seven records without skips. Architecture/purity and check-only validation pass against unchanged production/import/input scope. Focused source/patch consumers pass 28 top-level tests and 73 records with two unchanged missing-pinned-archive skips; CLI patch consumers pass 3 top-level tests and 5 records without skips. Vet and direct errortype pass. Changed-line fast lint passes and reaches errortype. Full configured lint falls from 273 to 266, removing exactly seven owned errcheck sites with no added/changed residual blocks; it still exits 2 before errortype. Scoped toolchain lint falls from 23 to 16 unchanged residuals.

VERSION/input/three early temporary Close faults, simultaneous multiple-cleanup faults and post-publication temporary-removal/directory-durability failures remain unexecuted. Original native Darwin full-host/builder/full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100, formal and predecessor requirements stay open wherever unproved. Linux stays deferred and unverified under fn128. Task21 consumes this correction and implements nothing. Commit verified progress separately; no task completion or formal SHIP is claimed. Evidence belongs under task-48.

stage: impl-review - skipped(policy: required full lint red and original qualification incomplete; independent source-progress review accepts this refined batch)
stage: plan-sync - skipped(config: disabled; task remains blocked)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests:
- PRs:
