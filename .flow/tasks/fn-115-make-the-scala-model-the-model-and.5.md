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
- Implement the closed fixture comparator only for oracles/job/{once,retried,closed}/table.json: require lossless original table decode/re-encode, exact original/approved actual row-key vectors and full per-key row equality after the exact two-field empty-list projection below, then permute only the expected Rows in memory before whole-inventory comparison. Keep every other byte/order strict and snapshots untouched; reject other orders, missing/duplicate/unknown keys and changed row contents, preserving inactive-artifact sensitivity through the shared comparator.
- For the synthetic job test-fixture transfer only, enforce the audit’s exact original/reader row sequences and index permutation `0,1,2,3,5,6,7,4`, then compare rows by key. Keep the original task-2 baseline and all production ordering strict; result/fact ordering, witnesses, Query answers, full Cases and fingerprints must remain identical. The only additional serialization projection is waiting-settle.results[0].facts and running-check.results[0].facts in those three tables: require exactly one result per named row, original nil and actual non-nil empty facts; project those expected fields to [] in memory. Reader result construction already initializes empty facts. Reject nonempty facts, unexpected nil/empty conversions elsewhere, result/fact reordering or any other difference; do not change the interpreter or snapshots.
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
