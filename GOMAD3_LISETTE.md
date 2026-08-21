# Gomad v3 Lisette Rewrite Strategy

**Strategy date:** 2026-08-14

## Decision

Rewrite Gomad v3 in Lisette except where Go source is required to integrate with
the Go toolchain, generated targets, operating-system facilities, or existing
Go callers. The rewrite must preserve Gomad's observable behavior. Internal
package boundaries, data representations, and algorithms may change when doing
so produces shorter code, better diagnostics, or deeper modules.

This is a source-language migration, not a redesign of Gomad's deterministic
contract. It must not weaken fail-closed capability review, runtime choice
control, deterministic I/O, artifact identity, crash consistency, replay, or
qualification evidence.

The current candidate compiler is Lisette 0.11.2. Lisette is pre-1.0 and may
make breaking changes, so implementation must pin an exact vetted release and
Rust toolchain rather than following the latest release. Requiring Rust for a
normal Gomad build is acceptable.

## Goals

The rewrite should:

- reduce maintained source code, measured as authored Lisette plus handwritten
  Go rather than generated Go;
- replace stringly typed states and dispersed error checks with algebraic data
  types, exhaustive matches, `Option`, and `Result`;
- make invalid states harder to represent and ignored failures harder to write;
- concentrate behavior in deep modules with small interfaces;
- preserve the existing CLI, Go integration surface, wire formats, artifact
  schemas, exit classifications, deterministic behavior, and qualification
  claims; and
- retain the existing comments when moving or rewriting their associated code.

Generated Go size, compile time, binary size, allocations, and runtime are
tracked separately. Generated code does not count toward the source-reduction
goal, but it remains trusted build input and must be reproducible and auditable.

## Non-goals

- Changing Gomad's commands, flags, defaults, output contract, or exit statuses.
- Redefining runtime choices, seed behavior, logical time, deterministic I/O,
  World semantics, or supported target boundaries.
- Changing an artifact or wire schema merely because another representation is
  more natural in Lisette.
- Replacing the patched Go runtime, compiler hooks, standard-library overlays,
  or Go code generated into the pinned toolchain.
- Keeping permanent Go and Lisette implementations of the same behavior.
- Maximizing the percentage of Lisette at the cost of larger code, worse
  diagnostics, an unstable interface, or a new runtime indirection.

## Behavioral compatibility contract

The baseline is a frozen Gomad commit and its complete test and qualification
evidence. Freeze that baseline only after current in-flight Gomad work is
stable. Every migrated slice is compared against the same baseline before its
old implementation is removed.

The compatibility contract includes:

| Surface | Required compatibility |
| --- | --- |
| CLI | Commands, argv handling, flags, defaults, status codes, stdout/stderr routing, JSON event schemas, and routine human output remain compatible. |
| Runtime | The same unchanged target, toolchain, inputs, and seed produces the same controlled decisions and semantic outcome. |
| Go integration | Existing Go import paths and exported integration interfaces, especially `world` and `world/child`, continue to compile and behave the same. |
| Wire protocols | Framing, field order, widths, bounds, canonical encodings, and rejection behavior remain byte-compatible. |
| Artifacts | Directory layout, permissions, schemas, canonical encoding, durability rules, hashes, validation, inspection, resume, and replay rules remain compatible. |
| Errors | Existing user-visible classifications and statuses remain compatible. Internal error representations may become richer. |
| Platform support | No new platform is claimed and no qualified platform is dropped as an incidental result of the rewrite. |
| Security | Empty target environments, path validation, symlink rejection, containment, resource bounds, and fail-closed behavior remain intact. |

Runner and compiler identities will necessarily change when their source and
build pipeline change. Differential artifact checks must allow only the
explicitly versioned identity fields and hashes derived from them to differ.
All identity-independent canonical payloads and wire bytes must match exactly.
Existing rules that require the exact original Runner for replay remain in
force; the rewrite must not pretend that a new binary is the old Runner.

## Success criteria

Proceed beyond the first production pilots only if all of the following hold:

1. The compatibility harness finds no unexplained observable difference.
2. The pilots materially reduce authored implementation code. A 25% reduction
   is the initial target, excluding tests, generated Go, and comments carried
   forward from the Go implementation.
