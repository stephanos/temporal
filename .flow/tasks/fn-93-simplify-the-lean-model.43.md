---
satisfies: [R12, R13]
---
# fn-93-simplify-the-lean-model.43 Final measurement, full gate run and fn-60 supersession

## Description
Close the campaign. Measure with the spec's command and split (generated / test / production) against both baselines (R12), run every gate in R13, fix the inherited `--wfail` warnings whose code survived, move fn-60 to `superseded` in `.plans/index.json` and in `.plans/UMPIRE4_ORDER.md`'s deferred table, and update fn-93's ORDER entry and index status per `tools/planindex` rules (read `tools/planindex/check.go:549-557` for what `superseded` and the closing disposition require).

**Size:** S
**Files:** `.plans/index.json` (fn-60 and fn-93 entries only), `.plans/UMPIRE4_ORDER.md` (fn-93 entry, fn-60 deferred row), Lean files carrying surviving `--wfail` warnings
**Touches:** [.plans/index.json, .plans/UMPIRE4_ORDER.md, model/**/*.lean]

### Approach
- The start baseline is the one the first fn-93 task's receipt recorded; report both, and each floor met or missed with the reason (never waived silently).
- `LEAN_NUM_THREADS=1 make lint-model` for the lint driver on a 16 GB machine.

### Quick commands
```sh
cd model && find . -path ./.lake -prune -o -name '*.lean' -print | xargs wc -l | tail -1
make umpire-check-regression umpire-check-retired-vocabulary umpire-check-plan-index lint-code-fast
LEAN_NUM_THREADS=1 make lint-model
go run ./tools/planindex
```

### Gate policy (2026-09-29)
- Tasks .1–.42 run scoped gates: the focused Lean build, the byte-identity checks their surface affects (goldens, conformance, canary Case), and a scoped `make lint-model-builtin`; they skip the live tests and the full regression bundle unless they touch Go, Testpilot, Case production or a live-test input. This closing task runs the whole-model `LEAN_NUM_THREADS=1 make lint-model` and the full `make umpire-check-regression` once for the spec, on a quiet host.

## Acceptance
- [ ] Receipt reports the measurement split against both baselines with each R12 floor met or explained
- [ ] Every R13 gate exits 0; inherited `--wfail` warnings on surviving code fixed
- [ ] fn-60 superseded in index and ORDER; fn-93's entries updated; `go run ./tools/planindex` green


## Done summary
Blocked:
Won't do (2026-10-01): the Lean model is retired in favour of the Scala front end (model/scalav2), and the Lean toolchain is removed. Spec closed as won't-do by the owner.
## Evidence
- Commits:
- Tests:
- PRs:
