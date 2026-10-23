# Rust qualification receiver

This is provisional fn-232.13 implementation, not a qualification pass. The
conductor owns final integration, the frozen candidate, review and Flow state.
The original `.flow/artifacts/wasm-qualification-driver` and receiver tool custody
are read-only inputs. No old Go host qualification producer runs through these
adapters. Go compiler, generator, model and independent oracle checks remain
separate retained inputs.

`driver.py serial` is the new serial reference. `driver.py parallel` uses the
original 96-stage DAG and resource scheduler unchanged. Both dispatch the same
stage process and adapter code with identical argv from the same configuration.
The serial reference adds a predecessor edge between successive DAG rows. The
parallel mode retains cores, memory, locks, scratch declarations, the 8 GiB live
memory guard, heavy/budgeted isolation and exclusive timing stages. Busy-boundary
and grouped stages run the original twelve CPU workers and reap them before
returning. Background overlap is disabled. The original durations are estimates,
not new timeouts or a performance claim.

`semantic-map.json` preserves each label, original function body, selected test
roots, inspection subtests and repeated grouped-crash roots. It is a review map,
never proof of execution. `required-final.json` additionally requires all 39
transferred roots, the exact 21 tamper cases at four admission entrypoints,
full test-wasm reach, terminal, stress, retained/operator, R4 and rotated 120-second
ringpop proofs. Every missing, duplicate, skipped, synthetic or unexecuted item
refuses completion. Actual readers must examine raw identity-bound results;
printing this manifest does not satisfy a gate.

## Candidate configuration

Start from `bindings.example.json`. It deliberately contains unbound paths,
zero identities, zero send time and unavailable endpoints. Replace these with
reviewed candidate bindings; it cannot supply a pass as shipped. All paths are
absolute and use regular files, including the canonical Python interpreter.
Installations must come from the actual .11 installer and discover admission.
Pin the real release binary and raw build-feature receipt:

```json
{"binary_sha256":"<native 64-character digest>","features":[],"profile":"release","role":"release","exit":0}
```

That receipt is reviewed build evidence, not a role label that overrides the
compiled implementation. Pinned host/helper values must equal installation and
Result values; public discover verifies installation admission, fixture controls
must refuse, and qualify/report validate the actual linked configuration.
Compiler, Rust tool and retained oracle/custody input files belong in `inputs`.
The freezer also inventories every tracked regular non-Flow source and symlink,
all original parallel driver inputs and the serial reference. No source or
identity change may inherit the resulting candidate binding.

Record `sent_epoch` **before** configuration/freezing and preamble work. It is
never reset by a stage, preparation, scheduler or final reader. Elapsed wall time
includes preamble, source hashing, host admission, suspension and final checks.
A limit above 3600 seconds is refused. Successful stages finish after an overrun
so complete evidence survives, but the whole attempt exits 3. Fresh evidence
roots are exclusive; repeating an attempt never replaces an earlier one.

From the repository root, after all endpoint bindings are integrated and reviewed:

```sh
python3 tools/gomad/rust/driver/configure.py \
  --bindings /absolute/freeze/bindings.json \
  --output /absolute/freeze/driver.json \
  --source-manifest /absolute/freeze/source.sha256
python3 tools/gomad/rust/driver/driver.py parallel \
  --config /absolute/freeze/driver.json
```

Use `serial` in place of `parallel` for the new serial reference, with an
independently fresh attempt configuration/root. `plan` prints the 96 original
semantic bodies and shared dispatch argv without running any gate. The same
frozen configuration selects actual adapter argv in either mode. Each command
records its argv, cwd, PID, start/end, numeric exit, stdout and stderr; each stage
records its actual subgate outcomes, source, identity and configuration digest.
The final outcome retains the schedule, receipts and send-to-end wall clock.
The stage entry installs interruption handlers and clears the signal mask inherited
from the frozen launcher. Before a command executes, a private pipe gates its
exec until the parent registers its process group; parent death closes the pipe.
The receiver launcher clears the child's inherited mask and preserves the target
argv and PID across exec. An interrupted/timed-out process is killed and reaped
and recorded inconclusive. Receiver-owned scheduler cleanup also kills registered
command groups left by a hard-killed stage; the retained scheduler is unchanged.

Exit 0 means every actual stage, subgate and final reader passed within the
window. Exit 1 means a stage failed (its receipt distinguishes UNEXECUTED exit 4).
Exit 2 is setup/final-admission refusal, exit 3 is complete overrun, and 130 is
interruption. A final reader with absent original evidence never yields exit 0.

## Executable semantic mappings

| Original stage family | Rust execution and retained acceptance |
| --- | --- |
| Source/build/lint/portable/Rust matrix and two corpus regenerations | `checked-command` runs the reviewed retained source/compiler/model or Cargo command, then its exact-root/count or independent semantic reader. No selector can pass with zero/duplicate/skipped roots. Paths that retirement changes remain explicitly pending until the integrated Make/CI/source owners supply the reviewed argv. |
| Go wasmmeasure/helper build/publication | Pending linked release-build, admission and measured owner readers. The obsolete Go measurement/helper producers are forbidden. Existing labels and obligations remain. |
| `runtime-0` … `runtime-31` | Public explore, exactly 25 fresh one-seed campaigns at `index*1000003`, diagnostics, original 60-second execution and ten-minute shard bound, then a fresh forced exact replay. Public inspect authenticates all payloads. Raw diagnostic counters, hook witnesses, stdout/stderr/WASI/choices, summary and actual 16-byte runtime terminal agree, including version 1 and a nonzero peak witness. Canary process status 1 remains application output. |
| `runtime-aggregate` | Reopens all 32 shards/artifacts and independently checks current identities, payloads and map/runq/timer/canary variation, distinct choices/stdout and both exits 0/17. A missing terminal payload remains UNEXECUTED. |
| Ten workload cells | Public qualify followed by installation-free report. Exactly fixture controlled/cooperative, actual durable SQLite, unchanged `./tests` workflow-type smoke, and actual independent Temporal replication, success/canary each. Seed 7, Repeat 2, distinct observed CLI PIDs, retained exact replays, native Rust Result/receipt lineage, full source and compiled identities, measured budgets and attributed publication tail are mandatory. |
| Fixture/onebox configuration | `configure.py` generates actual original source selections and budgets. It refuses to synthesize SQLite/replication scenarios; those come from the actual producer owner. Onebox uses the original six test arguments and 15/30-minute, 4 GiB, 2^50-fuel limits, not FrontendSystemInfo. |
| Workload corpus and post-relocation corpus | Reopen the exact ten mandatory pairs, enforce distinct producer PIDs and compare every primary Result/receipt hash before/after selected-state publication. New selected results use separate roots from creation; there is no mutable shared primary directory to relocate out of. |
| Four SQLite selections | Decode the actual primary scenario, preserve every original bound, select the full native state and pending context, bind the actual primary Result hash, then invoke fresh qualify/report in each separate root. The primary installation, compiler/runtime/helper/engine/model/profile/config/module and unchanged raw provenance inputs must remain equal; the base Scenario bytes and every preparation provenance field remain identical. Only the exact top-level RuntimeWorkload selection request is added to retained provenance; the actual cluster configuration identity seals that request. Primary bytes and complete enumeration remain equal. |
| SQLite catalogue | Reopen all five actual Results, compare exact membership/context and distinct PIDs, and run an independent retained Go enumeration/context oracle against the actual authenticated GMSTORE state with 32-state/8-operation/8-depth/4 MiB/2^16 limits. Sum four selected execution costs and four publication tails once; never count primary cost again. Missing oracle is UNEXECUTED. |
| Terminal module | Public plan and its content-addressed .11 preparation receipt, authenticated prepared artifact/module/provenance, plus the original unchanged-module/admission reader. Preparation receipt CPU fields are never invented. |
| Artifact/operator/retained cluster | Public inspect and full replay with actual expected application exit, then the original domain/subgate reader. A small retained fixture alone cannot satisfy these labels. |
| Actual storage/network/replication/inspection, terminal, stress and remaining live corpus | Reviewed actual producer and independent reader endpoints. Each remains UNEXECUTED until integrated. Exclusive host admission and original capacities still apply; no portable substitution is permitted. |

The native Rust digest encoding is preserved as 64 lowercase hexadecimal
characters. Native Go crash-state/context values retain their PascalCase fields
and `sha256:` identities byte-for-byte. A Go implementation/source Model identity
maps to the actual compiled linked helper source/feature identity; the Rust Result
`model_sha256` hashes a particular model configuration and is not silently treated
as the former Go implementation hash. Per-cell compiler/runtime/module/profile,
clock, collector and limits remain in raw retained provenance and manifests.

## Endpoint readers

`checked-command` takes pinned `argv`, `timeout_s` and either an exact-root/count
reader or a pinned independent `checker` argv. Live stages cannot use Go host
producers, shell entrypoints or an exit-only reader. A semantic checker returns
JSON only after its assertions:

```json
{"schema":"gomad.rust-driver-check/v1","stage":"storage","source_sha256":"<actual>","identity":{"host":"<actual>","helper":"<actual>"},"subgates":[{"id":"TestTemporalCommittedWorkflowFreshGuestCrashReopen","status":"PASS","executed":true,"exit":0}]}
```

`final_reader` is a pinned actual evidence reader, invoked with `--root ROOT`.
It returns `gomad.rust-driver-final-check/v1`, the same source/identity fields,
and the exact `required-final.json` subgate set. It must authenticate commands,
raw artifacts, actual role identities and all original assertions. The missing
final reader blocks completion even when the 96 stage receipts are present.

The SQLite independent oracle is pinned `oracle` argv plus `--input FILE`.
The request names the actual digest-authenticated GMSTORE state and unchanged
enumeration limits. Its response is `{ "enumeration": <full native Go value>,
"context": <full pending context> }`; both must equal the Rust primary exactly.
A stored four-state fixture or coverage manifest cannot replace this execution.

Focused framework tests run with:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -v
```

These synthetic negative fixtures establish receiver behavior only. They earn
no actual runtime, workload, stress, terminal or qualification gate.

The owned live receiver is `live.py`; `final.py --config CONFIG --root ROOT`
composes it with the independent `receiving.py` raw-evidence reader and
separately pinned `final_sections` readers. The live receiver
owns 39 transferred roots, 84 admission cells and the distinct complete
`test-wasm` reach obligation. These are 124 of the 145 final obligations. The integrated receiving reader owns
13 more: runtime, the mandatory ten result/receipt pairs, SQLite primary/four
selections/catalogue, and the five Temporal obligations. It must return all 13
actual passing rows; missing, duplicate, failed or unexecuted evidence refuses.
Its source and imported cost/workload modules are frozen inputs, never proof of
execution. Eight obligations remain explicitly unbound: the two fn185-R4 checks,
rotated ringpop inspection, detached operator, all-public-verb differential,
20 retained replays, full stress matrix, and current terminal host admission.
Missing separately owned readers, missing reach mapping or missing actual runs
refuse completion. Existing historical receipts are not upgraded into a fresh
candidate proof.

A frozen `live_corpus` configuration names `roles`, all 15 `families` from
`live.FAMILIES`, `original_reference_commit`, `root19_rust_admission` and
`test_wasm_reach`. Each family binds an existing stage through `stage`, exact
`argv` and `role`; that stage lists it in `live_families`. Its fresh output is
`stages/STAGE/live-FAMILY`. Ordinary release/helper test roles require no helper
features; the host fixture role permits only the existing host fixture feature.
Each role seals the executable and successful release build receipt, including
actual source, host/helper hashes and feature lists. Inputs and complete input
directories must be in the frozen file inventory. No role is inferred from a
binary's filename.

The adapter runs each configured family and retains actual command/process
journals before writing its raw-file proof. The reader reopens those journals,
all raw files and the original pure comparators. Deliberate application failures
use their original explicit expected dispositions; absent exits, zero selections,
skips, duplicate cases and changed resource bounds cannot become PASS. Root19
requires both the authenticated frozen Go isolated-reference observation and
the actual Rust single-helper hard-isolation refusal. Root35 additionally needs
the ordinary helper's exact private ignored test and all 16 real post-Start
reinsertion comparisons; CLI shutdown rows alone cannot satisfy it.

`test_wasm_reach.map` is an independently reviewed original-source map, not a
39-root alias. Every original callable and expanded subtest must name actual
receiving testcase/raw proof IDs. `reports` binds those IDs to owning static
stages; the live reader re-executes the static reader's pure authentication.
Known ignored source tests retain their receiving obligations: the two aggregate
regressions map to root34 open-streams/seed7 and storage-crash/seed7, private
reinsertion maps to root35, and codec/module-recipe tests require their separate
actual test executions. Original opt-in nonqualification tests are not reported
as executions. A complete reviewed map and fresh receiving reports remain
required inputs; this source implementation creates neither.

For the remaining eight nonowned obligations, `final_sections.NAME` specifies exact `required` IDs
and pinned reader `argv`. `final.py` appends `--root ROOT`. A section returns
`gomad.rust-driver-final-section/v1` with exact frozen `source_sha256`, `identity`
and authenticated `subgates`; their disjoint union with the live section must
match all 145 required IDs. Missing owners remain UNEXECUTED.

The original source inventory is
`testdata/parity/original-test-wasm-reach.json` (relative to `tools/gomad`).
Its `gomad.original-test-wasm-source-inventory/v1` schema deliberately differs
from an executable final reach map. It fixes assignment-base revision
`1d695061b6382009d0659238c652a5a765b1d5f5`, both original Makefiles and every
selected/excluded test source blob. `reach.inventory()` reproduces lexical
callable/helper spans, original command selections, fixture exclusions and
subtest sites from those Git objects. Source reviews and receiving candidates
add explicit predicates and gaps; they never fill `evidence` or export PASS.
The initial regex count was invalid because guest source strings contain test
functions. Current discovery finds 752 actual callables and 756 original root
command selections. The four double-selected roots require distinct actual
`@stage/STAGE@run/command-NNNN` proof references in the final map; one process
cannot substitute for two original Make invocations. Dynamic subtest domains
remain required and unresolved until an exact source-reviewed expansion or
actual bounded receiving derivation is bound.

For the single original all-decision inspection site, a final map may use
`{"derive":"runtime-inspection-boundaries"}`. This closed derivation accepts only
that original source expression. It derives each kind/decision from actual raw
choice records and complete inspection metadata, then requires the exact stopped
request, verified prefix, real stack/heap and one command for every decision.
Missing cases and fixed-count declarations refuse. Expanded obligation IDs retain
the original site byte coordinate, preventing equal labels at different source
sites from collapsing into one obligation.
