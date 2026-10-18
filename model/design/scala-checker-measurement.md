# Scala checker measurement

Decision: **no-go on deleting the Go check package now**. The native checker supplies the in-process oracle for the measured DSL questions, but refinement/replacement proof coverage and current Go consumers still block package retirement. This note neither releases Phase 3 nor performs deletion. The selected future Case producer is Scala emitting the Testpilot Case proto; implementing it remains a separate migration.

## Workloads and results

Product inputs are pinned at `e466baa9cafcf45db41ccd8f3b572205a90d3aae`, including the separate TwoPhaseCommit predicate simplification (`b53e677090f7b80c25010d196327b79d716ba62a`). Checker lineage is `8f84a96e4942fc4fb028bfd72e87ac78e379d329`. These sources still mix primary class and singleton Models; the later class-only task is outside this measurement. The initial matrix uses harness commit `eb3d074473` and artifact SHA-256 `c069491db416bec3bc52eae61cbc8ad4e517b6bd31f99f72c7530e7dce4f6a42`. Its complete compiler-input manifest is `.flow/tmp/fn1963/public-build3/source.sha256`.

| Owner/world | Budget | Starts | Reached states | Emitted steps | BFS depth | Ordinary graph | verifyAll seconds: first / next / next | Sampled live heap bytes / samples | Claim results |
| --- | ---: | ---: | ---: | ---: | ---: | --- | --- | ---: | --- |
| Kettle | 1,048,576 | 81 | 283 | 424 | 62 | complete | 0.365005 / 0.109971 / 0.111103 | 4,337,560 / 28 | 4 verified |
| Bank | 1,048,576 | 1 | 4,624 | 739,840 | 10 | complete | 82.897949 / 82.548915 / 83.140475 | 6,170,296 / 5,839 | 2 verified |
| TwoPhaseCommit | 1,048,576 | 1 | 288 | 1,025 | 10 | complete | 0.337563 / 0.066299 / 0.084652 | 4,396,464 / 24 | 2 verified |
| ReliableBroadcast | 1,048,576 | 1 | 125 | 266 | 11 | complete | 0.359974 / 0.134506 / 0.099556 | 4,524,944 / 31 | 4 verified |
| TwoLayeredCache | 4,096 | 1 | 7 | 15 | 4 | complete | 0.376972 / 0.064850 / 0.052421 | 5,628,848 / 27 | 2 verified, 2 unknown |
| Voting3, all 3 majorities | 1,048,576 | 1 | 599 | 10,782 | 10 | complete | 0.420373 / 0.156449 / 0.139183 | 4,556,040 / 34 | 2 verified |
| Voting5, all 10 majorities | 1,048,576 | 1 | 28,591 | 857,730 | 16 | complete | 6.267450 / 5.388685 / 4.809197 | 16,743,456 / 408 | 2 verified |
| Voting7, all 35 majorities, initial cut | 65,536 | 1 | 65,536 | 2,752,512 | 7 | cut | 15.431432 / 14.875095 / 14.593374 | 34,660,544 / 1,044 | 2 unknown |

Each row records three separate fresh-JVM modes, not one instrumented run. Statistics drain an ordinary `NativeCheck.Search` with `walk(onEdge = ...)`. Reached states are stored structural states; emitted steps count every yielded transition outcome, including refusals, self-loops, repeated destinations and destinations rejected by the state budget. Disabled class evaluations are excluded. These are base-graph statistics, not counters from the timed verifyAll invocation. BFS depth is not an executed-step count. All these statistics report zero unknown visibility holes; a cut still prevents completeness.

Timing calls public `Checker.verifyAll` through a concrete stable holder. The first invocation in a fresh JVM includes lazy checker initialization and JIT effects. The next two use the same preconstructed instance and reuse instance/global caches; they are not independent cold trials or a steady-state claim. The timer excludes compilation, packaging, JVM startup, construction, initial-state materialization, environment logging and output rendering. No intrusive GC sampler runs in the timing JVM.

