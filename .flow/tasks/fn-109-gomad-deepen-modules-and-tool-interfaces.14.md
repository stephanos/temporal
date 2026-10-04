---
satisfies: [R14]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.14 Hide generic model-wire slots behind typed network and volume commands

## Description
Stage 4, R14 (S2). Process-backend network and volume adapters fill and read the generic `String1/String2/Int1/Int2/Uint1/Uint2` slots of the compact model envelope by hand at about 70 sites, so each site must know which slot means what per operation. Introduce typed domain commands and one translation owner per domain; the wire bytes do not change.

**External coordination:** overlay edit; same fn-110 and toolchain-rebuild rules as the simulation-time task.

**Size:** M
**Files:** `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go`, `internal/gomadfs/{process_volume.go,process_volume_host.go,fs.go}`, new command/translation files beside them, tests, `toolchain/version/version.json` for new overlay files.
**Touches:** [tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadmodelwire/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk]

### Approach
- Slot users (count of generic-slot references): `gomadio/process_network.go` 27 (client `exchangeProcessNetwork` `:125`, host `applyProcessNetworkOperation` `:165-257`), `gomadfs/process_volume.go` 15 (for example `processResolve` `:27-30`, `processMkdir` `:32-39`, `processHandleOperation` `:126`), `gomadfs/process_volume_host.go` 27 (host application), `gomadfs/fs.go` 1. Operations are declared in `simulation/schema/modelwire.json` (`network_operations`, `volume_operations`); the generated envelope is `internal/gomadmodelwire/wire_generated.go`.
- One translation owner per domain maps each typed command to the existing `gomadmodelwire.Request` and each `Response` back to a typed result; client and host sides both use it, so slot meaning is written once. Framing stays in `gomadmodelwire`; operation semantics stay in the network and volume domain code.
- Builders that only rename the same slots add no depth. The test of success is that no adapter code outside the translation owner names a generic slot.
- Accepted wire shapes are unchanged: do not start rejecting requests that decode today (stronger shape rejection is a separate contract change). Preserve partial read/write counts returned together with an error, invalid/stale handle classification (`decodeProcessNetworkError` `process_network.go:343`, `decodeProcessVolumeError` `process_volume.go:226`), capacity results and "operation unavailable on this backend".
- Equivalence harness: before editing, capture encoded request and response bytes for every network and volume operation with representative arguments (including partial I/O with error); afterwards the typed path must produce identical bytes and decode to the same results.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go` (375 lines)
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadfs/process_volume.go` (272), `process_volume_host.go` (283)
- `tools/gomad3/simulation/schema/modelwire.json`, `modelwire.go.tmpl`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadmodelwire/wire_generated.go`, `wire_generated_test.go`
**Optional:**
- `tools/gomad3sim/runtime_network_wire.go`, `runtime_volume_wire.go` (host-side detached model wires, a different contract; do not merge)

### Quick commands
```bash
cd tools/gomad3
make generate && make validate
.toolchain/bin/go test -count=1 -tags test_dep internal/gomadio internal/gomadfs internal/gomadmodelwire
make toolchain && make overlay-test
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
```

### Constraints
- The conductor commits each verified task separately under MILESTONES item 5, preserving unrelated changes. Older user-only commit instructions are superseded; unavailable native gates remain incomplete and Flow acceptance stays open. Push only when authorized.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual host is linux/arm64 with stock Go. Native darwin/arm64 and linux/amd64 qualification remain incomplete; source checks and developmental stand-ins do not qualify either platform.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Typed network and volume commands own argument and response semantics; only one translation owner per domain names the generic `String*/Int*/Uint*` slots.
- [ ] Fixed vectors for every network and volume operation produce the same request and response bytes as before.
- [ ] Partial I/O with errors, invalid and stale handles, capacities and unavailable backend operations keep their domain information and error values.
- [ ] No previously accepted wire shape is newly rejected.
- [ ] Overlay inventories are regenerated and validated; overlay and gomad3sim process tests pass on darwin/arm64, with linux/amd64 recorded as incomplete.

## Done summary
Blocked:
# Task 14 acceptance remains open

The typed-command source candidate, pre-edit byte fixtures, red/green ownership
check, generation/validation and developmental tests are retained in handover.md
and evidence.json. Two fresh independent source audits found no concrete defect;
the conductor reran the bounded frozen-batch checks (source-audit.md).

This linux/arm64 host has no patched `.toolchain/bin/go`. Acceptance still needs
the supported darwin/arm64 and linux/amd64 rebuild, pinned focused tests, overlay
suite and real process/simulation conformance. External stock-runtime stand-ins
do not supply those results. Existing broad developmental vet findings are
retained in unchanged files without suppressing a gate.

Keep R14 and task 14 open. The user owns commits. MILESTONES.md permits the
next source task to advance after this reviewed candidate, while native adoption
and the final qualification tasks remain open.

Blocked:
# Task 14 acceptance remains open after source checkpoint

The exact task-14 source boundary and task-owned evidence are verified for a
separate progress commit. See source-checkpoint.md, checkpoint-reconstruction.json
and the original source-audit.md. Source checks are not supported-host acceptance.

Both native darwin/arm64 and linux/amd64 still require the patched toolchain,
overlay focused/full tests, real process/simulation conformance, gomad3sim toolchain
tests and full host gates listed in evidence.json. This linux/arm64 host cannot
provide those results; external stock-runtime stand-ins remain developmental.

Keep task 14 and R14 open until all required gates pass. The conductor commits
verified progress under MILESTONES item 5, preserving successor working-tree bytes
and unrelated changes. The older user-only commit instructions are superseded.
## Evidence
- Commits:
- Tests:
- PRs:
