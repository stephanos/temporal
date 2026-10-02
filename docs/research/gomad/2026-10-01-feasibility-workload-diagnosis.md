# Feasibility: workload, feedback, diagnosis, and compute ideas in GOMAD_CMP.md

Dated research snapshot (2026-10-01); read-only code study, not a statement of current support.

Read-only study of `/Users/stephan/Workspace/temporal/gomad` on branch `stephanos/gomad`
(working tree, 2026-10-01). Nothing was built or run except `grep`/`sed`/small JSON reads of
reports already on disk. Paths are relative to the repo root. `g3/` abbreviates `tools/gomad3/`
and `sim/` abbreviates `tools/gomad3sim/`.

Each claim is tagged **[V]** verified in source or in a retained report, or **[I]** inferred
from what was read and not executed.

## Findings that change the plan

1. **The target cannot receive a per-execution input other than the scheduling seed.** [V]
   A campaign fixes argv, environment, and mounts once (`g3/runner/runner.go:126-177`); the
   per-execution job carries only `seed`, a choice tape, and a simulation exploration plan
   (`g3/runner/runner.go:316-324`). The simulation fixture hard-codes its simulation seed
   because "the runtime keeps the Seed out of the target's environment"
   (`sim/testdata/simulation_exploration/main.go:22-25`). A typed candidate has no delivery
   channel today.
2. **`--guide` re-executes executions whose result is already known.** [V selection logic,
   I consequence] Guidance puts corpus seeds first, up to 75% of the campaign
   (`g3/runner/seeds.go:27-84`), and the corpus identity binds the exact target and argv
   (`g3/runner/internal/corpus/model.go:98-147`, `corpus.go:285`). By the determinism contract
   those seeds reproduce their retained records, so a guided campaign runs fewer new executions
   than an unguided one of the same `--count`. The Guide is a retention and regression store
   until a mutation operator exists.
3. **The guided corpus cannot hold Temporal functional cases.** [V sizes, I consequence]
   Every artifact embeds the target binary (`g3/artifact/publication.go:44-53`); prepared
   `./tests` binaries on disk are 155 to 179 MB; the corpus is capped at 1 GiB
   (`g3/runner/internal/corpus/model.go:32-35`, `corpus.go:379-390`). That is about six cases.
4. **Choice exploration on a `testcore` suite never leaves cluster bootstrap.** [V code and
   counts, I consequence] Expansion starts at decision ordinal 0 and stops at
   `MaxChoiceDepth` with no start offset (`g3/runner/internal/exploration/choice/engine.go:47-57,
   349-384`). The bare cluster boot (`TestFrontendSystemInfo`) already records 4,562 choice
   records; suites record 8,000 to 208,000.
5. **Cluster bootstrap is most of a functional execution's wall time.** [V data, I reading]
   The boot-only probe takes 1.7 s; the functional suites take 1.5 to 3.2 s (table in
   section 6). This is the prefix-cost evidence CMP asks for before any snapshot work, and it
   already exists on disk.
6. **The simulation track's "Temporal" scenario is a toy.** [V]
   `sim/temporal_scenario_toolchain_test.go` uses raw `net.Listen`/`Dial` and one
   `collection.SyncMap`; no Temporal service code runs on simulated nodes.
7. **Nothing hunts bugs today.** [V] Every Make target and CI job runs `qualify`/`qualify-set`
   (same-seed repeatability and replay). `explore`, `--guide`, and `minimize` appear in no
   Makefile or workflow.
8. **Choice sites are symbolizable today without a toolchain change.** [V fields, I workflow]
   Records carry the caller's text offset (`g3/choice/wire.go:74-84`,
   `g3/toolchain/runtime/overlay/src/runtime/gomad.go:673-679`) and every artifact retains
   the unstripped binary; `inspect --choices` hashes the offset away
   (`g3/choice/trace.go:325-335`).

---

## 1. Typed workload generator and guided fuzzing loop

### (a) What exists

- **Scenario is Go code; its control data is JSON.** `type Scenario func(context.Context, Cluster) error`
  (`sim/types.go:535`). Combinators `Sequence`, `Repeat`, `Choose`, `BoundedParallel` take Go
  closures (`sim/scenario.go:52-165`). What is serializable and tool-mutable:
  - `Spec` (nodes, links, volumes, seed, limits): `sim/spec.go:90-128`, canonical JSON.
  - `FaultPlan` with eight kinds (stop, crash, restart, disconnect, reconnect, partition, heal,
    delay) and stable match fields: `sim/fault.go:21-68`, strict decode at `fault.go:100-127`.
  - `ScenarioChoicePlan` (overrides for `Choose` points): `sim/scenario_control.go:16-58`.
  - `ExplorationPlan` (forced decisions across runtime, scenario, network, storage, fault,
    crash): `sim/exploration.go:20-60`.
  - The realized `ScenarioDecision` tape (action/choose/parallel ordinals):
    `sim/scenario.go:21-29`.
