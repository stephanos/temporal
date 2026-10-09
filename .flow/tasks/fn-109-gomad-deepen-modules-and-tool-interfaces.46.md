---
satisfies: [R18, R19]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.46 Preserve maintainer-command success reporting and operation precedence

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Make gomadtool return the documented status 3 when successful operation reports cannot reach stdout. Correct the 17 unchecked report sites retained in [the source audit](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/next-gate-source-audit-a6a28720af.md), anchored at `a6a28720af58fe65cb6fd3bc618ef8102765b50b`. Earlier operational failures keep their original statuses. Preserve healthy report bytes and completed publications.

This is one corrective fn-109 R18/R19 task. Retain fn-109.45 as the predecessor dependency and record source admission from its integrated, reviewed correction under the milestone delivery rule, while its original acceptance remains blocked. Task21 consumes this task through a direct acceptance dependency and continues to implement nothing; this task does not depend on task21 completion. Fn-113.3 retains authoring, approval and publication ownership; task23 retains lint-routing ownership. Previous corrections' native, full, formal and first-baseline obligations remain open.

**Touches:** [tools/gomad3/cmd/gomadtool/main.go, tools/gomad3/cmd/gomadtool/compatibility_pack.go, tools/gomad3/cmd/gomadtool/boundary.go, tools/gomad3/cmd/gomadtool/upgrade.go, tools/gomad3/cmd/gomadtool/maintainer_output*_test.go, tools/gomad3/cmd/gomadtool/testdata/maintainer-output/**, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-46/**]

The production surface is exactly these 17 stdout sites. Line numbers refer to the audited source.

| File | Sites | Existing report |
| --- | --- | --- |
| main.go | 178, 193, 206, 237, 318, 360 | Patch validation, script validation, materialization, regeneration, build key, conformance success |
| main.go | 406, 408 | Completed build's optional waiting line and ready line |
| compatibility_pack.go | 95, 128, 157, 176, 215 | Discovery digest, review digest, regenerate-all success, approved generation success, check success |
| compatibility_pack.go | 286, 311 | Aggregate and per-request qualification success |
| boundary.go | 39 | Source-discovered candidate lines |
| upgrade.go | 67 | Path after successful dossier publication |

### Approach

- Check each terminal report's existing fmt call and return 3 on its write error. Keep its format string and arguments exactly. Leave all earlier operational, cleanup and invalid-input returns unchanged, including toolchain injected-failure status 86. Add no report-error stderr fallback or rollback.
- Change the private `qualifyCompatibilityPackRequest` boundary to return `(status int, outputErr error)`. Earlier request/read/preparation/qualification/close failures return their original status and nil output error. Successful qualification attempts its existing report and returns status 0 plus that write error. The single-request command checks operational status first, then maps an output error to 3.
- In `qualifyAllCompatibilityPacks`, retain the first per-request output error while continuing the original `requestIDs` order, platform filtering and qualified counter. Stop only at the original operational failure and return its original status, even after an earlier stdout failure. When all requests succeed, attempt the unchanged aggregate summary, retain the first report error, then return 3 if any report failed. Preserve the zero-qualified status-1 branch.
- In boundary discovery, attempt all candidate lines in their existing order and retain the first output error separately. Check the discovery error first at its current boundary, returning 1; only an otherwise successful discovery maps its output error to 3. `DiscoverCandidates` currently returns nil candidates on error, so partial-discovery plus writer-failure coverage is unreachable and must not be mocked into existence.
- After successful toolchain build, attempt both existing report calls when `Waited` is true, even if the waiting write fails. Retain the first error and return 3 only after the ready line is attempted. Keep the completed cached build and stable toolchain publication.
- Use local error variables and the one private two-result boundary. No generic output/logging framework, callbacks, production test seam, stderr suppression wrappers, policy change, flag/default change, dependency, native-guard alteration or pin change belongs here. Report checking adds constant state and preserves all existing execution and transaction ordering.

### Regression and evidence

Capture literal healthy output controls against the audited production base before editing production. Introduce named `TestRunMaintainerOutput...` tests through public `run`, with genuine read-only-file failures using the existing `newRefreshOutput` EBADF pattern. Assert `errors.Is(output.err, syscall.EBADF)` before expecting status 3. Preserve invalid-input/status-2 and operational-failure/status-1 controls that attempt no stdout write. Exercise failed stderr alongside those primary failures without changing their statuses. Retain complete literal output expectations, including digests and newlines, rather than deriving expected values through the production reporting code.

Eight sites have direct portable success routes. Cover build-key, patch-validate, script-validate, boundary discovery, compatibility-pack review, generate-all, approved generate and check. Build deterministic valid recorded authoring inputs from the existing tiny fixture data and invoke the real commands without reviewer substitution. For authoring commands compare complete request, report, pack, approval and generation-manifest snapshots with the healthy completed operation, including file absence where relevant. Check remains read-only. Boundary discovery uses the actual pinned stock source and a retained literal candidate-output control bound to its platform/source identity.

Add portable patch-materialize and patch-regenerate success/error regressions by adapting the genuine tiny patch/source/archive fixtures in `toolchain/patch_test.go:128,196,572,626`. All source descriptors and archives remain temporary fixtures; production pins stay unchanged. Compare the materialized source, regenerated patch and unchanged inputs against the literal healthy result after stdout failure. Record an unavailable required fixture executable truthfully.

The seven remaining report sites require genuine successful operation inputs before their behavior can be claimed verified. They are conformance success, both completed-build reports, capability discovery, per-request qualification, aggregate qualification and final dossier success. Native evidence must cover failed waiting output followed by the ready attempt; `qualify --all` must cover an early output failure followed by later successful qualification, aggregate failure after successful per-request reports, and a later original status-1/status-2 operational failure outranking the earlier output error. Use real public commands and existing legitimate native operations. Keep each unavailable branch explicitly unproved; stock-host guard rejection, lint reduction and source review cannot replace these executions. Do not spoof a patched toolchain or success dossier.

Retain source identity, meaningful RED/green commands, actual writer error, exact healthy bytes, publication comparisons, exits, elapsed times, skipped/unexecuted branches and one fresh independent source-progress review. Compare unfiltered configured lint blocks with the audit's full baseline of 302 findings across 55 packages, including 244 errcheck, and the fresh scoped gomadtool baseline of 129 errcheck findings in `.flow/tmp/stdout-contract-a6a28720af/scoped-lint-base.log`. Expect these 17 stdout findings to disappear and disclose every unrelated residual or changed block; the full baseline exited 2 before errortype. Root owns Flow admission, task21's additive evidence link, review integration and the separate progress commit.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

- [ ] All 17 admitted stdout calls handle write errors with their existing formatting, arguments and attempt ordering. Successful operations with failed reporting return 3; original operational statuses, errors and cleanup precedence remain dominant.
- [ ] The private qualification boundary preserves all-request ordering, later operation-error precedence and final summary attempts. Boundary discovery preserves discovery-error precedence. Completed build preserves both report attempts. Retain source proof and identify every sequence lacking execution proof.
- [ ] Public-command portable regressions fail meaningfully on the original source and pass after correction, observing genuine EBADF. Literal healthy outputs, no-output status-1/status-2 controls and immutable publication snapshots pass without callback seams or changed assertions.
- [ ] Genuine conformance/build/discovery/qualification/dossier success and sequence checks pass where executable; each unavailable branch has its exact command/input requirement and remains incomplete. No unsupported-host run is labeled native qualification.
- [ ] Formatting, affected package tests, architecture/purity controls, vet/errortype, check-only validation and make lint-code-fast run with retained results. Unfiltered configured lint removes the 17 admitted findings without hidden residuals or exclusions. Full-gate exit and errortype reachability remain explicit.
- [ ] Fresh independent source/evidence review and the separate verified-progress commit feed task21. Original R18/R19 preservation, native Darwin, full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100 measurements, formal review and predecessor acceptance remain unchanged and open wherever unproved. Keep transferred native Linux obligations deferred and unverified under fn-128 until the owner's revival trigger is met. This correction grants neither SHIP nor completed original qualification.

## Quick commands

Use the pinned stock Go 1.27.1 and existing tools with `GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOWORK=off GOENV=off`, unless an existing fixture explicitly supplies its own local proxy. Root runs commands against frozen sources and serializes shared-cache/publication writers. Run matching baseline/final controls before broad checks.

```bash
go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool -run 'TestRunMaintainerOutput|TestRunBuildKey|TestRunPatchValidate|TestRunScriptValidate|TestRunCompatibilityPackRefresh|TestCompatibilityPackPaths'
go -C tools/gomad3 test -tags test_dep -count=1 ./cmd/gomadtool ./internal/compatibilitypack/authoring
go -C tools/gomad3 test -tags test_dep -count=1 -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$' .
go -C tools/gomad3 vet -tags test_dep ./cmd/gomadtool
make -C tools/gomad3 validate
make lint-code-fast GOLANGCI_LINT_BASE_REV=<task-base> GOLANGCI_LINT_FIX=false
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
```

From `tools/gomad3`, also run `golangci-lint run --config=../../.github/.golangci.yml --build-tags=test_dep --timeout=10m --fix=false ./cmd/gomadtool` and `GOFLAGS=-tags=test_dep errortype -test=true ./cmd/gomadtool`. Preserve task21's complete command ledger. On native Darwin the frozen batch still requires `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, the full `make -C tools/gomad3 test`, native/default integration, functional smoke, core/pack qualification and affected suites under their existing owners. These quick commands do not replace those gates.

## Done summary
Blocked:
Seventeen maintainer stdout reports now check write failures while preserving report bytes, completed publications, discovery and operation-error precedence, all-request qualification order and both completed-build report attempts. Ten portable public-command routes observe genuine EBADF and now return status 3; thirteen unchanged primary-error controls retain statuses 1/2 with no stdout attempt. Frozen final tests independently re-RED on exact baseline production and pass on the correction. Fresh same-family review is source-progress-acceptable with no introduced P0-P3 findings.

The original configured full lint gate falls from 302 to 285 findings, removing exactly the 17 stdout findings with no new or changed residual blocks after header-position normalization. Scoped CLI lint falls from 129 to 112; changed-line fast lint, vet/errortype, check-only validation and six architecture/purity controls pass. Portable selection passes 89 top-level tests and 508 records with one existing linux/arm64-profile skip; this is not the full CLI or native suite.

Seven conformance/build/discovery/qualification/dossier success reports and native build/multi-request output-ordering executions remain unproved. Original R18/R19 Darwin/full/default/functional/affected-consumer/matched-first-baseline/bounded 10-and-100/formal and predecessor requirements remain open wherever unproved. Linux remains deferred and unverified under fn128. Task21 consumes this correction and implements nothing. The handover, conductor checks and independent review are retained under task-46. Commit verified progress separately; no task completion or formal SHIP is claimed.

stage: impl-review - skipped(policy: required full lint red and original qualification incomplete; independent source-progress review accepts this bounded correction)
stage: plan-sync - skipped(config: disabled; task remains blocked)
Tracker sync: n/a (bridge inactive)

## Evidence
- Commits:
- Tests:
- PRs:


## Bounded fixture preservation amendment — 2026-10-09

Task51 owns only reconciliation of the compatibility-pack invalid-input failed-stderr expected status 2→1 with task8’s original checked-usage contract. See task-51/usage-status-20261009/admission.md and the parent’s dated decision. Production and all other assertions remain unchanged; historical task46 status-2 passes remain valid for their original source. All other acceptance, stdout report and authoring owners remain unchanged.
