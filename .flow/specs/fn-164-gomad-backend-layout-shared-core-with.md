# Gomad backend layout: shared core with native and wasm backends

## Goal & Context

Gomad has two execution backends. The native backend runs a patched Go toolchain with process supervision and a native I/O transcript. The wasm backend runs a stock Go compiler plus a source overlay inside a pinned Wasmtime helper. Today the two are not peers. Native is the implicit default woven through `runner`, while wasm plugs in through the four-method `runner/backend.Provider` interface from a separate module that imports 17 gomad3 packages and scrapes 27 functions out of the native runtime overlay source by name.

That shape has three costs. Nothing says which gomad3 packages are backend-neutral, so any native change could break wasm and no CI runs wasm at all. Wasm-only command changes land in the shared command grammar inside gomad3 and pay the 90-minute native CI bill. The shared runtime choice semantics have no owner: the wasm overlay depends on the private layout of a native source file.

This spec restructures `tools/gomad3` into one module with a shared core and two backend directories, so the rule "everything outside `backends/` is shared" is enforceable by a test, CI can filter by directory, and the runtime semantics both backends compile into guests become a declared contract.

## Architecture & Data Models

Target layout, one Go module:

```
tools/gomad3/
  cmd/gomad/            one launcher; backend selected by installation manifest
  cmd/gomadtool/        developer tooling
  record/ choice/ artifact/ world/ simulation/network/     evidence contracts and models
  runner/               campaigns, exploration, replay, minimize, inspect (backend-neutral)
  target/               Spec, Prepared and capability types only
  captured/ volume/     today deterministicio/readonlymount and deterministicio/volume
  qualification/        repeat, compare, report
  hostexec/ hostfs/ internal/canonicaljson
  runtimesemantics/     choice functions and generated wire codec that every runtime overlay includes verbatim
  backends/
    backend.go          Provider interface, Capabilities, registry
    native/             provider, execution (supervisor, bootstrap, coordinator processes),
                        transcript (today deterministicio root), build (toolchain-bound target
                        preparation, adapters, livecap), toolchain, upgrade
    wasm/               provider, wasi, wasmhost, overlay, installation, simulation, qualification
```

Dependency direction:

```
cmd -> backends/native -+
cmd -> backends/wasm   -+-> shared packages -> stdlib
shared packages  -/->  backends/*
backends/native  -/->  backends/wasm, and the reverse
```

Package moves:

| Today | Target | Reason |
| --- | --- | --- |
| `runner/internal/execution`, `runner/coordinator_process_unix.go`, native branches in `runner.go`, `backend.go`, `replay_operation.go` | `backends/native` | This is the native backend; 11k of runner's 22k lines. |
| `deterministicio` root | `backends/native/transcript` | Native I/O transcript protocol. Only `volume` and `readonlymount` are shared, and wasm imports only those. |
| `target` build, adapters, capability policy | `backends/native/build` | They import `toolchain` for build keys and pinned identity. `Spec` and `Prepared` stay shared because the Provider interface is typed on them. |
| `toolchain`, `upgrade`, `internal/gomadtool/{conformance,generation}` | `backends/native` | The import closure of the patched toolchain. |
| 26 functions and 2 types extracted from native `runtime/gomad.go`, plus the generated choice wire file | `runtimesemantics/` | Replaces AST scraping with a file both overlays copy in; the actual reviewed declaration inventory corrects the earlier estimate of 27 functions. |
| `tools/gomad_wasm/*` | `backends/wasm` | Removes a module, two `replace` directives, and the cross-module path to the native runtime root. |
| `cmd/gomad/api` plus the `gomad` and `gomad-wasm` launchers | one `cmd/gomad` | Provider declares `Capabilities`; the command layer registers only matching flags. Replaces per-flag reject lists. |

Unchanged: `record`, `choice`, `artifact`, `world`, `simulation/network`, `hostexec`, `hostfs`, `canonicaljson`, most of `runner`. `gomad3sim` and `gomad3integration` stay in the server root module so Temporal code can import the simulation harness without pulling gomad3.

