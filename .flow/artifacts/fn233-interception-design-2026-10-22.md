# fn233 Go-command interception design

Engineers should be able to run the existing Go driver and Temporal Make targets through Gomad, retaining ordinary Rust replay artifacts with intercepted provenance. This design covers fn233.1 through .4 as one integration batch. Implementation remains gated on completed fn232.7 and fn232.11. Public installation is pending. The root fn232.10 candidate stays frozen during its measured window.

This artifact records design, not acceptance evidence or a second task tracker. The survey used source at root HEAD `49ea645940c9df09ad2c84ce20bc18889f75f40f` and read the in-flight fn185 attribution worktree. No build, guest execution, task lifecycle operation, or source edit was performed. Prose follows `/home/agent/.codex/docs/flow-next/prose.md`.

## Existing owners and required seams

| Current owner | Observed contract | Receiving change |
| --- | --- | --- |
| `tools/gomad/rust/gomad/src/main.rs` | Collects UTF-8 `args()`, recognizes existing verbs, extracts global installation, prints errors on stderr | Dispatch `argv[0] == go` before verb parsing; preserve `args_os()` through the wrapper/shim; recognize mandatory wrapper separator and `run-guest` only |
| `campaign/prepare.rs` | Private `Installation` resolution validates Rust manifest, compiler digest, linked host/helper identities and profile; preparation generates overlay, inventories compiler/sources, compiles, verifies inventories, emits canonical provenance | Receive fn232.11 extraction into install/preparation APIs; share the actual recipe, inventory and receipt admission rather than invoking `explore` for a second build |
| `campaign/prepare/process.rs` | Private 0700 scratch, pinned compiler execution, bounded output and deadlines; guest environment removes GOFLAGS and sets wasip1/wasm, cgo off, local toolchain, nogreenteagc | Reuse recipe facts and lifecycle mechanics; retain semantic caller GOFLAGS and inherited stdio where Go owns execution; passthrough does not use this sanitized environment |
| `campaign/mod.rs` | `explore_prepared` retains prepared module before journal creation; seed scheduler owns durable admission/receipt/outcome; post-commit `Observer` sees completed results | Add a distinct internal launcher execution policy and output callback plumbing; the existing observer cannot stream before guest termination |
| `campaign/execution.rs` | Builds linked request, attaches Conditions/cancellation, runs under backstop, publishes output and replay inputs | Add optional output consumer to admitted execution; reuse publication/record/replay machinery |
| `wasmhost/src/linked.rs` | Per-execution `Prepared`, bounded `Sink`, fresh engine/store, final `Execution` with captured streams | Add a feature-independent request-owned output consumer and explicit consumer failure handling |
| `wasmhost/src/model/reply.rs` and `engine.rs` | Namespace output is queued; ordinary `flush(false)` emits full CHUNKs; terminal flush emits remaining bytes | Eagerly flush stdout/stderr when a linked streaming consumer is present, including a marker shorter than CHUNK |
| `campaign/ledger.rs`, evidence store and replay preflight | Canonical authenticated plans, publications, module/provenance/replay input payloads; existing lock discipline | Authenticate origin metadata and propagate it through replay/minimize/inspection and derived artifacts |

`campaign/origin.rs`, `cli/run_guest.rs`, `shim/`, `config/repo.rs` and `install/derive_cache.rs` do not exist in the surveyed root. They are proposed fn233 surfaces. fn232.11's task design already calls for separating manifest admission from build-input admission and exposing portable prepared-module receipts. fn233 should receive that implementation rather than create competing installation types.

The root README and WASM CLI guide still show the Go launcher installation recipe. fn232.11 owns the Rust public install documentation/default helper change. fn233.4 should build on that final guide and preserve its explicit resolution/status contract.

## Batch ownership and integration order

Use one writer for the tightly coupled launcher, campaign, provenance and linked sink changes. Use a separate writer for shim/wrapper/config/cache once the shared install/preparation interface is fixed. Give the Make proof and operator docs to the integration owner after the first launcher proof. The conductor owns integration, one independent integrated review, evidence and Flow completion. Keep concurrent writers in isolated worktrees under `/home/agent/wt`.

1. Finish fn232.11 and integrate its install and preparation interfaces. Keep fn185's request-local profiling changes coordinated with the linked owner. No fn233 implementation starts against the frozen root candidate.
2. Implement fn233.1's linked stream consumer, origin/receipt admission, sequential launcher and golden lane. Run the complete handwritten pinned-Go `-exec` proof before building the shim. Stop this dependency chain if Go cannot parse the forwarded output or parallel package launchers cannot share the artifact root safely.
3. Implement fn233.2 and .3 together against the admitted preparation API. Classification and metadata creation belong in the shim; child lifetime and wrapper identity belong in the wrapper; installation derivation remains in install. Integrate launcher before shim and shim before wrapper acceptance.
4. Prove fn233.4 on the unchanged Makefile using a registered supported selection, then complete the operator guide and integrated review. Commit each completed existing Flow task separately after shared batch evidence proves its requirements.

## Preparation and admission of Go's actual module

Today's `prepare_before_deadline` compiles its own temporary `module.wasm`. Its `prepared_exec` branch accepts only a canonical provenance document whose module digest, build key, configuration, compiler/source inventory, linked implementation identities and cooperative choice identity agree. Setting GOOS/GOARCH and accepting any WASM module would bypass those guarantees.

Receive these internal capabilities from fn232.11, extending them inside fn233 where needed. Names are implementation suggestions, not public grammar.

- A prepared driver recipe resolves the selected installation, validates the pinned compiler and complete inputs, generates the cooperative overlay through that compiler, resolves tags and inventory, and returns build environment plus immutable preparation bindings. Keep the existing compiler version/digest, overlay implementation, source inventory, configuration and choice identity checks.
- A driver invocation preparation resolves concrete packages with the pinned driver under the same effective build flags. Retain the package import paths, module-relative directories, inventories and per-package derived installation/configuration identities. A single Go pattern may resolve several packages; Temporal's guest PWD and schema-derived configuration are package-specific.
- Admission reads and hashes the actual module supplied by Go's `-exec`, validates it against that recipe and package binding, derives the cooperative choice identity for those exact bytes, and emits the same canonical prepared representation/portable receipt used by explicit execution. Read the module once into retained bytes before Go can remove it. Verify the relevant compiler, overlay and source inventories at admission and after the driver completes; a changed inventory makes the invocation fail explicitly.
- Bind the admitted module and provenance to the same installation and current linked host/helper identities. Retain that module, receipt and invocation metadata before execution. Standalone `run-guest` admits an existing authenticated receipt for that exact module digest and installation; it cannot bless an arbitrary module. Missing or changed receipts fail with the existing identity/refusal status.

Package mapping needs an explicit implementation test. Go 1.27.1's `load/test.go` creates `pmain.ImportPath = p.ImportPath + ".test"`; `load/pkg.go` embeds that path in build info. The pinned driver supports `go version -m -json MODULE`. Use that bounded pinned-compiler query to select the exact package from the immutable mapping and verify Go version, GOOS, GOARCH, cgo and admitted build settings. Do not select by temporary filename or basename, which can collide. Build info supplements the authenticated preparation receipt; it does not prove that an overlay was applied by itself. File-list mains require an unambiguous prepared `command-line-arguments` binding. No ambiguous package match is executed.

This module-admission seam is the early proof's main risk. The handwritten proof must retain generated overlay/configuration, complete tags including `test_dep`, compiler invocation and receipt bindings for the module Go actually produces. A separately compiled lookalike binary is insufficient evidence.

For `go build -o PATH`, publish the admitted module at `PATH.wasm` and a correctly quoted executable launcher at `PATH`. Keep its receipt in the authenticated preparation cache and make the script point to that durable receipt/installation association. It must still work after the wrapper removes its temporary run directory. Do not leave the launcher dependent on a temporary invocation file.

## Exact invocation metadata and origin

