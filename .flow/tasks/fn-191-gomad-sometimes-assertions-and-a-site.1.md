---
satisfies: [R1, R2]
---
# fn-191-gomad-sometimes-assertions-and-a-site.1 Sometimes/Unreachable in common/softassert and the retained catalog generator

## Description
Go only. Add softassert.Sometimes (no-op in ordinary builds; gomad-tagged build sets the catalog bit through the gomaddebug init seam) and softassert.Unreachable (delegates to Fail). Write the generator under tools/gomad/internal/backends/wasm/toolchain/cmd/catalog on go/packages: scan That/Fail/Sometimes/Unreachable/UnexpectedInternalErr/UnexpectedDataLoss and static tracer.Start span names; emit common/softassert/zz_catalog_gomad.go under gomad && wasip1 && wasm with stable IDs, kind, module-relative path, line, message; refusals per API Contracts; build-time binding via the generator's source overlay transformation preserving evaluation order. Add the stale-catalog check to make validate. Convert at least one existing site. Sequencing: whole spec after fn-240, fn-242 and fn-245.2 (spec-level deps).

## Acceptance
Ordinary build: Sometimes has no observable effect, Unreachable behaves as Fail; generator output is deterministic and checked in; validate fails on a stale catalog; each refusal case has a test naming the location; one converted site demonstrated.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
