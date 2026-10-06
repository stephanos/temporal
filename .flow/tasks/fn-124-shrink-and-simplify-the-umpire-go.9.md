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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
