---
satisfies: [R1, R2, R5]
---
# fn-149-safety-and-liveness-groups-for-object.4 Migrate Model claims and capability references with behavior pins

## Description
Migrate Model claims and capability references with behavior pins. Advances R1, R2, R5 of the parent spec.

**Size:** M
**Files:** `model/temporal/**`, `model/irgen/Capabilities.scala`, `model/irgen/Structure.scala`, `model/irgen/testdata/**`, `tools/umpire/check/*test.go`; complete generated IR/Cases remain isolated scratch seal outputs until fn-123.8.
**Touches:** [model/temporal/**, model/irgen/Capabilities.scala, model/irgen/Structure.scala, model/irgen/testdata/**, tools/umpire/check/*test.go, .flow/tmp/fn149/task4/**]

### Approach
- Freeze fn-140.6's independent witness seal as the grouping baseline. Inventory all current Activity subject/Dispatch owners and every Nexus declaration, including capabilities, inherited claims, helper bundles, derived designs and compositions. Classify by declaration kind, not names such as completes.
- Migrate all live authored properties to groups, update references/export selections and activate flat-layout rejection. Keep monitors in their existing declaration/attachment sections and group only their claim references.
- Consume fn-140's actually committed witness/`when`/`.live` vocabulary; do not recreate extracted witness-only Properties. Current package/schema moves and Activity split are baseline. Re-anchor again after fn-155/fn-156 source changes without guessing future APIs.
- Use the independent nexus_close_baseline and declaration/Model pin helpers for a complete before/after grouping comparison of tables, predicates on every applicable row, Query/progress receipts, bounds/assumptions, expected and live/replay assessment meaning, and canonical Cases. Preserve negative controls and all focused owners. Account individually for qualified names, Definition/Case identities, checksums and source spans; compose authorized fn-140/fn-155 mappings without broad normalization. Seeded omission, predicate, bound, receipt and Case deltas must fail the seal.
- Freeze a declaration-side capability-law inventory independently of both candidate scanners. If grouped definitions change discovery, update the lifter's companion scan and runtime adoption/registration tests together, preserving expected counts, monitor attachments, origins and per-instance waiver/reference resolution. A law omitted by both changed scans must still fail the frozen inventory oracle.
- Prepare an isolated scratch checkout/copy containing the candidate source state; from that scratch repository root run the existing `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` under an actual `fcntl`/`flock` lock on `/tmp/umpire-heavy-gates.lock`. The gate has no scratch-output flag. Retain its complete IR/Case/fixture comparison and exact pins as the migration seal; do not publish managed artifacts here. Complete proof selectors require nonzero expected/matched counts and preserve incomplete results, finite-bound counterexamples and independently established safety violations. Task .5 seals its later source/layout changes before fault work, then fn-123.8 publishes and owns full shared gates.

### Investigation targets
**Required:**
- `model/temporal/features/nexus/workflow/system/ClosePolicy.scala:617` - mixed claims and per-design bundles.
- `model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala:113`, `model/temporal/features/activity/standalone/system/DispatchWithWorker.scala:31` - current composition safety; `model/temporal/features/activity/standalone/system/System.scala:533` is lifecycle claims.
- `model/irgen/Capabilities.scala:418`, `model/temporal/capabilities/Properties.test.scala:124` - separate lifted/runtime shared-law inventories.
- `tools/umpire/check/nexus_close_baseline_test.go:256` - independent inventory, predicate and progress baseline.
- `tools/umpire/check/activity_parity_test.go:178` - current split Activity source/export inventory.
- `model/irgen/Structure.scala` - final layout refusal.

### Key context
Conditional Batch 2 entry follows the committed fn-140.6 witness seal. Re-anchor to the actual fn-155 mapping, fn-156 enforcement and fn-140 vocabulary; future APIs remain unknown here. Fn-141 comes later. Root owns placement/source gates; .5's final grouping seal precedes fn-123.1, and closure waits fn-123.8. Shared heavy work uses the real `/tmp/umpire-heavy-gates.lock`.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check -run 'Test.*(Close|Activity|Capabilities|Declarations|Composed)'
```

## Acceptance
- [ ] All authored Models and shared law references use the final grouping, with a refusal fixture for the retired flat form.
- [ ] Complete independent scratch seal preserves tables, predicates, Query/progress verdicts and Case/assessment meaning, including negative controls; seeded semantic/population/artifact deltas fail and publication remains deferred to fn-123.8.
- [ ] Every changed generated identity/provenance span has a declared cause; monitor attachment and independently expected shared-law counts are unchanged, and a law dropped by both scanners fails the frozen oracle. Complete selectors have nonzero matched counts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
