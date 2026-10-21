# Integrated guest verification

Model authors can reproduce both unwind-preserving guests from the integrated source.
The proof uses `ab9c8edb38318d497e4ce06cd6548fd96b8772c5` and the review boundary
remains `b27d38ecd059f1b160e6a0285c7d0deed66935b5`, before the first guest builder.
Interleaved fn-217 changes retain their own completed review evidence.

## Linux path independence

Exact runner `.flow/tmp/fn2301-integrated-artifacts/run-proof.sh` ran with:

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal
timeout 600s env -u RUSTFLAGS -u CARGO_ENCODED_RUSTFLAGS bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2301-integrated-artifacts/run-proof.sh
```

Exit 0, 2026-10-21 06:45:02 to 06:47:30 UTC. Linux 7.0.14/aarch64,
Rust 1.99.1 (`58cf8f9c35b355199727627d07ae34a4e2846d67`, LLVM 23.1.1),
Cargo 1.99.1, Go 1.27.0, protoc 29.5. Each build used `CARGO_BUILD_JOBS=4`.
The runner created `.flow/tmp/fn2301-integrated-proof.IJl2tf/` with `mktemp -d`.
Clones `source-a` and `different-source-b` had clean status before and after.
Initially absent targets `independent-target-a` and `independent-target-b`
have distinct identities `58:453312922` and `58:453329518`.

| Guest | Bytes in both clones | SHA-256 in both clones |
| --- | ---: | --- |
| Stock | 192255 | `5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1` |
| Temporal | 3330375 | `0d4e1b112097ba62f62f53c6a95ecc7776bfbf907df8594592586d3fe767ece0` |

Both comparisons passed. The warm gate's stock and Temporal output also compared
equal to these proof files after default, required and timing runs.

## Integrated gates

All commands ran from `/Users/stephan/Workspace/skunkworks/umpire/temporal`.
Builds used `CARGO_BUILD_JOBS=4`,
`CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo` and
`GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go`.
Logs live under `.flow/tmp/fn2301-integrated-artifacts/`.

| Command | Terminal result | Log |
| --- | --- | --- |
| `make model-check` | exit 0, host suites 0.336/8.081 s | `model-check.log` |
| `UMPIRE_BACKENDS=require make model-check` | exit 0, real TLC controls, host suites 0.870/8.905 s | `model-check-required.log` |
| `make model-time` | exit 0, full gate 163.764 s, inner loop 3.295 s | `model-time.log` |
| `bash model/scripts/tests/guest-parity.sh` with all five guest/native path variables | exit 0, stock and all panic/trap/import controls collected | `profile-parity.log` |
| `CGO_ENABLED=0 mise exec -- go -C model/umpire/hosts/go vet -tags test_dep ./...` | exit 0 | `go-vet.log` |
| `CGO_ENABLED=0 mise exec -- go -C model/temporal/hosts/go vet -tags test_dep ./...` | exit 0 | `temporal-go-vet.log` |
| `bash -n model/scripts/guest.sh model/scripts/tests/build-guests.sh model/scripts/tests/guest-parity.sh` | exit 0 | host command output |
| `python3 -m unittest discover -s model/scripts/tests -p 'test_*.py'` | exit 0, one table-driven control | host command output |
| `git diff --check` | exit 0 | host command output |

The full gate exceeds its 30 s soft target. Native tests consumed 98.880 s and
guest builds 55.257 s. The edited-checker loop meets its separate 10 s target.
These measurements are from a shared executor and establish no idle-host percentile.

Receipt creation refused the unrelated dirty `MILESTONES.md`; no reusable green
receipt or skip is claimed. The full commands above actually ran. The inherited
implementation baseline was green for focused script controls; this late-phase
continuation re-ran integrated gates rather than treating old lane results as current.

The verbose parity log names `TestGuestProfileCaughtPanicParity` for actual Tiny
EffectPanic and Overflow full native responses across three calls and Reset,
`TestGuestProfileUncaughtPanicsTrap` for invalidation, `TestKettleParity` for native
stock goldens, and `TestDeniedDescriptorImports`/`TestRejectsDescriptorImportContracts`
for BADF, unchanged full memory, namespaces and exact signatures.

DESIGN remains 999 lines. macOS and cross-OS identity remain open under fn-232;
task .2 owns committing the Temporal artifact, embedding and freshness integration.
Tracker is inactive, plan-sync is disabled, and QA is off. No Model, ABI, protocol,
Realization, original Go/SDK panic semantics or third-party version changed here.

## Post-review verification

The required Phase-5 cumulative classifier returned FULL (exit 1) against the
original task base. The final `make model-check` exited 0, including host suites
0.353/7.018 s, captured in `verify-model-check.log`. The final `make model-time`
exited 0, captured in `verify-model-time.log`. Its full gate took 159.170 s
(30 s soft target missed), and its edited-checker inner loop took 1.694 s
(10 s soft target met). Native tests consumed 98.642 s and guest builds 50.483 s.
After both commands terminated, the stock and Temporal gate files again compared
equal to the fresh proof. Receipt writes again refused foreign `MILESTONES.md`
dirt. No full gate was skipped and no source changed after the reviewed checkpoint.
