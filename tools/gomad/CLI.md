# Gomad CLI and maintainer guide

Gomad's execution backend is WASM. The native backend and its maintainer
commands were removed by fn-223; Git history retains their documentation.

Start with the [installation and first-failure guide](README.md). The
[WASM operator guide](internal/backends/wasm/CLI.md) documents installed
commands, configuration, campaigns, inspection, replay, recovery and minimization.
The [product specification](SPEC.md) and [architecture](ARCHITECTURE.md)
define the shared campaign and evidence contracts.

## Rust artifact inspection

The Rust host provides `gomad inspect [--json] [--choices] ARTIFACT` without
an installation or a guest execution. It opens only Rust storage lineage v2.
Go artifacts, unsafe roots, corrupt manifests and unauthenticated payloads
return status 2 with no report; report write failures return 3. Every payload
is authenticated through the shared pinned-root opener, including payloads
not displayed in the selected view. `--choices` requires recorded choice
metadata and validates the raw trace through the helper's replay projector.
Sites retain their existing fingerprints, kind, count and maximum alternatives;
listed decisions retain their recorded offsets and missing-site flags.

With `--choices`, every aggregate site and listed decision adds `symbol`,
computed only from the authenticated retained module. Available symbols contain
`package`, `function` and `inlined_from`, with `file` and `line` when proved.
An inline chain requires complete agreement among compatible retained select
calls. Physical fallback has an empty `inlined_from` and omits unproved
locations; it does not establish that the compiler performed no inlining.
Single-alternative sites receive symbols even without a listed replay decision.
Runnable sites recorded as missing retain `{"unavailable":"site_missing"}`.

Other unavailable reasons are `no_symbols`, `offset_out_of_range`,
`symbol_mismatch` and `symbol_table_too_large`. A real `-s` Go module can retain
pclntab but lack the name cross-check required for attribution; this yields
`no_symbols`, as does actual pclntab absence. Contradictory metadata yields
`symbol_mismatch`. Module read/digest failures retain the existing open error.
One immutable result map covers unique requested sites, with cumulative limits
of 64 MiB each for metadata/allocation and inspected code, 1,048,576 entries,
16,777,216 traversal steps, 256 inline frames and 1 MiB per name/path. Exhaustion
replaces all nonmissing results with `symbol_table_too_large`, including any
previously resolved site. No attribution is written into the artifact, record
hash, failure signature or execution identity.

Campaign expected/observed divergence symbols require fn-232.7/.8's authenticated
execution-ledger consumer. They are not artifact choice-trace fields; standalone
replay text does not supply that consumer. fn-229.2 remains open until its public
campaign fixtures cover both decisions against each execution's retained module.

The JSON schema changes from Go's `gomad.inspect/v8` to `gomad.inspect/v9`:

| Field | Rust v9 shape and change from Go v8 |
| --- | --- |
| `schema`, `kind`, `path` | Schema is v9; kind is `artifact`; path is absolute and lexically cleaned, as in Go. Campaign/store inspection arrives with its owning Rust verbs. |
| `artifact.record` | Full canonical `gomad.execution-record/v2`, including seed, arguments, environment, limits, virtual time, outcome, identities and streams. It replaces the separate Go `artifact_kind`, campaign/ordinal, replay-mode, runner, toolchain and target projections. Go lineage has no compatible identity or campaign fields to copy. |
| `artifact.record_sha256` | Bare lowercase hex record digest; replaces `record_hash`. |
| `artifact.failure_signature` | Bare lowercase hex seed-excluded signature, formerly nested in `outcome`. |
| `artifact.files`, `artifact.stored_bytes` | Authenticated relative payload map (`bytes`, `sha256`, `module`) and total retained artifact bytes, including manifest and storage envelope. |
| `artifact.stdout`, `artifact.stderr` | Preserve `full_sha256`, `total_bytes`, `retained_bytes`, `discarded_bytes`, `truncated`; hashes use bare hex. Absent output is the empty stream. |
| `artifact.record.streams` | Payload-keyed `bytes`, `sha256`, `records`; replaces the separate transcript projection and includes captured/diagnostic streams when retained. |
| `artifact.choices` | Null without `--choices`. Otherwise retains choice-inspection v3 summary, limits, profile, implementation/trace/tape digests, terminal state, sites and replay decisions. Integers are native JSON integers; `site_missing` is explicit; decision arrays and tape digest are empty when exact replay is unavailable. |
| `artifact.world` | Null or authenticated profile/implementation, seed, time, transition count, terminal kind/detail and bounded history payload descriptor. This exposes the World facts carried by the new lineage. |
| `artifact.simulation` | Null or profile, controller/execution/candidate/outcome/failure digests and bounded plan/record payload descriptors. Descriptors add the contained `file` and `limit`; absent failure digest is null. |
| `artifact.minimization` | Null or typed parent/candidate identities, attempt/forcing bounds, ordered accepted reductions and full predicate. The Rust detail schema is `gomad.minimization-evidence/v1`; all digest strings are bare hex. |
| Optional sections | `choices`, `world`, `simulation`, `minimization` are explicit nulls when absent. Unknown fields in retained typed details are refused. |