## API Contracts

Provider interface, extended from today's four methods only where the native extraction proves a path `runner` currently reaches directly:

```go
package backends

type Provider interface {
    Capabilities() Capabilities
    Prepare(context.Context, target.Spec) (target.Prepared, error)
    ValidatePrepared(target.Spec, target.Prepared, []string) error
    ValidateReplay(context.Context, *artifact.Opened) (target.Prepared, error)
    Run(context.Context, Request) (Result, error)
}

type Capabilities struct {
    Name            string   // "native" | "wasm"; appears in artifact manifests unchanged
    ReplayModes     []record.ReplayMode
    ChoiceProfiles  []string
    Simulation      bool     // multi-instance cluster execution
    ReadOnlyMounts  bool     // native mount capture
    IOTranscript    bool     // native syscall transcript evidence
    Commands        []string // command names the launcher registers for this backend
}
```

`Request`, `Result`, `ChoiceRequest`, `Termination` and `FailureError` keep their current shapes. The command layer asks `Capabilities()` before registering flags; a flag outside the declared set is not parsed, so the error is the ordinary unknown-flag error rather than a backend-specific rejection message.

`runtimesemantics` contract: one Go source file and one generated wire file. Each runtime overlay includes them byte for byte, and each overlay's identity hashes them. The native inventory tests guard the exported function set.

Import rule, enforced by the architecture test with the current layout before anything moves:

```
shared/*          may import  shared/*
backends/<b>/*    may import  shared/*, backends/<b>/*
cmd/*             may import  shared/*, backends/*
```

## Edge Cases & Constraints

- **Identity rotation.** The wasm model identity hashes the overlay builder's bytes, and the native build key hashes toolchain sources. Changing import paths inside those files changes the bytes. Every cached preparation invalidates once and every retained artifact's model identity stops matching the new binary. Treat the move as an explicit version bump of those identities and record it in the artifact compatibility notes.
- **Path-aware tooling.** The architecture analyzer's purity rules and effect allowances name directories, including a special case for the wasm `wasi` package. They move with the packages in the same change.
- **Evidence schemas do not change.** `gomad3.*` identity strings, artifact layouts, choice records and replay comparison stay byte-identical so native tooling keeps inspecting wasm artifacts and vice versa.
- **Provider growth bound.** If the native extraction needs more than three new Provider methods, stop and widen the interface deliberately rather than keep native inside `runner`.
- **Ordering.** Steps that move code without changing behavior land first and independently. Native Provider extraction and launcher unification follow the mechanical layout work. The owner-approved replacement for the Go-release-boundary wait is targeted compatibility and replay coverage: compile the shared Provider consumers, verify both backends' command/flag contracts, inspect genuine pre-move artifacts and check explicit model-identity rejection, and run fresh WASM recording with exact detached replay on the integrated candidate. These checks do not qualify patched-native execution on an unsupported host.
- **CI.** Path filters follow the directories: shared changes run both backends, `backends/native` runs native jobs, `backends/wasm` runs the wasm job (stock Go plus a cached helper build).

## Acceptance Criteria

