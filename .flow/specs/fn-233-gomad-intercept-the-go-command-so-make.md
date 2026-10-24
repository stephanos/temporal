# fn-233-gomad-intercept-the-go-command-so-make Gomad: intercept the go command so make targets and scripts run under WASM

## Conversation Evidence

> user: "it would be great if it makes use of it with Go a 1-liner."
> user: "maybe it should allow running a Go command's result with gomad? like `go test` and `go run`? it would need to do the overlay etc transparently"
> user: "it wou;dn't be able to replace the `go` command with its own forwarded in any context? like `gomad make <...>` which inokves a go command?"
> user: "yes file flow spec"

## Goal & Context

Running a Temporal test under Gomad today takes an `install` step with five absolute-path inputs and an `explore` command with a dozen flags before the `go-test` target. The build already applies the cooperative runtime overlay, the wasip1 target and the slim build tags transparently inside `explore`; what is not transparent is the operator grammar around it. The repository's own Makefile, its scripts and the test runner all invoke bare `go` from `PATH`, so a developer or CI job cannot point an existing target at Gomad without rewriting it.

This spec makes Gomad usable as a one-liner from Go's own command surface. `gomad -- <command> [args]` runs any child command with a `go` shim first on `PATH`; the shim is the Gomad binary itself. The `--` separator is mandatory and the only wrapper grammar, so that child commands and Gomad verbs can never collide, now or after new verbs are added. Inside the child, `go test` builds the guest with the pinned compiler, the overlay and the slim tags, and runs each test binary through Gomad's launcher using Go's standard `-exec` mechanism, so Go keeps ownership of package patterns, `-run`, `-v`, `-json`, PASS/FAIL parsing and the exit code. Every other `go` subcommand passes straight through to the real compiler with host settings. `gomad -- go test ./pkg` is the case where Go is the direct child, and `gomad -- make functional-test TEST_RACE_FLAG=off PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite` runs the Makefile unchanged with explicit WASM-compatible settings. The default race-enabled invocation is refused.

The installation manifest, the explicit verbs, the evidence lineage and qualification are unchanged and remain the rigorous path. The shim is a convenience front end that produces ordinary replayable artifacts and records in them that the arguments were intercepted rather than operator-written. Operators are the engineers who run tests locally and in CI; the product goal it serves is deterministic execution of arbitrary Temporal functional tests on the cooperative WASM profile.

This is Rust host work and lands after the fn-232 verbs it builds on (seed campaigns and install). No Go host-side implementation is started.

## Architecture & Data Models

**Wrapper.** `gomad [global flags] -- <command> [args...]`. Everything after the first `--` is the child command line and is never interpreted by Gomad, so a child named like a present or future verb is unambiguous. Gomad discovers and pins the real Go compiler on the caller's original `PATH`, then creates a private shim directory holding an executable named `go` that is the Gomad binary (hard link, copy or symlink as the platform allows; the shim dispatches on its own `argv[0]`), prepends that directory to `PATH`, exports the shim contract variables below, then resolves `<command>` on that modified `PATH` and executes the child with inherited stdio. Thus a direct child named `go` resolves to the shim; the real compiler remains available only through the pinned shim contract path. The child's exit status is the wrapper's exit status. The wrapper never parses or rewrites the child's arguments.

**Shim.** When the Gomad binary starts as `go`, it reads the shim contract from the environment and classifies the invocation:

