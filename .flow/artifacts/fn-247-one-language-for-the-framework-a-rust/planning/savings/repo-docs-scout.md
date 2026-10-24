# fn247 single-pass validation prerequisite

`time-gate.sh` already performs the requested work exactly once. Add one M prerequisite task that exposes required-TLC validation through `make model-validate`, preserving the existing default gate and timing entry point. This report reads current source only. It ran no Flow commands, builds, tests, probes or Git mutations and changed no implementation files.

## Exact reuse

| Current source | Reuse and constraint |
| --- | --- |
| `Makefile:1271,1274-1296` | RUST_MODEL_STEP wraps existing stages when UMPIRE_TIME_STEPS=1. The existing model-check owns every gate stage and its prerequisites. Reuse that target intact. |
| `scripts/time-gate.sh:34-45` | Prints the gate command, invokes `UMPIRE_TIME_STEPS=1 make -C .. model-check` once, captures its complete log, measures elapsed time and replays the log. Required mode can flow through the environment and recursive Make variables. |
| `scripts/time-gate.sh:47-50` | Checks the selected source exists, touches it once, then performs the existing `mise exec -- cargo test -p <crate> --quiet` benchmark once. Defaults remain umpire-checker and umpire/backends/native/src/lib.rs. |
| `scripts/time-gate.sh:7-8,52-55` | Keeps the 10s inner/60s gate soft budgets, custom overrides, printed timings and overrun notes. Exceeding a budget does not fail validation. |
| `Makefile:1305-1307` | Existing model-time already delegates to time-gate.sh. Leave its command and default backend selection unchanged. |

A minimal robust target shape is a new phony `model-validate` whose sole recipe delegates to `$(MAKE) UMPIRE_BACKENDS=require model-time`, using recursive-Make recipe handling. This reaches the unchanged helper and its single gate/benchmark. It creates no second model-check prerequisite or additional timing run.

The explicit recursive Make assignment matters. A caller can supply `make model-validate UMPIRE_BACKENDS=skip`. Merely prefixing the script with an environment assignment allows the caller's command-line variable to reach the helper's nested make through MAKEFLAGS and override that environment. Passing `UMPIRE_BACKENDS=require` explicitly on the recursive Make command settles the required mode. Preserve inherited UMPIRE_APALACHE, CARGO_TARGET_DIR, CI, inner source/crate, budgets and other existing settings. Do not clear MAKEFLAGS globally.

## Existing backend-mode controls

- `umpire/dev/fixtures/fixtures.rs:45-65` compares UMPIRE_BACKENDS exactly with `require`. Required mode insists that attribute Kettle is source-admitted and that every admitted fixture has a selected `tlc.snap.txt`, then passes an empty skipped-consumer list. Default mode passes `["tlc"]`.
- `temporal/testing/fixtures.rs:54-75` applies the same selection to system fixtures, checks every admitted fixture has its TLC golden, and retains explicit no-admitted-system-fixture reporting. Its external GAPS ownership stays intact.
- `umpire/dev/testing/tests/consumers.rs:172-200`, `skipped_backend_requires_selected_complete_goldens`, directly invokes run_at_skipping with an explicit skipped external consumer. It verifies the renderer does not run and an orphan pending snapshot fails. This is a synthetic skip-path control even when the enclosing gate requires TLC. Keep it unchanged; blanket removal of skip coverage would weaken the runner contract.
- `umpire/dev/testing/src/runner.rs:198-207` validates skipped consumer identities and reports that skipping supplies no execution/agreement evidence. A validation wrapper must select mode through the existing fixture callers instead of changing this generic API.
- `umpire/backends/tlc/tests/controls.rs:163,215,236,267`, `tlc/tests/scaling.rs:14` and `umpire/dev/testing/src/backend_tests.rs:446` select real-tool controls from exact required mode. Reusing the existing workspace test stage retains them and all existing mutants/goldens.
- `umpire/backends/apalache/tests/controls.rs:13-15` independently selects UMPIRE_APALACHE=require. `Makefile:1277` fetches its admitted tool when selected. Preserve that independent opt-in through model-validate.
- `scripts/test-capabilities.sh:7-40` checks that the core rejects filesystem/clock/thread APIs and types. It is unrelated to backend-mode selection. It remains the existing Makefile:1282 stage.

## Error and control coverage for the task

1. Use a bounded command-routing harness to prove model-validate requests required mode even with UMPIRE_BACKENDS unset, an ordinary environment skip value, or a conflicting command-line skip value. Assert one complete model-check invocation and one source-touch/test benchmark, in that order. Keep model-check/model-time default-mode cases, and retain optional Apalache propagation.
2. Keep `scripts/test-time-gate.sh:5-23` child-status controls for true, false and exit 37. Keep `:26-58` diagnostic replay/suppression and temporary-file cleanup controls. Keep `:60-85` default/custom soft-budget boundary cases.
3. Cover gate failure and benchmark failure separately. A gate failure must retain its diagnostic log and stop before the benchmark. A benchmark failure must retain diagnostics and fail validation after the successful single gate. Do not retry either phase. Missing selected inner source must fail before touch/test. Successful child output suppression and soft-overrun success remain as implemented.
4. Distinguish status layers. `step` directly returns child 37 (`time-gate.sh:17-20`), and `test-time-gate.sh` asserts that direct contract. An actual failed recipe is reported by GNU Make as a nonzero Make failure, normally status 2. The helper propagates the nested Make result (`:40-43`); an additional outer Make can also return 2. Do not promise end-to-end exit 37 from `make model-validate`, and do not classify Make's normal propagation as a lost child failure. Child diagnostics must survive all layers.
5. Run the new command once for its acceptance receipt, retaining stage timings and both soft-budget results. Count the new command's single required gate as the required model-check evidence. Do not immediately repeat model-check or model-time solely to obtain separate labels. Any separately required live or Apalache control remains required under the existing task/spec rules.

The routing controls must exercise the actual wrapper/helper commands using disposable fake gate/compiler commands, rather than asserting recipe text alone. The real acceptance run remains necessary because a routing harness supplies no TLC, mutant, golden or host evidence.

## Documentation gaps

- `model/README.md:76-87,149-152` describes the two existing gate/timing commands and benchmark overrides. Add model-validate as the single required-backend gate plus existing benchmark; keep the default model-time description.
- `model/README.md:830-844` documents fetching the pinned TLC tool, required execution and independent Apalache opt-in. The new command requires the same pinned tool readiness; it should not gain another downloader or change backend admission.
- `model/GUEST_BUILDS.md:91-103` currently lists required model-check and a separate default model-time command. Add the combined validation path for workflows needing both evidence sets, while retaining the individual entry points and historical receipts below.
- `model/DESIGN.md:3-5` gives current top-level commands; `:215-220` explains required versus default execution and two backend renders. Add the combined entry point without changing those semantics or rewriting historical measurements.
- `.plans/UMPIRE_MODULES.md:51-52` records existing backend mode ownership. No module-graph change is needed for this target-only prerequisite.

The savings come from callers choosing one existing required gate plus one existing edit/test measurement instead of invoking a separate required gate and then another timed gate. This task adds no caching, TLC batching, skipped stages, reduced rendering, parallel scheduling or new validation policy.