Optional facts live in canonical `evidence/inspection.json` under
`gomad.inspection-evidence/v1`; the execution record must bind that payload as a
stream. Its `choices`, `world`, `simulation`, `minimization` and `outputs` fields
are nullable. Payload descriptors authorize only manifest-listed contained
reads, with independent byte bounds and digests. Choice target/toolchain and
trace/tape consistency, World snapshot seed/time/count, output accounting and
minimization reduction chains are checked before rendering. Consumer bounds are
applied before allocation; remaining payloads authenticate with fixed-buffer
streaming through the same pinned opener. Choice overflow requires the matching
runtime-choice capacity failure; interrupted prefixes require a watchdog timeout
or explicit runner cancellation, and complete traces require an exact tape.
Minimization reductions admit `schedule_suffix`, `schedule_range`, and
`fault_entries`, with supported decision dimensions, unique removed coordinates,
effective hash-linked candidates and reconciled original/final counts. Standalone
choice suffixes must remove contiguous terminal ordinals; simulation suffixes
remove runtime controls, ranges remove non-fault controls, and fault entries
remove fault controls. Source paths and
installations are never reopened.

The mandatory local harness command is:

```sh
python3 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb inspect \
  --inspect-binary /absolute/fixture-enabled/gomad \
  --inspect-baseline /absolute/authenticated/go-baseline \
  --inspect-source /absolute/extracted/baseline/source \
  --inspect-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --output-root /absolute/new/inspect-check
```

Missing inputs fail this gate. Fresh independent Go and Rust executions cover
the six termination/model cases and real runtime choices; each binary inspects
its own newly published artifacts. Real low-limit runs also compare overflow
prefixes, partial site/count facts and unavailable exact replay. Exact inspect goldens preserve canonical
record bytes/hashes, identities, payload digests and raw ledger bytes; only the
declared output-root prefix in `path` changes. Regeneration requires
`GOMAD_GOLDEN_REGENERATE=1`, fresh repeated agreement and a byte-identical second
regeneration. `testdata/parity/inspect-corpus.json` declares this view.

The checked-in `inspect-details.json` is consumer coverage: World/simulation
facts derive from fresh retained-helper transport runs, and minimization facts
derive from an actual fresh Go minimizer run on a tiny concurrent failure
fixture. Original raw Go metadata remains a supplemental authenticated payload.
This is neither a Rust minimizer run nor linked-host cluster qualification.
The fixture encoders and provenance are documented in
`inspect-cluster-input.py` and `inspect-details-input.py`. Integration with the
Rust simulation/minimization producers belongs to fn-232.8/.9.

The existing ARM CI runner source runs the Rust inspect consumer, opener and
golden gates. Fresh Go/Rust CI parity still needs reproducible provisioning of
the authenticated baseline bundle, extracted source and compiler; fn-232.13
owns that final execution wiring. The local gate above remains mandatory.

## Rust retained replay

`gomad replay [--verify-only] [--observed DIR] ARTIFACT` consumes Rust-lineage
retained inputs through the same authenticated pinned-root opener as inspect.
It does not rebuild targets or reopen original source, captured input paths,
compiler or runtime files. Verify-only also validates the helper-owned semantic
choice tape before returning status 0, without creating an engine or guest.
Exact cooperative replay and observed stock replay keep the existing text
reports: matching success returns 0, matching target failure or divergence
returns 1, incompatible artifacts return 2 and execution infrastructure errors
return 3. `--toolchain-root` retains its native-only refusal.