Capture original `argv[0]` and all original Go arguments from `args_os()` before rewriting. Encode each Unix argument as canonical hexadecimal bytes, using the existing SHA/canonical JSON facilities. Empty arguments, spaces, quotes and invalid UTF-8 remain distinguishable. Use `OsString` directly for child execution; never join arguments for a shell or reconstruct original argv from `-test.*` arguments. The existing guest model accepts UTF-8 arguments, so unsupported guest encodings must refuse explicitly while provenance and host passthrough remain lossless.

Each wrapper allocates a unique private 0700 run directory with a run identity and lifetime lock. Each shim invocation creates a distinct identity and exclusive mode-0600 metadata file within that directory. Metadata contains the original argv bytes, wrapper and invocation identities, pinned real compiler path/digest, selected installation/configuration identities, recipe/tag/overlay identities and concrete package preparation bindings. Bound its size. Canonically encode once, publish immutably and calculate the digest of those exact bytes.

The wrapper supplies the authorized `GOMAD_SHIM_BINARY`, `GO`, `INSTALLATION`, `ARTIFACTS`, `SEEDS` and `RUN` contract values. The shim adds `GOMAD_SHIM_INVOCATION` and `GOMAD_SHIM_INVOCATION_SHA256` only to that compiler child's environment. It must not mutate process-global environment to hand metadata to concurrent children. Strip unrelated inherited `GOMAD_*` controls at the wrapper boundary while preserving the documented installation selection. Host passthrough preserves the environment the shim received.

The launcher opens a bounded regular file under the expected private run owner without following a substituted link, checks owner/permissions, exact digest, typed schema, run/invocation binding and installation/compiler/recipe identities, then copies the exact bytes into authenticated campaign evidence. A digest authenticates retained identity and corruption detection within this process contract; it is not a new protection boundary against a hostile same-UID process. Two overlapping Go invocations must retain different metadata even when all package launchers share one wrapper identity.

Add `campaign/origin.rs` as the one typed origin decoder/validator. Every newly created operator campaign records `origin: operator`; launcher campaigns record `origin: intercepted-go` and the authenticated metadata reference/digest plus wrapper/invocation identities. Bind origin into the campaign plan and retained preparation/execution metadata so it cannot be changed independently. An intercepted minimized result remains intercepted. Replay and inspection authenticate provenance without requiring the temporary wrapper directory or compiler. Add explicit qualification admission at input boundaries and retained-result validation so intercepted artifacts cannot be relabeled as an operator qualification source. Preserve the receiving fn232 format policy; do not change installation version or silently reinterpret old origin-less evidence.

Cleanup holds the wrapper lifetime lock through the child tree's completion, removes only that wrapper's completed invocation files and shim, and leaves retained campaign copies intact. Stale cleanup requires a demonstrably abandoned owner lock; age or a directory name alone cannot release another live wrapper's files. A descendant that continues to use the shim keeps its owner live until it exits. Unknown state is left intact.

## Request-local streaming and profiling coordination

Expose an optional output consumer on `linked::Prepared`, attached before `execute_prepared`. A suitable internal signature accepts a typed stdout/stderr stream and a borrowed byte slice and returns an I/O result. Store an owned, Send-capable consumer handle with the request; no global callback or environment selector. The consumer is independent of `diagnostic-profile`, stress and fixture features. Existing calls default to capture-only behavior.

The `Sink` continues to capture every evidence stream with the current total evidence bound. For stdout/stderr it retains an admitted chunk once, then invokes the output consumer outside the evidence-map mutex. Use a separate per-consumer serialization gate if several producers can append, preserving each stream's order and retaining no unbounded queue. Callbacks must not reenter the sink. Keep output limits in the namespace and evidence limits in the sink; do not truncate or forward bytes rejected by either limit. Record consumer failure once, stop execution through the existing failure/cancellation path, finish available evidence and return an infrastructure outcome with the failure diagnostic. Drain an admitted pending chunk exactly once even when forwarding it fails, so terminal flush cannot append it a second time. Avoid a top-level `Err` path that discards captured bytes before publication.

`engine::namespace_call` currently queues each fd_write result, then `model_operation_inner` uses `model.flush(false)`. `model/reply.rs::flush_streams` waits for CHUNK unless `all` is true. Installing a callback only in `Sink::append` therefore leaves a short marker buffered until exit. For streaming linked requests, flush stdout/stderr eagerly after each completed output operation, using the existing `flush_output` mechanics while retaining ordinary batching for choice/transcript/diagnostic streams. Keep capture-only final bytes byte-identical. Audit failure paths so an output operation whose consumer fails does not get duplicated by terminal flush.

Backpressure is synchronous and bounded. The launcher consumer writes accepted bytes fully to the inherited stream, observes its request cancellation/deadline and reports write/flush failures. Handle partial writes explicitly. Avoid a permanently blocking stdio-lock write inside a callback; the I/O adapter needs bounded cancellation-aware waiting without switching a shared inherited open-file description to nonblocking mode for other processes. A bounded handoff with a request-owned writer and deadline-aware completion is acceptable if teardown remains bounded and no bytes are silently dropped. The process backstop remains the last-resort bound, not the normal consumer-failure mechanism. A hard fail-stop can leave only previously published partial evidence and must not invent a replay command.

The fn185 worker reports these in-flight seams in `/home/agent/wt/fn185-rust-attribution`:

- `diagnostic::LinkedProfile::start()` creates a request-owned profile and bounded shared `LinkedSink`; `finish()` merges thread contributions.
- Existing private `linked_conditions::Session` carries an optional cfg-gated profile sink captured when `Conditions::enter()` starts. Its cloned token reaches cluster store threads; `session.profile(slot, incarnation)` attaches per-thread capture.
- Profiling uses per-thread active capture and explicit session propagation, avoiding transport `ACTIVE`/`FINISHED` globals. `Conditions` remains execution-role/event-order ownership plus profiling capture, not a general output API.

Reuse those session lifetime/isolation conventions and existing cancellation hooks. Keep the output consumer on `Prepared`/Sink so a normal release launcher can stream with profiling disabled. Coordinate edits to `linked.rs` and `linked_conditions.rs` with that worker and integrate profile changes before rebasing the final fn233 sink work.

The streaming test writes a marker shorter than CHUNK, waits for a controller release, then exits. A test-owned callback/control channel observes the marker and releases a test-only bounded synchronization fixture. It introduces no production guest host-I/O capability. Require controller observation while the guest is still blocked, exact final forwarded versus retained bytes, both streams, output/evidence exhaustion, consumer error and cancellation. A barrier-based concurrent test runs different markers and argv under separate requests and checks that bytes, metadata and completion cannot cross sessions. No sleep establishes the proof.

## Launcher policy, reporting and statuses

One launcher invocation creates one ordinary campaign. Seeds run sequentially with fresh guest stores. Reuse `SeedSelection`, the existing preparation retention, journal and publication primitives. Add a launcher-specific internal policy rather than change explore's public defaults. `parallel=1` and `on_failure=first` alone are insufficient because `failure_policy::stop` currently stops only target failures/watchdogs, and `campaign::report` reduces ordinary failure status to a boolean.

The launcher stops on every non-success, including infrastructure termination. The journal records only attempted executions and their actual dispositions, with remaining selected seeds explicitly unexecuted through selection/stop reporting. Do not admit the next seed before the current one succeeds. Retain each completed seed's execution evidence rather than applying explore's success-dropping default. Concurrent Go package launchers create separate campaign directories under the common artifact root and use the existing target-pool/store locks. They share wrapper identity, not a mutable journal or guest state.

Map success to 0, guest failure to its existing supported guest failure exit, and capacity/fuel/helper/output-consumer failures to the existing infrastructure status 3. Input, unsupported-option and identity refusals preserve their current status 2 where that is the existing CLI mapping. Keep this mapping centralized with existing CLI error conventions. A failed Go driver commonly converts launcher nonzero to its own status 1, and Make returns its own status. Report each layer's actual status rather than promising that the wrapper exposes a launcher's raw status through arbitrary children.

