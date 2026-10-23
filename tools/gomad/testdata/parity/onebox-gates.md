# Ordered Rust one-box gates

`onebox-gates.py` supplies bounded execution and command receipts to the fn222
support receiving harness. The existing
`rust/workload_tools/temporal-support.py` remains the qualification entry point;
its integrated adapter must delegate here while preserving its private frozen
observation context. Do not run both routes as separate qualification attempts.

Build the ordinary release `fn222-support` example from the integrated frozen
Rust source. The runner takes `--binary`, `--support`, `--repo`, `--root`,
`--compiler` and `--runtime-root`, all paths. `--root` must be absent. Optional
`--source-manifest`, `--cache`, `--captured-config`,
`--private-observation-config` and `--prepare-only` preserve the existing entry
point's explicit inputs. A private context requires its frozen source manifest
and cache; it is checked by the original producer admission, control and native
observation owners. Native plan/qualify observations are sealed under the actual
operator PID and checked again before publication.

Complete `--smoke-config`, `--frontend-config` and `--inspection-config` inputs
may be supplied. Otherwise `fn222-support temporal-policy REPO PACKAGE POLICY
OUTPUT` derives each qualification configuration through the shared Rust policy
owner; an existing `--captured-config` may supply inspection's captured input.
The driver never derives qualification by editing an installation policy.

The fixed sequence is smoke preparation, success, canary, shared onebox
inspection, frontend preparation, success, canary, support publication and
independent retained-list validation. Each support pair has a separate result
root. Preparation and inspection roots stay outside those roots. Preparation
runs without mandatory-workload intent; only smoke measured commands enable it.
A failure stops before the next stage; pending output contains only attempted
commands and completed gates. `--prepare-only` returns after smoke preparation
and makes no measured or support claim.

The outer deadline is 3600 seconds, shared by every process. Preparation gets at
most 600 seconds within that deadline. Per-stream command output is at most
64 MiB. Timeout/output overflow kills the entire command process group and
retains its actual negative signal status. A canary must retain raw CLI exit 1;
its validated wrapper gate has exit 0 only after an independent public `report`
and linked `check-result` both succeed. A CLI exit alone cannot earn a gate.

Every process retains canonical private `gomad.rust-onebox-command/v1` JSON
binding exact argv, executable path/hash, actual PID, numeric exit, wall-clock start/end, elapsed
nanoseconds, selected environment facts/hash, mandatory intent, and complete stdout/stderr sizes, hashes and
paths. Each measured command also retains a `qualification_spec` witness with
its clean absolute path, exact raw bytes in base64 and SHA-256 (at most 16 MiB).
The runner snapshots it before launch and refuses a changed or missing spec
after process exit while retaining the original witness and actual exit. The
independent receivers decode those exact bytes and compare the complete
registered envelope, including the 32 MiB World bound, rather than inferring
omitted limits from a Result or seed plan. Optional private observations preserve the original native seal. Stage
logs retain these nested command facts. The support request binds each Result
to its original command receipt and records all eight ordered row gates,
preparation receipt/hash/module, and the shared inspection receipt/hash. The
linked receiver independently authenticates Results, publication receipts,
artifacts, named PASS, unchanged limits, module agreement and both support rows.
The Rust list schema is `gomad.rust-wasm-support-list/v1`; the Markdown output is
its projection, retaining the Go contract's fixed per-test statement.

Tracked source files, explicit configuration inputs, the release operator,
receiver, runner and inspection tool are hashed before measured work and checked
before and after each stage. An external conductor source manifest is used
unchanged; absent one, a complete tracked-source manifest is retained. Integration
or another candidate change requires a new run. The final R12 validator still
owns independent admission of private observation seals; merely preserving an
optional seal field does not establish final qualification.

The existing inspection entry point delegates its onebox owner to the parity
gate, retaining the private observation context and original source/cache
controls. Shard and workflow owners retain their original reader contracts.
Each native command is registered before a gated launcher can execute it, so
the global driver can clean the transitive wrapper/native process tree.
Native compiler groups remain in the registered session; timeout cleanup stops
and kills all session members. No measured qualification has been run by these
bounded fixtures:

```sh
python3 tools/gomad/testdata/parity/onebox-gates-test.py
```
