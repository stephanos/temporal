# fn-182 current full/slim measurement preparation

The corrected measurement stopped at its first multi-instance baseline cell,
after all six one-box cells passed. Root owns the host window and final
acceptance. fn-180 correctness and completion gates are closed; fn-182's full
measurement and adoption gates remain incomplete.

## Current attempt status

The corrected invocation, session `75474`, exited 1 at
`base-multi-instance-rep1`. Its driver log is
`/tmp/fn182-prefix-r5-paths-20261016-driver.log`; the evidence root is
`/tmp/fn182-prefix-r5-paths-20261016`. Do not reuse that root or blindly restart
the command below. Driver/helper/test PIDs `1889659`, `1983941`, `1971026` are
gone. Launch HEAD was `00230127bf4dca80bd0ec1adeaf2e0673227317f`.

The six one-box cells passed in the original ABBAAB order, including the
driver's result, determinism, retained replay, provenance and budget checks.
All three baseline modules were 338,040,129 bytes, SHA-256
`21f9824815dfbda36c248d8fd98a8a04e5fc797a04976131aee665808ee0740f`;
all three slim modules were 306,428,662 bytes, SHA-256
`3350772a15206af271b440bbecc5499b5bd1e8c178c2136c216714be2c312b1e`.
This is partial evidence, not a completed performance comparison.

The failed multi-instance cell took 986.46 s including preparation. Its result
is `base-multi-instance-rep1/workloads/wasm-qualifications/v1/result-6b4d8bb1278115628ff920c7b16fb9de9d63966a6f783bb1627242e2715d7772.json`
under that evidence root. It has no completed executions or replay receipts.
The retained failure is `invalid_input`, with repeated empty
`cluster helper infrastructure:` messages. The shared helper wait retained
900.056536179 s wall, 898.531107 s CPU and 3,847,946,240 bytes peak RSS
(upper-bound attribution), but no terminal guest metrics or stdout. This is
consistent with the unchanged 15-minute deadline; the serialized result does
not retain an explicit watchdog flag, so timeout is not independently proved.
Missing terminal helper metrics also caused the secondary cost/budget errors
`actual cluster helper peak RSS and resident instances are required`.
No remaining multi-instance cells or canaries ran.

The final source postcheck passed all 5,927 regular entries; link mappings and
gitlink state, executable identities and clean launch HEAD were unchanged.
The frozen full-source manifest SHA-256 was
`c9afc27e47ee6f8b2501763d9a67c4fbc7d25420f0af8210d1878daef42f82dc`.
No budget, guard, workload or profile was changed after failure.

Read-only diagnosis found no concrete guest-environment or route mismatch:
current generated inputs match the retained fn-178 seed-7 configuration apart
from intended namespace/profile renames (captured mount payloads excluded from
that comparison). Both guests start before the first wait; the broker may run
either runnable guest. The retained failure cannot locate guest progress or
prove snapshot overhead. Qualification's error classifier handles runner host
errors but not `wasi.ClusterHostError`, explaining why its label alone cannot
distinguish infrastructure/deadline failures from bad inputs. Next diagnosis
must retain live progress and a bounded helper stack observation, with original
execution limits and evidence collection unchanged; it is not a benchmark.

## Historical admission failure and driver repair

The once-launched original session `45549`, driver PID `1882489`, exited 1.
`base-onebox-rep1` stopped after 0.212632 s at `onebox_test.go:83`:
`config/development.yaml is not a regular file`. No compiler/helper/guest ran.
Failed evidence remains `/tmp/fn182-prefix-r5-20261016` and
`/tmp/fn182-prefix-r5-20261016-driver.log`; it is not a performance sample.
Source/helper/binary/driver hashes are unchanged and processes are reaped.
The driver's two ignored cluster-fixture copies and empty failed preparation
directory remain retained; no staging deletion is claimed. HEAD at launch was
`aa1529d2c570498ac9925fb5ee05ecb6761aa300`;
the driver's full-repository source manifest hash was
`3d7e7de1ca7f8cfe6beaec18d7f717445a9dd77adfd607a8645505f37fb07eba`.

Diagnosis: inventory `Path.is_file()` followed the tracked configuration link,
but the existing qualification validator correctly rejects symlink manifest
entries. A second tracked schema link has the same admission mismatch. The
driver-only repair must retain logical link mappings plus contained resolved
regular-file hashes, with the mapping itself bound into manifest identity.
Do not weaken the validator, drop unbound links or replace repository links.
The failed root must not be reused. The updated command below names a new
evidence root, released only after the repaired-driver checks below pass and
all workers are quiet.

The driver-only repair is integrated with 19 installed tests passing (0.114 s).
It emits canonical regular-file hashes and a hashed `source-links.json` binding
both tracked symlinks, plus the index OID and verified empty/uninitialized state
of the Grafana gitlink. Changed mappings (even equal-byte retargets), modified
sidecar/manifest, escapes, broken links and populated/changed gitlinks refuse
before child execution. Qualification's existing regular-file validator and
all caps remain unchanged. Root's full-checkout hash-only capture/verify and
external SHA check passed at `/tmp/fn182-source-path-admission-eVLio4`:
5927 regular entries, two links and one gitlink. New driver SHA256 is
`b8987216f8a4df2d1c286310380b8aeda3480120ab4b2f39fad885e4858f17e2`.
Private RED/GREEN evidence: `/tmp/fn182-manifest-path-fix-20261016/`.

The measurement driver now uses an explicit system/cache/proxy allowlist for
every child and serialized command environment. Unrelated credentials and
inherited Go/Gomad/resource overrides are excluded; credential-bearing or
malformed proxy settings refuse without printing values. Every child uses
controlled `GOMAXPROCS=2`, identically for both profiles. The old policy's
synthetic credential leaks were reproduced before the fix; no real measurement
had started. Private RED/GREEN evidence is retained at
`/tmp/fn182-measurement-driver-hardening-20261016/`.
Root verified the installed seven-test suite (exit 0, 0.032 s) and unchanged
16-cell dry run (`dispatch=false`); the evidence root still does not exist.
Driver SHA256 is
`3217fc6edf9a807cbd1585d753522dfdc20e97cd90f012014bc5d2ca4bd287f5`.

Root releases the original 16-cell window after fn-180 completion review/closure,
all preparation workers reporting terminal/hold, clean main and matching
source/helper/binary hashes. Host preflight: load 0.50, approximately 17 GiB
available; no competing heavy process observed. The existing per-cell load
threshold 1.5 and minimum 8 GiB available remain unchanged. Freeze main sources
and serialize all owned heavy work for the entire invocation. Use the command
below once; retain terminal failures without blind restart or budget changes.
The driver records its actual start HEAD, full source inventory, executable and
input hashes, boot identity, process snapshots and per-cell commands/exits.
Release is not a measured result or workload-adoption claim.

## Current prefix-verifier preparation

Earlier compact, snapshot, listener-order and listener-history preparations are
historical, superseded by `/tmp/fn182-prefix-preparation-20261016-0K2eOu` on
source `e4f63ac483`. Pinned Go 1.27.1 Linux/arm64
`GOMAXPROCS=2 -p=1 -tags test_dep` compile passed, exit 0;
`qualification.test` SHA-256 is
`44f5f7eb0f07b00dddc61df55ba772d208fa0da907f1cf5b742e9885936b62ce`.
The private single-link helper is `0e3c62c5041a3bcd205a6d70b61eddf30b2b65e56ef796963f23596bb300bad7`.
Both profile-binding and configuration-only tests passed through that binary;
the three input hashes are unchanged. All 2,012 entries of the absolute source
manifest passed before and after preparation; its SHA-256 is
`597b200d8f83acd6f5c9115d46e44c8fcc2dac2957efb771c4db424dfeb646da`.
The dry run passed, exit 0, with `dispatch=false`, twelve successful cells and
four canaries. Logs and schedule remain in that private preparation directory.
No guest/helper or timed cell was launched. Proposed evidence directory
`/tmp/fn182-prefix-r5-20261016` does not exist. Listener-order prep
`aH4Z8O` (binary `1e6558e4…`, helper `2379a19f…`) and snapshot prep `N5lH1P`
(binary `f01668a7…`) remain historical. The history repair rotates the network,
WASI model, cluster/scenario and helper inputs, so those preparations cannot
qualify this candidate. The prefix verifier additionally rotates the shared
cluster and composed scenario identities, without changing helper, runtime or
WASI identity. Listener-history preparation `wfMn9s` (binary `7a23b3ca…`,
manifest `ac056b30…`) remains historical and was never measured. Its three JSON
inputs are byte-identical to this preparation. Full commands and input hashes
remain in the new private `summary.md` and `evidence.json`.

Historical corrected command (already ran and failed at cell seven; do not rerun):

```sh
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
python3 -I .flow/artifacts/fn182-current-2026-10-16/measure.py \
  /Users/stephan/Workspace/skunkworks/gomad/temporal \
  /tmp/fn182-prefix-r5-paths-20261016 \
  --stock-go /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go \
  --helper /tmp/fn182-prefix-preparation-20261016-0K2eOu/gomad-wasmhost \
  --binary /tmp/fn182-prefix-preparation-20261016-0K2eOu/qualification.test \
  --inputs /tmp/fn182-prefix-preparation-20261016-0K2eOu/inputs \
  --run
```

Without `--run`, this command validates preparation without creating the
evidence directory or launching guests. No measurement, workload adoption or
R4/R5 completion is claimed by preparation.

## Matched baseline decision

Both sides use the same current checkout, test binary, Go 1.27.1 compiler,
cooperative runtime, final release helper, engine, seed 7, two executions per
cell, captured schema, network/storage inputs and original resource budgets.
The baseline uses `disable_grpc_modules,sqlite3_dotlk,test_dep`; the candidate
adds only `gomad_wasm_slim`. Actual tags and modules remain bound by existing
preparation provenance and compiled-cache admission.

This re-establishes the full-closure baseline on the current real Linux/arm64
host while retaining the qualified fn-178 architecture and exact workload
selections. Applying identical fn-180 instrumentation to both profiles isolates
dependency trimming from instrumentation costs. The original fn-178 commit
`d087db1e48723d1164c19f559f78d75fc5731eb9` remains the historical baseline
reference; its measurements and helper identities are not relabeled as current.
No cross-build choice-tape compatibility is claimed.

The one-box selection remains the unchanged
`TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` with SQL/SQLite. Its
intentional failure remains the existing invalid-regexp harness canary, not a
new storage oracle. The multi-instance selection remains unchanged
`TestIndependentClusterReplicationRecovery`: active/standby replication,
directed partition/heal, both durable crashes and fresh-incarnation reopen.
Its canary retains the extra unexpected-history-event oracle failure.

## Preparation completed

The test-only profile selector defaults to slim; explicit `full` removes only
the slim tag and unknown selections refuse. The two existing qualification
drivers now honor `GOMAD_WASM_TEST_HELPER`, preserving their default helper
path. The configuration-only producer reconstructs the checked fn-178
producer's schema capture, entropy keys, environment, 4 GiB guest memory,
`1<<50` fuel, 256 MiB durable volumes, network limits and named partition/heal.
It builds no module and dispatches no helper.

Current compact-candidate preparation supersedes the earlier private preparation;
its helper and driver are not reused. Prepared files are under
`/tmp/fn182-compact-preparation-20261016-fPqjlj`:

- `qualification.test`: current frozen-source qualification driver, compiled
  with Go 1.27.1 Linux/arm64, `-p=2 -tags test_dep`, exit 0. SHA-256
  `059cd35f559500884a7f145a5a4cfba75116262ee34593254d4f9cd5dfeb1f91`.
- `inputs/`: three validated current configuration JSON files; producer
  command exited 0 and retained their hashes in `prepare-inputs.log`.
- `gomad-wasmhost`: private single-link copy of
  `/tmp/gomad-fn180-release-helper-compact`, SHA-256
  `a8dfd7fe41013d41e7206a6029759087b25fc39920d98688d44c8f5e20bdba97`.
- `schedule.json`: successful non-dispatch dry run, exit 0.
- `source.sha256`: copy of root's frozen compact-candidate manifest
  `/tmp/gomad-fn180-qualification-compact-v8lRoB/source.sha256`, SHA-256
  `3c990db2097f9490edc5b9df3977d6a8512c89b287237b14247a3f88bf3aa5f1`.
  Both `source-check-before.log` and `source-check-after.log` passed all entries.