Default launcher stdout contains only guest stdout. Forward guest stderr unchanged to stderr, then append Gomad diagnostics there after the stream drains. Never call the existing human campaign reporter on guest stdout. Standalone `--json` suppresses guest forwarding to stdout and emits structured outcome rows while retaining guest output. The shim never adds `--json`; caller `go test -json` stays a Go-driver option.

After failure, publish/authenticate available evidence and determine replayability with the existing replay preflight requirements. Infrastructure traces may be incomplete and the current ordinary retention path drops runner-failure publications. The launcher must deliberately retain available partial evidence while distinguishing it from a replayable artifact. When preflight admits replay, emit diagnostics first and the properly quoted exact `gomad --installation FILE replay ARTIFACT` command as the final stderr line. Otherwise print an explicit no-replayable-artifact message and no command. Cleanup or a generic main error printer must not append another line afterward.

The required first-failing-seed fixture selects a failure followed by a known would-pass seed. Assert that only the first ran. Include success, guest failure, unprepared module, capacity/fuel/helper termination, replayable failure, unavailable replay and standalone JSON fixtures. Replay, minimize and inspection use the same retained module after Go's temporary files and all source/build inputs are removed. Qualification refuses intercepted evidence with the existing status.

## Shim classification and effective flags

Classify only driver arguments, respecting Go 1.27.1's actual parsing. Its `testFlags` recognizes known flags before and after package lists, stops at `-args`/`--args`, forwards `--` and its suffix, and has special unknown-flag/value handling. `GOFLAGS` uses Go's quoted-field parser and applies before explicit argv. Build a bounded parser for this admission policy from that pinned behavior and test it against the driver. Do not use shell splitting, substring matching, or scan the guest suffix for refusals.

Insert injected driver flags immediately after the subcommand, safely before package/guest boundaries. Preserve all unrelated original argument bytes and relative order. Derive effective settings across GOFLAGS, aliases, `-flag=value`, separate values and repeated flags. Preserve caller `-count`, including GOFLAGS; otherwise inject `-count=1`. A caller-owned effective `-exec`, including `true`, receives the module unchanged through Go, with no Gomad execution evidence claim. Recognize a nontrivial fixture launcher as caller-owned too. Values after the boundary such as guest `-exec`, `-count`, `-tags`, `-race` or `-cover` remain guest arguments.

Merge the caller's admitted tags with the installation's full required tags in the effective tag setting; preserve caller-selected tags and all unrelated flags. Resolve controlled duplicate settings once rather than append defaults that a later caller flag overrides. Retain the exact original arguments separately. Overlay/build-environment/identity inputs must match the shared recipe; caller flags cannot silently replace an authenticated overlay or compiler recipe. Guest tests use wasip1/wasm, CGO_ENABLED=0, GOTOOLCHAIN=local and the same remaining runtime recipe controls as preparation.

Refuse effective guest `-race`, `-fuzz`, `-cover`, `-coverprofile`, `-coverpkg`, `-msan`, `-asan`, inherited CGO_ENABLED=1, excluded tags, or incompatible GOOS/GOARCH before source compilation. Honor effective boolean forms and GOFLAGS precedence. Emit one line with the existing unsupported-option status. Invalid driver syntax should retain the pinned driver's error behavior where it creates no unauthorized guest execution. Do not silently remove unsupported flags.

Guest `go test` receives the build environment whether launcher-owned or caller-owned. `go run` and `go build` become guests only for configured main package entries. Unmatched run/build and every other subcommand pass original `OsString` argv and inherited environment byte-for-byte to the pinned real compiler. `go env GOOS` must report the host. Resolve package opt-in with host-side pinned-driver metadata; neither generators nor linters become guests merely because they use `go run`.

Refusal and injection fixtures cover both args terminators, `--`, package lists, unknown test flags, quoted GOFLAGS, repeated flags, guest flag lookalikes, caller count and both kinds of caller launcher. Multi-package driver tests cover `-run`, `-v`, `-json`, Go's package parallelism and exact status/output handling.

## Wrapper, configuration and auto-installation

Recognize `gomad [--installation FILE] [--seeds N|LIST] [--artifacts DIR] -- COMMAND ...`. Parse only the prefix before the first wrapper separator. Preserve everything after it as child argv. Dispatching a child named `replay` never reaches the replay verb. A missing separator remains the unknown-verb error with the wrapper hint; empty command, missing executable and shim-directory failure retain the existing status families. Add no `gomad go`, `gomad make` or additional public verb.

Resolve the real `go` from the original PATH before inserting the shim, canonicalize and authenticate the executable under fn232.11's pinned stock-Go rules, and retain that exact compiler selection. An explicit installation requires the same admitted compiler identity, not merely a version string. Create executable `go` as a hard link/copy/symlink to the Gomad binary in the private 0700 shim directory. Resolve COMMAND after prepending this directory so direct `gomad -- go test ...` reaches argv0 dispatch. Inherit stdio and propagate the actual child's exit/signal semantics using existing subprocess conventions. No downloads or compiler switching.

Read `gomad.toml` at the containing module root. Accept exactly typed `runtime_root`, `derive` and `guest` values. Reject duplicates/unknown keys, unsupported derive values, absolute or escaping guest paths and runtime roots that are not module-relative. Check canonical containment as well as lexical traversal so an escaping symlink cannot choose another runtime root. The Rust workspace currently has serde, serde_json, sha2 and rustix, with no TOML dependency. Implement bounded parsing for the declared string/string-array configuration data model using existing facilities and cover comments/quoting/arrays explicitly; do not silently reinterpret arbitrary TOML as ad hoc key splitting or add an unapproved dependency. Missing optional `guest` means host run/build. The Temporal config sets the authorized runtime root and temporal derivation without expanding the support list.

Explicit CLI installation wins, then documented installation environment selection. Invalid explicit selections fail without auto-derivation. In auto mode resolve the package set in the shim, derive current configuration through fn232.11's same default/Temporal install derivation, capture schema bytes, and validate all required source inputs before cache lookup. Derivation is per concrete package. `installation/temporal.go` currently captures `schema/sqlite` and computes `/workspace/<package>`; the Rust receiving implementation must produce the same configuration bytes.

Key auto-installation entries by authenticated compiler identity, runtime identity, derivation, concrete package, full derived configuration digest including captured schema bytes, and current linked host/helper implementation identities. Verify the entry's full manifest and identities before reuse. Publish exclusively under a per-key lock and admit the winner after a race. Reuse unchanged entries; schema mutation produces a new identity; missing schema refuses even with a previous cache; a changed linked executable cannot reuse its predecessor. Explicit installations are never rewritten. Per-package bindings in the invocation metadata select the derived entry at launcher admission, avoiding one mutable environment installation path shared between packages.

Golden cases cover true/false children, named-verb children, direct go interception, empty/missing commands, directory failure, every config refusal, unchanged cache reuse, schema mutation/removal, changed implementation and concurrent different invocations. Check raw identities and byte-identical second generation in addition to stable projections.

## Actual Temporal Make proof

The unchanged `Makefile` has these relevant steps:

1. Make prepends `$(ROOT)/.bin` to inherited PATH. Verify that it contains no earlier executable `go` shadowing the wrapper shim. Its `go env` queries and `go clean -testcache` are host passthrough.
2. `functional-test` depends on `clean-test-output`, then calls three bare `go test` commands for `FUNCTIONAL_TEST_ROOT`, `FUNCTIONAL_TEST_NDC_ROOT` and `FUNCTIONAL_TEST_XDC_ROOT`.
3. `COMPILED_TEST_ARGS` expands timeout, race, shuffle, parallel flags, TEST_ARGS and `-tags disable_grpc_modules,...,test_dep,...`. The recipes append `-persistenceType` and `-persistenceDriver`.
4. Each command merges stderr into stdout and pipes through `tee -a test.log`. The final recursive `make verify-test-log` requires a nonempty log, some `^ok`, and absence of `^--- FAIL`. The Makefile does not use pipefail or directly preserve each Go exit. The wrapper returns Make's actual status and does not repair that Make policy.

