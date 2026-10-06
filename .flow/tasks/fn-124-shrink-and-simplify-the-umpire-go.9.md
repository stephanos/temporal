---
satisfies: [R9]
---
# fn-124-shrink-and-simplify-the-umpire-go.9 Retire the P export

## Description
Implements R9 (owner decision 2026-10-05): retire the P export.

The P second opinion (`tools/umpire/export/p.go`, 854 lines, plus tests and testdata) checks one monitor (`terminalFinality`) on two machines for traces of up to five steps, and it brings in the .NET SDK through the backend install. The Quint export recomputes every table and Property reading independently, and Apalache bounded-checks monitors, so P adds little.

Steps:
1. Confirm the Quint side covers what P checked: the agreement on those machines' monitor products, or a `quint verify` check of `terminalFinality`. If not, add that check to the Quint export first, in its own commit, and show it would catch a seeded violation.
2. Remove the P exporter, its receipts and kinds, its tests and testdata, the Makefile targets and the P/.NET parts of `umpire-install-backends` and `umpire-check-backends`.
3. Update `tools/umpire/export/README.md`, `.plans/UMPIRE_MODULES.md`, `model/README.md`, `model/SEMANTICS.md` and the ownership test, so nothing names P as a backend.
4. Report lines removed and the backend install's size and time before and after.

Ordering: after fn-126 closes (fn-126.7 edits `p.go` for the `party` → `actor` field) and before fn-124.8, so the package split has less to move. Not in this task: switching the Quint verify backend to TLC.