Heap uses a separate fresh JVM and one public verifyAll invocation. The sampler checks that the worker stack contains `NativeCheck$.verifyAll` before requesting `System.gc` and after reading a completed SerialGC `MarkSweepCompact` GcInfo. It accepts only observations inside that same PID/invocation with an increased full-GC ID. Bytes sum `memoryUsageAfterGc` for HEAP pools only, captured at the collection safepoint. Current occupancy, allocation, metadata/nonheap and RSS are not substituted for live heap. Raw GC causes/uptime intervals accompany the unique GC IDs; GCInfo IDs and unified-log event IDs are distinct sequences.

The live number is the largest accepted **sampled whole-JVM live heap**, including model, checker, harness and process caches. It is not an exact peak or isolated visited-set size. Stack bracketing does not guarantee coverage of every internal search phase. GC perturbs scheduling, collection, class unloading and JIT. Samples stream immediately, retaining PID/invocation, GC ID, bytes, start/end uptime and successful bracketing; rejected terminal intervals are recorded separately. Initial matrix cadence is a 10 ms sleep after each observation, not a guaranteed 100 Hz sampling rate. All eight heap runs had zero rejected intervals. A zero-sample run is inconclusive and earns no heap value. A post-return collectible working graph is never used as graph live heap.

Host: Linux `dev-build 7.0.14`, aarch64, 12 online CPUs; Scala 3.9.0, scala-cli 1.17.1, Temurin/OpenJDK 27. Runtime flags are `-Xms256m -Xmx2g -XX:+UseSerialGC`; reported maximum heap is 2,147,418,112 bytes. Only one heavy JVM runs at a time. Statistics use the existing private search hook; timing/heap include public stable-holder admission and alter no Model effects.

### Exact worlds and claim coverage

- Bank has four addresses, holder king, supply 2 and both denominations. Sends retain signed amounts -1 through supply+1; reachable balance updates are guard-constrained to -supply through supply. This is not a whole-state type/domain restriction: supplied arbitrary Int states remain admitted. Nor is this the older nonnegative Bank world.
- Voting uses two values and ballots 0/1. Voting3 includes all three two-of-three majorities, Voting5 all ten three-of-five majorities, Voting7 all thirty-five four-of-seven majorities. No unanimity substitution makes the growth experiment smaller.
- TwoPhaseCommit includes all three managers. Commit deadline remains 3, verified longest remains 3 and ordinary graph depth remains 10.
- Kettle includes all 81 declared initial temperatures, -40 through 40, with heating and beeping false. This differs from the test's explicit room/coldest pair. Its `boils` longest is 141.
- ReliableBroadcast uses its declared two processes, two payloads, reliable unordered capacity-four channel and initial root; `everyoneDelivers` longest is 6.
- TwoLayeredCache uses its declared root and intended history budget 4,096. `readFillsL1` and `agreement` verify (longest 1). `readAfterWrite` and `writeAfterWrite` remain unknown, incomplete at depth 10: their ordered-history monitor products are unbounded. Ordinary graph closure does not close those products.

verifyAll covers each owner's invariant/postcondition sections, scoped claim environments, progress searches and monitor products. It excludes separate Query/find targets and deliberately incorrect derived owners. The native gate separately preserves refuted controls, the 251-state alternate Voting quorum world, cross-refinement unknown, cache histories, progress deadlines/fairness, effect-error/refusal and explicit-root behavior. Those tests are semantic regression evidence, not extra measured graphs or throughput credited to this table. Sample ordinary counts do not account for all work performed by verifyAll.

### Scaled and held graphs

These additional runs retain the same seven-acceptor world. They are distinct from the initial 65,536-budget row.

| Voting7 budget | Reached states / emitted steps / BFS depth | Ordinary graph | First verifyAll invocation | Sampled live heap | Returned verdict |
| ---: | --- | --- | --- | --- | --- |
| 262,144 | 262,144 / 11,010,048 / 9 | incomplete, cut | 92.546721210 s, one repetition | 124,451,704 bytes / 81 accepted samples | both unknown, incomplete, depth 9 |
| 1,048,576 (default) | 1,048,576 / 44,040,192 / 13 | incomplete, cut | operational timeout at the 600 s command limit; no completed TIME | not run at this budget | no returned verdict |

