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
Relocated the protocol generator, shared schemas and output table into the parent module. Default protocolgen builds without native selection and names stale files with the required exit statuses. Retained zero relocation-only body diff; final generated body changes are restricted to the two choice identity literals explicitly authorized in 72b3ef4cb7 and accounted for by fn-219.4. Corrected alternate-replacement generation to read shared inputs from the replaced parent while writing native files and reading native-owned inputs under the selected checkout. Regression covers all outputs, named stale detection and absence of replacement-tree writes. Fresh protocol CLI/library/native tests, native protocol-generate -check, make generate validate and selected make generate-native validate-native exit 0. Independent source review SHIP. Evidence: .flow/artifacts/fn219-acceptance-audit-2026-10-20.md and raw review-*-validation.log/review-output-root-tests.log in /home/agent/wt/evidence/fn219/. Full selected archcheck/gomadtool baseline failures are retained separately, not represented as passing.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: aa43211daf, ab052d04e9, b6a8296c1f, 5f9e92af84, 8aca21a9df, cbfd5f75a3, a84a807829, 38b4c5c6f2
- Tests: go test -tags test_dep -count=1 ./internal/model/volume/... ./internal/model/network/... ./internal/model/generation/protocol/... ./internal/model/cmd/protocolgen/... ./internal/archcheck (exit 0), make generate validate (exit 0), GOFLAGS='-tags=gomad_native -p=2' make generate-native validate-native (exit 0), go run -tags gomad_native go.temporal.io/server/tools/gomad/internal/backends/native/cmd/gomadtool protocol-generate -check (exit 0), required-helper TestNamespaceConformance and TestVolumeConformance (exit 0), cargo +1.99.1 test --release --locked --test network_corpus --test world_conformance --test clock_conformance (exit 0), integrated go test -p=1 -tags test_dep ./cmd/tools/lintcode (exit 0), integrated go test -p=1 -tags test_dep ./internal/archcheck (exit 0; integrated-archcheck-fixed.log), independent source review and budget correction review: SHIP
- PRs: