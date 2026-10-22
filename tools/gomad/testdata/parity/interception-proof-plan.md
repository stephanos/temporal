# fn233 task 1 proof and corpus

This is an execution plan, not completed evidence. Keep the failed v1/v2 roots
unchanged. Root must grant a new immutable fixture build and each actual Go wave.
Do not start shim tasks before the single-package and real parallel-package gates.

## Fresh single-package Go proof

Freeze one integrated source commit containing the shared preparation corrections,
structured outcome rows and reviewed launcher fixes. Record clean source identity,
compiler/helper/host hashes, binary hash, build command, numeric exit and logs. Copy
the fixture executable to a new evidence root; never replace the v2 binary.
The following shell variables name those concrete frozen inputs:

```sh
PROOF_ROOT=/home/agent/evidence/fn233/interception-early-v3-20261022
PROOF_SOURCE=/absolute/frozen/integrated/checkout
PROOF_BINARY="$PROOF_ROOT/gomad-fixture"
PROOF_COMPILER=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go
```

Create a NEW root with mode0700. Under the granted setup window, run the existing
zero-guest installation and shared recipe exporter, preserving argv/env/cwd and
stdout/stderr/status as separate files:

```sh
GOMAXPROCS=2 "$PROOF_BINARY" install \
  --compiler "$PROOF_COMPILER" \
  --runtime-root "$PROOF_SOURCE/tools/gomad/internal/model/runtimesource" \
  --temporal-checkout "$PROOF_SOURCE" --temporal-package ./tests/gomadfunctional \
  --cache "$PROOF_ROOT/preparation" --output "$PROOF_ROOT/installation.json"

GOMAXPROCS=2 "$PROOF_BINARY" fixture-driver-prepare \
  --output "$PROOF_ROOT/manual-invocation" \
  --installation "$PROOF_ROOT/installation.json" \
  --choices --seeds 1 --execution-timeout 5m --overall-timeout 10m \
  --artifacts "$PROOF_ROOT/artifacts" --working-dir "$PROOF_SOURCE" \
  go-test ./tests/gomadfunctional -- '-test.run=^TestFrontendSystemInfo$' -test.v=true
```

Do not invent the final Go command. Read it verbatim from the exporter response,
including the generated overlay, full tags `disable_grpc_modules,gomad_wasm_slim,
sqlite3_dotlk,test_dep`, readonly module mode, disabled PGO, trimpath, buildvcs=false,
count1 and quoted `-exec '<immutable-binary> run-guest'`. Record the exact private
invocation/recipe bytes and digests. Preserve the installation's original4GiB,
2^50 fuel,300s execution and600s overall limits; verify them in admitted records.

After root's start message, an external controller runs the existing command:

```text
IMMUTABLE_BINARY fixture-driver-run --invocation ABS_INVOCATION --sha256 EXACT_DIGEST
```

Spawn it in a new process group; it fsyncs the registration before exec and keeps
its PID as the actual pinned Go PID. Supervise externally for600s, then SIGTERM,
5s cleanup and SIGKILL only if necessary. Persist the controller/Go PID, exact
argv/env, launcher PIDs, original temporary module paths, admission receipts,
pre-exit output timing and final statuses. `/proc` observation may corroborate on
Linux; it is not the production admission path. An observation polling expiry
never restarts the command. On any actual failure, stop this dependency chain and
retain the admitted or explicitly unaccepted actual bytes; do not rebuild a lost
module or label an unaccepted input replayable.

Require actual Go test PASS for TestFrontendSystemInfo, package `ok`, and exit0.
Authenticate the actual retained module, record, intercepted origin, full argv,
distinct launch/driver/wrapper identities, compiler/settings and recipe association.
The helper's controlled pre-exit handshake remains separate from observing output
before the Go driver exits; do not conflate those observations.

## Replay with inputs unavailable

First confirm Go removed its original temporary module. Use per-process hiding,
not deletion of shared source/cache paths. With the working bubblewrap facility,
read-only-bind `/`, isolate user/PID/network, replace `/proc` and `/dev`, then hide
`/Users`, `/home/agent/wt`, `/home/agent/.cache`, the Go module cache if present,
`PROOF_ROOT/preparation`, `PROOF_ROOT/manual-invocation` and `/tmp` with empty
mounts. Hide the selected installation file with `/dev/null` too. Preserve only
system libraries, the immutable binary and retained evidence/target pool.

Execute the exact explicit replay argv printed by the launcher (or its equivalent
explicit `IMMUTABLE_BINARY --installation INSTALLATION replay ARTIFACT` on success)
inside that namespace. Record the full bubblewrap argv, proof that each hidden
input is unavailable, stdout/stderr/status, and `reproduced=true`. No source,
compiler, external module cache or Go temporary file may be recreated for replay.

## Real parallel package proof

The source extension uses a bounded PRIVATE collection contract and the shared
prepare/admit algorithm. It is not compiled or actual acceptance evidence yet.
The original single-recipe encoding remains unchanged. No operator verb or shim
is introduced.

The collection must bind one real Go PID and full argv to an exact set of two
existing packages under the same installation/profile. Bind each package's source
inventory, actual import path, flags/settings/configuration and compiler identity.
Select the actual module only through strict framed build-info package binding;
reject absent, ambiguous, duplicate or mismatched members. Retain a distinct
per-launch invocation identity and private attempt/campaign while preserving the
shared authenticated Go-run and wrapper association. No input-metadata match alone
admits a module.

