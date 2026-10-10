# fn-154 checkpoint and first-wave admission audit

Read-only audit against source HEAD `73c507fbf09318f36ae6ddf30ad8a917d801e868` and the current canonical fn-154 tasks. Routing remains the supplied session fallback `jev-unavailable(no_key)` for the requested research model; no actual-model claim. No reviewed artifact, product source, Flow state or git state was changed. No test, gate or bridge ran.

## Decision 1. Admit new record vocabulary before its producers

Tasks .5/.6 cannot currently promise their real simulation checkpoints while all new record vocabulary remains with .8. This is an acceptance dependency cycle, even though string-valued transition production can compile. Task .5 replaces partition_drop with held writes and .6 commits a timeout reset. Both precede .8, whose current scope alone includes public enums, validation, lanes and semantic-codec migration.

The bounded source set for this question was the three task artifacts (.5/.6/.8), `tools/gomad3sim/types.go`, `tools/gomad3sim/record.go`, `tools/gomad3sim/runtime_network_wire.go` and overlay `internal/gomadio/simulation_network.go`.

- `types.go:124` allows listen, dial, accept, write, deliver, close, listener_close, partition, heal, delay, disconnect, reconnect, directional_delay, stop and crash. `:142` defines ok, closed, refused, deadline, partition_drop, stale_drop, reset, capacity and unsupported. Neither list defines a held outcome or timeout outcome/kind.
- `record.go:1078` validates kind through a closed switch; `:1091` validates outcome through another. Unknown strings fail. `:810` calls these checks on every transition, verifies lane order and authenticates the encoded transition digest. Record encode/decode/replay preparation invokes cluster validation (`record.go:18`, `:47`, `:61`). A new held/timeout result therefore fails at the public record boundary even when the overlay can emit it.
- The current network wire is structurally permissive. `runtime_network_wire.go:201` writes Kind and Outcome as strings, alongside Ordinal, endpoints, Connection, Delivery, Bytes, Count, DelayNanos and PayloadSHA256. It decodes the strings at `:215`. Adding a new string alone need not change field layout, but does not bypass semantic validation. The admitted fn-109.14 generated codec may additionally own enum validation, so its exact inventory must decide that part.
- Overlay `simulation_network.go:958` appends/checks replay transitions, and `:1006` assigns transition lanes. A genuinely new timeout kind falls into its unknown lane until that switch changes; public `record.go:1109` has the corresponding lane owner. Merely aliasing timeout to an existing kind/outcome to pass validation would conceal the new persistent timeout contract.

The smallest coherent correction is to move the additive admission portion from .8 into .2, before .5/.6. Extend .2's title/scope/acceptance to admit held-write and timeout record vocabulary, endpoint requirements, host/overlay lane rules, malformed/unknown controls, and any corresponding generated semantic schema/vector/version changes. Add `types.go`, `record.go` and focused record tests to .2's concrete inventory. The task already waits for .1, so that root-module overlap is serialized. Define the producer shapes at this point without emitting them yet. Round-trip synthetic valid records, reject malformed variants and retain every old still-produced shape through this checkpoint.

Keep .8's destructive removal of delivery steps, hashes, delivery-count fields and partition_drop, plus the final coherent identity cutover. Do not move all of .8 earlier; doing so requires coordinated rewrites of producers that .4-.7 have not yet supplied. The statement that checkpoints are unreleased formats does not make a checkpoint that rejects its own valid run results acceptable. Each additive checkpoint needs matched current-build encoding, validation and identity tests; final old-format rejection stays explicit.

This correction needs no new task or task-number change. Alternatively, .5 could own held admission and .6 timeout admission, but both would need the same exact cross-layer Files/Touches and single-codec regeneration duties. Centralizing additive admission in .2 makes that ownership visible once and prevents runtime producers from depending on later .8 acceptance.

## Decision 2. Tasks .1 and .3 can precede .2 in parallel after scoped corrections

The public configuration work in .1 and the error-contract/transport work in .3 have no intrinsic data dependency. Task .1 edits root-module spec/default/record files; .3 edits the nested module's overlay errors, process adapters, modelwire schema/template and generated transport. A stall-error round trip needs no configured limit or semantic network activation frame. Their existing Files/Touches are disjoint from each other. Task .2 overlaps .3's schema/generator outputs and consumes .1's new fields, so .2 must follow integration of both.

The focused targets were task artifacts .1/.2/.3, `internal/gomadtool/generation/protocol/protocol.go`, `simulation/schema/modelwire.json`, `simulation/schema/modelwire.go.tmpl` and overlay `internal/gomadio/network.go`. Prior source inspection of adapter forwarding supplies the existing transport context, without a second plan survey.

Required corrections before declaring that wave ready:

1. **Complete task .3's generator write inventory.** `protocol.go:292` declares a closed schema Errors struct; `:787` rejects unknown JSON fields; `:804` pins the complete sequence of error codes 0 through 21. Adding a stall-timeout field to modelwire.json without updating this source makes generation fail. Task .3 currently lists neither protocol.go nor its adjacent generator tests in Files/Touches. Admit those exact files. `modelwire.go.tmpl:387` also rejects codes above ErrorCapacity, so adding a constant alone is insufficient; update its valid-error bound/check without loosening unknown-code rejection.
2. **Keep generated fan-out owned by .3 until its commit.** `protocol.go:450` lists four modelwire outputs: overlay `gomadmodelwire/wire_generated.go`, its generated test, host `runner/internal/execution/simulation_model_wire_generated.go`, and overlay `gomadsim/model_transport_generated.go`. .3 already allows these paths. The two transport files might remain byte-identical if only the error list changes; derive the actual changed set from generator output and do not force artificial edits. .2 may regenerate the same inventory only after .3 lands.
3. **Separate transport identity proof from connection-state proof.** Task .3 currently requires that changing a deadline cannot clear a persistent reset reason, but `network.go:118` still has a reset boolean and separate deadlines, with no reset-reason state. Its connection mechanics are owned by .4, and autonomous expiry by .6. .3 can prove a stable sentinel/type, process round trips, partial-count preservation and selected wrapper classification using supplied errors. Move actual post-reset SetDeadline/read/write persistence acceptance to .4/.6, or explicitly admit the minimal connection-state change and its files into .3. The former is smaller and keeps .3 independent of the new queue owner.
4. **Bind current fn-155 adapter consumers before dispatch.** .3 already requires exact admission of descriptor error consumers. That decision must use the then-current fn-155 candidate, not assume descriptor_backend.go alone covers every status-to-errno mapping. This does not create a dependency on .1/.2. It preserves the existing fn-155-first priority and task-owned evidence requirements.

The generated choice and live-capability digests do not introduce a hidden dependency between .1 and .3 under these scopes. Their explicit input lists are `protocol.go:564` and `:594`; neither consumes root `gomad3sim` limits/records nor modelwire.json/process_commands.go. They do consume runtime/gomad.go and other named compiler/runtime files. If .3 widens into those files, re-inventory the generated closure before proceeding. Overlay edits still change the toolchain build identity, so tests and final evidence must bind the integrated candidate. No independent worktree result automatically qualifies the combined overlay.

Recommended graph after those edits:

```text
.1 public config ----+
                    +--> .2 config propagation + additive record admission --> .4 --> .5 --> ... --> .12
.3 timeout transport+
```

Set .3's dependency list empty, .2 to [.1, .3], and .4 to [.2]; .3 remains a transitive predecessor of .4. Use disjoint isolated worktrees for the .1/.3 source work. Integrate both before .2, serialize all shared Go/lint/generator/toolchain gates, and bind .2's exact generated semantic-codec inventory after fn-109.14 source acceptance. Do not parallelize .2 with .3 or put both on shared generated outputs. No new spec-wide dependency, native waiver or implementation-priority change follows.
