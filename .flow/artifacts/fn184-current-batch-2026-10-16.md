# fn-184 isolation feasibility preparation

Task .1 is in progress; no mechanism is adopted and no acceptance is complete.
Main remains the qualified fn-180 snapshot source `90c1230140` plus task/docs
commits. Fn-180 R6 and fn-182 precede main-tree isolation adoption. The R5
baseline must be measured on unchanged main in a quiet Linux/arm64 window
before integration; no prototype timings count as that baseline.

One worker owns two disposable worktrees rooted at `90c1230140`:

- `/tmp/gomad-fn184-nested-prototype-20261016`, branch
  `prototype/fn184-nested-20261016`.
- `/tmp/gomad-fn184-tag-prototype-20261016`, branch
  `prototype/fn184-tag-20261016`.

They test module/constraint feasibility with explicitly temporary CLI stubs,
not production CLI behavior or portable-test preservation. Original native
sources, overlay bytes, fixtures and ledgers must remain preserved. Root owns
integration, commits, review and Flow completion. The worker was dispatched with
`gpt-6.1-sol` at medium; the tier judge was unavailable (`no_key`), so the
explicit project implementer model applies. Full prototype evidence is to be
retained at `/tmp/gomad-fn184-prototype-evidence-20261016`.

Read-only baseline preflight is retained at
`/tmp/fn184-r5-preflight-20261016-6UDVtR/commands.txt`. Current canonical
`test-harness` invokes patched-native runtime fixtures; those cannot run under
the existing hibernation waiver and must not be relabeled as portable coverage.
The source build command is `go build ./...`, not the Make runner/toolchain
build. Linux lint requires command-line tool overrides and
`GOLANGCI_LINT_FIX=false`. Race/cgo compiler support is available but has not
yet been exercised. Capture selected package/file inventories with `go list`,
deduplicating test variants; raw directory line counts are not selected-set
measurements. No baseline gate or timed cell was launched in this preflight.

Native execution stays deferred under fn-128/fn-149. No push, PR or CI action
is authorized. The prototypes neither change WASM semantics/identities nor
waive eventual default/selected source checks, preservation, baseline timings,
production CLI refusal controls or independent review.

## Disposable experiment results

The prototypes are retained only on their throwaway branches:
`1d72581886` (nested module) and `ffac982ecb` (pure tags, intentionally failing).
Root independently reran nested default CLI dependency discovery, stock build
and vet: exit 0, zero native packages. Pure-tag wildcard build again exited 1
because preserved runtime overlay packages remain visible to stock Go.
Both explicit native host source builds/vet passed in the worker evidence,
including vet of test-only qualification packages; no native execution ran.
Nested version/protocol checks passed, while tagging generated native sources
made the corresponding tag-candidate checks stale. Both preserve all 99
tracked overlay assets (97 Go files), fixtures and shared/WASM model sources.
The helper was not rebuilt and no runtime qualification is claimed.

These are feasibility stubs, not production seams: they temporarily exclude
69 mixed/portable test files containing 600 test functions. The complete
stub ledger and original failures are preserved beside `summary.md` and
`evidence.json` in the prototype evidence directory. No such exclusions may
be adopted; the real seam must split mixed tests and retain portable coverage.
Nested touches fewer native files and is the current recommendation, pending
the unchanged-main baseline and production design/verification.

Pin-impact on the parent reports the same four inherited pack invalidations
for both candidates, without a pin update. Its default baseline mapping for
the new child fails because the baseline has no child go.mod. New module
inventory/baseline mapping, archcheck selection, Make/lint/CI source wiring,
full CLI grammar/refusal preservation and the R5 measurements remain open.

## Provisional production source batch

The nested module is selected for isolated production preparation, not main
adoption. A clean worktree at main `8fe5c364de` is
`/tmp/gomad-fn184-production-20261016`, branch
`gomad-fn184-production-20261016`. No feasibility stub or broad test exclusion
is imported. The existing worker continues the coherent .1–.4 source batch
there; root remains sole committer and owns integration, timing and review.