Root approved the concrete existing pair `./tests/gomadfunctional` /
`TestFrontendSystemInfo` and `./common/tasktoken` / `TestWorkerTaskID` with selector
`^(TestFrontendSystemInfo|TestWorkerTaskID)$`. Source inspection of tasktoken found
no TestMain/init; the selected test checks existing string/strconv helpers with
testify assertions. This approval registers the inputs, not an execution grant.
After the single-package proof and the separate setup grant, generate a fresh
private collection at a new path, using the same installation and unchanged caps:

```sh
GOMAXPROCS=2 "$PROOF_BINARY" fixture-driver-prepare \
  --output "$PROOF_ROOT/parallel-invocation" \
  --installation "$PROOF_ROOT/installation.json" \
  --package ./common/tasktoken \
  --choices --seeds 1 --execution-timeout 5m --overall-timeout 10m \
  --artifacts "$PROOF_ROOT/artifacts" --working-dir "$PROOF_SOURCE" \
  go-test ./tests/gomadfunctional -- \
  '-test.run=^(TestFrontendSystemInfo|TestWorkerTaskID)$' -test.v=true
```

The private exporter derives the exact `-p=2` Go command, freezes both distinct
import paths/inventories and registers the full argv before exec. Preserve those
actual bytes/digests and the real registration; do not construct a lookalike
single-package invocation. Launch the exported invocation only after root grants
that actual wave, under the original external 600s supervision/5s cleanup.

Run ONE pinned `go test -p=2` with TWO DIFFERENT package operands, count1, the complete
shared recipe and the real launcher. Repeated identical package operands are
Go-deduplicated and are not proof. Cap active launchers at2, retain actual overlap
observations, distinct receipts/campaigns/origins and both package statuses at one
artifact root. Separate concurrent Go drivers are useful additional checks but do
not replace this wave. Replay both artifacts after temporary inputs are hidden.

Focused source regressions to run only after a compile-lane grant: original single
recipe canonical bytes and argv; exact two-package argv/member selection;
missing/duplicate/ambiguous/settings/config/compiler refusal; per-launch identity
and receipt renewal; four sequential/concurrent campaigns with the same driver
association and shared artifact root; and authenticated legacy artifact reading
without the optional launch extension. These tiny WAT/source fixtures supplement
the actual Go proof and do not establish its acceptance.

## Public CLI corpus automation

`interception-check.py` runs the six declared cases twice, independently checks
campaign/artifact commitments, validates every structured outcome row against its
record/manifest, checks default stdout against exact retained guest bytes, proves
first-failure/unexecuted semantics (including a separate later-would-pass probe),
checks exact final replay argv/refusal, runs replay and inspect, minimizes the
replayable failure, and probes actual qualification provenance admission.

Supply an input JSON with schema `gomad.interception-corpus-inputs/v1` and `cases`
containing exactly the names in `interception-corpus.json`. Each case provides:

- `name`, absolute `module`, actual guest `args`, `seeds`, `json`, expected `status`,
  `replayable`, and `outcomes` in execution order, each with `ordinal`, `seed`,
  `class`, `termination`, and mapped launcher `status`.
- Every prepared case additionally has absolute `receipt` and `receipt_sha256`
  from a genuine actual-module preparation; the unprepared case omits these.
- `first-failure` additionally has `later_would_pass_seed` and `later_outcome`
  (`class`/`termination`), causing a separate actual successful probe campaign.

Use existing guest fixtures and registered inputs; do not translate Temporal tests
or manufacture accepted receipts. The infrastructure case must use an admitted
actual module and its original explicitly configured low resource limit, not an
admission parse error. The first-failure case must prove a later selected seed
would pass while its original campaign retains that seed as unexecuted. Keep a
separate actual compiled-Go callback/partial-evidence case in the final acceptance;
the unit WAT boundary is supplemental.

The runner also requires a genuine previously accepted qualification Result and
an ordinary release binary. It first proves `report --result` accepts that Result,
then uses the existing qualification tamperer to copy/readdress Result+receipt with
ONLY genuine authenticated intercepted provenance substituted. The public reader
must refuse at retained-provenance admission, not an earlier fixture-role/parse
error. Separately, ordinary `qualify --spec` is probed with a prepared-artifact
source and its existing source-only refusal is labelled accurately. There is no
invented `qualify ARTIFACT` command or new public acceptance path.

```text
GOMAD_GOLDEN_REGENERATE=1 python3 interception-check.py
  --binary IMMUTABLE_FIXTURE --release-binary ORDINARY_RELEASE
  --installation ABS_INSTALLATION --qualification-result ACCEPTED_RESULT
  --inputs ABS_ACTUAL_CASE_INPUTS --work NEW_PRIVATE_EVIDENCE_ROOT
```

All argv are passed as arrays without a shell. Each process keeps the600s overall
and5s cleanup supervision and is never retried on expiry. The runner requires two
fresh observations to produce byte-identical canonical golden views. Only known
artifact/campaign paths and independently authenticated per-launch record/origin
commitments are represented by stable labels; guest byte hashes and every raw
binding remain exact in evidence. Do not check in invented goldens: the golden is
created only by a real successful double regeneration, then rerun without the
regeneration variable against another fresh work root. Record every numeric exit,
raw output hash, actual receipt/module/binary hashes and remaining gap in the
single handover before task1 can be accepted.