`TestQualificationGuestProfilesBindOnlySlimSelection` and
`TestPrepareTemporalQualificationInputs` passed through this freshly compiled
binary. Configuration hashes remain unchanged. The binary includes the current
compact metadata v3 decoder and model provenance from the frozen source.
No guest/helper was dispatched and no measurement evidence root was created.
Rebuild the test binary after any later source edits and before the reserved
host window.

To recreate preparation without executing guests:

```sh
cd /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad
fn182_prepare=$(mktemp -d /tmp/fn182-compact-preparation-20261016-XXXXXX)
GOMAD_WASM_QUALIFICATION_INPUTS_ROOT="$fn182_prepare/inputs" \
  go test -p=2 -v -tags test_dep -count=1 \
  -run '^TestPrepareTemporalQualificationInputs$' ./internal/backends/wasm/qualification
go test -p=2 -tags test_dep -c -o "$fn182_prepare/qualification.test" \
  ./internal/backends/wasm/qualification
cp /tmp/gomad-fn180-release-helper-compact "$fn182_prepare/gomad-wasmhost"
```

## Measurement sequence and preserved acceptance

[measure.py](fn182-current-2026-10-16/measure.py) runs the existing qualification
drivers. It does not add another execution or measurement owner. For each
workload, three fresh roots per side run in `AB, BA, AB` order: `ABBAAB`, where
A is full and B is slim. This is three independent cold rounds per side,
including the ABBA reversal; it is not three ABBA blocks (six rounds per side).
The raw schedule is retained. Four further fresh canary cells cover both
profiles and workloads.

Every successful cell retains two recordings and two fresh retained replays.
Cluster process costs are counted once per helper, and startup/execution stay
per incarnation. Within-cell warm samples are not represented as independent
measurement rounds. Fresh roots exclude preparation and engine-cache reuse
between rounds; the first helper compiles and later helpers load that cell's
cache. Qualification, replay, intentional-failure counts and budget results
must all pass before a cell can contribute to acceptance.

The existing retained summarizer supplies module/cache bytes, initial Go
preparation, cold compilation, warm load/engine preparation, cold/warm fresh
instantiation, execution and image RSS, with raw samples, medians/ranges and
gain-versus-spread labels. No startup, execution or memory gain is inferred
from fewer linked bytes. The driver also binds the actual dependency tags,
source inventory, host/CPU/boot information, binaries and input hashes; checks
identical compiler/runtime/helper/engine/model/config/limits between profiles;
and refuses source, executable or input changes during the run.

Runtime equivalence uses the genuine preparation provenance's cooperative
overlay key and runtime implementation digest. The qualification manifest's
separate `RuntimeSHA256` inventory is retained as observed: it includes selected
standard-library files and absolute private overlay paths, so it need not be
identical across profiles or fresh roots.

The original budgets remain in `EvaluateBudget`: preparation ≤10 min,
execution/replay ≤15 min, compiler ≤6 GiB, cold single-helper image ≤6 GiB,
warm single-helper image ≤4 GiB, resident helper bound ≤8 GiB, operator
lifetime peak ≤2 GiB, and guest peaks within their configured memory limits.
The host guard waits for load below 1.5 and MemAvailable ≥8 GiB before each
cell, retaining before/after host observations. A guard failure, missing
telemetry, failed replay, changed canary or resource violation stays an actual
failure; it supplies no accepted comparison.

## Reserved-window command

Run only after root releases the measurement window. The evidence root must
not exist; a failed/interrupted root is preserved and a later run uses a new
root. This command runs twelve measured success cells plus four canary cells.
Prior cells took minutes each, so allow hours for the complete set.

```sh
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
python3 -I .flow/artifacts/fn182-current-2026-10-16/measure.py \
  /Users/stephan/Workspace/skunkworks/gomad/temporal \
  /tmp/fn182-compact-r4-20261016 \
  --stock-go /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go \
  --helper /tmp/fn182-compact-preparation-20261016-fPqjlj/gomad-wasmhost \
  --binary /tmp/fn182-compact-preparation-20261016-fPqjlj/qualification.test \
  --inputs /tmp/fn182-compact-preparation-20261016-fPqjlj/inputs \
  --run
```

Without `--run`, the same command validates preparation and prints its
schedule without creating an evidence root or launching guests. Final R4/R5
receipts, measured deltas, runnable-document reconciliation, broad gates,
independent review and Flow completion remain pending. Native execution stays
deferred under fn-128/fn-149.
