# Gomad v3 compiler-defined standard-library interception

## Status

Proposed design and implementation plan.

## Decision

Replace eligible standard-library source hunks with a small, version-pinned Go
compiler pass that instruments selected function and method definitions before
inlining. Keep the hook implementations as additive same-package Go files such
as `src/os/gomad.go` and `src/net/gomad.go`.

Do not rewrite call sites and do not attempt linker-only interposition. Function
definition interception is the only single seam that naturally covers direct
calls, concrete methods, interface dispatch, method values, method expressions,
and inlining.

This is a hybrid design. It should remove the repetitive `net` and `os`
entry-point hunks, but it should not try to replace runtime changes that alter
control flow inside the scheduler, timer heap, initialization, or testing
lifecycle.

## Goals

- Eliminate most repetitive standard-library patch hunks.
- Avoid a separate AST parser, translated source tree, or whole-file fork.
- Keep each interception visible in one declarative, Go-version-specific list.
- Preserve the exact upstream function body for the unhandled and disabled
  paths.
- Cover every Go invocation form, including interface calls and inlined calls.
- Fail during toolchain compilation if a target or hook changes incompatibly.
- Keep the interception implementation behind one small compiler interface.

## Non-goals

- Eliminating all Gomad runtime patches.
- Providing general-purpose function interposition to arbitrary programs.
- Supporting unknown Go versions or signatures dynamically.
- Replacing the current I/O model with a virtual syscall or file-descriptor
  kernel.
- Reintroducing Gomad v2's application and standard-library AST translation.
- Supporting generic, assembly-defined, or bodyless interception targets in the
  first implementation.

## How a Go invocation reaches code

The Go runtime does not resolve ordinary package calls at execution time.

For a direct call such as:

```go
os.Stat(name)
```

the compiler resolves `os.Stat` during type checking. During SSA construction,
`cmd/compile/internal/ssagen` obtains the function's linker symbol and emits a
static call operation. The object file contains a call relocation referencing
that symbol. The linker resolves the relocation to a text address, and the CPU
calls that address directly.

Concrete method calls follow the same path after method selection is resolved;
the receiver is passed as an argument. Other invocation forms differ:

- an interface call loads a method code pointer from the interface's itab and
  calls it indirectly;
- a function or method value calls through a closure code pointer;
- compiler-generated method wrappers adapt receiver shapes and method
  expressions; and
- an inlined invocation has no call instruction or relocation left for the
  linker to redirect.

The Go 1.26.4 implementation makes these paths explicit:

- `cmd/compile/internal/ssagen/ssa.go` emits `OpStaticLECall`,
  `OpInterLECall`, or `OpClosureLECall` depending on the resolved call form;
- `runtime/iface.go` fills `itab.Fun` from method metadata; and
- devirtualization and inlining run before final function compilation in
  `cmd/compile/internal/gc/main.go`.

Consequently, changing only direct call relocations is incomplete. It would
also need to update itabs, function references, generated wrappers, reflection
metadata, ABI wrappers, and any calls not already removed by inlining.

## What Gomad v2 does

Gomad v2 translates nearly all application and standard-library source at the
AST level. Its standard-library hook mechanism is definition-oriented rather
than a simple call-site substitution:

- `internal/translate/rewrites.go` identifies selected non-method function
  declarations;
- it removes the upstream body; and
- it adds a `//go:linkname` to a Gomad hook implementation.

All calls then reach the replacement because the original definition itself
has been replaced. The same translator also performs much broader language
rewrites for goroutines, channels, maps, globals, imports, and types. Reusing
that pipeline would bring back the translated-tree and source-maintenance costs
that Gomad v3 intentionally avoids.

## Proposed seam: typed function definitions

Add one compiler module with this external interface:

```go
func Apply(target *ir.Package)
```

`cmd/compile/internal/gc` calls `Apply` immediately after
`noder.LoadPackage` and before devirtualization and inlining. The module owns:

- the Go 1.26.4 interception manifest;
- target and hook lookup;
- signature validation;
- typed-IR prologue synthesis; and
- diagnostics proving that every declared interception was applied exactly
  once.

This is a deep module: the compiler pipeline learns one call, while all
version-specific matching, validation, and IR construction remain local to the
module and are tested through `Apply`.

The likely source layout is:

```text
overlay/src/cmd/compile/internal/gomadintercept/
  intercept.go
  spec_go126.go
overlay/src/os/gomad.go
overlay/src/net/gomad.go
go1.26.4.patch
```

The patch retains a small import-and-call change in
`cmd/compile/internal/gc/main.go`. The new compiler module and hook
implementations are additive overlay files. The overlay validator must
explicitly allow the new compiler package without weakening its existing
runtime and standard-library allowlists.