Replay inspection retains `--stop`, `--instance`, `--incarnation`, `--dump`,
`--probe`, `--check`, `--choice-range`, `--search-mode`, `--debug-output` and
their existing bounded search/debug controls. Stacks and heap use the built-in
`--dump stacks` and `--dump heap` selectors. A stop validates a retained prefix
and reports inspection results; it does not claim full reproduction. Inspection
requires complete cooperative metadata, with an explicit target for clusters.
Cluster replay uses typed, bytes-owned incarnations over the existing helper
broker, storage, namespace and clock owners. No artifact pathname is reopened.
Retained cluster replay preserves its existing refusal of `--verify-only` and
`--observed`; those options apply to standalone replay.

The Go launcher's diagnostic launcher-plus-helper pair maps to one matching
compiled Rust binary. Build `helper-stress-jitter` for `--stress-jitter`, or
`helper-determinism-fixture` for `--determinism-fixture`; these remain explicit
cooperative inspection opt-ins. An ordinary release binary refuses either role.
The separate `determinism-fixture` host feature exposes only test drivers and
does not enable a helper role. Qualification uses an ordinary release with
neither host fixtures nor diagnostic roles. Diagnostic conditions and bounded
event-order collection belong to each fresh execution, including cluster threads.

The mandatory local command supplies the authenticated frozen Go baseline and
its extracted source and pinned compiler:

```sh
python3 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb replay \
  --replay-binary /absolute/fixture-enabled/gomad \
  --replay-stress-binary /absolute/stress-role/fixture-enabled/gomad \
  --replay-determinism-binary /absolute/determinism-role/fixture-enabled/gomad \
  --replay-release-binary /absolute/ordinary-release/gomad \
  --replay-baseline /absolute/authenticated/go-baseline \
  --replay-source /absolute/extracted/baseline/source \
  --replay-cluster-baseline /absolute/current-go-and-helper-reference \
  --replay-cluster-source /absolute/current-go-source \
  --replay-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --replay-preparation-root /absolute/retained/replay-toolchain-preparation \
  --output-root /absolute/new/replay-check
```

The standalone replay/divergence baseline stays frozen. Cluster, network and
registered inspection lanes use an archived current Go launcher and matching
helper because fn235 deliberately changes cluster failure semantics and
inspection World commitments. The original failing registered guest remains a failure case; a
separate retained success variant proves all three incarnations and restart.
The standalone network lane compares every field and type in the modeled JSON
document, including local faults and World history, and its exact encoded byte
count. Rust orders JSON object keys differently from Go; each lineage's raw
payload digest is independently authenticated before that comparison.

`testdata/parity/replay-corpus.json` declares the golden view. Records, manifests,
ledgers, statuses, replay reports and observed output digests remain exact; only
the verified artifact operand's output-root prefix changes. Independent raw
record/payload and ledger validation precedes comparison. Set
`GOMAD_GOLDEN_REGENERATE=1` to regenerate after reviewing expectation changes;
each run first compares two fresh executions, and a second regeneration must
produce identical golden bytes. This lane supplies replay acceptance evidence,
not final runtime/workload qualification or a compiler hard-wall guarantee.

Keep the declared preparation root fixed across recordings and regeneration.
Frozen Go binds absolute overlay paths into its toolchain provenance, so a new
preparation directory is a different authenticated input, even when guest bytes
agree. Every execution still produces fresh Go/Rust artifacts in its new output
root; neither provenance identities nor raw hashes are normalized.

## Rust seed campaigns

The linked Rust host accepts the original WASM seed grammar, environment policy,
failure policy and retention bounds. `go-run` and `go-test` preparation invokes
the pinned stock Go compiler and original runtime overlay builder; campaign
selection, scheduling, journal transactions and recovery stay in Rust. The
internal `gomad.rust-wasm-installation/v1` manifest binds the compiler, runtime,
helper and host identities. Public installation publication belongs to the
installation task; a frozen fixture target is not a replacement for source
preparation.

Campaigns use `gomad.campaign-storage/v2` and hash-linked canonical journal
envelopes. Resume authenticates the complete prefix and retained prepared target
before admitting work. Only a torn final append may be repaired under the writer
lock; complete malformed records, changed identities or changed bounds refuse.
Admission, raw completion receipts and ordered policy decisions are separate
durable events. A committed policy decision reconciles discarded artifacts;
shared module pool links remain authenticated by the existing store capability.

`execute-shard --shard INDEX/COUNT PLAN` reads a retained seed plan. `merge`
authenticates every source shard and its original plan before publishing a
global-order result; `--partial` records missing ordinal intervals explicitly.
Duplicate, overlapping or incompatible shards refuse. `inspect --json --choices`
projects typed divergence from the authenticated execution ledger and resolves
raw expected/observed sites only against that execution's retained module.

