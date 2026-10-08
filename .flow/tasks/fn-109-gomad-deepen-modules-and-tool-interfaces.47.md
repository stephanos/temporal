---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.47 Preserve source archive cleanup failures and tar normalization

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Source acquisition must report resource failures while retaining verified archive publication and its existing retry policy. Correct the eleven unchecked cleanup sites and redundant legacy tar selector in `toolchain/source.go` identified by [the source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-4a5f2cf2f8.md), anchored at `4a5f2cf2f84332727b44ddbc14b4844490ea9ef8`.

This corrective R18/R19 owner retains task46 as its dependency. Its integrated, independently reviewed source permits source admission under MILESTONES while its original acceptance stays blocked. Task21 consumes this correction through a direct dependency and implements nothing. Task10 retains installation-description ownership. No original requirement or dependency is removed.

**Touches:** [tools/gomad3/toolchain/source.go, tools/gomad3/toolchain/source_cleanup_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-47/**]

### Approach

- Check all eleven existing root/file/gzip/body/temporary Close and temporary Remove sites at their existing lifetime boundaries. Named results and local deferred cleanup follow surrounding owners. Preserve the exact primary error when cleanup succeeds. Only a genuine cleanup failure joins it, with the primary first and both identities discoverable. Close each handle once; suppress only the expected missing temporary pathname after rename. No callback seam, generic cleanup framework, rollback or changed validation belongs here.
- Body Close remains after temporary cleanup and publication. A newly surfaced close failure may accompany an already published verified archive. EnsureSource still applies its existing explicit/default retry count, cancellation boundary and final wrapper; it does not recheck the cache inside its retry loop. A subsequent call may reuse the verified archive. Keep Sync/Close/Rename/directory-sync order, cache bytes/modes and failure behavior unchanged when cleanup succeeds.
- Remove only redundant `tar.TypeRegA` from the regular-file case after literal raw NUL-type header controls establish pinned Go1.27.1 reader normalization for regular files and trailing-slash directories. Keep archive limits, traversal/unsupported/duplicate rejection, partial extraction and all existing comments.

### Regression and evidence

Before production edits, run existing archive controls and additive healthy/error controls against BASE. Reproduce the missed body-close error through existing SourceSpec.Client using a real already-closed os.File for Close. Verify the actual os.ErrClosed identity and one close per attempt. With Retries=1 the meaningful RED must expose BASE's false success despite publication. Pair failed cleanup with HTTP-status, read, checksum and cancellation primary errors, plus nil-cleanup controls. Exercise default/explicit retries, a later successful attempt, published-cache reuse and literal final error ordering/identity.

Reproduce temporary Remove failure through a test-only response reader that replaces the download's temporary pathname with a nonempty directory before returning a read error. Observe actual ENOTEMPTY, preserve read-error identity first, assert obstruction contents and clean only that test-owned obstruction. Do not expand this into a publication race fix. Add healthy download bytes/mode/temp-absence controls and extraction/cache-read failure controls. Derive legacy-header expectations independently and recompute the raw tar checksum; do not rely on tar.Writer to retain the legacy flag.

Retain exact BASE/final commands, exits, durations, source/log hashes, meaningful RED and one fresh independent source-progress review in a compact handover. Explicitly identify Close/multiple-cleanup paths that cannot be genuinely exercised. Compare actual unfiltered lint blocks with the fresh 285-finding full baseline (227 errcheck, 2 exhaustive, 11 forbidigo, 45 staticcheck), which exits 2 before errortype. Expected removal is eleven errcheck plus one staticcheck; disclose all actual residual/additional changes. No exclusions, pin/guard changes or weakened existing assertions.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

- [ ] The eleven admitted cleanup sites report genuine failures at their original lifetimes, preserving primary errors on nil cleanup and joining primary-first otherwise. Healthy archive bytes/modes, publication, limits and cleanup order stay fixed.
- [ ] Public EnsureSource regressions meaningfully fail against BASE and pass after the correction for actual body-close and removal errors. Explicit/default retry counts, later-success and subsequent verified-cache reuse remain fixed; previously completed publication remains visible.
- [ ] Primary HTTP/read/checksum/cancellation controls retain errors and ordering. Raw legacy regular/directory entries, cache and extraction error controls pass on BASE and final source. Existing tests/comments remain unchanged.
- [ ] Focused archive/build-source consumer selection, vet/errortype, architecture/purity controls, check-only generator validation and make lint-code-fast run on frozen source. Unfiltered scoped and integrated lint disclose the twelve-site delta and every remaining finding. Full-gate exit and errortype reachability stay explicit.
- [ ] Fresh independent source/evidence review and a separate verified-progress commit feed task21. Required native Darwin full-host/builder/full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100, formal and predecessor requirements remain open wherever unproved. Transferred Linux stays deferred and unverified under fn128. Source progress grants neither SHIP nor completed original qualification.

## Quick commands

Use pinned stock Go1.27.1, `-tags test_dep`, existing offline dependencies and one serialized Go/cache lane. Run baseline controls before editing production.

```bash
go -C tools/gomad3 test -tags test_dep -count=1 ./toolchain -run 'Test(Ensure|Extract|SourceArchive|SourceCleanup|CopyWithContext|CopyExactly)'
go -C tools/gomad3 vet -tags test_dep ./toolchain
go -C tools/gomad3 test -tags test_dep -count=1 . -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$'
make -C tools/gomad3 validate
make lint-code-fast GOLANGCI_LINT_BASE_REV=4a5f2cf2f84332727b44ddbc14b4844490ea9ef8 GOLANGCI_LINT_FIX=false
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
```

Retain scoped configured toolchain lint and `GOFLAGS=-tags=test_dep errortype -test=true ./toolchain`. Native Darwin still requires `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, `make -C tools/gomad3 test-builder`, full `make -C tools/gomad3 test` and each original affected gate under its owner. Unsupported-host guard failures and portable selections do not replace those commands.

## Done summary
Blocked:
Eleven source-archive cleanup omissions now retain genuine failures with primary-first error identity, existing resource lifetimes, retries and completed publication. Raw legacy tar controls preserve pinned reader normalization after removing redundant TypeRegA. Real closed-file, closed-pipe, nonempty-directory and corrupt-deflate cases pass after meaningful exact-BASE failures. The callback control detects the initial close-before-format regression, passes BASE and final source with actual EBADF, and explicitly skips on non-Linux or inaccessible procfs.

Fresh independent source-progress review resolved both P2 findings and reports no introduced P0-P3 issues. Root final checks pass eight new top-level regressions with 27 records, six architecture/purity controls, check-only validation and changed-line fast lint. Portable source/patch consumers pass 38 top-level tests and 98 records with two unchanged missing-pinned-archive skips. Vet and direct errortype pass. Full configured lint falls from 285 to 273 findings, removing exactly the eleven cleanup and one TypeRegA findings with no added/changed residual blocks; scoped toolchain lint falls from 35 to 23. Full lint still exits 2 before errortype.

Actual extraction root/file Close, cache-reader Close, five private temporary Close failures and combinations requiring those faults remain unexecuted. Native Darwin full-host/builder/full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100, formal and predecessor requirements remain open wherever unproved. Linux stays deferred and unverified under fn128. Task21 consumes this correction and implements nothing. Commit verified progress separately; no completion or formal SHIP is claimed. Evidence belongs under task-47.

stage: impl-review - skipped(policy: required full lint red and original qualification incomplete; independent source-progress review accepts this corrected batch)
stage: plan-sync - skipped(config: disabled; task remains blocked)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests:
- PRs:
