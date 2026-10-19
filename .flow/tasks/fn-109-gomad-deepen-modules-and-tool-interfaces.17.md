---
satisfies: [R12]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.17 Select backend-specific network listener and connection implementations at creation

## Description

Source-work resumption (2026-10-07). The owner requested unblocking and completing the source tasks on the current gomad branch. This task returns to todo for its retained source work, with all dependency/admission and acceptance requirements preserved except the expressly scoped owner decisions in [source-unblocking-20261007/owner-decisions.md](../artifacts/source-unblocking-20261007/owner-decisions.md). Historical Done summary and Evidence below retain their original provenance; current lifecycle status comes from flowctl. Native qualification remains deferred under fn-128/fn-149 and is not revived by this resumption.


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 5, first half of R12 (F11): the network handle family. `Listener` and `Conn` each hold fields for three backends (process handle, in-process simulation endpoint, standalone state) and every method branches on which is set. Choose the implementation once at creation and let each implementation own its valid state. Filesystem handles follow in the next task; do one family at a time.

**External coordination:** overlay edit; same fn-110 and toolchain-rebuild rules as the simulation-time task.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/{network.go,process_network.go,simulation_network.go}`, new per-backend files, tests, `toolchain/version/version.json` for new overlay files.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/**, tools/gomad3/toolchain/runtime/overlay/src/net/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3sim/*_test.go, tools/gomad3/network_handles_ownership_test.go, tools/gomad3/simulation_gate_selection_test.go, tools/gomad3/runner/internal/execution/*_test.go, tools/gomad3/Makefile]

### Approach
- Current shape: `Listener` (`network.go:41-52`: `processHandle`, `owner`, `network`, local `pending`/`closed`/`deadline`) and `Conn` (`:54-68`). Process dispatch branches at `network.go:214`, `:254`, `:288` (listener) and `:330`, `:409`, `:485`, `:523`, `:542`, `:569`, `:586`, `:602` (connection), forwarding to `processNetworkAccept` / `ListenerClose` / `ListenerSetDeadline` / `ConnRead` / `ConnWrite` / `ConnOperation` (`process_network.go:55-108`). Simulation versus standalone is a second branch on `network`/`owner`.
- Target: creation (`ListenTCP` `network.go:102`, `DialTCP` `:145`, accept, `processNetworkConn` `process_network.go:110`) picks a private implementation of a small internal interface; `Listener` and `Conn` keep their exported methods and the patched `net` callers do not change. Each implementation holds only its own fields, so an impossible combination cannot be constructed.
- Not a registry: three concrete implementations selected by the existing backend conditions, no plugin mechanism, no generic dispatch helper that just relocates the `if`.
- Domain model stays shared: `simulation_network.go` (1,102 lines) keeps the semantics for both simulation backends; do not duplicate it per implementation. Host-side registration (`registerProcessNetworkConn` `:259`, `revokeProcessNetworkResources` `:300`) keeps incarnation-bound revocation.
- Preserve: local-model lock ownership and ordering, deadlines on accept/read/write, close and reset semantics (EOF on graceful stop, reset on crash), duplicate bind rejection, partial I/O results, capacity errors, stale-incarnation rejection before model mutation, and transcript recording.
- Shared operation tests run the same cases against standalone, in-process simulation and process backends; backend-specific tests keep the hard-isolation distinctions that only the process backend provides. Include actual process cases in the existing Runner root-integration selector so unavailable direct-root transport skips cannot masquerade as process coverage. A root-module architectural ownership test may check the production AST, with retained expected failure on the old optional-state handles and passing final implementation; behavior tests assert operation outcomes, not source text.
- Connect every new process case to the canonical `make test-simulation` integration filter used by CI, not only to Runner's case list. Preserve the separately selected forward-clock regression and the existing strict-delay watchdog exclusion. A focused gate-selection regression must demonstrate actual selection of these cases and the preserved exclusion, with retained old-filter failure; adding names behind an excluding filter is incomplete coverage.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go` (654 lines)
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go:23-125,259-310`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go` (outline first)
- `tools/gomad3sim/network_toolchain_test.go`, `tools/gomad3/runner/internal/execution/io_network_toolchain_test.go`, `io_net_bind_toolchain_test.go`
**Optional:**
- `tools/gomad3/deterministicio/network_patch_test.go`

### Quick commands
```bash
cd tools/gomad3
make generate && make validate
.toolchain/bin/go test -count=1 -tags test_dep internal/gomadio
make toolchain && make overlay-test
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Network|NetBind'
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Network|Process|Backend'
```

### Constraints
- Commit each verified task separately under MILESTONES item 5, including its implementation, tests, documentation and Flow records. Preserve unrelated changes; keep unavailable native gates and acceptance open. Do not push without authorization.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual development host is `linux/arm64`, not the inherited `darwin/arm64` assumption. Native `darwin/arm64` and `linux/amd64` runtime/process gates remain incomplete; stock source checks do not qualify them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Network listener and connection creation selects one private backend implementation; exported handle methods no longer branch on `processHandle` or backend fields.
- [ ] Each implementation owns only its valid state; the simulation network model is shared, not duplicated, and no generic backend registry exists.
- [ ] Shared operation tests pass for standalone, in-process and process backends; backend-specific tests retain hard-isolation distinctions.
- [ ] Duplicate bind, deadline, close/reset, partial I/O, capacity, stale incarnation and replay divergence behave as before, with validation before mutation.
- [ ] Overlay inventories validate, the toolchain rebuilds, and gomad3sim network/process tests pass on darwin/arm64; linux/amd64 is recorded as incomplete.

## Done summary
Won't do: native backend retired by the 2026-10-19 owner decision; closed without implementation.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs:
## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.


## Format-compatibility amendment (2026-10-09)

Per the spec's 2026-10-09 amendment, byte-for-byte and format compatibility is no longer required. The connection-core merge first added here moved to fn-154-gomad-simpler-virtual-network-with (simpler virtual network with stalling partitions), which rewrites the same connection code; it is not part of this task. Must pass `make -C tools/gomad3 overlay-test test-toolchain test-simulation` on current source.