- **R1:** An architecture test enforces the import rule above against the current tree before any package moves, with an allowlist naming the gomad3 packages the wasm backend may import. Errors: a forbidden edge fails the test naming both packages; an allowlist entry that no longer exists fails the test.
- **R2:** `runtimesemantics` exists, both runtime overlays include its files verbatim, and the wasm overlay builder no longer parses the native runtime source. Errors: a missing or renamed exported function fails the native inventory test; a byte difference between an overlay's copy and the source fails that overlay's identity test.
- **R3:** `toolchain`, `upgrade`, the `deterministicio` root package and the conformance and generation tooling live under `backends/native`, and no shared package imports them. Errors: no error surface beyond R1.
- **R4:** `tools/gomad_wasm` is a directory under `backends/wasm` of the gomad3 module; its separate `go.mod` and replacement of the gomad3 core are gone. The existing server simulation dependency retains one `go.temporal.io/server => ../..` replacement in the merged module; shared core dependency closures remain server-free. `make -C tools/gomad3 test-wasm` runs the wasm suite with stock Go and never builds the patched native toolchain. Errors: a wasm test that references the native toolchain build fails the test target.
- **R5:** `target` exports only `Spec`, `Prepared` and capability types from the shared package; toolchain-bound preparation lives in `backends/native/build`. Errors: no error surface beyond R1.
- **R6:** The native backend implements `backends.Provider` and `runner` reaches native execution only through that interface; `runner` imports no `backends/*` package. Errors: a native-only code path left in `runner` fails R1; a Provider method added beyond `Capabilities` plus three is recorded in Decision Context with its reason.
- **R7:** One `cmd/gomad` launcher registers commands and flags from `Capabilities()`; a flag the selected backend does not declare is rejected as an unknown flag, and no per-backend reject list remains in the command layer. Errors: unknown flag → status 2 with the standard usage message; a manifest naming an unregistered backend → status 2 naming the backend.
- **R8:** Retained artifacts produced before the move inspect unchanged; replay of a pre-move artifact fails with an explicit model identity mismatch naming both identities rather than a generic error. Errors: as stated.
- **R9:** CI runs the wasm suite on changes under `backends/wasm` and under shared packages, and does not run native jobs for changes confined to `backends/wasm`. Errors: no error surface beyond workflow syntax validation.

## Boundaries

- No change to evidence schemas, identity namespaces, artifact layout or replay semantics.
- No physical split into separate modules or repositories; one module, directory boundaries, test-enforced.
- `gomad3sim` and `gomad3integration` stay in the server root module.
- No new backend. The layout admits one, but adding it is separate scope.
- No performance work, no qualification changes, no change to what either backend supports.
- The Rust helper stays under `backends/wasm/wasmhost`; its protocol and pinned engine are untouched.

## Decision Context

The owner approved replacing the calendar-based Go-release-boundary prerequisite with the targeted compatibility and replay checks above. This removes the scheduling hold on R6/R7, not their acceptance requirements or deferred native qualification.

The module fold retains the pre-existing server dependency needed by the WASM Temporal simulation harness. Removing that replacement would change the integration boundary rather than merely fold the two tool modules; no new dependency or shared-core server coupling is admitted.

Native extraction may add three typed Provider methods: `Describe` replaces Runner's direct native profile/toolchain/semantic configuration reads; `Execute` replaces native bootstrap execution and its complete detached observations; `Evidence` replaces shared corpus decoding of native transcript operations. Existing `Request` and `Result` remain unchanged. Execution journal sinks may be concrete host-only `*os.File` handles with the original Runner open/sync/close ownership; no callbacks, opaque native data or guest host-I/O access are introduced. Implementation and misuse coverage remain acceptance requirements.

Native sits inside `runner` because it came first and wasm was added as the first external Provider. Keeping that asymmetry means every rule about what is shared is implicit, and the only safe CI policy is to run everything on every change. Making native a peer behind the same interface is more work than an import allowlist alone, which is why the mechanical steps (R1 through R5) are ordered first and the extraction (R6, R7) last.

Rejected: splitting wasm into its own repository or keeping it as a separate module with a published core API. Both freeze the core surface before it is understood, and the wasm backend already imports 17 core packages. Rejected: forking evidence schemas per backend; cross-backend comparison of the same artifact is one of the few tools that locates a bug in Temporal rather than in a backend. Rejected: a lint-only boundary without moving packages; the current `gomad3` name means both "the native backend" and "the core", and that ambiguity is what the layout change removes.

The `runtimesemantics` contract is small and lands early because it is the one coupling that can break silently: a native refactor of `runtime/gomad.go` changes wasm's build key today without any wasm test noticing until the next guest build.
