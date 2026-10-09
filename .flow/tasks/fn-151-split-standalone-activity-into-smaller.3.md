---
satisfies: [R4]
---
# fn-151-split-standalone-activity-into-smaller.3 Verify behavior and generated artifacts after the activity model split

## Description
Prove the joined Activity subject split against structural baseline `4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd`, then regenerate and review its artifacts once after tasks .1 and .2. Keep the three tasks serial and publish generated trees together. This task preserves the existing Model and realization semantics; inherited strict Activity completion, fatal-failure and pause/resume failures remain assigned to Batch 5, and heavyweight Quint JSON memory work remains deferred to fn-154. The 2026-10-09 resource-gate disposition below assigns new native-memory and scratch-storage failures to fn-157.

**Size:** M
**Files:** the three standalone Activity IR files and their lint companions, affected standalone Activity Cases, the Case manifest and two source-position goldens
**Touches:** [model/ir/activity-standalone.json, model/ir/activity-standalone-record.json, model/ir/activity-standalone-race.json, model/ir/activity-standalone.lint.json, model/ir/activity-standalone-record.lint.json, model/ir/activity-standalone-race.lint.json, model/cases/activity-standalone*-case.json, model/cases/manifest.json, model/irgen/testdata/lifts/expected/hints.json, model/irgen/testdata/lifts/expected/hintsRefused.json]

The third Activity artifact is `activity-standalone-race.json`; no `activity-standalone-taskqueue.json` exists. Tasks .1 and .2 own source, consumer and layout-documentation updates. If this gate discovers a missing update, name the exact file and obtain root's scope authorization before changing it. Current functional generated fixtures and the canary pin contain Nexus Cases only. Check those managed trees without rewriting them; a proven affected pin requires its exact files and root authorization before regeneration. No `tools/**` wildcard, semantics-document edit or unconditional canary update belongs to this task.

### Approach

- Capture baseline and candidate source/artifact hashes before comparison. Reuse the reader-table, Check-receipt, Query-answer, deterministic identity and generated-Case comparison mechanisms in `/tmp/umpire-fn1454.aTUadX/.flow/tmp/fn1454/equivalence.go`. Replace its fn-145-specific schema equality and mappings with an explicit fn-151 ledger of moved source positions and changed declaration owners. Do not add a generic identity normalizer, gate framework, cache or trust mechanism.
- Independently hold the original Activity subset to three IR Models, 168 Check receipts and 154 Query occurrences. Account for every original Query once in its original artifact placement, with identical form, Property meaning, Scenario start/path, Limits, total, expectation, exploration, monitor semantics and answer. Candidate derived owners can add the machine/refinement receipts their declarations require; enumerate these additions explicitly instead of preserving an obsolete candidate receipt count. The shared `completes` Property has one original owner and three intentional replacement owners. Preserve its meaning separately for each consuming Query.
- Keep `competingTimers.scheduleToStartFirst` and `competingTimers.scheduleToCloseFirst` in `activity-standalone-record.json`, with `no-realization` standing and no RunExpectation. Their deadline subject's executable Queries are rooted separately in the primary Activity IR. Check exports and realization availability per artifact, since exporting a whole subject's `queries` beside its realization could change these two standings.
- Compare all original machine tables, including ordered results, evidence, starts/ends, assumptions, monitors and refinements. Compare each new subject's complete table and inherited phase/end/refinement against ActivitySystem. Preserve Product visibility and closedness through the unchanged refinement map. Independently validate paired Product/System projections. Keep exact declaration inventory equality using family/owner/kind/name so the three `completes` owners cannot collapse into one simple-name entry.
- Exercise the comparison with bounded negative specimens. It must reject an omitted Query/root, a weakened Property, a removed refinement or monitor, changed bounds/totals/expectations, and changed controller instructions or evidence. Use the existing comparison seams; retain original tables and assertions as independent oracles. Verify the inherited pause hold, committed pause/unpause and release order remains causal.
- Map changed Definition, Model, Query, realization and Case identities explicitly. Recompute dependent Program/Contract references and Case checksums in their existing dependency order. After regeneration, compare complete generated Programs, Contracts, expectations and manifest standings under the declared mappings. Explain every remaining delta. Update the `hints`/`hintsRefused` source-position goldens only when their actual bytes changed. Regenerate an affected functional mirror only after proving the changed selected pin and receiving authorization for its concrete files; unchanged Nexus mirrors and canary pins retain their bytes. Historical Runs, receipts and replay companions remain byte-for-byte unchanged; historical compatibility continues through its original Case companions and strict identity checks.
- Run the production Scala/lift/Case gate and the full canonical Go suite once with `test_dep`, `-p 2`, `-timeout 30m` and JSON output. The full suite covers affected packages; add a focused invocation only when a task-specific identity proof requires a check outside that coverage. Use the existing shared heavy-run serialization and source/resource receipts. The production gate may use `MODEL_GATE_ARGS=--skip-go-checks` only alongside the separately recorded covering Go suite. Record actual full-suite RED and inherited failures separately from split regressions. Preserve all strict Batch 5 assertions and original export/native/receipt/replay domains. An unexplained new semantic, identity or assertion failure blocks completion; named resource interruptions follow the explicit fn-157 disposition below. A filtered pass cannot replace the canonical result. No heavyweight memory repair or repeated full run to pursue green belongs to this split.

### Resource-gate disposition, 2026-10-09

