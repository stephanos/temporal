# Generated differential testing spike verdict

Retain source generation and place the future harness in `model/umpire/dev/generate`.
Choose two generated source Models in one required-TLC batch, and one source Model
with compiled/interpreted comparison in default mode. Decline the optional tree-only
layer. The spike establishes complete graph comparison and detects seeded lowering
mutations. It activates no gate, adds no production crate and changes no framework API.
Task .2 builds the maintained harness; .7 and fn254.12 own admission to the gate budgets.

The retained probe and measurements are under `.flow/tmp/fn255-spike/`. Its committed
`evidence/reproducible-spike.tar.gz` and `evidence/archive-manifest.json` preserve the
probe, printed source, exact graphs, process observations and logs without build caches.
The README documents restoration and fresh execution. Evidence uses framework source
at `d9a378401bbf9c9e6a7aaf60c03e121c392512f5`, the existing prep checkout, rather than
claiming a later integrated fn254 source revision. A source projection against integrated root `0c83ab3d1e4b8593e7415d780902789d8bad7c90`
found only the TLC runner/cache-control changes. IR, front end, DSL, native checker,
emitter, testing support, protocol and framework dependency manifest/lock were
identical. Every spike execution sets `UMPIRE_QUICK_CACHE=0` and
`UMPIRE_CACHE_DISABLE=1`; retained runtime logs show actual execution, so the cache
changes do not invalidate this source projection. The host was Linux aarch64, 12 CPUs
and about 19.8 GiB RAM. Rust 1.99.1 and mise-pinned Temurin 27 ran TLC 1.7.4, with the
backend's expected jar SHA-256. BASH_ENV was removed from all subprocess environments.

## Source admission and independence

The seeded printer emits two two-variant enum fields, an integer with a reachable
upper bound of one or two, `Option<Token>` with a two-value `Finite` wrapper, four
unit actions, guarded checked arithmetic, an effect helper using `reject!`, optional
facts and a bounded invariant. Declarative starts select count 0 or counts 0 and 1.
Rustc compiles each printed Model's actual handlers. The interpreter evaluates its
embedded admitted tree. TLC receives a separately parsed complete printed source
through `Lift::parse` and `Tla::emit`. Compiled states or edges never construct TLC
expectations. Every selected backend must complete before snapshot comparison.

The preparation had compile errors and unsupported match expressions. The retained
failed logs distinguish those proposal defects from successful final measurements.
The probe now uses admitted conditional expressions and a unit-returning rejection
helper. No framework source changed to accommodate the generator.

`Interpreted::from_tree` accepts separately built typed data, but still requires a
compiled typed World, catalog and callback contract. A tree layer is possible in that
restricted sense; this spike did not establish an independent tree-construction
strategy. Native-only source controls already take milliseconds once compiled. A
second tree generator does not justify its extra construction and shrinking rules
for this first harness. Tree coverage cannot replace the mandatory source batch.

## Complete TLC extraction

An extra exploration records each real emitted `Edge`, predecessor fields and a
start/step marker, retaining the emitter's action, choice, outcome and ordered-fact
observers as decoded fields. Initial assignments have marker false. Every real
transition writes marker true and its predecessor. Revisiting a start cannot erase
that distinction. The emitter's tick distinguishes rejection/self-loop transitions.
The comparison projects complete start sets, reachable states, canonical labeled
edges and authored claim verdicts, rather than accepting state-set agreement alone.

The augmented transition relation adds records without restricting the original
`Edge`. From every reachable original state, every enabled action produces an
augmented successor containing that edge's full label. Conversely each marker-true
assignment comes from a real `Edge`; projecting it yields an original transition.
Complete exploration therefore yields the original relation. The generated Refuse
action is enabled in every state, so this family has no deadlocks. Synthetic
`Complete` transitions are excluded. Other families need their own completion proof.

The batch holds an unchanged case selector and nests independently emitted modules
with identical schemas. This forms a disjoint union, avoiding a cross product.
A conjunction guarded by the selector checks each case's invariant; any refutation
fails the entire batch. Reachability plus invariant checking take two model JVMs.
The same reachability dump supplies complete edge extraction, eliminating a separate
extraction JVM for each case.

For seed 255003, both starts in each Model survived exact graph comparison. Case 0
had 10 reachable states and 30 edges; case 1 had 6 states and 16 edges. Individual
extractions had 62 and 34 complete assignments. The batch had exactly 96. Retained
native/TLC and native/batch snapshot files are byte-equal. The batch thereby exercises
a two-start case, rejection self-loops, action labels, outcomes and nonempty facts.

## Mutation, refusal and reduction evidence

Every final case catches an interpreter-tree `<` to `>` mutation after successful
mutant exploration. It independently catches erased fact sequences and a rejection
outcome changed from NotFound to Unavailable. Both label-only controls preserve
reached states and claim verdicts. Full canonical edge comparison rejects them.
A checked mutant failure or refusal cannot count as detection.

