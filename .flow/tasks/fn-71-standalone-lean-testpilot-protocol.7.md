---
satisfies: [R8]
---
# fn-71-standalone-lean-testpilot-protocol.7 Retire parallel ownership and verify the Testpilot cutover

## Description
Finish the cutover by removing parallel handwritten protocol and serializer ownership, reconciling active documentation, and running the complete migration gates. Compatibility names may remain only as thin aliases or forwarders to generated types and the sole library codec with a stated removal point.

**Size:** M
**Files:** `model/Umpire/Case/{Value,Program,Run,Contract,ProtoJSON}.lean`, `model/Umpire/Case.lean`, `model/Temporal/Testpilot/TestpilotProtoJSON.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tests/testcore/testpilot/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`, affected public module docstrings and import tests
**Touches:** [`model/Umpire/Case/{Value,Program,Run,Contract,ProtoJSON}.lean`, `model/Umpire/Case.lean`, `model/Temporal/Testpilot/TestpilotProtoJSON.lean`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`, `model/README.md`, `tests/testcore/testpilot/README.md`, `.plans/UMPIRE4_SPEC.md`, `.plans/UMPIRE4_COMPONENTS.md`, affected public Lean docstrings/import tests]

### Approach
- Remove handwritten Testpilot wire-shaped structures and independent Umpire/Temporal serializer bodies after every producer has migrated.
- Retain old module names only where source compatibility materially requires aliases/forwarders; cover equivalent output and document their removal point.
- Update active architecture, component, README, and public module documentation to name the `.proto` schema, generated protocol, authoring facade, `Protobuf.Json`, Umpire provenance, and Go Prepare owners.
- Leave historical design records, unrelated generated catalogs, protobuf schemas, and broader CI expansion untouched.
- Run focused protocol/generation/import/fixture/Go checks, then the full model build, `make lint-model`, and `make lint-code GOLANGCI_LINT_FIX=false`; fix introduced failures and evidence any inherited baseline failure.

### Investigation targets
**Required** (read before coding):
- all migrated compatibility modules
- `model/Umpire/ARCHITECTURE.md`
- `model/ARCHITECTURE.md`
- `model/README.md`
- `tests/testcore/testpilot/README.md`
- `.plans/UMPIRE4_SPEC.md`
- `.plans/UMPIRE4_COMPONENTS.md`

**Optional** (reference as needed):
- `.flow/memory/declined/generated-api-drift-verification.md`
## Acceptance
- [ ] No parallel handwritten Testpilot wire type or independent Umpire/Temporal field serializer remains.
- [ ] Any retained compatibility surface delegates directly to generated types or `Testpilot.ProtoJSON` and has equivalent-output coverage plus a removal point.
- [ ] Active documentation consistently states final schema, protocol, authoring, codec, provenance, and Go admission ownership.
- [ ] Historical/generated/out-of-scope artifacts remain untouched unless an owned generator legitimately changes their output.
- [ ] Focused and full specified gates are recorded with no new proof placeholders or trust dependencies; introduced failures are fixed and inherited failures have concrete baseline evidence.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

