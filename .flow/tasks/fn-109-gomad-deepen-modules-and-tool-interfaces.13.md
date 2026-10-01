---
satisfies: [R7]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.13 Generate host and runtime simulation-time codecs from one versioned definition

## Description
Stage 4, R7 (F6). The 40-byte request and 32-byte response of the simulation-time protocol are hand-written twice: in the host Runner and in the allocation-restricted runtime overlay. Add one definition to the existing protocol generator and consume generated layouts on both sides. Bootstrap generation is already complete and is not part of this task.

**External coordination:** this edits the runtime overlay and its source inventory, which fn-110 (runtime patch minimization) also touches. Check `flowctl tasks --spec fn-110-gomad-minimize-the-runtime-patch` and do not run concurrently with an fn-110 overlay task. An overlay change alters the toolchain build key and triggers a toolchain rebuild; never rebuild `.toolchain` while another task's tests are running.

**Size:** M
**Files:** new schema and templates under `tools/gomad3/simulation/schema/`, `internal/gomadtool/generation/protocol/protocol.go` and test, `runner/internal/execution/simulation_time.go` plus a generated host file, `toolchain/runtime/overlay/src/runtime/gomad.go` plus a generated runtime file, `toolchain/version/version.json` and its generated consumers, tests.
**Touches:** [tools/gomad3/simulation/schema/**, tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/runner/internal/execution/simulation_time*.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3/internal/gomadtool/conformance/**]

### Approach
- Host hand-written codec: `runner/internal/execution/simulation_time.go:14-131` (sizes, kinds, magic, offsets, `encode`/`decode`/`validate` for request and response, `zeroSimulationTime`, `encodeSimulationActivationTime`).
- Runtime hand-written copy: `toolchain/runtime/overlay/src/runtime/gomad.go:57-69` (sizes, response kinds, magics `GOMADTQ\x01` / `GOMADTR\x01`) and `:1068-1170` (`gomadSimulationTimeQuiesce`, `Write`, `Read`, `Put64`, `Put32`, `Get64`).
- Generator: `GenerateProtocols` (`internal/gomadtool/generation/protocol/protocol.go:315-408`) already emits host and runtime outputs for I/O (`deterministicio/schema/iowire_runtime.go.tmpl` -> `runtime/gomad_iowire_generated.go`) and choice wire. Add a simulation-time schema with an explicit version, a host template and a runtime-safe template, and register both outputs the way `modelOutputs` is (`:381-386`).
- Runtime constraints are the hard part: the generated runtime code runs on the allocation-forbidden quiescence path with a static response buffer (comment at `gomad.go:71-72`). No allocation, no reflection, no new imports, no stack growth beyond today's, same `nosplit` annotations. Descriptor I/O, native timers and the quiescence hook stay hand-written in runtime.
- Inventories: register the schema, templates and generated files in `toolchain/version/version.json` (`overlay_allowlist`, `:114`) and regenerate; `make validate` must detect drift in each generated output.
- Preserve bytes exactly: magic, kind numbers, reserved-zero checks, generation correlation and monotonic-time checks. Shared vectors (golden, truncated, wrong magic, wrong kind, nonzero reserved, generation mismatch, time regression) must be consumed by the host codec test and by the runtime through an actual runtime execution (conformance `test-runtime` or the process simulation tests), not only by a host re-implementation.

### Investigation targets
**Required:**
- `tools/gomad3/runner/internal/execution/simulation_time.go:14-165`
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:55-75,181-200,1009-1170`
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go:315-410,497-535,668-695`
- `tools/gomad3/deterministicio/schema/iowire_runtime.go.tmpl` and `toolchain/runtime/overlay/src/runtime/gomad_iowire_generated.go`
- `tools/gomad3/toolchain/version/version.json`
**Optional:**
- `tools/gomad3/runner/internal/execution/simulation_time_test.go`
- `tools/gomad3/ARCHITECTURE.md` section "Binary protocol ownership"

### Quick commands
```bash
cd tools/gomad3
make generate && make validate
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./internal/gomadtool/... ./runner/internal/execution -run 'SimulationTime|Protocol'
make toolchain && make test-runtime
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,integration ./runner/internal/execution -run 'TestRootProcessSimulationUsesRunnerTransport'
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Simulation-time layout and codecs come from one versioned definition with generated host and runtime-safe consumers; the hand-written offsets, magics and kind constants are gone from both sides.
- [ ] Fixed vectors show request and response bytes are unchanged; truncated frames, wrong magic/kind, nonzero reserved bytes, generation mismatch and time regression fail as before on both consumers.
- [ ] The runtime consumer is exercised by an actual runtime execution on darwin/arm64 and keeps its allocation, stack, dependency and nosplit constraints; linux/amd64 runtime consumption is recorded as incomplete.
- [ ] `make validate` fails on drift of any new generated output, and the schema, templates and generated sources are registered in the version and overlay inventories.
- [ ] The toolchain rebuilds from the changed overlay and `make test-runtime` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
