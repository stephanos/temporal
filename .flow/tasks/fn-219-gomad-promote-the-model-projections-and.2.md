---
satisfies: [R3, R4]
---
# fn-219-gomad-promote-the-model-projections-and.2 Relocate the protocol generator and I/O wire schema into the parent module

## Description
Move internal/backends/native/transcript/schema/{iowire.json,iowire.go.tmpl,iowire_test.go.tmpl} to internal/model/schema and internal/backends/native/generation/protocol to internal/model/generation/protocol with a thin internal/model/cmd/protocolgen main (generate and -check, exit 0/1/2 per the spec). Split the output table into the shared set (written by protocolgen) and the native set (written by native gomadtool protocol-generate, which imports the shared package under gomad_native and resolves schema paths from the parent root via the module replace, not a hard-coded cwd). Change shared headers to name internal/model/cmd/protocolgen. Diff every shared output before and after ignoring header lines and record a zero-length diff.

**Size:** M
**Files:** tools/gomad/internal/model/schema/**, tools/gomad/internal/model/generation/protocol/**, tools/gomad/internal/model/cmd/protocolgen/**, tools/gomad/internal/backends/native/cmd/gomadtool/protocol.go, tools/gomad/internal/backends/native/generation/** (removal), generated outputs listed in the spec

## Acceptance
- [ ] go run ./internal/model/cmd/protocolgen -check exits 0 with stock Go and no tags
- [ ] Every shared output body is byte-identical to the previous output ignoring header lines; the diff record is retained
- [ ] Native gomadtool protocol-generate -check exits 0 under gomad_native and writes only the native set
- [ ] -check reports a named stale file when a shared output is edited

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
