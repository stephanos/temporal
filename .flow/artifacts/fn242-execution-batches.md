# fn-242 execution batches — owner decision, 2026-10-24

Use 16 batches for the remaining 47 original tasks. Keep six writer lanes and two heavy build slots. Finish the existing .12/.13 implementations without restarting them. Preserve any review already in flight; subsequent reviews follow this owner-authorized policy. This policy changes execution grouping and Flow admission, preserving product acceptance.

The original 267-edge task DAG and the additive leader graph are acyclic. No dependency is removed and no task is created. Independent review barriers fall from 47 task gates to 16 batch gates. The equal-duration review critical path falls from 17 to 7 barriers; the widest earliest wave has six batches. These are graph counts, not a wall-time forecast.

| Batch | Leader | Members in original topological receipt order | External original dependencies | Checks / functions / endpoints / semantics | Wave |
| --- | --- | --- | --- | --- | --- |
| B01 custody-inputs | fn-242.14 | .14, .16 | .1, .11, .13 | 221 / 54 / 5 / 0 | 1 |
| B02 attempt-admission | fn-242.15 | .15 | .11, .13 | 24 / 0 / 0 / 0 | 1 |
| B03 bindings-composition | fn-242.17 | .17, .19 | .1, .11, .14, .16 | 69 / 26 / 2 / 0 | 2 |
| B04 reader-foundation | fn-242.18 | .18, .21 | .1, .11, .14, .16 | 315 / 19 / 0 / 0 | 2 |
| B05 guest-inspection | fn-242.24 | .24, .25, .35 | .1, .18, .21 | 301 / 7 / 0 / 1 | 3 |
| B06 world-lifecycle | fn-242.27 | .27, .36 | .1, .18, .21 | 264 / 0 / 0 / 0 | 3 |
| B07 retained-workload-cells | fn-242.26 | .26, .34, .45 | .1, .18, .21, .27 | 359 / 31 / 1 / 11 | 4 |
| B08 runtime-retained-native | fn-242.20 | .20, .22, .23 | .1, .13, .16, .17, .18 | 272 / 37 / 1 / 4 | 3 |
| B09 static-installed-producers | fn-242.28 | .28, .31 | .1, .13, .15, .19 | 138 / 70 / 7 / 0 | 3 |
| B10 build-primary-identity-producers | fn-242.29 | .29, .40, .41 | .1, .13, .15, .19 | 264 / 91 / 5 / 0 | 3 |
| B11 workload-reference-oracle-producers | fn-242.30 | .30, .32, .33 | .1, .13, .15, .19 | 138 / 66 / 8 / 0 | 3 |
| B12 terminal-profile-reference-readers | fn-242.39 | .39, .42, .43, .46, .48 | .1, .13, .18, .32, .33 | 548 / 37 / 2 / 3 | 4 |
| B13 installed-cluster-public-readers | fn-242.37 | .37, .38, .47 | .1, .18, .28, .31, .41 | 387 / 17 / 3 / 1 | 4 |
| B14 live-reach-retained-integration | fn-242.44 | .44, .49, .50 | .1, .14, .18, .20, .24, .25, .26, .27, .30, .34, .35, .36, .45 | 235 / 1 / 1 / 41 | 5 |
| B15 routes-original-aggregates | fn-242.51 | .51, .52, .2, .3, .4, .9, .10 | .1, .6, .11, .12, .13, .14, .15, .16, .17, .18, .19, .20, .21, .22, .23, .24, .25, .26, .27, .28, .29, .30, .31, .32, .33, .34, .35, .36, .37, .38, .39, .40, .41, .42, .43, .44, .45, .46, .47, .48, .49, .50 | 91 / 43 / 2 / 84 | 6 |
| B16 parity-cutover-handoff | fn-242.5 | .5, .7, .8 | .1, .2, .3, .4, .6, .9, .10 | 0 / 0 / 0 / 0 | 7 |

Start B01 (.14+.16) and B02 (.15) after actual .13 completion. After B01 ships, start B03 (.17+.19) and B04 (.18+.21) together. After B02+B03, start the three producer bundles immediately; they need no final-reader completion. B04 releases the reader lanes. Admit later bundles as their own dependencies finish, without a whole-wave barrier.

The batch critical path is B01 -> B04 -> B06 -> B07 -> B14 -> B15 -> B16. Earliest wave widths are 1=2, 2=2, 3=6, 4=3, 5=1, 6=1, 7=1. The DAG maximum antichain width is 6.

