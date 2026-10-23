# Gomad CLI and maintainer guide

Gomad is the single Rust `gomad` binary built from the `tools/gomad/rust`
workspace (`make -C tools/gomad build`), with the WASM engine linked in. Its
execution backend is WASM. The Go CLI, Runner and host producers were retired
by fn-232.13, and the native backend and its maintainer commands by fn-223;
Git history retains their documentation. Comparisons below against the frozen
Go reference are retained parity evidence, not a supported Go command.

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
| `artifact.stdout`, `artifact.stderr` | Metadata for the global `output/stdout` and `output/stderr` files: `full_sha256`, `total_bytes`, `retained_bytes`, `discarded_bytes`, `truncated`; hashes use bare hex. Absent global files are empty streams. Cluster aggregate and per-incarnation facts are in `artifact.cluster`. |
| `artifact.cluster` | Present for authenticated cluster observations. `stdout_bytes` and `stderr_bytes` are checked sums of retained per-incarnation bytes. `offered_output_bytes` counts all offered output separately; `discarded_output_bytes` is offered minus retained, and `truncated` means that difference is nonzero. `instances` binds node/incarnation, interruption state, per-stream retained counts, offered/discarded counts, truncation and an authenticated stdout suffix of at most 4096 bytes. `virtual_elapsed_nanos` and `exhausted` preserve the original elapsed time and exact capacity-bound facts. No combined output hash or cross-incarnation byte order is invented. |
| `artifact.record.streams` | Payload-keyed `bytes`, `sha256`, `records`; replaces the separate transcript projection and includes captured/diagnostic streams when retained. |
| `artifact.choices` | Null without `--choices`. Otherwise retains choice-inspection v3 summary, limits, profile, implementation/trace/tape digests, terminal state, sites and replay decisions. Integers are native JSON integers; `site_missing` is explicit; decision arrays and tape digest are empty when exact replay is unavailable. |
| `artifact.world` | Null or authenticated profile/implementation, seed, time, transition count, terminal kind/detail and bounded history payload descriptor. This exposes the World facts carried by the new lineage. |
| `artifact.simulation` | Null or profile, controller/execution/candidate/outcome/failure digests and bounded plan/record payload descriptors. Descriptors add the contained `file` and `limit`; absent failure digest is null. |
| `artifact.minimization` | Null or typed parent/candidate identities, attempt/forcing bounds, ordered accepted reductions and full predicate. The Rust detail schema is `gomad.minimization-evidence/v1`; all digest strings are bare hex. |
| Optional sections | `choices`, `world`, `simulation`, `minimization` are explicit nulls when absent. Unknown fields in retained typed details are refused. |

The cluster projection is an additive Rust v9 extension. Its instance catalogue,
stream descriptors and byte budgets authenticate before either JSON or text is
rendered. Configured and retained handles bind interruption ownership when typed
cluster replay inputs are present; duplicate instances and orphan payloads refuse.
This projection changes cluster JSON output; final two-run raw/golden receiving
lanes must review the changed candidate after its identities settle.

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
returns 1. Authenticated inputs with an incompatible implementation, identity or
grammar return 2; unsupported storage lineage also returns 2. Corrupt manifests,
missing or unauthenticated payloads and execution infrastructure failures return
3. Integrity failures identify `runner_failure: artifact integrity` before any
guest is created. `--toolchain-root` retains its native-only refusal.

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
  --replay-cluster-baseline /absolute/pinned-pre-v4-go-and-helper-reference \
  --replay-cluster-source /absolute/extracted/pre-v4/source \
  --replay-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --replay-preparation-root /absolute/retained/replay-toolchain-preparation \
  --output-root /absolute/new/replay-check
```

The standalone replay/divergence and network baselines stay frozen. Cluster
and registered inspection lanes use the pinned pre-v4 Go launcher, matching
helper and source archive declared as `cluster_reference` in
`testdata/parity/replay-corpus.json`, because fn235 deliberately changes cluster
failure semantics and inspection World commitments. The lane refuses any other
launcher, helper or archive bytes, and authenticates the extracted source tree against
that archive (revision, contents, modes, links and no unarchived files) before use;
`replay-source-pin.py --selftest` proves each modification refuses. The newest Go launcher records a complete v4
choice trace but its artifact validator still requires v3, so it exits 3 before
publication; that refusal is retained as regression evidence, not normalized. The original failing registered guest remains a failure case; a
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

## Deadline backstop

Every linked execution, whether standalone, cluster, simulation, replay or
inspection, arms a process-wide fail-stop backstop before compilation and
keeps it armed through store teardown. The epoch watchdog and cooperative
cancellation act first. If an armed execution remains in its window 5 seconds
(`CLEANUP_GRACE`, Go's cluster stop grace) after its wall deadline, the CLI
exits with infrastructure status 3 and `gomad: runner_failure: ... deadline
backstop ended the process`, without joining stalled slot threads. This ends
every slot of that invocation and starts no subprocess. Published records stay
authenticated; an interrupted campaign continues with `resume`. Fixture builds
alone accept `GOMAD_FIXTURE_BACKSTOP_STALL=compile|teardown`, with
`GOMAD_FIXTURE_BACKSTOP_STALL_WINDOW` and `GOMAD_FIXTURE_BACKSTOP_STALL_FILE`,
which stall one armed window forever. The campaign lane observes the
backstop's bounded exit, the absence of surviving processes, the authenticated
committed prefix and its resume from outside the process.

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
all four operations. The frozen Go WASM backend has no positive shard/merge
surface: its public CLI does not register `plan`, `execute-shard` or `merge`
(status 2, `unknown gomad command`), and the unchanged original Runner API,
called through a fixture-only adapter, refuses the WASM provider's prepared
bundle layout (status 3, `prepared campaign target is outside its private
preparation directory`). The lane verifies both refusals exactly on every run.
Conductor decision 2026-10-21: Rust `execute-shard` and `merge` are compared
against the fresh unsharded Go public `explore` view (ordinal, seed, outcome,
termination and failure group), which is what a correct shard/merge must equal.
This is not a claim that Go shards or merges WASM campaigns, and no Go policy or
preparation owner is changed to manufacture one.

The mandatory lane also includes typed scenario public execution and exact
replay, with malformed-input refusals and independently authenticated retained
reports. `campaign-scenario-check.py` can run this Rust scenario check directly
with `--binary`, `--input`, `--go-compiler`, `--preparation-root` and `--work`.
`campaign-scenario-reference-check.py` independently authenticates fresh original
Go public-CLI captures and compares lifecycle steps, guest output bytes and
handled assertion reports with retained Rust scenario runs.

Raw record, payload, plan and journal integrity checks run independently of
goldens. `GOMAD_GOLDEN_REGENERATE=1` requires repeated fresh agreement and a
byte-identical second regeneration. Host-only journal interruption controls
require `determinism-fixture` and are absent from ordinary release binaries.

## Rust choice and simulation exploration

`explore --strategy=choice-exploration` explores helper-validated forced prefixes
in bounded breadth-first rounds. Select exactly one seed and provide positive
`--max-executions`, `--max-choice-depth` and `--max-exploration-bytes` bounds.
`--choice-start-ordinal` optionally skips earlier branching decisions.
`--site-scope=PACKAGE[,PACKAGE...]` restricts forcing to Go package paths or
prefixes ending in `/...`. For example,
`--site-scope=go.temporal.io/server/service/history/...,net/http` includes those
packages and history's subpackages. The flag requires choice exploration;
empty patterns, whitespace and invalid import paths return status 2 before
execution. Scoped targets require symbol information. Unavailable symbols are
out of scope; other decisions remain seed-driven and recorded for replay.

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

Scoped campaign plans retain `site_scope.patterns`, `resolved_sha256` and
`resolved_count`. Each round reports `forced_in_scope` and `skipped_out_of_scope`;
an unmatched base trace stops with `site_scope_empty` and retains its base
artifact. Select decisions use their site package; runnable decisions use
alternative goroutines' creation packages when the trace carries creation sites.
Older traces without creation sites leave runnable decisions out of scope.
`resume --site-scope=PATTERNS CAMPAIGN` checks the persisted scope rather than
changing it; omission restores it. Changed patterns or resolved site digests
refuse with the campaign changed-bound error.

The reusable choice plan configuration validates scope blocks when a campaign
opens. Public Rust `plan --output FILE` accepts the exploration flags and target
grammar. Seed plans prepare and retain the target with zero guest executions.
Choice plans accept `--site-scope`; resolving a scope executes and retains exactly
one base execution. This lane cannot satisfy a prepare-only qualification gate.
An empty scope still retains the base artifact and records `site_scope_empty`.

`plan --json` returns `gomad.plan-result/v1` with `plan`, `plan_sha256`, `campaign`,
`module_sha256`, `guest_executions`, `envelope`, `preparation_receipt` and
`preparation_receipt_sha256`. The choice `envelope` reports the resolved scope,
base-execution count and stop reason; seed plans return null there. The plan and `FILE.preparation.json` are exclusive,
mode-0600 canonical JSON publications. Existing destinations are refused.
The `gomad.rust-preparation-receipt/v1` receipt binds the exact portable plan,
authenticated preparation record, module bytes and digest, and complete retained
`gomad.rust-wasm-provenance/v1` payload. Provenance retains compiler/runtime/source
inventories, build key, source package, directory, tags, arguments, configuration,
namespace/model/schema inputs and implementation identities. Receipt admission
recomputes these retained bindings through pinned handles without reopening build
inputs. The command vector, numeric exit, elapsed preparation observation and
operator PID are retained. Raw process resource snapshots bracket target
preparation before any scoped base execution. Compiler CPU is the checked
waited-child CPU delta; `compiler_lifetime_peak_rss_bytes` is the raw cumulative
waited-child high water, with both snapshots retained. Unavailable observations
remain null. These observations do not reset qualification phase counters.
The synthetic `prepared` record is not an executed success or replay, and the
receipt makes no measured qualification or cost claim.
Retained admission also checks that provenance source kind, package, absolute
working directory and sorted unique build tags are well formed, and that final
arguments, scenario, network and assertion policy agree with their retained
configuration and plan bindings.

Linked qualification tooling derives complete policies through
`install::config::temporal_policy` with explicit `TemporalPolicy::Qualification`
or `TemporalPolicy::Inspection`. Qualification retains the captured SQLite schema,
frozen clock, entropy key and bounds with null policy arguments and no network.
Inspection supplies its separate network, debug environment, zero entropy and
output/transcript limits. Public Temporal installation uses
`TemporalPolicy::Installation`; the tooling can pass a derived configuration to
public `install --config FILE`.

The mandatory exploration lane runs through the behavioral harness, separately
from the seed campaign lane:

```sh
python3 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb explore \
  --exploration-binary /absolute/host-fixture-enabled/gomad \
  --exploration-divergence-binary /absolute/explicit-helper-fixture/gomad \
  --exploration-conformance-helper /absolute/current/conformance-helper \
  --exploration-baseline /absolute/authenticated/go-baseline \
  --exploration-source /absolute/extracted/baseline/source \
  --exploration-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --exploration-input /absolute/replay-fixture-input.json \
  --exploration-go-base /absolute/original-go-wasi-request.json \
  --exploration-preparation-root /absolute/stable/exploration-preparation \
  --output-root /absolute/new/exploration-check
```

`--verb resume` selects the same complete lane. Exploration routing takes
precedence over `--campaign-binary` for these two verbs. All listed inputs are
required; missing roles or reference inputs fail before execution. Use a fresh
preparation root for the current checkout; both simulation runs reuse that exact
root and compare observations and all eight raw simulation record digests.

The lane requires nine choice and twelve simulation controller cases, actual
public choice exploration against fresh Go CLI results, two six-dimension live
simulation runs with eight artifact replays each, scenario startup against fresh
original Go API executions, and real typed divergence inspection. The divergence
binary explicitly enables the helper fixture role; its mode-sensitive Go guest
uses real runtime decisions. The ordinary role must refuse that fixture without
committing an execution. This fixture proves inspect projection, not ordinary
workload qualification.

Frozen Go's public combined simulation CLI refusal is retained as a boundary
receipt. The scenario API comparison and required current Go Namespace, Volume,
network adapter, network cluster and broker conformance tests are separate
evidence; they do not establish positive combined Go CLI parity. Receipts and the
final result are written under the fresh output root. The inexpensive routing
check is `python3 tools/gomad/testdata/parity/exploration-lane.py --selftest`.

## Rust minimization

```sh
gomad minimize [--json] [--resume] [--attempt-budget N] \
  [--site-scope PACKAGE[,PACKAGE...]] [--artifacts DIR] [--max-bytes SIZE] ARTIFACT_DIR
```

The attempt budget defaults to 64. Output lives under `DIR/minimized`, with
`DIR` defaulting to `.gomad/artifacts`. `--max-bytes` accepts a positive byte
count or `KiB`, `MiB`, `GiB` suffix; omitting it leaves that optional bound unset.
`--resume` continues the artifact's authenticated minimization checkpoint.
Eligibility requires a standalone cooperative target failure with complete exact
choice evidence. Cluster and simulation-profile minimization retain their public
refusals; the six-dimensional controller corpus does not add those capabilities.

`--site-scope` accepts the same package paths and `/...` prefixes as explore.
Without the flag, minimize inherits a valid scope from the parent's authenticated
`provenance/search-context.json`, including after the artifact is copied away
from its campaign. Legacy artifacts without that payload are unscoped. An
explicit flag replaces the selected patterns; malformed present provenance
still refuses with status 2. Scoped minimization requires retained module symbols.
Changing the selected scope or its authenticated source refuses resume with
status 3 before another attempt.

The controller tries removals containing only out-of-scope decisions first,
then mixed removals, then removals containing only in-scope decisions. Order
within each category stays stable. Public standalone choice minimization keeps
its existing increasing-prefix search and suffix removals. Scope changes its
checkpoint binding and reports, while the runtime acceptance and exact replay
checks stay the same.

`--json` returns `artifact_path`, `record_hash`, `changed`, `attempts`,
`attempt_budget`, `accepted` and `stop_reason`. A changed result refers to the
published decorated artifact. Adding minimization evidence changes its full
record and seed-excluded failure signature. The immutable
`predicate.failure_signature` retains the failure signature used to admit
candidates; `parent_failure_signature` separately preserves the parent's full
failure signature. These identities must not be compared as though decoration
left the complete record unchanged.

Each accepted reduction in `minimize --json` also includes `removed_in_scope`
and `removed_out_of_scope`. Their sum is the number of removed decisions.
When a scope is selected, `site_scope` contains `patterns`, `resolved_sha256`
and `resolved_count`, as in explore. Unscoped results omit `site_scope`, report
zero in-scope removals and count every removed decision as out of scope.

For minimized artifacts, `inspect --json` adds a `symbol` to each removed
decision and `final_forced_by_symbol` to `artifact.minimization`. Resolved groups
contain `package`, `function` and `count`, sorted by package and function;
unavailable groups contain `unavailable` and `count`, sorted by reason after
resolved groups. Counts sum to `final_forced_decisions`. Reporting uses the
retained module and authenticated original source tape, so neither the parent
artifact nor its campaign or source checkout is needed. Legacy missing joins
report `site_missing`; stripped modules report `no_symbols`. Inspection does
not change retained minimization evidence or record identities.

The mandatory local lane runs fresh public Go and Rust minimizers against
retained cooperative failures, using the authenticated frozen Go baseline:

```sh
python3 tools/gomad/scripts/behavioral-parity \
  --corpus tools/gomad/testdata/parity --verb minimize \
  --minimize-binary /absolute/host-fixture-enabled/gomad \
  --minimize-baseline /absolute/authenticated/go-baseline \
  --minimize-source /absolute/extracted/baseline/source \
  --minimize-go-compiler /absolute/pinned/go1.27.1/bin/go \
  --output-root /absolute/new/minimize-check
```

These inputs are required. Optional `--minimize-go-cache` and
`--minimize-preparation-root` select reusable cache roots;
`--minimize-golden FILE` compares the reviewed semantic projection.
The lane authenticates each lineage's retained bytes before comparing semantic
results. Runner provisioning of the frozen baseline, extracted source and Go
compiler remains necessary before this lane can run in CI.

## Rust qualification

The public grammar is unchanged:

```sh
gomad --installation /absolute/rust-installation.json qualify --spec /absolute/spec.json
gomad report --result /absolute/root/rust-qualifications/v1/result-HASH.json
```

`qualify-set`, `merge-set` and `compare-support` keep their WASM refusal
(status 2, unsupported command). Grammar errors and unreadable, malformed or
`Options`-bearing specs return 2; freeze, run and publication failures return 3;
otherwise the exit status is the existing qualification classification
(`qualified` 0; `target_failure`, `nondeterministic`, `replay_divergence` 1).

The spec keeps the owner field names (`Manifest`, `Campaign`, `ArtifactRoot`) so
one file drives either binary. Strict decoding refuses duplicate members,
trailing data, nesting over 128 and unknown fields. `Campaign` accepts
`Target{Backend,Kind,Source,Args,BuildTags,WorkingDir,PreparationRoot}`,
`Environment`, the nanosecond `ExecutionTimeout`, `OverallTimeout` and
`TerminateGrace`, `OnFailure`, `FailureBudget`, `OutputLimit`,
`WorldTransitionLimit`, `ChoiceTraceLimit`, `SuccessBytesLimit`, `Artifacts` and
`RunnerBuild`. Every other Go `CampaignSpec` field is refused rather than ignored.
`PreparationRoot` must be absolute; the Rust installation owns its preparation
cache. `RunnerBuild` has no Rust meaning, because records bind the linked
implementation identity. `Target.Backend` must be `wasm` and `Kind`
`go-run`/`go-test`. `Target.Scenario` carries a multi-instance cell as Go's
base64 encoding of the same bounded scenario JSON that `explore --scenario FILE`
reads. The cell must declare `Workload: "multi-instance"`, and its Manifest binds
`scenario_sha256`.

The Go host's public `qualify --spec` refuses typed scenarios with
`invalid_input` (status 2), because its owner forces cooperative choice coverage.
Go qualifies clusters only through its in-package test API. Admitting the cell
on the Rust verb is therefore a deliberate availability change, made because the
Rust binary has no test-API entry point. The campaign maps onto the public `explore`
grammar and inherits its validation. Qualification then fixes the selection:
seed `Manifest.Seed`, one slot, `keep-successes=all` with one success artifact,
and, for a cooperative installation without a scenario, choices with at least
1 MiB and `coverage=choice`. Every declared deadline and limit
(`ExecutionTimeout`, `OverallTimeout`, `TerminateGrace`, `OutputLimit`,
`WorldTransitionLimit`, `ChoiceTraceLimit`, `SuccessBytesLimit`) is passed
through unchanged, and the installation's fuel and memory bound each store. A
record's `limits.wall_millis` is the declared execution timeout.

Freeze runs one actual preparation and binds Rust-lineage identities in
`gomad.rust-qualification-manifest/v1`:

- integrated source;
- workload source and runtime (compiler/overlay) inventories;
- compiler, linked helper and host implementations;
- model, engine, configuration, module and toolchain build key;
- host OS/arch, fuel, memory, seed and repeat.

A declared `sha256:` identity that differs fails closed. A final
(non-provisional) cell also needs an operator baseline, a `success`/`canary`
cell, and the conductor's source manifest, whose listed files are re-read and
authenticated.

Run reuses that frozen preparation. Each of the `Repeat` repetitions is a fresh
single-seed campaign whose execution runs in a fresh Wasmtime engine and store.
Every retained artifact, success or failure, is then replayed in a fresh store
through the shared authenticated opener. Determinism is equality of canonical
`gomad.execution-record/v2` hashes. A failed repetition, a missing artifact or a
replay error ends the cell with an explicit `failure` (classification, message,
iteration), and the completed executions are retained.

### Results and receipts

The Result `gomad.rust-qualification-result/v1` and the receipt
`gomad.rust-qualification-receipt/v1` are canonical JSON. They are published
content addressed with no-replace rename under
`ArtifactRoot/rust-qualifications/v1/`, as
`result-SHA256.json` and `result-SHA256.json.publication-RECEIPTSHA256.json`;
the receipt is addressed by its own bytes. A collision is reused only when its
bytes are identical, and a second, different receipt for one Result is refused. stdout is the bounded
`{"operator_receipt":{...},"result_path":"..."}` response (8 KiB, absolute path
of at most 1024 bytes).

JSON differences from the Go host, by design of the new lineage:

- The receipt fields are `result_sha256`, `operator_process_id` and
  `linked_after`.
- `report` emits `{"classification","publication","receipt","result"}`
  instead of the bare Result. `publication` holds the receipt's terminal
  `cpu_nanos`, `peak_rss_bytes` and a `status` of `within`, or `unattributed`
  when the terminal linked peak exceeds the operator-only 2 GiB ceiling.
- `report` requires exactly one receipt and refuses a missing, duplicate,
  readdressed-in-place, unbound or wrong-process receipt.
- Costs carry integer `{numerator,denominator}` ratios instead of floats.

Results and receipts are content addressed, not signed. `report` refuses any
edit that is not consistently re-hashed and recomputes every verdict, cost and
budget input; a writer who rewrites records, Result and receipt together can
still forge identity fields that do not feed the verdict.

`report` reads no installation, source, compiler or helper. It refuses a changed
content address, non-canonical bytes and unknown fields. It also refuses any
verdict, record hash, cost or budget claim that differs from its recomputation
over the retained records and raw observations. The Result retains the
prepared target's `gomad.rust-wasm-provenance/v1` preimage (`provenance`), and
`report` recomputes every manifest identity from it: workload sources, runtime
and compiler, helper and host, model, engine, configuration, module, build key
(toolchain), scenario, profile and limits. The provenance's duplicated
scenario, network and configuration must agree with its hashed build inputs
(the configuration differs only in the cooperative runtime implementation), and
a cooperative choice identity (`wasip1`/`wasm`) must bind the build key, the
module and the configuration's runtime implementation, which `report`
recomputes from the build key and the choice sources compiled into this lineage; a controlled
configuration carries no runtime implementation. A matched cooperative replay
records `exact` choice replay and a matched controlled one `none`; neither
profile records the other's value. Every record
must bind the frozen
module, toolchain, configuration, limits, implementation (recomputed from the
helper and host digests), platform (`host_os/host_arch`) and exact runtime
profile (`cooperative` is `gomad.wasi-cooperative/v1`, `controlled` is
`gomad.wasi-stock-exploratory/v1`; any other is refused). A standalone record
must also carry exactly the retained configuration's limits, arguments and
environment, as replay preflight requires; a cluster record carries the
retained limits except for an evidence bound of exactly the configuration's
bound times the number of incarnations the retained scenario starts or
restarts; the scenario's `scenario_evidence_bytes` bounds oracle evidence. Each
execution's samples, and each matched replay's, must measure exactly the trace
bytes its record's trace streams retain. A success outcome is a normal exit with status zero,
and any other zero exit is a target failure with soft-assertion evidence, as
the campaign ledger requires. The provenance carries exactly its declared
fields at every typed level, and its build inputs exactly theirs. Every store sample,
executed or replayed, must carry the frozen guest limit, and only
multi-instance stores carry `node/incarnation` handles. Each measured cluster
window, executed or replayed, carries exactly the incarnations its record's
evidence streams name (`evidence/cluster/NODE/INCARNATION/...`), and every
cluster execution is measured. A retained failure
carries one of the runner's failure classes, never a success class.

### Linked-process accounting

The Go contract measured a separate helper process for each execution. The Rust
host links the helper, so the operator, every Wasmtime engine and every guest
store share one process self figure (`linked`, `RUSAGE_SELF` CPU, Linux `VmHWM`
peak). Only the Go compiler and its descendants run as children (`children`,
`RUSAGE_CHILDREN`). The Result retains raw `start`, `prepared` and `executed`
snapshots of both. It also retains, for each store, the wall time, fuel, guest
linear-memory peak and limit, trace bytes (the Go owner's WASI transcript,
choice, diagnostic, storage and network traces; a cluster's shared network
trace counts once, on its first store, and never per store) and virtual time, plus the self
counters at the store's boundaries.

- **CPU** is the children delta plus the linked delta, each counted once. Store
  boundary counters only order the stores and prove they ran one at a time;
  they are never added. There is no helper CPU field to double-count.
- **Exact limits:**
  - preparation wall, 10 min;
  - largest execution or replay store wall, 15 min;
  - largest waited compiler child peak, 6 GiB;
  - each guest linear-memory peak, at most its instance limit.
- **Phase attribution (Linux).** `qualify` resets the image high water at each
  phase boundary (writes `5` to `/proc/self/clear_refs`, then reads `VmHWM`).
  The reset restarts `VmHWM` at the current RSS, and glibc keeps freed heap
  resident, so between reading a phase's high water and the reset, `qualify`
  returns freed heap to the kernel (`malloc_trim(0)`). Otherwise the next phase
  would start from the previous phase's freed memory, for example a store
  window's compiler and guest heap charged to the operator. Memory that stays
  live across a boundary is still charged to the phase that holds it.
  `fixture-qualify-phases` proves both in a fresh process
  (`goldens-qualify/phases.json`).
  This separates operator phases (everything outside guest stores) from each
  fresh store window (Wasmtime compilation, instantiation and guest execution).
  Each phase is mapped to the ceiling that bounded it in the Go host
  (`internal/backends/wasm/QUALIFICATION.md#measured-costs`, Go
  `qualification.EvaluateBudget`):

  | Phase | Historical ceiling | Check |
  | --- | --- | --- |
  | Operator phases | fresh operator lifetime peak, 2 GiB | exact |
  | Single-guest store window | cold-cache-miss helper peak, 6 GiB | exact |
  | Single-guest cache-hit store window | warm-cache-hit helper peak, 4 GiB | exact |
  | Cluster store window | resident helper peak, 8 GiB | exact |

  The cold ceiling governs every single-guest window. Each sample retains the
  helper's `cache_hit`: a window that loaded its compiled module from the
  installation's compile cache is a warm window, and the warm ceiling is checked
  exactly against the largest warm window. Executions after the first and the
  replays, which open the installation's compile cache as the `replay` verb does,
  are warm. When no window reports a cache hit, the largest window peak is only
  an upper bound of the warm phase: within under 4 GiB, unattributed otherwise.
  As in the Go host, cluster windows are not subject to the single-guest limits. The
  receipt's terminal operator check uses the operator phases through
  publication.
- **Attributed phases account for the lifetime peak.** Every window lies within
  the lifetime linked peak and within the lifetime peak observed after it; any
  rise across a window's boundaries is explained by the window or by operator
  work; any other rise (preparation, between windows, after the last window)
  is explained by operator work alone; the operator figure covers the initial
  peak, because operator phases begin before the baseline snapshot; and the
  largest phase reaches the lifetime peak. Otherwise the budget is refused
  (`budget_failure`). Even when attribution is incomplete, every measured
  window must fit the lifetime peaks around it before the fallback is used,
  and a measured phase that already exceeds its ceiling stays `exceeded`: a
  missing measurement never erases a known violation (the receipt's terminal
  check keeps a retained or receipt operator overrun the same way). A zero phase peak is a missing measurement. A
  window whose stores carry `node/incarnation` handles is a cluster window,
  even with a single node.
- **Without attribution** (another platform, a refused `clear_refs` write, or any
  window lacking its peak), the lifetime linked peak upper-bounds every component.
  It proves a limit only when it fits under it. Above a limit, the check is
  `unattributed` and listed in `pending`: it is neither charged to one owner nor a
  pass.
- **Concurrent stores.** A cluster execution or replay retains one sample per
  incarnation store (`node/incarnation`). Each sample carries that store's own
  fuel, trace bytes, guest linear-memory peak and limit. All of them share one
  wall time, virtual time and boundary window. The checks:
  - each store's guest peak is checked exactly against its own limit;
  - fuel and trace bytes are summed over the stores;
  - wall time is counted once per logical execution;
  - `resident_stores`, the most distinct nodes in one window, is checked exactly
    against the protocol's two resident guests.

  A missing handle, a duplicate handle, mismatched windows or a wrong resident
  count refuse cost and budget. Host CPU and RSS cannot be split per store, so
  they stay linked-process figures under the upper-bound rule above.
- **Missing observations** produce `cost_failure` or `budget_failure`, never a
  measured zero.
- **Final cells** return 3 for any failure, violation or pending check, after the
  Result and receipt are retained.
- **The receipt's terminal linked peak** must fit the 2 GiB ceiling, or
  `qualify` returns 3 for every cell, as the Go host did. `report` still opens
  such a Result and labels the receipt `unattributed`. Missing or backward
  counters and binding errors are refused. With attributed phases, a terminal
  lifetime peak above both the Result's lifetime peak and the terminal
  operator phase is unexplained and refused.

### Verification lanes

The controlled accounting corpus drives the same evaluators with fixture
observations. It checks CPU counted once, threshold crossings, unattributed
peaks, missing measurements, overlapping or concurrent stores, receipt tails and
bug yield against an independent oracle and
`testdata/parity/goldens-qualify/accounting.json`:

```sh
python3 tools/gomad/testdata/parity/qualify-accounting-check.py \
  --binary /absolute/fixture-enabled/gomad --work /absolute/new/accounting
```

The mandatory fresh Go/Rust lane runs it, then `qualify-lane.py`:

```sh
python3 tools/gomad/scripts/behavioral-parity --corpus tools/gomad/testdata/parity \
  --verb qualify --output-root /absolute/new/qualify \
  --qualify-binary /absolute/fixture-enabled/gomad \
  --qualify-go-binary /absolute/frozen/bin/gomad --qualify-go-helper /absolute/frozen/bin/gomad-wasmhost \
  --qualify-go-source /absolute/frozen/source --qualify-go-compiler /absolute/stock/go1.27.1/bin/go \
  --qualify-input /absolute/replay-input.json --qualify-preparation-root /absolute/stable/preparation
```

Parity runs use lane-local budgets (`LANE_CAMPAIGN` in `qualify-lane.py`): a
300 s execution and 20 min overall deadline, 8 MiB output, 64 MiB World
transitions and the lane installations' fuel. These budgets are deliberately
wide, so that shared host load cannot turn a verdict comparison into a watchdog
result. They are not qualification budgets. Every cell then runs again as a
final canonical cell on a quiet host (1-minute load below 2.5), with the owner's
declared values unchanged and an installation pinned to the owner's fuel and
memory:

| Cell | Owner source | Execution / overall / grace | Output / World | Fuel / memory |
| --- | --- | --- | --- | --- |
| `fixture`, `fixture-canary` | `qualification/integration_test.go` | 60 s / 3 min / 100 ms | 1 MiB / 1 MiB | 4e9 / 256 MiB |
| `sqlite` | `qualification/sqlite_test.go` | 60 s / 3 min / 100 ms | 1 MiB / 1 MiB | 4e9 / 256 MiB |
| `cluster` | `qualification/cluster_test.go` bounds | 60 s / 3 min / 100 ms | 1 MiB / 1 MiB | 4e9 / 256 MiB |
| `onebox` | `qualification/onebox_test.go` | 15 min / 30 min / 1 s | 256 MiB / 32 MiB | 2^50 / 4 GiB |

The onebox parity and canonical runs both use its owner values, plus choice
traces of 64 MiB, success bytes of 2 GiB and the owner environment. Each canonical
cell runs twice in fresh processes and roots, and both stable views must agree.
Each canonical installation also pins the owner's `wasi.Config` model and
namespace limits (calls, transcript bytes, pending events, descriptors, files,
filesystem and output bytes; `OWNER_LIMITS` in `qualify-lane.py`), and the
retained configuration must carry exactly those values.
Each run must retain `limits.wall_millis` equal to the owner execution timeout
and pass the unchanged cost/budget gate with its declared status: no cost or
budget failure, no violation, no pending check and an attributed terminal
operator peak within 2 GiB.

For each cell (default `fixture`, `fixture-canary`, `sqlite` and `cluster`;
`onebox` is selected with `--qualify-cells onebox`), the lane runs one Go and two Rust
qualifications. The Go reference for `cluster` records Go's public `qualify`
refusal. Its verdict instead comes from two fresh Go `explore --scenario`
campaigns and a fresh `replay` of each retained artifact. It authenticates each Rust Result, receipt,
record hash and retained artifact on raw bytes. Both Rust runs must agree on the
declared stable view, and the Go and Rust verdicts must match. The lane also runs
`report` without an installation and exercises the grammar, spec refusals
(including a multibyte character at a source-manifest separator) and the tamper
refusals in `goldens-qualify/tamper.json`, among them in-place and duplicate
receipts, readdressed implementation, platform and profile changes, a failure
claiming success and a malformed retained source manifest. The stable
view compares cell identities, records, replays, outcomes, verdicts, structural
costs, store handles and budget statuses. Its kept and excluded field sets are
declared at every level (Result, report, executions, observations, cost,
samples, budget and checks); any other field refuses the projection. Each
retained campaign journal is authenticated and must commit exactly the Result's
execution record. It excludes the measured PID, CPU, RSS and wall
values and replaces the campaign IDs and installation/spec paths with tokens.
Goldens are per cell (`goldens-qualify/cells/NAME.json`, with the parity view,
the canonical view and the Go verdict) plus `goldens-qualify/tamper.json`.
Regenerate them with `GOMAD_GOLDEN_REGENERATE=1` twice and compare the bytes.
Linked-process upper-bound statuses follow measured RSS, so the golden keeps
their names and limits but not their statuses. Each run authenticates them
through the binary's recomputation.

The frozen Go reference `fn232-replay-294-reference` predates the fn-230 R12 v4
host repair, and its replay of a one-box Temporal trace fails
(`choice trace malformed`). The `onebox` cell therefore runs as its own
invocation (`--qualify-cells onebox`) against a Go reference built from a
v4-repaired `gomad` revision. On this host the Rust one-box cell qualifies
semantically, matching Go's verdict. Before phase attribution, its lifetime
linked peak of 6,203,392,000 bytes (5.777 GiB, 6.20 GB), almost all in-process Wasmtime compilation (the guest
peak is 141 MiB), left the operator and warm checks unattributed, and the
command exited 3. With phase attribution, the operator phases and store windows
are checked against their own ceilings. The attributed onebox run is pending the
conductor's execution window. These are provisional fixture-binary lanes. The onebox row is the corpus family
cell (`./tests/gomadfunctional`, `TestFrontendSystemInfo`). The mandatory
`unchanged-workflow-type-smoke` selection and every one-box gate remain with
fn-222.1. Qualification on the release binary alone remains with fn-232.13.

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

## Rust installation and discovery

The linked Rust host accepts the existing installation grammar:

```sh
gomad install --compiler /absolute/go --cache /absolute/cache --output gomad-wasm.json \
  [--helper /absolute/gomad] [--runtime-root /absolute/runtimesource] \
  [--compile-cache /absolute/cache] [--profile PROFILE] [--go-version go1.27.1] \
  [--config FILE] [--memory-bytes BYTES] [--fuel COUNT]
gomad install --compiler /absolute/go --cache /absolute/cache --output gomad-wasm.json \
  --runtime-root /absolute/runtimesource --temporal-checkout /absolute/temporal \
  --temporal-package ./tests/gomadfunctional
gomad [--installation FILE] discover
gomad doctor [--artifacts DIR] [--json]
```

`install` publishes a new, exclusive mode-0600
`gomad.rust-wasm-installation/v1` manifest. Its `admission` block retains the
original guest configuration and digest, complete compiler/runtime/overlay input
inventory and digest, actual host executable digest, and any supplied helper
path/digest. The existing Rust manifest fields retain the actual compiled host,
linked helper and runtime implementation identities. Go-lineage manifests are
refused. Configuration JSON uses the existing WASI configuration grammar;
Temporal setup derives the complete TemporalInstallation policy and guest tags.
Source preparation replaces configuration argv with the requested target argv,
as the retired Go Runner did, and preserves its configured environment.

The linked implementation is the default, so `--helper` is optional. An explicit
path must name the running Rust host executable or a byte-identical copy of it.
The installer authenticates its complete executable bytes and records that digest;
it never executes the supplied path. A standalone helper, legacy Go helper,
modified executable or differently compiled role is refused. This is the explicit
linked-host default/requiredness exception; it does not select another execution
implementation.

Installation validates the existing pinned Go compiler and invokes the retained
Go overlay generator through it. `GOTOOLCHAIN=local`, `GOPROXY=off` and
`GOSUMDB=off` prohibit downloads. The manifest pins the overlay generator's source
owner as well as runtime source and generated replacement bytes. Preparation
revalidates the pinned inputs before building a target.

Resolution chooses a nonempty leading `--installation`, then
`GOMAD_WASM_INSTALLATION`, then `gomad-wasm.json` beside the executable. A chosen
invalid installation fails without fallback. `discover` emits
`{path, source, sha256, manifest, capabilities}` and authenticates the manifest's
compiled identities and configuration preimages without opening compiler/runtime
build inputs. It therefore works after those inputs are removed. Its capability
list includes the newly available Rust `plan` verb. Its `ChoiceProfiles` retains
the current `gomad-choice-trace/v4` wire format received from fn-230; the frozen
Go reference advertises `gomad-choice-trace/v3`. Both literal capability objects
remain in the differential corpus.

Default `doctor` preserves the unavailable report (`gomad.doctor/v3`, status 2).
An installation-selected `doctor` and native-only `analyze` retain the actual Go
WASM unsupported-command refusal (status 2). Invalid installation inputs return
2, exclusive manifest publication failures return 3, and selected discovery
failures return 3.

`recover [--json] CAMPAIGN` authenticates and repairs existing seed or exploration
round journals under their writer lock. It starts no guest. The Rust recovery
JSON retains `{schema, path, action, changed}`; `action` is `repair` after an actual
repair and `none` on a subsequent no-op. Rust journals have no Go lifecycle-state
fields, so those fields are omitted from the Rust response.
