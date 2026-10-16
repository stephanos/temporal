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
