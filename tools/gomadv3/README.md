# Gomad v3

Gomad v3 is an opt-in Go 1.26.4 toolchain with a small deterministic-runtime
patch and source overlay. It uses native Go goroutines, channels, `select`,
maps, synchronization, `go run`, and `go test`.

Build the argv-safe CLI and cached toolchain from the repository root:

```sh
make gomadv3
```

The complete Gomad v3 Runner and deterministic-I/O contract is qualified only
on `darwin/arm64`. The builder rejects other hosts before starting a toolchain
build. Runtime source may remain portable to other Unix systems, but those
systems are not supported Runner platforms until they have their own adapters,
publication primitives, and complete test gate.

Use the CLI directly so every package and target argument crosses exactly one
argv boundary:

```sh
tools/gomadv3/.bin/gomad explore --seeds 1 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --seeds 1 go-test ./path/to/package -- -test.run=TestName
```

`make gomadv3-runner` is an alias for the same CLI build, while
`make gomadv3-go` builds only the toolchain. The older single-seed wrappers
remain for compatibility:

```sh
GOMADSEED=1 make gomadv3-run GOMADV3_RUN=./cmd/example
GOMADSEED=1 make gomadv3-test GOMADV3_PACKAGES=./path/to/package
```

Those wrappers remove `GOMADSEED` from the custom `go` process and use Go's
`-exec` hook to enable Gomad only in the resulting binary or generated test
binary. Direct execution of a prebuilt binary remains
`GOMADSEED=<seed> ./binary`.

Check whether the complete Runner and deterministic-I/O contract is available
before starting a campaign:

```sh
tools/gomadv3/.bin/gomad doctor
tools/gomadv3/.bin/gomad doctor --json --artifacts .gomad/artifacts
```

The report includes the host, toolchain and Runner build identities, boundary
manifest, I/O implementation, available adapters, artifact-directory access,
resolved installation source, and a location-specific repair instruction.

Every command that executes or verifies a target resolves the pinned toolchain
in the same order: `--toolchain-root`, `GOMADV3_TOOLCHAIN_DIR`, a
`gomadv3-install.json` bundle manifest adjacent to the executable or its parent,
then an adjacent `.toolchain` directory. CLI and environment roots must be
absolute, clean, non-root paths. The source builder uses
`GOMADV3_TOOLCHAIN_DIR` too.

A standalone bundle can place this manifest beside `bin/gomad`'s parent:

```json
{
  "schema": "gomadv3.installation/v1",
  "toolchain_root": "lib/gomadv3/toolchain"
}
```

Relative manifest roots are resolved from the manifest directory. Malformed
or unknown manifests fail closed instead of falling back to another location.

Explore `go run`, `go test`, or a prepared executable target, then replay a
retained failure exactly or verify its immutable inputs without executing it:

```sh
tools/gomadv3/.bin/gomad explore --seeds 0-999 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --count 1000 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --coverage=semantic --keep-successes=novel --success-limit=32 --success-bytes=1GiB --count 1000 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --guide --corpus .gomad/corpus --count 1000 go-run ./cmd/example -- arg
tools/gomadv3/.bin/gomad explore --seeds 0,7,42 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad explore --seeds 0-99 exec --provenance ./example.provenance.json -- ./example arg
tools/gomadv3/.bin/gomad qualify --seed 7 --repeat 2 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad qualify --seed 7 --repeat 2 --choices --replay-successes --success-limit=1 --success-bytes=128MiB go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad analyze --format=json go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad analyze --capability-mode=linked --format=json go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad qualify-set --manifest corpus.json --working-dir ./target --output report.json
tools/gomadv3/.bin/gomad compare-support --baseline baseline.json --candidate report.json
tools/gomadv3/.bin/gomad explore --choices --choice-bytes=8MiB --seeds 0-99 go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad explore --strategy=choice-frontier --seeds 7 --max-runs=128 --max-choice-depth=32 --max-frontier-bytes=64MiB go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad plan --seeds 0-99 --output campaign.plan.json go-test ./path/to/package -- -test.run=TestName
tools/gomadv3/.bin/gomad run-shard --shard 0/4 campaign.plan.json
tools/gomadv3/.bin/gomad merge --output merged-batch campaign.plan.json .gomad/artifacts/v1/run-*
tools/gomadv3/.bin/gomad recover .gomad/artifacts/v1/run-INTERRUPTED
tools/gomadv3/.bin/gomad resume .gomad/artifacts/v1/run-INTERRUPTED
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad inspect --choices .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/run-*/successes/sha256-*
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/run-*/failures/sha256-*
tools/gomadv3/.bin/gomad replay .gomad/artifacts/v1/run-*/successes/sha256-*
tools/gomadv3/.bin/gomad replay --verify-only .gomad/artifacts/v1/run-*/failures/sha256-*
```

