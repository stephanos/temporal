# fn-182 current full/slim measurement preparation

Prepared, with no timed cells launched. Root owns the host window and final
acceptance. The active WASM suite, runtime qualification and Temporal correctness
gates must finish before the measurement command is dispatched.

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

Current command, only after root releases the quiet host window:

```sh
cd /Users/stephan/Workspace/skunkworks/gomad/temporal
python3 -I .flow/artifacts/fn182-current-2026-10-16/measure.py \
  /Users/stephan/Workspace/skunkworks/gomad/temporal \
  /tmp/fn182-prefix-r5-20261016 \
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