- **How a target receives inputs from the Runner.**
  - argv and build tags, fixed per target identity (`g3/target/target.go:49-62`).
  - `--env` entries (`g3/cmd/gomad/internal/cli/cli.go:456`), validated against a reserved list
    (`g3/runner/runner.go:1355-1367`). At HEAD the patched `syscall.copyenv` hid them; the
    uncommitted patch in the working tree now restricts the scrub to direct `GOMADSEED`
    launches (`g3/toolchain/runtime/go1.27.1.patch:895-899`, D17 task 30, not yet qualified).
  - Lazy read-only mounts (`cli.go:458`; `g3/runner/internal/execution/process.go:75-79`).
  - World bootstrap: `world/process.Open` reads a config descriptor and installs the recorded
    World (`g3/world/process/session.go:26-65`).
  - Simulation: the coordinator-role target pulls an exploration plan and pushes one
    `ClusterRecord` over private frames (`sim/cluster.go:91-108, 208-212`;
    `sim/runtime_process.go:138-155`; `g3/runner/internal/execution/simulation.go:27-41, 67`).
  - The seed itself: environment `GOMADSEED` on the launch (`runner.go:1551-1556`) but never
    readable by Go code.
- **Per-execution variation.** `runJob` = ordinal, seed, choice mode, choice replay plan,
  simulation plan (`g3/runner/runner.go:316-324`). Simulation exploration reuses one base seed
  and varies only the plan (`g3/runner/simulation_exploration_campaign.go:245-256`).
- **Guide and corpus.** Features come from the outcome, World transitions, I/O operation names
  and results, operation pairs, boundary probes, and choice features
  (`g3/runner/internal/corpus/features.go:14-61`). Ranking is rarity within rank bands
  (`model.go:163-241`). Admission publishes the artifact, computes features, and replays before
  the index advances (`admission.go:21-44`). Output of the Guide is a list of seeds
  (`model.go:163`).
- **Semantic probes.** 131 probes, all standard library: 69 `stdlib.net`, 60 `stdlib.os`, one
  each `stdlib.ossignal` and `stdlib.osuser` (`g3/deterministicio/boundary_generated.go:13-`).
  The compiler inserts them into intercepted definitions from a fixed table
  (`g3/toolchain/runtime/overlay/src/cmd/compile/internal/gomadintercept/intercept.go:104-128`,
  `spec_go127.go`). The emitter `ObserveBoundary` lives in `internal/gomadtrace`, records each
  ID once per process, and has a fixed capacity of 256
  (`.../internal/gomadtrace/trace.go:38, 144-166`). An unknown probe in a transcript is a
  decode error (`g3/deterministicio/semantic_coverage.go:45-48`). **Application code cannot
  declare or emit a probe.**
- **Application-reachable feedback channels that already feed features.**
  - World transitions: a target that opens a World session contributes
    `world.register/<adapter>/<resource kind>/<request kind>` and pair features with strings it
    chooses (`corpus/features.go:63-134`). No in-repo target outside tests uses it.
  - Simulation `Cluster.Observe`, `RecordOperation`, `RecordOracle`
    (`sim/controller.go:245-362`) land in `ClusterRecord` (`sim/types.go:431-464`), but the
    corpus feature extractor never reads the simulation record, and guidance rejects both
    exploration strategies (`g3/runner/runner.go:1175, 1194`).

### (b) Gaps

- No per-execution input channel (finding 1). No `Candidate` type above `seed`.
- No API for the target to read its seed or an input blob. `math/rand` global draws are
  seed-derived through the runtime stream, so a generator using them is coupled to the
  schedule [I]; CMP wants independent workload, schedule, and fault seeds.
- Scenario bodies are closures, so "canonical scenario data, operation arguments, actor count"
  need a new interpreter layer: a data scenario (list of typed operations) executed by a
  Go-side dispatcher registered like `RegisterBoot` (`sim/registry.go:14`).
- Corpus identity omits environment and clock tick (`corpus/model.go:37-47, 98-107`). This is
  a hole today and also the cheapest hook for a first experiment (below).
- Feature schema has `FeatureCodeEdge` declared with a rank and no producer
  (`corpus/model.go:28, 234`).
- Guide loop steps 2, 5 (partly), and 6 of CMP's loop do not exist; steps 1, 3, 4, and the
  replay-before-admission half of 5 do.

### (c) Smallest experiment

Workload seed through the environment, no Runner change:

1. Land the D17 environment fix (already in the working tree).
2. Write one `go-test` target whose body draws its operation mix from
   `os.Getenv("WORKLOAD_SEED")` with its own PRNG (`FuzzMatcherData`'s tape interpreter is a
   ready-made operation generator, `service/matching/matcher_data_test.go:1132-1160`).
3. Run K campaigns `explore --env WORKLOAD_SEED=k --seeds 0-N` and compare distinct failure
   signatures and feature counts against one campaign of K×N seeds with a fixed workload.

This answers CMP's "Input search" gate (does input variation find anything schedule seeds do
not) for about a day of work, before building a candidate type.

### (d) Size and risk

- Experiment: **S**.
- Real typed candidate (job field, bootstrap frame or descriptor for the input blob, artifact
  payload, replay binding, corpus entry keyed by candidate rather than seed): **M to L**. The
  simulation exploration plan path is the template: `SimulationCapability.ExplorationPlan`
  already carries per-execution bytes to the target and a record back.
- Main risk: identity sprawl. Every new input must enter Campaign, Artifact, plan, corpus, and
  replay identity or it becomes a divergence source.

### (e) Stale or contradicted in GOMAD_CMP.md

- "Start with a typed workload generator and the existing Runner/Guide" understates the gap:
  the existing Runner has no slot for candidate data, and the existing Guide emits seeds only.
- "Guidance currently reuses observed seeds and transcripts" is accurate and omits the
  consequence in finding 2.
- "semantic/choice coverage" exists, but "novel semantic interactions" from the application
  cannot be expressed: probes are stdlib-only.

---

## 2. Go native fuzzing bridge

### (a) What exists

- Test targets build with `go test -c -trimpath -buildvcs=false` plus tags; no other build
  flags are exposed (`g3/target/target.go:702-727`, `target.Spec` at `:49-62`).
- Standard packages are exempt from the forbidden-import review (`g3/target/capability.go:687,
  869-871`), so `testing` → `internal/fuzz` → `os/exec` links.
- `os.StartProcess` is an inventoried boundary with disposition `deny`
  (`g3/deterministicio/boundary/manifest.json:2569-2578`).
- No mention of fuzzing anywhere in `tools/gomad3` source or docs (grep).
- Two native fuzz tests exist in the server: `FuzzMatcherData`
  (`service/matching/matcher_data_test.go:1132`) and `FuzzCalendar`
  (`service/worker/scheduler/calendar_test.go:372`). `./service/matching` already has three
  qualified workloads in `temporal.json`.

### (b) Answers and gaps

- **`-test.fuzz` under Gomad: no.** [I] The fuzz coordinator starts workers with `os/exec`,
  which reaches the denied `process.start` hook and fails closed. It also needs a host
  `-test.fuzzcachedir`, which the in-memory filesystem does not provide.
- **Seed-corpus mode (`-test.run=FuzzX`): should work.** [I] It runs `f.Add` entries and
  `testdata/fuzz/FuzzX/*` inline in one process. The corpus directory must be supplied with
  `--io-ro-mount`, because reads outside declared mounts do not reach host files
  (`g3/README.md:590-594`). Neither fuzz test has a `testdata/fuzz` directory today.
- **`-cover`: not accepted and not exported.** [V/I] There is no flag to request it
  (`target.Spec`, CLI `--build-tag` only at `cli.go:457`). If it were built, counters would be
  written through `os` into the in-memory filesystem and vanish at exit; nothing exports
  in-memory files. Coverage would also be a new execution profile with its own target
  identity, as CMP says.
- **Existing channels that could carry counters.** [I] The I/O transcript is a fixed-record
  shared-memory log of modeled operations (64 MiB default) and the wrong shape. The choice
  terminal frame is a fixed 312 bytes (`g3/choice/schema/choicewire.json:15-24`). The closest
  fit is a new inherited descriptor or shared-memory region declared in the launch plan, the
  way the choice trace mapping is (`gomad.go:203-249`); the corpus already reserves
  `code_edge` at the lowest rank.

### (c) Smallest experiment

Offline bridge, zero Gomad change:

1. Run stock `go test -fuzz=FuzzMatcherData -fuzztime=10m ./service/matching` natively to grow
   `testdata/fuzz/FuzzMatcherData`.
2. `gomad explore --io-ro-mount ./service/matching/testdata=<target path> --seeds 0-31
   go-test ./service/matching -- -test.run='^FuzzMatcherData$'`.
3. Record: does it run, how many inputs × seeds per second, any failure the native fuzzer did
   not report.

This tests CMP's "start by importing generated inputs" directly on a package that already
qualifies.

### (d) Size and risk

- Offline import: **S**. Risk: the working directory and `testdata` lookup path inside the
  in-memory namespace may need a mount destination that matches what `testing` computes.
- Online bridge (Gomad children returning features to an engine): **L**; needs the candidate
  channel from idea 1 plus a coverage profile.
- Coverage profile: **M**; risk is requalification of a second profile and replay identity.

### (e) Stale or contradicted

Nothing contradicted. CMP's caution ("needs a feedback bridge") is correct; the code adds that
even the non-bridged fuzz mode is closed by the process boundary.

---

## 3. Semantic feedback and Antithesis-style assertions

### (a) What exists

- Oracle constructors: `StateInvariant` (caller supplies the boolean), `ExactHistory`,
  `NoDuplicateOrLost` (multiset equality), `EventualConvergence` (byte equality across
  participants) (`sim/oracle.go:44-117`). Results are identity-hashed and recorded with
  `Cluster.RecordOracle`; the first failed oracle sets outcome `oracle_failed`
  (`sim/cluster.go:186-189`).
- `--require-probe` fails a campaign when a named probe is unobserved, and accepts only names
  in the generated stdlib table (`g3/deterministicio/semantic_coverage.go:86-111`).
- `Cluster.Observe` records arbitrary `(ID, Kind, Value)` observations
  (`sim/controller.go:245-291`).

### (b) Gaps

- No SDK callable from code under test. `always` exists only as "the test fails";
  `sometimes` and `reachable` have no representation: a campaign cannot say "property P was
  true in at least one execution" for an application-defined P.
- All sim evidence APIs hang off the `Cluster` value handed to the scenario, so service code
  several layers down cannot reach them without plumbing.
- An assertion catalogue needs: (1) a declaration step so "never reached" is distinguishable
  from "does not exist" (the stdlib table plays this role for probes); (2) an emit call legal
  from any goroutine that consumes no runtime randomness; (3) a transport out of the process;
  (4) per-campaign aggregation with `must-hit` semantics like `--require-probe`; (5) identity
  in the corpus instrumentation hash (`corpus/model.go:158-161`).

### (c) Smallest experiment

Use the World channel that already reaches the Guide. A 50-line helper in the target calls
`world/process.Open`, and `Sometimes(name)` registers a World request with
`Resource{Adapter: "assert", Kind: name}`. Those appear as `world.register/assert/<name>/...`
features with no Runner or toolchain change (`corpus/features.go:107-114`). Check that
`--coverage=semantic --keep-successes=novel` retains the first execution that hits each name.

Caveat [I]: `world.New` must be created with the execution's seed
(`world/process/session.go:55-57`) and the target cannot read it, so the helper probably has
to run under replay-style bootstrap or the session needs a `SeedFromConfig` constructor. That
would be the first concrete requirement the experiment surfaces.

### (d) Size and risk

- Helper experiment: **S**.
- First-class assertion catalogue (declared names, emit, campaign aggregation, CLI
  `--require-assertion`): **M**. Risk: an emit path inside production code needs a build-tag
  seam like `testhooks` (`common/testing/testhooks/test_impl.go`, tag `test_dep`), which is
  the natural host for it in Temporal.

### (e) Stale or contradicted

CMP says "check whether interesting situations occurred" as if a mechanism exists. Only the
stdlib-boundary version does.

---

## 4. Diagnosis

### (a) What exists

- `inspect --choices` prints: schema, profile, implementation hash, limit, payload bytes,
  record count, branching count, terminal state, tape hash, decision count, exact-replay
  availability, counts per kind (runnable, select-poll, select-result), and a list of sites as
  `{fingerprint, kind, count, maximum_alternatives}` (`g3/runner/inspect.go:166-190, 523-568`).
- A choice record carries: ordinal, kind, flags, alternatives, selected rank, data,
  `SiteOffset`, `SelectedIdentity`, `AlternativeSetDigest` (`g3/choice/wire.go:74-84`).
  - `SiteOffset` is the text-section offset of the `select` caller's PC
    (`gomad.go:673-679`; patch `go1.27.1.patch:685`).
  - Runnable decisions carry no site: flag `site_missing`, offset 0 (`gomad.go:801`).
  - `SelectedIdentity` for a runnable decision is a goroutine lineage hash: parent identity,
    child ordinal, and the `go` statement's site (`gomad.go:745-776`).
- Divergence already reports ordinal, reason, expected and observed decision
  (`g3/choice/tape.go:382-388`).
- The retained binary is built without `-s -w`, so pclntab and DWARF are present [I from the
  build arguments at `target.go:710`].

### (b) Gaps

- The inspect fingerprint is `sha256(target, kind, offset)`; the offset is not printed, so no
  file:line (`g3/choice/trace.go:325-335`).
- No goroutine-creation record. Kinds are only runnable, select_poll, select_result
  (`choicewire.json:25-29`). The lineage hash is one-way, so a per-goroutine timeline cannot
  name "which goroutine" from the trace alone.
- No link between choice ordinals and I/O transcript ordinals, World transitions, virtual
  time, or test output lines. Each stream has its own ordinal space.
- Debugger stop at an ordinal: no Runner mode launches the target under a debugger or waits
  for attach; the wall watchdog would kill a paused target.

### (c) Smallest experiments

1. **Symbolized sites, no toolchain change.** In `projectChoices`, open the artifact's
   `target` payload with `debug/gosym` (or shell out to the pinned `go tool addr2line`) and
   print `function file:line` for each select site's `SiteOffset`. Evidence: a failure
   artifact from a Temporal suite lists the hot `select` statements by name.
2. **Goroutine names by replay.** Replay the artifact with a diagnostic environment flag that
   makes the runtime print `identity → go-statement offset, parent identity` at each
   `gomadChoiceAssignGoroutineIdentity`. This changes the overlay and therefore the toolchain
   build key, so it must be a separate diagnostic toolchain or a new record kind with a wire
   version bump.
3. **Delve feasibility probe.** Launch a retained target under `dlv exec` with a conditional
   breakpoint on `runtime.gomadChoiceDecision` when `runtime.gomadChoiceDecisionRecords == N`
   (`gomad.go:43, 498`).

### Delve specifics

- ASLR re-exec is not a blocker [I]: `gomadDisableASLR` returns immediately when the image
  slide is already zero (`gomad_aslr_darwin.go:90-93`), which is how Delve and lldb launch.
- Host pauses do not advance virtual time [V by design]: time advances only at runtime
  quiescence (`g3/ARCHITECTURE.md:229-239`).
- Real obstacles [I]:
  - A Runner-managed target needs its inherited descriptors (bootstrap frame, transcript and
    choice mappings, mount pipes). A plain `GOMADSEED=n ./binary` launch uses host I/O and
    will not follow the artifact's schedule. A `gomad replay --debug` mode has to place the
    debugger between supervisor and target, or stop the target at start for attach, and
    disable the wall watchdog.
  - Delve's Go-version check against a patched 1.27.1 and its handling of the extra `g`
    fields are untested.
  - Run-queue decisions happen on the system stack (`gomad.go:777-781`), so a stop there
    shows scheduler frames; the useful stop is the first user instruction after the switch.
  - `-gcflags=all=-N -l` changes target identity, so optimized-binary debugging is the only
    mode that replays an existing artifact.

### (d) Size and risk

- Symbolized inspect: **S**, low risk.
- Timeline with goroutine names and cross-stream ordinals: **M**; touches the choice wire
  version and toolchain identity.
- Debugger stop: **M** for a working prototype on one platform, **L** to qualify; risk is that
  ptrace stops perturb host-timed runtime paths the README already lists as fragile
  (`g3/README.md:773-805`).

### (e) Stale or contradicted

- CMP: "connecting logical actors, choice ordinals, available source sites". Source sites are
  more available than CMP implies (offset + retained binary). Logical actors are less
  available (hash only).
- README line 137 says inspect reports "target-specific site fingerprints"; that is exact, and
  it is the reason the output is not actionable.

---

## 5. Minimizer

### (a) What exists

- Pure controller with sealed, hash-identified JSON state `gomad3.minimizer-state/v1`
  (`g3/runner/internal/minimizer/minimizer.go:13, 49-60`), `Next`/`Commit` transitions
  (`:84-119`), and three proposal families generated in a fixed order (`:167-185`):
  - `schedule_suffix`: drop a trailing run of runtime overrides;
  - `schedule_range`: drop any contiguous range of non-fault overrides;
  - `fault_entries`: drop any contiguous range of fault overrides.
  Ranges are enumerated longest first (`:201-221`).
- Driver: accepts only `ArtifactTargetFailure` with exact replay, a simulation profile, and a
  retained choice tape (`g3/runner/minimize_operation.go:155-167`). Each attempt runs in a
  fresh process, must keep failure signature and outcome, and must replay exactly
  (`:269-367`). The result records parent, reductions, budget, and predicate (`:443-461`).

### (b) Gaps

- **Resume.** The state is already serializable and self-validating, but `Minimize` holds it
  in memory and works in `os.MkdirTemp` that `close()` deletes
  (`minimize_operation.go:85-104, 189-198, 487-498`). Missing: write `state` after each
  `Commit`, keep the last accepted trial artifact, and reopen.
- **Only forced decisions shrink.** A candidate is a list of exploration overrides. A failure
  from a plain seed campaign has no overrides and no simulation profile, so it is rejected.
  That covers every Temporal functional-test failure.
- **No typed input shrinking.** There is no input to shrink (idea 1).
- **Search cost.** Contiguous-range enumeration is O(n²) proposals per accepted step, and
  `proposals` is recomputed in every `Next` and `seal`. Adequate for tens of overrides.
- **No shrinking of spec-level data** (node count, links, fault plan entries that are not
  exploration overrides, scenario steps).

### (c) Smallest experiment

Persist-and-resume: write `state` (already canonical JSON) and the accepted artifact path
under `--output` after each `Commit`; add `minimize --resume DIR`. Kill the process mid-run on
the existing fixture and check that attempts are not repeated (`Evaluated` already lists
them).

A second, more valuable experiment: **seed-failure tape shrinking**. Take a choice-traced
failure, project its tape, and try "keep the first k decisions forced, let the suffix run from
the seed" for a binary search on k. Prefix replay exists as an internal primitive
(`g3/choice/tape.go:177-236`). This would give Temporal functional failures a first reducer.

### (d) Size and risk

- Resume: **S**.
- Seed-failure prefix shrinking: **M**; risk is that a freed suffix rarely reproduces the same
  signature in 60k-decision executions.
- Typed scenario shrinking: blocked on idea 1; **M** after it.

### (e) Stale or contradicted

None. README, CLI, and NEXT BUG-5 match the code. CMP's "extend the current combined-simulation
reducer" omits that the reducer cannot open the failures Temporal engineers will actually
have.

---

## 6. Checkpoints and snapshots

### (a) What exists

- World: `Snapshot` and validated `Restore` (`g3/world/snapshot.go:40, 69`).
- Simulation network and volume: canonical snapshots are exported as evidence and identity
  (`sim/types.go:247-313`, `sim/record.go:1029-1076`). Crash-state enumeration is bounded and
  resumable through `VolumeCrashFrontier` (`sim/types.go:315-352`).
- Campaign and exploration progress: hash-linked round segments with `ReplaySegment`
  (`g3/runner/internal/exploration/simulation/frontier.go:271, 352`;
  `.../choice/engine.go:202, 300`).
- Prepared-target cache keyed by full build identity (`g3/README.md:348-359`).

### (b) Gaps

- Simulation snapshots are export-only. `VolumeSpec` is `{ID, CapacityBytes}`
  (`sim/spec.go:120-123`) and `Spec` has no initial network or volume state, so an execution
  cannot start from a model snapshot.
- No process snapshot of any kind, as CMP says.
- No recorded phase split inside an execution (init versus test body).

### Timing data on disk

`wall_elapsed_nanos` is execution-only wall time (`g3/runner/runner.go:879`,
`g3/qualification/qualification.go:62`). Values below are from
`g3/.toolchain/{temporal,smoke}-qualification/qualifications/v1/*.json` (local darwin/arm64,
traced) and agree with the table in
`.flow/tasks/fn-100-gomad-f6-a-package-level-functional.2.md`.

| Workload | Wall per execution | Virtual time | Choice records | Peak goroutines |
| --- | --- | --- | --- | --- |
| `./tests/gomadfunctional TestFrontendSystemInfo` (boot + 1 RPC) | 1.7 s | 0 s | 4,562 | 587 |
| `TestUserTimersTestSuite` | 1.5–2.4 s | 4.0 s | 8,123 | 663 |
| `TestWorkflowTimerTestSuite` | 1.5–1.6 s | 8.0 s | 12,865 | 704 |
| `TestCronTestSuite` | 1.5–2.2 s | 15.0 s | 14,615 | 709 |
| `TestCancelWorkflowSuite` | 1.7 s | 0 s | 31,103 | 1,356 |
| `TestActivityTestSuite` | 1.7–3.5 s | 13.6 s | 42,165 | 1,451 |
| `TestSignalWorkflowTestSuiteChasm` | 2.2–2.3 s | 57.0 s | 86,243 | 1,945 |
| `TestWorkflowUpdateSuite` | 2.9–3.2 s | 22.4 s | 207,900 | 4,584 |
| Tier-2 package tests | 0.3–2.1 s | 0–8 s | 6–3,205 | 2–1,028 |

Readings:

- Bootstrap cost [I]: the boot-only probe costs as much wall time as several whole suites, so
  roughly 55% to 100% of a functional execution is prefix.
- Native comparison: no head-to-head table is recorded. Data points: virtual time compresses
  57 s of logical waiting into 2.2 s; D18 records one native leaf at about 3.05 s
  (`docs/research/gomad/GOMAD_D18_WORKER_CANCEL_DELIVERY.md:177`); native `./tests` leaf runs
  in retained logs take 11 to 54 s including their cluster setup
  (`g3/.toolchain/d17-correction-native-suite.log`).
- Preparation dominates the first run: a whole `qualify` command took 69 to 150 s for
  executions of 1.5 to 4 s (fn-100 task 2 summary).

### (c) Smallest experiment

Prefix-cost study with existing commands: `gomad qualify --choices` on `TestFrontendSystemInfo`
and on one suite, then diff wall time and choice-record counts. Add one number the reports
lack: the choice ordinal at which the first test body starts (print a marker line and find
its position, or use a one-shot World/assert feature from idea 3).

### (d) Size and risk

- Study: **S**.
- Model snapshot-in for the simulation (initial volume contents, initial link state): **M**.
- Live process snapshot: **L+**, platform-specific, and in tension with the ASLR and
  GC-determinism work the README documents.

### (e) Stale or contradicted

CMP: "Evaluate OS/VM snapshots only after profiling proves prefix execution dominates." The
retained reports already suggest it does for `testcore` suites. CMP's table row "Explicit model
snapshot ... Existing snapshots can support model-owned search" is half true: World can
restore; the simulation network and volume models cannot.

---

## 7. Parallelism and distributed rounds

### (a) What exists

- `plan`: unguided seed campaign with `--on-failure=all` only
  (`g3/runner/portable_plan.go:77`). Bundle = verified target + captured mount trees.
- `execute-shard`: ordinal-modulo ownership (`g3/runner/campaign_shard.go:23-25`), full bundle
  revalidation (`g3/runner/campaign_shard_execution.go:33-95`).
- `merge`: same plan identity, no overlap, no gaps unless `--partial`
  (`g3/runner/campaign_merge.go:42-62`).
- `qualify-set --shard` / `merge-set` partition workloads instead of seeds
  (`tools/gomad3integration/README.md:149-175`).
- Local rounds: both exploration engines are pure controllers.
  `NextRound()` yields candidates, `CommitRound(state, round, results)` yields the next state
  and a segment, `ReplaySegment` rebuilds state from a segment
  (`.../exploration/simulation/frontier.go:215, 271, 352`; `.../choice/engine.go:171, 202, 300`).
  Completions are committed in candidate order
  (`g3/runner/simulation_exploration_campaign.go:268-300`).
