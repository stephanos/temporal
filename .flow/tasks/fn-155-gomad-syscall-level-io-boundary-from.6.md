---
satisfies: [R5, R6]
---
# fn-155-gomad-syscall-level-io-boundary-from.6 Classify the 15 deterministic-I/O adapters and run network adapters excluded

## Description
Classify every adapter with evidence, and for the networking-only ones run their covered workloads with the adapter excluded and the boundary on. Cover the three unexercised gRPC adapter paths.

**Size:** M
**Files:** classification report under `.flow/artifacts/fn-155-gomad-syscall-level-io-boundary-from/`; workload or test additions under `deterministicio/testdata/` and `qualification/` as needed.
**Touches:** [tools/gomad3/deterministicio/testdata/**, tools/gomad3/qualification/corpus/**, tools/gomad3/qualification/*.json, .flow/artifacts/fn-155-gomad-syscall-level-io-boundary-from/adapters/**]

### Approach
- Start from the registry (`deterministicio/profile.go:63-181`) and each `*_adapter.go`: grpc, xnet, sockaddr (network); memberlist, cactusstatsd (UDP — expected to remain, since the boundary serves stream sockets only); sprig, validator (DNS); pebble, memory (filesystem/os); libc (semantic); sentry, otelsdk (process); hashicorpmetrics, fx, temporalsdk (signal).
- For grpc, xnet and sockaddr: run their tests/fixtures (`grpc_adapter_test.go`, `grpc_dns_test.go`, `xnet_adapter_test.go`, `sockaddr_boundary_test.go`) as workloads with the adapter excluded and the boundary on; same seed twice → identical transcripts.
- gRPC: exercise the DNS resolver path (expected: still needs an adapter or a DNS decision) and disconnect errno classification. Channelz socket-option introspection exists only in gRPC's Linux files; on darwin/arm64 record it as not reachable on the platform in use, deferred to fn-128.
- Escape inventory: statically scan the evaluated workloads' closures for calls that bypass the `syscall` edge (assembly issuing raw syscalls, `x/sys/unix` `SyscallNoError`/`RawSyscallNoError` on linux, cgo) and list them in the report.

### Investigation targets
**Required:**
- `tools/gomad3/deterministicio/profile.go:63-181`
- `tools/gomad3/deterministicio/grpc_adapter.go`
- `tools/gomad3/deterministicio/xnet_adapter.go`
- `tools/gomad3/deterministicio/sockaddr_adapter.go`

## Acceptance
- [ ] A report classifies all 15 adapters with evidence (rewrite targets, test or run result).
- [ ] Each networking-only adapter's workload ran with the adapter excluded and the boundary on; result recorded (deterministic, or reclassified with the observed failure).
- [ ] gRPC channelz, DNS resolver and errno classification paths are each exercised, classified as still needing an adapter, or recorded as not reachable on the platform in use.
- [ ] The report lists static escapes found in the evaluated closures.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
