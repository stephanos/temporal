---
satisfies: [R11]
---
# fn-155-gomad-syscall-level-io-boundary-from.8 Admit the syscall edge's socket entry points in the capability guard and closure policy under the boundary

## Description
Admit exactly the socket entries served by .1's edge under the checked boundary profile from .2. Guarded-mode compiler admission and closure-mode source receipts must contain every admitted operation, while default syscall/x/sys denials remain unchanged. This owner changes capability policy and generated protocol code; it does not grant generic import or trap access.

**Size:** M
**Files:** protocol generation, compiler gomadguard and runtime guard, target source collector and pure capability policy, prepared evidence/cache, profile identity, optional inline record source receipt; adjacent tests.
**Touches:** [tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/toolchain/runtime/overlay/src/cmd/**, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go, tools/gomad3/target/**, tools/gomad3/deterministicio/profile.go, tools/gomad3/record/types.go, tools/gomad3/record/identity.go, tools/gomad3/record/validation.go, tools/gomad3/record/record_test.go, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go, tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_choicewire_generated.go]

### Approach
- Follow [selection-admission-design.md](../artifacts/fn-155-gomad-syscall-level-io-boundary-from/selection-admission-design.md). This description-only admission changes no acceptance, dependency or task status; implementation waits for .2.
- Generate the admitted table from .1's actual platform-specific routes. Bind exact entry/signature, wrapper and bridge body digests, operation and argument roles into the selected profile only. Review pure helpers separately from effectful socket entries. Keep the ordinary unconditional guard on every unreviewed entry.
- Entry-specific conditional guards preserve operation/descriptor-role, inherited/reserved/stale descriptor, address, option/length and fcntl refusal at the edge. Generic-entry guard prefixes must be verified nosplit scalar leaves or run after joint typed pointer handoff. Reuse actual moving-stack fixtures with guard instrumentation; returning from a profile-wide guard is insufficient.
- Collect executable references across the complete compiled application source set, including dead functions, initializers, aliases/dot imports, function values and indirect calls. Join proved function sets conservatively. Unresolved targets, unsafe/reflection/linkname constructions and generic traps without exact operation proof retain their existing source finding. Keep the collector effectful and capabilitypolicy a pure evaluator.
- Treat pinned Go 1.27.1 syscall and golang.org/x/sys v0.47.0 as reviewed boundary substrate through exact source/foreign inventories, sums, wrapper/bridge summaries and init proof. Darwin x/sys has its own trampoline targets; Linux forwarding must terminate at reviewed stdlib entries. Direct no-error assembly, unknown variants and changed bodies stay denied. A socket plus Kill reference remains rejected even in dead code; whole-file approval grants no general API access.
- Retain normalized bounded source-admission evidence in CapabilityClosure, provenance/cache review and Prepared.Record. Add an omitted-by-default inline record.Target receipt, distinct from CapabilityManifest, with selection/profile/policy, source/package/entry summary and init-order identities. Preserve closure-mode refusal of linked manifests and default canonical bytes. Identity/cloning/validation and replay reject missing or mutated required proof before execution; refuse proof overflow rather than truncating it.
- Regenerate exact admitted outputs; hash every admission implementation input. The listed record and choice-wire paths supplement the existing target/compiler/protocol scope. Name any new runtime hook, syscall fixture or patch/version path before editing it.
- Keep the excluded-adapter gRPC closure success requirement open until the actual closure passes. No exception waives nonnetwork references, unknown initialization or forwarding. Source checks and native containment/guard execution remain separate evidence.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go`
- `tools/gomad3/internal/gomadtool/generation/protocol/protocol.go`
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go`
- `tools/gomad3/target/internal/capabilitypolicy/policy.go` and target source/provenance/cache owners
- `tools/gomad3/record/types.go`, `identity.go` and `validation.go`
- .1's final syscall edge and pointer/readiness audit reports
**Optional:**
- `tools/gomad3/deterministicio/grpc_adapter.go`
## Acceptance
- [ ] Boundary off: guarded-mode and closure-mode tests show unchanged denials for `syscall`/`x/sys` entry points and imports.
- [ ] Boundary on, guarded mode: upstream `net` TCP and unadapted gRPC's keepalive `x/sys` socket-option call run without `GOMAD_CAPABILITY_DENIED`; a non-modeled entry point (e.g. `syscall.Kill`) still throws.
- [ ] Boundary on, closure mode: preparing the gRPC workload with its network adapters excluded succeeds, and the admission appears in the prepared target's evidence and profile identity; a package importing non-modeled `x/sys` calls is still rejected.
- [ ] Generated protocol code is regenerated, not hand-edited; `make -C tools/gomad3 validate-toolchain test-toolchain intercept-test test-host` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
