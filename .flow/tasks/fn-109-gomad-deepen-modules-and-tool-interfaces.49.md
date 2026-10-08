---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.49 Preserve adapter cache cleanup and stable publication

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Correct the unchecked scratch RemoveAll in deterministicio prepareCachedAdapter at `8486dcb98d2b15e1f985d5bb1e79ae7da81a0d5a`. Own the complete preparation, publication, inventory verification and reuse lifecycle under R18/R19. The [primary-source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-8486dcb98d.md) grounds this correction. One lint finding represents a transaction that currently reports success and writes build modfiles after scratch cleanup fails.

Task48 remains the dependency. MILESTONES item3 permits source admission from its integrated reviewed candidate while its acceptance stays blocked. Task21 consumes this correction through a direct dependency and implements nothing. Existing adapter semantic, installation and qualification owners retain their contracts; preserve all predecessor requirements and fn128 deferral.

**Touches:** [tools/gomad3/deterministicio/adapter_registry.go, tools/gomad3/deterministicio/adapter_cache_cleanup_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-49/**]

### Approach

- Check deferred scratch RemoveAll at its existing lifetime with the surrounding conditional pattern. Keep exact primary error identity when cleanup succeeds, a sole cleanup error directly, and primary-first errors.Join only on actual failure. Preserve the prepared value inside this private owner while letting registry.prepare withhold target/evidence/modfile output on an infrastructure failure. Do not classify cleanup failure as invalid target configuration.
- Preserve callback lifetime, replacement Rename and inventory verification, stable cache paths and relocation, fixed identity bytes, comments and public signatures. Leave completed publication intact; a subsequent healthy preparation must verify and reuse it. Do not add production hooks, timing races, new dependencies, rollback or synthetic cleanup errors.
- Write tests first through the existing adapterImplementation.prepare callback and real filesystem. Cover preparation-primary plus permission cleanup error, successful publication plus cleanup error, reuse plus cleanup error, healthy publication/reuse and malformed/corrupt-cache controls. Test exact identity/ordering, withheld modfile/evidence, fixed cache bytes/modes/path, source go.mod/sum preservation, permission restoration and healthy retry. UID0 permission skips must be explicit; demonstrate execution on current UID1000.
- Retain real BASE failures and passing controls before the production change. Use hand-derived literal fixtures/digests, not expectations derived by the owner under test. Exercise the registry consumer as well as direct owner error identity where useful. Public pinned adapter/native qualification remains unproved by this private-owner fixture.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

- [ ] The existing cleanup site handles actual failures at its original lifetime, with unchanged primary identity on nil cleanup, direct sole cleanup error and primary-first actual joins. Stable publication, inventory checks, bytes, comments, signatures and classification remain fixed.
- [ ] Genuine permission-fault regressions fail exact BASE before implementation and pass final source, with concrete OS causes and no skips on UID1000. Healthy and malformed/corrupt controls pass BASE/final. Published cache remains intact and healthy retry reuses its fixed location; failed preparation publishes no build modfile or adapter evidence. Existing assertions remain unchanged.
- [ ] Focused registry/adapter consumers, vet/direct errortype, architecture/purity, check-only validation and changed-line fast lint run against frozen source. Unfiltered scoped/full lint disclose the actual delta from266 and any changed residuals; full exit and errortype reachability remain explicit.
- [ ] Fresh independent source/evidence review and a separate verified-progress commit feed task21. Original native Darwin full-host/adapter/full/default/functional/affected-consumer, matched-first-baseline, bounded10/100, formal and predecessor requirements remain open wherever unproved. Linux stays deferred and unverified under fn128. Source progress grants neither SHIP nor completed qualification.

## Quick commands

Use pinned stockGo1.27.1 and the offline dependency cache, always `-tags test_dep`, in one serialized Go/cache lane. Baseline before production edits.

```bash
go -C tools/gomad3 test -tags test_dep -count=1 ./deterministicio -run 'Test(Prepare|Adapter|NewAdapter|DetectModule|ProfileVerifies)'
go -C tools/gomad3 vet -tags test_dep ./deterministicio
go -C tools/gomad3 test -tags test_dep -count=1 . -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$'
make -C tools/gomad3 validate
make lint-code-fast GOLANGCI_LINT_BASE_REV=8486dcb98d2b15e1f985d5bb1e79ae7da81a0d5a GOLANGCI_LINT_FIX=false
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
```

Retain configured unfiltered deterministicio lint and `GOFLAGS=-tags=test_dep errortype -test=true ./deterministicio`. Each original native/full/affected gate remains required under its owner; portable controls do not qualify it.

## Done summary
Blocked:
The cached-adapter owner now reports real scratch cleanup failures at its existing deferred lifetime. It retains exact primary identity after successful cleanup, returns a sole *os.PathError directly, joins real additional failures primary-first, and keeps completed publication intact. Registry preparation returns a zero target and no evidence, withholding gomad.mod/sum on cleanup failure; restoring owned permissions permits verified reuse at the literal stable cache path. Existing comments, public signatures, validation, source pins, relocation and dependency files remain unchanged.

Four fault cases failed exact BASE before the production edit. Frozen final tests expose five failing BASE fault subcases and eight passing controls; root final execution passes all 13 subcases and 17 test records without skips on UID 1000. Supplemental portable consumers pass 13 top-level tests. The exact admitted wide Quick remains exit 1 with the same 12 inherited missing-patched-toolchain/unsupported-host failures. Its command and acceptance remain unchanged. Vet and direct errortype pass; architecture/purity, check-only validation and changed-line fast lint pass. Fast lint reaches errortype. Full configured lint falls from 266 to 265, removing the single owned omission with zero added/changed residual blocks; it exits 2 before errortype. Scoped lint falls from 2 to 1 unchanged forbidigo.

Fresh independent source/evidence review accepts this frozen source progress with no actionable Critical, Important or Minor finding. Its BASE/final executable verification reproduces five fault failures/eight controls and all 13 final subcases without skips. The source-review.md and review-observations.json receipts retain exact commands, hashes and unproved gates. Root retains sole lifecycle and commit ownership. Task21 keeps all 26 prior dependencies plus task49 and implements nothing; its original Acceptance-to-EOF bytes remain fixed.

Additional uncached/no-selection controls proposed by the audit were not newly executed. Private callback fixtures establish no public pinned adapter qualification. Original native Darwin/full-host/full/default/integration/functional/affected-consumer, matched-first-baseline, bounded 10/100, formal and predecessor requirements stay required and open wherever unproved. Linux stays deferred and unverified under fn128. Retain this verified source progress separately without task completion or SHIP. Evidence belongs under task-49.

stage: impl-review - skipped(policy: original wide Quick and full lint red; original qualification remains incomplete)
stage: plan-sync - skipped(config: disabled; task remains blocked)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests:
- PRs:
