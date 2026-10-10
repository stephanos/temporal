---
satisfies: [R1, R4, R5]
---
# fn-153-checked-property-examples-pilot.4 Generate compact checked documentation through the model gate

## Description
Render checked inputs and explanations as one compact Markdown document. Wire its freshness and classification checks into the existing gate. The production pilot declarations arrive in .5.

**Size:** M
**Files:** new `tools/umpire/cmd/umpire-illustrations/main.go` and `main_test.go`, `model/check/Gate.scala`, command Make wiring if needed, `.plans/UMPIRE_MODULES.md` command ownership
**Touches:** [tools/umpire/cmd/umpire-illustrations/**, model/check/Gate.scala, Makefile, .plans/UMPIRE_MODULES.md]

### Approach
- Introduce only the narrow entry point needed by the requested gate/document flow. It reads lifted IR through the reader, runs .3's checker, and writes deterministic Markdown to stdout. Reuse existing command test patterns; no generic renderer subsystem, new config surface, server or visualization dependency.
- Render each declared illustration beside its Property, with concrete action/result/before-state, explanation, expected/actual result and visibly unclaimed reachability. Use stable Model/Property/declaration ordering; escape author text so labels and explanations cannot create misleading Markdown structure. Include the separately derived unrelated-action classification in the applicability explanation without inventing an authored expectation.
- Fail classification before publishing a document. Bind generation to the current transient lifted IR, not last generation's checked-in tree. Gate.settle at `Gate.scala:559` compares/installs the single managed `model/docs/property-illustrations.md` outside the managed IR tree. The command writes no path under model; only the gate manages this output.
- Add small document/command goldens and missing, stale, orphan, reordering, broken-association and classifier-error checks. A predicate change must recompute and fail stale classifications even if the old document is present. Preserve a prior document on a failed update rather than making it appear freshly successful.
- Describe the command's narrow ownership and permitted imports in the module map. Defer user-facing author documentation and pilot assessment to .6.

### Investigation targets
**Required:**
- `model/check/Gate.scala:402` - non-skippable IR checks
- `model/check/Gate.scala:522` - transient lifted tree and update flow
- `model/check/Gate.scala:559` - deterministic managed output comparison
- `tools/umpire/cmd/umpire-lint/main.go` - reader/check command edge pattern
- `.plans/UMPIRE_MODULES.md:25` - module and command ownership
**Optional:**
- `tools/umpire/internal/cli` - common command output/no-model-write policy
- `Makefile:764` - existing model command/gate wiring

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/cmd/umpire-illustrations
make lint-model
```
Also run the gate's focused document-settle fixtures without an unnecessary full production regeneration.

## Acceptance
- [ ] The stdout-only entry point uses .3's checker and emits escaped deterministic checked documentation, including the not-applicable explanation and visible reachability disclaimer.
- [ ] The gate consumes current transient IR, rejects stale/missing/orphan output, and owns the one managed document outside model/ir.
- [ ] Located mismatches/errors fail before output publication; an old successful document cannot bypass classification or appear freshly updated.
- [ ] Command/golden and focused gate freshness negatives pass; module ownership and imports remain enforced without broader viewer infrastructure.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
