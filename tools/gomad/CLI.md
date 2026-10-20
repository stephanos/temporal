# Gomad CLI and maintainer guide

Gomad's execution backend is WASM. The native backend and its maintainer
commands were removed by fn-223; Git history retains their documentation.

Start with the [installation and first-failure guide](README.md). The
[WASM operator guide](internal/backends/wasm/CLI.md) documents installed
commands, configuration, campaigns, inspection, replay, recovery and minimization.
The [product specification](SPEC.md) and [architecture](ARCHITECTURE.md)
define the shared campaign and evidence contracts.

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