Human-readable exploration writes preparation and running progress to stderr,
including attempted, active, successful, failed, watchdog, replay-divergence,
distinct-failure, retained-success, and retained-success byte counts. Its final
result and every retained artifact path with a copy-paste replay command are
written to stdout.

Use `--choices` on `explore` or `qualify` to observe bounded runtime runnable
and select decisions. `--choice-bytes` defaults to 8 MiB, is valid only with
`--choices`, and is part of batch and artifact identity. The trace is
recorded as v2 stable logical decisions. Artifact replay automatically derives
an identity-bound, read-only decision tape and validates every choice before it
is applied; no replay flag is required. `inspect --choices` validates the
retained payload and reports choice kinds, decision and branching counts, the
tape digest, exact-replay availability, and target-specific site fingerprints.
Legacy v1 traces remain inspectable, but replay reports
`choice_profile.replay_unavailable` and does not execute an uncontrolled target.
Prefix replay is an internal bounded-frontier primitive and is not a public CLI
mode in this slice.

Use `--strategy=choice-frontier` to explore every non-selected runnable or
ready-`select` rank observed within explicit bounds. The strategy requires one
base seed plus positive `--max-runs`, `--max-choice-depth`, and
`--max-frontier-bytes` values. It implies choice recording and rejects
`--count`, multiple seeds, and guided exploration. Candidates run in
deterministic breadth-first rounds ordered by forced-prefix length and identity;
parallel completion timing cannot change the committed frontier. A target
failure remains expandable while the selected failure policy permits it.

Each completed round is an immutable, hash-linked transaction below the batch.
An interrupted round is archived and rerun in full on `gomad resume`; logical
and recovery execution counts are reported separately. `frontier_exhausted`
and `choice_depth_complete` are bounded completions, while `max_runs` and
`frontier_capacity` identify incomplete search envelopes. Outcome deduplication
reduces retained evidence only and never removes a distinct forced prefix.

`gomad analyze` defaults to `--capability-mode=closure`, which reviews a
`go-run` or `go-test` target without compiling or executing it. Explicit
`--capability-mode=linked` builds but never launches the target, extracts the
pinned linker record, and separates live blockers from closure blockers removed
by final reachability. Linked mode has no closure fallback: malformed records,
identity mismatches, and capacity failures fail closed. The report uses exact
compatibility-pack decisions, lists every active and eliminated blocker with a
canonical shortest dependency path, and projects conservative deterministic
I/O requirements over the full closure. To keep reports path-free, arguments
containing path separators are represented by stable SHA-256 identities.
`--format=json` emits `gomadv3.capability-analysis/v3`. Status 0 means
supported, 1 unsupported, 2 invalid input or package configuration, and 3
analysis infrastructure failure.

Add `--json` to emit newline-delimited `gomadv3.explore-event/v2` records on
stdout and no routine output on stderr. Event types are `progress`, `result`,
`artifact`, and `error`. Result classifications are `success`,
`target_failure`, `watchdog_observation`, `replay_divergence`, and
`mixed_failure`; error classifications are `invalid_input`,
`unsupported_target`, `semantic_coverage_failure`, and `runner_failure`.

Use `--coverage=semantic`, `--coverage=choice`, or
`--coverage=semantic+choice` to retain versioned semantic probes, canonical
choice features, or both. Choice coverage requires `--choices`. Repeat `--require-probe`
to make an unobserved known probe fail the campaign with classification
`semantic_coverage_failure` and status 1:

```sh
tools/gomadv3/.bin/gomad explore --coverage=semantic \
  --require-probe=stdlib.os.openfile --count 100 go-test ./path/to/package
```

Use `--guide --corpus DIR` to feed replay-verified, semantically novel seeds
back into later campaigns. Guidance enables semantic coverage unless an
explicit incompatible `--coverage` was supplied. Each batch selects from one
immutable corpus snapshot: at most three quarters of its seeds come from the
corpus and at least one quarter remain in the requested seed set. Corpus cases
are ranked by reproducible failures, invariant and terminal states, abstract
World and I/O outcomes, operation and transition pairs, boundary probes, and
smaller reproductions. World feature values omit seeds, internal identities,
logical times, resource keys, and payloads.

The corpus is private, single-writer, and bounded to 1,024 cases and 1 GiB. Its
identity binds the prepared target and arguments, pinned toolchain, reviewed
boundary, semantic instrumentation, and record contract. Every entry retains
the exact-replay artifact, seed, captured I/O and World identities, semantic
coverage, novelty reasons, and matching replay result. A case is published and
replayed before the canonical corpus index advances atomically; interrupted
unreferenced cases are removed when the corpus next opens. A changed identity,
corrupt case, divergent replay, symbolic-link corpus, concurrent writer, or
capacity violation fails visibly. Human and JSON results report the corpus
path, retained entry count, and additions made by the batch.

Guidance currently reuses realized seeds and transcripts; it does not mutate
World scenarios, faults, or inputs and never forces runtime choices. Those
extensions require evidence that retained seeds cannot reproduce minimized
failures. Code coverage remains separate from versioned semantic probes and is
not collected by this mode.

Successful runs are discarded from the batch by default; guided corpus
retention is independent. `--keep-successes=novel` retains the first completed
success that adds a new semantic probe or choice feature and therefore requires
semantic or choice coverage; `--keep-successes=all` retains every success. Both modes
require a positive `--success-limit` and `--success-bytes`. Crossing either
bound fails the campaign visibly instead of silently dropping replay evidence.
Each retained success is an immutable exact-replay artifact, and its stored byte
count and novelty reasons are recorded in the batch journal. Success replay
returns status 0 only when the recorded successful outcome matches.

`gomad qualify` prepares and executes the target independently two or more
times with one seed, compares bounded canonical evidence, and automatically
retains a private `gomadv3.qualification/v4` report below
`ARTIFACTS/qualifications/v4`. Readers also normalize v1 through v3 reports. Evidence includes the exact target, argv,
toolchain and Runner identities, full output hashes, transcript, captured-mount
identity, World identity, outcome, semantic probes, and optional choice
features. Replay evidence is attached to its corresponding repetition. Add
`--replay-successes` with explicit positive `--success-limit` and
`--success-bytes` bounds to retain and replay every success. Repeat `--require-probe`
to enforce known conditional probes; `--repeat` is bounded to 2 through 32.
Add `--json` for newline-delimited `gomadv3.qualify-event/v1` progress, result,
and error records. Unsupported targets retain their first boundary and exact
command in the qualification report.

Run or validate a versioned qualification manifest explicitly with:

```sh
tools/gomadv3/.bin/gomad qualify-set \
  --manifest=/absolute/path/to/manifest.json \
  --working-dir=/absolute/path/to/target/module \
  --artifacts=.gomad/qualification --output=qualification-set.json
tools/gomadv3/.bin/gomad qualify-set --check \
  --manifest=/absolute/path/to/manifest.json \
  --working-dir=/absolute/path/to/target/module
```

Manifest v3 binds the expected module, tier, invariant, ordered seeds, choice
capacity, capability mode, successful-replay requirement, and explicit
retention bounds. The
orchestrator analyzes every workload before executing any supported target,
checkpoints after each completed phase, and publishes a private, path-free
`gomadv3.qualification-set-report/v6`. Unsupported analysis is completed
evidence and is never executed. Readers normalize report v2 through v5 while
marking unavailable historical dimensions explicitly. Status 0 means all expectations
matched, 1 means a retained mismatch, 2 means invalid input, and 3 means
cancellation, timeout, child, or publication infrastructure failure.

Compare two validated reports with `gomad compare-support`. Clean and improved
comparisons return 0, regressions or review-required changes return 1,
incomparable inputs return 2, and output failures return 3. A boundary change
prints an exact domain-separated digest; approval applies only when
`--approve-boundary-diff=SHA256` matches that digest. Expectation matching and
actual supported/unsupported counts remain separate.

`make -C tools/gomadv3 core-qualification` runs the checked
`qualification/core.json` corpus from its self-contained fixture module. Its
five assertion-based workloads cover concurrent state invariants, filesystem
lifecycle semantics, loopback TCP request/response, SQLite commit/rollback,
and the direct modernc/libc file boundary. The aggregate and all evidence are
retained below `.toolchain/core-qualification*`.

The checked sixteen-workload Temporal corpus currently qualifies 5 workloads
and retains 11 exact unsupported analyses. Every qualified workload runs two
seeds and requires matching execution, World, I/O, and choice-tape replay.

An interrupted campaign retains a canonical `gomadv3.batch-plan/v5` beside
its prepared target. A guided plan also records the selected corpus snapshot
identity and the already-mixed seed selection, so resume never reselects seeds.
The current plan records the seed or choice-frontier strategy, its controller
identity, every search bound, immutable-segment limits, simultaneous partial
runs, and success, failure, transcript, and aggregate artifact capacities. New
unsharded batch v3 and sharded batch v4 publications reference `runs/index.json`, which binds each private,
zero-padded JSONL segment by record count, byte count, and SHA-256. Readers
retain narrow support for published batch v1/v2 and interrupted plan v1-v4
records.

`gomad plan` publishes a canonical `gomadv3.campaign-plan/v1` and adjacent
private bundle containing the verified prepared target and complete bounded
copies of configured read-only mount trees. Plan identity is independent of
the plan output path and original mount source paths. The initial protocol
accepts only unguided seed campaigns with `--on-failure=all`; dynamically
discovered choice-frontier prefixes require a later round coordinator.
`gomad run-shard --shard INDEX/COUNT` uses a zero-based ordinal-modulo
partition, revalidates the entire bundle before execution, and records global
selection ordinals in batch v4. `gomad merge` accepts only shards from the same
plan, rejects duplicate or missing ordinals unless `--partial` is explicit,
deduplicates retained evidence by content identity, enforces aggregate bounds,
and publishes a new `gomadv3.merged-batch/v1` without mutating shard artifacts.
Both plan and aggregate are available through `gomad inspect`.
The batch store records the explicit `planned`, `prepared`, `running`,
`committing`, `published`, and `recoverable-failure` lifecycle. A validated
`batch.json` is authoritative even when a crash leaves private state behind.
`gomad recover BATCH` locks the batch and either finishes that private cleanup,
normalizes an interrupted commit to its validated running state, or reports
that the batch is invalid or not recoverable without changing it. Add `--json`
for the stable `gomadv3.recovery/v1` result. Invalid or non-recoverable input
returns status 2; storage, locking, and publication failures return status 3.

`gomad resume BATCH` uses the same store-owned preflight, locks that batch, verifies the exact
Runner, toolchain, I/O profile, prepared binary, completed records, and every
referenced failure or successful-run artifact, archives incomplete per-seed state, and schedules
only unfinished selection ordinals. Closed run segments remain immutable;
resume may incorporate one contiguous segment whose rename completed before
its index update, and it archives an active segment before excluding only a
torn terminal record. It appends to and eventually publishes the original
batch; repeated resumes are safe when the recorded aggregate deadline is too
short to finish all remaining seeds. Published batches, changed inputs,
concurrent resumes, and interrupted preparation fail closed. `gomad inspect`
reports the index identity, segment totals, journal limits, and artifact
capacity. Add `--json` to use the same stable campaign event stream as
`explore`.

| Status | `explore` / `resume` | `qualify` | `replay` |
| --- | --- | --- | --- |
| 0 | All selected or remaining runs succeeded. | Every repetition succeeded with identical evidence. | Verification-only succeeded, or a retained success replayed exactly. |
| 1 | A target failure, watchdog observation, or replay divergence was retained. | Evidence diverged, a target failed, a required probe was absent, or replay diverged. | The stored observation reproduced exactly, or replay diverged; inspect `reproduced=true|false`. |
| 2 | Input is invalid, the target is unsupported, or the resume journal is incompatible. | Input was invalid or the unsupported boundary was retained. | Input or artifact compatibility validation failed. |
| 3 | Runner or host infrastructure failed. | Qualification or report infrastructure failed. | Replay infrastructure failed. |

The Runner prepares one immutable target, launches every seed or forced-prefix candidate in a fresh
contained process and work directory, enforces wall deadlines, computes full
stream hashes while retaining bounded output, and publishes canonical,
content-addressed artifacts. `--count N` selects seeds `0` through `N-1` and is
mutually exclusive with `--seeds`. Arguments following `--` use an argv-safe
interface. Trusted repository tooling preparing an `exec` target must use
`target.ReviewCapabilityClosure` and `target.WriteProvenance` to produce v2
provenance for the exact binary. Runner revalidates its package policy, pinned
standard-library membership, module closure, build information, and binary
identity; v1 or arbitrary binaries are rejected.

`gomad inspect` validates the batch journal or immutable failure/success artifact before
printing its identity, outcome, transcripts, captured mounts, truncation,
distinct failure paths, retained successes and byte totals, novelty reasons,
copy-paste replay commands, and batch lifecycle, resumability, repairability,
and recovery reason. Interrupted batches can be inspected before publication.
Add `--json` for the stable `gomadv3.inspect/v3` report.

### Deterministic I/O

Every Runner-managed target uses the versioned deterministic-I/O boundary by
default. It is independent of the target package, arguments, and application:

```sh
tools/gomadv3/.bin/gomad explore \
  --seeds 7 --parallel 1 --run-timeout 2m --overall-timeout 5m \
  --artifacts .gomad/qualify/seed-7 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Schema-v2 artifacts must contain this deterministic-I/O identity and its
matching environment marker. Profile-less v2 artifacts are rejected as
incomplete; replay never falls back to host I/O.

Gomad replaces supported loopback TCP operations, filesystem operations,
hostname, and entropy with process-local in-memory implementations. Optional
built-in adapters are an immutable collection generated from `version.json`.
The current version-pinned `modernc.org/libc` adapter redirects supported
filesystem, entropy, and time operations to those same generic boundaries.
The exact `google.golang.org/grpc@v1.80.0` adapter removes its Unix raw-socket
keepalive callback because Gomad's in-memory TCP connections have no kernel
socket to configure; it preserves the negative `KeepAlive` value and does not
claim kernel keepalive support.
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
the production boundary manifest or shipped compiler table. `make intercept-test`
builds a temporary compiler from a Go overlay containing those fixtures, proves
the production compiler ignores their package paths, and then runs the positive
and fail-closed compiler cases through that test-only compiler.

Every modeled operation is appended to a bounded shared-memory transcript.
Retained artifacts keep the canonical transcript, and replay supplies it to
the target through a read-only shared-memory region so divergence stops at the
first mismatching ordinal:

```sh
tools/gomadv3/.bin/gomad replay ARTIFACT_DIR
tools/gomadv3/.bin/gomad replay --verify-only ARTIFACT_DIR
```

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
tools/gomadv3/.bin/gomad explore \
  --io-ro-mount ./fixtures=/fixtures \
  --seeds 7 --parallel 1 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Mount sources are resolved relative to the Runner working directory; target
destinations are normalized into its virtual absolute namespace and may not
overlap. Symlinks, hard-linked files, special entries, unstable captures, and
capacity overflow fail closed. Write-capable opens within mounts return
`EROFS`, and reads outside declared mounts do not fall through to host files.

The compatibility-only `GOMADV3_RUN`, `GOMADV3_PACKAGES`, and `GOMADV3_ARGS`
variables are trusted Make recipe shell fragments, not an argv-safe public
interface. Shell metacharacters and values that require quoting must be quoted
for both Make and the recipe shell.

The stable Go command is `tools/gomadv3/.toolchain/bin/go`. The build verifies
the official Go source checksum, snapshots and validates
`toolchain/runtime/go1.26.4.patch` and `toolchain/runtime/overlay`, rejects
upstream overlay collisions, copies the exact overlay
snapshot, applies the exact patch snapshot with zero fuzz, and caches immutable
builds by the Go version, source checksum, patch and overlay checksums, host OS
and architecture, bootstrap Go version, and canonical build environment.
Same-key builds use an atomic owner lock, and ambient Go experiment,
architecture, C/C++ tool, and compiler/linker tuning is cleared before
`make.bash`. Set `GOMADV3_BOOTSTRAP_GO` to choose a bootstrap `go` command.

Host-side policy is implemented in typed Go packages. `toolchain` provides the
build, patch, validation, and upgrade interface; `toolchain/cmd/gomadtool` is
its command adapter, and `toolchain/internal/conformance` owns bounded
black-box fixture execution and semantic result classification. The remaining
scripts are reviewed argv adapters:
POSIX compatibility entrypoints, the two upstream `-exec`/`-toolexec`
adapters, and the Darwin-only DTrace audit. `make validate` rejects an
unowned script or new Bash/Perl policy. Linux CI exercises the platform-neutral
host packages, but does not qualify the Gomad runtime on Linux.

To upgrade Go, update the canonical `toolchain/version/version.json` descriptor
and `deterministicio/boundary/manifest.json`, materialize the old patch against the new pinned
source, and regenerate the patch with `go -C tools/gomadv3 run
./toolchain/cmd/gomadtool patch-regenerate --root="$PWD/tools/gomadv3"
--candidate-root=GO-SOURCE-ROOT`. The `regenerate-patch.sh GO-SOURCE-ROOT`
compatibility entrypoint delegates to the same typed command. `make -C
tools/gomadv3 generate` derives the Make, Go, compiler-spec,
interception-report, public-inventory, and upgrade-guide consumers. The
descriptor's patch and overlay allowlists must exactly equal the checked trees.

Run the version-specific command from the generated upgrade guide, or directly:

```sh
make -C tools/gomadv3 upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>
```

The command first requalifies the neutral core corpus, then publishes
`.toolchain/upgrade-dossier.json` even when a behavioral gate fails. It records
the complete upstream patch, semantic boundary diff, interception evidence,
overlay collision audit, disabled upstream results, mandatory probes,
host-clock audit, the checked `gomadv3-core` corpus, and platform qualification.
The boundary diff compares canonical complete entries, including generated hook
policies and fields introduced by a newer manifest, instead of projecting onto
the fields known to the previous dossier implementation.
A dossier cannot report `qualified=true` without that canonical corpus report,
a baseline boundary manifest, and either an empty boundary diff or explicit
approval. After reviewing a non-empty diff, rerun with
`GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256=<boundary_manifest_diff.sha256>` to
record approval for that exact canonical diff. Supported-host CI reads the same
value from the `GOMADV3_APPROVED_BOUNDARY_DIFF_SHA256` repository variable, so
an administrator can approve and rerun an intentional boundary-change check.
CI uploads both the dossier and its retained core-corpus evidence
on every run.

The standard-library boundary is declared in
`tools/gomadv3/deterministicio/boundary/manifest.json`, and the cross-process
deterministic-I/O layouts are declared in
`tools/gomadv3/deterministicio/schema/iowire.json`. After changing
either schema or its templates, regenerate and verify the derived artifacts
with:

```sh
make -C tools/gomadv3 generate
make -C tools/gomadv3 validate
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

Enabled targets start at midnight UTC on 2000-01-01. Standard `time.Now`,
monotonic elapsed time, sleeps, timers, tickers, callbacks, and context
deadlines use the process virtual clock. When no goroutine is runnable, the
runtime advances directly to the earliest native timer deadline. Runnable
work is never skipped to deliver a future timer, and equal-deadline timers use
the seeded runtime choice stream. An explicit `testing/synctest` bubble keeps
its private clock and takes precedence over the process clock.

The standard `go test` harness also observes virtual time. In particular,
`-test.timeout` is a logical-time deadline and may fire immediately in wall
time when it is the next event. A separate wall-time process watchdog is still
required for CPU loops, unsupported host operations, and toolchain failures.

For a fixed toolchain, architecture, program, deterministic external inputs,
and seed, supported runtime-controlled choices repeat across fresh processes.
Different seeds explore different choices when alternatives exist. Runtime
choices must finish before output or other external I/O is performed.
When v2 choice recording is enabled, exact replay forces stable logical
goroutine and select-poll alternatives independent of their physical queue
order, consumes the complete tape, and still compares final observation
records. Choice traces and tapes remain explicitly byte-bounded; overflow is a
Runner failure and cannot claim exact replay.

Deterministic mode supports internally linked pure-Go targets on the qualified
`darwin/arm64` host. Enabled cgo or externally linked binaries fail before package
initialization. Windows, plugins, foreign threads, the race detector, signals,
finalizers, and host-dependent network, filesystem, process, and other I/O
readiness are outside the contract. Launch targets compile with
`CGO_ENABLED=0` and set `TZ=UTC`. The public `go-test` target preserves only
explicit `--build-tag` values; Temporal's root wrapper selects `test_dep`
explicitly.

The runtime system monitor is disabled with asynchronous preemption, so a
CPU-bound goroutine or `select` polling loop may run forever and prevent
virtual-time advancement. Unsupported blocking I/O is likewise bounded by the
external wall watchdog rather than treated as a clock event. Calling
`runtime.GOMAXPROCS` to raise the value after startup is unsupported.

The Go test driver retains at most 1 MiB from each child output stream while
continuing to drain both streams. Every harness result directory records
`output-truncated` separately from `timed-out` and the child `status`. The Gomad
Runner has a separate configurable per-stream limit that defaults to 8 MiB.

The mode is intended only for trusted tests. Deterministic map seeds remove a
hash-randomization defense and must not be enabled in production. Each process
uses one P, so run different seeds in separate processes for parallelism. The
shared runtime random state also means program changes can change later choices.
Exact choice tapes are bound to the target, pinned toolchain build key,
platform, and choice-controller implementation and are not portable across
those identities.

## Compatibility-pack development

Compatibility packs use only the strict `gomadv3.compatibility-pack/v2`
contract. Every allowed fact is bound to an exact module, complete compiled Go
and foreign-source inventories, a package source-set digest, governance, and a
`darwin/arm64` platform scope. Local module replacements are rejected unless
they are created by a registered deterministic-I/O adapter and carry the exact
profile, adapter, original/replacement inventory, and prepared source-set
identities.

The development workflow is discover, review, exact approval, generate, check,
and qualify:

```sh
go -C tools/gomadv3 run ./toolchain/cmd/gomadtool compatibility-pack discover \
  --root="$PWD/tools/gomadv3" --request=target/internal/compatibility/requests/<id>.json \
  --working-dir=<target-module>
go -C tools/gomadv3 run ./toolchain/cmd/gomadtool compatibility-pack review \
  --root="$PWD/tools/gomadv3" --request=target/internal/compatibility/requests/<id>.json \
  --output=target/internal/compatibility/reports/<id>.md
go -C tools/gomadv3 run ./toolchain/cmd/gomadtool compatibility-pack generate \
  --root="$PWD/tools/gomadv3" --request=target/internal/compatibility/requests/<id>.json \
  --approve-review=<exact-review-sha256>
make -C tools/gomadv3 validate compatibility-pack-qualification
```

Malformed or non-canonical requests and packs are invalid input. Source,
toolchain, module-cache, adapter, publication, and cleanup failures are
infrastructure failures. Fresh-review disagreement is unsupported drift. None
of these cases falls back to an older pack, partial inventory, arbitrary local
replacement, host access, or truncated evidence. Requests, generated v2 packs,
review reports, mutation fixtures, and their generation manifest live under
`target/internal/compatibility`.

The obsolete `temporal-backoff-overflow` request was retired after the exact
gRPC adapter removed its active blockers and the workload qualified with exact
replay. The `xnet-socket-activity-candidate` request remains unapproved: it
covers only four facts within its current 60-blocker closure and still requires
direct linkname/syscall containment proof.

## Simulation contract

`simulation/parity/manifest.json` is the canonical SIM-0 behavioral contract.
It maps thirteen Gomad v2 behaviors to named v3 cases, exact source
tests, intentional replacement decisions, delivery stages, limits, and
backend/fidelity requirements. Twelve cases now have implemented in-process
evidence; fresh arbitrary package globals remain planned for the process tier.

The root `tools/gomadv3sim` package defines the no-dependency application
harness. Its v4 schemas provide bounded specs, stable node and incarnation
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
Process-backed hard isolation remains unimplemented and is not implied by the
in-process model.

## World

`world` is a pure in-memory model for deterministic events outside the Go
runtime. It performs no host I/O, starts no goroutines, invokes no callbacks,
and requires no runtime hook. Callers register requests, mark them ready, and
explicitly quiesce to choose and deliver ready events.

`world/mailbox` is the initial explicit adapter. It demonstrates lifecycle,
snapshot/restore, and replay without giving World ownership of application
state. `runner/internal/execution` composes World semantic records with the
Runner's raw process record while keeping those identities separate. A target connects
its World with `world/target.Open`, takes the session-owned World returned by
`Session.World`, performs all modeled work, and calls
`Session.Finish` after that work has stopped, or `Session.FinishError` for a
typed World error. The trusted bootstrap validates replay input before target
activation; `Open` installs that recorded initial World rather than accepting a
target-created substitute and returns it through `Session.World` before modeled work.
The session writes one bounded record with a structured idle, deadlock,
capacity, invalid-input, or replay-divergence terminal result through inherited
descriptors only at the process boundary, so host pipe readiness cannot affect
event ordering. Connected replay requires the executing child to emit the same
semantic bundle and fails closed if it is missing or divergent.

## Design

- [Glossary](GLOSSARY.md) defines the ubiquitous language in one page.
- [Architecture](ARCHITECTURE.md) records the durable runtime, Runner, World,
  artifact, replay, and deterministic-I/O decisions.

## Development

Run source validation and the black-box suite with:

```sh
make -C tools/gomadv3 test
make -C tools/gomadv3 test-builder
make -C tools/gomadv3 test-runtime
make -C tools/gomadv3 test-upstream
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 world-test
make -C tools/gomadv3 core-qualification
make -C tools/gomadv3 upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>
```

`test` retains the full gate and runs the builder, runtime, and upstream tiers
in that order. The focused targets reproduce the corresponding portion without
weakening the full gate.

The suite compares disabled `go run` and `go test` behavior with a local stock
Go 1.26.4 toolchain; benchmarks disabled clock reads against that toolchain;
covers the fixed clock, native timer behavior, context deadlines, logical test
timeouts, nested synctest, cgo/link rejection, non-progress, bounded output,
and deadlock; runs focused upstream `runtime`, `time`, and `testing/synctest`
tests; audits map key families across seeds; and repeats prebuilt map and
scheduler fixtures under distinct allocation layouts and bounded unrelated CPU
load. Supported-host CI additionally runs `make clock-audit`: a privileged,
positive-controlled DTrace gate that rejects seeded calls to `clock_gettime` or
`mach_absolute_time` after Gomad activation. Set `GOMADV3_STOCK_GO` when the
stock Go executable cannot be resolved
from the module-selected toolchain in `PATH`; the test never downloads one.

The address-only check disables automatic GC so retained padding changes the
tested layout without adding GC activity that consumes the shared seeded
runtime stream. The ordinary repeatability and host-load checks retain the
default GC behavior.

Generated source, binaries, downloads, and toolchain builds remain under
`tools/gomadv3/.toolchain` and are not committed.