Use a registered supported selection for all three positive invocations, supplying existing Make variables without editing the Makefile. A concrete candidate is the registered `TestFrontendSystemInfo` selection, subject to the installed Rust support gate at execution time:

```sh
gomad -- make functional-test \
  TEST_RACE_FLAG=off TEST_SHUFFLE_FLAG=off \
  PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite \
  FUNCTIONAL_TEST_ROOT=./tests/gomadfunctional \
  FUNCTIONAL_TEST_NDC_ROOT=./tests/gomadfunctional \
  FUNCTIONAL_TEST_XDC_ROOT=./tests/gomadfunctional \
  TEST_ARGS='-run=^TestFrontendSystemInfo$ -v' \
  TEST_TIMEOUT=5m
```

`tests/gomadfunctional/frontend_test.go` imports `tests/testcore`, whose `flag.go` init registers `persistenceType` and `persistenceDriver`. The package can therefore accept the persistence arguments that the unchanged recipe appends. This command remains a proposed integration proof, subject to the installed candidate's runtime acceptance. It is not a completed command result.

Retain the exact Make variable values and expanded argv captured by the shim, emitted module receipt/digest, inserted Go `-exec` chain, wrapper/invocation identities, Go PASS/FAIL output, campaign origin, launcher and child/Make/wrapper statuses, and explicit replay result. The artifact must replay after Go's temporary binaries are gone. The default race-enabled invocation must refuse before build and produce no intercepted execution campaign. In a fresh test log the unchanged verifier should yield a Make failure; record its actual status.

Run the broad default-root invocation separately only as routing/classification evidence. It cannot claim support for all functional, ndc or xdc tests. Test repository generators and linters separately using their actual `go run`/tool targets and fixture captures. `functional-test` itself invokes neither. A controlled Make fixture proves mixed host-tool/guest-test routing and nonzero child status propagation independently of this repository's log-verifier policy.

## Integrated evidence and review obligations

| Flow owner | Required evidence |
| --- | --- |
| fn233.1 / R4, R8 | Pre-exit marker handshake; isolated concurrent streams/argv; first-seed stop; statuses and final replay line; complete pinned-Go handwritten -exec proof; replay/minimize/inspect acceptance and qualification refusal; partial evidence and no invented replay command |
| fn233.2 / R2, R3, R8 | Every classification/refusal; effective argv/GOFLAGS boundaries; byte-exact host environment/argv; caller launchers; Go -run/-v/-json/multi-package behavior; actual passed-module admission |
| fn233.3 / R1, R5, R8 | Wrapper child dispatch/status/cleanup; config refusals; identical explicit versus derived configuration; cache mutation/removal/implementation and concurrent isolation cases |
| fn233.4 / R6, R7 | Unchanged Makefile with exact selector/variable expansion, actual -exec chain and statuses; default-race refusal; separate host-tool passthrough; complete operator guide and truthful support/evidence-origin limits |

The existing parity driver needs a new Rust-owned `run-guest` lane and classification/wrapper fixture lanes. There is no Go host implementation of these verbs to compare against. Use fn232's projection discipline, repeated execution and byte-identical second golden regeneration, while authenticating raw metadata/digests rather than normalizing them away. The positive auto-derived smoke still needs retained runtime evidence.

After integration, run affected Rust formatting/lint/package/boundary checks, focused linked streaming and Rust CLI tests, the owned golden/parity lanes twice, and `make -C tools/gomad validate` if protocol/generator/toolchain inputs changed. Bind final evidence to one frozen integrated candidate and retain exact tool versions, numeric exits and remaining failures in one handover. Shared helper identity changes invalidate earlier qualification claims; route resulting qualification obligations to their existing owners. One independent review covers the complete R1-R8 mapping and the live proof receipts before Flow completion.

No worktree or scratch directory was created for this design. The existing fn185 attribution worktree remains owned by its active worker, and the root fn232.10 source remains untouched.
