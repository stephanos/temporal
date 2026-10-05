---
satisfies: [R7]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.13 Generate host and runtime simulation-time codecs from one versioned definition

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

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
- Follow MILESTONES.md: root commits this task’s verified progress separately, preserves unrelated changes, and keeps required native acceptance open. Push, stash, worktree creation and history rewrites require separate authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- The current host is linux/arm64 without the patched toolchain. Both darwin/arm64 and linux/amd64 native gates remain incomplete; stock developmental checks do not qualify either.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Simulation-time layout and codecs come from one versioned definition with generated host and runtime-safe consumers; the hand-written offsets, magics and kind constants are gone from both sides.
- [ ] Fixed vectors show request and response bytes are unchanged; truncated frames, wrong magic/kind, nonzero reserved bytes, generation mismatch and time regression fail as before on both consumers.
- [ ] The runtime consumer is exercised by an actual runtime execution on darwin/arm64 and keeps its allocation, stack, dependency and nosplit constraints; linux/amd64 runtime consumption is recorded as incomplete.
- [ ] `make validate` fails on drift of any new generated output, and the schema, templates and generated sources are registered in the version and overlay inventories.
- [ ] The toolchain rebuilds from the changed overlay and `make test-runtime` passes.

## Done summary
Blocked:
# Task 13 acceptance remains open

The source candidate and feasible focused, generation, validation, architecture,
vet and developmental runtime checks are retained in handover.md/evidence.json.
Two fresh independent source audits found no concrete defects (source-audit.md).

This host is linux/arm64 and has no `.toolchain/bin/go`. Acceptance still needs
the supported-platform patched toolchain rebuild, test-runtime, pinned focused
tests, process transport integration, gomad3sim toolchain suite and full
quiescence/nosplit call-chain checks. Developmental stock-runtime execution and
cross-compilation do not supply those results.

The committed-range CLI review excluded uncommitted source and ended
NEEDS_HUMAN; it is not an acceptance receipt. Commits remain user-owned.
Do not mark this task done. MILESTONES.md permits the next reviewed-source task
to proceed while these required acceptance gates remain open.

Blocked:
# Task 13 acceptance remains open

The exact 22-file source candidate and its developmental checks are retained
in `task-13/source-checkpoint.md`, with independent source and checkpoint-boundary
audits. Root is authorized to commit this task's verified progress separately;
the old user-only commit restriction is superseded by MILESTONES.md.

This linux/arm64 host cannot run either supported native platform's patched
toolchain rebuild, runtime vectors, process transport, gomad3sim execution or
full quiescence/nosplit checks. Exact required commands remain in
`task-13/native-gates-open.md`; keep each incomplete until source-bound results
exist. The earlier empty committed-range review did not accept this candidate.

The integrated ad90b462e0 first-party clock bridge also has an inherited stale
policy pin, present at the task-13 base, recorded in source-checkpoint.md. It
needs its owning-task repair; neither a source checkpoint nor native-host
availability waives that non-native gap. Keep task 13 blocked and R7 open.

Blocked:
# Task 13 acceptance remains open

The exact 22-file source candidate and its developmental checks are retained
in `task-13/source-checkpoint.md`, with independent source and checkpoint-boundary
audits. Root is authorized to commit this task's verified progress separately;
the old user-only commit restriction is superseded by MILESTONES.md.

This linux/arm64 host cannot run either supported native platform's patched
toolchain rebuild, runtime vectors, process transport, gomad3sim execution or
full quiescence/nosplit checks. Exact required commands remain in
`task-13/evidence.json`'s `native_commands` and the task's Quick section;
keep each incomplete until source-bound results
exist. The earlier empty committed-range review did not accept this candidate.

The integrated ad90b462e0 first-party clock bridge also has an inherited stale
policy pin, present at the task-13 base, recorded in source-checkpoint.md. It
needs its owning-task repair; neither a source checkpoint nor native-host
availability waives that non-native gap. Keep task 13 blocked and R7 open.

Blocked:
# Task 13 acceptance remains open after the D26 pin repair

Task 13's verified 22-file time-wire source checkpoint is committed at
`58b718565044ab3bc3385d3323ee908a6d54328e`. Its historical reports remain
unchanged. The inherited first-party bridge-pin mismatch they found was repaired
separately by its D26 owner in
`5350185a3601921c0a5f9ba07e1f05bdad7df81f`; it is no longer a current source
blocker. See the retained D26 `task-31/bridge-pin-repair/conductor-checkpoint.md`
and `conductor-verification.json`. The repair grants no generic I/O capability
and changes no runtime bridge source or task-13 codec behavior.

The conductor freshly verified the current bridge SHA-256
`211c01f57125ba62115b1ffce5d2479d3c22116d51a41aefcfb1a576e8b393a9`,
the exact policy SHA-256
`6e8e072c7d47aa73f7e9f05d938ebe625e0e6ba56f317381978e539b30342bd8`,
and the rejection-test SHA-256
`88a4b3046dc1d9103f18c1206c817f05bdec5d49e875a2a74297fc8c9b41b728`.
Policy binds that bridge and the ordered Advance, Current and TakeArrivals
directives. From `tools/gomad3`, these fresh stock Go1.27.1 commands exited 0:

```text
go test -count=1 -tags test_dep ./target -run '^TestBuiltInSimulationLinknamesPinCurrentFirstPartySources$'
go test -count=1 -tags test_dep ./target/internal/capabilitypolicy
```

The exact executable was
`/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`;
GOWORK=off, GOTOOLCHAIN=local, GOENV=off, GOFLAGS empty and GOMAXPROCS=2,
with GOMADSEED and GOMAD3_CHILD_SEED unset. Package times were 0.014s and
0.011s; the complete batch exited 0. An initial path check used repository-relative
paths from the nested-module directory and stopped before either test; the
corrected path check and both tests then succeeded.

Task 13 still requires source-bound patched-toolchain rebuild, runtime vectors,
real process transport, gomad3sim execution and full quiescence/nosplit checks
on native darwin/arm64 and linux/amd64. Exact commands remain in
`task-13/evidence.json`'s `native_commands` and the task's Quick section.
This linux/arm64 host supplies neither qualification. The earlier empty-range
formal review did not accept the committed candidate. Keep task 13 blocked
and R7 open; resolving the pin mismatch closes none of those gates.

Commit verified progress under MILESTONES instruction 5, preserve historical
evidence and unrelated changes, and push only when authorized.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