Tasks .2–.4 are claimed for this preparation. Flow's ordinary .3 start refused
its unfinished dependencies; root verified .3/.4 were still unclaimed `todo`
and used dependency-only start overrides with explicit batch-first claim notes.
Dependency edges and every acceptance criterion remain intact. No task is
completed, and no baseline, native execution or main-tree adoption is inferred.
Task .3's stale versioned command/workflow names were reconciled through Flow
to the already-authoritative unversioned namespace; scope is unchanged.

The production seam must keep native code behind the explicit selection,
refuse native execution without fallback, retain installation-free historical
inspection and split mixed tests so portable coverage survives. Default and
selected source checks and ordinary portable tests may run in the isolated
worktree. Root's current R6 source stays frozen. Quiet unchanged-main R5
baseline, fn-182 timing, integration, current-candidate gates and independent
review still precede completion; native runtime execution remains deferred.

Module-boundary preservation clarification: the production worker may update
only command recipe strings in native `toolchain/version/descriptor.go` and
regenerate their maintained documentation, because the existing parent-module
`go run` recipes cannot enter the new child module. The replacement must keep
the Gomad-root working-directory and `--root` semantics, with runnable-command
coverage and generated validation. Record this wiring-only exception separately
in the preservation ledger and review its exact diff. Native runtime behavior,
descriptor values, patches, overlay assets, fixtures, ledgers and retained
identities remain unchanged; this is no license for a native implementation
rewrite or native execution.

The same wiring-only exception covers native `conformance/driver.go` fixture
command paths/tags and `upgrade/adapterregen/transaction.go` generator/verifier
command tables, with their exact-argv test pins. Root inspected the embedded
parent-relative package paths: they cannot cross the nested-module boundary.
Preserve fixture behavior, generator ordering, verifier coverage, working
directories, root arguments and staged-copy resolution, including the new
child module metadata. No native runtime execution or semantic rewrite is
authorized. List these specific adaptations separately in preservation review.

Selected staged adapter verification also needs the original outer module
metadata to resolve the new module graph. The worker may preserve a private
`tools/gomad`-depth stage and copy only the original outer `go.mod`/`go.sum`
into its enclosing private module root. Do not escape the temporary root,
copy server sources, rewrite pins or add a workspace. Those outer files are
read-only staging inputs, never publication/journal candidates. Preserve
existing publication/CAS/cleanup behavior and test actual generator/verifier
resolution and byte preservation; further source needs require actual failure
evidence before expansion. This is a module-wiring adaptation, not native
execution or a behavior rewrite.

## R5 executable preparation (not measurement)

Private driver, tests, dry-run plans and evidence are retained at
`/tmp/fn184-r5-measurement-preparation-20261016`. Driver SHA-256 is
`947f90f7b3657d3d3467fd26882c7fa351659ede14349bd3bd4ef043ab4c0319`.
Root inspected it and independently reran all nine Python tests: exit 0,
including raw numeric failure retention and credential exclusion from the
child environment and evidence. The driver defaults to no-dispatch dry-run;
future execution requires supplied source/HEAD/host bindings and a released
quiet window. Three independent cold-cache samples preserve literal gate
failures, separate module/tag inventories, and canonical native-harness
deferral from portable constituent timing. After-candidate harness timing
requires an audit of its final recipe; it is not automatically waived.
Quiet-host enforcement and concurrent-process capture remain root-owned,
outside this driver. Neither before/after measurement root exists, no host
gate or inventory was run, and R5 remains unverified. Current helper paths
are provisional inputs and must be rebound after the listener-history repair.

## Isolated source checkpoint and listener integration

