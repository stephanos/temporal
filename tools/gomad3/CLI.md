# Gomad v3 CLI: from first target to qualified release

Gomad v3 has two command-line products:

- `gomad` is the user workflow. It reviews a target, explores executions, retains evidence, replays observations, and qualifies support.
- `gomadtool` is the maintainer workflow. It builds and validates the toolchain, governs generated contracts and compatibility packs, runs conformance campaigns, and produces upgrade evidence.

This guide follows one target through both workflows. The [product specification](SPEC.md) defines the requirements and owns the [canonical vocabulary](SPEC.md#productvocabulary-ubiquitous-language).

```text
maintain toolchain
       |
       v
doctor -> analyze -> explore -> inspect -> replay -> minimize
                         |                    |
                         +-> recover/resume <-+
                         |
                         v
               qualify -> qualify-set -> merge-set -> compare-support
                         |
                         v
                 plan -> execute-shard -> merge
                         |
                         v
             conformance -> upgrade-dossier
```

## Before the journey: command and target syntax

From the repository root, build the supported toolchain and user command:

```sh
make gomad3
```

The user command is then:

```sh
tools/gomad3/.bin/gomad
```

Maintainer examples invoke the developer command directly from its module:

```sh
go -C tools/gomad3 run ./cmd/gomadtool COMMAND
```

Commands that prepare a target use one of three forms:

```sh
tools/gomad3/.bin/gomad explore go-run ./path/to/main -- application-argument
tools/gomad3/.bin/gomad explore go-test ./path/to/package -- '-test.run=^TestName$'
tools/gomad3/.bin/gomad explore exec --provenance ./target.provenance.json -- ./target application-argument
```

`--` ends Gomad's target description and begins the target's arguments. The `exec` form is for a trusted prebuilt binary with exact Gomad provenance; it is not an escape hatch for an arbitrary executable. `analyze` accepts only `go-run` and `go-test`, because its job is to derive and review the Go dependency boundary.

Place Gomad flags before the target kind. `go-test` accepts one package and passes the arguments after `--` to the compiled test binary; use `-test.run`, not the `go test` driver's `-run`. Repeat `--build-tag=TAG` for each required build tag. Public `go-test` commands add no implicit `test_dep` tag; Temporal's root wrapper selects it explicitly.

The qualified hosts are `darwin/arm64` and `linux/amd64`. Targets compile with `CGO_ENABLED=0` and execute with `TZ=UTC`; toolchain, platform, reviewed boundary, and inputs are part of the recorded identity.

## Step 1: make sure the road exists with `doctor`

Your first target is a test that occasionally fails under concurrency. Before spending time exploring it, ask whether this installation can make a trustworthy claim at all:

```sh
tools/gomad3/.bin/gomad doctor
```

`doctor` checks the host platform, resolved toolchain, Runner and boundary identities, deterministic-interaction adapters, and access to the Artifact root. It also explains where the installation came from and prints a location-specific repair when the complete contract is unavailable.

For automation, request stable JSON and check the Artifact directory you intend to use:

```sh
tools/gomad3/.bin/gomad doctor --json --artifacts=.gomad/artifacts
```

An unavailable installation returns status 1. Invalid arguments return 2, and a failure to inspect or report the installation returns 3. Do not move on to a campaign merely because a patched `go` binary exists; `doctor` verifies the larger product contract.

## Step 2: ask whether the target fits with `analyze`

The installation is healthy, but that does not mean the target stays inside Gomad's modeled boundary. Review it without launching it:

```sh
tools/gomad3/.bin/gomad analyze \
  --format=json \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

The default closure review examines the target's package dependency closure. It is fast and conservative: reachable packages can contribute blockers even when the final linker would remove their code.

When that distinction matters, build and inspect the final linked target without running it:

```sh
tools/gomad3/.bin/gomad analyze \
  --capability-mode=linked \
  --timeout=5m \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`--capability-mode=guarded` is the third mode. Like linked review, it inspects the linked target. It retains findings protected by reviewed runtime guards separately from active blockers; reaching a denied guarded operation still fails closed during execution.

Analysis defaults to a 30-second wall limit in closure mode and two minutes in linked or guarded mode. `--timeout` can raise the bound to at most 30 minutes. Select the same capability mode when exploring the reviewed target; a standalone analysis does not change later command defaults.

Analysis answers a narrower question than execution: “Can this exact target be prepared under the reviewed boundary?” Status 0 means supported, 1 means unsupported, 2 means invalid input or package configuration, and 3 means analysis infrastructure failed. Fix or explicitly govern a blocker before exploring; do not treat unsupported analysis as a flaky test result.

## Step 3: take the first trip with `explore`

Start with a small seed range and let each seed run in a fresh process:

```sh
tools/gomad3/.bin/gomad explore \
  --seeds=0-99 \
  --parallel=4 \
  --execution-timeout=30s \
  --overall-timeout=10m \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`explore` prepares the target once, then executes the selected seeds under bounded concurrency. `--count=100` is shorthand for seeds 0 through 99 and cannot be combined with `--seeds`. Parallel completion timing does not change selection order or durable publication order.

Without an explicit selection, `explore` uses seed 1. Default concurrency is the smaller of the host CPU count and 8; the wall limits are 30 seconds per execution and 10 minutes overall. The test binary's `-test.timeout` measures virtual time, so it does not replace the wall watchdog.

The default failure policy stops after the first retained failure. Use `--on-failure=budget --failure-budget=N` to stop after N distinct failure signatures, or `--on-failure=all` to finish the complete selection.

Human-readable progress goes to stderr. The final classification, Campaign path, retained Artifact paths, and copy-paste replay commands go to stdout. The final classification distinguishes a target failure, watchdog observation, replay divergence, mixed failure, and success.

### Add evidence before adding search

Seeds control runtime decisions, but they do not explain which decisions mattered. Record bounded choice evidence and semantic coverage:

```sh
tools/gomad3/.bin/gomad explore \
  --choices \
  --coverage=semantic+choice \
  --choice-bytes=8MiB \
  --seeds=0-99 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`--choices` retains a Choice Trace, from which exact runtime Replay derives an identity-bound Decision Tape. Coverage is a separate summary of observed semantic events or runtime choices. Repeat `--require-probe=NAME` with `--coverage=semantic` or `--coverage=semantic+choice` when a known semantic boundary must be observed. Missing a required probe becomes a visible Campaign failure.

`--diagnostics` on `explore` and `plan` records runtime-state diagnostics and implies `--choices`; it accepts only `--strategy=seed` and rejects forced-prefix exploration. Diagnostic traces support locating differences between fresh executions. They do not establish exact replay.

Choice recording defaults to 8 MiB and accepts at most `--choice-bytes=64MiB`; overflow fails visibly. Output retention defaults to 8 MiB per stream, adjustable with `--output-limit`. Deterministic I/O transcript capacity defaults to 64 MiB and accepts `--io-transcript-bytes` from 64 MiB through 1 GiB in whole MiB increments. A larger transcript does not increase choice capacity.

With the default `--clock-tick=strict` policy, virtual time stands still while work is runnable, so repeated `time.Now` reads can tie. A test that orders records by timestamp may fail on those ties. Add `--clock-tick=forward` to `explore` or `qualify` to add a cumulative seeded offset of 1 to 1024 nanoseconds per `time.Now` read. The native timer clock still advances only when work cannot proceed, and `time.Since` and `time.Until` read that clock for a reading that still carries its monotonic value: an elapsed time measured from such a reading can be short or negative, a deadline computed from it can expire later than a timer set for the same duration, and timestamps truncated above nanosecond resolution can still tie. A reading stripped of its monotonic value, as by serialization or parsing, is compared against a fresh ticked `time.Now` instead. The policy is part of the Campaign and Artifact identity, and replay restores it.

To run a module that lives elsewhere, such as one that depends on the server through a local `replace`, pass `--working-dir=/absolute/module/root` instead of changing directories.

Successful executions are discarded by default. Retain only successes that add coverage:

```sh
tools/gomad3/.bin/gomad explore \
  --choices \
  --coverage=semantic+choice \
  --keep-successes=novel \
  --success-limit=16 \
  --success-bytes=256MiB \
  --count=1000 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Success retention always needs explicit count and byte limits. Crossing either limit fails visibly instead of silently discarding evidence.

### Let prior evidence guide later seeds

Once you have replay-verified semantic evidence, a bounded corpus can guide later seed selection:

```sh
tools/gomad3/.bin/gomad explore \
  --guide \
  --corpus=.gomad/corpus \
  --count=1000 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Guidance excludes seeds answered by replay-verified, matching cases in one immutable corpus snapshot. It executes the requested selection minus those seeds, substitutes nothing, and reports requested, answered, guided, and new execution counts. With no unanswered corpus seed to prioritize, guidance selects none. A fully answered request executes zero seeds and exits 0. Add `--guide-regression` to re-run corpus cases; this mode reserves at least one quarter of the selection, rounded up, for the requested seed pool. Resume preserves the recorded selection and mode; an explicit conflicting `resume --guide-regression=true|false` is rejected.

### Move from sampling to Choice Exploration

Seed exploration samples schedules. When one execution exposes concrete runnable or `select` alternatives, Choice Exploration follows those alternatives in deterministic breadth-first rounds:

```sh
tools/gomad3/.bin/gomad explore \
  --strategy=choice-exploration \
  --seeds=7 \
  --max-executions=128 \
  --max-choice-depth=32 \
  --max-exploration-bytes=64MiB \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

The strategy requires one base seed and explicit positive bounds. It implies choice recording and does not combine with `--count` or guided exploration.

`--choice-start-ordinal=N` expands only replay-plan decisions at ordinal N or later, while retaining earlier decisions in each forced prefix. The default is 0. Find the ordinals with `inspect --choices`; resume restores the recorded start ordinal. Using this flag with seed or Combined Exploration is invalid input. Select polls with fewer than two ready cases are recorded but do not expand the Frontier; results report their omitted alternatives separately.

For a Gomad simulation target, Combined Exploration coordinates runtime, scenario, network, storage, fault, and crash-state alternatives through `--strategy=simulation-exploration`. Every dimension is explicit so “complete” always means complete within the declared bounds:

```sh
tools/gomad3/.bin/gomad explore \
  --strategy=simulation-exploration \
  --seeds=7 \
  --max-executions=128 \
  --max-forced-decisions=32 \
  --max-runtime-decisions=32 \
  --max-scenario-decisions=32 \
  --max-network-decisions=32 \
  --max-storage-decisions=32 \
  --max-fault-decisions=32 \
  --max-crash-decisions=32 \
  --max-exploration-bytes=64MiB \
  --max-exploration-result-bytes=16MiB \
  go-test ./path/to/simulation -- '-test.run=^TestScenario$'
```

Both strategies retain the Frontier of remaining alternatives. Their reports identify alternatives omitted by the applicable execution, depth, dimension, or capacity bounds. Inspect that evidence before interpreting a completion claim.

## Step 4: follow the evidence with `inspect`, `replay`, and `minimize`

The Campaign reports a retained failure path. Inspect the Campaign first to understand the whole search:

```sh
tools/gomad3/.bin/gomad inspect .gomad/artifacts/v1/campaign-CAMPAIGN
```

Then inspect the immutable failure itself:

```sh
tools/gomad3/.bin/gomad inspect \
  --choices \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

`inspect` validates before reporting. For a Campaign it shows lifecycle, selection, journal, limits, exploration state, failures, retained successes, and replay commands. For an Artifact it shows the exact Target, outcome, output hashes, transcript, captured mounts, World and simulation evidence, and choice trace.

Before executing anything, you can verify that the Artifact is internally complete and compatible:

```sh
tools/gomad3/.bin/gomad replay \
  --verify-only \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

Then reproduce the stored observation using the retained binary and recorded inputs:

```sh
tools/gomad3/.bin/gomad replay \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

Replay never rebuilds from today's source tree and never substitutes live input. Reproducing a retained failure returns status 1 because the target-level failure still occurred; a matching retained success returns 0. Status 2 means the input or compatibility contract was invalid, while status 3 means replay infrastructure failed.

Read `choice-replay` in the result to distinguish recorded runtime Choice replay from seed-based repetition. An Artifact without a supported recorded Choice Trace cannot claim exact runtime Choice replay. A watchdog observation uses diagnostic replay and returns status 1 even when the observation matches; matching a wall-time termination does not prove exact replay.

If the failure came from combined simulation and exact runtime and simulation replay are available, reduce it:

```sh
tools/gomad3/.bin/gomad minimize \
  --attempt-budget=64 \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

`minimize` tries bounded candidates in fresh processes. It accepts a reduction only when the normalized failure, outcome, runtime choices, and simulation replay remain exact. The original Artifact stays immutable; the result records its parent and every accepted reduction. This command is currently specific to supported combined-simulation target failures, not a general-purpose test reducer.

To continue an interrupted minimization, repeat the command with `--resume`, the same parent Artifact, `--artifacts` root, and bounds. The parent has its own persisted checkpoint under that root, including attempt order, consumed budget, accepted reductions, and replay evidence. Missing or corrupt state, changed identities or bounds, and concurrent writers are refused; resume does not reset the attempt budget.

## Step 5: turn a reproduction into a support claim

One replay answers whether one observation repeats. `qualify` asks whether independent repetitions of the same seed produce equal bounded evidence:

```sh
tools/gomad3/.bin/gomad qualify \
  --seed=7 \
  --repeat=3 \
  --choices \
  --replay-successes \
  --success-limit=1 \
  --success-bytes=128MiB \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Qualification prepares and executes independently for each repetition, compares canonical evidence, and retains its own report. Optional successful replay proves that a passing observation is reproducible, not merely equal by summary.

The default is seed 1 with two repetitions; `--repeat` accepts 2 through 32. Qualification collects semantic coverage, and `--choices` adds choice coverage and runtime replay evidence. `qualify --diagnostics` records runtime-state diagnostics for its fresh repetitions and implies `--choices`. `--replay-successes` requires explicit count and byte bounds per repetition.

The two flags are independent and opt-in, and the claim follows the flags. With neither, `qualified` means same-seed repeatability: fresh repetitions produced equal evidence, and nothing was replayed. With `--choices` the compared evidence includes the Choice Trace's tape digest, but a success keeps its Decision Tape only when it is retained. With `--replay-successes` alone each retained success is replayed from its seed and the result reports `choice-replay=none`. With both flags each retained success is replayed from its tape, and the result's `choice-replay=exact` is the verified choice-tape replay claim. A qualification-set workload states the same choice with `choice_bytes`, `replay_successes`, and its success limits, and there success replay requires a Choice Trace; an untraced workload's seeds report `replayed: false` and no `choice_replay_exact`.

A product claim usually contains more than one workload. First validate the qualification manifest without running targets:

```sh
tools/gomad3/.bin/gomad qualify-set \
  --check \
  --manifest="$PWD/qualification-set.json" \
  --working-dir="$PWD/path/to/target"
```

Then execute it and publish the aggregate report:

```sh
tools/gomad3/.bin/gomad qualify-set \
  --manifest="$PWD/qualification-set.json" \
  --working-dir="$PWD/path/to/target" \
  --artifacts=.gomad/qualification \
  --output=.gomad/qualification-report.json \
  --format=json
```

`qualify-set` analyzes every workload before executing any supported Target, checkpoints completed phases, retains unsupported analyses, and compares results with declared expectations.

A set too large for one machine's budget runs as shards. `--shard INDEX/COUNT` is the same zero-based ordinal-modulo partition `execute-shard` uses, applied to the manifest's workloads, so shards never overlap and together cover the manifest. Each shard publishes an ordinary set report that carries the whole manifest's digest; `merge-set` then combines them into the report one run would have published:

```sh
for shard in 0/3 1/3 2/3; do
  tools/gomad3/.bin/gomad qualify-set \
    --manifest="$PWD/qualification-set.json" \
    --working-dir="$PWD/path/to/target" \
    --artifacts=.gomad/qualification-${shard%/*} \
    --output=.gomad/qualification-shard-${shard%/*}.json \
    --shard="$shard"
done
tools/gomad3/.bin/gomad merge-set \
  --manifest="$PWD/qualification-set.json" \
  --output=.gomad/qualification-report.json \
  .gomad/qualification-shard-*.json
```

`merge-set` refuses shards of another manifest or run configuration, a repeated or missing workload, and a shard count larger than the manifest; it publishes no partial aggregate. Its statuses match `qualify-set`: 0 when every expectation matched, 1 when the merged report retains a mismatch, 2 for invalid shards, and 3 when the report cannot be written.

On the next release or branch, compare the new report with the baseline:

```sh
tools/gomad3/.bin/gomad compare-support \
  --baseline=.gomad/baseline-qualification.json \
  --candidate=.gomad/qualification-report.json
```

Clean and improved support return 0. Regressions or changes that require review return 1; incomparable reports return 2. When the reviewed boundary changed intentionally, inspect the reported digest and approve exactly that identity:

```sh
tools/gomad3/.bin/gomad compare-support \
  --baseline=.gomad/baseline-qualification.json \
  --candidate=.gomad/qualification-report.json \
  --approve-boundary-diff=sha256:REVIEWED_DIFFERENCE_HEX
```

Approval is not a wildcard. It applies only to the exact canonical difference printed by the comparison.

## Step 6: distribute a campaign with `plan`, `execute-shard`, and `merge`

The local workflow is trustworthy, but the seed set is too large for one host. Freeze a supported seed Campaign into a portable plan:

```sh
tools/gomad3/.bin/gomad plan \
  --seeds=0-999 \
  --output=campaign.plan.json \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`plan` packages the verified Prepared Target, complete selection, identities, bounds, environment, and captured read-only inputs. The portable format accepts seed Campaigns and fixes the failure policy to complete all planned work. Guided plans freeze the corpus snapshot, selection, and regression mode; shards execute that selection without opening or updating the live corpus.

Workers must match the plan's platform and recorded toolchain, Runner, boundary, adapter, and compatibility-pack identities. Portability distributes work among compatible workers; it does not make a target binary portable across architectures or operating systems.

Run deterministic ordinal-modulo shards, potentially on different compatible workers:

```sh
tools/gomad3/.bin/gomad execute-shard --shard=0/4 campaign.plan.json
tools/gomad3/.bin/gomad execute-shard --shard=1/4 campaign.plan.json
tools/gomad3/.bin/gomad execute-shard --shard=2/4 campaign.plan.json
tools/gomad3/.bin/gomad execute-shard --shard=3/4 campaign.plan.json
```

Each worker revalidates the complete bundle and reports its published Campaign path. Pass those exact paths to `merge`:

```sh
tools/gomad3/.bin/gomad merge \
  --output=.gomad/merged-campaign \
  campaign.plan.json \
  /absolute/path/to/shard-zero-campaign \
  /absolute/path/to/shard-one-campaign \
  /absolute/path/to/shard-two-campaign \
  /absolute/path/to/shard-three-campaign
```

`merge` rejects mixed plans, overlapping ordinals, unexplained gaps, corrupt evidence, and aggregate capacity overflow. It deduplicates retained evidence by content and never mutates shard Campaigns. Use `--partial` only when publishing an explicitly incomplete aggregate is the intended result; missing ordinal ranges remain visible.

## Step 7: survive interruption with `inspect`, `recover`, and `resume`

A machine dies halfway through a Campaign. Do not immediately rerun the original command: the interrupted directory contains the authority for what finished.

Start read-only:

```sh
tools/gomad3/.bin/gomad inspect .gomad/artifacts/v1/campaign-INTERRUPTED
```

Inspection reports whether the Campaign is published, resumable, repairable, or invalid. If publication stopped in a recognized repairable storage state, repair that state without executing unfinished work:

```sh
tools/gomad3/.bin/gomad recover .gomad/artifacts/v1/campaign-INTERRUPTED
```

`recover` either completes safe private cleanup, normalizes an interrupted commit to its validated state, or refuses to change the directory. It does not resume target execution.

Once the Campaign is resumable, continue it:

```sh
tools/gomad3/.bin/gomad resume .gomad/artifacts/v1/campaign-INTERRUPTED
```

`resume` verifies the original Runner, toolchain, prepared binary, strategy, bounds, completed records, and retained Artifacts. It locks the Campaign, archives incomplete work, and schedules only unfinished logical ordinals or exploration rounds. Published, changed, incompatible, or concurrently resumed Campaigns fail closed.

The practical order is therefore always `inspect`, then `recover` only when inspection calls for repair, then `resume`.

## Step 8: automate without losing meaning

Commands expose machine-readable output where it is part of their contract:

- `explore`, `execute-shard`, and `resume` use newline-delimited progress, result, Artifact, and error events with `--json`.
- `doctor`, `inspect`, `recover`, `minimize`, and `merge` emit one stable JSON result with `--json`.
- `analyze`, `qualify-set`, `merge-set`, and `compare-support` select text or JSON with `--format`.
- `qualify` emits newline-delimited qualification events with `--json`.

Across user workflows, exit statuses preserve the same broad meaning:

| Status | Meaning |
|---:|---|
| 0 | The requested operation completed and its success condition held. |
| 1 | A target-level failure, mismatch, ordinary or World replay divergence, unavailable installation, or review-required result was retained. |
| 2 | Input was invalid, unsupported, incompatible, or incomparable. |
| 3 | Gomad infrastructure or output publication failed, or a Choice Exploration forced-prefix candidate diverged. |

Choice Exploration returns status 3 for a forced-prefix candidate divergence, including mixed failures, because the search cannot trust that candidate. Ordinary seeded and World replay divergence retain status 1.

Replay refines status 1 to mean that a retained failure reproduced or that replay diverged; inspect its result rather than interpreting status 1 as a generic command crash.

## Step 9: return to the maintainer side with `gomadtool`

The user journey depends on a maintained deterministic product. `gomadtool` provides the lower-level commands used by Make and CI to keep that product reproducible. These commands are intentionally more exacting: they operate on canonical release inputs and produce evidence for review.

### Keep generated contracts synchronized

Four commands generate or verify different canonical domains:

```sh
go -C tools/gomad3 run ./cmd/gomadtool version-generate --check --root=.
go -C tools/gomad3 run ./cmd/gomadtool protocol-generate --check --root=.
go -C tools/gomad3 run ./cmd/gomadtool boundary-generate --check --root=.
go -C tools/gomad3 run ./cmd/gomadtool qualification-manifest-generate --check --root=../.. \
  --spec=tools/gomad3integration/qualification/tests.generator.json \
  --output=tools/gomad3integration/qualification/tests.json
```

- `version-generate` derives consumers of the release descriptor.
- `protocol-generate` derives both endpoints and tests for the declared cross-process protocols.
- `boundary-generate` derives the reviewed host-capability inventory and compiler interception evidence.
- `qualification-manifest-generate` derives a qualification-set manifest with one workload per top-level test of a package, the tests `go test -list` reports under the spec's build tags, from a spec of defaults, per-test overrides, and exclusions. `--check` fails when the manifest is stale relative to the package or the spec. The checked-in `./tests` spec defaults to no Choice Trace and no success replay; a test opts in by overriding `choice_bytes`, `replay_successes`, `success_artifact_limit`, and `success_bytes_limit` with a `reason`.

Without `--check`, these commands update generated outputs. `boundary-generate` also supports focused maintenance modes: `--discover` lists candidate host-capability entry points, `--qualify` verifies declared signatures and candidate coverage, `--refresh` updates reviewed source fingerprints, and `--check-compiler-tests` validates compiler conformance declarations.

### Maintain the runtime patch and toolchain

Validate the governed runtime patch and overlay:

```sh
go -C tools/gomad3 run ./cmd/gomadtool patch-validate --root=.
```

Apply the exact patch to an already verified Go source tree when inspecting or updating it:

```sh
go -C tools/gomad3 run ./cmd/gomadtool patch-materialize \
  --root=. \
  --source-root=/absolute/path/to/go-source
```

After making reviewed changes in a candidate source tree, regenerate the canonical patch:

```sh
go -C tools/gomad3 run ./cmd/gomadtool patch-regenerate \
  --root=. \
  --candidate-root=/absolute/path/to/modified-go-source
```

`build-key` derives the cache identity from the Go release and archive digest, patch, overlay, host, bootstrap toolchain, recipe, and sterile build environment. Build scripts normally call it because omitting any of those dimensions would make reuse unsafe.

`toolchain-build` performs the verified build and immutable publication:

```sh
go -C tools/gomad3 run ./cmd/gomadtool toolchain-build --root=.
```

The ordinary repository entry point remains `make gomad3`; direct maintainer commands are useful when debugging one stage or qualifying an upgrade.

### Extend support through compatibility packs

A compatibility pack is not handwritten policy dropped into the tree. It moves through a reviewable workflow. Starting from a draft request below the compatibility directory, discover the exact source facts:

For an exact dependency bump, first compare candidate and saved baseline
`go.mod` files, each with an adjacent `go.sum`:

```sh
go -C tools/gomad3 run ./cmd/gomadtool pin-impact --root=. \
  --baseline=/absolute/baseline/go.mod --candidate=/absolute/candidate/go.mod \
  --format=json > pin-impact.json
```

Status 1 means an invalidated or unknown pin. Review the adapter and pack-rule
entries before making changes. For each affected adapter, inspect the dry-run
source diff and exact anchor proposal, then explicitly approve its digest:

```sh
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate --root=. \
  --module=<module-path> --version=<exact-version>
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate --root=. \
  --module=<module-path> --version=<exact-version> --approve=sha256:<reviewed-digest>
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh --root=. \
  --impact-report=/absolute/path/to/pin-impact.json
```

Refresh discovers and renders fresh reviews for mapped affected requests on
the current host. It stops before approval and reports other-platform requests
without changing them. A supplied impact report covers only its candidate
module; omit `--impact-report` to scan every mapped directory for a checkout
bump. Review each changed report, then use the per-request
`compatibility-pack generate --approve-review` command below. Repeat refresh
and qualification on each supported host; a prior digest does not approve new
evidence. The [README](README.md#compatibility-pack-development) gives the
complete dependency-bump procedure.

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack discover \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --working-dir=/absolute/path/to/target-module
```

Publish a human review and capture the digest it prints:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack review \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --output=internal/compatibilitypack/reports/PACK.md
```

After reviewing that exact report, generate only with its exact approval digest:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack generate \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --approve-review=sha256:REVIEWED_REPORT_HEX
```

Then qualify the request against its target and verify the complete generated set:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack qualify \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --working-dir=/absolute/path/to/target-module

go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack check --root=.
```

Calling `compatibility-pack generate --root=.` without a request or approval regenerates already approved packs, and removes the packs and reports it no longer renders; it does not approve a new request.

`internal/compatibilitypack/working-directories.json` maps every request to the module directory, relative to the compatibility directory, that it is discovered and qualified in. `check` rejects a request without an entry, an entry whose directory holds no `go.mod`, and, for this repository's root, a missing table. `adapter-regenerate` verifies its staged copy of the module with `check --staged-copy`, which still requires the table but not a `go.mod` in the directories it maps outside the copy. `compatibility-pack qualify --root=. --all` qualifies every request naming the host platform in its mapped directory, which is what `make compatibility-pack-qualification` runs.

After a dependency bump is applied to the checkout, refresh every invalidated request up to approval in one command:

```sh
go -C tools/gomad3 run ./cmd/gomadtool compatibility-pack refresh \
  --root=.
```

Refresh runs the pin impact report once per mapped directory (the working tree is the candidate, `--baseline-ref`, default `HEAD`, the baseline) over the packs in the refreshed root's `packs/`, whatever `GOMAD3_COMPATIBILITY_PACKS` names. Pass `--baseline-ref=REV` if `HEAD` no longer holds the baseline, or `--go=GO` to select the pinned Go command. An explicit `--impact-report=FILE` is scoped to the module represented by that report and cannot discover pin changes in other mapped directories. Refresh selects the requests whose pack rules it reports invalidated or unknown, every request without an approval, and every host-platform request bound to a deterministic I/O profile other than the current one. It discovers each selected request in its own directory into memory and compares: a request whose stored approval equals the review digest of the fresh evidence is current and untouched, so an approval of older evidence never counts. Otherwise the fresh evidence is written with its approval cleared only when it differs from what is stored, the reports and packs are regenerated, and refresh prints the review digest with the `generate --approve-review` command to run after reviewing the report. Approval stays per request; rerunning after approving some requests reports only the others. A request for another platform is reported `not-evaluable` and left untouched, and a pack whose directory no longer requires its modules is reported `unselected`. Status 0 means nothing is left to do, 1 that a request awaits approval, cannot be evaluated here, failed, or is unselected, 2 invalid input such as a request without a working directory, and 3 an infrastructure failure.

For a downstream module, pass `--compatibility-root=/absolute/pack-root` to these authoring commands. That root owns `requests/`, `reports/`, `packs/`, `generation.json`, and, for `qualify --all` and `refresh`, `working-directories.json`; requests and review output must remain below it. Load the approved packs for user commands with `GOMAD3_COMPATIBILITY_PACKS=/absolute/pack-root/packs`. External packs undergo the same strict validation as embedded packs, and their exact identities must also be available for replay, resume, and shard execution.

### Run bounded conformance commands

`test` executes one selected conformance campaign against an explicit toolchain:

```sh
go -C tools/gomad3 run ./cmd/gomadtool test \
  --root=. \
  --mode=test-builder \
  --go="$(command -v go)"
```

The available tiers cover the builder, live capability semantics, runtime behavior, interception cases, and disabled upstream compatibility. The complete supported-platform claim requires the aggregate gate; a passing neutral builder tier alone is not runtime qualification.

`diagnostic-diff` compares two complete runtime diagnostic traces and reports the first divergent ordinal and fields:

```sh
go -C tools/gomad3 run ./cmd/gomadtool diagnostic-diff --json EXPECTED_TRACE ACTUAL_TRACE
```

Flags precede both trace paths. Status 0 means equal, 1 means different, 2 means invalid arguments or an unreadable, malformed, or incomplete trace, and 3 means output failed. Both traces are fully validated before comparison; differences are diagnostic evidence, not a replay guarantee.

`checked-run` is the small bounded process adapter beneath several scripted checks. It verifies an expected exit status and records stdout, stderr, status, timeout, and truncation separately:

```sh
go -C tools/gomad3 run ./cmd/gomadtool checked-run \
  30 0 go-version .toolchain/checked-go-version -- \
  "$PWD/tools/gomad3/.toolchain/bin/go" version
```

`script-validate` keeps shell at reviewed argument and platform boundaries:

```sh
go -C tools/gomad3 run ./cmd/gomadtool script-validate --root=.
```

### Bump a dependency

A dependency bump never widens a pin; three commands find and re-derive the pins it breaks, and a person approves each re-derived pin. Apply the bump with `go get MODULE@VERSION` in the target module, leave it uncommitted, and run:

1. `gomadtool pin-impact` to list the adapters, pack rules, interception fingerprints, and clock references the bump invalidates. For a module other than the repository root module, pass its directory with `--module=DIR`.
2. For each named adapter, `gomadtool adapter-regenerate --module=PATH --version=VERSION`, review the printed upstream diff, then repeat with `--approve-review=DIGEST`. Hand-edit any other reference to the previous version it lists.
3. `gomadtool compatibility-pack refresh --root=.`, review each printed report, and run the printed `compatibility-pack generate --approve-review=DIGEST` command per request. Each platform's host refreshes and approves its own requests.
4. Rebuild `.bin/gomad` with `make gomad3`, run `make -C tools/gomad3 validate compatibility-pack-qualification` on each platform, and requalify the regenerated adapter's workloads.

`pin-impact` and `refresh` compare against `HEAD` by default. A pack pinned to the replaced version stays invalidated after the bump is committed, because its module still requires the pack's activation modules at other versions. A module the bump removes is reported stale, and its pack `unselected`, only against a baseline that still requires it, so after committing such a bump pass the revision before it with `--baseline-ref`. `modernc.org/libc` uses the same review, approval, and publication workflow with a syntax-aware rewrite; a changed or ambiguous rewrite still needs a person. A Go release is not a dependency bump: it follows [`upgrade-dossier`](#close-the-loop-with-upgrade-dossier).

### Report the pins a dependency bump invalidates

Before applying a dependency bump, report which exact-version pins it breaks:

```sh
go -C tools/gomad3 run ./cmd/gomadtool pin-impact \
  --root=. \
  --module=/absolute/path/to/candidate-module
```

The candidate is the `go.mod` and `go.sum` in `--module`, by default the repository root module. The baseline is the same module at `--baseline-ref` (default `HEAD`), or the module in `--baseline-module`. The report reads adapter identities from the adapter registry, rules from the compatibility-pack loader, and the boundary manifest and reviewed host-clock inventory from `--root`, and judges each pin by exact module identity, so it covers every platform's packs. It lists invalidated adapters, pack rules with their source-set digests, and interception fingerprints and clock references, which only a candidate requiring a newer Go than the pinned release leaves unknown. When a dependency, not the candidate itself, requires a newer Go, the go command cannot resolve the module graph, so the adapter and pack pins the graph decides are unknown as well. Module graphs resolve in a scratch copy with a private module cache under the exported proxy settings, so the candidate's files never change. `--json` writes the path-free canonical report to stdout and `--output=FILE` also writes it to a file. Status 0 means no pin is invalidated, 1 means at least one pin is invalidated or unknown, 2 means invalid input, and 3 means an infrastructure failure such as an unreachable module proxy. A pin whose module the candidate no longer requires is reported stale and does not change the status. A pack that neither side selects although the candidate requires every one of its activation modules, at other versions, is invalidated, so a committed bump still reports the packs it stranded.

### Regenerate an adapter for a new module version

When the report names an adapter, re-derive its anchors for the new exact version. A dry run writes nothing:

```sh
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate \
  --module=google.golang.org/grpc --version=v1.84.0
```

It downloads the pinned and the candidate version into a private module cache under the exported proxy settings and applies each adapter's exact-occurrence rewrites, or the syntax-aware rewrite for `modernc.org/libc`. A missing or ambiguous match, a changed syntax form, or a rewritten file the candidate no longer provides stops the run with status 1 for human review. Otherwise it prints the upstream diff of every file a rewrite reads, the proposed anchors (sum, source inventories, each rewritten file's source and replacement digests, and each platform's prepared source set, all computed from source), the compatibility-pack bindings the change leaves stale, and an approval digest over the anchors and the reviewed sources. The proposed anchors must pass the adapter's own fail-closed preparation before they are printed. `--go` must be the pinned Go release, whose release tags select each platform's prepared files; `--json` writes the review as JSON.

Add `--stage-only` with the printed digest to stage and verify the regeneration and list every file the apply would publish, with the diff of each `go.mod` and `go.sum` that fixture tidying changed, without publishing anything. After reviewing the changed source, apply with the printed digest:

```sh
go -C tools/gomad3 run ./cmd/gomadtool adapter-regenerate \
  --module=google.golang.org/grpc --version=v1.84.0 \
  --approve-review=sha256:REVIEWED_DIGEST
```

An apply builds the complete output set in a scratch copy of the Gomad module first: the adapter constants and the tests that name the version, the `version.json` entry, the test fixture modules that require the module (then `go mod tidy`), and the outputs of the module-local `make generate` steps. It verifies the staged copy (the commands build, the adapter tests vet, generated files and packs check, and the staged adapter accepts the candidate under every pin) and then publishes under an exclusive lock, after checking that no checkout file changed since staging. Publication commits a journal under `.toolchain/adapter-regeneration` before it touches the checkout, so an interrupted publication is completed by the next apply or by `adapter-regenerate --recover`, and a journal that was never committed is discarded. A failed generation, a failed verification, a changed checkout, or a competing apply publishes nothing. A wrong digest is status 2. The apply lists any other references to the previous version for review; the stale packs are repaired by refreshing them, and `.bin/gomad` must be rebuilt and the adapter's workloads requalified because the adapter's identity changed. The syntax-aware `modernc.org/libc` rewrite follows the same approval and staged-publication path and fails closed if its reviewed source form changes.

### Close the loop with `upgrade-dossier`

When the Go release, patch, overlay, adapters, or reviewed boundary changes, run the complete upgrade workflow rather than choosing a few reassuring tests:

```sh
make -C tools/gomad3 upgrade-dossier \
  GOMAD3_BASELINE_REF=BASELINE_COMMIT
```

The Make target supplies the retained core qualification report and invokes:

```sh
go -C tools/gomad3 run ./cmd/gomadtool upgrade-dossier \
  --root=. \
  --baseline-ref=BASELINE_COMMIT \
  --corpus-report=.toolchain/core-qualification-set.json
```

`upgrade-dossier` runs validation, patch/compiler, Runner, World, probe, builder, runtime, disabled-upstream, host-clock, and cached-build gates. It publishes the dossier even when a completed gate rejects the upgrade, preserving the first failure and bounded diagnostic output.

If the reviewed boundary changed intentionally, review its reported digest and rerun with `--approve-boundary-diff=sha256:REVIEWED_DIFFERENCE_HEX`, substituting the exact lowercase digest. That approval covers only the exact boundary difference. A qualified dossier therefore connects the end of the maintainer journey back to the start: the next user's `doctor` can trust the installation it reports.

## Command index

### `gomad`

| Command | Role in the journey |
|---|---|
| `doctor` | Verify the installation before doing work. |
| `analyze` | Review a Go Target without launching it. |
| `explore` | Execute bounded seed or exploration Campaigns. |
| `inspect` | Validate and explain plans, Campaigns, aggregates, and Artifacts. |
| `replay` | Verify or reproduce a retained Artifact. |
| `minimize` | Reduce an eligible combined-simulation failure while preserving replay. |
| `qualify` | Compare independent repetitions of one Target and seed. |
| `qualify-set` | Validate or run a manifest of qualification workloads, whole or as one shard. |
| `merge-set` | Combine the shard reports of one manifest into its whole set report. |
| `compare-support` | Compare candidate support evidence with a baseline. |
| `plan` | Freeze a supported seed Campaign into a portable bundle. |
| `execute-shard` | Execute one deterministic shard of a portable plan. |
| `merge` | Publish a validated complete or explicitly partial shard aggregate. |
| `recover` | Repair a recognized interrupted publication state without running work. |
| `resume` | Continue only unfinished work in a validated interrupted Campaign. |

### `gomadtool`

| Command | Role in the journey |
|---|---|
| `toolchain-build` | Build, cache, and publish the pinned toolchain. |
| `build-key` | Derive the complete immutable build identity. |
| `patch-validate` | Validate the governed runtime patch and overlay. |
| `patch-materialize` | Apply the reviewed patch to a verified source tree. |
| `patch-regenerate` | Recreate the patch from a reviewed candidate tree. |
| `version-generate` | Generate or check release-descriptor consumers. |
| `protocol-generate` | Generate or check cross-process protocol endpoints and tests. |
| `qualification-manifest-generate` | Generate or check qualification workloads from a package's top-level tests and declared dispositions. |
| `boundary-generate` | Discover, qualify, generate, refresh, or check the capability boundary. |
| `compatibility-pack` | Discover, review, generate from exact approval, qualify, and check compatibility packs. |
| `compatibility-pack refresh` | Discover and review affected mapped requests without approving them. |
| `script-validate` | Enforce the reviewed script ownership and policy boundary. |
| `checked-run` | Run and record one bounded external command with expected status. |
| `diagnostic-diff` | Compare complete runtime diagnostic traces and locate the first divergence. |
| `pin-impact` | Report every pin a candidate `go.mod` invalidates before the build rejects it. |
| `adapter-regenerate` | Re-derive an adapter's anchors for a new module version behind an approval digest. |
| `test` | Execute a selected conformance campaign. |
| `upgrade-dossier` | Run upgrade gates and retain the complete acceptance evidence. |
