# fn-222.1 Rust one-box gates: checkpoint 2026-10-22

Status: **pending**. Gate 1 (unchanged source preparation) has no Rust entry
point, so no measured gate was run. The task requires reporting a gate-1 failure
before any measured gate. The QUALIFICATION.md one-box row is unchanged.

## Candidate

| Identity | Value |
| --- | --- |
| Integration commit | `gomad` 7aacd7cb21 (includes fn-232.10 partial merge 1c804fd576) |
| Rust release binary (`cargo +1.99.1 build --release --locked`, no features) | sha256 `ebb8eedf1fa908a53a0b009cd70dbf37b9755548558860a04a59411e6607b678` |
| Linked helper implementation (`helper_sha256`) | `54773ee79492c60e23585b9dfe3f71dd4e9e3aada19d06a5b7af045768b3c2a3` (per `.flow/artifacts/fn232.8-identity-handoff-2026-10-22.txt`) |
| Compiler | stock Go 1.27.1, `.build-cache/fn232-stock-go1.27.1/bin/go`, sha256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64` |
| Host | linux/arm64 |

Host, runtime-overlay, module, engine and model identities are bound only by a
Rust `qualify` freeze (`gomad.rust-qualification-manifest/v1`). No freeze ran,
so none is recorded for this candidate.

## Gate mapping on the Rust binary

The gates in `QUALIFICATION.md#one-box-cell-gates` are Go test-API commands
(`go test ./qualification`, `./backend`). The Rust binary has no test API. Its
public verbs are `explore`, `resume`, `execute-shard`, `merge`, `minimize`,
`qualify`, `report`, `replay` and `inspect`.

| Gate | Rust equivalent | Numeric exit |
| --- | --- | --- |
| `unchanged-source-preparation` (prepare-only, retained module receipt) | none: `qualify --spec` always runs the measured repetitions after freeze; no prepare-only mode or spec field exists | not run (no command) |
| `measured-pass-replay-success` | `gomad --installation INST qualify --spec SPEC` with `Manifest.Cell=success`, then `gomad report --result R` | not run (gate 1 first) |
| `measured-pass-replay-canary` | same with `Cell=canary` and `-test.run=[` | not run (gate 1 first) |
| `storage-transport-inspection` | Result/receipt validation is `report` (fn-232.10); the inspection cell is the Go test `TestTemporalInspectionRetainedFailure/onebox` in `./backend`, with no Rust command | not run (no command) |
| R2 support list (`gomad.wasm-support-list/v1`) | none: `BuildSupportList` is Go and accepts only Go-lineage Results | not produced |

Probes on the candidate binary
(`/home/agent/wt/evidence/fn222-onebox/probe-verbs.txt`):

| Command | Exit | stderr |
| --- | --- | --- |
| `gomad install` | 2 | `gomad: unsupported command` |
| `gomad support-list` | 2 | `gomad: unsupported command` |
| `gomad compare-support` | 2 | `gomad: unsupported command` |
| `gomad qualify-set` | 2 | `gomad: unsupported command` |
| `gomad qualify --spec /nonexistent.json` | 2 | `gomad: open /nonexistent.json: no such file or directory` |

## Further gaps for the measured gates

- **No Rust installation verb.** The fn-232.10 lanes derive Rust installations
  from a Go `install` manifest through test tooling
  (`testdata/parity/qualify-check.py --prepare-only`, `fixture-replay-input`).
  That tooling is hard-coded to the `./tests/gomadfunctional` namespace
  (`frontend-system-info`). The mandatory smoke cell needs the `./tests`
  namespace (working directory `/workspace/tests`).
- **Frontend-system-info on the Rust binary.** The fn-232.10 lanes already ran
  it as a final success cell with the owner budgets on the fn232-qualify
  candidate (evidence `/home/agent/wt/evidence/fn232-qualify/d/v17/`, exit 0).
  That candidate is not this one, and no canary was run, so it supplies no row.

## Decisions needed (not taken here)

Each needs host code or test-tooling changes, which this task's stop rule
forbids. They are recorded rather than started:

1. A Rust prepare-only freeze (gate 1) that retains the module receipt.
2. A Rust installation path for the `./tests` smoke namespace.
3. A Rust storage/transport inspection cell equivalent to
   `TestTemporalInspectionRetainedFailure/onebox`.
4. A Rust port of `BuildSupportList`/`ValidateSupportList` over Rust Results
   and receipts. The task's Touches already name `rust/gomad/src/qualification/**`.

No capacity limit was changed.

## Current Rust run readiness (source-only continuation)

This section supersedes the historical tooling-gap inventory above, not its old
candidate observations. fn222.1 is running under the conductor's current ordering;
R1/R2 and the QUALIFICATION onebox row remain pending. No current-candidate gate
has been attempted. fn221's original frozen Go journal remains 4/344 in its exact
original lineage; native v2 Sweep is not a replacement for it.

Current source has public Rust `install` for both namespaces and seed `plan`
with a retained `gomad.rust-preparation-receipt/v1`, module/provenance and zero
guest executions. `rust/workload_tools/temporal-support.py` is the existing runner:
`--prepare-only` now performs smoke install+preparation only, then returns the
actual preparation response without measured, inspection, frontend, support-list
or PASS claims. Failure still stops before measured work. Full mode retains the
original order smoke preparation/success/canary, shared onebox inspection, then
frontend preparation/success/canary, each native measured operation in a fresh
producer process. Exact smoke/frontend args remain verbose/count1/timeout72h,
registered selection, SQL persistence and sqlite driver in their original order.
Preparation/frontend/inspection commands explicitly omit mandatory-workload
intent; only smoke success/canary subprocesses receive
GOMAD_WASM_QUALIFY_WORKLOADS=1. Every command witness observes that eighth field,
including explicit null when absent. The original exact selected verbose PASS
must appear once per execution; duplicates, bare wrapper PASS, changed leaf,
SKIP and no-tests refuse. No budget/profile/selection changed.

| Gate | Current actual endpoint | Current status |
| --- | --- | --- |
| unchanged-source-preparation | temporal-support.py --prepare-only invokes release install+seed plan; receipt/digest and guest_executions=0 retained | Code ready; execution UNEXECUTED pending candidate/resource grant and private observation seal integration |
| measured-pass-replay-success | existing producer.py --workload smoke/frontend --mode success, native qualify+installation-free report | UNEXECUTED; requires matching prepared module, Repeat2 own exact matched replays and original budget |
| measured-pass-replay-canary | same mode canary; native target-failure exit1, wrapper exit0 only after authentic expected outcome/replay/bug/budget assertions | UNEXECUTED; no CLI exit rewritten |
| storage-transport-inspection | temporal-inspection.py --owner onebox; unchanged prototype_temporal source, original typed scenario/1GiB effective incarnation trace, detached exact replay and stopped dumps/probes | Source endpoint exists; UNEXECUTED on current candidate; shared only for matching full candidate |
| first accepted candidate-bound support JSON+Markdown | receiving owner's private support.py build/validate/publish/open_list integration (no new publicCLI) | Still UNBOUND/WIP: must authenticate both rows' actual pairs/preparation/namedPASS and shared inspection; runner support-receipt alone is insufficient |

Legacy qualify-check.py --prepare-only builds a Go installation and depends on
fixture controls; qualify-lane.py's onebox is hard-coded to frontend with a
noncanonical old argument order. Those remain legacy parity lanes; they are not
this task's mandatory smoke producer, first-gate endpoint or support-row proof.
No Go qualification producer is requested for these Rust runs.

### Exact held commands and input bindings

The conductor must bind every FN222 variable below to its frozen current release
candidate/input inventory before dispatch. FN222_RELEASE is the ordinary default
release executable, not a fixture/diagnostic binary; FN222_SOURCE_MANIFEST is
that candidate's exact source manifest; FN222_COMPILER is pinned stock Go1.27.1;
FN222_RUNTIME is its source-bound runtime overlay. FN222_CAPTURE is an existing
complete CapturedInputs schema/sqlite descriptor+payload configuration whose
bytes producer.captured revalidates against the selected checkout. It may reuse
schema data after full revalidation, never an old installation/Result identity.
The release/source-manifest/capture bindings are currently UNBOUND; no SHA or
candidate identity is invented here. Preparation and full roots must be fresh,
distinct private directories, and all original evidence/scratch >=12GiB native
admission and native filesystem checks must pass before either command.

```sh
cd "$FN222_CHECKOUT"
env -u GOMAD_WASM_QUALIFY_WORKLOADS "$FN222_PYTHON" \
  tools/gomad/rust/workload_tools/temporal-support.py \
  --binary "$FN222_RELEASE" --repo "$FN222_CHECKOUT" \
  --compiler "$FN222_COMPILER" --runtime-root "$FN222_RUNTIME" \
  --cache "$FN222_CACHE" --captured-config "$FN222_CAPTURE" \
  --source-manifest "$FN222_SOURCE_MANIFEST" --root "$FN222_PREP_ROOT" \
  --prepare-only
```

Only after that actual zero-execution preparation succeeds and its numeric exit,
module/receipt/source/compiler/host/helper/runtime/engine/model identities and raw
logs are retained, the same frozen candidate may receive the full command:

```sh
cd "$FN222_CHECKOUT"
env -u GOMAD_WASM_QUALIFY_WORKLOADS "$FN222_PYTHON" \
  tools/gomad/rust/workload_tools/temporal-support.py \
  --binary "$FN222_RELEASE" --repo "$FN222_CHECKOUT" \
  --compiler "$FN222_COMPILER" --runtime-root "$FN222_RUNTIME" \
  --cache "$FN222_CACHE" --captured-config "$FN222_CAPTURE" \
  --source-manifest "$FN222_SOURCE_MANIFEST" --root "$FN222_FULL_ROOT"
```

The full run performs its own fresh preparation and measured operators. Initial
prepare-only is early feedback, not adopted measured cost or a support publication.
Final accepting support evidence must use its own complete frozen candidate cycle.
Private native observation capture is being added at the existing producer/driver
witness boundary in coordination with the qualification owner: separate campaign
sidecars, actual owner PID, explicit campaign set and immediate outer receipt/raw
hash seals. Volatile observations stay outside canonical Record/Store/plan.
These exact private capture options/reader binding must join before final runs;
no nonexistent support-list CLI command or actual proof is assumed here.

### Resource hold and bounded verification

Read-only native filesystem observation during this continuation showed225260KiB
available (df exit0), below the unchanged12GiB requirement. Parent's later hold
also reports217MiB and an exclusively leased host build cache with88GiB; host
build capacity does not waive native evidence/scratch admission. No actual guest,
Go/build/prepare/qualify gate ran, so there is no numeric gate exit to invent.
First release build is root-owned after reviewed cache/lifecycle/native auxiliary
source joins and an explicit resource grant. No deletion or cache cleanup was
performed; current compiled binaries, evidence and worktrees are retained.

Causal bounded portable regressions against the old runner reproduce9wrong
intent subcases, accepted duplicate/bare PASS witnesses, and missing
--prepare-only option exit2. Current5runner tests pass, including actual bounded
Python child intent observation and mocked adapter dispatch/early-failure guards;
these are explicitly synthetic unit fixtures, not gate evidence. Integrated
workload suite22tests and driver113tests pass; source AST/diff checks pass. Narrow
coordinated receiving intent dependency c6e2ac48ff was integrated without its
unrelated parent WIP. Support-list raw integration and native auxiliary capture
remain separately owned/in progress, and this task continues toward actual runs
when the conductor supplies the frozen candidate and resource grant.