**Touches:** tools/umpire/**, Makefile, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, MILESTONES.md

Closing .9/.8 validation batch (owner's validation-reuse instruction): before removing P, execute the Quint coverage proof for the two P machines. Measure install size and time on equivalent cache/input conditions before and after; do not claim a cold-install speedup from a warm-cache run. After removal, run the focused export tests, ownership/path checks, read-only task-base Go lint, the live-reference scan and Case byte checks. Commit coherent checkpoints and obtain per-task SHIP. The shared full model gate, instrumented full Go tooling suite, Testpilot tests, full batch-base Go lint and Case/fixture/canary checks run once at .8's closing boundary, along with the required Quint-only backend gate after install. Record that full-suite obligation as deferred to .8 in .9's done evidence; never mint full-gate receipts from focused commands. Neither spec nor milestone closes until both tasks and that full boundary are verified.
## Acceptance
- [ ] Before removal, the Quint side is shown to check what P checked (`terminalFinality` on its two machines), by existing agreement or a new Quint check that fails on a seeded violation.
- [ ] The P exporter, its tests and testdata, receipts, Makefile targets and the P/.NET backend install are gone; `grep -rni '\bP export\|p\.go\|dotnet\|\.pproj' tools/umpire Makefile model .plans/UMPIRE_MODULES.md` finds no live reference.
- [ ] Docs and the ownership test name only the Quint backend; the Quint export and its agreement are unchanged.
- [ ] The full Go suite, lint-code-fast and `make umpire-check-backends` (Quint only, run locally after `make umpire-install-backends`) pass; the summary reports lines removed and the install's size and time before and after.


## Done summary
Quint is now Umpire's sole export backend. Removed the P exporter, tests, runner, pins, environment wiring and trace-only receipt fields, together with the P/.NET install steps. The retained shared literal decoder moved unchanged from the deleted exporter into slice.go. Updated the export/tooling guides, Model guides and module map, and added the ownership guard.

Before removal, the actual Quint evaluator agreement covered terminalFinality on both P machines. activityRecord compared all 31 reachable product steps over 11 product states, with no violation. trustingActivityRecord compared all 111 steps over 30 product states and replayed its terminalFinality counterexample through a fresh Go interpretation and the reader's checker. A closed product compared step by step covers every finite path, including P's former five-step traces. Existing TestQuintDisagreesOnAnotherModel/a_monitor's_notion_of_over_narrowed caught the seeded exported-monitor mismatch. Logs, exports and receipts are retained under .flow/tmp/fn-124/task9/quint-proof.log and quint-proof/. No added Quint check was necessary; Quint translation, agreement, verify implementation, versions and backend selection are byte-unchanged.

Removed 1,203 lines and added 76, a net reduction of 1,127 across the implementation and named docs. The tracked export tree had no P testdata or persisted receipt files to remove. The retirement guard failed before deletion because the exporter existed, then passed after deletion. The first post-deletion compilation identified the shared literal decoder; preserving its existing implementation restored the build. The guard uses whole-token matching so UMPIRE_PINNED_RUNS is not mistaken for a retired tool selector.

baseline: green. Pre-edit export and focused ownership/layout suites, Case byte checking and read-only task-base lint passed. The full fn126.13 evidence was inspected (17 packages, 2,866 tests/subtests, exit 0, 108 wall seconds) but no full baseline handoff or full-suite receipt reuse is claimed.

Focused verification passed: go test -count=1 -tags test_dep ./tools/umpire/export; focused ownership/layout checks including TestQuintOwnsBackendExport; make umpire-check-cases; GOLANGCI_LINT_BASE_REV=a912c3a89b85ccd2ce2faa671ee82c168a3f4fef GOLANGCI_LINT_FIX=false make lint-code-fast (zero findings). Logs are under .flow/tmp/fn-124/task9/. A word-boundary live scan finds no P backend reference in tools/umpire, Makefile, model or .plans/UMPIRE_MODULES.md; the literal substring p.go would also match unrelated group.Go. The artifact diff and retained Quint source diff are empty.

Installation measurement used the same already-installed warm tool/cache inputs, without deleting any cache. TIMEFORMAT timing of make umpire-install-backends was 0.799 wall seconds before and 10.943 afterward, retained in install-before.log and install-after.log. The warm target became slower because it now starts pinned Quint through npm; no cold-install timing or speedup is claimed. The attributable required local install footprint drops from 629,020 KiB (617,700 SDK plus 11,320 P, 614.3 MiB total) to zero. Quint's existing npm cache and Apalache's first-verify download remain common requirements. Existing P/.NET cache directories were retained, so this is a reduction in required installation footprint, not reclaimed workspace disk. Cold install timings and an Apalache download were not measured.

DEFERRED_TO: fn-124-shrink-and-simplify-the-umpire-go.8. The amended task/spec closing batch assigns the full model gate with MODEL_GATE_ARGS=--skip-go-checks; instrumented full Go tooling suite (go test -json -count=1 -tags test_dep -p 2 -timeout 30m ./tools/umpire/...); Testpilot/canary tests; full batch-base Go lint; Case/fixture/canary checks and artifact smoke; and make umpire-check-backends after make umpire-install-backends to .8's closing boundary. gate classify returned FULL; this deferral is explicit owner/spec policy, not tier-B classification. No full-gate receipt was minted from a focused command. R9's final full-suite/backend obligation remains pending until .8 discharges it. The parent spec remains open.

Tier: session (jev-unavailable(no_key))
Review uses the explicitly requested codex:gpt-6.1-sol:high in a fresh read-only context. Reviewer and writer are from the same model family.
stage: impl-review - ran (SHIP; codex:gpt-6.1-sol:high; a912c3a89b85ccd2ce2faa671ee82c168a3f4fef..1ee8cae19f562db1010fbb4d7982db3666bc22e8; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-124-shrink-and-simplify-the-umpire-go.9.json). All three lenses reported zero introduced findings and R9 met. Review-side test attempts were blocked by their read-only sandbox; the worker's focused green commands above supply execution evidence.
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)
## Evidence
- Commits: 1ee8cae19f562db1010fbb4d7982db3666bc22e8
- Tests: UMPIRE_BACKENDS=require UMPIRE_BACKENDS_OUT=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-124/task9/quint-proof mise exec -- go test -C tools/umpire/export -v -count=1 -timeout 10m -tags test_dep -run "^(TestQuintAgreesWithGo|TestQuintDisagreesOnAnotherModel)$/^(activity-record|a_monitor.s_notion_of_over_narrowed)$", go test -count=1 -tags test_dep ./tools/umpire/export, go test -count=1 -tags test_dep ./tools/umpire/model -run "Test(QuintOwnsBackendExport|CheckerAndProducerHaveOneLiveOwner|LiveModelDependencyGraph|EveryToolingPackageHasALiveCaller|ModelDependencyGraphRejectsCrossedOwners|RetiredModel.*|ModelFilesLeaveOutTheBuildCaches)$", GOLANGCI_LINT_BASE_REV=a912c3a89b85ccd2ce2faa671ee82c168a3f4fef GOLANGCI_LINT_FIX=false make lint-code-fast, make umpire-check-cases, make umpire-install-backends (equivalent warm input/cache measurement before and after; exit 0 each), git diff --check, DEFERRED_TO: fn-124-shrink-and-simplify-the-umpire-go.8 closing boundary: full model, full Go tooling, Testpilot/canary, batch-base lint, artifact smoke and Quint-only backend gate after install; no full-gate receipt minted
- PRs: