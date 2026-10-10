---
satisfies: [R1, R3, R9]
---
# fn-155-gomad-syscall-level-io-boundary-from.2 Select the boundary and adapter exclusions per run and bind them into artifact identity

## Description
Add checked per-run boundary selection and adapter exclusions, then carry that same selection through preparation, execution, recorded artifacts and replay. Under the selected boundary, only legacy TCP hooks stand aside; resolver, interface and unsupported-network refusals retain their current ownership.

**Size:** M
**Files:** `deterministicio/profile.go`, `adapter_registry.go`, record and Runner selection consumers, CLI and qualification consumers, preparation/inspection and target cache owners, selected bootstrap generation, runtime activation and TCP routing; adjacent tests.
**Touches:** [tools/gomad3/deterministicio/**, tools/gomad3/record/**, tools/gomad3/runner/**, tools/gomad3/cmd/gomad/**, tools/gomad3/qualification/set/**, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/gomadio.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/internal/preparation/preparation.go, tools/gomad3/internal/preparation/inspection.go, tools/gomad3/internal/preparation/preparation_test.go, tools/gomad3/internal/preparation/inspection_test.go, tools/gomad3/internal/preparation/composition_test.go, tools/gomad3/qualification/analysis/prepared_review.go, tools/gomad3/qualification/analysis/prepared_review_test.go, tools/gomad3/target/target.go, tools/gomad3/target/prepared_cache.go, tools/gomad3/target/target_test.go, tools/gomad3/target/prepared_cache_test.go, tools/gomad3/toolchain/runtime/overlay/src/net/gomad.go, tools/gomad3/internal/gomadtool/generation/protocol/protocol.go, tools/gomad3/internal/gomadtool/generation/protocol/protocol_test.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadwire/wire_generated_test.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_iowire_generated.go, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/protocol_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadvfd/descriptor.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadvfd/descriptor_test.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_vfd.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_vfd_export_test.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_vfd_test.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend_test.go]

### Approach
- Follow the checked-selection and startup decision in [selection-admission-design.md](../artifacts/fn-155-gomad-syscall-level-io-boundary-from/selection-admission-design.md). This description-only admission changes no acceptance, dependency or task status; implementation waits for .1's acceptance.
- Give deterministicio one immutable checked selection constructor and one checked recorded reconstruction. Copy, sort and validate exact exclusions; reject unknown names and duplicates before preparation, custom preparer entry or downloads. An absent selection returns the existing Default singleton with unchanged fixed-input identity and omitted JSON/environment fields.
- Filter the adapter registry before version, replacement, cache, sum-pin, download and rewrite work. Bind all explicit exclusions, including unused modules, into the selected profile's implementation and inventory identities.
- Carry the same checked profile through preparation and selected inspection, prepared-target validation, analysis, local/coordinator execution, recorded evidence, replay/verify-only, resume/shards/minimization, corpus reopen, analyze/qualify and cache restore. Keep target selection identity neutral to avoid a target/deterministicio import cycle. Preserve custom-preparer stage and cleanup/error precedence.
- Preserve non-default selection markers in corpus identity. Qualification sets remain homogeneous; reject mixed boundary/exclusion selections before analysis, command execution and shard partitioning without widening the set-report schema.
- Preserve kind 1's 212-byte default bootstrap and vectors. Generate a bounded selected-profile frame with canonical selector, exclusion bitset, registry/policy binding and whole-frame checksum, within the existing 4096-byte IOConfig bound. Validate it on host and runtime; malformed or unsupported generic dispatch fails before target initialization.
- Own one TCP-specific predicate consumed by standalone and later simulation hooks. Preserve compiler semantic probes, resolver/interface behavior and UDP/IP/Unix/DNS refusals. Add the selected localhost gomadio event without changing default transcript bytes.
- Wire runtime-owned early scalar selection through the frozen .1 leaf/runtime interfaces before gomadio initialization. The current leaf Enabled/SetEnabled test switch requires a backend and cannot alone represent selected pre-registration ownership. The early accessor must remain nosplit/norace-safe and avoid the syscall-to-gomadio import cycle; a selected operation with no backend refuses rather than entering a host socket.
- Prove that a registration anchor is linked and initializes before every socket-capable initializer, including variable initializers and indirect calls. Unproved order and raw-syscall-only targets without that anchor fail preparation. Later backend refusal does not replace startup admission.
- Admit only the listed existing descriptor, runtime and backend paths for early activation/registration and their regressions. This admission selects no new overlay file or patch hunk. Name any additional hook, patch, version allowlist or generated input path before editing it. Generate derived outputs and hash every actual implementation input; unrelated generated drift remains outside scope.

### Investigation targets
**Required:**
- `tools/gomad3/deterministicio/profile.go` and `adapter_registry.go`
- `tools/gomad3/internal/preparation/preparation.go` and `inspection.go`
- `tools/gomad3/record/validation.go` and Runner recorded-selection consumers
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go` and `gomad_vfd.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadvfd/descriptor.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go`
**Optional:**
- memory entries profile-adapter-changes-leave-libc-2026-10-01 and shard-merge-and-prepared-target-cache-2026-09-29
## Acceptance
- [ ] `gomad` run/explore/qualify accept the boundary option and an adapter-exclusion option; both appear in the recorded environment and artifact identity.
- [ ] Excluding an unknown adapter is rejected before preparation; a workload needing an excluded adapter fails with the existing closure error.
- [ ] Replay of an artifact under a different boundary or adapter set is refused before execution (tests for each).
- [ ] With the boundary selected, std-level TCP hooks do not record; with it off, existing behavior and identities are unchanged.
- [ ] Under the boundary, `localhost` resolution still records at the gomadio level (test).
- [ ] Selecting the boundary with an injected unsupported dispatch table fails at startup with a clear error (unit test).
- [ ] `make -C tools/gomad3 test-host` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