The owner directed the conductor to defer gates stuck for roughly an hour and deliver independently verifiable work. Under that rule, `fn-157-bound-native-verification-memory-and` retains the five kernel-confirmed canonical Go victims, the no-update Model gate's generator OOM and scratch-storage exhaustion, and the ordinary Case/fixture generator OOMs. This changes this task's gate acceptance. The commands below remain required recorded invocations, with their actual RED results preserved, but their resource-fit restoration is deferred to fn-157 rather than required before this structural split closes. Fn-154 retains exhaustive Quint JSON agreement and its missing fixture selections. Neither deferral supplies passing evidence or certifies concurrent fit.

The complete native comparison and complete seven-Model isolated Case comparison must independently pass with pinned inputs, exact mappings and exercised negative controls. Production generation, Scala lint, Go lint and the exact corrected consumers must pass. Preserve current generated IR/Cases, unchanged functional mirrors and historical Runs. Report killed and unexecuted tests explicitly. The corrected completion guard mutation still requires a task-specific actual result; strict completion/fatal/pause remain Batch 5 debt with assertions unchanged. Canary controller/publication failures remain separate and must receive evidence-backed affected-package classification before review. No memory optimization, fixture reduction, forced GC, package-concurrency reduction, assertion weakening or repeated covering Go run belongs to this task.

Use the existing cumulative split lint target with `GOLANGCI_LINT_BASE_REV=4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd` and `GOLANGCI_LINT_FIX=false`. It covers all three tasks' Go changes. Retain the default lint invocation's 743-finding RED separately; its February local-main comparison selects existing branch debt, and source/log classification identifies no fn151-introduced finding. A passing cumulative check grants no default full-branch lint pass. Restore the nine accidental default-autofix edits to proven pre-lint bytes before any accepted verification.

The focused corrected completion guard negative started its exact parent and child, then exited 1 after 39.446 seconds when the kernel OOM-killed conformance at 23:02:45 UTC. Its expected GuardError and assertion remain UNOBSERVED. Fn-157 explicitly retains this required negative's restoration with unchanged mutation and assertion. This recurring native-resource interruption is deferred under the same owner policy and grants no negative-test pass credit. Both complete Canary packages passed on restored source through the corrected runtime wrapper; classify their canonical failures as environment-sensitive with the low-level ENOTEMPTY cause still unknown, preserving the canonical RED and the documented inference about temporary-directory selection.

### Investigation targets

- `tools/umpire/check/activity_parity_test.go` and Activity Scala regressions for exact declaration inventory and independent transition/refinement pins.
- `model/temporal/features/activity/standalone/Standalone.scala`, the new subject owners and `system/Realization.scala` for roots, attachments and per-artifact realization availability.
- `model/cases/manifest.json`, lowerer identity/expectation checks and affected generated consumers for standings and dependent references.
- The fn-145 equivalence implementation and sealed baseline receipts for reusable comparison mechanisms and inherited RED dispositions.

### Quick commands

Run from the repository root. Record the adapted fn-151 equivalence command and its baseline/candidate hashes when its concrete interface is prepared. Add generation commands for selected functional or canary pins only if their exact changed files have been proven and authorized.

```bash
make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
go test -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...
make umpire-check-cases
make umpire-check-fixtures
make lint-model
make lint-code-fast
```
## Acceptance
- [ ] The comparison is anchored to `4755faca73354e2ab169d1a53e3e0e9ad0cf2bfd` with source/artifact hashes and an explicit fn-151 position/owner/identity ledger. The original Activity subset accounts for three Models, 168 receipts and 154 Query occurrences; new derived-machine receipts are explicitly enumerated.
- [ ] Every original Query occurs once in its original artifact placement, with unchanged Property meaning, Scenario, form, bounds, totals, expectations, exploration and answer. The `completes` ownership mapping is intentionally one-to-three, and both competing-timer Queries retain record-IR placement, `no-realization` standing and no expectation.
- [ ] Original complete tables and new subjects' inherited tables, starts/ends, evidence, assumptions, monitors, phase projection and refinement agree. Exact owner-qualified declaration inventory, independent paired Product/System validation and causal pause hold/unpause/release checks retain their assertions.
- [ ] Negative comparison specimens reject omitted Queries/roots, weakened Properties, lost refinements/monitors, changed bounds/totals/expectations and changed realization controller/evidence declarations. No broad normalization hides an unlisted delta.
- [ ] Regenerated Activity IR, Cases, manifest standings and mapped Definition/Model/Query/realization/Case identities, dependent Program/Contract references and checksums are reviewed together. Source-position goldens change only where required by actual deltas; unchanged Nexus mirrors/canary pins and all historical recordings, receipts and replay companions retain their bytes. Any affected selected mirror has concrete-file root authorization before regeneration.
- [ ] Every listed production, Case, fixture, lint and canonical Go command has an actual recorded result. The covering Go suite runs once with `test_dep`, `-p 2`, `-timeout 30m` and JSON evidence. Production generation, complete pinned native/Case equivalence with negative controls, both lints and exact corrected consumers pass. Record the corrected completion guard mutation's actual result. Preserve canonical RED and all killed/unexecuted selections; separately name Batch 5 semantic debt, fn-154 Quint debt and the explicit fn-157 resource deferral. Canary failures receive evidence-backed affected-package classification; any unexplained new semantic/identity/assertion failure still blocks review and completion. Strict assertions and complete domains remain unchanged; no focused success replaces the covering RED.
- [ ] Final subject filenames, roots, realization attachments, regression consumers and layout documentation match the joined split delivered by tasks .1/.2. Any missing concrete source/consumer/doc update receives root's scope authorization before task .3 edits it. Evidence records exact commands, exit statuses, source hashes and all authorized mappings; no filtered success is presented as a full-suite pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
