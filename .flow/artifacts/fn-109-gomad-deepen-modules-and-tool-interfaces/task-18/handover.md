# Task 18 source handover

Handle and Mapping now select one private LOCAL or PROCESS implementation at creation. The public facades delegate every existing operation; local resource indexes hold concrete owners, while process owners retain typed-command identity/cache state. Public os/libc APIs, process wire literals, shared volume persistence/replay, and filesystem path-operation selection are preserved.

Source is frozen at `final-source.sha256` (14 files; manifest SHA256 `1d42629bd964f696412940bf75619878c965b07f54cfe693f7eb31e91d5861be`). The descriptor inventories 79 overlay inputs. All worker commands have terminated; no worker subagents are live. Task remains `in_progress`; commits `[]`, base/HEAD `0dd05b313acd0986312da7fd3159520e6a21f1bf`. No acceptance or review verdict is claimed.

## Evidence

- `ownership-red.log` observes the actual old production optional-state and receiver redispatch violations (exit 1, 0.202s); `ownership-green.log` and `gate-selection-green.log` pass the migrated architecture, package boundaries, prior network ownership, and typed-command guards. This was an architectural assertion failure, not a compilation failure.
- New literal behavioral expectations pass both preserved old production and migrated production: `behavior-old/new.log`, `process-host-old/new.log`, and `direct-old/new-final.log`. These preserve behavior rather than asserting that refactoring changes it. The full direct filesystem runs explicitly exclude native subprocess transport, which the developmental shim cannot supply.
- `gate-selection-red.log` observes all four new process entrypoints excluded by the old canonical Make filter (exit 1); `gate-selection-green.log` passes the amended selection (exit 0). Runner root selection includes the same four top-level process tests. Strict node-delay watchdog exclusion and its separate forward invocation are unchanged.
- Generator/version tests, `make generate && make validate`, final `make validate`, nested/root/overlay vet, preservation checks, and `git diff --check` pass. Developmental overlay tests, twenty focused repetitions, and focused race tests pass. Every log records its exact command, exit code, and elapsed seconds; `evidence.json` indexes them.
- Root toolchain-tagged tests and Runner integration tests compile/link (`root-developmental-link.log`, `runner-integration-compile.log`). Those binaries were not executed: compilation is not native process, IPC, virtual-time, replay, or isolation qualification.

## Coverage and preservation

The shared public-operation table has standalone, in-process, and actual-process entrypoints for partial EOF/offsets, writes and append restrictions, metadata/truncate, sorted incremental directory reads, Chdir, unlink survival, and closed identity/access errors. Actual process entrypoints explicitly select PROCESS/hard isolation, admit and wait for nodes, check `NodeStateExited`, and record/replay. A separate actual-process fixture checks rejected replay writes before mutation. These native fixtures are present and selected, but have not executed on this host.

Direct tests cover local alias charging and charge transfer, overlap/capacity, mappings surviving file close, readonly mounts, observer rejection before mutation, 100k-handle capacity, and generation revocation. Existing local mapping zeroing/flush and volume tests remain. A native pipe/framed transport fixture covers malformed partial counts, copied/cached/nil mapping bytes, writable-map precedence, failed versus successful closes, and revoked-domain behavior. Direct host-registry tests exercise 100k combined resource rollback, wrong domain/kind, successful-close-only removal, and domain revocation. The pipe fixture compiles but is not counted as executed coverage.

Explicit backend differences remain: PROCESS writable Map returns ENOTSUP before closed/access/bounds checks; LOCAL mapping aliases shared storage with one byte charge; PROCESS bytes are copied then cached without revalidation (including the existing nil-cache refetch); PROCESS nil ReadDir results normalize to an empty slice. No semantics were uniformized. `preservation.log` verifies unchanged baseline inventory paths, old-production preimages, and byte-identical tested gomadfs copies. Typed codecs and all 41 task-14 vectors/82 literal frames, host registry production, runtime transport, shared volume model, os/libc adapters, earlier network/lifecycle changes, generated protocol mirrors, and unrelated dirty files are retained.

## Native qualification remains open

Baseline generation/validation passed, but canonical patched overlay/Runner/root commands failed before production editing because `.toolchain/bin/go` is absent (exit 127; `baseline-overlay/root/runner.log`). Actual host is linux/arm64. Task-17 `native-toolchain.log` (exit 2) is reused because complete mode still supports only darwin/arm64 and linux/amd64; unchanged task-16 `linter-platform.json` records the incompatible Mach-O ARM64 linter. Neither unsupported builder nor incompatible linter was retried or replaced. Supported native gates, full host/test-host, subprocess IPC/isolation/replay, and native virtual time remain unqualified; there is no waiver.

Developmental scratch is external: `/tmp/gomad-task18.U1uS6Z/go`, with exact old production in `/tmp/gomad-task18.U1uS6Z/old-go` and captured inputs in `baseline-overlay`/`baseline-version`. It derives from task-17 `/tmp/gomad-task17.WFwzYI/go`; stock executable is Go1.27.1 linux-arm64. The unchanged runtime shim `src/runtime/gomad_task14_development.go` has SHA256 `8630de3ebb792517ccd2051a16bf1daab70aaf44e204326c6e04fc0cda404af6`: IO profile/control disabled, global domain token, stock nanotime, inert external arrivals, unavailable blocking/trace transport. This is DEVELOPMENTAL evidence only; no production runtime/net stand-in was introduced.

Tier: session (jev-unavailable(no_key)). No executed-model identity is inferred from the dispatch configuration. No staging, commits, Flow lifecycle mutations, or formal review dispatch occurred.

stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)