3. State and error variants are exhaustively matched, and fallible operations
   cannot be silently discarded without an explicit suppression.
4. Lisette diagnostics identify the authored `.lis` source and are clearer than
   the corresponding Go compiler or test failure for representative mistakes.
5. Generated Go is reproducible from pinned inputs.
6. Runtime, allocation, build-time, and binary-size budgets set from the frozen
   baseline show no unexplained material regression.
7. A clean build and the full Gomad test and qualification gates pass without
   using an unpinned network dependency.

The line-count target is evidence, not an optimization game. Do not merge
unrelated modules, remove validation, compress names, or delete useful tests
and comments merely to reduce the count.

## Architecture

### Dependency direction

Use three logical tiers with one dependency direction:

```text
Go compatibility façades and Go target integration
                        |
                        v
         generated Go from Lisette domain modules
                        |
                        v
              Go native leaf adapters

Patched runtime/compiler/stdlib overlay and generated toolchain Go remain
independent, sharing only existing versioned wire contracts.
```

The Go compatibility façades preserve exported Go interfaces. Lisette owns the
domain behavior behind them. Native Go adapters expose facilities that Lisette
cannot or should not implement directly. Native adapters must not import
generated Lisette packages, and Lisette modules must not import compatibility
façades. This prevents dependency cycles and keeps policy out of Go adapters.

Lisette supports library projects that emit Go packages, but Gomad must verify
the exact generated interface and module layout before relying on that support.
The Phase 0 interop spike is a hard gate because Go-facing compatibility is more
important than language coverage.

### Code that remains Go

Go is retained when one of these criteria is demonstrated:

- the file is compiled into the pinned Go runtime, compiler, standard library,
  or source overlay;
- the file is generated Go consumed by those packages;
- the code must preserve an existing Go interface that Lisette's emitted
  interface cannot reproduce exactly;
- the code requires build constraints, `unsafe`, raw syscalls, exact descriptor
  inheritance, signals, process-group control, linker/compiler directives, or
  another facility not safely expressible through established Lisette interop;
- the code must be usable by an arbitrary target built without Lisette, such as
  bootstrap and child-side integration; or
- a measured Lisette implementation is longer, less clear, slower beyond its
  budget, or produces worse failures.

Expected Go areas include:

- `overlay/src/runtime`, `overlay/src/cmd/compile`, and the standard-library
  overlay packages;
- the generated Go halves of the I/O and runtime-choice protocols;
- low-level process launch, bootstrap, supervision, descriptor, signal, file
  lock, no-follow open, atomic replacement, and platform-specific adapters;
- a compatibility façade for public Go packages such as `world` and
  `world/child` if emitted Lisette cannot preserve their exact interfaces; and
- minimal build/bootstrap entry points required before Lisette output exists.

These are expected areas, not blanket exemptions. Each remaining handwritten
Go file must have a short, testable justification at the end of the migration.

### Code that moves to Lisette

Lisette should own all portable domain behavior, including:

- command parsing, validation, dispatch, result classification, and rendering;
- World state transitions, ordering policy, replay, snapshots, and recording;
- choice-frontier, coverage, guidance, and campaign state machines;
- canonical records, artifact state, retention decisions, and validation;
- qualification policy, capability results, support comparison, and reports;
- target and I/O policy above Go analysis and host adapters;
- replay and resume control flow above native process and durable-file adapters;
- installer, doctor, inspect, and upgrade policy; and
- generators whose output must be Go but whose orchestration and schema logic
  do not themselves require Go.

Do not mechanically recreate all existing Go packages. The rewrite is an
opportunity to deepen related modules while preserving their external
behavior. The intended Lisette modules are:

| Module | Owns | Native seam |
| --- | --- | --- |
| Application | CLI grammar, validation, rendering, and exit classification | Standard streams, environment, executable path |
| Campaign | Selection, scheduling policy, frontier, stop rules, resume state, and aggregate outcomes | Process executor and wall watchdog |
| World | Requests, events, ordering, logical time, replay, snapshots, and recording | Go compatibility façade and child transport |
| Evidence | Canonical records, identities, schema validation, and semantic projections | Cryptographic and serialization primitives where needed |
| Artifact store | Publication state machine, journals, retention, and recovery decisions | Durable filesystem operations and locks |
| Qualification | Capability decisions, expectations, comparisons, and report projections | Go package/compiler analysis |
| Tooling | Version, protocol, boundary, and upgrade generation policy | Go parser, formatter, compiler, and build driver |

