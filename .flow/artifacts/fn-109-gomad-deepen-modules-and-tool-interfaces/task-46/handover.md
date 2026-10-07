# Task 46 source handover

Maintainers now receive status 3 when any of the 17 admitted successful-operation stdout reports fails. The correction preserves existing formatting, publications, original operation-error statuses, qualification order, discovery-error precedence and both completed-build report attempts. This is verified source progress; task acceptance remains incomplete.

Task: `fn-109-gomad-deepen-modules-and-tool-interfaces.46`, status `in_progress`.
Base: `a6a28720af58fe65cb6fd3bc618ef8102765b50b`, branch `gomad`.
Workspace: `/Users/stephan/Workspace/skunkworks/gomad/temporal`.
The conductor owns commits, Flow records, remaining checks and independent review. The worker made no commit, lifecycle mutation or review verdict. All worker commands exited and the Go lane was released with sources frozen.

Tier: session (jev-unavailable(no_key)); explicit AGENTS model overrides unavailable optionaljudge.
stage: impl-review - skipped(policy: host-deferred source-progress-only; conductor owns independent review and required full lint remains red)

## Verification

[observations.json](observations.json) retains actual command strings, exit codes, Bash `time -p` durations, source hashes and local full-log references with SHA-256. Local bulk logs stay under `.flow/tmp/stdout-contract-a6a28720af/`.

| Observation | Exit | Real seconds | Evidence |
| --- | --- | --- | --- |
| Original focused controls before edits | 0 | 5.10 | `focused-baseline.log` |
| Final ten-route public-command RED on original production | 1 | 2.18 | `red-publications.log` |
| Focused regressions and existing controls after correction | 0 | 2.31 | `focused-green.log` |
| Same final tests over exact original-production Go overlay | 1 | 1.73 | `overlay-red.log` |
| Expanded portable CLI/pack/authoring families after correction | 0 | 2.57 | `portable-green.jsonl` |
| Original portable families over exact original-production overlay | 0 | 0.92 | `portable-base-overlay.jsonl` |
| Scoped CLI vet | 0 | 0.13 | `vet.log` |
| Standalone scoped errortype | 0 | 0.52 | `errortype.log` |
| Unfiltered scoped configured lint | 1 | 1.06 | `scoped-lint-final.log` |

Each of the ten RED routes logged `stdout write observed EBADF` before failing on status 0 versus 3. All ten pass after correction with the unchanged final tests. The routes are patch/script validation, build key, source boundary discovery, pack review, generate-all, approved generation, check, patch materialization and patch regeneration. Thirteen primary-error controls preserve ten invalid-input status-2 and three operational status-1 results while stdout remains untouched and stderr observes actual EBADF.

Authoring commands start healthy and failed-output runs from separate identical request seeds. Full tree snapshots compare requests, reports, packs, approvals, generation manifests and generated consumer tests. Check also proves its tree is unchanged. Unapproved regeneration preserves pack absence. Tiny materialization proves the complete expected source tree and unchanged patch inputs; regeneration proves literal complete patch bytes, unchanged candidate and source archive inputs. Real patch, Git and gofmt executables ran successfully. No reviewer or qualification substitution was added.

The boundary literal control contains 217 exact lines captured against the pinned stock Go 1.27.1 linux/arm64 source. Its toolchain and source inventory identities are retained in the JSON. On other hosts only an unavailable literal-control subtest skips; the actual discovery/report-failure regression still executes.

The original portable selection passes 84 top-level tests and 480 test records. The final selection passes 89 top-level tests and 508 records. Both cover three packages and retain one actual `TestHostPacksBindCurrentProfile` skip because linux/arm64 lacks the deterministic profile. This explicit family selection does not qualify the complete CLI package or native/full suite.

Scoped lint falls from 129 to 112 errcheck findings. Multiset comparison of the complete diagnostic blocks, normalizing header line/column positions only, removes exactly the 17 admitted stdout calls and preserves all 112 residual blocks without new or changed diagnostics. The original full lint baseline remains red at 302 findings across 55 packages, exit 2 before errortype. Formatting and `git diff --check` pass. The conductor runs fresh focused JSON, six architecture/purity controls, validation, fast lint and original full lint after the freeze.

## Unexecuted success and ordering proof

The following seven report sites remain unproved by successful native execution. Source inspection is their current evidence. The stock linux/arm64 host has no patched Go driver and is outside the qualified Runner platforms; guards stay unchanged.

| Report | Required public command and genuine inputs |
| --- | --- |
| Conformance success | `gomadtool test --root=<module> --mode=test-runtime --go=<patched-go>` with all selected real fixtures succeeding on qualified native Darwin. |
| Completed build waiting and ready reports | `gomadtool toolchain-build --root=<module> --bootstrap-go=<stock-go>` with a genuine successful qualified-host build. A real contending same-key build must produce `Waited=true`; fail the waiting stdout write and observe the ready attempt while verifying cached and stable publication. |
| Capability discovery digest | `gomadtool compatibility-pack discover --root=<module> --request=<legitimate-draft> --working-dir=<actual-target-module>` with successful real preparation and publication. |
| Per-request qualification | `gomadtool compatibility-pack qualify --root=<module> --request=<recorded-request> --working-dir=<mapped-module>` with genuine successful current review and close. |
| Aggregate qualification | `gomadtool compatibility-pack qualify --root=<module> --all` using the real working-directory table and at least two successfully qualified host-platform requests. Prove an early output failure followed by later success, final aggregate attempt, aggregate-only failure and a later original status-1/status-2 failure overriding earlier output failure. |
| Final dossier path | `gomadtool upgrade-dossier --root=<module> --baseline-ref=<baseline> --corpus-report=<retained-gomad3-core-report>` with genuine successful mandatory gates and a successfully published dossier. |

`DiscoverCandidates` returns nil candidates on error, so simultaneous partial-discovery and output failure is unreachable. No fake successful toolchain, dossier, qualification or partial discovery was introduced. Original R18/R19 native Darwin, full/default/functional/affected-consumer, matched-first-baseline, bounded 10/100-job, formal review and predecessor acceptance remain open. Linux qualification remains deferred and unverified under fn-128.

## Defect route

- Prior fixes: history includes `a45ebab979` for refresh reporting; the 17 admitted calls remained unchecked. Reporting memory was read. GitHub PR/issues were unchecked because the existing GH_TOKEN failed authentication.
- Diagnosis: exact healthy public output and completed publication precede genuine EBADF and the incorrect status 0 on ten routes. The missing fmt-result checks explain the observed failure.
- Introduced by: not bisected; no known-good revision was supplied.
- Base: original sources fail all ten intended regressions; exact-source overlay repeats the same result with unchanged final tests.
- Head: ten portable regression routes and original portable controls pass on the frozen source correction.
- Live: public `run` command tests supply portable proof. The seven native success reports and required ordering sequences remain unexecuted.

Fixture preparation initially used an unanchored approval literal, omitted archive directory headers and omitted Git-generated patch index/hunk bytes. Those setup mismatches were corrected from actual original-source healthy observations before production edits and are excluded from the final RED claim. No old assertions, generated files, pins, dependencies, flags or host guards changed. The two pre-existing unrelated `.turbo` files remain untouched.
