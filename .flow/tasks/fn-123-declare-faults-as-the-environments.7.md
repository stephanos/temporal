---
satisfies: [R7, R8]
---
# fn-123-declare-faults-as-the-environments.7 Trace output and Quint export of derived crashes and budget fields

## Description
Trace output and the Quint export read the fault metadata (R7, R8). Traces mark each budget field with its fault and show, for a derived crash, each changed field with its classification. Quint writes the derived crash as an action that applies the field map and keeps the budget field in its state variable. No IR change, so this runs beside task 6.

**Size:** M
**Files:** `tools/umpire/explore/trace.go` (+ `trace_test.go`), `tools/umpire/export/quint.go` (+ tests), `tools/umpire/export/README.md` ("What the exporters refuse")
**Touches:** [tools/umpire/explore/**, tools/umpire/export/**]
**Order:** After task 5. Parallel candidate with task 6: disjoint files and no regeneration.

### Approach
- Trace: `RenderTrace` (`trace.go:36`) and the witness view (`:109`) render steps. Annotate a budget field with its fault, and for a derived-crash step list each changed field with its classification. Extend `TestTraceRenderingIsStableAndSourceLinked` (`trace_test.go:11`) with a derived crash and a budgeted fault.
- Quint: the exporter refuses whole IR files with channels (`quint.go:83-84`). The converted machines' IR files carry no channels today, so they reach the per-action path. Emit the derived crash as an action whose effect applies the field map. An unsupported fault construct is an `UnsupportedError` with its Scala position (`slice.go:116-125`, built at `quint.go:144`), and nothing is exported around it.
- Run `make umpire-check-backends` locally on the converted machines (agreement check, `export/agreement.go:26`) and record the result. It is not a CI gate (spec Open Questions).

### Investigation targets
**Required** (read before coding):
- `tools/umpire/explore/trace.go:36-140` - trace rendering
- `tools/umpire/export/quint.go:70-150, 700-720` - refusals and action emission
- `tools/umpire/export/slice.go:116-125` - `UnsupportedError`

**Optional** (reference as needed):
- `tools/umpire/export/agreement.go:26` - `QuintAgreement`
- `tools/umpire/export/README.md` - refusal list

## Acceptance
- [ ] A trace through a budgeted fault marks the budget field with its fault; a trace through a derived crash lists each changed field and its classification. The trace golden test covers both.
- [ ] The Quint export of the converted queue and `LostStartAnswer` carries the derived crash and the budget field. An unsupported fault construct is an `UnsupportedError` with its Scala position, with a test.
- [ ] The Quint agreement check passes on every converted machine the exporter does not already refuse; the done summary records the local `make umpire-check-backends` result.
- [ ] `go test -tags test_dep ./tools/umpire/explore/... ./tools/umpire/export/...` and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