- *Guest test*: `go test` without a caller-supplied `-exec`. The shim applies the guest build environment (pinned compiler, `GOOS=wasip1`, `GOARCH=wasm`, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, the overlay flag, the installation's build tags, and `-count=1` unless the caller set `-count`), inserts `-exec <gomad binary> run-guest` into the Go-driver flag segment before any -args/--args boundary, and executes the pinned compiler with the caller's remaining arguments unchanged. Go compiles one test binary per package and invokes the launcher for each.
- *Guest run or build*: `go run` or `go build` whose main package matches a `guest` pattern in the repository configuration. `run` is executed like a guest test with `-exec`. `build -o PATH` produces `PATH.wasm` plus a launcher script at `PATH` that executes `run-guest` on that module, following the shape of Go's own `wasm_exec` support.
- *Caller-owned execution*: a `go test` with a caller-supplied `-exec`, including the compile-only `-exec=true -count=0` idiom. The guest build environment is applied so the compile proves wasip1 buildability, but the caller's `-exec` is kept. Go invokes that caller-owned launcher; only a no-op launcher such as true supplies compile-only behavior. Gomad does not claim intercepted execution evidence for an external launcher.
- *Host passthrough*: every other subcommand (`env`, `list`, `mod`, `generate`, `vet`, `tool`, `version`, `install`, and `run`/`build` of unmatched packages) executes the pinned compiler with the caller's environment and arguments unchanged, so generators, linters and host tools keep producing host binaries.

**Launcher.** `gomad run-guest MODULE [guest args...]` is a public verb that Go's `-exec` calls once per test binary. It resolves the installation (explicit, environment, auto-derived), runs the module under the cooperative profile for the configured seeds sequentially, forwards the guest's stdout verbatim so Go's test driver can parse it, writes Gomad diagnostics to stderr, retains the executions in the artifact root as an ordinary campaign, and stops at the first non-successful seed, returning that guest failure status or the existing infrastructure status for capacity, fuel or helper failure. It returns zero only when every requested seed succeeds; skipped remaining seeds are recorded as unexecuted, not successful. On a failure with a retained replayable artifact, the final stderr line is the exact `gomad --installation FILE replay ARTIFACT` command. If preparation or infrastructure failure produced no replayable artifact, report that explicitly and print no fabricated replay command.

**Repository configuration.** A `gomad.toml` file at the Go module root, read by the shim and the wrapper, declares the per-repository facts that today live in `install` flags:

```toml
runtime_root = "tools/gomad/internal/model/runtimesource"
derive = "temporal"          # or "default"
guest = ["./cmd/server"]     # go run/build main packages executed as guests
```

`runtime_root` is module-relative. `derive = "temporal"` applies the existing fn-220 Temporal derivation (schema capture, `/workspace/<package>` working directory and `PWD`, writable roots, capacities, slim build tags) with the test package taken from the package Go is building. `derive = "default"` applies the stock cooperative namespace. `guest` is a list of module-relative main-package paths; absent means every `run` and `build` is host passthrough. No other keys are read; an unknown key is refused.

**Auto-derived installation.** When no `--installation` flag or `GOMAD_WASM_INSTALLATION` is set, the shim derives a manifest for the compiler found on the caller's original `PATH`, the configured runtime root and the configured derivation, writes it under the Gomad cache directory keyed by compiler/runtime identities, derivation, package, the complete derived configuration digest (including captured schema bytes), and required linked Gomad/helper implementation identities, and reuses it on later runs. Derivation goes through the same code `install` uses, never downloads, and refuses the same inputs `install` refuses. An explicit installation always wins and is never rewritten.

**Shim contract variables.** Set by the wrapper, read by the shim and the launcher; they are the only channel between them because arbitrary programs sit in between:

```
GOMAD_SHIM_BINARY=<absolute path of the gomad binary>
GOMAD_SHIM_GO=<absolute path of the compiler found on the original PATH>
GOMAD_SHIM_INSTALLATION=<absolute manifest path, or empty for auto-derivation>
GOMAD_SHIM_ARTIFACTS=<absolute artifact root>
GOMAD_SHIM_SEEDS=<seed count or comma-separated seed list>
GOMAD_SHIM_RUN=<wrapper run identity shared by every launcher in the child tree>
GOMAD_SHIM_INVOCATION=<absolute immutable per-go-invocation metadata path>
GOMAD_SHIM_INVOCATION_SHA256=<digest of those exact metadata bytes>
```

**Evidence.** Each launcher invocation is one campaign in the existing Rust lineage. Its campaign record carries `origin: intercepted-go` with the intercepted `go` argv and the wrapper run identity; operator-written campaigns carry `origin: operator`. Artifacts replay, inspect and minimize exactly like operator artifacts. Qualification admits only `origin: operator` evidence.

### Invocation, cache and streaming contracts

Each shim call creates a unique invocation identity beneath the wrapper identity and an immutable mode-0600 metadata file in its private mode-0700 run directory. The file retains the exact original go argv (encode OS argument bytes losslessly), wrapper/invocation identities, pinned compiler, installation/configuration identities, build tags/overlay and preparation bindings. Pass its path and digest only to that go subprocess through the invocation contract variables; all package launchers from that invocation inherit them. Launchers authenticate the metadata, retain the provenance with their campaign before temporary files disappear, and never reconstruct original argv from rewritten test arguments. Concurrent go calls get distinct files/identities. The wrapper cleans only its own completed invocation files after children exit, never another live wrapper's directory.

Before auto-cache lookup, derive and hash current captured configuration inputs, including Temporal schema bytes, using the same install path. Missing required schema inputs refuse even if a previous cache entry exists. Bind entries to the current linked host/helper identities as well as compiler/runtime/configuration/package inputs, and authenticate them before reuse. Schema mutation or a changed executable must yield a fresh matching derivation or an explicit refusal, never stale data; invalid explicitly selected installations still never fall back. Add cache tests for unchanged reuse, schema mutation, absent schema after caching, and changed linked implementation.

Go argument classification follows the pinned driver's parsing boundary. Inject Go flags after the subcommand and before any -args/--args terminator, preserving the relative order and exact bytes of all caller arguments; preserve caller values according to the stated policy instead of appending conflicting duplicates. Arguments after that boundary are guest arguments, even when they look like -exec/-tags/-count. Validate effective driver flags from argv and GOFLAGS without interpreting guest arguments as build options. Test both terminator spellings, misleading guest flags and caller-supplied launchers.

Task .1 extends linked execution with request-local bounded stdout/stderr callbacks so bytes can reach Go's driver before guest termination while the same bytes are retained in evidence. The existing return-at-end Sink alone cannot satisfy this contract. Keep callbacks isolated between concurrent executions, preserve each stream's order, and honor output limits and cancellation/backpressure without unbounded queues or silent drops. An output-consumer failure is explicit and retains available partial evidence. A fixture writes a marker, waits for a controller action, then exits; the controller must observe the marker before releasing it, and final retained stream bytes must match the forwarded bytes.

## API Contracts

- `gomad [--installation FILE] [--seeds N|LIST] [--artifacts DIR] -- <command> [args...]` runs `<command>` under the shim. Exit status is the child's. Errors: a bare `gomad make ...` without `--` is the existing unknown-verb error and suggests the `--` form; an empty command after `--` is the existing usage error; a `<command>` not found on `PATH` exits with the existing input status and names it; the shim directory cannot be created exits with the existing filesystem status.
- `gomad run-guest MODULE [args...]` as specified above. `--json` is a standalone reporting mode and must not be enabled by the Go -exec shim: launcher stdout in intercepted execution contains guest bytes only. Standalone --json writes structured outcome rows to stdout and keeps guest output in the retained artifacts; it does not forward guest bytes to that channel. Intercepted execution always uses the default guest-only stdout mode. Errors: a module not prepared by this binary's installation is refused with the existing identity status; capacity, fuel and helper failures exit with the existing infrastructure status and print a termination diagnostic; append the exact replay command as the final line only when a replayable artifact was retained, otherwise explicitly report that none exists.
- Refusals inside a guest invocation, each with a one-line stderr message and the existing unsupported-option status, before anything is built: `-race`; `-fuzz`; `-cover`, `-coverprofile` and `-coverpkg`; `-msan` and `-asan`; `CGO_ENABLED=1` in the caller's environment; a `-tags` value naming a tag the installation excludes; `GOOS`/`GOARCH` set to anything other than unset or `wasip1`/`wasm`.
- Host passthrough is byte-exact: arguments, environment and the compiler path, so `go env GOOS` reports the host.
- `gomad.toml` is refused with the existing input status on unknown keys, a non-module-relative `runtime_root`, an absolute or escaping `guest` entry, or a `derive` value other than `temporal` or `default`.
- The explicit verbs, flags, status codes, environment variables and manifest format of fn-232 are unchanged by this spec.

## Edge Cases & Constraints

- Go's test cache keys on binary, arguments and environment; the shim forces `-count=1` unless the caller set `-count`, so a seeded run is never satisfied from cache.
- Intermediate programs that locate the compiler on `PATH` (make, shell scripts, gotestsum, the repository test runner) are intercepted; programs that hardcode an absolute compiler path are not, and the documentation says so.
- The `-exec` launcher is invoked once per package binary with Go's own `-p` parallelism; launchers in one wrapper run share the artifact root and run identity and never share preparation or compile caches unsafely, using the existing lock discipline.
- The compiler on the caller's `PATH` must be the pinned stock version; another version is refused by the existing installation preflight, never adapted.
- `GOTOOLCHAIN` switching is disabled inside guest builds exactly as the Runner disables it today.
- Guest output is forwarded unbuffered and unmodified; Gomad never writes to the guest's stdout.
- The shim directory is private to the wrapper process, created with mode 0700 under the Gomad cache, and removed on exit; a stale directory from an interrupted run is removed on the next wrapper start.
- Nothing is downloaded; nothing inherits `GOMAD_*` variables other than the shim contract and the documented installation variable.
- No native path, no fallback, no new guest capability.

## Acceptance Criteria

- **R1:** `gomad -- <command>` runs the child with a `go` shim first on `PATH`, returns the child's exit status, and leaves non-Go children untouched; no verb name forwards to a child. Errors: missing `--`, empty command, unknown command and shim-directory failure as in API Contracts; a child named like a verb (for example `gomad -- replay`) is still run as a child.
- **R2:** Inside the child, `go test` without caller `-exec` builds each package with the pinned compiler, overlay, wasip1 target and installation tags, runs each binary through `run-guest`, and reports PASS/FAIL and exit status through Go unchanged, including `-run`, `-v`, `-json` and multi-package patterns. Errors: the refusal list in API Contracts, each before any build; a caller `-count` is preserved.
- **R3:** Every non-guest `go` subcommand, and `run`/`build` of packages not listed in `guest`, pass through byte-exact to the pinned compiler with host settings. Errors: no error surface beyond the compiler's own.
- **R4:** `run-guest` retains each seed's execution as a campaign with `origin: intercepted-go`, the intercepted argv and the wrapper run identity; the artifact replays exactly under the explicit `replay` verb and is accepted by `minimize` and inspection; qualification refuses it. Errors: unprepared module and infrastructure terminations as in API Contracts; the replay command is the last stderr line on failure with a retained replayable artifact; failures without one explicitly say so.
- **R5:** With `gomad.toml` declaring `derive = "temporal"` and no explicit installation, `gomad -- go test ./tests/gomadfunctional -run '^TestFrontendSystemInfo$'` from a clean checkout with the pinned compiler passes and retains evidence, and the derived manifest has the same configuration bytes as `install --temporal-checkout/--temporal-package` for the same inputs. Errors: configuration refusals as in API Contracts; a missing compiler or schema source is refused with the existing installation status.
- **R6:** `gomad -- make functional-test TEST_RACE_FLAG=off PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite` in the Temporal checkout, with the Makefile unchanged, routes selected functional tests through the launcher and exits with make's status. Scope the positive smoke run to the registered supported selection; broad-suite refusals/failures stay classified and are not a suite support claim. Test repository generator/linter host passthrough separately: this Make target directly calls go test and does not run those tools. Errors: `-race` requested through the Makefile's default `TEST_RACE_FLAG=on` is refused with the one-line message rather than silently dropped.
- **R7:** The operator guide documents the wrapper, the shim classification, the refusal list, `gomad.toml`, the interception limits and the evidence-origin rule, and states that intercepted evidence is not qualification evidence. Errors: a support claim for tests beyond those the sweep and one-box specs have proven is a finding.
- **R8:** Golden CLI corpus cases cover each classification branch, each refusal, the auto-derived installation reuse and the `origin` field, under the fn-232 per-verb projection discipline. Errors: a golden or raw-integrity difference fails with its path.

## Boundaries

- No change to the explicit verbs, their flags, statuses, environment variables or manifest version.
- No coverage extraction from the guest filesystem; `-cover` is refused in this spec.
- No embedding of the runtime overlay sources into the binary; `runtime_root` comes from `gomad.toml` or an explicit installation.
- No support claim for any functional test beyond the one-box and sweep specs; the wrapper changes how a test is invoked, not what the guest can do.
- No interception of absolute compiler paths, `GOROOT`-relative invocations or other toolchains.
- No Go host-side implementation; this is Rust work after fn-232.7 and fn-232.11.
- No CI run, PR or push.

## Decision Context

The owner asked for a Go one-liner and then for interception in any context, naming `gomad make`. Go's `-exec` flag exists for exactly this: it hands a cross-compiled test binary to a launcher while Go keeps package resolution, flags, output parsing and the exit code. Putting the shim on `PATH` rather than adding a `gomad go` verb alone means Makefiles, scripts and wrapper tools need no edits; the Temporal Makefile calls bare `go` throughout. A busybox-style `argv[0]` dispatch keeps one binary and one identity. The owner asked how child commands and future verbs stay apart; the answer is the mandatory `--` separator, the standard end-of-options convention, so the verb namespace can grow freely. The owner then preferred `gomad -- go` over a reserved `gomad go` alias for consistency, so there is exactly one wrapper spelling.

Rejected: resolving `gomad make` by verb precedence (a future `make` or `test` verb would silently change what a Makefile does); a named wrapper verb such as `gomad exec make` (`exec` is already a refused target kind, and `--` needs no new word); a reserved `gomad go` alias (two spellings for one thing); a `GOFLAGS`/`-exec` recipe in the docs (every caller repeats the pinning and nobody records provenance); patching the Makefile to call `gomad explore` (one repository only, and every target rewritten); teaching `explore` to accept Go's flag spelling (duplicates Go's test driver); treating `go run` as guest by default (the repository's linters, generators and test runner run through `go run` and must stay host, so guest is opt-in by package); embedding the overlay sources in the binary (removes one config key at the cost of a new build step and identity rule; revisit if a second Go module adopts Gomad). Intercepted campaigns are marked and excluded from qualification so that convenience never dilutes the evidence a support claim rests on.

## Early proof point

Task .1 uses fn-232.11's pinned compiler and shared preparation API to obtain the cooperative overlay, full installation build tags (including test_dep), guest configuration and module-admission bindings. Retain those generated inputs and the exact compiler invocation with `-exec 'gomad run-guest'`, the installation selected explicitly, and the TestFrontendSystemInfo selector. Bare GOOS/GOARCH alone is not sufficient preparation. The launcher must admit the bytes Go actually passes through -exec using that same preparation contract and retain them before Go removes its temporary binary. Prove Go reports PASS and the retained artifact replays after Go's temporary files and build inputs are removed. If Go's test driver cannot parse the forwarded output or the launcher cannot share one artifact root across `-p` parallel packages, stop and report before building the shim.

## Quick commands

```sh
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad run_guest
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad shim
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad wrapper
# after the verbs exist:
gomad --installation /absolute/operator/temporal.json -- go test ./tests/gomadfunctional -run '^TestFrontendSystemInfo$' -v
gomad -- make functional-test TEST_RACE_FLAG=off TEST_SHUFFLE_FLAG=off PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite TEST_ARGS='-run ^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$'
tools/gomad/scripts/behavioral-parity --help
# Extend the parity/golden driver with the new run-guest lane in task .1; use its documented explicit binary inputs.
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Wrapper runs any child after `--` with the go shim on PATH; no alias. | .3 | — |
| R2 | Guest test classification and refusals through Go's -exec. | .2 | — |
| R3 | Byte-exact host passthrough. | .2 | — |
| R4 | Launcher evidence with intercepted origin; replay, minimize, qualification refusal. | .1 | — |
| R5 | gomad.toml and auto-derived installation reproduce the Temporal derivation bytes. | .3 | — |
| R6 | Temporal Makefile target runs unchanged through the wrapper. | .4 | — |
| R7 | Operator documentation. | .4 | — |
| R8 | Golden corpus coverage per branch and refusal. | .1, .2, .3 | — |

## Amendments (2026-10-24, conductor)

1. **Status.** The `--` wrapper dispatch (`main.rs`), the shim classifier (`shim/classify.rs`, `shim/build.rs`), `run-guest` (`cli/run_guest.rs`) and `gomad.toml` at the Temporal module root already exist on `gomad`. Task .1 is in progress; .2 to .4 remain, and the task boundaries above stand as the acceptance map even where code landed early.
2. **Formal rerun after the cycle.** The launcher, shim and Make lanes run in the fn-232.13 dry run; the formal fn-233 rerun (about one hour) starts on `gomad` as soon as that cycle is green (MILESTONES ruling 9 and the opportunistic-merge decision). Load-only timeouts are classified as contention and rechecked in the quiet-host window.
3. **R6 shape.** Temporal's `functional-test` target runs three roots (`./tests`, NDC, XDC) with `COMPILED_TEST_ARGS`, which carries `-race` when `TEST_RACE_FLAG=on` and `-shuffle on` when `TEST_SHUFFLE_FLAG=on`; `CGO_ENABLED` defaults to 0. The positive smoke run therefore passes `TEST_RACE_FLAG=off`, `TEST_SHUFFLE_FLAG=off` and a `TEST_ARGS='-run …'` selector for the registered supported selection, as in Quick commands. The NDC and XDC roots are expected to be refused or to fail classification; that is recorded, not a support claim.
4. **`gomad.toml` today.** It declares `runtime_root` and `derive = "temporal"` only; `guest` is absent, so every `go run`/`go build` is host passthrough. This is the intended default for the Temporal checkout.