Each module should expose a small interface expressed as typed requests and
results. Native adapters perform mechanisms, not policy: they must not decide
whether a target is supported, how a campaign stops, which artifact is
canonical, or how an error is rendered.

## Error strategy

Better errors have two meanings here:

1. development errors should be caught at the Lisette source with exhaustive
   matches, non-null types, explicit mutation, and mandatory `Result` handling;
2. runtime errors should carry structured context internally while preserving
   the current public classification, status, and output contract.

Use domain error enums rather than propagating arbitrary formatted strings.
The top-level model should distinguish at least:

- invalid input;
- unsupported target or platform;
- target failure;
- watchdog observation;
- replay divergence;
- deterministic capacity exhaustion;
- cancellation;
- native adapter or host infrastructure failure;
- publication or validation failure; and
- violated internal invariant.

Expected target failures and watchdog observations are outcomes, not generic
infrastructure errors. Adding a variant must make every relevant match fail to
compile until its status, event, human output, JSON output, artifact behavior,
and tests are updated.

Go adapters return structured operation, kind, and bounded context fields.
Lisette converts them into domain errors and owns user-facing formatting. Do
not parse Go error strings to recover meaning. When an operation has both a
primary failure and cleanup failures, preserve all causes in a bounded error
report; do not discard cleanup failures while translating across the seam.

Unknown enum values, malformed payloads, interop conversion failures, and
unrecognized native error kinds fail closed. Panics remain limited to genuine
internal invariant violations. The feasibility gate must also inspect panic and
stack-trace behavior: routine errors should point to `.lis` source, while
generated Go needed to diagnose an invariant failure must be retained with the
build or CI failure artifact.

## Build and repository integration

### Toolchain pinning

Pin and record:

- exact Lisette release or commit and distribution checksum;
- exact Rust toolchain used to build Lisette when a prebuilt compiler is not
  used;
- Lisette lockfile and Go prelude/runtime dependencies;
- exact bootstrap Go and pinned Gomad Go toolchain;
- Lisette source digest, handwritten Go source digest, emitted Go digest, and
  final Runner binary digest; and
- generator and schema input digests.

The current Lisette 0.11.2 and Rust 1.97 requirements are candidates, not a
floating dependency policy. Lisette upgrades require their own generated-code,
behavioral, performance, and qualification diff.

### Generated Go policy

Prefer generating Go into a private ignored staging directory rather than
committing it. The reviewed source of truth is Lisette plus handwritten Go,
schemas, templates, and pinned compiler inputs. A build should:

1. create a fresh private staging directory;
2. run the pinned Lisette frontend to emit Go;
3. assemble emitted packages, Go façades, and native adapters without creating
   a module cycle;
4. build with the same pinned Go toolchain and environment controls Gomad uses
   today;
5. record the emitted-tree digest in the Runner build identity; and
6. publish the binary atomically only after all generation and compilation
   succeeds.

An interrupted or failed generation must leave the prior Runner untouched. CI
should retain the emitted tree on failure for diagnosis. Two clean emissions
from identical pinned inputs must be byte-identical after excluding no fields;
if Lisette cannot satisfy that requirement, generated Go must be normalized or
checked in until the compiler is fixed.

Use `lis emit` followed by the pinned Go build rather than allowing an ambient
Go executable to choose the final binary. Clear Gomad activation variables and
set `GOWORK`, `GOTOOLCHAIN`, timezone, and CGO controls as explicitly as the
current Makefile does.

### Proposed source topology

The physical module layout is finalized by the Phase 0 spike. It must represent
these source groups without a dependency cycle:

```text
tools/gomadv3/
  lisette/             authored Lisette project and tests
  native/              Go leaf adapters available through inbound interop
  facade/              Go compatibility sources compiled after Lisette emit
  overlay/             patched toolchain Go sources
  protocol/            shared wire schemas and Go-output templates
  qualification/       unchanged compatibility corpora
  testdata/            unchanged Go target fixtures
  .toolchain/          ignored generated Go, caches, toolchain, and binaries
```