The mandatory fresh local lane is:

```sh
python3 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb explore \
  --campaign-binary /absolute/fixture-enabled/gomad \
  --campaign-baseline /absolute/authenticated/go-baseline \
  --campaign-source /absolute/extracted/baseline/source \
  --campaign-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --campaign-input /absolute/identity-matching/replay-input.json \
  --campaign-preparation-root /absolute/retained/campaign-preparation \
  --output-root /absolute/new/campaign-check
```

The same lane accepts `--verb resume`, `execute-shard` and `merge` and exercises
all four operations. The frozen Go WASM public CLI refuses shard/merge verbs;
the lane records those real refusals separately and calls the unchanged Go
Runner API through a fixture-only reference adapter to attempt shard/merge
semantics. The adapter calls the original provider and Runner owners without
altering their preparation, authentication or capability policy.
Those API-reference results are not public-CLI positive parity.
Rust positive paths remain actual public CLI invocations. This admission
discrepancy remains an explicit integrated-acceptance question, not a normalized
semantic match.

The frozen portable-plan implementation also drops backend metadata when it
reopens a prepared target and rejects backend payload files in its bundle.
Consequently the original WASM portable API lane currently fails closed rather
than supplying a positive shard/merge reference. Its failure is an unresolved
differential criterion, not a skipped gate or an authenticated match.

Raw record, payload, plan and journal integrity checks run independently of
goldens. `GOMAD_GOLDEN_REGENERATE=1` requires repeated fresh agreement and a
byte-identical second regeneration. Host-only journal interruption controls
require `determinism-fixture` and are absent from ordinary release binaries.

## Rust choice and simulation exploration

`explore --strategy=choice-exploration` explores helper-validated forced prefixes
in bounded breadth-first rounds. Select exactly one seed and provide positive
`--max-executions`, `--max-choice-depth` and `--max-exploration-bytes` bounds.
`--choice-start-ordinal` optionally skips earlier branching decisions.

```sh
gomad explore --strategy=choice-exploration --seeds=7 --parallel=2 \
  --max-executions=16 --max-choice-depth=4 --max-exploration-bytes=64MiB \
  --artifacts /absolute/campaigns go-test ./package -- '-test.run=^TestExample$'
```

`--strategy=simulation-exploration` combines runtime, scenario, network, storage,
fault and crash decisions. It requires positive `--max-executions`,
`--max-forced-decisions`, `--max-exploration-bytes`,
`--max-exploration-result-bytes` and all six `--max-DIMENSION-decisions` bounds.
Its actual execution decisions are retained in `simulation/record.json`;
controller fixtures alone do not establish producer or workload qualification.

Both strategies retain typed transactions containing a complete round and its
authenticated execution observations. `resume CAMPAIGN` restores the recorded
bounds and frontier; it refuses malformed complete journal entries and may
truncate an interrupted final entry before retrying that round. `inspect --json
--choices CAMPAIGN` reports the explored envelope and typed replay divergence,
with site symbols resolved from each retained execution's matching module.

## Shared generated contracts

The Go models, cooperative runtime overlay and Rust helper share wire schemas
under `internal/model`. Generate their codecs with stock Go and no build tag:

```sh
make -C tools/gomad generate
make -C tools/gomad validate
```

`internal/model/cmd/protocolgen` owns generation. The volume and network models
are owned source, with differential corpora holding the Rust implementations
equal to the Go reference. The
[WASM implementation guide](internal/backends/wasm/README.md) documents helper
builds, corpus regeneration, focused checks and measurement commands.

## Replay ownership

For forced runtime Choice replay, the helper validates the retained raw trace and
projects its canonical tape before instantiating the guest. A malformed trace,
changed execution identity or mismatched trace/tape digest is refused at this
boundary; it is not a guest decision divergence. Prefix exploration and
inspection send separate controls without changing retained evidence. Helper or
model identity changes require fresh compatible artifacts; replay does not
silently migrate old ones. This ownership change does not alter exit statuses or
establish qualification for a newly built helper.

## Qualification

Use the [WASM qualification contract](internal/backends/wasm/QUALIFICATION.md)
and [milestones](../../MILESTONES.md) to identify required gates. Source checks
for native deletion do not establish a runtime or workload qualification pass.
Final changed-candidate runtime, mandatory workload and retained-replay gates,
including the complete behavioral reach of `make -C tools/gomad test-wasm`,
remain pending under fn-232 R12/task .13's frozen Rust qualification cycle.