## Interception manifest

Each entry identifies one resolved definition and one same-package hook:

```go
type spec struct {
    PackagePath string
    Receiver    string
    Function    string
    Hook        string
}
```

Conceptually:

```go
{
    PackagePath: "os",
    Receiver:    "*os.File",
    Function:    "Read",
    Hook:        "gomadInterceptFileRead",
}
```

The receiver should use the compiler's canonical type identity rather than a
source-formatted string internally. The string form above is only the stable,
reviewable declaration format.

The manifest is compiled into the version-pinned compiler. It is therefore
included in the compiler and toolchain build hashes automatically. A Go upgrade
must define a new manifest rather than silently reusing the Go 1.26.4 list.

## Hook interface

A hook receives the original receiver, if any, followed by the original
parameters. It returns the original results followed by a final `handled bool`.

For example:

```go
func gomadInterceptFileRead(
    file *File,
    destination []byte,
) (n int, err error, handled bool)
```

The compiler transforms the start of the original definition as if it
contained:

```go
if n, err, handled := gomadInterceptFileRead(f, b); handled {
    return n, err
}
```

The untouched upstream body follows. A no-result function's hook returns only
`handled bool`; a one-result function returns that result plus `handled`.

The compiler rejects the package when:

- the target is absent or appears more than once;
- the hook is absent;
- receiver, parameter, variadic, or result types do not match exactly;
- the final hook result is not `bool`;
- the target is generic, bodyless, or assembly-defined; or
- a manifest entry was not applied.

Hooks execute before the upstream body. They must return `handled=false` when
Gomad is disabled or when the object/path is outside the modeled profile. If a
current patch hook occurs after upstream validation, the hook must preserve
that observable behavior itself before returning `handled=true`. Same-package
hooks can call unexported validation helpers such as `checkValid`.

The first version should not support arbitrary insertion points. A target that
cannot preserve behavior from a function-entry hook remains a surgical patch.
This stop rule prevents the compiler module from becoming a second general AST
rewriter.

## Why definition interception covers every caller

The original function or method retains its name, linker symbol, method set,
reflection metadata, and ABI. Only its body gains a typed prologue.

- Direct callers reach the instrumented definition.
- Interface itabs still point at that method or its normal compiler-generated
  wrapper, which reaches the instrumented definition.
- Function and method values retain their normal closure wrappers.
- Same-package calls behave identically.
- If the compiler inlines the function, it inlines the interception prologue
  with the upstream body.

No caller needs to know Gomad exists. There is also no recursion problem: the
hook decides whether it handled the operation and returns to the inserted
prologue; it does not replace or call the original symbol.

Do not mark intercepted functions `noinline` by default. Inlining should
preserve the prologue and avoids changing normal optimization policy. Tests
must prove this with inlining enabled and disabled. A target-specific
`noinline` fallback is acceptable only if a demonstrated compiler issue makes
it necessary.

## Expected patch reduction

The initial candidates are entry-point branches currently spread across:

- `src/net/net.go`;
- `src/net/tcpsock.go`;
- `src/net/tcpsock_unix.go`;
- `src/net/dial.go`;
- `src/os/sys.go`;
- `src/os/file.go`;
- `src/os/file_unix.go`;
- `src/os/file_posix.go`;
- `src/os/dir.go`;
- `src/os/stat_unix.go`;
- `src/os/path.go`; and
- `src/os/stat.go`.

These hooks mostly have the required shape: ask a same-package Gomad helper to
handle the operation, otherwise continue into the upstream implementation.

The following should remain outside the first migration:

- linker detection of external linking;
- runtime initialization and scheduler changes;
- runtime random seeding and run-queue selection;
- virtual clock and timer-heap changes;
- testing lifecycle completion hooks; and
- any standard-library change requiring mid-body mutation or multiple exit
  hooks.

`crypto/rand.Reader` is a variable rather than a call target. It is better
handled by an additive `crypto/rand` init file, if safe, than by compiler
interception.

## Alternatives considered

### 1. Linker or object relocation rewriting

Rewrite calls from an original symbol to a Gomad symbol after compilation.

Rejected as the primary design. It must account for interface itabs, method and
function values, method wrappers, ABI wrappers, reflection metadata, assembly
references, and calls already removed by inlining. Disabling inlining globally
would reduce one problem while introducing a broad performance and behavior
change. The resulting linker module would have a much larger effective
interface than the proposed definition pass.

### 2. External `-toolexec` wrapper

Use the stock `go` command to wrap compiler and linker invocations.