This is a logical layout, not permission for a preliminary mass move. Preserve
existing import paths and make targets. Move files only when their migrated
slice is ready and its compatibility test is in place.

### Developer commands

Integrate Lisette into the existing commands rather than creating a parallel
unsupported workflow. The resulting build should provide at least:

- a pinned Lisette bootstrap/check step;
- Lisette format, check, and test targets;
- Go adapter and compatibility-façade tests with `-tags test_dep`;
- the existing Gomad generation, validation, runner, overlay, World, runtime,
  builder, integration, and qualification gates; and
- repository `make fmt-imports` and `make lint-code` coverage for handwritten
  and emitted Go where applicable.

Lisette tests must pass Go flags needed by the project, including
`-tags test_dep`. The generated staging directory must not pollute repository
formatting, lint discovery, source archives, or ordinary Git status.

## Migration strategy

### Phase 0: feasibility and baselines

Do not start the production rewrite until a disposable spike proves:

1. A Lisette binary can call a local Go native adapter.
2. An existing Go package can call an emitted Lisette library through a thin
   façade without changing the public interface.
3. The final assembled module is acyclic and works with `internal` package
   visibility and existing import paths.
4. Lisette output builds with the exact Gomad Go version and flags.
5. JSON tags, numeric widths, named types, `Option`, `Result`, interfaces,
   channels, arrays, maps, and error values cross both seams without semantic
   drift.
6. Wire fixtures generated by Go and Lisette decode and re-encode identically,
   including invalid and capacity cases.
7. `lis check`, `lis test`, the race detector, Go tests, and repository lint can
   operate on the assembled tree with `-tags test_dep`.
8. Clean emissions are reproducible, and diagnostics and panic traces are
   supportable.

Record source size, generated size, clean and incremental build time, binary
size, allocations, and benchmark results. If the two-way source integration
requires callbacks, cyclic modules, reflection-heavy translation, or large
handwritten façades, stop and narrow the Go/Lisette seam before proceeding.

### Phase 1: compatibility harness and Lisette root

Build the differential harness before translating behavior. It must run the
frozen Go Runner and candidate Runner against the same fixtures and compare:

- exit code, stdout, stderr, and JSON event sequence;
- canonical wire payloads;
- artifact trees and manifests under the explicit identity-difference policy;
- inspect, replay, resume, and qualification projections;
- filesystem modes, symlink behavior, atomic publication, and crash residue;
  and
- same-seed equality and different-seed diversity.

Introduce a Lisette entry point that initially delegates every command to one
temporary legacy Go adapter. This establishes the final build, argv, standard
stream, and status-code path without changing command behavior. The adapter is
a migration scaffold with an explicit deletion condition, not a permanent
module.

### Phase 2: production pilots

Migrate two bounded modules after their current behavior is stable:

1. `supportcompare`, to exercise enums, canonical results, error
   classification, JSON/text projection, and pure differential testing;
2. `choicefrontier`, to exercise a nontrivial bounded state machine, replay
   validation, capacity errors, and performance-sensitive collections.

Replace each module as one implementation behind its existing behavioral
interface. Do not layer a Lisette implementation beneath a permanent Go
implementation with duplicated validation. Keep the old implementation only
long enough for differential testing, then delete it.

Review the success criteria after both pilots. Stop or revise the architecture
if the code is not materially shorter, errors are not better, generated code is
unmanageable, or performance crosses its predeclared budget.

### Phase 3: read-only vertical slices

Migrate commands whose primary effect is analysis and rendering:

1. `inspect`;
2. `doctor`;
3. `analyze`; and
4. `compare-support`.

For each command, move the handler and its portable dependency closure to
Lisette while leaving Go parsing, build-information, filesystem, and toolchain
operations behind narrow native adapters. This validates errors and public
output before migrating publication or process control.

### Phase 4: models, evidence, and public integration

Migrate the deep in-memory and canonical modules:

1. World transition, ordering, snapshot, recording, and replay behavior;
2. record validation, canonical encoding, identities, outcome projection, and
   World-record composition;
3. choice and I/O wire codecs on the host side; and
4. compatibility and qualification models.

Keep the public Go `world` surface behind a façade if emitted Lisette cannot
preserve it exactly. Cross-language conformance tests must exercise every
exported operation, error, snapshot, replay transition, and concurrent access
pattern. Go-side tests use `require`, including inside `Eventually` blocks.

### Phase 5: durable state and orchestration

Migrate the write paths only after the model and read paths are stable:

1. artifact publication and batch journal state;
2. guide and corpus state;
3. qualification execution and report publication;
4. replay and resume orchestration; and
5. runner campaign coordination and exploration.

Native adapters retain file-descriptor ownership, process launch, signal and
process-group behavior, locks, permission-sensitive operations, no-follow
opens, and atomic filesystem primitives. Lisette owns the state transitions,
ordering, validation, retry/stop policy, and error classification around those
operations.

Use kill-point and fault-injection tests at every durable transition. A crash
must leave either the prior complete state or explicit recoverable partial
state, never a complete-looking corrupt artifact. Preserve aggregate and
per-run capacity bounds before optimizing throughput.

### Phase 6: generators, build tooling, and residue review

Move portable generator and upgrade policy to Lisette while continuing to emit
the exact Go required by the toolchain. Retain Go parser, formatter, compiler,
archive, and source-analysis mechanisms as adapters where Lisette interop is
clearer than reimplementation.

Then inventory every remaining handwritten Go file. For each file, record the
Go-only criterion, the Lisette module it supports, and its conformance test.
Delete the temporary legacy adapter, obsolete package-forwarding layers,
duplicate tests that only inspect removed implementations, and unused generated
paths. Preserve black-box and interface-level tests.

## Verification strategy

### Per-module tests

- Lisette internal tests cover state transitions, exhaustive variants, bounds,
  malformed inputs, and failure modes through the module interface.
- Lisette external tests cover each public module as a caller sees it.
- Go adapter tests cover success, every error kind, partial I/O, short writes,
  cleanup failures, cancellation, deadlines, signals, descriptor inheritance,
  permissions, symlinks, and platform differences.
- Go façade tests compile representative existing callers and verify exact type,
  method, serialization, concurrency, and error behavior.
- Shared golden corpora verify Go/Lisette wire and canonical-data equivalence in
  both directions.

Tests should assert observable results through module interfaces. Once a slice
is replaced, delete unit tests that exist only to inspect the old
implementation; retain or rewrite the behavior they protect at the new module
interface.

### End-to-end gates

Run, with `-tags test_dep` wherever Go tests are involved:

1. Lisette format, check, and test;
2. Go native-adapter and façade tests, including race tests;
3. generated-output reproducibility and validation checks;
4. differential CLI, artifact, replay, resume, and qualification tests;
5. `make -C tools/gomadv3 test`;
6. `make gomadv3-integration-test`;
7. core and Temporal qualification contracts at their established cadence;
8. `make fmt-imports`; and
9. `make lint-code`.

The full gate must use an empty compiler/network cache at least in CI to prove
all dependencies are pinned and available through the supported bootstrap.

### Performance and 10x-load behavior

Benchmark at least:

- World registration, readiness, cancellation, quiescence, snapshot, and
  replay;
- choice trace and frontier encoding/decoding;
- record validation and canonical hashing;
- artifact publication and journal replay;
- guide selection and corpus validation;
- campaign coordination at current parallelism; and
- clean/incremental Lisette plus Go build time and final binary size.

Lisette ADTs may generate larger tagged values or more copying than the current
Go representation. Inspect allocation profiles and emitted Go in hot paths
rather than assuming source brevity implies runtime efficiency.

At 10x seeds, corpus entries, journal segments, or qualification workloads, the
rewrite must preserve streaming, capacity checks, deterministic ordering,
backpressure, and bounded memory. Build cost should be amortized once per
Runner, not once per seed. Generated-code size must not cause per-run process
startup or resident memory to grow without an explicit accepted budget.

## Failure modes and recovery