## Leader execution and receipt contract

1. Create no tasks and remove no dependency. Expand each existing leader description to its full member scope and declared Touches to the conservative member union. The original exact touch declaration is retained alongside its normalized conflict globs. Broader ambiguous integration/test/CLI scopes stay conservative.
2. Add only the listed union external dependencies to each leader. Do not add an internal member dependency to a leader. Preserve every original dependency and proof owner. Add the leader as a dependency of each receipt-only member so mechanical admission cannot dispatch it separately. The batch manifest maps original task to execution leader only.
3. Admit a leader only after every union external dependency is actually done. This is stronger than its original start condition and protects later internal execution. Mechanical admission reserves the entire conservative union Touches until the batch finishes.
4. The leader implements every member in implementation_and_receipt_order. Internal compile/fixture checkpoints may run during implementation; internal members have no review or worker handoff gate. No member can be credited from a declaration or an unreviewed future implementation.
5. After one independent whole-batch SHIP review, start/done each member in original topological order with the exact same actual reviewed source revision and shared review plus per-member proof evidence. The leader may finish before later receipt-only members only in this post-SHIP transaction. Receipt-only members record the actual implementation and their own original acceptance checks, not an administrative fake closure.
6. B16 receives three independent final cutover draws covering the whole final candidate. All previous batches receive one independent fresh-context review. A changed batch after review requires appropriate revalidation and review of the changed scope.
7. Preserve every original acceptance, exact original check/function/endpoint ID and all 145 semantic obligations. Original aggregate .2/.3/.4/.9/.10 scope is exercised in B15; its implementation reuses completed child surfaces. Their original acceptance is never replaced by child status alone.
8. Original R1 requires a catching test OR mutation for every retained guarantee. Direct malformed/input mutation and valid control executions cover every retained guarantee with exact row/source/failure/trust-boundary/command/exit bindings. Use representative high-risk source mutants; do not require recompiling one source mutant per guard. Shared fixtures may prove several rows only when each row has a direct catching outcome. Hash-only, ancestor-only, future named tests and status rewrites do not prove a row.
9. Keep the roughly 100k-token iteration guardrail. Check/function counts guide batching, not mandatory splits or proof waivers. If a real iteration does not fit, stop at a source/proof checkpoint within the batch and continue its next bounded iteration without inventing task closure or dropping proof. Repartition only by revising this manifest and revalidating both DAGs before dispatch.

## Effort reduction within each batch

Apply the owner's request to reduce fn-242 effort to both implementation and evidence work.

- Group original check IDs by concrete failure and immutable trust boundary before writing fixtures. One executed catching case may cover several IDs when it directly exercises each guarantee. Add another case only for a distinct failure or boundary; preserve independent receiving opens and exact original-ID coverage.
- Port the required behavior through existing Rust validators, codecs and process APIs. Fold duplicate Python helpers and bookkeeping checks into their actual enforcing boundary with an explicit inventory disposition. The Python module/function layout does not prescribe the Rust layout.
- Keep one compact machine-readable proof map per batch. Derive source/input/log hashes and original-ID crosswalks from executed fixture output where possible. Reuse raw evidence by reference rather than copying it into separate per-task narratives or building another proof framework.
- Reuse a worker's successful focused gate after integration when the conductor verifies identical tested source and relevant dependencies, exact command/environment, numeric exit and raw log bindings. Run additional checks for actual integration changes or uncovered interactions. Integration alone does not require repeating an identical build/test command.
- Freeze a coherent source unit before compiling. Reuse compiled fixtures for input mutations and independent valid controls; reserve source recompilation for representative high-risk mutants. Keep the existing one-review-per-batch policy and the two heavy-build slots.

These reductions preserve every original acceptance and proof obligation. Check IDs remain
traceability entries; they do not each require their own test, source mutant or document.

## Bounds and final qualification

