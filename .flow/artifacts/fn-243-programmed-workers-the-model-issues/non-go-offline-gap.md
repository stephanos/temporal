# R4: non-Go executor recorded, not proven

Task: fn-243-programmed-workers-the-model-issues.3. Checked 2026-10-24 at source
`8295401a5dab0d90c2b5661566e74a39e8264cde` on branch `umpire`.

No usable non-Go gRPC stack was found in the checked offline environment. R4 takes its
explicit recording alternative: no non-Go executor was implemented or executed, and neither
Complete nor Fail is claimed to pass in another language. No dependency was installed or
fetched. The available compiler alone does not supply a gRPC client runtime.

## Offline inventory

All commands ran from the repository root. The refreshed full output is retained locally at
`.flow/tmp/fn2433-inventory.log`; the commands, checked locations and relevant results below are
the durable record. This is a bounded inventory of the current environment, not a claim that
no other stack exists anywhere.

| Check | Observed result / missing piece |
| --- | --- |
| `mise exec -- protoc --version`; `mise where protoc` | Exit 0: `libprotoc 29.5`, `/home/agent/.local/share/mise/installs/protoc/29.5`. Protobuf compiler is available. Bare PATH absence was misleading. |
| `python3` with `importlib.util.find_spec` for `grpc`, `google.protobuf`, `temporalio` | Python 3.14.4: grpc and temporalio return None; google.protobuf's parent package is absent. No Python gRPC or protobuf runtime is installed. A Temporal SDK is optional for raw execution. |
| Rust Cargo source, archive and index caches; `model/Cargo.lock` | prost/prost-build/prost-derive/prost-types 0.14.4 are cached; no tonic-family package found. Encoding support exists; an established Rust gRPC transport/client and its offline dependency closure do not. |
| `node --version`; `node -e` using `require.resolve` for `@grpc/grpc-js`, `@grpc/proto-loader`, `protobufjs`, `@temporalio/worker` | Node v22.22.1; all four modules NOT_FOUND. Raw Node execution needs a gRPC runtime plus protobuf loading/generated bindings; it does not need the SDK. |
| `npm cache ls` filtered for grpc/proto-loader/protobufjs/temporalio; `npm root -g` | Only the `@temporalio/worker` registry metadata URL is listed, no matching runtime packages. Global root `/usr/local/share/npm-global/lib/node_modules` contains `@openai/codex`, no matching stack. Metadata is not an installable dependency closure. |
| `java -version`; `shutil.which` for mvn/gradle | OpenJDK 25.0.3 exists; mvn and gradle absent. No gRPC/protobuf-java/temporal-sdk jars found in the checked Maven/Gradle cache roots. Java needs the runtime and generated service/message bindings. |
| `shutil.which` for grpcurl/grpc_cli/grpcui/buf/protoc-gen-grpc-java/grpc_python_plugin/grpc_node_plugin | All absent. No alternate ready client or non-Go gRPC generator was found on PATH. |
| `model/temporal/hosts/go/go.mod` | Established Go gRPC v1.83.1 is available. Go execution is the existing oracle, not different-language evidence. |

Inventory details: `CARGO_HOME=/home/agent/.cargo`; VIRTUAL_ENV, CONDA_PREFIX and
NPM_CONFIG_CACHE were unset. `python3 -m pip cache dir` and `npm config get cache` reported
`/home/agent/.cache/pip` and `/home/agent/.npm`. `site.getsitepackages()` and
`site.getusersitepackages()` identified the Python roots below.

The cache scan used `os.walk(root)` and case-insensitive name fragments
`tonic`, `prost`, `grpc`, `protobuf-java`, `temporal-sdk`, `proto-loader`, `protobufjs`,
`grpcio`, `temporalio`. Roots checked:

- `/home/agent/.cargo` (prost only); `/root/.cargo` (absent).
- `/home/agent/.m2`, `/root/.m2`, `/home/agent/.gradle`, `/root/.gradle` (all absent).
- `/home/agent/.cache/pip` (no matching names); `/root/.cache/pip` (absent).
- `/home/agent/.local/lib` (absent); `/usr/local/lib/python3.14/site-packages` (absent).
- `/usr/local/lib/node_modules` (absent); `/home/agent/.npm` (no matching names).

