# Gomad: one-command Temporal test setup on WASM

## Conversation Evidence

> user: "our goal is definitely to run arbitrary functional tests with gomad (wasm)."
> user: "yes file these specs"

## Goal & Context

Running a Temporal functional test under the cooperative WASM profile works today only with a hand-written `wasi.Config` JSON file. `tools/gomad/README.md` (lines 150-172) states that the one existing functional guest, `TestFrontendSystemInfo` in `tests/gomadfunctional`, needs captured SQLite schema bytes, larger capacities than the first fixture, a guest working directory of `/workspace/tests/gomadfunctional` with matching `PWD`, writable `/workspace`, `/tmp` and guest working directories, immutable captured `schema/sqlite` bytes at `/workspace/schema/sqlite`, and explicit file, descriptor, call and modeled network limits, supplied through `GOMAD_TEMPORAL_CONFIG`. The README says outright that the CLI has no generic mount-capture/setup command, so arbitrary Temporal tests are not yet a one-command workflow.

The integration code that already captures those inputs exists as test fixtures: the Temporal guest configuration fixture in `internal/backends/wasm/wasi/measure_integration_test.go` and the one-box inspection fixture in `internal/backends/wasm/backend/temporal_inspection_integration_test.go`, which builds the guest with `backend.TemporalGuestBuildTags()` (`disable_grpc_modules`, `gomad_wasm_slim`, `sqlite3_dotlk`, `test_dep`) plus `gomad`. This spec turns that fixture knowledge into a consumer-facing command so that a Temporal test package can be prepared and explored from a clean checkout with one install command and one explore command.

This is the first step toward arbitrary functional tests on WASM. It does not widen what the guest can do; it removes the manual configuration step that blocks everything after it.

## Architecture & Data Models

**Derived configuration.** A new operator path, either a `gomad` verb or an extension of `install`/`discover` (task .1 decides and records the choice), derives the complete cooperative `wasi.Config` for a Temporal test package from the checkout. Inputs: the checkout root, the test package path, the pinned stock compiler, the helper and the runtime root (the same inputs `gomad install` already takes per `internal/backends/wasm/CLI.md`). Outputs: the cooperative installation with its configuration, carrying

- captured immutable SQLite schema bytes mounted at `/workspace/schema/sqlite`;
- guest working directory `/workspace/<package path>` and a matching `PWD`;
- writable `/workspace`, `/tmp` and the guest working directory;
- default file, descriptor, call and modeled network limits, with the defaults taken from the existing fixtures and recorded in the task;
- the slim guest build tags from `backend.TemporalGuestBuildTags()`.

**Ownership.** Host capture of immutable schema bytes and its single mapping definition belongs to `internal/backends/wasm/internal/temporalinput`, independent of WASI types. Complete guest policy assembly belongs to the pure `internal/backends/wasm/wasi` owner and consumes already captured inputs. Installation, measurement, inspection and qualification callers perform the shared host capture and pure derivation without post-hoc overrides; white-box WASI fixtures use the same definitions without an import cycle. Build-tag selection remains with `internal/backends/wasm/backend`, and operator grammar with `internal/backends/wasm/command` and `cmd/gomad/internal/cli`. Existing complete configuration bytes, capture errors and CLI contracts are preserved. This ownership repair follows the actual both-platform architecture failure found by integrated validation on 2026-10-20; the pure import boundary is not widened.

**Reuse.** The two integration fixtures are the implementation references and must end up calling the same derivation code rather than keeping a private copy, so that the fixture and the consumer command cannot drift.

## API Contracts

- One install-side command derives the configuration and pins the installation; one `gomad explore ... go-test <package> -- -test.run=^TestFrontendSystemInfo$` runs the test. No `GOMAD_TEMPORAL_CONFIG`, no hand-written JSON. The exact verb spelling and flags are decided in task .1 and documented in `internal/backends/wasm/CLI.md`.
- The derived configuration validates under the existing installation preflight and is refused with the existing status and message when the checkout lacks the schema source, the compiler or the helper.
- Existing hand-written configurations keep working unchanged; the derived path is additive.
- `gomad discover` on a derived installation reports the same shape as on a hand-written one.

