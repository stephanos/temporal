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
Sites retain their existing fingerprints and raw module offsets.

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
minimization reduction chains are checked before rendering. Source paths and
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