Additional direct package-directory checks found no matching grpc/protobuf/temporal/google
packages in `/usr/local/lib/python3.14/dist-packages` or `/usr/lib/python3/dist-packages`;
`/usr/lib/python3.14/dist-packages` was absent. The npm global root was checked separately
as above. Python import resolution, Node module resolution, lockfile inspection and npm's own
cache listing supplement the filename scan (which alone cannot identify hashed cache contents).

## Reuse when a stack is available

The executor can consume the same binary `google.protobuf.Any` wrapping
`umpire.temporal.v1.WorkerProgram`, exported by `umpire program --model <guest.wasm>
--world <world.binpb> --out <program.binpb>`. Schemas are
`model/temporal/proto/umpire/temporal/v1/{temporal,activity}.proto`; generate the selected
language's bindings for these and the upstream Temporal WorkflowService API dependency closure.
Protoc is ready; the selected language runtime, service bindings and a runnable standalone
executor/test entrypoint are still needed.

Follow `model/temporal/hosts/go/reference/worker/program.go` for startup validation,
namespace/task-queue guards, ordered exact scenario/attempt/activity-type matching, strict
`binary/plain` scenario-header decoding and non-retryable `umpire.unprogrammed` fallback.
Follow `reference/worker/raw/raw.go` for PollActivityTaskQueue and sending the selected
Completed/Failed request with the actual token, namespace and executor identity. Preserve all
other programmed protobuf content. The process must receive only the program and Temporal
connection configuration; no Umpire/Model/Driver connection or Go-worker wrapper qualifies.

Reuse these existing behavior assertions as an oracle; port assertions to the new language
rather than treating a Go process as its implementation:

- `model/temporal/hosts/go/reference/worker/raw/raw_test.go`,
  `TestFollowExactResponseAndUnprogrammed`: exact completion result/token and nonmatching-task
  non-retryable failure. This unit test does **not** exercise a matched programmed Fail row.
- `model/temporal/hosts/go/reference/worker/program_test.go`,
  `TestProgramValidationAndOrderedMatch`: first-match ordering, exact-key mismatches,
  unknown-key startup refusal and empty program admission.
- `model/temporal/hosts/go/program_test.go`, `TestResetProgramRowsAndScenarioHeader`:
  issued complete/fail/retry rows, every admitted attempt, payload/failure content and stability.
- `tests/testpilot_model_slice_umpire_test.go`, `umpireSlice.program`,
  `TestTestpilotModelSliceThroughUmpire/{raw,sdk}/{Complete,Fail}`:
  real program-file export, actual CLI exploration and server-history judgments, including
  `Worker::RespondCompleted Accepted` and `Worker::RespondFailed { retryable: false } Accepted`.
  The current harness selects Go raw/sdk internally; it needs explicit external-executor
  provisioning before it can prove a non-Go worker. Keep the Go harness/judgment assertions,
  launch the independent process on that program file and queue, and record its actual kind/version.
  Run complete and fail with Umpire disconnected from the worker; retain the resulting Runs.
  Existing deviation, empty-program and server-free replay controls remain useful after that.

The supported checkpoint retains exported program SHA256
`518a416709b0ec47f968ff6f97fba0cfa6dc536bce59f233d59906f0497548cd` and guest SHA256
`b4a4f57f1a64525ae568bd56a73eff85b0b253aeae44d9e4de3e2d88f546cf2a`, recorded by task .2
at executable source `fad276dfc399766a28b51d4262829f90be6e73b6`. Program bytes depend on
the World; this hash identifies the retained export, not every future namespace/queue.
This task changes none of those artifacts. The atomic Model, original counterexamples and
open-state unheld Cancel refusal remain as recorded; deferred R2 work in task .5 and parent
specs fn-243/fn-228 remain open.

## Verification scope

Before edits, `cd model && mise exec -- cargo test -p umpire-temporal --test program_history
--test proposal` exited 0: 12 program_history and 7 proposal tests, none filtered out.
Log: `.flow/tmp/fn2433-baseline.log`. No honorable receipt existed for this exact command.
This documentation-only task does not rerun broad Model/live gates and adds no mirror test.
The inventory establishes a recorded gap; it does not establish cross-language execution.