| Failure | Required response |
| --- | --- |
| Lisette compiler regression | Keep the pinned compiler, minimize a reproducer, and block the upgrade. Do not patch generated Go by hand. |
| Generated output changes unexpectedly | Fail reproducibility or generated-diff validation before building or publishing the Runner. |
| Generation or compilation crashes | Leave the prior Runner intact and retain private diagnostics; discard incomplete staging on the next build. |
| Interop cannot preserve a Go interface | Keep a thin Go façade or the complete module in Go; do not expose a breaking generated interface. |
| Differential behavior diverges | Stop that slice, reduce the case, and resolve the semantic difference before deleting Go. |
| Runtime or allocation regression | Inspect emitted Go, specialize the representation, or keep the hot implementation in Go with a measured justification. |
| New error variant is unhandled | Treat the compile failure as intended; update all status, rendering, artifact, and test projections. |
| Native adapter returns an unknown kind | Fail closed as infrastructure failure with bounded context. |
| Crash during artifact operation | Preserve existing atomicity, sync, recoverability, and validation rules through kill-point tests. |
| Lisette becomes unavailable upstream | Builds continue from the pinned, checksummed compiler source or artifact and Rust toolchain. |

Rollback is by reverting the affected migration slice and rebuilding from the
same frozen inputs. Do not ship a public runtime switch between Go and Lisette:
it would double the supported state space and enter Runner identity and replay
semantics. Temporary dual binaries are allowed only inside differential tests.

## Trade-offs and considerations

### Correctness and maintainability

Algebraic data types and exhaustive matching fit Gomad's many state machines
and error domains. They can eliminate sentinel values, nil checks, invalid
field combinations, and switch defaults that hide new states. The benefit is
largest in World, campaign, artifact, replay, and qualification logic.

The cost is a second language and compiler, generated-code debugging, and an
interop seam. Deep Lisette modules and narrow Go adapters are essential; a
file-by-file transliteration would keep the old complexity and add the new
toolchain cost.

### Performance and scalability

Lisette emits Go, so process scheduling and Gomad's target runtime remain Go.
There is no required RPC or foreign-function boundary at runtime. However,
emitted representations may allocate or copy differently, clean builds gain a
Rust frontend step, and generated helpers may increase binary size. Measure the
inner state machines and the complete campaign rather than relying on the
compilation target alone.

### Complexity

The permanent complexity should be one pinned compiler and a small number of
real Go seams. Avoid per-package adapters, reflection-based generic bridges,
parallel model types, callbacks across the seam, and long-lived forwarding
packages. If a seam needs nearly the entire old Go interface, the module is too
shallow or should remain Go.

### Security and supply chain

Lisette and Rust become trusted build dependencies. Pin checksums and locks,
retain license and provenance data, generate without ambient network access,
and include the compiler and emitted-source identities in release evidence.
Generated Go receives the same static analysis and review expectations as
other generated trusted code.

The compiler runs only while building Gomad. It must not run inside a target,
receive target credentials, weaken the empty target environment, or become part
of process containment. The rewrite does not turn Gomad into a sandbox and
must preserve the existing trusted-test threat model.

### Lisette maturity

Lisette 0.11.2 is pre-1.0, its dependency management is described as early
preview, and its interop surface is still evolving. Pinning limits accidental
change but transfers maintenance risk to Gomad if the compiler has a blocking
bug. The feasibility pilots and upgrade dossier are therefore release gates,
not optional confidence checks.

### Diagnostics

Lisette can improve compile-time diagnostics and test failures at authored
source, but generated Go may make panics, profiling, coverage, or debugger
stepping less direct. Verify `//line` or equivalent source mapping rather than
assuming it exists. Retain generated sources with CI and release diagnostics
when they are necessary to interpret a production failure.

## Random translation experiment

To test the premise without choosing Lisette-friendly examples, two files were
randomly sampled from handwritten, non-test Go files between 50 and 240 lines.
Files with build constraints, generated names, `unsafe`, raw syscalls,
`os/exec`, C imports, or linkname directives were excluded. The draw selected:

- [`internal/romount/config.go`](tools/gomadv3/internal/romount/config.go); and
- [`internal/safefile/open.go`](tools/gomadv3/internal/safefile/open.go).

The translations below are illustrative and are not production changes. They
were formatted, checked, and emitted successfully with Lisette 0.11.2. They
have not passed Gomad's behavioral or differential tests.