Root read the frozen production handover at
`/tmp/gomad-fn184-production-evidence-20261016/{summary.md,evidence.json}`
and checkpointed the 94-path implementation as `d40debd4db` on the isolated
branch only. Its initial regex ledger reported 3002 test/benchmark/fuzz matches.
The later AST-level audit identifies 2998 actual top-level declarations,
with no missing declarations and 82 moved. The net difference of four is
26 embedded-Go string matches incorrectly counted, minus 22 real compact
functions missed by the regex. Exact file/line/name reconciliation is retained
in `premerge-audit-correction.json` in the private handover root. This corrects
the counting method, not a waiver for missing tests. All 99 overlay
assets and original WASM/shared-model bytes were preserved against its base;
approved native command wiring is listed separately. The bounded final audit
also adapted exactly three qualification `GOMAD_PACKAGES` selectors after
stock Go proved old relative resolution failed and qualified imports resolved;
fixture assertions and native execution deferral remain unchanged.

The provisional full WASM command exited 2 solely on missing canonical-helper
setup; all other packages passed, and the entire affected qualification package
passed after a verified helper copy. This is explicitly not a full green gate.
Default/selected source, CLI/Runner, architecture, lint, World race, generator
and safe adapter-host checks passed; original failures remain in the handover.
Four inherited pin-impact invalidations remain unchanged, not relabeled passes.

Root merged main `5e8bd20128` INTO the isolated branch, conflict-free, as
`2ca10ed67418b8ed2dc20cd54d1a1b0011cac654`. Main was not changed or adopted.
Root verified every WASM/shared-model/overlay source byte equals current main;
the listener repair is carried as predecessor work, not attributed to isolation.
The same worker is verifying this post-listener candidate with new private
helpers `0e3c62c5…` / `467a2f1b…`, including a required full WASM rerun, identity
probes and affected default/selected source checks. Preservation baselines must
now reference current main explicitly. Quiet unchanged-main R5 baseline,
fn-182 timing, final acceptance, independent review and Flow completion remain
open. No task, native execution or performance acceptance is claimed here.

The post-listener source preparation is now terminal. Full default WASM,
default harness, build/vet, default/selected lint, selected portable/static
checks, model tests, World race and generator validation passed; exact exits
and logs are in the private handover's `postmerge-gates.jsonl` and
`evidence.json`. The initial harness failure exposed the old filesystem-glob
selector guard; it now checks actual tagged Go package test-file metadata,
still requiring exactly three selectors and failing closed. Root independently
ran `go test -tags=test_dep ./internal/archcheck -run
'^TestNativeWrapperFixtureSelectorsResolve$' -count=1` (exit 0), then committed
only that guard as `ac48f5cc9f90e2ab5fac4ada0a301213fbdfaeac` in the isolated
branch. The worktree is frozen and clean. No main source adoption occurred.

Current-main AST preservation accounts for 3001 top-level declarations with
zero missing, including three listener-predecessor additions beyond the
premerge 2998. All 82 moved declarations have actual enabled Go-list vectors;
all 99 overlay assets remain byte-identical. Broad raw-overlay listing and
the four inherited pin-impact invalidations remain explicitly nonzero.

A standalone three-axis source-preparation review is being dispatched via
the Flow wrapper against base `5e8bd20128` and frozen HEAD `ac48f5cc9f`;
receipt `/tmp/gomad-fn184-source-preparation-impl-review.json`. It examines
R1–R4 source preservation and wiring, not full-spec acceptance. R5 measurements,
native qualification deferrals, final integrated acceptance and all four Flow
task completions remain open regardless of that source-review verdict.

Source-preparation review is finalized NEEDS_WORK on all three axes, rid
`cd8e197870316ff656939b52dcbde517`, same receipt. Root read every draw and
retained seven introduced defects: clock-audit script filesystem path; selected
conformance-unit gate omission; historical public `inspect --choices` routing;
unambiguous preinstallation WASM command routing; selected clean-checkout harness
toolchain ordering; default production/test native-dependency rejection; and
explicit native selection for the choice-exploration test's child supervisor
build. The four inherited pin invalidations are pre-existing, not repair scope.
The contracts draw had an invalid markdown anchor; its delivered verdict was
not retried. Root used the supported legacy merged-file finalizer to retain
every substantive finding, deduplicate overlaps and preserve the backend verdict.
Merged findings: `/tmp/gomad-fn184-source-preparation-merged-review.md`.

