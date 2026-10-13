---
satisfies: [R7]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.13 Generate host and runtime simulation-time codecs from one versioned definition

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

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
### Current source standards handback (2026-10-08)

fn-112.5 current-source acceptance identified one unchanged gofmt observation in runtime/overlay/src/runtime/gomad.go, the missing separator immediately before the simulation-time response-buffer comment. It traces to this task's generated simulation-time extraction at 58b718565044, predates the private-field compactions, and is not a new fn-112.5 change. Reconcile this producer-owned formatting obligation during this task's source acceptance, preserving comments, semantics, generated closure and fixed-identity attribution. The raw overlay remains nonclean; no formatting waiver or full standards pass follows from source-evidence reuse. See ../artifacts/fn-112-gomad-determinism-assurance-and-test/task-5/source-acceptance-20261008/handover.md and its retained format/lint receipts. All existing scope, acceptance and historical evidence remain unchanged.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Simulation-time layout and codecs come from one versioned definition with generated host and runtime-safe consumers; the hand-written offsets, magics and kind constants are gone from both sides.
- [ ] Fixed vectors show request and response bytes are unchanged; truncated frames, wrong magic/kind, nonzero reserved bytes, generation mismatch and time regression fail as before on both consumers.
- [ ] The runtime consumer is exercised by an actual runtime execution on darwin/arm64 and keeps its allocation, stack, dependency and nosplit constraints; linux/amd64 runtime consumption is recorded as incomplete.
- [ ] `make validate` fails on drift of any new generated output, and the schema, templates and generated sources are registered in the version and overlay inventories.
- [ ] The toolchain rebuilds from the changed overlay and `make test-runtime` passes.

## Done summary
Host and runtime simulation-time codecs are generated from one versioned definition (R7); stale pin fixed. Patched-toolchain gates transfer to fn-128/fn-149. Completed under the 2026-10-13 WASM-queue shared-subset closeout amendment; fast tier on gomad c9ac570968 (portable failure set unchanged, architecture, lint, validate, test-wasm).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 58b7185650, 7849801ec6
- Tests: .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-13/ (reviewed evidence), fast tier on c9ac570968, evidence: .flow/artifacts/wasm-section-acceptance-2026-10-12.md
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.


## Format-compatibility amendment (2026-10-09)

Per the spec's 2026-10-09 amendment, byte-for-byte and format compatibility is no longer required. Drop the requirement that generated simulation-time codecs preserve existing bytes and golden vectors. Keep the single versioned definition and generator; replace byte-equality vectors with host/runtime round-trip tests. Must pass `make -C tools/gomad3 overlay-test test-toolchain test-simulation` on current source.