| Sample | Current Go | Lisette source | Emitted Go | Immediate result |
| --- | ---: | ---: | ---: | --- |
| `romount/config.go` | 60 lines, 56 nonblank | 86 lines, 79 nonblank | 102 lines | `Result` propagation is cleaner, but the direct translation is longer and still has an unstructured `error`. |
| `safefile/open.go` | 50 lines, 45 nonblank | 59 lines, 55 nonblank | 179 lines | Failure states become explicit, but the sample omits the required Go compatibility façade, so the real replacement is larger. |

### Sample 1: read-only mount configuration

```rust
import f "go:fmt"
import fp "go:path/filepath"
import host "go:os"
import s "go:strings"
import slash "go:path"

pub struct Mapping {
  pub source: string,
  pub target: string,
}

pub fn parse_mappings(values: Slice<string>, working_directory: string) -> Result<Slice<Mapping>, error> {
  let mut mappings = Slice.new<Mapping>().reserve(values.length())
  for value in values {
    let (source, target, found) = s.Cut(value, "=")
    if !found {
      return Err(f.Errorf(
        "read-only mount %q must use HOST_DIRECTORY=TARGET_DIRECTORY",
        value,
      ))
    }
    if source == "" {
      return Err(f.Errorf("read-only mount source is required"))
    }

    let target_path = slash.Clean(target)
    if invalid_target(target, target_path) {
      return Err(f.Errorf("invalid read-only mount target %q", target))
    }
    let target_path = s.TrimPrefix(target_path, "/")

    let source_path = if fp.IsAbs(source) {
      source
    } else {
      fp.Join(working_directory, source)
    }
    let source_path = fp
      .Abs(source_path)
      .map_err(|err| f.Errorf(
        "resolve read-only mount source %q: %w",
        source,
        err,
      ))?
    let info = host
      .Lstat(source_path)
      .map_err(|err| f.Errorf(
        "inspect read-only mount source %q: %w",
        source,
        err,
      ))?
    if !info.IsDir() || info.Mode() & host.ModeSymlink != 0 {
      return Err(f.Errorf(
        "read-only mount source %q is not a directory",
        source,
      ))
    }
    mappings = mappings.append(
      Mapping { source: fp.Clean(source_path), target: "/" + target_path },
    )
  }

  for left in 0..mappings.length() {
    for right in left + 1..mappings.length() {
      if overlaps(mappings[left].target, mappings[right].target) {
        return Err(f.Errorf(
          "read-only mount target %q overlaps %q",
          mappings[left].target,
          mappings[right].target,
        ))
      }
    }
  }
  Ok(mappings)
}

fn overlaps(left: string, right: string) -> bool {
  left == right || s.HasPrefix(left, right + "/") ||
  s.HasPrefix(right, left + "/")
}

fn invalid_target(target: string, cleaned: string) -> bool {
  target == "" || s.IndexByte(target, 0).is_some() || cleaned == "." ||
  cleaned == ".." ||
  cleaned == "/" ||
  s.HasPrefix(cleaned, "../")
}
```

The main improvement is the two `?` chains: every `filepath.Abs` and `os.Lstat`
failure is necessarily returned after its context is added. The translation is
not shorter, largely because Go interop names and the formatter expand the long
validation and formatting expressions. Introducing a `MappingError` enum would
improve exhaustiveness but increase the direct file-level line count further.

The likely useful rewrite is therefore not this file alone. A deeper mount
module could own parsing, normalization, overlap detection, capture policy, and
canonical persistence behind one typed interface, while a Go adapter performs
source inspection. That larger replacement is where duplicated validation and
error projection may disappear.

### Sample 2: verified safe-file opening

