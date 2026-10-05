# Current Choice Trace guidance

README, ARCHITECTURE and TUTORIAL now identify the current Choice Trace and
replay evidence as v3 in all six selected passages. The architecture explains
select-result readiness projection onto matching poll decisions, unknown
readiness without a matching result, stored-v2 refusal, and legacy-v1 inspection
without a replay plan. Only these six replacements and the seven-line
architecture paragraph change product bytes.

Task fn-109.20 remains `in_progress`. Root owns the independent source-progress
review, staging, commit and Flow lifecycle. The worker made no commits or Flow
writes. Base is `d108c313dc114c2a83f1301a75255f9e4a4dc5e6`; the range is
`d108c313dc114c2a83f1301a75255f9e4a4dc5e6..HEAD` with the product changes
uncommitted.

## Claim mapping

Lines refer to both the saved BASE and current candidate. Exact before/after
text also lives in `before.json` and `evidence.json`.

| Guide and line | Before | After | Authoritative behavior and existing control |
| --- | --- | --- | --- |
| README:135 | recorded as v2 stable logical decisions | recorded as v3 stable logical decisions | `choice/trace.go:133` DecodeStoredTrace; `choice/readiness_test.go:192` TestChoiceReadersRejectOtherWireVersions |
| README:867 | When v2 choice recording is enabled | When v3 choice recording is enabled | `choice/tape.go:191` ProjectReplayPlan requires complete v3; `choice/tape_test.go:206` TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace |
| ARCHITECTURE:278 | complete v2 trace into a Decision Tape | complete v3 trace into a Decision Tape | `choice/tape.go:191` ProjectReplayPlan; `choice/readiness_test.go:66` TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions |
| TUTORIAL:389 | bounded v2 Choice Trace | bounded v3 Choice Trace | `choice/trace.go:133` DecodeStoredTrace; `choice/readiness_test.go:192` TestChoiceReadersRejectOtherWireVersions |
| TUTORIAL:395 | complete v2 Choice Trace | complete v3 Choice Trace | `choice/tape.go:191` ProjectReplayPlan; `choice/tape_test.go:206` TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace |
| TUTORIAL:497 | replayable v2 choice | replayable v3 choice | `choice/trace.go:133` DecodeStoredTrace and `choice/tape.go:191` ProjectReplayPlan; reader-version and legacy refusal controls above |

`choice/tape.go:232` projectSelectReadiness matches result origin, site and poll
step before attaching readiness. Unnamed decisions retain unknown readiness.
ProjectReplayPlan excludes observations and alternatives below two from forced
decisions. `choice/legacy_v1.go:21` decodes v1 for inspection; its record decoder
preserves decision/observation flags. ProjectReplayPlan returns
ErrReplayUnavailable for v1. No claim that every legacy record is an observation
is made.

The migration remains owned by [fn-114.11's retained gates](../../../fn-114-gomad-correct-search-path-defects-and/task-11/gates.md).
Those historical native results qualify their own source snapshot. fn-114.14
retains final-consumer qualification ownership.

## Verification

`baseline: green` from fresh, captured command exits. `baseline-receipts.json`
and `final-receipts.json` retain exact argv, cwd, offline environment overrides,
seed variables removed, timestamps, elapsed times, log hashes and source hashes
before and after each command. Every command exited 0 in both phases:

- `grep -n 'darwin/arm64\|linux/amd64' SPEC.md ARCHITECTURE.md README.md`
- `go test -json -count=1 -tags test_dep . -run '^(TestCurrentVocabularyHasNoLegacyCampaignBoundary|TestMakeTargetsMatchTheirOwnership)$'`
- `go test -json -count=1 -tags test_dep ./choice -run '^(TestChoiceReadersRejectOtherWireVersions|TestProjectReplayPlanCarriesSelectReadinessOntoPollDecisions|TestProjectDecisionTapeRejectsObservationOnlyAndLegacyTrace)$'`
- `make -C tools/gomad3 validate`

Both phases observed exactly the two and three expected passing test names.
`baseline-doc-check.json` records exit 1 on exactly the six stale claims; the
remaining baseline document checks pass. `final-doc-check.json` records all
25 checks passing. It checks literal claims, all other guide bytes, complete
fenced-command preservation, balanced fences, local links/fragments, unrelated
v2 references, whitespace, source scope, unchanged HEAD and both user files.

`before.json` freezes 1,108 tracked files and their Git-index records before
guide edits, including all of tools/gomad3, tools/gomad3sim,
tools/gomad3integration, tests/gomadfunctional, root Makefile, .github
configuration, module pins, AGENTS and MILESTONES. The BASE closure hash is
`b38ef68ebb35740962c8f150287880f5e92e0618aa115ea50e0104a640881ffc`.
`final-source.json` binds the candidate to
`218bfa4a6e4e1459a35135a7042fc5af99d545f6f782c065a80ff959a59e6db5`.
Only the three guides differ. Production tests, generated output, pins, CLI,
SPEC, MILESTONES and qualification dispositions retain their BASE bytes.
Existing `.turbo/plans/gomad3-glossary-update.md` and
`.turbo/technical-debt.md` retain their hashes.

`host.json` freshly records Linux/aarch64, stock go1.27.1 linux/arm64 and the
absent patched toolchain. All new execution evidence is developmental. The
[retained actual integrated lint log](../../../fn-113-gomad-reduce-version-pin-maintenance/task-3/v041-restoration-20261005/root-integrated-lint.stdout.log)
still records 317 diagnostics, with errortype unreached. This correction runs
no broad Go lint and supplies no new lint result. Executable source is unchanged.

Original task-19 dependency, R18 recorded-format preservation reconciliation,
formal review, required Darwin/full/affected gates and fn-105.5 D5 acceptance
remain open. This documentation correction implements no legacy compatibility
and supplies no native qualification. Transferred Linux evidence stays with
fn-128 and is nonblocking here. Historical receipts remain unchanged.

Tier: session (jev-unavailable(no_key)).

stage: impl-review - skipped(policy: original gates remain open; conductor owns independent source-progress review)
