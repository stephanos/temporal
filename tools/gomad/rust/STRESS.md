# Fresh same-seed stress

`stress` drives the Rust binary and its linked helper. It is an explicitly admitted
measurement producer, outside test and qualification targets. Task fn-188.5 stays
open until the integrated, frozen candidate has a retained complete matrix; the
implementation and unit checks do not establish that pass.

Build one instrumented binary with `--features helper-stress-jitter`. Use that same
binary and its actual helper/host/engine identities for the unperturbed reference
and every axis. The default release excludes the jitter implementation; `qualify`
refuses any binary with a compiled nonordinary helper role before setup, including
stress, determinism fixtures and diagnostic profiling.
The normal release remains the qualification binary. Installation and preparation
receipts from another compiled role cannot be rewritten to make identities match.

```sh
GOMAD_WASM_STRESS=1 /absolute/frozen/stress-gomad stress --spec /absolute/frozen/stress-spec.json
```

The strict spec has schema `gomad.rust-stress-spec/v1` and fields:

- `root`: new absolute retained directory, normally beneath
  `.flow/artifacts/wasm-determinism-stress`.
- `host_description`: explicit host identity and quiet-host admission. The root
  scheduler serializes this run behind builds and other guests.
- `source_manifest`, `source_sha256`: an absolute source manifest and its SHA256.
  The manifest uses the qualification owner's SHA256/two-spaces/absolute-path
  lines. Its current files are authenticated before preparation and between cells.
- `repetitions`: K, 2–256; `parallel`: concurrent cohort size, 2–256;
  `cpu_workers`: real busy threads on the CPU axis, 1–256.
- `workloads`: exactly four objects with `name`, absolute `installation`,
  `arguments` (existing `explore` argument vector), and optional absolute
  `scenario`. Installations are actual Rust manifests from the stress binary.

Every argument vector enables `--choices --diagnostics --parallel 1`, declares the
original timeout/output/choice bounds (overall timeout equals execution timeout
plus one hour), and supplies the original guest environment
and source root. `scenario` is supplied separately so the internal stress owner
can enable diagnostics without widening public `explore --scenario` policy.
Preparation freezes each source once through the existing Rust preparation owner,
retains module and provenance, and re-admits that exact prepared `exec` in each
fresh process. Every execution creates fresh Wasmtime engines, stores and model
state. The existing artifact publisher shares only authenticated immutable module
bytes through its target pool; every run retains its actual observation and PID.

| Workload | Seeds | Unchanged source/selection | Guest memory / fuel / wall | Choice / output / WASI transcript |
| --- | --- | --- | --- | --- |
| `runtime` | `index*1000003`, index 0–31 | nested Gomad module, `go-run ./internal/backends/wasm/testdata/runtime_probes`; `ONLY=captured`, `EXTRA_ENTROPY=no_` | 256MiB / 4000000000 / 1m | 1MiB / 1MiB / 64MiB |
| `two-cluster` | 7, 19, 42 | unchanged staged `temporal_storage/recovery_test.go` plus `temporal_broker/replication_test.go`; `^TestIndependentClusterReplicationRecovery$` | 4GiB / 2^50 / 15m per incarnation | 32MiB / 8MiB / 128MiB |
| `workflow` | 0, 1000003, 2000006 | full `./tests`, `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` | 4GiB / 2^50 / 15m | 32MiB / 256MiB / 1GiB |
| `activity` | 0, 1000003, 2000006 | full `./tests`, `^TestActivityTestSuite$` | 4GiB / 2^50 / 15m | 32MiB / 256MiB / 1GiB |

Temporal argv is exactly `-test.v -test.count=1 -test.timeout=72h
-test.run=SELECTION -persistenceType=sql -persistenceDriver=sqlite`; the cluster
uses `-test.timeout=1h`. Source preparation uses the original slim guest tags
`disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep`, original captured
SQLite schema and writable namespace, pinned compiler and cooperative overlay.
Calls, pending events, namespace capacities, assertions and model limits retain
the original stress recipes. The spec must provide those installations; the
producer rejects changed guest and WASI bounds, forced modes and assertion bypass.
Every workload must have a successful canonical outcome before its custom
oracles can pass. The original runtime exit-17 canary is the sole exception and
requires its exact witness without a retained assertion failure.

The original two-cluster scenario starts active/standby incarnation 1, crashes
both after their grouped observation acknowledgments, restarts incarnation 2,
and waits for both. It preserves bidirectional links and the partition/heal at
1ns/30s. `stress::cluster::bind` updates the existing typed `scenario::Bound`
request and retained replay configuration together: node/phase/peer environment,
node-derived entropy and grouped crash marker. It never adds public scenario
policy. Original and reopened observations must preserve independent cluster IDs,
converged history bytes, sequential event IDs and event types. All raw incarnation,
World, network, volume, namespace, output and inspection evidence remains retained.

Each workload/seed gets one unperturbed reference. CPU and layout cells run K
fresh subprocesses; concurrent cells run K full cohorts of `parallel` subprocesses
behind a launch barrier. Layout varies the real executable hardlink pathname,
process cwd, temporary directory pathname and host environment allocation. These
are outside the recorded guest environment. The former external-helper cache
pathname is inapplicable: the linked engine compiles in process and has no such
cache owner. This fact is recorded alongside the actual layout perturbations.
Cluster handoff cells run K fresh subprocesses with runtime jitter enabled.
Standalone handoff cells are explicitly skipped as inapplicable.

The compiled stress role records event order in every cell. Only the handoff axis
sets `GOMAD_WASM_STRESS_JITTER=1`; baseline and other axes remove it. Actual baton
grants and modeled peer replies delay by a fresh host-random 0–1000 microseconds.
The producer retains delay counts and requested total delay outside guest evidence
and refuses to credit a handoff axis that made no actual delays.

`report.json` uses schema `gomad.rust-stress/v1`. Canonical JSON retains the complete
41-by-4 matrix, K/cohort/completed counts, pass/divergence/skipped, every comparison,
source/helper/host/module/toolchain/configuration/model identities, raw stream
hashes, PIDs, host observations and exact rerun argv. Prefix checkpoints retain
completed cells if a later infrastructure failure interrupts publication. A short
cell is skipped, an identity or forced-mode change is rejected, and any divergence
or undiagnosable comparison fails the run. All such findings name fn-188.6.

Comparisons reopen and authenticate both Rust artifacts, independently reject
forced retained models, require per-incarnation diagnostics, and compare every
canonical record stream plus supplemental helper event order. The first differing
event retains kind, owner/incarnation, local ordinal, shared coordinate, virtual
time and both values using `gomad.wasm-first-divergence/v1`. A replay stop is emitted
only after the common preceding event marker matches a retained inspection
boundary and the linked helper authenticates its actual stream/World prefix.
Missing diagnostics or prefix evidence is explicitly `undiagnosable`. Exact cell
reruns invoke a supervisor with the retained private request and a new output
root. The bound CPU worker count, full concurrent cohort, launch barrier and
layout repetition are recreated. Direct child argv stays separately recorded as
`executed_command`; it is not the perturbation reproducer. Executable hardlinks
stay inside the retained report tree; rerun argv restores the actual cwd through
`env -C` as well as the captured host environment.

Focused role checks (no matrix or workload execution):

```sh
python3 tools/gomad/testdata/parity/stress-role-check.py --ordinary /absolute/ordinary-gomad --stress /absolute/stress-gomad
```