```rust
import "go:io/fs"
import "go:os"
import "go:path/filepath"

pub struct OpenedFile {
  pub file: Ref<os.File>,
  pub info: fs.FileInfo,
}

pub enum SafeOpenError {
  SymbolicLink(string),
  NotRegular(string),
  Native(error),
  Changed { name: string, cause: Option<error>, close: Option<error> },
  InvalidLinkCount { cause: error, close: Option<error> },
}

pub fn open_regular(
  path: string,
  lstat: fn(string) -> Result<fs.FileInfo, error>,
  open: fn(string) -> Result<Ref<os.File>, error>,
  validate_link_count: fn(fs.FileInfo) -> Result<(), error>,
) -> Result<OpenedFile, SafeOpenError> {
  let info = lstat(path).map_err(SafeOpenError.Native)?
  let name = filepath.Base(path)
  if info.Mode() & os.ModeSymlink != 0 {
    return Err(SafeOpenError.SymbolicLink(name))
  }
  if !info.Mode().IsRegular() { return Err(SafeOpenError.NotRegular(name)) }
  validate_link_count(info).map_err(|cause| SafeOpenError.InvalidLinkCount {
    cause,
    close: None,
  })?

  let file = open(path).map_err(SafeOpenError.Native)?
  let opened_info = match file.Stat() {
    Ok(opened_info) => opened_info,
    Err(cause) => {
      return Err(
        SafeOpenError.Changed {
          name,
          cause: Some(cause),
          close: file.Close().err(),
        },
      )
    },
  }
  if !os.SameFile(info, opened_info) || opened_info.Mode() != info.Mode() || opened_info.Size() != info.Size() {
    return Err(
      SafeOpenError.Changed { name, cause: None, close: file.Close().err() },
    )
  }
  if let Err(cause) = validate_link_count(opened_info) {
    return Err(
      SafeOpenError.InvalidLinkCount { cause, close: file.Close().err() },
    )
  }
  Ok(OpenedFile { file, info: opened_info })
}
```

This version makes a real error-model improvement. `Ref<os.File>` cannot be
nil, a successful result always contains both the file and its matching
metadata, and every validation, race, native, and cleanup failure is explicit.
Adding an error variant forces the façade's renderer and tests to handle it.

It is not a complete behavioral replacement. A Go façade would still need to
preserve `OpenPath`, nullable `OpenRoot` input rejection, `ErrSymbolicLink`,
`errors.Is` behavior, exact messages, `openNoFollow`, platform-specific link
counts, and aggregation order for primary and close errors. With that façade,
the rewrite is substantially larger than the current Go file. This file should
probably remain a native Go adapter unless a larger artifact-store module can
absorb the Lisette error model without duplicating the safe-open interface.

### Experiment conclusion

The random samples do not support a file-by-file rewrite. Neither became
shorter, and the security-sensitive sample needs more Go after translation.
They do support the strategy's deep-module approach: use Lisette where several
Go files and their error projections collapse behind one interface, and retain
small, exact host mechanisms in Go. Repeat this experiment on the two proposed
production pilots before committing to the full migration.

## Exit criteria

The rewrite is complete when:

- all portable Gomad behavior is owned by Lisette deep modules;
- every remaining handwritten Go file has a demonstrated integration,
  platform, performance, or compatibility reason;
- the temporary legacy adapter and duplicate implementations are gone;
- authored implementation code is materially smaller than the frozen Go
  baseline;
- error and state variants are structured and exhaustive;
- generated Go and the complete Runner build are reproducible from pinned
  inputs;
- public Go integration, CLI, artifacts, replay, and qualification behavior
  satisfy the compatibility harness;
- performance and 10x-load budgets are met; and
- the full Gomad and repository verification gates pass.

## References

- [Gomad v3 architecture](tools/gomadv3/ARCHITECTURE.md)
- [Gomad v3 current behavior](tools/gomadv3/README.md)
- [Gomad v3 next functionality](GOMAD3_NEXT.md)
- [Lisette repository](https://github.com/ivov/lisette)
- [Lisette Go interop](https://github.com/ivov/lisette/blob/lisette-v0.11.2/docs/reference/13-go-interop.md)
- [Lisette packages and emitted Go libraries](https://github.com/ivov/lisette/blob/lisette-v0.11.2/docs/reference/12-packages.md)
- [Lisette concurrency](https://github.com/ivov/lisette/blob/lisette-v0.11.2/docs/reference/14-concurrency.md)
- [Lisette testing](https://github.com/ivov/lisette/blob/lisette-v0.11.2/docs/reference/16-testing.md)
- [Lisette roadmap](https://github.com/ivov/lisette/blob/lisette-v0.11.2/docs/intro/roadmap.md)
- [Lisette 0.11.2 release](https://github.com/ivov/lisette/releases/tag/lisette-v0.11.2)