Insufficient on its own. `-toolexec` can observe and replace tool invocations,
but the stock compiler exposes no flag for function-definition interception.
The wrapper would have to rewrite Go source or version-specific object data,
recreating one of the rejected approaches. `-toolexec` could configure a
patched compiler later, but it is not the interception mechanism.

### 3. Lower syscall or virtual-descriptor seam

Intercept `internal/poll`, `syscall`, or raw kernel calls so ordinary `os` and
`net` methods work unchanged.

Rejected for this scope. Gomad's current files and connections are Go objects
without real pollable descriptors. A lower seam would require a synthetic file
descriptor namespace, pointer-safe syscall argument translation, runtime
netpoll integration, platform-specific behavior, and comprehensive fail-closed
coverage for raw syscalls. That is a virtual kernel, not a simplification of
the existing profile.

### 4. Gomad v2-style AST translation

Technically capable, but rejected because it parses and materializes nearly all
source separately, changes package identities, and carries a large language
translation surface. Gomad v3 only needs a small set of definition hooks.

### 5. Whole-file replacements

Easier to edit than patches but still fork upstream source and obscure the
actual delta. The compiler pass should be prototyped before adopting any
whole-file replacement for an eligible entry-point file.

## Implementation plan

### Phase 0: establish a trustworthy baseline

1. Regenerate `go1.26.4.patch` from the exact pinned Go source so it is valid
   and fully applicable.
2. Make patch syntax and applicability part of `make -C tools/gomadv3
   validate` or the earliest build stage.
3. Run the existing disabled-mode, deterministic runtime, I/O profile, and
   upstream toolchain tests to capture the baseline.

The current inspected patch cannot serve as a comparison baseline because its
final `src/os/file.go` hunk does not apply.

### Phase 1: build the compiler module against synthetic targets

1. Add `cmd/compile/internal/gomadintercept` as an additive overlay module.
2. Add the minimal `gc/main.go` import and `Apply(typecheck.Target)` call after
   `noder.LoadPackage` and before devirtualization/inlining.
3. Implement exact target and hook lookup.
4. Implement signature validation and fail-closed diagnostics.
5. Synthesize entry prologues in typed IR.
6. Add compiler testdata containing a top-level function and a method with
   matching hooks.

Synthetic tests must cover:

- direct function calls;
- concrete method calls;
- interface method calls;
- method values and method expressions;
- same-package and cross-package calls;
- handled and fallthrough outcomes;
- nil receivers;
- zero, one, and multiple result values;
- inlining enabled and disabled; and
- a missing target, missing hook, duplicate target, and every signature
  mismatch category.

Do not migrate production hooks until these invocation forms pass. The purpose
of this phase is to prove the seam, not merely the easiest direct-call path.

### Phase 2: migrate one real vertical slice

Migrate the smallest coherent file-operation path that demonstrates a method
through an interface. A read-only file slice will likely require these targets
together:

- `os.OpenFile` or the narrowest existing creation entry point;
- `os.(*File).Read`; and
- `os.(*File).Close`.

Adapt the existing same-package helpers to the standardized hook interface.
Remove only the corresponding patch hunks. Test:

- direct `File.Read`;
- `io.Reader.Read` through an interface;
- `io.ReadAll` or an equivalent normal standard-library consumer;
- a captured `file.Read` method value;
- mounted and unmounted paths;
- closed and nil file behavior;
- record and replay;
- malformed broker responses and fail-closed behavior; and
- disabled-mode parity with stock Go.

Compare the materialized toolchain source against the baseline to verify that
only the intended hunk deletions and injected compiler behavior changed.

### Phase 3: migrate the remaining filesystem hooks

Move entry-compatible hooks incrementally, grouped by behavior:

1. stat and directory reads;
2. seek and file metadata;
3. path mutations and read-only rejection; and
4. hostname and directory creation.

For each group:

- add manifest entries;
- adapt same-package hook signatures;
- add best-case and failure-mode tests;
- remove the old patch hunks;
- run disabled-mode compatibility; and
- inspect the remaining patch before continuing.

Leave any hook patched when moving it to function entry would duplicate a
substantial upstream implementation or weaken its validation/error ordering.

### Phase 4: migrate network hooks

Migrate network behavior in coherent groups:

1. dial/listen construction and address parsing;
2. connection read/write/close/deadlines;
3. listener accept/close/deadline/address; and
4. TCP options and unsupported raw-connection/file escape hatches.

Tests must exercise calls both through concrete `*net.TCPConn` values and the
`net.Conn`/`net.Listener` interfaces. Include method values, HTTP or another
normal standard-library consumer, deadline failures, closed connections,
unsupported address families, and disabled-mode parity.

Network migration is complete only when no modeled connection can reach a host
file descriptor or raw syscall through an invocation form missed by the pass.

