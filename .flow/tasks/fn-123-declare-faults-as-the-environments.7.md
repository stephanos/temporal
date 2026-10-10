---
satisfies: [R7, R8]
---
# fn-123-declare-faults-as-the-environments.7 Trace output and Quint export of derived crashes and budget fields

## Description
Trace output and the Quint export read the fault metadata (R7, R8). Traces mark each budget field with its fault and show, for a derived crash, each changed field with its classification. Quint writes the derived crash as an action that applies the field map and keeps the budget field in its state variable. No IR change, so this runs beside task 6.

**Size:** M
**Files:** `tools/umpire/explore/trace.go` (+ `trace_test.go`) and the existing exploration candidate/command trace entry seam, `tools/umpire/export/quint.go` (+ tests), `tools/umpire/export/README.md` ("What the exporters refuse")
**Touches:** [tools/umpire/explore/**, tools/umpire/export/**]
**Order:** After task 5. Parallel candidate with task 6: disjoint files and no regeneration.

### Approach
- Trace: separate rendering a checked model witness from requiring a lowered Case. Add a checked Model/Query/receipt entry path, verify that its witness belongs to those checked inputs, and render its steps without a realization or invented Case/Run. The existing executable entry delegates to it; when execution evidence is supplied, retain exact Case/program/Run identity checks, require a real Case, and refuse mismatched identity. Audit the existing candidate/command seam so a converted model-only queue crash actually reaches rendering before lowering refusal. Case KnownGaps are absent on model-only traces, not fabricated. Annotate a budget field with its fault, and for a derived-crash step list each changed field with its classification. Extend `TestTraceRenderingIsStableAndSourceLinked` (`trace_test.go:11`) with a real converted model-only queue-crash witness carrying no Case/Run and a budgeted fault; check changed fields/classification, source links and model-only presentation. Keep executable trace goldens and mismatched Case/program/Run refusal tests.
- Quint: the exporter refuses whole IR files with channels (`quint.go:83-84`). The converted machines' IR files carry no channels today, so they reach the per-action path. Emit the derived crash as an action whose effect applies the field map. An unsupported fault construct is an `UnsupportedError` with its Scala position (`slice.go:116-125`, built at `quint.go:144`), and nothing is exported around it.
- Run focused scratch export and trace proofs here. Fn-123.8 owns the single complete local `make umpire-check-backends` on every converted machine the exporter does not already refuse (agreement check, `export/agreement.go:26`); link its result here, retaining explicit unsupported receipts. It is not a CI gate (spec Open Questions).

### Investigation targets
**Required** (read before coding):
- `tools/umpire/explore/trace.go:36-140` - trace rendering
- `tools/umpire/export/quint.go:70-150, 700-720` - refusals and action emission
- `tools/umpire/export/slice.go:116-125` - `UnsupportedError`

**Optional** (reference as needed):
- `tools/umpire/export/agreement.go:26` - `QuintAgreement`
- `tools/umpire/export/README.md` - refusal list

## Acceptance
- [ ] A trace through a budgeted fault marks the budget field with its fault; a trace through a derived crash lists each changed field and its classification. The trace golden test covers both through the checked-model-witness path, including a converted model-only queue crash without a fabricated Case; executable traces retain their exact identity guards.
- [ ] The Quint export of the converted queue and `LostStartAnswer` carries the derived crash and the budget field. An unsupported fault construct is an `UnsupportedError` with its Scala position, with a test.
- [ ] The Quint agreement check passes on every converted machine the exporter does not already refuse; the done summary links the single local `make umpire-check-backends` result owned by fn-123.8.
- [ ] `go test -tags test_dep ./tools/umpire/explore/... ./tools/umpire/export/...` and `make lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
