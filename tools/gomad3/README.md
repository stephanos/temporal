# Gomad v3

Gomad v3 is an opt-in Go 1.27.1 toolchain with a small deterministic-runtime
patch and source overlay. It uses native Go goroutines, channels, `select`,
maps, synchronization, `go run`, and `go test`.

Build the argv-safe CLI and cached toolchain from the repository root:

```sh
make gomad3
```

The complete Gomad v3 Runner and deterministic-I/O contract is qualified on
`darwin/arm64` and `linux/amd64`, the platforms the boundary manifest names.
The builder rejects other hosts before starting a toolchain build. Runtime
source may remain portable to other Unix systems, but those systems are not
supported Runner platforms until the manifest qualifies them and they have
their own adapters, publication primitives, and complete test gate. Artifacts
replay only on the platform that produced them.

Use the CLI directly so every package and target argument crosses exactly one
argv boundary:

```sh
tools/gomad3/.bin/gomad explore --seeds 1 go-run ./cmd/example -- arg
tools/gomad3/.bin/gomad explore --seeds 1 go-test ./path/to/package -- -test.run=TestName
```

`make gomad3-runner` is an alias for the same CLI build, while
`make gomad3-go` builds only the toolchain. The older single-seed wrappers
remain for compatibility:

```sh
GOMADSEED=1 make gomad3-run GOMAD3_RUN=./cmd/example
GOMADSEED=1 make gomad3-test GOMAD3_PACKAGES=./path/to/package
```

Those wrappers remove `GOMADSEED` from the custom `go` process and use Go's
`-exec` hook to enable Gomad only in the resulting binary or generated test
binary. They compile with `GOEXPERIMENT=nogreenteagc`, as Runner preparation
does. Direct execution remains `GOMADSEED=<seed> ./binary`; the prebuilt binary
must use that collector profile. Seeded activation of a Green Tea binary
fails before user code runs. Unseeded binaries retain their compiled profile.

Check whether the complete Runner and deterministic-I/O contract is available
before starting a campaign:

```sh
tools/gomad3/.bin/gomad doctor
tools/gomad3/.bin/gomad doctor --json --artifacts .gomad/artifacts
```

The report includes the host, toolchain and Runner build identities, boundary
manifest, I/O implementation, available adapters, artifact-directory access,
resolved installation source, and a location-specific repair instruction.

Every command that executes or verifies a target resolves the pinned toolchain
in the same order: `--toolchain-root`, `GOMAD3_TOOLCHAIN_DIR`, a
`gomad3-install.json` bundle manifest adjacent to the executable or its parent,
then an adjacent `.toolchain` directory. CLI and environment roots must be
absolute, clean, non-root paths. The source builder uses
`GOMAD3_TOOLCHAIN_DIR` too.

`explore`, `plan`, `qualify`, and `analyze` build the target from the current
directory unless `--working-dir` names another module root, which must be an
absolute, clean path to a directory holding a `go.mod`; `qualify-set` has the
same flag. Build adapters are selected from the `go.mod` of the module that
owns the target, so a module outside this repository can depend on the server
through a local `replace` and still be prepared. A read-only mount source may
be an absolute path, such as the server's `schema` directory in that
replacement or in the module cache. Every target is built with
`CGO_ENABLED=0 GOENV=off GOFLAGS= GOTOOLCHAIN=local GOWORK=off TZ=UTC
GOEXPERIMENT=nogreenteagc` and listed with `-mod=readonly`: a workspace file
is not honored, so its replacements must be `replace` lines in `go.mod`;
vendored modules are unsupported; settings made with `go env -w` are ignored,
so `GOPROXY`, `GOPRIVATE`, `GONOSUMDB`, and `GOSUMDB` must be exported
variables.

A standalone bundle can place this manifest beside `bin/gomad`'s parent:

```json
{
  "schema": "gomad3.installation/v1",
  "toolchain_root": "lib/gomad3/toolchain"
}
```

Relative manifest roots are resolved from the manifest directory. Malformed
or unknown manifests fail closed instead of falling back to another location.

Explore `go run`, `go test`, or a prepared executable target, then replay a
retained failure exactly or verify its immutable inputs without executing it:

```sh
tools/gomad3/.bin/gomad explore --seeds 0-999 go-run ./cmd/example -- arg
tools/gomad3/.bin/gomad explore --count 1000 go-run ./cmd/example -- arg
tools/gomad3/.bin/gomad explore --coverage=semantic --keep-successes=novel --success-limit=32 --success-bytes=1GiB --count 1000 go-run ./cmd/example -- arg
tools/gomad3/.bin/gomad explore --guide --corpus .gomad/corpus --count 1000 go-run ./cmd/example -- arg
tools/gomad3/.bin/gomad explore --seeds 0,7,42 go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad explore --seeds 0-99 exec --provenance ./example.provenance.json -- ./example arg
tools/gomad3/.bin/gomad qualify --seed 7 --repeat 2 go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad qualify --seed 7 --repeat 2 --choices --replay-successes --success-limit=1 --success-bytes=128MiB go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad analyze --format=json go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad analyze --capability-mode=linked --timeout=5m --format=json go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad qualify-set --manifest corpus.json --working-dir ./target --output report.json
tools/gomad3/.bin/gomad qualify-set --manifest corpus.json --working-dir ./target --output shard-0.json --shard 0/3
tools/gomad3/.bin/gomad merge-set --manifest corpus.json --output report.json shard-0.json shard-1.json shard-2.json
tools/gomad3/.bin/gomad compare-support --baseline baseline.json --candidate report.json
tools/gomad3/.bin/gomad explore --choices --choice-bytes=8MiB --seeds 0-99 go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad explore --strategy=choice-exploration --seeds 7 --max-executions=128 --max-choice-depth=32 --max-exploration-bytes=64MiB go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad plan --seeds 0-99 --output campaign.plan.json go-test ./path/to/package -- -test.run=TestName
tools/gomad3/.bin/gomad execute-shard --shard 0/4 campaign.plan.json
tools/gomad3/.bin/gomad merge --output merged-campaign campaign.plan.json .gomad/artifacts/v1/campaign-*
tools/gomad3/.bin/gomad recover .gomad/artifacts/v1/campaign-INTERRUPTED
tools/gomad3/.bin/gomad resume .gomad/artifacts/v1/campaign-INTERRUPTED
tools/gomad3/.bin/gomad inspect .gomad/artifacts/v1/campaign-*
tools/gomad3/.bin/gomad inspect .gomad/artifacts/v1/campaign-*/failures/sha256-*
tools/gomad3/.bin/gomad inspect --choices .gomad/artifacts/v1/campaign-*/failures/sha256-*
tools/gomad3/.bin/gomad inspect .gomad/artifacts/v1/campaign-*/successes/sha256-*
tools/gomad3/.bin/gomad replay .gomad/artifacts/v1/campaign-*/failures/sha256-*
tools/gomad3/.bin/gomad replay .gomad/artifacts/v1/campaign-*/successes/sha256-*
tools/gomad3/.bin/gomad replay --verify-only .gomad/artifacts/v1/campaign-*/failures/sha256-*
tools/gomad3/.bin/gomad minimize --attempt-budget=64 .gomad/artifacts/v1/campaign-*/failures/sha256-*
```

Human-readable exploration writes preparation and running progress to stderr,
including attempted, active, successful, failed, watchdog, replay-divergence,
distinct-failure, retained-success, and retained-success byte counts. Its final
result and every retained artifact path with a copy-paste replay command are
written to stdout.

Use `--choices` on `explore` or `qualify` to observe bounded runtime runnable
and select decisions. `--choice-bytes` defaults to 8 MiB, is valid only with
`--choices`, and is part of Campaign and Artifact identity. The trace is
recorded as v3 stable logical decisions. Artifact replay automatically derives
an identity-bound, read-only decision tape and validates every choice before it
is applied; no replay flag is required. `inspect --choices` validates the
retained payload and reports choice kinds, decision and branching counts, the
tape digest, exact-replay availability, and target-specific site fingerprints.
Prefix replay is an internal bounded-exploration primitive and is not a public CLI
mode in this slice.

Use `--strategy=choice-exploration` to explore every non-selected runnable or
ready-`select` rank observed within explicit bounds. The strategy requires one
base seed plus positive `--max-executions`, `--max-choice-depth`, and
`--max-exploration-bytes` values. It implies choice recording and rejects
`--count`, multiple seeds, and guided exploration. Candidates run in
deterministic breadth-first rounds ordered by forced-prefix length and identity;
parallel completion timing cannot change the committed exploration. A target
failure remains expandable while the selected failure policy permits it.

`--choice-start-ordinal=N` keeps decisions before replay-plan ordinal N in each
forced prefix but expands alternatives only at N and later. The default is 0;
use `inspect --choices` to find the replay-plan ordinals. This option is valid
only for Choice Exploration and is frozen in the Campaign plan for resume.
Select polls of the seven proven shapes with two polled non-nil cases and
fewer than two ready cases remain in the Choice Trace but do not create frontier
branches. Unknown readiness, shapes outside
[the proven list](choice/no_op_select.go), and selects with three or more polled
cases stay expanded. Source clauses with nil channels
do not count as polled cases. The result reports their omitted alternatives
separately from execution, depth, and byte bounds.

Each completed round is an immutable, hash-linked transaction below the Campaign.
An interrupted round is archived and rerun in full on `gomad resume`; logical
and recovery execution counts are reported separately. `exploration_exhausted`
and `choice_depth_complete` are bounded completions, while `max_executions` and
`exploration_capacity` identify incomplete search envelopes. Outcome deduplication
reduces retained evidence only and never removes a distinct forced prefix.

`gomad minimize` currently accepts an exact combined-simulation target-failure
artifact. It runs every suffix, forced-range, and fault-entry proposal in a
fresh process under an explicit attempt bound. An accepted reduction must keep
the normalized failure and outcome, exact choice replay, and exact simulation
replay. The parent remains immutable; a changed result is published by record
identity with inspectable parent, reduction, budget, and predicate evidence.
Each parent has persisted minimizer state below the selected artifact root.
After interruption, `minimize --resume` with the same parent, artifact root,
and bounds continues the recorded attempt order and budget. The state binds
the parent, implementation, accepted artifacts, and replay evidence; changed
inputs, corrupt state, or a concurrent writer fail closed. Target-declared
scenario shrinking is not yet implemented.

For the CLI, state lives in
`ARTIFACTS/minimized/.minimize/sha256-<parent record hash>/`, with its lock
beside that directory as `sha256-<parent record hash>.lock`. Different parents
can minimize into the same root independently. A plain run refuses existing
state for its parent; resume refuses a parent without state. Empty lock files
remain after completion.

`gomad analyze` defaults to `--capability-mode=closure`, which reviews a
`go-run` or `go-test` target without compiling or executing it. Explicit
`--capability-mode=linked` builds but never launches the target, extracts the
pinned linker record, and separates live blockers from closure blockers removed
by final reachability. Closure analysis has a 30-second default wall bound and
linked analysis has a two-minute default; `--timeout` can raise either explicit
bound up to 30 minutes for large targets. Linked mode has no closure fallback:
malformed records, identity mismatches, and capacity failures fail closed. Only
guarded mode compiles `-gomadguard` capability guards into the target; closure
and linked modes review without them (see [Contract](#contract)). The
report uses exact compatibility-pack decisions, lists every active and
eliminated blocker with a canonical shortest dependency path, and projects
conservative deterministic I/O requirements over the full closure. To keep
reports path-free, arguments containing path separators are represented by
stable SHA-256 identities.
`--format=json` emits `gomad3.capability-analysis/v1`. Status 0 means
supported, 1 unsupported, 2 invalid input or package configuration, and 3
analysis infrastructure failure.

Add `--json` to emit newline-delimited `gomad3.explore-event/v3` records on
stdout and no routine output on stderr. Event types are `progress`, `result`,
`artifact`, and `error`. Result classifications are `success`,
`target_failure`, `watchdog_observation`, `replay_divergence`, and
`mixed_failure`; error classifications are `invalid_input`,
`unsupported_target`, `semantic_coverage_failure`, `capacity`, `cancelled`,
`overall_timeout`, and `runner_failure`.

Use `--coverage=semantic`, `--coverage=choice`, or
`--coverage=semantic+choice` to retain versioned semantic probes, canonical
choice features, or both. Choice coverage requires `--choices`. Repeat `--require-probe`
to make an unobserved known probe fail the campaign with classification
`semantic_coverage_failure` and status 1:

```sh
tools/gomad3/.bin/gomad explore --coverage=semantic \
  --require-probe=stdlib.os.openfile --count 100 go-test ./path/to/package
```

Use `--guide --corpus DIR` to exclude requested seeds already answered by
replay-verified, matching corpus cases. Guidance enables semantic coverage by
default; an explicit `--coverage` must select semantic or choice coverage, and
`--coverage=none` is rejected. Each Campaign selects from one immutable corpus
snapshot, executes the requested selection minus answered seeds, and substitutes
nothing. When the corpus offers no unanswered seed to prioritize, guidance
selects none. A fully answered request executes zero seeds and exits 0.

Add `--guide-regression` to re-run corpus cases. This mode selects at most three
quarters of its seeds from the corpus and reserves at least one quarter, rounded
up, for the requested seed pool. Corpus cases are ranked by reproducible
failures, invariant and terminal states, abstract World and I/O outcomes,
operation and transition pairs, boundary probes, and smaller reproductions.
World feature values omit seeds, internal identities, logical times, resource
keys, and payloads. The Campaign plan freezes the selection, snapshot, and mode
for resume and shards; shards neither reopen nor update the live corpus. An
explicit conflicting resume mode is rejected. Human and JSON results report
requested, answered, guided, and new execution counts.

The corpus is private, single-writer, and bounded to 1,024 cases and 1 GiB. Its
identity binds the prepared target and arguments, explicit environment and
clock-tick policy, pinned toolchain, reviewed
boundary, semantic instrumentation, and record contract. Every entry retains
the exact-replay artifact, seed, captured I/O and World identities, semantic
coverage, novelty reasons, and matching replay result. A case is published and
replayed before the canonical corpus index advances atomically; interrupted
unreferenced cases are removed when the corpus next opens. A changed identity,
corrupt case, divergent replay, symbolic-link corpus, concurrent writer, or
capacity violation fails visibly. Human and JSON results report the corpus
path, retained entry count, and additions made by the Campaign.

Regression guidance reuses realized seeds and transcripts; it does not mutate
World scenarios, faults, or inputs and never forces runtime choices. Those
extensions require evidence that retained seeds cannot reproduce minimized
failures. Code coverage remains separate from versioned semantic probes and is
not collected by this mode.

Successful Executions are discarded from the Campaign by default; guided corpus
retention is independent. `--keep-successes=novel` retains the first completed
success that adds a new semantic probe or choice feature and therefore requires
semantic or choice coverage; `--keep-successes=all` retains every success. Both modes
require a positive `--success-limit` and `--success-bytes`. Crossing either
bound fails the campaign visibly instead of silently dropping replay evidence.
Each retained success is an immutable exact-replay artifact, and its stored byte
count and novelty reasons are recorded in the Campaign journal. Success replay
returns status 0 only when the recorded successful outcome matches.

`gomad qualify` prepares and executes the target independently two or more
times with one seed, compares bounded canonical evidence, and automatically
retains a private `gomad3.qualification/v1` report below
`ARTIFACTS/qualifications/v1`. Evidence includes the exact target, argv,
toolchain and Runner identities, full output hashes, transcript, captured-mount
identity, World identity, outcome, semantic probes, and optional choice
features. Replay evidence is attached to its corresponding repetition. Add
`--replay-successes` with explicit positive `--success-limit` and
`--success-bytes` bounds to retain and replay every success. `--choices` and
`--replay-successes` are independent opt-in flags, so a `qualified` result makes
up to three claims and the report says which:

- **Same-seed repeatability** is what every `qualified` result establishes:
  each repetition was prepared and executed afresh and their evidence digests
  are equal. Repetitions that differ are `nondeterministic` and a failing target
  is `target_failure`, with or without either flag.
- **Tape availability** needs `--choices` and a retained Artifact. The evidence
  then carries a choice profile with its `tape_sha256`, which the repetitions
  must also agree on, but a success is retained only with `--replay-successes`,
  so without it no Decision Tape is kept to replay.
- **Verified choice-tape replay** needs both flags: each retained success is
  replayed from its tape, and its repetition records a `replay` entry with
  `match` and `choice_replay_status: exact`.

`--replay-successes` without `--choices` still retains and replays each
success, re-executing it from its seed: the `replay` entry then records `match`
with `choice_replay_status: none`, which is a replay of the Artifact and not of
a choice tape. A repetition without a `replay` entry was not replayed, so a
qualification run with neither flag claims repeatability and nothing about
replay. Repeat `--require-probe`
to enforce known conditional probes; `--repeat` is bounded to 2 through 32.
Add `--json` for newline-delimited `gomad3.qualify-event/v1` progress, result,
and error records. Unsupported targets retain their first boundary and exact
command in the qualification report.

Run or validate a versioned qualification manifest explicitly with:

```sh
tools/gomad3/.bin/gomad qualify-set \
  --manifest=/absolute/path/to/manifest.json \
  --working-dir=/absolute/path/to/target/module \
  --artifacts=.gomad/qualification --output=qualification-set.json
tools/gomad3/.bin/gomad qualify-set --check \
  --manifest=/absolute/path/to/manifest.json \
  --working-dir=/absolute/path/to/target/module
```

Manifest v3 binds the expected module, tier, invariant, ordered seeds, choice
capacity, capability mode, successful-replay requirement, and explicit
retention bounds. The
orchestrator analyzes every workload before executing any supported target,
checkpoints after each completed phase, and publishes a private, path-free
`gomad3.qualification-set-report/v1`. Unsupported analysis is completed
evidence and is never executed. An expectation names one classification;
`unrepeatable` accepts either `nondeterministic` or `replay_divergence` for a
workload whose same-seed evidence is still being made to reproduce, so the
report records whichever the run produced without counting it as a surprise.
An `unsupported_target` expectation names its boundary with `import_path` and
`capability`; every other non-qualified expectation names the blocker it
accepts with a `finding` identity, such as the milestone section that records
it, and a `qualified` expectation names neither.
A workload with `choice_bytes` 0 and `replay_successes` false runs without
`--choices` and `--replay-successes`. Its `qualified` seeds establish same-seed
repeatability: the seed reports the `evidence_sha256` its repetitions agreed on
with `replayed` and `replay_match` false, `choice.available` false, and no
`choice_replay_exact`. A workload that sets `choice_bytes`, `replay_successes`,
and both success limits reports `choice.exact_replay_available` when its seed
retained one Decision Tape and `choice_replay_exact` only when that tape was
replayed and matched. A workload that traces choices in a set must also replay
its successes, because the set reads choice coverage from the retained success
Artifacts. The report's `replayed` count and `trace_bytes` total cover the
replayed and traced seeds only, and `dimensions` names the evidence kinds the
report format carries, not what each workload recorded. `compare-support`
reports a seed that replayed in the baseline and not in the candidate as a
replay regression.
Status 0 means all expectations
matched, 1 means a retained mismatch, 2 means invalid input, and 3 means
cancellation, timeout, child, or publication infrastructure failure.

A set that exceeds one machine's budget runs as shards. `--shard INDEX/COUNT`
applies `execute-shard`'s zero-based ordinal-modulo partition to the manifest's
workloads: shard INDEX owns every workload whose manifest position modulo COUNT
is INDEX, so the shards never overlap and together cover the manifest. A shard
publishes an ordinary set report for its own workloads under the whole
manifest's digest, and `merge-set --manifest MANIFEST --output REPORT
SHARD_REPORT...` combines the shard reports into the report a whole run would
have published. It requires every shard to come from the same manifest, module,
platform, toolchain, I/O profile, seeds, and pruning choice, and the shards to
cover each manifest workload exactly once; a repeated, missing, or foreign
workload is invalid input, never a partial aggregate. A count larger than the
manifest is refused rather than run as an empty shard. `merge-set` returns the
same statuses as `qualify-set`.

By default every Campaign a set run produces stays under `--artifacts`.
Campaign, corpus, and minimizer stores share each prepared binary through a
content-addressed target pool and hard links. The corpus counts each pool target
once and private fallback copies separately. Merge counts a target SHA-256 once
as if all evidence occupied one store; it copies no evidence, so separate shard
roots still hold separate pool copies. Each Artifact's stored bytes and a
Campaign's byte limits include its full target, even when linked. Those limits
bound standalone copies and can exceed physical shared storage. Artifact
payloads and replay validation remain complete. `--prune-qualified-artifacts`
(`GOMAD3_QUALIFICATION_PRUNE=1` for the Make targets) bounds that to one seed:
once a seed is `qualified`, its successful repetitions were replayed exactly,
and the set report holds its evidence, the run deletes that seed's retained
Campaigns and keeps its qualification report. Seeds with any other outcome, and
workloads that do not replay successes, keep everything. The set report records `qualified_artifacts_pruned` for the run and
`artifacts_pruned` for each pruned seed, so those seeds cannot be replayed
later; `OpenReport` rejects a pruned seed that is not a replayed, matching,
qualified one. The checkpoint marks a seed pruned before its Campaigns are
deleted, and a pruning failure stops the run with status 3. A run also stops,
with status 3 and the remaining workloads recorded as infrastructure failures,
before a seed would start on an artifact volume with less than
`--min-free-bytes` (2 GiB by default) free, so it never fills a shared disk.

The [historical task 10 measurement](../../.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/retained-bytes.md)
at `b12b15b1c` on 2026-10-02 retained 1,177,358,336 bytes on disk for the
unpruned representative set, versus 11,442,111,182 bytes counting those same
files as private copies. The report's standalone `artifact_bytes` sum remained
11,421,193,338. These are that candidate's measurements, not a current-candidate
qualification or a storage bound for other workloads.

Preparing a go target retains the built binary under
`.toolchain/builds/<key>/prepared-targets/<identity>`, where the identity binds
every build input: the toolchain and its Go settings, kind, package, tags,
capability mode, overlay, module files, the reviewed closure with its source
digests, module sums, and adapters, each dependency module's language version
and a local module's go.mod, and the files packages embed. Builds do not stamp
repository state. A later preparation with the same identity restores the copy
instead of linking again, after the same capability review, and discards a
copy that no longer hashes to its record. The most recently used binaries stay
within 2 GiB, and the entry just prepared always stays. The target build cache
beside it is trimmed to 4 GiB, least recently used entries first, after each
build, and only while no other build holds it.

Compare two validated reports with `gomad compare-support`. Clean and improved
comparisons return 0, regressions or review-required changes return 1,
incomparable inputs return 2, and output failures return 3. A boundary change
prints an exact domain-separated digest; approval applies only when
`--approve-boundary-diff=SHA256` matches that digest. Expectation matching and
actual supported/unsupported counts remain separate.

`make -C tools/gomad3 core-qualification` runs the checked
`qualification/core.json` corpus from its self-contained fixture module. Its
seven assertion-based workloads cover concurrent state invariants, filesystem
lifecycle semantics, loopback TCP request/response, SQLite commit/rollback,
a WAL-mode SQLite database, read-only mount lookups while the collector
cycles, and the direct modernc/libc file boundary. The aggregate and all
evidence are retained below `.toolchain/core-qualification*`.

The checked representative Temporal corpus
(`tools/gomad3integration/qualification/temporal.json`) holds fifteen tier 2
package workloads and thirteen tier 3 functional workloads: twelve `./tests`
suites and one `./tests/gomadfunctional` probe. Its expectations are per
platform. On darwin/arm64 all twenty-eight workloads are expected to qualify;
nine of the package workloads build with the `gomad` tag. On linux/amd64 five
package workloads qualify and ten retain
exact unsupported analyses (the amd64 xxhash assembly), because the packs that
admit their facts are scoped to darwin/arm64, and the thirteen tier 3 workloads
are expected `intermittent`: the twelve `./tests` suites cite the linux replay
divergence recorded in the
[milestones](../../MILESTONES.md#open-findings), and the probe
cites its own finding, one linux run whose seed 17 did not reproduce its
evidence. Every
qualified workload runs two
seeds and requires matching execution, World, I/O, and choice-tape replay.
The generated `tests.json` manifest beside it holds one tier 3 workload per
top-level `./tests` test. It is an on-demand local set rather than a CI gate,
and it is untraced by default: its generator spec sets `choice_bytes` 0 and
`replay_successes` false, so a `qualified` workload there establishes same-seed
repeatability and makes no choice-tape replay claim. A test records a bounded
choice trace and replays its successes only where the spec opts it in by name.
The tracing-enabled replay gates are this representative corpus, the smoke
selection copied from it, and `qualification/core.json`: each of their
workloads records a choice trace and replays its successes, and a seed there
verifies choice-tape replay when it reports `choice_replay_exact`. An
`intermittent` expectation in those manifests still accepts a seed that
diverged. The milestones record the remaining dispositions.

An interrupted campaign retains a canonical `gomad3.campaign-plan/v1` beside
its prepared target. A guided plan also records the selected corpus snapshot
identity, regression mode, and frozen seed selection, so resume never reselects seeds.
The current plan records the seed or choice-exploration strategy, its controller
identity, every search bound, immutable-segment limit, simultaneous partial
Executions, and success, failure, transcript, and aggregate Artifact capacities.
Campaign v1 publications reference `executions/index.json`, which binds each
private, zero-padded JSONL segment by record count, byte count, and SHA-256.

`gomad plan` publishes a canonical `gomad3.campaign-plan/v1` and adjacent
private bundle containing the verified prepared target and complete bounded
copies of configured read-only mount trees. Plan identity is independent of
the plan output path and original mount source paths. The initial protocol
accepts only seed campaigns with `--on-failure=all`; dynamically
discovered choice-exploration prefixes require a later round coordinator.
`gomad execute-shard --shard INDEX/COUNT` uses a zero-based ordinal-modulo
partition, revalidates the entire bundle before execution, and records global
selection ordinals in Campaign v1. `gomad merge` accepts only shards from the same
plan, rejects duplicate or missing ordinals unless `--partial` is explicit,
deduplicates retained evidence by content identity, enforces aggregate bounds,
and publishes a new `gomad3.merged-campaign/v1` without mutating shard artifacts.
Both plan and aggregate are available through `gomad inspect`.
The Campaign store records the explicit `planned`, `prepared`, `running`,
`committing`, `published`, and `recoverable-failure` lifecycle. A validated
`campaign.json` is authoritative even when a crash leaves private state behind.
`gomad recover CAMPAIGN` locks the Campaign and either finishes that private cleanup,
normalizes an interrupted commit to its validated running state, or reports
that the Campaign is invalid or not recoverable without changing it. Add `--json`
for the stable `gomad3.recovery/v1` result. Invalid or non-recoverable input
returns status 2; storage, locking, and publication failures return status 3.

`gomad resume CAMPAIGN` uses the same store-owned preflight, locks that Campaign, verifies the exact
Runner, toolchain, I/O profile, prepared binary, completed records, and every
referenced failure or successful-Execution Artifact, archives incomplete per-seed state, and schedules
only unfinished selection ordinals. Closed Execution segments remain immutable;
resume may incorporate one contiguous segment whose rename completed before
its index update, and it archives an active segment before excluding only a
torn terminal record. It appends to and eventually publishes the original
Campaign; repeated resumes are safe when the recorded aggregate deadline is too
short to finish all remaining seeds. Published batches, changed inputs,
concurrent resumes, and interrupted preparation fail closed. `gomad inspect`
reports the index identity, segment totals, journal limits, and artifact
capacity. Add `--json` to use the same stable campaign event stream as
`explore`.

| Status | `explore` / `resume` | `qualify` | `replay` |
| --- | --- | --- | --- |
| 0 | All selected or remaining Executions succeeded. | Every repetition succeeded with identical evidence. | Verification-only succeeded, or replay matched a retained success. |
| 1 | A target failure, watchdog observation, or ordinary or World replay divergence was retained. | Evidence diverged, a target failed, a required probe was absent, or replay diverged. | Replay matched a retained failure or watchdog observation, or diverged; inspect `reproduced=true|false`. |
| 2 | Input is invalid, the target is unsupported, or the resume journal is incompatible. | Input was invalid or the unsupported boundary was retained. | Input or artifact compatibility validation failed. |
| 3 | Runner or host infrastructure failed, or a Choice Exploration forced-prefix candidate diverged. | Qualification or report infrastructure failed. | Replay infrastructure failed. |

Choice Exploration forced-prefix candidate divergence returns status 3, including
mixed failures; ordinary seeded and World replay divergence retain status 1.
Read `choice-replay=exact` for a verified choice-tape replay claim; matching a
seed-only replay or watchdog observation does not establish that claim.

The Runner prepares one immutable target, launches every seed or forced-prefix candidate in a fresh
contained process and work directory, enforces wall deadlines, computes full
stream hashes while retaining bounded output, and publishes canonical,
content-addressed artifacts. `--count N` selects seeds `0` through `N-1` and is
mutually exclusive with `--seeds`. Arguments following `--` use an argv-safe
interface. Trusted repository tooling preparing an `exec` target must use
`target.ReviewCapabilityClosure` and `target.WriteProvenance` to produce v3
provenance for the exact binary. Runner revalidates its package policy, pinned
standard-library membership, module closure, build information, and binary
identity; arbitrary binaries are rejected.
Build information must record cgo disabled, an executable build mode, and no
race detector, shared-library linking, external linking, or plugin linking.
Coverage-instrumented targets are rejected during preparation, provenance
validation, and replay: host coverage-counter flushing is outside the
deterministic-I/O contract. Semantic and choice coverage use Gomad's bounded
recorded probes and do not enable Go code coverage.

`gomad inspect` validates the Campaign journal or immutable failure/success Artifact before
printing its identity, outcome, transcripts, captured mounts, truncation,
distinct failure paths, retained successes and byte totals, novelty reasons,
copy-paste replay commands, and Campaign lifecycle, resumability, repairability,
and recovery reason. Interrupted Campaigns can be inspected before publication.
Add `--json` for the stable `gomad3.inspect/v5` report.

### Deterministic I/O

Every Runner-managed target uses the versioned deterministic-I/O boundary by
default. It is independent of the target package, arguments, and application:

```sh
tools/gomad3/.bin/gomad explore \
  --seeds 7 --parallel 1 --execution-timeout 2m --overall-timeout 5m \
  --artifacts .gomad/qualify/seed-7 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Schema-v2 artifacts must contain this deterministic-I/O identity and its
matching environment marker. Profile-less v2 artifacts are rejected as
incomplete; replay never falls back to host I/O.

Gomad replaces supported loopback TCP operations, filesystem operations,
hostname, and entropy with process-local in-memory implementations. The
in-memory filesystem starts with the root and the process temp directory
(`/tmp`, plus the directory `TMPDIR` names when it is set), so a program that
writes scratch files where `os.TempDir` points finds the directory a host would
provide instead of failing closed on the first open. Local file mappings in
standalone and in-process execution use shared memory: every mapping of one
file region shares a single buffer, a
store through it becomes visible to file reads on the next read, sync, or
unmap, and a file write lands in the mapped bytes. A writable mapping is
available only for volatile files, because stores through memory bypass the
volume journal; overlapping regions with different bounds and the total mapped
bytes (64 MiB) fail closed. Surviving aliases keep the buffer's byte charge
after its original owner closes. That is the local contract SQLite's WAL index
needs. Process-backend mappings support copied read-only bytes; writable Map
returns `ENOTSUP` before handle/access/bounds checks. Cached nonnil bytes are
returned without revalidation, and nil cache state refetches. Optional
built-in adapters are an immutable collection generated from `version.json`.
The current version-pinned `modernc.org/libc` adapter redirects supported
filesystem, entropy, and time operations to those same generic boundaries.
Its `system`, `pause`, and `signal` refuse unconditionally, so the prepared
module imports neither `os/exec` nor `os/signal`; pack validation rejects any
pack or request that admits `os/exec`, `os/signal`, or `os/user`.
The exact `google.golang.org/grpc@v1.83.2` adapter removes its Unix raw-socket
keepalive callback because Gomad's in-memory TCP connections have no kernel
socket to configure; it preserves the negative `KeepAlive` value and does not
claim kernel keepalive support. On Linux it also compiles gRPC's own non-Linux
`internal/channelz`, `internal/syscall`, and ready-reader implementations, so
channelz socket introspection, TCP user timeouts, CPU-time reads, and
non-blocking ready reads stop at the same stubs darwin uses instead of reaching
raw connections or `x/sys/unix`. On every platform it also rewrites the three
portable gRPC files that import `syscall`: the disconnect-reason label keeps
only its context and deadline classifications, credentials stop wrapping TLS
connections in a `syscall.Conn`, and the ready reader always takes the
blocking path, because Gomad's connections never expose a descriptor.
Three exact adapters keep the Temporal server's own dependencies away from the
host so that closure mode can be claimed for `./tests`: `go.uber.org/fx@v1.24.0`
keeps its shutdowner channel plumbing but never registers with `os/signal` and
names its signals locally instead of through `x/sys/unix`;
`go.temporal.io/sdk@v1.48.0` returns an interrupt channel that never fires
instead of installing a signal handler; and `go.opentelemetry.io/otel/sdk@v1.44.0`
reports `<unknown>` for the process owner and `uname` the way the module already
does on unsupported platforms, and its BSD host-id command runner refuses
instead of reaching `os/exec`. Each rewrite is anchored to exact file digests,
so an upstream edit fails the build instead of shifting the rewrite;
`gomadtool adapter-regenerate` re-derives the anchors for a new exact version
only after a person approves the changed source (see the dependency bump
procedure below).
Each target records the exact adapters it selected, and resume and replay fail
before execution if an identity is unavailable or changed. Entropy is
independent of `GOMADSEED`; that seed controls scheduling only.

The version-pinned compiler inserts typed entry prologues into the selected
`os` and `net` definitions before optimization. This keeps the standard names,
method sets, interfaces, and call sites intact while routing every invocation
form through additive same-package hooks. Before rewriting, the compiler
validates each definition's complete formatted declaration fingerprint as well
as its name and signature, so a signature-stable upstream body change fails the
build. It marks intercepted definitions non-inline so serialized pre-rewrite
bodies cannot bypass the hook.

Compiler conformance interceptions live in `boundary/compiler-tests.json`, not
the production boundary manifest or shipped compiler table.
`make -C tools/gomad3 intercept-test`
builds a temporary compiler from a Go overlay containing those fixtures, proves
the production compiler ignores their package paths, and then runs the positive
and fail-closed compiler cases through that test-only compiler.

Every modeled operation is appended to a bounded shared-memory transcript.
Retained artifacts keep the canonical transcript, and replay supplies it to
the target through a read-only shared-memory region so divergence stops at the
first mismatching ordinal:

```sh
tools/gomad3/.bin/gomad replay ARTIFACT_DIR
tools/gomad3/.bin/gomad replay --verify-only ARTIFACT_DIR
```

The transcript holds 64 MiB by default, about half a million operations; a
target that fills it stops with an incomplete transcript. `--io-transcript-bytes`
on `explore` and `qualify` (`"io_transcript_bytes"` on a `qualify-set`
workload) raises it in whole MiB up to 1 GiB. The Runner sizes both transcript
backings to the bound and writes it as the produced header's capacity, which
the runtime maps; the bound is recorded in the execution limits, the campaign
plan when it differs from the default, and the evidence, so replay, resume, and
shards restore it.

Unsupported calls entering an inventoried shim fail closed before host I/O.
This boundary is not an OS sandbox: trusted target code must not bypass the
reviewed boundaries with a direct raw syscall. DNS, non-loopback sockets,
subprocesses, cgo, plugins, external linking, and unrecognized native I/O are
outside its supported contract.

### Lazy read-only inputs

The Runner can expose an explicit host directory through a repeatable lazy
read-only mount. It captures only entries first observed by the target,
serves subsequent reads from memory, and stores captured inputs in retained
failure artifacts so exact replay does not reopen the host directory:

```sh
tools/gomad3/.bin/gomad explore \
  --io-ro-mount ./fixtures=/fixtures \
  --seeds 7 --parallel 1 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Mount sources are resolved relative to the Runner working directory; target
destinations are normalized into its virtual absolute namespace and may not
overlap. Symlinks, hard-linked files, special entries, unstable captures, and
capacity overflow fail closed. Write-capable opens within mounts return
`EROFS`, and reads outside declared mounts do not fall through to host files.

The compatibility-only `GOMAD3_RUN`, `GOMAD3_PACKAGES`, and `GOMAD3_ARGS`
variables are trusted Make recipe shell fragments, not an argv-safe public
interface. Shell metacharacters and values that require quoting must be quoted
for both Make and the recipe shell.

The stable Go command is `tools/gomad3/.toolchain/bin/go`. The build verifies
the official Go source checksum, snapshots and validates
`toolchain/runtime/go1.27.1.patch` and `toolchain/runtime/overlay`, rejects
upstream overlay collisions, copies the exact overlay
snapshot, applies the exact patch snapshot with zero fuzz, and caches immutable
builds by the Go version, source checksum, patch and overlay checksums, host OS
and architecture, bootstrap Go version, and canonical build environment.
Same-key builds use an atomic owner lock, and ambient Go experiment,
architecture, C/C++ tool, and compiler/linker tuning is cleared before
`make.bash`. Set `GOMAD3_BOOTSTRAP_GO` to choose a bootstrap `go` command.

Host-side policy is implemented in typed Go packages. `toolchain` provides the
build, patch, validation, and upgrade interface; `cmd/gomadtool` is
its command adapter, and `internal/gomadtool/conformance` owns bounded
black-box fixture execution and semantic result classification. The remaining
scripts are reviewed argv adapters:
POSIX compatibility entrypoints, the two upstream `-exec`/`-toolexec`
adapters, and the Darwin-only DTrace audit. `make -C tools/gomad3 validate` rejects an
unowned script or new Bash/Perl policy. Linux CI builds the toolchain, runs the
harness, toolchain, interception, overlay, world, builder, live-capability,
upstream, runtime, and host tiers as gates, and
qualifies the Linux compatibility packs and the core corpus. The macOS sandbox
test and the DTrace audit stay darwin-only. The modernc libc adapter covers both
platforms: on darwin it models the libc functions themselves, on linux/amd64 it
models the syscall numbers behind the musl trampolines, and each platform has
its own compatibility pack for the facts the rewritten module still carries.
Adapter replacements are published under `.toolchain/adapters` by identity and
inventory, because the go command records a directory replacement's path in the
binary and a per-campaign path would change the target identity between
repetitions.

To upgrade Go, update the canonical `toolchain/version/version.json` descriptor
and `deterministicio/boundary/manifest.json`, materialize the old patch against the new pinned
source, and regenerate the patch with `go -C tools/gomad3 run
./cmd/gomadtool patch-regenerate --root="$PWD/tools/gomad3"
--candidate-root=GO-SOURCE-ROOT`. `make -C
tools/gomad3 generate` derives the Make, Go, compiler-spec,
interception-report, public-inventory, and upgrade-guide consumers. The
descriptor's patch and overlay allowlists must exactly equal the checked trees.

Run the version-specific command from the generated upgrade guide, or directly:

```sh
make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=<previous-commit>
```

The command first requalifies the neutral core corpus, then publishes
`.toolchain/upgrade-dossier.json` even when a behavioral gate fails. It records
the complete upstream patch, semantic boundary diff, interception evidence,
overlay collision audit, disabled upstream results, mandatory probes,
host-clock audit, the checked `gomad3-core` corpus, and platform qualification.
The boundary diff compares canonical complete entries, including generated hook
policies and fields introduced by a newer manifest, instead of projecting onto
the fields known to the previous dossier implementation.
A dossier cannot report `qualified=true` without that canonical corpus report,
a baseline boundary manifest, and either an empty boundary diff or explicit
approval. After reviewing a non-empty diff, rerun with
`GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256=<boundary_manifest_diff.sha256>` to
record approval for that exact canonical diff. Supported-host CI reads the same
value from the `GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable, so
an administrator can approve and rerun an intentional boundary-change check.
CI uploads both the dossier and its retained core-corpus evidence
on every run.

A dependency bump keeps every pin exact. Apply it to the target module
(`go get MODULE@VERSION`) without committing it, then run from the repository
root (for a module other than the root module, add `--module=DIR` to
`pin-impact`):

```sh
go -C tools/gomad3 run ./cmd/gomadtool pin-impact --root=.
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate \
  --module=<adapted-module> --version=<new-version>
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate \
  --module=<adapted-module> --version=<new-version> \
  --approve-review=<digest the dry run printed>
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh --root=.
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack generate --root=. \
  --request=internal/compatibilitypack/requests/<id>.json \
  --approve-review=<digest refresh printed>
make gomad3
make -C tools/gomad3 validate compatibility-pack-qualification
```

`pin-impact` names every adapter, pack rule, interception fingerprint, and
host-clock reference the candidate `go.mod` invalidates, with status 1 when
any is invalidated or unknown. For each named adapter, the
`adapter-regenerate` dry run prints the changed upstream source and the
proposed anchors; after review, the apply with the printed digest writes the
adapter, its tests, `version.json`, and the generated outputs together, and
lists any other reference to the previous version for a hand edit. A rewrite
whose source no longer matches its reviewed form stops with status 1 and writes
nothing; this applies to the syntax-aware `modernc.org/libc` adapter too.
`compatibility-pack refresh` re-reviews every request the bump invalidates and
prints one `generate --approve-review` command per request. Each platform's
host refreshes, approves, and qualifies its own platform's requests, and
reports the others as not evaluable. `make gomad3` rebuilds `.bin/gomad`,
because a regenerated adapter changes the target identity, and the adapter's
workloads are requalified on both platforms. `pin-impact` and `refresh`
compare the working tree with `HEAD`. A pack pinned to the old version stays
invalidated once the bump is committed, because the module still requires its
activation modules at other versions; a module the bump removes is reported
stale only against a baseline that requires it, so after committing such a
bump pass the revision before it with `--baseline-ref`. [CLI.md](CLI.md#bump-a-dependency) gives each command's flags and
exit statuses.

The standard-library boundary is declared in
`tools/gomad3/deterministicio/boundary/manifest.json`, and the cross-process
deterministic-I/O layouts are declared in
`tools/gomad3/deterministicio/schema/iowire.json`. After changing
either schema or its templates, regenerate and verify the derived artifacts
with:

```sh
make -C tools/gomad3 generate
make -C tools/gomad3 validate
```

Generated host and overlay tests consume the same golden vectors. The normal
Gomad v3 test target also tests the overlay codec and typed mount client inside
the patched toolchain.

## Contract

A directly launched target activates Gomad with `GOMADSEED`. A Runner-managed
target activates deterministic I/O through an identity-bound inherited
bootstrap frame and a private activation marker. When neither path is present,
the toolchain follows the upstream runtime paths. Activation forces the initial
`GOMAXPROCS` to one, disables asynchronous preemption, and seeds existing
runtime choice paths. Seed `0` is valid; empty, malformed, and overflowing
direct seed values fail before user initialization.

Runner-managed targets see `TZ=UTC` and explicitly supplied environment entries
from package initialization onward. Runtime control variables remain hidden.
Direct seeded launches expose only `TZ=UTC`, preserving isolation from the
caller's inherited environment.

Enabled targets start at midnight UTC on 2000-01-01. Standard `time.Now`,
monotonic elapsed time, sleeps, timers, tickers, callbacks, and context
deadlines use the process virtual clock. When no goroutine is runnable, the
runtime advances directly to the earliest native timer deadline. Runnable
work is never skipped to deliver a future timer, and equal-deadline timers use
the seeded runtime choice stream. An explicit `testing/synctest` bubble keeps
its private clock and takes precedence over the process clock.

Under the default `strict` tick policy every `time.Now` within one busy stretch
returns the same instant. `--clock-tick=forward` on `explore` and `qualify`
(`"clock_tick": "forward"` on a `qualify-set` workload) advances the clock at
every `time.Now` by 1 to 1024 nanoseconds, drawn from a stream derived from the
seed and separate from the scheduling choices, so consecutive reads advance
even while work remains runnable. Each draw advances the process virtual clock
itself, so `time.Now`, monotonic elapsed time, timers, sleeps, context deadlines,
and simulation time observe one clock. A draw can make a timer due while work
is runnable; the runtime delivers it at its next timer check without skipping
runnable work. Process-simulation participants report forward progress when
they activate, quiesce, and issue host-model requests; the host arbiter adopts
their maximum reported time before advancing to the earliest timer deadline,
and the coordinator adopts each request's reported time before applying its
model operation. Consecutive readings differ at nanosecond resolution only;
timestamps truncated to a coarser unit can still tie. Repeatability and exact
replay remain workload qualification claims; the
[milestones](../../MILESTONES.md#open-findings) record remaining divergence.
Runtime-internal clock reads do not tick. The policy reaches the target as
`GOMAD3_CLOCK_TICK=forward`, which is part of the
recorded environment and therefore of Campaign, Artifact, plan, and evidence
identity; `strict` is recorded as the entry's absence, so its identities are
unchanged, and replay, resume, and shards restore the recorded policy. A direct
`GOMADSEED` run honors the same variable, and any other value stops the
process before user initialization.

The standard `go test` harness also observes virtual time. In particular,
`-test.timeout` is a logical-time deadline and may fire immediately in wall
time when it is the next event. A separate wall-time process watchdog is still
required for CPU loops, unsupported host operations, and toolchain failures.

Some reporting surfaces intentionally remain on host time. A completed
collection stamps `runtime.MemStats.LastGC`, `runtime.MemStats.PauseEnd`,
`debug.GCStats.LastGC`, and `debug.GCStats.PauseEnd` with host wall time. Both
stamps reach text heap profiles and the `expvar` `memstats` value; `LastGC`
also reaches heap dumps and Prometheus's
`go_memstats_last_gc_time_seconds` gauge. The proposed overwrite hook would
edit the already-allowed `runtime/proc.go`, not a prohibited collector file,
but the patch-policy decision declines it because scheduler-owned code would
mutate collector-owned state and cross the prohibition in substance while
leaving the underlying host read. An emitting target can therefore produce
different evidence while its choice tape still replays exactly. A target that
branches on a stamp can also change its later behavior and choices.

Three other host-time paths are target-visible under explicit gates. On
linux/amd64, the FIPS `monoTime` input is host monotonic time when FIPS mode is
enabled and Gomad's seeded testing reader is absent; it then seeds the DRBG and
later random reads. Runner-managed deterministic-I/O programs install that
reader only when they link `crypto/rand`. On darwin/arm64 the monotonic input
is virtual. When execution tracing is started, its clock snapshot carries host
wall time on both platforms and host monotonic time on linux/amd64, although
trace event timestamps remain virtual and the runtime never reads the snapshot
back; the difference reaches evidence only when the target or harness retains
the trace. An exact compatibility-pack rule may admit the `syscall` import,
after which an explicit `syscall.Gettimeofday` call returns host wall time
through the linux vDSO or Darwin libc trampoline. The pack gate is at the
import level, not the individual function.

On linux/amd64, `cputicks` reads host cycle counts with `RDTSC` or `RDTSCP`.
When block or mutex profiling is enabled, those counts steer profiler sampling
and runtime-lock stack retention. Written text profiles carry the derived
cycles-per-second value, and protobuf profiles with samples use it to convert
durations. With both profile rates zero and no profile written, the remaining
read is unused. On darwin/arm64 `cputicks` uses the virtual monotonic clock.
The reporting, FIPS, tracer, and `Gettimeofday` paths above do not feed the
runtime's own scheduling, GC pacing, or allocation decisions. `cputicks` does
steer the profiling decisions described here and can consume a profiling
random draw; its downstream effects on linux/amd64 remain an open finding.
None of these values is covered by the deterministic-time guarantee. If a
target emits one or lets it steer target behavior, same-seed evidence can
differ; qualification compares only what reaches its bounded evidence.

For a fixed toolchain, architecture, program, deterministic external inputs,
and seed, supported runtime-controlled choices repeat across fresh processes.
Different seeds explore different choices when alternatives exist. Runtime
choices must finish before output or other external I/O is performed.
When v3 choice recording is enabled, exact replay forces stable logical
goroutine and select-poll alternatives independent of their physical queue
order, consumes the complete tape, and still compares final observation
records. Choice traces and tapes remain explicitly byte-bounded; overflow is a
Runner failure and cannot claim exact replay. The choice terminal frame also
carries the peak live goroutine count, sampled at every goroutine creation, and
execution evidence records it as `peak_goroutines` next to
`virtual_time_elapsed_nanos`, how far the simulation clock advanced before the
target exited; both are deterministic and compared between repetitions.
Qualification reports add each execution's `wall_elapsed_nanos`, which is
informational and outside the evidence digest.

An opt-in diagnostic trace (`--diagnostics` on `explore`, `plan`, and
`qualify`) records a runtime-state digest at every choice point: the virtual
time, allocation count, GC cycle and phase, run-queue length, and the draw
counters of the seeded streams (run queue, scheduler, select, runtime rand,
cheap rand, timer, and clock tick). A divergence on an untaped draw therefore
shows as a counter delta at the next choice point. The trace is its own record
kind, `gomad3-diagnostic-trace/v1`, on its own inherited descriptor; its
capacity is derived from the choice capacity so that every choice record fits,
and never exceeds 64 MiB. Overflow is a Runner failure and supports no
localisation. Recording neither allocates on the Go heap nor draws from the
seeded stream. Enabling it binds `GOMAD3_DIAGNOSTIC_PROFILE` in execution
identity, like `--choices`; with it off, plans, Campaigns, Artifacts, and
evidence keep their bytes. Each completed execution keeps its trace as a
private sidecar under its Campaign's `diagnostics/`, even when its successful
Artifact is discarded. `gomadtool diagnostic-diff` and a qualification report's
`diagnostic_divergence` name the first ordinal whose digest differs, the
differing fields, and both records; a fault fixture that injects one
host-timed draw localises at ordinal 5, field `runtime_cheap_rand_draws`.
Diagnostic traces compare fresh executions. Replay re-executes with collection
off, and a diagnostic trace never establishes exact replay.

The local run queue preserves the head's goroutine class on each dispatch.
A runtime-owned head runs deterministically; a user head chooses only among
queued user goroutines and records a decision only when at least two exist.
Both classes advance through the queue. Finalizer and cleanup goroutines
executing user callbacks use the runtime's user classification. Run-next,
the global queue, timer delivery, and collector workers picked outside the
local queue retain their existing rules. This rule does not make a CPU loop
that never yields progress, and tapes from the preceding controller identity
are rejected.

Deterministic mode supports internally linked pure-Go targets on the qualified
`darwin/arm64` and `linux/amd64` hosts. Enabled cgo or externally linked binaries fail before package
initialization. Windows, plugins, foreign threads, the race detector, signals,
finalizers, and host-dependent network, filesystem, process, and other I/O
readiness are outside the contract. Launch targets compile with
`CGO_ENABLED=0` and set `TZ=UTC`. The public `go-test` target preserves only
explicit `--build-tag` values; Temporal's root wrapper selects `test_dep`
explicitly.

Closure capability mode performs dependency review without compiling
`-gomadguard` guards; an exact compatibility-pack admission does not make host
operations deterministic, and admitted code must stay within the declared
deterministic boundaries. Pack admissions of `syscall` are per package, so
admitted code runs live unless the workload qualifies in guarded mode, and the
determinism soak includes one guarded-mode workload for that reason.

These channels are declared outside the contract rather than given a fixture,
because no host Gomad qualifies on can vary them as a positive control:

- Real-socket and descriptor readiness delivered by host netpoll is outside the
  determinism contract; supported modeled loopback TCP uses deterministic
  in-memory readiness instead.
- SIGPROF delivery and CPU profiling are outside the determinism contract
  because signal arrival and CPU samples depend on host execution.
- Enabling block or mutex profiling is outside the determinism guarantee:
  host-dependent contention timing and profile sampling can change random
  draws, profile allocations, and subsequent runtime state, especially on
  linux/amd64.
- `runtime.NumCPU` is not virtualized and reports OS-detected CPU availability
  at process startup; workloads that use this value require the same host CPU
  configuration for repeatability, and cross-host CPU-count equivalence is
  outside the contract.

Equal-deadline timer ties and overflowing run-queue shuffles, which draw from
the seeded stream without a choice record, have seeded conformance fixtures
(`timer_ties`, `runq_shuffle`) with positive controls: seeds 0 to 31 give 32
distinct completion orders, and each seed repeats under bounded host load.

The determinism claim is quoted from the scheduled soak, not from
two-repetition qualification, which a defect at a one-in-26 rate usually
passes. `gomadtool soak` runs fresh same-seed repetitions of the functional
smoke suites and the guarded-mode frontend probe on seeds 11 and 17, in
`qualify --diagnostics` batches of 32 under two busy host threads, and compares
every batch with its cohort's baseline: one workload, seed, platform, and
execution identity, so a new toolchain build key starts a new cohort. The gate
accepts zero divergences; a trace overflow, target failure, or infrastructure
failure is reported separately and is not a pass, and a divergence retains
both diagnostic traces and the differ output. Its bound is per platform and per
cohort: the cumulative fresh repetitions of clean batches across retained
scheduled runs while the cohort has no divergence, measured with diagnostics
on; repetitions of overflowed or failed batches never count toward it. No native bound is retained
yet. The soak's mechanics were exercised on an ARM64 Linux development host,
where the patched toolchain does not build, against a stand-in `gomad
qualify`; that run measured no Gomad bound. The first retained scheduled or
dispatched run on each platform supplies the native bound, and linux/amd64
stays informational while the
[linux replay divergence](../../MILESTONES.md#open-findings) (fn-105 D12) is
open.

The runtime system monitor is disabled with asynchronous preemption, so a
CPU-bound goroutine or `select` polling loop may run forever and prevent
virtual-time advancement. Unsupported blocking I/O is likewise bounded by the
external wall watchdog rather than treated as a clock event. Calling
`runtime.GOMAXPROCS` to raise the value after startup is unsupported.

The Go test driver retains at most 1 MiB from each child output stream while
continuing to drain both streams. Every harness result directory records
`output-truncated` separately from `timed-out` and the child `status`. The Gomad
Runner has a separate configurable per-stream limit that defaults to 8 MiB.

Runtime decisions that only shape host-side scheduling draw from the M's own
random stream rather than the process-wide seeded one: lock hand-off
anti-starvation wakes, the wait-time sample an M takes before it sleeps on a
contended runtime lock, work-steal order, and pcvalue-cache eviction all happen
at host-timed moments (contended runtime locks, idle windows whose length the
Runner decides, stack walks on whichever M holds the P), and drawing them from
the seeded stream moved every later type-assertion-cache fill and semaphore
ticket between same-seed runs. The wait-time sample was the last of these to
move: an M that holds the P sleeps on the scheduler lock only when another M,
parking or returning from a Runner syscall, keeps it past the spin, and that
single draw shifted the stream, so a type-assertion cache grew at a different
call site, two heap-span refills swapped order, and a few same-seed replays in
a hundred of a functional suite diverged on darwin/arm64. A garbage-collector
stack scan, and any other `suspendG`, first waits for a goroutine inside a
plain host syscall (a pipe write, a read-only mount lookup) to return and
queue itself as an arrival, so the collector's view of live memory does not
depend on when the host answered; simulation transport reads are exempt
because they block until the simulation advances.

The seeded stream's draw sites are classified, not assumed.
`toolchain/draw_inventory_test.go` lists every reference to the patched
runtime's seeded random helpers on both qualified platforms (269 references in
135 classified rows when it was taken) as target-ordered or host-timed, and the
toolchain tier fails on an unclassified reference, a changed count, or a
reference that disappeared, so a Go upgrade cannot add an unreviewed draw.
Host-timed sites use the M-local stream, including host netpoll and no-P batch
admission, which reach the run-queue shuffle through an explicit host batch
origin while target admission keeps its seeded shuffle, and the linux CPU
profiler's timer sample. In diagnostic mode the runtime fails the process when
a path marked host-timed reaches the seeded stream. One host-timed seeded draw
remains: the classic collector's `enlistWorker` in `runtime/mgcpacer.go`, a
collector file the patch policy prohibits editing. Seeded activation rejects
the Green Tea collector before user code, because its worker draw is outside
the qualified profile. The inventory, its runtime check, and the rerouted sites
are a merged candidate whose native darwin/arm64 and linux/amd64 gates have not
yet run.

Remaining current-candidate native Darwin qualification is deferred under
[fn-149](../../.flow/specs/fn-149-gomad-deferred-darwin-qualification.md), and native
Linux qualification and Linux CI work remain deferred under
[fn-128](../../.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md).
The [native transfer manifest](../../.flow/artifacts/native-scope-transfer-2026-10-07.md)
preserves the exact inherited gates and retained source requirements. These
deferrals establish no native pass or soak bound and authorize no PR, push or CI run.

At the start of every mark phase, while the world is still stopped, the
runtime greys every M with its g0, gsignal and `self` handle, every P's `oldm`
handle, and every goroutine, and an idle M does not keep the `allp` snapshot
that `findRunnable` takes before dropping its P. Which M runs a goroutine after
a syscall hand-off, and which Ms park through `findRunnable`, are host timing;
the collector otherwise shaded those structures from the write barrier or from
the parked M at host-timed points, and because the assist that ends a mark
phase stops on a work boundary, the same-seed run-queue order of a
cluster-sized target diverged from its first collection on darwin/arm64. The
`oldm` handle is the 8-byte weak pointer `acquirep` copies from the M that took
the P; scanning the P greyed whichever handle the last hand-off had left there,
which moved 8 bytes of scan work between drain slices, then a work buffer,
then every later page of the heap, so a same-seed replay of a functional suite
printed a different `%p` in its logs.

On darwin/arm64 an activated target re-executes itself once with ASLR
disabled before runtime initialization continues. The darwin/arm64 linker
produces only position-independent executables and the kernel slides every
image, so the addresses of type descriptors, globals, and functions differed
between two runs of one binary; caches keyed by those addresses (reflect2's
type cache, reflect's lookup caches, `map[reflect.Type]`) then allocated a
different number of hash-trie nodes during package initialization, and the
heap layout, the first collection, and every later run-queue order followed
the slide. The re-execution uses `posix_spawn` with `POSIX_SPAWN_SETEXEC` and
the ASLR-disabling attribute debuggers use, so the pid, inherited descriptors,
and process group the Runner supervises are unchanged; a kernel that ignores
the attribute fails the target closed instead of looping. linux/amd64 targets
are not position independent and need no re-execution.

Targets build with `GOEXPERIMENT=nogreenteagc`. The Green Tea collector
attributes scan work per span batch, so the pacer's view of a cycle followed
the order in which the marker reached objects, and that order followed which
M held the P (a host-timing race at every syscall hand-off, visible through
`p.m`, `g.m`, and the idle M list). The classic collector attributes scan work
by object layout, which does not depend on traversal order, and with it the
same-seed evidence of the functional suites reproduces.

The mode is intended only for trusted tests. Deterministic map seeds remove a
hash-randomization defense and must not be enabled in production. Each process
uses one P, so run different seeds in separate processes for parallelism. The
shared runtime random state also means program changes can change later choices.
Exact choice tapes are bound to the target, pinned toolchain build key,
platform, and choice-controller implementation and are not portable across
those identities.

## Compatibility-pack development

Compatibility packs use only the strict `gomad3.compatibility-pack/v2`
contract. Every allowed fact is bound to an exact module, complete compiled Go
and foreign-source inventories, a package source-set digest, governance, and an
explicit platform scope. Local module replacements are rejected unless
they are created by a registered deterministic-I/O adapter and carry the exact
profile, adapter, original/replacement inventory, and prepared source-set
identities.

The development workflow is discover, review, exact approval, generate, check,
and qualify:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack discover \
  --root="$PWD/tools/gomad3" --request=internal/compatibilitypack/requests/<id>.json \
  --working-dir=<target-module>
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack review \
  --root="$PWD/tools/gomad3" --request=internal/compatibilitypack/requests/<id>.json \
  --output=internal/compatibilitypack/reports/<id>.md
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack generate \
  --root="$PWD/tools/gomad3" --request=internal/compatibilitypack/requests/<id>.json \
  --approve-review=<exact-review-sha256>
make -C tools/gomad3 validate compatibility-pack-qualification
```

`internal/compatibilitypack/working-directories.json` names the module
directory each request is discovered and qualified in; every request needs
exactly one entry in a directory holding a `go.mod`, and `check` rejects a
request without one and, for this repository's root, a missing table.
`compatibility-pack-qualification` qualifies every request that names the host
platform in its mapped directory (`compatibility-pack qualify --all`).

For a dependency bump, compare the candidate with its baseline before
repairing pins. The checkout workflow above uses the working tree and `HEAD`;
a saved pair of `go.mod` files with adjacent `go.sum` files can be compared
explicitly:

```sh
go -C tools/gomad3 run ./cmd/gomadtool pin-impact \
  --root=. --baseline=/absolute/baseline/go.mod \
  --candidate=/absolute/candidate/go.mod --format=json > pin-impact.json
```

Status 1 means an invalidated or unknown adapter, pack-rule, interception, or
clock-inventory pin; 2 means invalid input, and 3 means an infrastructure
failure. A missing sum or source identity remains unknown rather than
unaffected. For each invalidated adapter, inspect a dry run's source diff and
anchors, then approve the exact printed digest. `--approve=sha256:<digest>`
and `--approve-review=<digest>` name that same approval. The command verifies
rewrite occurrences and publishes the descriptor, anchors, fixtures, and
generated consumers together only after approval.

After a dependency bump is applied to the checkout, one command repeats
discover and review for every request the bump invalidates and stops at
approval:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh \
  --root="$PWD/tools/gomad3"
```

Pass `--baseline-ref=<rev before the bump>` if `HEAD` no longer holds the
baseline. The saved `pin-impact` JSON can be passed with
`--impact-report=/absolute/path/to/pin-impact.json` for its scoped module.
Refresh always reevaluates every mapped target directory with the working
tree as candidate and the baseline revision as baseline, then validates and
merges the saved report. The report supplements live discovery and cannot
suppress a mapped module. It refreshes the
requests whose pack rules it reports invalidated or unknown, every request
without an approval, and every host-platform request bound to another
deterministic I/O profile. Each request is reviewed in its own directory. A
request is current only when its stored approval equals the review digest of
the fresh evidence; otherwise refresh writes the fresh evidence with the
approval cleared, regenerates the reports and packs (which drops the packs
that are no longer approved), and prints the review digest and the exact
`generate --approve-review` command. Requests for another platform are reported
as not evaluable and left for a host of that platform, and a pack whose
directory no longer requires its modules is reported as unselected. Rerunning
after approving some requests reports only the rest. Status 0 means nothing is
left to do, 1 that a request awaits approval, cannot be evaluated here, failed,
or is unselected, 2 invalid input, and 3 an infrastructure failure. An external
pack root is refreshed with `--compatibility-root`, as below; refresh judges
the packs in that root's `packs/` whatever `GOMAD3_COMPATIBILITY_PACKS` names.

A pack variant is removed together with its request, report, and
working-directory entry, and only with retained evidence that no module,
corpus, or fixture in the repository selects it.

Inspect each changed report and use `compatibility-pack generate
--approve-review=<exact-review-sha256>` for each approved request. Then run
`make -C tools/gomad3 validate compatibility-pack-qualification
core-qualification-set` and the full `test` gate on both supported hosts.
A Darwin run does not qualify Linux packs or adapters. The generated
[upgrade guide](deterministicio/boundary/upgrade-go1.27.1.md) covers the
Go-release dossier and this dependency flow.

A module outside this repository keeps packs for its own dependencies in its
own tree, so they never have to be committed here. Every authoring command
takes `--compatibility-root=/absolute/dir`, which holds that module's
`requests/`, `reports/`, `packs/`, `generation.json`, and, for `qualify --all`
and `refresh`, `working-directories.json` in the same layout as
`internal/compatibilitypack`; requests and review output must stay below it.
Setting `GOMAD3_COMPATIBILITY_PACKS=/absolute/dir/packs` loads those packs next
to the embedded ones for every command, including the Runner's supervisor and
coordinator processes. They pass the same strict validation, so no external
pack can admit `os/exec`, `os/signal`, `os/user`, `plugin`, or `runtime/cgo`;
an ID that collides with an embedded pack, a relative or missing directory, and
any entry that is not a pack named by its ID fail closed. Each selected pack's
ID and SHA-256 are part of the target identity, so replaying, resuming, or
executing a shard without the same pack fails before execution.

Malformed or non-canonical requests and packs are invalid input. Source,
toolchain, module-cache, adapter, publication, and cleanup failures are
infrastructure failures. Fresh-review disagreement is unsupported drift. None
of these cases falls back to an older pack, partial inventory, arbitrary local
replacement, host access, or truncated evidence. Requests, generated v2 packs,
review reports, mutation fixtures, and their generation manifest live under
`internal/compatibilitypack`.

`temporal-functional-tests-linux-amd64` admits the amd64 assembly, the
reflect2 linknames, and the procfs process-metrics reads that the `gomad` build
of the Temporal functional test package reaches on linux/amd64; with the
server's `gomad` build seams and the fx, SDK, otel, and gRPC adapters it closes
`gomad analyze --capability-mode=closure go-test ./tests` with zero blockers.
On darwin/arm64 `temporal-functional-compute-darwin-arm64` admits the arm64
assembly and `temporal-functional-tests-darwin-arm64` admits the Prometheus
client's darwin process-collector imports, which close the same analysis.
`temporal-leaf-xsys-darwin-arm64` and `temporal-leaf-xxhash-darwin-arm64`
admit the `golang.org/x/sys/unix` and arm64 xxhash facts that the smaller
`gomad` closures of the Temporal leaf test packages reach without the modules
that activate the functional packs.
The obsolete `temporal-backoff-overflow` and
`xnet-socket-activity-candidate` requests were retired after exact gRPC and
x/net adapters removed their active blockers. The gRPC workload qualifies
with exact replay; the x/net adapter removes the socket linknames, excludes
the Darwin assembly bridge, and denies raw socket options deterministically.

## Simulation contract

The SIM-0 behavioral contract was `simulation/parity/manifest.json`, which mapped
thirteen Gomad v2 behaviors to named v3 cases by citing exact source tests under
`tools/gomad2/`. It was removed by fn-81 together with the Gomad v2 tree it
cited, so every one of its source paths would now dangle. The thirteen cases it
tracked remain implemented through the sixteen in-process and process prototypes
described below, including process evidence for fresh arbitrary package globals
and hard isolation. Parity Case, the name for one of those mapped cases, is a
historical term: the manifest was its only carrier, and it is not part of the
current vocabulary.

The root `tools/gomad3sim` package defines the no-dependency application
harness. Its versioned schemas provide bounded specs, stable node and incarnation
identities, boot registration, detached results, lifecycle and topology
control, typed scenario composition, stable histories and oracles, inspect,
and exact replay. The in-process backend supplies deterministic
multi-address TCP, per-node ports, bounded listeners/connections/deliveries,
fixed link delay, partition/heal, graceful-stop versus crash/reset behavior,
incarnation-bound delayed delivery, canonical network snapshots, and typed
replay divergence. Its separate durable-volume model provides file and
directory sync, dependency-valid partial persistence, persisted-only crash,
restart, bounded resumable crash-state enumeration, and exact replay. Fault
plans bind stable match fields and realized targets independently; scenario,
fault, network, volume, and runtime-choice tapes retain separate identities.
The process backend adds private bounded bootstrap and model IPC, host-owned
time arbitration, fresh package initialization, hard crash/reap, and
cross-backend detached-model conformance. None of those process-isolation
claims is implied by the in-process model.

## World

`world` is a pure in-memory model for deterministic events outside the Go
runtime. It performs no host I/O, starts no goroutines, invokes no callbacks,
and requires no runtime hook. Callers register requests, mark them ready, and
explicitly quiesce to choose and deliver ready events.

`world/mailbox` is the initial explicit adapter. It demonstrates lifecycle,
snapshot/restore, and replay without giving World ownership of application
state. `runner/internal/execution` composes World semantic records with the
Runner's raw process record while keeping those identities separate. A target connects
its World with `world/process.Open`, takes the session-owned World returned by
`Session.Model()`, performs all modeled work, and calls
`Session.Finish` after that work has stopped, or `Session.FinishError` for a
World error. General `Error`, `Is`, and `Unwrap` callbacks are normalized at
this process boundary, with detail captured before classification. Direct
`Recorder.FinishError` accepts original World sentinels and concrete model-owned
errors only; it rejects custom errors and external wrappers without invoking
callbacks or closing the recorder. Callers that normalize errors themselves
use `Recorder.FinishTerminal` with detached capacity, replay-divergence, or
invalid-input data. `Finish` retains quiescence inference. Rebinding public
sentinel variables does not change the pure model's identities or messages.
The trusted bootstrap validates replay input before target
activation; `Open` installs that recorded initial World rather than accepting a
target-created substitute and returns it through `Session.Model()` before modeled work.
The session writes one bounded record with a structured idle, deadlock,
capacity, invalid-input, or replay-divergence terminal result through inherited
descriptors only at the process boundary, so host pipe readiness cannot affect
event ordering. Connected replay requires the executing child to emit the same
semantic bundle and fails closed if it is missing or divergent.

## Design

### Go caller migrations

These intentional Go source and behavior changes preserve the CLI grammar and
recorded JSON formats. The [interface inventory](../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md)
retains the consumer lists and source-specific preservation evidence.

- Runner removes `Executor`, `ReplayExecutor`, and the `Executor` fields of
  `CampaignSpec`, `ResumeSpec`, `CampaignShardSpec`, `ReplaySpec`, and
  `MinimizeSpec`. Callers delete those fields and invoke the existing operation
  functions, which select supervised process execution. `Preparer` and
  `ArtifactReplayer` and their existing request fields remain usable public
  substitutions. Same-package tests pass fakes through private `...With`
  operations and `executionDependencies`; external callers do not construct
  private execution descriptors. Public campaign request field names remain
  unchanged by the internal options owner. `ParseStrategy` and
  `ParseCoverageMode` are additive shared parsers; existing callers need no
  migration.
- `artifact.OpenArtifact(path)` returns `*artifact.Opened`. The caller closes
  it and uses `Path()`, `Manifest()`, and `StoredBytes()` instead of mutable
  fields. `Snapshot()` replaces `Detached()` and returns a deep-copied
  `artifact.Artifact` reference. Replace package-level `OpenPayload`,
  `ReadPayload`, `CopyPayload`, and `TargetSharingOf` calls with the handle's
  `OpenPayload`, `ReadPayload`, `CopyPayload`, and `TargetSharing` methods.
  Publication still returns a detached `Artifact`; it owns no root and needs
  no Close. Handle metadata remains readable after Close, while payload access
  fails through its existing validation order.
- Runner report literals use `ExecutionJournalLimitsInspection` for
  `CampaignPlanInspection.Journal` and `ExecutionJournalInspection.Limits`, and
  `ArtifactCapacityInspection` for the plan and Campaign inspection capacity
  fields. Outcome fields are strings. Callers construct these public detached
  values instead of private campaign plans; operational transitions remain
  inside the Campaign owner.
- `target.CompatibilityPackEvidence` retains its outer name and complete
  report. Construct its nested values using `CompatibilityPackGovernance`,
  `CompatibilityModuleEvidence`, `CompatibilityPackAdapter`,
  `CompatibilityPackageRuleEvidence`, `CompatibilityPackSource`,
  `CompatibilityPackForeignSource`, and `CompatibilityLinknameEvidence` from
  `target`, replacing internal policy types. Explicit projections copy nested
  evidence while preserving report field order, tags, nil/empty slices, and
  pointer presence.
- `pinimpact.Spec.Packs` becomes `PacksDirectory string`. Pass the directory
  containing pack files, such as an authoring root's `packs/` child, to
  override selection. An empty value keeps embedded/environment selection.
  Pack loading still follows baseline/candidate module validation; load errors
  remain pin-evaluation errors. `compatibility-pack refresh` supplies its
  selected root's pack directory. No new CLI flag is needed.
- `target.DigestAdapterSourceInventory` is removed. Repository target/adapter
  implementation callers use the neutral private `internal/sourceinventory`
  owner, translating capacity errors to their existing public error types.
  External callers must stop calling the removed implementation helper; the
  public target preparation and capability-review operations retain inventory
  validation. No new public hashing API replaces it.
- `toolchain/installation` adds `Layout`, `Build`, and validated `Description`
  values. Existing toolchain resolution and target identity APIs keep their
  signatures. Implementation callers obtain owned paths from the description;
  no external request migration is required.

Direct World recorder completion has a bounded behavior migration
(`WORLD.MODEL`, `WORLD.LIFECYCLE`, `WORLD.REPLAY`). `Recorder.FinishError` keeps
its signature and accepts original sentinels, nonnil concrete
`*world.CapacityError`/`*world.ReplayDivergenceError`, and private model-generated
classified errors without invoking arbitrary `Error`, `Is`, or `Unwrap`.
Custom errors, externally constructed wrappers/joins, typed-nil errors, and
rebound public sentinel values fail without closing the recorder. The rejection
has a fixed unsupported-input message; prior callback-derived admission,
messages, and wrapped identity for those direct inputs intentionally change.
Exported sentinel-variable rebinding cannot redefine immutable model identities
or messages. Original sentinels and the owned typed errors retain their normal
Error/Unwrap/Is/As relationships and known-error recording bytes.

Callers normalize custom errors outside World and call `FinishTerminal` with
detached `world.Terminal` data. Use a capacity, replay-divergence, or invalid-input
kind and a nonempty Detail. Empty, inferred, and quiescence kinds are rejected;
`Finish()` remains the inference operation. For example, after caller-side
capacity classification:

```go
recorded, err := recorder.FinishTerminal(world.Terminal{
    Kind: world.TerminalCapacity,
    Detail: detail,
})
```

A connected target can instead report through `world/process.Session.FinishError`.
That effectful seam validates the session, captures `Error` detail before `Is`
classification in capacity, replay-divergence, then invalid-input order, and
preserves unknown-error wrapping and descriptor cleanup/error precedence.
The pure recorder never dispatches those callbacks.

- [Product specification](SPEC.md#productvocabulary-ubiquitous-language) defines
  the canonical vocabulary and current product requirements.
- [Architecture](ARCHITECTURE.md) records the durable runtime, Runner, World,
  artifact, replay, and deterministic-I/O decisions.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomad3 test
make -C tools/gomad3 test-builder
make -C tools/gomad3 test-runtime
make -C tools/gomad3 test-upstream
make -C tools/gomad3 test-host
make -C tools/gomad3 world-test
make -C tools/gomad3 core-qualification
make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=<previous-commit>
```

`test` retains the full gate: the harness, toolchain, interception, host,
overlay, simulation (`test-simulation`), and World tests, then the builder, live-capability, runtime, and
upstream tiers
in that order. The focused targets reproduce the corresponding portion without
weakening the full gate.

`test-simulation` runs the directly seeded `tools/gomad3sim` toolchain tests
and the Runner transport selection in the Makefile, including process network
and filesystem handle cases. The strict
`TestProcessBackendSynchronizesNodeClockWithModelDelay` case retains its open
watchdog finding; its forward-mode regression runs separately. `overlay-test`
includes `internal/gomadsim`,
`internal/gomadmodelwire`, `internal/gomadio`, `os`, and
`cmd/internal/gomadcap`. In the host tier, `TestModelConformanceFilesystem` and
`TestModelConformanceTCP` (`runner/internal/execution`) run 64-operation
sequences generated from five fixed seeds against the in-memory filesystem and
loopback TCP models and against stock Go on the host, and report the shortest
reproducing prefix of a mismatch. `modelDeclaredDifferences` in
`model_conformance_test.go` lists each platform's declared differences, today
only directory allocation size on both, and each has a test. The `cmd/gomad`
end-to-end tests drive `explore`, `replay`, and a SIGKILLed then resumed
coordinator through the built CLI.

The determinism soak runs from the repository root:

```sh
make gomad3-soak GOMAD3_SOAK_WORKLOAD=functional-activity GOMAD3_SOAK_SEED=11
```

It builds the Runner, runs `tools/gomad3integration/qualification/soak.json`,
and writes its ledger, report, summary, and any divergence evidence under
`tools/gomad3/.toolchain/soak`. The scheduled `determinism-soak-darwin` and
`determinism-soak-linux` jobs of `gomad3.yml` run one workload and seed each,
restore the ledger from the latest retained scheduled or dispatched run, and
upload it again with the report.

The suite compares disabled `go run` and `go test` behavior with a local stock
Go 1.27.1 toolchain; benchmarks disabled clock reads against that toolchain;
covers the fixed clock, native timer behavior, context deadlines, logical test
timeouts, nested synctest, cgo/link rejection, non-progress, bounded output,
and deadlock; runs focused upstream `runtime`, `time`, and `testing/synctest`
tests; audits map key families across seeds; and repeats prebuilt map and
scheduler fixtures under distinct allocation layouts and bounded unrelated CPU
load. Supported-host CI additionally runs `make -C tools/gomad3 clock-audit`:
a privileged, positive-controlled DTrace fixture gate that rejects its seeded
fixture's calls to `clock_gettime` or `mach_absolute_time` after Gomad
activation. On both platforms the toolchain
tier pins every standard-library reference to the host clock (`nanotime1`,
`walltime`, `time_now`, `cputicks`, the Darwin libc `gettimeofday` trampoline,
and the linux vDSO clock symbols) against a reviewed, classified inventory and
checks that `nanotime` and `time_runtimeNow` return on activation before
reaching it. linux/amd64 reads the clock through the vDSO, which a syscall
tracer cannot observe, and reads cycle counts without a syscall, so this static
check is its escape gate. The inventory records the known escapes described in
the Contract above. Set `GOMAD3_STOCK_GO` when the
stock Go executable cannot be resolved
from the module-selected toolchain in `PATH`; the test never downloads one.

The address-only check disables automatic GC so retained padding changes the
tested layout without adding GC activity that consumes the shared seeded
runtime stream. The ordinary repeatability and host-load checks retain the
default GC behavior.

Generated source, binaries, downloads, and toolchain builds remain under
`tools/gomad3/.toolchain` and are not committed.