Separate compiled and interpreted explorations with state cap one produce the same
real graph-cap refusal. Comparing those matching refusals fails. The probe avoids
the existing helper's matching-refusals-as-zero-states behavior. Runner controls also
verify that exit status 7 and an actual timeout propagate as failures.

A three-attempt deterministic source reducer retained successful detection of all
three mutations while reducing the first shape from `[201190345, 2, On]` through
`[0, 2, On]`, `[0, 1, On]`, and `[0, 1, Off]`. Every candidate recompiles and completes
compiled/interpreted checks; no TLC runs during reduction. The smallest retained
shape has one start, 6 states and 16 edges. This proves shrinking in the selected
three-coordinate strategy, without claiming global minimality over all Rust source.
Dropped-guard controls and broader structural shrinking remain task .2 work.

## Measurements and proposed R7 allocation

The same two printed Models produced these warm observations. Times include source
lifting and mutation controls; runtime is not a pure Java timer. Shared host load
varied, so these are individual measurements rather than a speedup guarantee.

| Execution | Compile seconds | Runtime seconds | Model JVMs | Discovery JVMs | Cases/minute including compile |
| --- | ---: | ---: | ---: | ---: | ---: |
| Individual TLC checks | 5.673 | 5.220 | 6 | 2 | 11.0 |
| Shared TLC batch | 9.129 | 2.140 | 2 | 1 | 10.6 |
| Final syntax-clean shared batch rerun | 4.109 | 1.477 | 2 | 1 | 21.5 |

Runtime-only throughput was 23.0 cases/minute for individual checks and 56.1 for the
first shared batch. Process sampling every 10 ms observed every expected JVM and a
peak of one. Retained TLC execution records independently show six versus two model
launches. The old backend's discovery probe has no explicit heap flag; model runs
use 512 MiB and one worker. Do not extend model-heap arithmetic to that probe or total
RSS. Later fn254 discovery caps must be retained when the harness is integrated.

Across the exploratory warm successful runs, the largest two-Model compile was
20.871 s and shared runtime was 7.020 s. Final shrink candidate compilation took
3.030, 3.511 and 3.880 s; their native/control runtimes were 0.012, 0.041 and 0.040 s.
Cold dependency compilation is a separate bootstrap cost and has no successful
cold-tier timing claim. One isolated target reached about 1.1 GiB. The initial failed
907 MiB target was cleaned with cargo clean; source and failure logs were preserved. Final cleanup removed 1.6 GiB of the
then-idle shared target. The last executed 76,987,152-byte probe binary remains
outside that target with its SHA-256 in committed evidence; the archive retains
rebuildable source rather than the binary.

The fn254.2 baseline records the existing healthy property-batch reduction from four
to two model launches and labels its missing fresh production timing baseline as
owner-waived. This spike uses that same exploration/property split. It measures an
additional extraction/batching workload and makes no whole-gate timing claim.

| Proposed work or time bound | Basis and enforcement |
| --- | --- |
| 36 states, 144 emitted edges, shortest depth at most 35, at most 2 starts per Model | Cartesian ceiling `2*2*3*3`, four actions, and a shortest simple reachable path; native preflight rejects cuts/over-cap output |
| 290 augmented assignments per Model; 580 per two-Model batch | At most two initial assignments plus two tick copies of every edge; decoded result refuses excess |
| Default mode, 1 source Model, 0 TLC launches | Preserve mandatory source coverage; native/control runtime measured below 0.05 s after compilation |
| Required mode, 2 source Models, 1 shared batch | Measured two-Model batch; 2 model JVMs plus 1 discovery, property_workers=1, cache bypass enabled |
| Default differential share at most 30 s | One warm compile and native comparison; no shrink during a successful gate |
| Required differential share proposed at most 45 s | Warm compile cap 30 s plus runtime cap 15 s, against observed maxima 20.871 and 7.020 s; leaves 75 s of the spec's 120 s for other techniques |
| Per model JVM 10 s; output 8 MiB; dump 16 MiB | Limits exercised by the probe; overall spike runtime ceiling is 45 s per selected case |
| Failure reduction at most 3 candidates | Actual measured reductions; production harness must fit reduction inside its owning stage deadline or report remaining work |

These are proposals for fn254.12's authoritative table, not activated allocations.
The throwaway runner keeps a separately capped 120 s bootstrap compile allowance.
The decoded augmented cap is a postcondition, not a live TLC state counter; timeout
and byte limits bound execution before decode. The finite source-family proof gives
the deterministic work ceiling. Any expansion to nondeterminism, more observers or
deadlocks must renew that proof and the measurements. Task .7 must remeasure on the
integrated fn254 baseline and cut counts if the combined gate misses its targets.

The three parked decisions are settled for this family. Source coverage remains
mandatory, the optional tree layer is declined, counts are one/default and
two/required subject to the proposed bounds, and TLC batching is adopted because it
reduces the required pair from eight total JVM launches to three. No gate activation
or prototype production implementation is part of this verdict.