The existing isolated worker is repairing this coherent batch with focused
RED/GREEN regressions and affected portable/static gates. Root retains commit,
single-session re-review, adoption and completion ownership. No native execution,
R5 timing or main source adoption is authorized by this repair dispatch.

The eight-path repair batch (seven existing files plus a new controlled Make
integration regression) is checkpointed as `424cc83bbf`. All seven findings
have intended RED/GREEN evidence, with final default harness, selected
architecture/launcher/supervisor regressions, build/vet, dependency closure,
default/selected lint and stock-enabled WASM command consumers passing. Root
read the updated handover and actual logs, and independently reran public
launcher regressions. Intermediate structural-budget failures were fixed by
reusing existing inventory logic without raising the original budget. A full
initial lint failure log was accidentally overwritten; its numeric exit and
captured diagnostic remain explicitly recorded, not misrepresented as retained
full output. Later final lint logs have distinct names and pass.

Root merged main prefix repair `e4f63ac483` INTO this isolated candidate as
`68f70ceec85118154ae34518d2fd5279efa6a1c7`, conflict-free. Root verified every
WASM/shared-model/debug/Temporal diagnostic source byte equals main. This does
not adopt isolation on main. The worker is now running final post-prefix
integrated checks and one full WASM gate, refreshing preservation against e4
and accounting for its three predecessor regression declarations (3004 base
declarations expected, versus the prior 3001). Root will resume the existing
review receipt once green; no second fan-out, native execution or R5 claim.

Post-prefix integrated verification is terminal PASS at frozen `68f70cee`.
Full WASM: backend 319.072 s, cluster 118.404 s, command 55.319 s,
qualification 3.836 s and WASI 357.500 s, exit 0. All other affected
portable/static gates passed; expected overlay and inherited pin failures remain
explicit. Final handover is rebased to e4 with 3004 base declarations, no
missing tests, all 82 moved declarations enabled and 95 isolation paths.

Root read the final handover/logs and resumed the SAME review receipt once,
against original base `5e8bd20128`. The backend returned SHIP at `68f70cee`,
session `01a143ae-4735-7de2-956f-e8e6ef8126d6`, `gpt-6.1-sol:high`, exit 0.
All seven introduced findings are fixed. The reviewer verified preservation
hashes, enabled test membership and actual regression/full-WASM logs; the four
inherited pack invalidations remain nonblocking and unchanged. The overwritten
initial lint-red log remains an explicit retention gap, not reconstructed by
later green logs. Receipt: `/tmp/gomad-fn184-source-preparation-impl-review.json`.
This is R1–R4 SOURCE-preparation approval only. R5 is wholly unmeasured; main
adoption, task/spec completion and native qualification remain unapproved.

R5 structural preparation now retains 15 metadata-only Go-list captures under
`/tmp/fn184-r5-measurement-preparation-20261016/structural-inventories`.
No timed window or source adoption has started. The original physical-root LOC
calculation was invalid: before isolation, the stock GOROOT lives inside the
Gomad tree and was incorrectly counted as project code. Original raw captures
and invalid summaries remain retained, not overwritten as corrected evidence.

The driver now requires declared module identity and matching module directory,
excludes Standard/GOROOT and foreign modules, rejects ambiguous ownership and
retains raw overlay errors. Corrected outputs use `.module-owned.json` names
and `module-owned-summary.json`, derived from the SAME raw records with no new
queries. Driver SHA256 is
`f19d507174d159f6cf62c4adb505dbeb05fa5704e47be11cb707bdbae4d1dbe7`;
root independently ran `python3 -I test_measure.py`: 14 tests PASS, 0.153 s.
Build-inventory structural counts are 80 before/default host packages versus
52 after/default, with 118133/131561 versus 80898/85253 production/test lines.
The after/selected set contains 82 packages. Before and selected raw queries
retain overlay errors; query exit 0 is not a complete clean inventory claim.
These are source-selection facts, not wall-time savings or R5 acceptance.