Both statistics runs report zero unknown visibility holes. The largest completed ordinary graph observed is Voting5 (28,591 states); the largest completed budget-cut verifyAll invocation is Voting7 at 262,144. Default-budget statistics genuinely ran, but their cut does not establish full size. The timed default attempt exited 124 without TIME/RESULT output: 600 s is a resource limit, not measured checker time, unknown, or refutation. Its repeated metadata-threshold full-GC causes remain raw timing-JVM context, not a diagnosis or graph-live-heap measurement. No default-budget heap success is inferred.

Scaled statistics/timing retain the initial immutable artifact. The separate 262,144 heap run uses the formatted optional-cadence harness (Scala SHA-256 `9041023dd9ff81a939fc39e9fb8b06264838d9e69a71f70cf82d997530ba50dc`, artifact SHA-256 `dfa7b2ee1f9f643523a66a9f32f90af2324e48fb7da87c308bbaa7f062358995`) and a 1,000 ms sampling sleep. Its 81 samples had zero rejected intervals. The initial 65,536 heap probe's last GC occurred at about 54 s JVM uptime, versus about 15 s of measured checker time without the sampler; reducing sampling pressure for the larger probe is explicitly a methodology difference, not a checker optimization. GC remains intrusive at the lower cadence and neither cadence promises phase-complete peak coverage. Raw artifacts/input manifests are retained independently in `scale-quarter-v1/`, `scale-default-v1/` and `scale-heap-build1/` under `.flow/tmp/fn1963/`.

The parked Temporal equivalence roots cannot run unchanged against the current DSL. Retained ActivitySystem imports `framework.*`, `framework.Finite` and `framework.outcomes`, and extends `Machine[State,Outcome,Fact]`; current `framework.dsl.Machine[Outcome,Fact]` synthesizes var state. Parked compositions use removed `DerivedComposition`. Running their bundled old framework would measure another runtime; translating them would be held Temporal migration. This is source-compatibility evidence, not an invented compiler failure. Retained catalog/table-line counts do not establish a current reachable native graph. The authorized scaled-sample fallback supplies the growth experiment.

## Fresh-instance retention

Final 64-instance batches found 0/64 constructed-only Models alive, 64/64 initialized Models alive and 64/64 checked Models alive after observed full collections. All 128 survivors had direct identity-attributed cache roots; none was left unattributed. Cumulative post-GC heaps were 2,554,136, 2,953,648 and 4,995,968 bytes respectively. These cumulative process snapshots are not isolated per-group or per-instance sizes.

Each initialized survivor matched one `VarRuntime.lives` key's `Vars.machine`; after both initialized and checked batches this map held 128 entries. Every checked survivor also matched two `NativeCheck.spelled` action keys through actual `samples.Voting$Model$voter$.$outer` fields; the name map held 128 entries. Channel, claim, hole and declared-claim maps were empty for these controls; both runtime evaluation frames were empty. Their emptiness does not exonerate other owners/workloads or establish that all discovery indexes have safe lifetimes.

Runtime lifetime repair belongs to fn-196.6; all discovery-index coverage and combined remeasurement belong to fn-196.7, after this task and before fn-196.5 documentation and ModelSuite adoption. Both are future work. This note claims no cache fix or post-fix measurement.

The retention JVM returns only weak Model references from each batch helper. Constructed Models never initialize state; initialized Models call init and discard State; checked Models call public verifyAll with budget 1 and discard Report. Each batch receives an observed full GC. Attribution dereferences weak references only after collection and reads existing entries without calling lazy Model/signature getters or clearing caches. It reports overlapping roots separately and leaves unmatched survivors unattributed. Single-live-invocation heap and repeated fresh-instance retention are different experiments.

The observed strong seams are `VarRuntime.lives` -> `Vars.machine` and `NativeCheck.spelled` -> action actor -> actual enclosing Model field. Read-only reflection records private field names/types and identity matches. Other source-backed roots or growing metadata caches are not assumed to retain a particular instance merely from their sizes. No visited-set optimization is justified by mentioning hash keys: search already structurally deduplicates and retains concrete predecessor states/paths. Future optimization must preserve collision identity, every outcome and cut semantics, without concealing instance retention.

## Historical Go context

The retained fn172-r6 harness times `check.Check` on loaded historical IR with profiling enabled. Its after-sharing single observations were Kettle 14.589416 ms, TwoPhaseCommit 9.254639588 s and Bank 44.572743355 s; raw post-check HeapAlloc snapshots were 8,198,880, 2,999,833,760 and 7,232,966,640 bytes. That harness does not collect after checking, so these include uncollected garbage and are neither measured live heap nor peak RSS. Allocated-during-check bytes are a third quantity. No live-heap equivalence or speedup ratio is inferred.

The older six-sample warm-cache table in [reachable-exploration.md](reachable-exploration.md), section 12.1, reports Cache 0.01 s, Kettle 0.7 s, Broadcast 5.5 s, TwoPhaseCommit 10.6 s, Bank 48.6 s and Voting 52.0 s. Those full catalogs and default-scope Go checks include different state domains, Queries, refinements and receipts. Native reached states, historical serialized lines, enabled steps and whole catalogs are not interchangeable. No current Go/toolchain execution ran here. Source inventory and historical raw-path mapping are retained in `.flow/tmp/fn1963-prep/measurement-sources.md`.

## Retirement and lowering decision

The no-go reopens the spec's deletion boundary. These prerequisites must close before fn196.4 can delete the package/interpreter dependencies:

| Obligation | Disposition |
| --- | --- |
| Refinement simulation/cross-owner claims | Moved to native checking, pending. Preserve catalog compatibility, mapped starts/rows, visible outcomes/facts, stutter, unread visibility holes and incomplete regions. Native cross-owner claims currently answer unknown. Keep existing Go implementation until its obligation has a native owner and evidence. |
| Replacement composition proof | Moved to native composition admission/checking, pending. Ordinary product execution and same-vars replacement typing prove neither replacement refinement nor start coverage. Preserve Go `check/compose.go` obligations until transferred. |
| Rule disjointness | Kept as an unresolved semantic gate. `Effects.stepFunction` takes the first firing rule; RuleTable enforces structural binding restrictions, not pairwise guard disjointness. SEMANTICS' overlap-refusal promise needs native enforcement or a separately authorized semantic change. Deleting an assumed IR-only check cannot settle it. |
| Whole-catalog Query totals | Dropped under the approved 2026-10-13 reachable-exploration decision in UMPIRE4_SPEC. Future IR migration must remove both assertion validation and generator RequireTotals admission. Native graph counts do not equal old static totals. |
| IR/schema/realization admission | Kept at remaining artifact boundaries. Retiring a behavioral oracle does not admit malformed IR, descriptors, identities, unsupported declarations or Case provenance. |
| Recorded-Run Model assessment | Planned transfer to Scala under fn196 R5. Existing replay, binding, table/Property and admission guarantees remain until replaced. Go conformance planning consumes Realizer/Bound and table states; relabeling an entrypoint cannot transfer them. |
| Backend witness replay | Planned transfer with an explicit successor contract. Approved SPEC and Vision still require Go replay. Keep those guarantees until replacement is authorized and evidenced; export currently sends backend Queries through `check.Check`. |
| Numeric/export parity | Kept unresolved. Bank uses Scala Int; approved IR specifies checked signed 64-bit arithmetic. Bounded worlds establish no general numeric equivalence. |

Case lowering will run in Scala and emit the Testpilot Case proto. This removes a second Model interpreter from the intended end state. A future explicit producer interface can consume native witnesses and retained Scala realization values. This is not an existing feature of `Conversion.toIR`, which emits `Umpire.ir.Realization` today.

