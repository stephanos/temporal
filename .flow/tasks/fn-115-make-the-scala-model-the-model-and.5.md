---
satisfies: [R2, R5, R16, R17, R18]
---
# fn-115-make-the-scala-model-the-model-and.5 Encapsulate the checker behind the reader interface

## Description
Encapsulate the checker behind the reader interface. Implements R2, R5, R16, R17, R18 using the reviewed parent contracts.

**Size:** M
**Files:** model/scalav2/goir public surface and new internal checker; copied producer under the current lowerer; lowering/conformance/export/exploration callers
**Touches:** [model/scalav2/**, model/go/umpire/**, model/go/caseproducer/**, tests/**, tools/umpire/cmd/**, .plans/umpire-migration-*.json]

### Approach
- Omit original checker replay_test.go from the private copy because it imports handwritten Models; preserve its comment and exact own-witness replay/rebound-Action-ID rejection assertions in reader-owned tests over admitted activity/Nexus IR with nonvacuous expected Query inventory. Original source stays frozen and Query.Replay needs no new API.
- Preserve existing ReceiptKind constant names/values where checker Outcome names collide. Alias the required Outcome type and explicitly convert same-valued receipt constants at outcome comparisons; verify equality with private checker constants, without merging domains or adding a synonym constant family. A shared job IR builder may use the existing test-only internal/golden package with schema-only dependencies.
- Apply only the exact synthetic-job representation projection recorded in UMPIRE_MODULES.md to oracles/job/{once,retried,closed}/{table,ids,case,identity}.json. Reader classes sorts action keys, disproving the earlier declaration-order row assumption. Require exact original/reader Actions, Rows and Reachable vectors; lossless decode/re-encode and full row equality except the two named nil-to-empty facts fields with exact cardinality checks. IDs follow the exact action permutation. Keep interpreter/output tables/snapshots unchanged.
- Validate the derived Query fingerprint chain: replacing only the admitted canonical Query action-domain fingerprint with the exact original action-domain fingerprint must reproduce the frozen Query provenance fingerprint. Equivalent reconstruction from frozen scenario/property/table inputs with Table.PropertySemantic and an original Query fingerprint check is allowed. Only then project that unique Query provenance fingerprint in the losslessly decoded expected Case and recompute its canonical/runtime identities, first validating original identities. No copying actual Case/identity into expected data. Program/Contract/answers/witnesses/property/scenario/target fingerprints and all other bytes remain strict. Reject additional order/content/cardinality/canonical/provenance/hash differences and preserve whole-inventory inactive-artifact sensitivity. Any additional difference stops transfer.
- Follow `test_claim_transfers` for copied producer tests: retain all ten occurrence/projection/fingerprint claims through equivalent admitted generic-job IR fixtures and the two keyed behavioral claims in same-package lowerer tests. Keep assertions/comments and exact evidence-source refusal behavior; do not expose NewTable/TableSpec/KeyProperty/KeyScenario/KeyFind solely for tests. Public Row remains required data in the table signature closure.
- Use the reviewed map to establish the reader-facing vocabulary while current paths still compile. Copy only the live checker implementation behind the reader's internal directory, preserving the original archive source.
- Before switching lowering to the copied checker’s types, copy the live producer into `model/scalav2/goir/testpilot/internal/producer` and change that copy to use the reader facade. The frozen original `model/go/caseproducer` continues using its original checker; never pass a copied-checker Query to that original producer. Task 6 relocates this coherent live producer/lowerer together.
- Make loading/checking and result types available through the reader's intentional public surface; use type aliases only where they preserve one declaration without exposing a second caller-owned package. Do not create forwarding-only packages.
- Move lowering, conformance, export and exploration callers away from direct checker imports. Keep lower's actual producer dependency separate from reader semantics and preserve no-Testpilot imports in the reader.
- Exercise real downstream callers and test-only dependency graphs. Leave concern splits and unused-declaration cleanup to task 7; this task establishes a coherent compilable interface.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/goir/checking.go:16`
- `model/scalav2/goir/machine.go:180`
- `model/scalav2/goir/claims.go:29`
- `model/scalav2/goir/testpilot/lower.go`
- `model/scalav2/goir/conformance`
- `model/go/umpire`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/...; mise exec -- go list -tags test_dep -deps -test ./model/scalav2/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] The live producer copy and copied checker share the reader’s public types before relocation; original producer/checker bytes remain frozen.
- [ ] All live semantic callers use the reader's reviewed interface; internal checker import ownership is enforceable.
- [ ] Reader remains independent of Testpilot, with no facade-only helper package added.
- [ ] Original archive material and complete semantic/artifact goldens remain unchanged.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: