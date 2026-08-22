# Gomad v3 CLI: from first target to qualified release

Gomad v3 has two command-line products:

- `gomad` is the user workflow. It reviews a target, explores executions, retains evidence, replays observations, and qualifies support.
- `gomadtool` is the maintainer workflow. It builds and validates the toolchain, governs generated contracts and compatibility packs, runs conformance campaigns, and produces upgrade evidence.

This guide follows one target through both worlds. It is intentionally a journey rather than a flag catalog. The [product specification](tools/gomadv3/SPEC.md) defines the corresponding requirements.

```text
maintain toolchain
       |
       v
doctor -> analyze -> explore -> inspect -> replay -> minimize
                         |                    |
                         +-> recover/resume <-+
                         |
                         v
               qualify -> qualify-set -> compare-support
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
make gomadv3
```

The user command is then:

```sh
tools/gomadv3/.bin/gomad
```

Maintainer examples invoke the developer command directly from its module:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool COMMAND
```

Commands that prepare a target use one of three forms:

```sh
tools/gomadv3/.bin/gomad explore go-run ./path/to/main -- application-argument
tools/gomadv3/.bin/gomad explore go-test ./path/to/package -- '-test.run=^TestName$'
tools/gomadv3/.bin/gomad explore exec --provenance ./target.provenance.json -- ./target application-argument
```

`--` ends Gomad's target description and begins the target's arguments. The `exec` form is for a trusted prebuilt binary with exact Gomad provenance; it is not an escape hatch for an arbitrary executable. `analyze` accepts only `go-run` and `go-test`, because its job is to derive and review the Go dependency boundary.

## Step 1: make sure the road exists with `doctor`

Your first target is a test that occasionally fails under concurrency. Before spending time exploring it, ask whether this installation can make a trustworthy claim at all:

```sh
tools/gomadv3/.bin/gomad doctor
```

`doctor` checks the host platform, resolved toolchain, Runner and boundary identities, deterministic-interaction adapters, and access to the Artifact root. It also explains where the installation came from and prints a location-specific repair when the complete contract is unavailable.

For automation, request stable JSON and check the Artifact directory you intend to use:

```sh
tools/gomadv3/.bin/gomad doctor --json --artifacts=.gomad/artifacts
```

An unavailable installation returns status 1. Invalid arguments return 2, and a failure to inspect or report the installation returns 3. Do not move on to a campaign merely because a patched `go` binary exists; `doctor` verifies the larger product contract.

## Step 2: ask whether the target fits with `analyze`

The installation is healthy, but that does not mean the target stays inside Gomad's modeled boundary. Review it without launching it:

```sh
tools/gomadv3/.bin/gomad analyze \
  --format=json \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

The default closure review examines the target's package dependency closure. It is fast and conservative: reachable packages can contribute blockers even when the final linker would remove their code.

When that distinction matters, build and inspect the final linked target without running it:

```sh
tools/gomadv3/.bin/gomad analyze \
  --capability-mode=linked \
  --timeout=5m \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Guarded review is the third mode. It preserves explicitly guarded findings as evidence while separating them from active blockers according to the current compatibility policy.

Analysis answers a narrower question than execution: “Can this exact target be prepared under the reviewed boundary?” Status 0 means supported, 1 means unsupported, 2 means invalid input or package configuration, and 3 means analysis infrastructure failed. Fix or explicitly govern a blocker before exploring; do not treat unsupported analysis as a flaky test result.

## Step 3: take the first trip with `explore`

Start with a small seed range and let each seed run in a fresh process:

```sh
tools/gomadv3/.bin/gomad explore \
  --seeds=0-99 \
  --parallel=4 \
  --execution-timeout=30s \
  --overall-timeout=10m \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`explore` prepares the target once, then executes the selected seeds under bounded concurrency. `--count=100` is shorthand for seeds 0 through 99 and cannot be combined with `--seeds`. Parallel completion timing does not change selection order or durable publication order.

The default failure policy stops after the first retained failure. Use `--on-failure=budget --failure-budget=N` to stop after N distinct failure signatures, or `--on-failure=all` to finish the complete selection.

Human-readable progress goes to stderr. The final classification, Campaign path, retained Artifact paths, and copy-paste replay commands go to stdout. The final classification distinguishes a target failure, watchdog observation, replay divergence, mixed failure, and success.

### Add evidence before adding search

Seeds control runtime decisions, but they do not explain which decisions mattered. Record bounded choice evidence and semantic coverage:

```sh
tools/gomadv3/.bin/gomad explore \
  --choices \
  --coverage=semantic+choice \
  --choice-bytes=8MiB \
  --seeds=0-99 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Repeat `--require-probe=NAME` when a known semantic boundary must be observed. Missing a required probe becomes a visible Campaign failure rather than an optimistic coverage report.

Successful executions are discarded by default. Retain only successes that add coverage:

```sh
tools/gomadv3/.bin/gomad explore \
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
tools/gomadv3/.bin/gomad explore \
  --guide \
  --corpus=.gomad/corpus \
  --count=1000 \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Guidance selects from one immutable corpus snapshot while reserving part of the Campaign for the originally requested seeds. The corpus advances only after a retained case replays exactly. Guidance reuses observed seeds and transcripts; it does not claim to mutate scenarios or enumerate schedules.

### Move from sampling to bounded explorations

Seed exploration samples schedules. When one execution exposes concrete runnable or `select` alternatives, choice-exploration exploration follows those alternatives in deterministic breadth-first rounds:

```sh
tools/gomadv3/.bin/gomad explore \
  --strategy=choice-exploration \
  --seeds=7 \
  --max-executions=128 \
  --max-choice-depth=32 \
  --max-exploration-bytes=64MiB \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

The strategy requires one base seed and explicit positive bounds. It implies choice recording and does not combine with `--count` or guided exploration.

For a Gomad simulation target, simulation-exploration exploration can coordinate runtime, scenario, network, storage, fault, and crash-state alternatives. Every dimension is explicit so “complete” always means complete within a declared envelope:

```sh
tools/gomadv3/.bin/gomad explore \
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

## Step 4: follow the evidence with `inspect`, `replay`, and `minimize`

The Campaign reports a retained failure path. Inspect the Campaign first to understand the whole search:

```sh
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/campaign-CAMPAIGN
```

Then inspect the immutable failure itself:

```sh
tools/gomadv3/.bin/gomad inspect \
  --choices \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

`inspect` validates before reporting. For a Campaign it shows lifecycle, selection, journal, limits, exploration state, failures, retained successes, and replay commands. For an Artifact it shows the exact Target, outcome, output hashes, transcript, captured mounts, World and simulation evidence, and choice trace.

Before executing anything, you can verify that the Artifact is internally complete and compatible:

```sh
tools/gomadv3/.bin/gomad replay \
  --verify-only \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

Then reproduce the stored observation using the retained binary and recorded inputs:

```sh
tools/gomadv3/.bin/gomad replay \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

Replay never rebuilds from today's source tree and never substitutes live input. Reproducing a retained failure returns status 1 because the target-level failure still occurred; a matching retained success returns 0. Status 2 means the input or compatibility contract was invalid, while status 3 means replay infrastructure failed.

If the failure came from combined simulation and exact runtime and simulation replay are available, reduce it:

```sh
tools/gomadv3/.bin/gomad minimize \
  --attempt-budget=64 \
  .gomad/artifacts/v1/campaign-CAMPAIGN/failures/sha256-ARTIFACT
```

`minimize` tries bounded candidates in fresh processes. It accepts a reduction only when the normalized failure, outcome, runtime choices, and simulation replay remain exact. The original Artifact stays immutable; the result records its parent and every accepted reduction. This command is currently specific to supported combined-simulation target failures, not a general-purpose test reducer.

## Step 5: turn a reproduction into a support claim

One replay answers whether one observation repeats. `qualify` asks whether independent repetitions of the same seed produce equal bounded evidence:

```sh
tools/gomadv3/.bin/gomad qualify \
  --seed=7 \
  --repeat=3 \
  --choices \
  --replay-successes \
  --success-limit=1 \
  --success-bytes=128MiB \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

Qualification prepares and executes independently for each repetition, compares canonical evidence, and retains its own report. Optional successful replay proves that a passing observation is reproducible, not merely equal by summary.

A product claim usually contains more than one workload. First validate the qualification manifest without running targets:

```sh
tools/gomadv3/.bin/gomad qualify-set \
  --check \
  --manifest="$PWD/qualification-set.json" \
  --working-dir="$PWD/path/to/target"
```

Then execute it and publish the aggregate report:

```sh
tools/gomadv3/.bin/gomad qualify-set \
  --manifest="$PWD/qualification-set.json" \
  --working-dir="$PWD/path/to/target" \
  --artifacts=.gomad/qualification \
  --output=.gomad/qualification-report.json \
  --format=json
```

`qualify-set` analyzes every workload before executing any supported Target, checkpoints completed phases, retains unsupported analyses, and compares results with declared expectations.

On the next release or branch, compare the new report with the baseline:

```sh
tools/gomadv3/.bin/gomad compare-support \
  --baseline=.gomad/baseline-qualification.json \
  --candidate=.gomad/qualification-report.json
```

Clean and improved support return 0. Regressions or changes that require review return 1; incomparable reports return 2. When the reviewed boundary changed intentionally, inspect the reported digest and approve exactly that identity:

```sh
tools/gomadv3/.bin/gomad compare-support \
  --baseline=.gomad/baseline-qualification.json \
  --candidate=.gomad/qualification-report.json \
  --approve-boundary-diff=sha256:REVIEWED_DIFFERENCE_HEX
```

Approval is not a wildcard. It applies only to the exact canonical difference printed by the comparison.

## Step 6: distribute a campaign with `plan`, `execute-shard`, and `merge`

The local workflow is trustworthy, but the seed set is too large for one host. Freeze a supported seed Campaign into a portable plan:

```sh
tools/gomadv3/.bin/gomad plan \
  --seeds=0-999 \
  --output=campaign.plan.json \
  go-test ./path/to/package -- '-test.run=^TestName$'
```

`plan` packages the verified prepared Target, complete selection, identities, bounds, environment, and captured read-only inputs. The current portable format accepts unguided seed Campaigns and fixes the failure policy to complete all planned work.

Run deterministic ordinal-modulo shards, potentially on different compatible workers:

```sh
tools/gomadv3/.bin/gomad execute-shard --shard=0/4 campaign.plan.json
tools/gomadv3/.bin/gomad execute-shard --shard=1/4 campaign.plan.json
tools/gomadv3/.bin/gomad execute-shard --shard=2/4 campaign.plan.json
tools/gomadv3/.bin/gomad execute-shard --shard=3/4 campaign.plan.json
```

Each worker revalidates the complete bundle and reports its published Campaign path. Pass those exact paths to `merge`:

```sh
tools/gomadv3/.bin/gomad merge \
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
tools/gomadv3/.bin/gomad inspect .gomad/artifacts/v1/campaign-INTERRUPTED
```

Inspection reports whether the Campaign is published, resumable, repairable, or invalid. If publication stopped in a recognized repairable storage state, repair that state without executing unfinished work:

```sh
tools/gomadv3/.bin/gomad recover .gomad/artifacts/v1/campaign-INTERRUPTED
```

`recover` either completes safe private cleanup, normalizes an interrupted commit to its validated state, or refuses to change the directory. It does not resume target execution.

Once the Campaign is resumable, continue it:

```sh
tools/gomadv3/.bin/gomad resume .gomad/artifacts/v1/campaign-INTERRUPTED
```

`resume` verifies the original Runner, toolchain, prepared binary, strategy, bounds, completed records, and retained Artifacts. It locks the Campaign, archives incomplete work, and schedules only unfinished logical ordinals or exploration rounds. Published, changed, incompatible, or concurrently resumed Campaigns fail closed.

The practical order is therefore always `inspect`, then `recover` only when inspection calls for repair, then `resume`.

## Step 8: automate without losing meaning

Commands expose machine-readable output where it is part of their contract:

- `explore`, `execute-shard`, and `resume` use newline-delimited progress, result, Artifact, and error events with `--json`.
- `doctor`, `inspect`, `recover`, `minimize`, and `merge` emit one stable JSON result with `--json`.
- `analyze`, `qualify-set`, and `compare-support` select text or JSON with `--format`.
- `qualify` emits newline-delimited qualification events with `--json`.

Across user workflows, exit statuses preserve the same broad meaning:

| Status | Meaning |
|---:|---|
| 0 | The requested operation completed and its success condition held. |
| 1 | A target-level failure, mismatch, divergence, unavailable installation, or review-required result was retained. |
| 2 | Input was invalid, unsupported, incompatible, or incomparable. |
| 3 | Gomad infrastructure or output publication failed. |

Replay refines status 1 to mean that a retained failure reproduced or that replay diverged; inspect its result rather than interpreting status 1 as a generic command crash.

## Step 9: return to the maintainer side with `gomadtool`

The user journey depends on a maintained deterministic product. `gomadtool` provides the lower-level commands used by Make and CI to keep that product reproducible. These commands are intentionally more exacting: they operate on canonical release inputs and produce evidence for review.

### Keep generated contracts synchronized

Three commands generate or verify different canonical domains:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool version-generate --check --root=.
go -C tools/gomadv3 run ./cmd/gomadtool protocol-generate --check --root=.
go -C tools/gomadv3 run ./cmd/gomadtool boundary-generate --check --root=.
```

- `version-generate` derives consumers of the release descriptor.
- `protocol-generate` derives both endpoints and tests for the declared cross-process protocols.
- `boundary-generate` derives the reviewed host-capability inventory and compiler interception evidence.

Without `--check`, these commands update generated outputs. `boundary-generate` also supports focused maintenance modes: `--discover` lists candidate host-capability entry points, `--qualify` verifies declared signatures and candidate coverage, `--refresh` updates reviewed source fingerprints, and `--check-compiler-tests` validates compiler conformance declarations.

### Maintain the runtime patch and toolchain

Validate the governed runtime patch and overlay:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool patch-validate --root=.
```

Apply the exact patch to an already verified Go source tree when inspecting or updating it:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool patch-materialize \
  --root=. \
  --source-root=/absolute/path/to/go-source
```

After making reviewed changes in a candidate source tree, regenerate the canonical patch:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool patch-regenerate \
  --root=. \
  --candidate-root=/absolute/path/to/modified-go-source
```

`build-key` derives the cache identity from the Go release and archive digest, patch, overlay, host, bootstrap toolchain, recipe, and sterile build environment. Build scripts normally call it because omitting any of those dimensions would make reuse unsafe.

`toolchain-build` performs the verified build and immutable publication:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool toolchain-build --root=.
```

The ordinary repository entry point remains `make gomadv3`; direct maintainer commands are useful when debugging one stage or qualifying an upgrade.

### Extend support through compatibility packs

A compatibility pack is not handwritten policy dropped into the tree. It moves through a reviewable workflow. Starting from a draft request below the compatibility directory, discover the exact source facts:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool compatibility-pack discover \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --working-dir=/absolute/path/to/target-module
```

Publish a human review and capture the digest it prints:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool compatibility-pack review \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --output=internal/compatibilitypack/reports/PACK.md
```

After reviewing that exact report, generate only with its exact approval digest:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool compatibility-pack generate \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --approve-review=sha256:REVIEWED_REPORT_HEX
```

Then qualify the request against its target and verify the complete generated set:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool compatibility-pack qualify \
  --root=. \
  --request=internal/compatibilitypack/requests/PACK.json \
  --working-dir=/absolute/path/to/target-module

go -C tools/gomadv3 run ./cmd/gomadtool compatibility-pack check --root=.
```

Calling `compatibility-pack generate --root=.` without a request or approval regenerates already approved packs; it does not approve a new request.

### Run bounded conformance commands

`test` executes one selected conformance campaign against an explicit toolchain:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool test \
  --root=. \
  --mode=test-builder \
  --go="$(command -v go)"
```

The available tiers cover the builder, live capability semantics, runtime behavior, interception cases, and disabled upstream compatibility. The complete supported-platform claim requires the aggregate gate; a passing neutral builder tier alone is not runtime qualification.

`checked-run` is the small bounded process adapter beneath several scripted checks. It verifies an expected exit status and records stdout, stderr, status, timeout, and truncation separately:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool checked-run \
  30 0 go-version .toolchain/checked-go-version -- \
  "$PWD/tools/gomadv3/.toolchain/bin/go" version
```

`script-validate` keeps shell at reviewed argument and platform boundaries:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool script-validate --root=.
```

### Close the loop with `upgrade-dossier`

When the Go release, patch, overlay, adapters, or reviewed boundary changes, run the complete upgrade workflow rather than choosing a few reassuring tests:

```sh
make -C tools/gomadv3 upgrade-dossier \
  GOMADV3_BASELINE_REF=BASELINE_COMMIT
```

The Make target supplies the retained core qualification report and invokes:

```sh
go -C tools/gomadv3 run ./cmd/gomadtool upgrade-dossier \
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
| `qualify-set` | Validate or run a manifest of qualification workloads. |
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
| `boundary-generate` | Discover, qualify, generate, refresh, or check the capability boundary. |
| `compatibility-pack` | Discover, review, generate from exact approval, qualify, and check compatibility packs. |
| `script-validate` | Enforce the reviewed script ownership and policy boundary. |
| `checked-run` | Run and record one bounded external command with expected status. |
| `test` | Execute a selected conformance campaign. |
| `upgrade-dossier` | Run upgrade gates and retain the complete acceptance evidence. |