## Edge Cases & Constraints

- Schema bytes are captured from the checkout and are immutable in the guest; a changed schema source changes the installation identity.
- Capacities are defaults, not guarantees. A test that exceeds them fails with the existing capacity termination; raising defaults is a recorded decision, not silent.
- The command must not read inherited environment configuration; explicit inputs only, matching the `wasmmeasure` precedent of refusing inherited measurement variables.
- No native path, no fallback, no new WASM capability.
- No CI run, PR or push is authorized by this spec; workflow files are untouched.

## Acceptance Criteria

- **R1:** From a clean checkout with the pinned stock compiler and a built helper, one install-side command produces a cooperative installation whose configuration carries the schema mount, working directory, `PWD`, writable roots, limits and slim build tags listed above, with no hand-written JSON and no `GOMAD_TEMPORAL_CONFIG`. Errors: a missing input is refused with the existing installation status and a message naming the input.
- **R2:** `gomad explore` on that installation runs `TestFrontendSystemInfo` in `tests/gomadfunctional` to a pass, with retained evidence, and `gomad replay` of the retained result succeeds. Errors: a pass that depends on a manual configuration step is a finding.
- **R3:** The two integration fixtures use the shared derivation code; an installation derived by the command and one derived by the fixture for the same inputs have the same configuration bytes. Errors: a divergence between the fixture and the command is a test failure.
- **R4:** `internal/backends/wasm/CLI.md` and `tools/gomad/README.md` document the one-command workflow and no longer state that arbitrary Temporal tests lack a setup command; neither claims support beyond the one named test. Errors: a support claim for other tests is a finding.

## Boundaries

- No new guest capability, no slim build tag changes, no change to what the WASM runtime refuses.
- No triage of other functional tests; that is the sweep spec.
- No qualification claim; the one-box cell is a separate spec.
- No native backend involvement.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-19): the native backend is retiring and WASM is the product, with the goal of arbitrary Temporal functional tests. The hand-written configuration is the first blocker on that path and already has working reference code in two integration fixtures, so the lowest-risk move is to promote that code to a command. Rejected: documenting the hand-written config as the supported workflow (keeps every later step manual); a generic mount-capture language (bigger than the request, no second consumer yet).

## Early proof point

Task .1 derives the configuration for `tests/gomadfunctional` from the fixture code behind a temporary flag and shows `gomad discover` accepting it. If the fixtures cannot be factored into shared code without changing the configuration bytes, stop and report before adding the verb.

## Quick commands

```sh
cd tools/gomad && go test -tags test_dep ./internal/backends/wasm/command ./internal/backends/wasm/installation ./internal/backends/wasm/backend
# after the verb exists (spelling decided in task .1):
/absolute/operator/bin/gomad <derive-verb> ... --output /absolute/operator/bin/gomad-temporal.json
GOMAD_WASM_INSTALLATION=/absolute/operator/bin/gomad-temporal.json /absolute/operator/bin/gomad explore --json --seeds=7 --parallel=1 go-test ./tests/gomadfunctional -- -test.run=^TestFrontendSystemInfo$
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | One install-side command derives the complete cooperative configuration. | fn-220-gomad-one-command-temporal-test-setup.1, fn-220-gomad-one-command-temporal-test-setup.2 | — |
| R2 | TestFrontendSystemInfo passes and replays from the derived installation. | fn-220-gomad-one-command-temporal-test-setup.2 | — |
| R3 | Fixtures and command share the derivation code with identical bytes. | fn-220-gomad-one-command-temporal-test-setup.1, fn-220-gomad-one-command-temporal-test-setup.4 | — |
| R4 | Docs describe the one-command workflow without over-claiming. | fn-220-gomad-one-command-temporal-test-setup.3 | — |
