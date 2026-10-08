I’ll read the spec and task requirements, then check the diff and supporting evidence for completion gaps.

The spec explicitly transfers native Darwin and Linux qualification to fn-149 and fn-128. I’ll check the source requirements that remain here, including approval controls, mapped-module refresh, preservation, documentation, and measured bump steps.

The added tests cover real two-module discovery, partial approval and rerun, the default adapter publication pipeline, and both supported source sets. The measurement distinguishes the observed 6→5 repair steps from the normalized 6→4 comparison. Local validation and focused tests stopped before execution because the sandbox blocked Go’s toolchain-verification network request; I’ll check the retained receipts against the reviewed source.

The retained evidence checks out: all 119 task-2/3/4 log hashes match, and 1,033 of the latest manifest’s 1,034 source paths are unchanged. The remaining difference is the milestone status update. I found no completion gap in the retained requirements. Nine preparation and status artifacts concern other specs; I’ll flag them as untraced changes without treating them as blockers.

## Requirements Extracted

1. R1: Retain a re-measured inventory of every pin class, counts, and existing bump commands/manual steps; correct milestone references.
2. R2: Evaluate candidate `go.mod`/`go.sum` inputs and report affected adapters, pack rules/source identities, interception fingerprints, and clock references.
3. Produce path-free canonical JSON and human output using the build’s descriptors.
4. Classify unevaluable pins as unknown; implement exit statuses 0/1/2/3.
5. Cover adapted/packed fixture bumps, changed sums, removed/replaced modules, and indirect-only bumps.
6. Resolve outside target modules with private caches, exported proxy settings, infrastructure-error classification, and unchanged target module files.
7. Cover both supported source sets and preserve unavailable-platform handling.
8. R3: Regenerate exact-version adapter anchors using existing structural rewrites; reject missing, moved, or ambiguous anchors.
9. Print upstream changes, proposed anchors, and their approval digest; dry runs and wrong approvals publish nothing.
10. Stage and verify the complete output set; lock, revalidate, journal, recover, and handle competing publication.
11. Exercise a real adapter bump, or document that no adapted dependency moved and use the authorized fixture.
12. Report stale libc-bound packs.
13. R4: Refresh every invalidated request in its mapped module through discovery/review/generation up to explicit approval; reject missing mappings.
14. Validate fresh approval digests, preserve partial progress, and reject older approvals.
15. Preserve foreign-platform requests; remove variants only with evidence that nothing selects them.
16. Preserve exact pins, fail-closed behavior, identities, replay compatibility, public contracts, and `gomad` grammar.
17. R5: Update README, SPEC, CLI, architecture, generated upgrade guidance, roadmap, and milestones; check links and command inventories.
18. Measure manual steps against the original matched baseline, counting commands and hand edits.
19. Retain source acceptance: portable assertions, lint, both-source-set static checks, generated validation, preservation, and integrated review.
20. R6: Native full-test, pack, core, consumer, and replay qualification remains with the explicitly transferred Darwin/Linux owners; partial source evidence must retain its limits.

## Coverage Verification