The Go-over-IR lowerer cannot simply lose one call: `lower.NewProducer` constructs `check.NewRealizer` and calls `check.Check`; `Producer.check` obtains `Realizer.Find` and interpreter tables; `lower/internal/producer.newProduction` independently calls `q.Answer` and reads table/witness identities. Export, exploration, lint and conformance also consume checker APIs. Keeping this lowerer unchanged keeps a Go Model interpreter. The Scala producer must transfer descriptor/command admission, Property-to-Contract clauses, witness-to-Program ordering, identities/fingerprints, unsupported standings, manifests and pin selection with exact Case evidence. Testpilot, Driver and canary remain in Go and execute Cases. No transfer is implemented here.

Source anchors: [Go lowerer](../../tools/umpire/lower/lower.go), [internal producer](../../tools/umpire/lower/internal/producer/producer.go), [conformance planning](../../tools/umpire/conformance/plan.go), [Scala realization conversion](../framework/realize/Conversion.scala), [native effect selection](../framework/checker/Effects.scala), [semantic obligations](../framework/SEMANTICS.md), and [approved reachable-exploration decision](../../.plans/UMPIRE4_SPEC.md). The task's source inventories retain the specific call sites and historical raw paths; these are inspected dependencies, not newly executed Go evidence.

## Reproduction and evidence

The committed `.sh` builds the committed `.scala` against current framework/sample sources using inherited strict compiler flags, then runs a fresh JVM. Use a new absolute `UMPIRE_MEASUREMENT_OUT` for each artifact/run family: existing build/classpath and per-mode GC logs are refused, not overwritten. Example:

```bash
env -u BASH_ENV UMPIRE_MEASUREMENT_OUT="$PWD/.flow/tmp/fn1963/example" bash model/design/scala-checker-measurement.sh build
env -u BASH_ENV UMPIRE_MEASUREMENT_OUT="$PWD/.flow/tmp/fn1963/example" bash model/design/scala-checker-measurement.sh voting5 stats 1048576 1
env -u BASH_ENV UMPIRE_MEASUREMENT_OUT="$PWD/.flow/tmp/fn1963/example" bash model/design/scala-checker-measurement.sh voting5 time 1048576 3
env -u BASH_ENV UMPIRE_MEASUREMENT_OUT="$PWD/.flow/tmp/fn1963/example" bash model/design/scala-checker-measurement.sh voting5 heap 1048576 1
```

Initial final matrix: `.flow/tmp/fn1963/public-matrix-v2.log` contains every expanded command and exit; all 25 children and aggregate runner handle 57510 exited 0. Raw observations are `public-observations-v2/<graph>-<mode>.log` (plus `retention-64.log`); matching GC logs and artifact/input receipts are in `public-build3/`. Retention's earlier `public-build1` evidence remains separate. `.flow/tmp/fn1963/final-commands.md` records development failures, source boundaries, limits and verification commands without rounding failures into passes.

Fresh allowed verification completed all 18 stages with exit 0: declarations/core/plugin, 664 native tests, 7 class-lint tests, 31 rule tests, 66 boundary tests, 173 source classifications, sample source/client compilation, literal realization-fixture equality, realization source and separate-library execution, and public-question source/library/client proofs (2 client tests). Strict compiler flags and held-source exclusions were preserved. Exact expanded commands and terminal exits are in `.flow/tmp/fn1963/final-generation.log`; frozen source and admission-file hashes were checked again afterward. FULL classification required these gates; no docs-only skip was taken.

Pre-simplification/private-entrypoint observations remain in `.flow/tmp/fn1963/historical-commands.md`. At owner steering, the already-running Bank timing finished before future launches were refused; those refusals earn no corpus credit. Historical and final artifacts are never merged. Full verification uses fresh allowed declaration/core/plugin/native/source/separate-library gates. Disabled Temporal files, broad GuideSourcesTest, both real-Go sample bodies and held Make routes remain held before execution. No Make-green, Go deletion, completed oracle-documentation migration or independent review verdict is claimed.