- No batch dispatches a full qualification attempt. Both global attempts remain unused through fn-242. Preserve fixed journal binding, ambiguous-spawn charges and the immediately rechecked 45 GiB HOST / 20 GiB overlay / 12 GiB inner floors.
- Preserve all 96 semantic stages, exact unchanged leaf argv, explicitly mapped orchestration replacements, bounds, resource/isolation declarations, 8 GiB live-memory guard, 12-worker busy groups and 3600-second policy with stop_at_deadline=False timing-only deferral. Correctness/resource/replay/infrastructure failures retain original classification.
- Actual full candidate production, full qualification and independent receiving remain fn-246 after all fn-242 and all 13 fn-240 tasks. B16 handoff preserves authentic frozen Go baseline inputs and every still-pending R4/public18/original R6 acceptance.
- Known prerequisite fn-232.33 remains explicit. Mandatory storage/network/clusters producer installations need --compile-cache ROOT/engine-cache; non-provisional missing-cache admission must refuse with status 2 and named reason; standalone success/canary prove within_budget=true and cache_hit=true on every non-first window. Keep its original acceptance and source owner; B11/B10 port routes must carry the same binding, and fn-246 cannot start without the actual prerequisite proof.
- B14 implements strict authenticated reach-proof handling while gaps remain explicitly uncredited. Actual fn-246.1 proof activates strict receiving; old fn-232.14 task status cannot supply proof or cause a start deadlock.

## Exact declared write scopes

Each leader reserves the complete normalized union below. The JSON also retains every member's original Touches line, exact original acceptance, original dependencies, original check/function/endpoint/semantic IDs and source hashes. Ambiguous CLI/test/producer scopes stay conservative. No early batch claims aggregate .2's cycle-wide Touches.

### B01 custody-inputs

Authenticate source, build and immutable preparation inputs together.

Leader dependency additions: fn-242.1.

- `tools/gomad/rust/gomad/src/cycle/build/**`
- `tools/gomad/rust/gomad/src/cycle/config/**`
- `tools/gomad/rust/gomad/src/cycle/custody/**`
- `tools/gomad/rust/gomad/src/cycle/preparation/inputs/**`

### B02 attempt-admission