- `g3/runner/coordinator.go` is an isolated Runner subprocess for a local campaign
  (`coordinatorConfig` at `:22-62`). It is not a distributed round coordinator.

### (b) What a round coordinator needs

1. A portable round bundle: plan identity + `State` hash + the `Round` (candidates already
   have content identities).
2. `execute-round-shard`: run candidates `i % count == index`, publish `Result` records plus
   artifacts keyed by candidate hash.
3. `commit-round`: validate coverage of every candidate, call the existing `CommitRound`,
   publish the segment, emit the next round bundle.
4. Resume and duplicate-ownership rules equal to PROD-4's wording
   (`.plans/GOMAD_NEXT.md:116-121`).

The control logic exists; the missing parts are the three file formats and CLI verbs. For
choice exploration, candidates carry whole tape prefixes (`PrefixBytes`,
`.../choice/engine.go:59-66`), 96 bytes per record, so deep rounds make large bundles.

### (c) Smallest experiment

Before building it, measure whether rounds are worth distributing: run local
`--strategy=choice-exploration` on one tier-2 workload at `--parallel` 1, 4, 8 and record
executions per second and round sizes. Finding 4 says the strategy is not yet useful on
functional suites, so distributing it has no consumer.

### (d) Size and risk

- Round coordinator: **M**. Risk: building distribution for a search policy that NEXT.md
  already reports as showing "no advantage over seed sampling" on its fixture
  (`.plans/GOMAD_NEXT.md:31-35`).
- Deterministic guided epochs: blocked on idea 1.

### (e) Stale or contradicted

None. CMP's statement that shard/merge covers unguided seed campaigns only matches the code.

---

## 8. Oracles

### (a) What exists

- Four oracle helpers and a validated history type (section 3). `HistoryOperation` has
  caller-supplied `Invocation`/`Completion` integers, checked only for `0 < inv ≤ comp`
  (`sim/oracle.go:19-28, 132-134`). They are not tied to logical time.
- No linearizability or serializability checker anywhere in the Gomad trees (grep); the only
  hit is a World unit-test name.
- `sim/temporal_scenario_toolchain_test.go:17-180`: a server boot that reads six bytes and
  records duplicates in a `collection.SyncMap`, a client boot that dials `10.0.0.1:7233`, a
  two-action fault plan (disconnect and reconnect the ack direction), and
  `NoDuplicateOrLost` expected to fail, then exact replay. The only Temporal import is
  `common/collection` (`:14`).

### (b) Why the simulation track "cannot host testcore"

The milestone states it without a reason (`MILESTONES.md:367-368`); no other
document explains it. Constraints found in code:

- [V] A simulation target's seed must equal a compile-time constant (fixture comment,
  `main.go:22-25`), so a harness-run suite cannot take campaign seeds.
- [V] In-process nodes share package globals and crashed goroutines stay computationally
  live; hard isolation needs the process backend (`g3/ARCHITECTURE.md:66-73, 109-113`).
- [V] The process backend boots each node from a registered `BootFunc` in a fresh process
  (`sim/registry.go:14`, `sim/process_node.go:80-123`). Volumes are per-node; there is no shared
  volume or external-store adapter (listed as a candidate at `.plans/GOMAD_NEXT.md:176`).
- [I] `testcore` builds every service in one process over one in-memory SQLite
  (`tests/gomadfunctional/frontend_test.go:12`). Splitting it across simulated nodes needs
  either a persistence node that other nodes reach over the virtual network or a shared
  store model. Neither exists.
- [V] The `gomad` build tag drops ringpop membership
  (`tools/gomad3integration/README.md:189-192`), so multi-node membership has no
  implementation under Gomad today.
- [V] Harness linknames are admitted only for the exact package in the main module
  (`g3/target/capability.go:775-783`), and the compiler intercepts that package by path and
  declaration hash (`.../gomadintercept/simulation_specs.go:8-`). That does not block
  `testcore`, which is in the main module, but it blocks a downstream copy.

### (c) Smallest experiment

Skip the simulation harness for the first oracle. Add one independent invariant to an existing
qualified functional suite and run it across seeds with `explore`. The retained candidate is
the update-admission ordering bug Gomad already found (`compareAdmission`,
`service/history/workflow/update/registry.go:512-515`, fixed in `97d9925ac`): reintroduce the
bug on a scratch branch and measure seeds-to-detection for `TestWorkflowUpdateSuite`. That
gives the first "known bug, corrected counterpart" row CMP's Confidence stage asks for.

### (d) Size and risk

- Known-bug benchmark: **S**.
- Bounded linearizability checker over `HistoryOperation`: **M**; needs real invocation and
  completion stamps (virtual time plus a sequence), and a sequential model per object.
- Multi-node Temporal in the simulation: **L+**; the persistence and membership seams are the
  cost, not the harness.

### (e) Stale or contradicted

CMP proposes "legal workflow/update transitions and preservation of acknowledged durable
effects" as candidate properties without noting that no Temporal service runs in the
simulation, so fault and recovery oracles have no Temporal subject yet.

---

## 9. Consumers

### (a) What exists

- `tests/gomadfunctional/frontend_test.go`: one 16-line probe (cluster up, one RPC).
- `tools/gomad3integration`: `temporal.json` (28 workloads: 15 package tests, 12 `./tests`
  suites, the probe), `smoke.json` (four suites), generated `tests.json` (one workload per
  top-level `./tests` test, 147 targets, untraced by default), and an outside-in test of the
  Make targets.
- Make targets (`Makefile:167-229`): toolchain and runner builds, two single-seed
  compatibility wrappers, and four qualification targets.
- CI: `.github/workflows/gomad3.yml` (host tooling, linux core gates, macOS upgrade dossier,
  representative qualification on schedule or dispatch) and `gomad3-smoke.yml` (smoke
  qualification on linux and macOS for PRs touching relevant paths).
- Server changes made for Gomad: the `gomad` build tag seams and five adapters
  (`g3/README.md:509-531`).

### What a Temporal engineer gets today

- A CLI to run one functional suite under N scheduling seeds with virtual time, keep a failing
  execution as an artifact, and replay it exactly on the same platform
  (`g3/TUTORIAL.md:650-662`).
- Suites run in 1.5 to 4 s each after a one-time build of about one to two minutes.
- One real server bug found during qualification (update-admission ordering). Five test or
  clock issues diagnosed (D16 to D20), three resolved by test changes.
- On linux, about one tier-3 seed-run in 26 still diverges (D12), so a failing artifact there
  may not reproduce.

### (b) Gaps

- No bug-hunting job or target; all automation asserts determinism of passing runs.
- No failure triage path for a Temporal engineer: `minimize` rejects their artifacts and
  `inspect --choices` shows hashes.
- No in-repo scenario with Temporal services under faults.

### (c) Smallest experiment

A nightly `gomad explore --count 200 --on-failure=budget` over the four smoke suites, on
darwin where replay is qualified. Publish failures-per-compute-hour. This is the baseline
every adoption gate in CMP compares against, and it does not exist.

### (d) Size and risk

**S**. Risk: failures will mostly be virtual-time test assumptions (the D16 to D20 class), so
the first weeks produce test fixes, not server bugs. That is still the measurement CMP needs.

### (e) Stale or contradicted

CMP's opening goal ("find more distinct Go concurrency bugs per unit of compute") has no
current measurement; "Establish a qualified benchmark set before judging a new policy" is
unstarted.

---

## New opportunities not in GOMAD_CMP.md

1. **Stop spending guided budget on known seeds.** Exclude corpus seeds from selection or run
   them only as an explicit regression mode (finding 2). S.
2. **Deduplicate the target binary across artifacts.** One content-addressed target per
   campaign or corpus would shrink the 11 GiB qualification footprint
   (`g3/README.md:333-334`) and make the corpus usable for 155 MB targets (finding 3). M,
   touches artifact schema.
3. **Exploration start offset.** Let choice exploration expand only decisions at ordinal ≥ K,
   with K found from a marker (first test body). Without it the strategy cannot reach test
   logic in `testcore` suites (finding 4). S to M.
4. **Symbolize from the retained binary** (section 4). S.
5. **`FuzzMatcherData` as the bridge pilot** (section 2): an existing tape-driven operation
   generator in a package that already qualifies.
6. **Reuse Temporal's own seams.** `common/testing/testhooks` (tag `test_dep`) is an existing
   injection point for assertion emits or targeted yields; `common/testing/event_generator.go`
   is an existing model-based history generator; `faultinjectiontest` and `grpcfaultstest`
   exist for fault inputs. None is referenced by Gomad.
7. **Feed simulation evidence to the Guide.** Observation IDs/kinds, oracle names, and fault
   realizations are already canonical in `ClusterRecord` and unused as features. S once
   guidance and simulation strategies can be combined.
8. **Corpus identity hole.** Environment and clock-tick policy are not in the corpus identity
   (`corpus/model.go:37-47`). Decide whether that is a bug to close or the seam for
   workload-seed campaigns.
9. **Expose the seed or an input blob to the target** through a tiny reviewed API. This single
   primitive unblocks ideas 1, 3, 5, and the simulation seed constant.

## Suggested order by evidence per day of work

| Step | Idea | Size | Produces |
| --- | --- | --- | --- |
| Nightly `explore` on smoke suites | 9 | S | failures per compute-hour baseline |
| Reintroduce the update-admission bug, measure seeds-to-detection | 8 | S | first known-bug benchmark row |
| Symbolized `inspect --choices` | 4 | S | actionable failure output |
| `WORKLOAD_SEED` via `--env` on a tape-driven test | 1 | S | does input variation add failures |
| Offline `FuzzMatcherData` corpus import | 2 | S | fuzz inputs × schedule seeds throughput |
| Boot-prefix study (probe versus suite) | 6 | S | prefix share with ordinals |
| Minimizer resume | 5 | S | recoverable reduction |
| Candidate channel + seed/input API | 1, 3, 5 | M–L | the shared prerequisite |