### Phase 5: harden the build and audit interface

1. Include the compiler module and manifest in overlay validation and cache-key
   tests.
2. Emit deterministic compiler diagnostics naming the unapplied spec.
3. Add a build-time report listing every applied interception.
4. Compare that report with the expected manifest and reject missing or extra
   entries.
5. Retain source-surface audit tests, but make them inspect the manifest and
   compiled behavior instead of searching only `go1.26.4.patch`.
6. Document the Go-upgrade procedure: new source checksum, new interception
   manifest, compiler tests, disabled-mode standard-library tests, then removal
   of obsolete version data.

### Phase 6: evaluate and stop

Measure the result against the goal:

- number of standard-library patch files and hunks removed;
- compiler module size and interface size;
- number of exceptional targets left patched;
- Go-upgrade conflicts;
- disabled-mode overhead; and
- clarity of reviewing one interception end to end.

Keep the design only if the compiler module has a small, uniform interface and
removes substantially more standard-library complexity than it introduces.
Do not expand it into arbitrary statement insertion, exit instrumentation, or
source rewriting to eliminate the final few patches.

## Error handling and failure modes

### Toolchain compilation

Missing targets, hooks, or signature mismatches are compiler errors. There is
no warning-and-continue mode. A partial manifest must never produce a toolchain.

### Runtime disabled path

Every hook returns `handled=false` without mutating state when its profile is
disabled. The upstream body then runs normally. Disabled-mode standard-library
tests guard this invariant.

### Runtime handled path

Once a hook returns `handled=true`, the compiler-generated prologue returns its
results and never falls through to host behavior. Existing invariant failures
remain fail-closed panics or explicit errors according to the current profile
contract.

### Crashes and partial builds

The immutable build-key and publication logic already prevent a failed
toolchain build from replacing the stable toolchain. Compiler-pass failure
occurs before publication. Manifest, hook source, compiler source, and the
remaining patch must all contribute to the same build identity.

### Go upgrades

An upstream rename or signature change causes target/hook validation to fail at
compile time. This is preferable to a whole-file replacement silently masking
new source. Changes to compiler IR may require adapting the single compiler
module and its synthetic tests.

### Optimization changes

The interception is inserted before inlining, but future compiler phases could
change assumptions about generated IR. Invocation-form integration tests must
run for every supported Go version with ordinary optimization, `-l`, and the
relevant architecture targets.

## Performance, scalability, complexity, and security

### Performance

Every intercepted invocation adds a hook call and handled branch unless the
compiler safely inlines them. Benchmark disabled and enabled `Read`, `Write`,
`Stat`, dial, and accept paths. Avoid global disabling of inlining. Some small
disabled-mode overhead is acceptable, but it must be measured rather than
assumed.

### Scalability

The interception lookup is compile-time and proportional to the small manifest.
Runtime overhead is O(1) per intercepted call. A 10x workload does not change
the architecture, though it magnifies existing locks in the Gomad file and
network maps; the compiler pass neither fixes nor worsens their asymptotic
behavior.

### Complexity

Complexity moves from dozens of fragile source hunks into one compiler module,
one declarative list, and same-package hook implementations. This improves
locality only while the hook interface stays uniform. Arbitrary insertion
points would erase that advantage and are explicitly out of scope.

### Security

Definition interception reduces the risk of missing interface or method-value
calls that could escape to host I/O. The remaining risks are a missing manifest
entry, an incorrect handled decision, or a stale compiler transformation. Exact
manifest accounting, signature validation, fail-closed runtime behavior, and
host-escape integration tests are required security gates.

## Verification commands

Determine the narrow Go compiler and Gomad package commands from the implemented
test layout, always including `-tags test_dep`. At minimum, completion requires:

```sh
make -C tools/gomadv3 validate
make -C tools/gomadv3 toolchain
make -C tools/gomadv3 runner-test
make -C tools/gomadv3 test
make fmt-imports
make lint-code
```

Run focused compiler and package tests before the full commands. Integration
tests add the `integration` tag only where they actually require integration
infrastructure.

## Success criteria

- Direct, interface, closure, method-value, and inlined invocations all reach
  the same hook.
- Disabled mode executes the untouched upstream body with stock behavior.
- A missing or incompatible interception prevents toolchain publication.
- The filesystem and network source hooks are removed from the patch without
  whole-file replacements.
- The compiler integration remains one small pipeline call plus an additive
  deep module.
- Runtime, linker, timer, and lifecycle changes that do not fit function-entry
  interception remain explicit surgical patches.
- A Go upgrade fails clearly at changed targets and requires one versioned
  manifest review.