Keep fixed enrollment, durable attempt accounting and reservation floors independently reviewable.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/admission/**`
- `tools/gomad/rust/gomad/src/cycle/journal/**`

Conductor integration extension: B02 owns the exact scheduler parser and re-export files below. Preserve one strict byte parser and the existing GiB wrapper; run existing scheduler tests. Original acceptance and proof ownership are unchanged.

- `tools/gomad/rust/gomad/src/cycle/scheduler/admission.rs`
- `tools/gomad/rust/gomad/src/cycle/scheduler/mod.rs`

### B03 bindings-composition

Freeze role bindings and compose the fixed executable plan without waiting for reader implementations.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/preparation/bindings/**`
- `tools/gomad/rust/gomad/src/cycle/preparation/composition/**`

### B04 reader-foundation

Implement independent command/artifact custody and retained binary codecs as the common reader boundary.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/readers/command/**`
- `tools/gomad/rust/gomad/src/cycle/readers/core_codec/**`

### B05 guest-inspection

Share decoded guest and inspection fixtures, then validate installed inspection and recovery.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/readers/core_guests/**`
- `tools/gomad/rust/gomad/src/cycle/readers/core_inspection/**`
- `tools/gomad/rust/gomad/src/cycle/readers/inspection/**`

### B06 world-lifecycle

Keep World/network/boot/cluster-wire decoding and lifecycle recovery in one implementation lane.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/readers/core_lifecycle/**`
- `tools/gomad/rust/gomad/src/cycle/readers/core_world/**`

### B07 retained-workload-cells

Join retained mutation matrices, mandatory Results/SQLite/history validation and cell closure.

Leader dependency additions: fn-242.27.

- `tools/gomad/rust/gomad/src/cycle/readers/cells/**`
- `tools/gomad/rust/gomad/src/cycle/readers/core_retained/**`
- `tools/gomad/rust/gomad/src/cycle/readers/workload/**`

### B08 runtime-retained-native

Review runtime replay aggregates, retained/operator/ringpop final sections and native command custody together.

Leader dependency additions: fn-242.16, fn-242.17.

- `tools/gomad/rust/gomad/src/cycle/final_readers/native/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/retained/**`
- `tools/gomad/rust/gomad/src/cycle/readers/runtime/**`

### B09 static-installed-producers

Keep static/broad and installed/lifecycle/interception/ringpop producer work free of final-reader admission dependencies.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/producers/installed/**`
- `tools/gomad/rust/gomad/src/cycle/producers/static_adapter/**`

### B10 build-primary-identity-producers

Build actual CLI/harness outputs, then execute primary/runtime/cell/corpus and identity/divergence adapters.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/producers/build/**`
- `tools/gomad/rust/gomad/src/cycle/producers/identity/**`
- `tools/gomad/rust/gomad/src/cycle/producers/primary/**`

### B11 workload-reference-oracle-producers

Implement workload/recovery/support, frozen-reference/public-role and SQLite/terminal producer adapters in parallel with the other producer lanes.

Leader dependency additions: none.

- `tools/gomad/rust/gomad/src/cycle/producers/reference/**`
- `tools/gomad/rust/gomad/src/cycle/producers/sqlite_terminal/**`
- `tools/gomad/rust/gomad/src/cycle/producers/workload/**`

### B12 terminal-profile-reference-readers

Join terminal, independent SQLite/reference readback, current profiles and diagnostic/release preservation. Keep current and frozen inputs independently authenticated.

Leader dependency additions: fn-242.13, fn-242.32.

- `tools/gomad/rust/gomad/src/cycle/final_readers/profile_reference/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/profiles/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/reference/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/sqlite/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/terminal/**`

### B13 installed-cluster-public-readers

Read installed/interception/original-cluster/ringpop evidence and public/identity/divergence results after the producer bundles ship.

Leader dependency additions: fn-242.28, fn-242.41.

- `tools/gomad/rust/gomad/src/cycle/final_readers/clusters/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/installed/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/public/**`

### B14 live-reach-retained-integration

Validate live families and their 40 transferred semantics, ordered support/reach proof, then compose independent retained-reader sections.

Leader dependency additions: fn-242.20, fn-242.26, fn-242.30, fn-242.34, fn-242.35, fn-242.45.

- `tools/gomad/rust/gomad/src/cycle/final_readers/reach_support/**`
- `tools/gomad/rust/gomad/src/cycle/readers/integration/**`
- `tools/gomad/rust/gomad/src/cycle/readers/live/**`

### B15 routes-original-aggregates

Close final partitions, integrate genuine routes, wire the public cycle CLI and discharge every original aggregate acceptance from the implemented child surfaces.

Leader dependency additions: fn-242.6, fn-242.11, fn-242.12, fn-242.13, fn-242.14, fn-242.15, fn-242.16, fn-242.17, fn-242.19, fn-242.21, fn-242.24, fn-242.25, fn-242.28, fn-242.29, fn-242.30, fn-242.31, fn-242.32, fn-242.33, fn-242.40, fn-242.41, fn-242.45, fn-242.46.

- `tools/gomad/CLI.md`
- `tools/gomad/rust/gomad/src/**`
- `tools/gomad/rust/gomad/src/cli/**`
- `tools/gomad/rust/gomad/src/cycle/**`
- `tools/gomad/rust/gomad/src/cycle/config/**`
- `tools/gomad/rust/gomad/src/cycle/final/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/**`
- `tools/gomad/rust/gomad/src/cycle/final_readers/admission/**`
- `tools/gomad/rust/gomad/src/cycle/preparation/**`
- `tools/gomad/rust/gomad/src/cycle/producers/**`
- `tools/gomad/rust/gomad/src/cycle/readers/**`
- `tools/gomad/rust/gomad/src/cycle/routes/**`
- `tools/gomad/rust/gomad/src/lib.rs`
- `tools/gomad/rust/gomad/src/main.rs`
- `tools/gomad/rust/gomad/tests/**`

### B16 parity-cutover-handoff

Execute parity dispositions, audit all original proof rows, delete replaced Python, run focused cutover verification and publish the fn-246 handoff.

Leader dependency additions: none.

- `.flow/artifacts/fn242-handoff-*`
- `.flow/artifacts/fn242-obligation-inventory*`
- `.flow/artifacts/fn242-parity-dispositions.md`
- `.flow/artifacts/fn242-size.md`
- `.github/workflows/gomad.yml`
- `tools/gomad/ARCHITECTURE.md`
- `tools/gomad/CLI.md`
- `tools/gomad/Makefile`
- `tools/gomad/internal/backends/wasm/QUALIFICATION.md`
- `tools/gomad/internal/backends/wasm/determinism/inventory.json`
- `tools/gomad/rust/driver/**`
- `tools/gomad/rust/gomad/src/**`
- `tools/gomad/rust/gomad/tests/**`
- `tools/gomad/rust/wasmhost/src/**`
- `tools/gomad/rust/wasmhost/tests/**`
- `tools/gomad/rust/workload_tools/**`
- `tools/gomad/scripts/**`
- `tools/gomad/testdata/parity/**`

## Review and resource policy

B01-B15 each receive one independent fresh-context whole-batch review. B16 receives three independent final cutover draws. The proposal requires 18 review draws in total. Child tasks consume evidence from the same actually reviewed implementation. They retain their own acceptance and proof IDs.

Six source writers total, including nested implementation delegates, may edit disjoint admitted unions. The conductor allocates those lanes across active batches; leaders remain the sole committers and reconcile every delegate. Read-only support does not consume a writing lane. Only two heavy builds or compilation-causing tests may run at once, including reviewer verification and cutover checks. Review can overlap disjoint source work. Check disk before every heavy wave, respect current target ownership and preserve active/shared cache inputs.

## Sizing and proof limits

- Critical-path numbers count independent review/dispatch barriers and assume equal batch durations. They do not predict wall time or promise 17/7 speedup. Wider batches have unequal implementation cost.
- The 548-check B12 is the largest reader bundle. Its five closely related terminal/profile/reference modules retain independent current/frozen trust boundaries and direct negative fixtures. It is a sizing watchpoint, not permission to weaken proof.
- Function count zero in core readers means no function_mapping partition was assigned there; it never means zero implementation. B16 owns an all-row audit and cutover even though it adds no proof-owner IDs.
- The 145 semantic IDs count authoritative proof ownership exactly once. Combined receiving may compose existing semantic rows, including .50 narrative references, without minting duplicate owners.

## Re-run the DAG and ownership validator

Run from the Temporal repository root. This validator reads the proposal only and writes nothing.

```sh
python3 - <<'PY'
import json, pathlib
p = json.loads(pathlib.Path(".flow/tmp/fn242-batch-proposal.json").read_text())
snapshot = {int(k.rsplit(".",1)[1]): [int(x.rsplit(".",1)[1]) for x in v] for k,v in p["original_dependencies"].items()}
B = p["batches"]; owner = {}
for b in B:
 for t in b["members"]:
  assert t not in owner, t
  owner[t] = b["id"]
assert len(owner) == 47
assert set(owner) == {t for t in p["original_dependencies"] if int(t.rsplit(".",1)[1]) not in {1,6,11,12,13}}
def acyclic(d):
 todo = set(d); done = set()
 while todo:
  ready = {t for t in todo if set(d[t]) <= done}
  assert ready, ("cycle", sorted(todo))
  todo -= ready; done |= ready
d = {k:list(v) for k,v in p["original_dependencies"].items()}
for b in B:
 members = set(b["members"]); ext = {x for t in members for x in d[t] if x not in members}
 assert set(b["external_dependencies"]) == ext
 assert not (set(d[b["leader"]]) & members)
 assert set(d[b["leader"]]) <= set(b["proposed_leader_dependencies"])
 assert set(b["proposed_leader_dependencies"]) == set(d[b["leader"]]) | ext
 leader = b["leader"]
 d[leader] = b["proposed_leader_dependencies"]
acyclic(d)
contracted = {b["id"]:sorted({owner[x] for x in b["external_dependencies"] if x in owner}) for b in B}
acyclic(contracted)
for field, expected in [("check_ids",3626),("function_mapping_ids",499),("endpoint_ids",37),("semantic_obligation_ids",145)]:
 ids = [x for b in B for x in b["proof"][field]]
 assert len(ids) == len(set(ids)) == expected, (field,len(ids),len(set(ids)))
print("PASS: 47 original tasks, 16 batches, both DAGs acyclic, zero removed edges, 3626 checks, 499 functions, 37 remaining endpoints, 145 semantics")
PY
```

## Durable authority and admission

The owner explicitly requested batching, greater risk tolerance for throughput, and durable capture. This policy supersedes one-worker/one-review-per-original-task defaults for these batches. Read the JSON manifest before admission or review. Only batch leaders run implementation workers. Receipt-only members depend on their leader and close from the same genuinely reviewed implementation, in original topological order. All original acceptance remains binding.

Six coding lanes share two conductor-granted heavy build/test slots. A worker requests a slot and exclusive target before heavy compilation, then releases it only after its command is terminal. Other lanes continue source and fixture work. Run one fresh high-effort correctness/integration review over all member acceptance per batch; B16 receives the final three-axis review. A review that returns findings still requires fixes and revalidation.

The global two-attempt allowance remains policy/activity accounting. The fixed operational enrollment binding was absent at the read-only survey; this is not a verified durable counter or permission to recreate missing history. Actual enrollment and verification precede fn-246 dispatch; fn-242 uses fixture-only journals.