1. **COVERED** — Baseline and current inventory retain all six pin classes and bump steps. Evidence: `.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/baseline.json:1`, `MILESTONES.md:309`.
2. **COVERED** — Evaluation reads adapter identities, packs, interception inventory, and clock references. Evidence: `tools/gomad3/upgrade/pinimpact/pinimpact.go:149`.
3. **COVERED** — Shared registry/pack inputs, canonical encoding, human rendering, and explicit-root diagnostic redaction are implemented. Evidence: `tools/gomad3/upgrade/pinimpact/pinimpact.go:161`, `tools/gomad3/upgrade/pinimpact/render.go:13`, `tools/gomad3/upgrade/pinimpact/packs_directory_test.go:21`.
4. **COVERED** — Unknown pins count as invalidated; command status controls are retained. Evidence: `tools/gomad3/upgrade/pinimpact/pinimpact.go:187`, `tools/gomad3/cmd/gomadtool/pin_impact_diagnostics_test.go:13`.
5. **COVERED** — Fixture, checksum, removal, replacement, and indirect-bump assertions exist, with portable source counterparts distinguished from native wrappers. Evidence: `tools/gomad3/upgrade/pinimpact/portable_fixtures_test.go:14`, `tools/gomad3/upgrade/pinimpact/pinimpact_test.go:375`.
6. **COVERED** — Private cache and external resolution preserve proxy inputs and target files; unavailable-module errors have focused coverage. Evidence: `tools/gomad3/upgrade/pinimpact/modules.go:198`, `tools/gomad3/upgrade/pinimpact/resolver_test.go:76`.
7. **COVERED** — Added tests reproduce all 15 adapters’ two-platform source pins; foreign requests remain untouched. Evidence: `tools/gomad3/deterministicio/adapter_regenerate_portable_test.go:69`, `tools/gomad3/internal/compatibilitypack/authoring/refresh.go:125`.
8. **COVERED** — Existing regeneration owns structural rewriting; added refusals assert zero/two matches, deleted source, and untouched scratch. Libc has corresponding controls. Evidence: `tools/gomad3/deterministicio/adapter_regenerate_portable_test.go:291`, `tools/gomad3/deterministicio/adapter_regenerate_portable_test.go:342`.
9. **COVERED** — Approval comparison precedes apply; dry-run source preservation and default-pipeline publication are exercised. Evidence: `tools/gomad3/upgrade/adapterregen/adapterregen.go:186`, `tools/gomad3/upgrade/adapterregen/default_pipeline_test.go:44`.
10. **COVERED** — Staging, verification, revalidation, journal publication, recovery, and release composition remain integrated. Evidence: `tools/gomad3/upgrade/adapterregen/transaction.go:77`, `tools/gomad3/upgrade/adapterregen/lock_release_test.go:13`. Original drift, concurrency, and interruption tests remain.
11. **COVERED** — No root adapted dependency had moved; the controlled Sprig v3.3.0→v3.2.3 fixture runs the default pipeline and verifies published digests. Native workload qualification remains transferred. Evidence: `tools/gomad3/upgrade/adapterregen/default_pipeline_test.go:15`, `.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.2.md:332`.
12. **COVERED** — Stale adapter-bound packs are collected and tested. Evidence: `tools/gomad3/upgrade/adapterregen/adapterregen.go:219`, `tools/gomad3/upgrade/adapterregen/adapterregen_test.go:535`.
13. **COVERED** — Live evaluation precedes saved-report merge; real two-module tests demonstrate distinct candidate versions and refusal before writes for invalid mappings. Evidence: `tools/gomad3/cmd/gomadtool/compatibility_pack_refresh.go:93`, `tools/gomad3/cmd/gomadtool/compatibility_pack_refresh_resolved_test.go:196`.
14. **COVERED** — The real-resolution test rejects an old approval, approves one request, and preserves it on rerun while reporting the other. Evidence: `tools/gomad3/cmd/gomadtool/compatibility_pack_refresh_resolved_test.go:239`.
15. **COVERED** — No variant is removed in this range. Read-only selection proves both v041 and v047 remain selected; foreign-platform requests retain their artifacts. Evidence: `tools/gomad3/internal/compatibilitypack/authoring/variant_selectors_test.go:12`, `tools/gomad3/internal/compatibilitypack/authoring/refresh.go:125`.
16. **COVERED** — Private helper extractions preserve public validation order and comparison/encoding bodies. Production dependency pins, runtime patch/overlay, and `gomad` grammar are unchanged. Evidence: `tools/gomad3/deterministicio/adapter_registry.go:123`, `tools/gomad3/deterministicio/requirements.go:16`, `tools/gomad3/deterministicio/bootstrap.go:25`, `tools/gomad3/deterministicio/profile.go:295`.
17. **COVERED** — Documentation consistently describes mandatory live discovery plus saved-report merge; the guide template and generated output agree. Evidence: `tools/gomad3/README.md:1156`, `tools/gomad3/CLI.md:609`, `tools/gomad3/SPEC.md:527`, `tools/gomad3/ARCHITECTURE.md:991`, `tools/gomad3/toolchain/version/descriptor.go:300`. The retained audit covers seven documents, links, registered verbs, and actual help invocations.
18. **COVERED** — Measurement retains the first denominator and distinguishes observed repair **6→5**, normalized repair **6→4**, workflow total **10**, and instrumentation-inclusive total **16**. Evidence: `.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/source-acceptance-20261008/measurement.md:7`.
19. **COVERED** — Source-bound receipts retain portable coverage, scoped checks, both supported static source sets, preservation, and independent reviews. Evidence: `.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/conductor-source-acceptance-20261008/integrated-candidate-audit.md:3`. I verified all **119** task-2/3/4 receipt-log hashes and **1,033/1,034** latest manifest paths; only the milestone status update differs.
20. **COVERED through explicit deferral** — Native obligations remain unverified under fn-149/fn-128, with no source-only or historical pass represented as current native qualification. Evidence: `.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md:5`, `.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md:73`.

Fresh validation and focused-test attempts could not execute here: the default Go launcher required sandbox-blocked network verification, and direct pinned Go could not create its build directory on the read-only filesystem. Coverage conclusions use inspected code and hash-verified retained evidence; no fresh runtime pass is claimed.

## Reverse Coverage (untraced changes)

The 470 fn-113 artifact files support R3–R6 acceptance, measurements, preservation, and review. The fn-113 metadata, production changes, tests, and documentation trace to the requirements above. The added qualification-manifest documentation row and corrected core-corpus wording are **LEGITIMATE_SUPPORT** for R5’s command/document consistency.

These nine files concern other specs:

- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/conductor-preparation-20261008/dispatch.json` — **UNRELATED_CHANGE** — Preparation dispatch for fn-109.7.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/conductor-preparation-20261008/findings.md` — **UNRELATED_CHANGE** — Future preparation-owner acceptance findings.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-7/conductor-preparation-20261008/scout-state.json` — **UNRELATED_CHANGE** — fn-109.7 preparation state.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-8/conductor-preparation-20261008/dispatch.json` — **UNRELATED_CHANGE** — Preparation dispatch for fn-109.8.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-8/conductor-preparation-20261008/findings.md` — **UNRELATED_CHANGE** — Future inspection-owner acceptance findings.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-8/conductor-preparation-20261008/scout-state.json` — **UNRELATED_CHANGE** — fn-109.8 preparation state.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/conductor-preparation-20261008/findings.md` — **UNRELATED_CHANGE** — Future transport-owner acceptance findings.
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-9/conductor-preparation-20261008/judge-state.json` — **UNRELATED_CHANGE** — fn-109.9 preparation metadata.
- `.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/conductor-completion-gate-observation-20261008.md` — **UNRELATED_CHANGE** — Observation of another spec’s retained completion review.

These are acknowledgment flags, with no identified fn-113 completion impact.

## Gaps Found

None.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Original baseline, current inventory, manual-step ledger, milestone links |
| R2 | met | Shared-input report, canonical output, error/status and fixture controls |
| R3 | met | Governed regeneration, structural refusals, complete publication, real source fixture; native qualification transferred |
| R4 | met | Mapped discovery, fresh approval, partial rerun, foreign-platform preservation, retained selected variants |
| R5 | met | Updated procedure, generated guide, document audit, matched measurement |
| R6 | deferred | Native Darwin/Linux gates transferred explicitly; retained source checks covered |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>
