# Task 17 historical source checkpoint

The conductor can review the exact corrected task-17 source in
`/home/agent/.cache/codex-build/tmp/fn109-task17-checkpoint.0nly2wj9`.
All 17 historical identities match the post-correction manifest. Exactly 13
source paths differ from committed HEAD. The focused stock checks passed.
Full scratch validation retains a measured root-qualification input gap;
native acceptance and R12 remain open.

The helper archived exact HEAD
`add55cf6116fbed23e8c4051e120521b2f3f033c`. Selected paths were `tools/gomad3`,
committed `tools/gomad3sim`, root `go.mod`/`go.sum`, and the qualification
generator spec and generated manifest under `tools/gomad3integration`.
The archive contained 931 regular files, SHA-256
`3198fdd5e9649438b88c4e354eded42068fe78e0267a123c8838023d84acb857`.
Every member was checked for absolute paths, traversal, links and special
entries before extraction. The complete module came from committed HEAD;
no later working-tree architecture checker, World source or other task-19
source entered scratch. The existing architecture gate stayed at HEAD.

The frozen source manifest remains
`final-source-post-correction.sha256`, SHA-256
`1692362f697d85d42aeabf6b96e878e77245c03848280e0ac7b72cf6bd487095`.
`checkpoint-verification.json` records every exact full-file hash, source origin,
scratch path and before/after inventory. Four origins are retained preimages.

| Historical source | Retained origin | SHA-256 |
| --- | --- | --- |
| Runner `simulation_root_integration_test.go` | `/tmp/fn109-task17-preimage.yZkyiGoM/tools/gomad3/runner/internal/execution/simulation_root_integration_test.go` | `bc4c86b6210232026f0eb94edbee209fa9817a62b66d58a2da7b627e9bbbf833` |
| `toolchain/version/version.json` | `/tmp/gomad-task18.U1uS6Z/baseline-version/version.json` | `75d2429a49e42df374e2fc2b77710d1694322c5198a4b54f6c9bfb95ab88c8de` |
| `Makefile` | `/tmp/gomad-task18.U1uS6Z/Makefile` | `28318e32d3360cfcd644298944e864e704670ac80f55c550d32461a55afc8497` |
| `simulation_gate_selection_test.go` | `/tmp/gomad-task18.U1uS6Z/simulation_gate_selection_test.go` | `57febc14101facc1b4db77ba3f8dba55669e51036e23790eb1372c27dec38bbb` |

The helper independently reproduced the Runner preimage by removing exactly
the four task-18 filesystem selector additions from the current source. All
other 13 manifest entries matched their working-tree bytes. Four of those,
the generated choice/live-capability mirrors, already matched HEAD and add
no checkpoint delta.

The actual source delta is limited to these 13 paths.

- `tools/gomad3/Makefile`
- `tools/gomad3/network_handles_ownership_test.go`
- `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go`
- `tools/gomad3/simulation_gate_selection_test.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network_handles_test.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_commands_export_test.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/process_network.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_handles.go`
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/simulation_network.go`
- `tools/gomad3/toolchain/version/version.json`
- `tools/gomad3sim/network_handles_toolchain_test.go`
- `tools/gomad3sim/network_process_handles_toolchain_test.go`

All stock checks used the pinned linux/arm64 Go 1.27.1 executable with
`GOWORK=off`, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, seed variables unset and
`-tags test_dep` for tests/vet. The 12-test root-package selection passed,
including network ownership, typed-command ownership, canonical process gate
selection and all nine inherited architecture checks. Its list command ran
first. The gate regression discovers all 13 actual process test declarations
and checks the expanded Make filters, including the separate forward-delay
invocation and strict-delay watchdog exclusion.

Generation/protocol/version unit tests, `version-generate -check`,
`protocol-generate -check` and scoped root-package vet passed. `gofmt -l`
returned no paths. All archived and overlaid source files remained byte-identical
after the checks. Only the scratch `.toolchain` generator cache is excluded
from the source inventory.

The helper ran the complete `make validate` command against both baseline and
candidate. Both exited 2 at the same `validate-qualification` boundary because
the deliberately selected archive lacks the root `tests` directory. Before
that boundary, version, protocol, boundary/compiler-fixture checks,
patch/overlay inventory, script ownership, compatibility-pack checks and the
host-profile test passed in both runs. This checkpoint does not claim a fresh
full validation pass and did not alter the validation target or broaden the
archive to unrelated root workloads. The original hash-bound successful
`make generate && make validate` log remains SHA-256
`2006c04a8a5a7409754d947dbe816a7288a94a16ad827643a5c9aefc7c0098b8`.
The original post-correction `make validate` success log remains SHA-256
`65eaf67c634db065bd53d54c47ea45d54187f8f48e412e70aa7054dd69302450`.
Exact new argv, environment, exit codes and log hashes are in the new manifest.

The original direct-model, scoped race, 20-repeat, old-filter RED and corrected
GREEN evidence is retained under its reviewed historical source identities.
This preparation introduced no implementation change and did not repeat those
workloads. The two root simulation fixture files are exact source-checked
entries. Their original developmental root link-only binary and Runner
compile-only binary remain available and are hashed in the manifest. Neither
binary was executed. No new root workloads or runtime shim were used.

The original source audit found no remaining actionable finding after the
canonical filter correction. Its source boundary and original/corrected logs
remain immutable. The native builder's retained linux/arm64 rejection, absent
patched toolchain, scoped Mach-O linter boundary, all native Quick commands,
real Runner transport and both supported-platform qualifications remain open.
Developmental stand-ins, compile/link results and source selection cannot
establish native interception, IPC, virtual timers/scheduling, hard isolation
or exact replay. D12, resolved D14 and the strict-delay watchdog disposition
remain unchanged. Task 18 and supported-platform conformance are still needed
to satisfy R12.

The once-only fresh bounded tier judgment returned `no_key`.
`checkpoint-judgment.json` retains that result. The requested implementer is
`gpt-6.1-sol` at high; the writer/reviewer family is Codex and actual backend
metadata remains unknown. No agents or bridges were dispatched.

The preparation wrote only new task-17 checkpoint artifacts and isolated
scratch. It made no shared source/docs, Git/index or Flow mutation, and no
commit, push, stash or worktree. No live commands remain. This report supplies
source-checkpoint evidence without a formal SHIP verdict or Flow completion.

`checkpoint-prepare.py` has SHA-256
`00285c395f02d07dbbb3f16f5f739007f5bb4d031b3887e05d2d8ce333b692bc`.
`checkpoint-verification.json` has SHA-256
`d5229d0d6e4c0ef31990ab651a03c80d1d84430eb50dc04d8597ab7a0d030ed0`.
