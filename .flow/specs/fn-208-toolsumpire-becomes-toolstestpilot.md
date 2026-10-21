# `tools/umpire` becomes `tools/testpilot`

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-18): "then we could rename tools/umpire to tools/testpilot and have that Go code only deal with that part?"
> owner (2026-10-18): approved filing as the last spec of the series.

## Goal & Context

Once refinement (fn-199), lint (fn-200), the internal IR (fn-201), the framework schema (fn-202), Case production (fn-203), the conformance descriptor (fn-204) and the backends (fn-206, fn-207) have landed, no Go code reads a Model. What remains is the Testpilot runtime: the Driver, the live commands, the campaign and replay coordinators, the repeat harness, and a Contract-and-descriptor assessor. This spec deletes every Model-reading Go package, moves the surviving commands under `tools/testpilot`, rewires the Make targets, module map and boundary tests, amends the SPEC and Vision clauses that required Go replay, and closes fn-196.4 and fn-196 R5 by reference.

## Architecture & Data Models

- Deleted: `ir`, `interp`, `internal/engine`, `check`, `realization`, `lower` and its producer, `explore`, `lint`, `export`, `internal/runtimecel`, and the Model-reading half of `conformance`; `umpire-gen-cases`, `umpire-ir-bridge`, `umpire-lint`, `umpire-check-samples`.
- Moved: `umpire-run`, `umpire-fuzz`, `umpire-replay`, `umpire-repeat`, `umpire-assess` and `internal/cli` to `tools/testpilot/cmd` with the `testpilot-` prefix; the descriptor interpreter stays in `common/testing/testpilot`.
- Make: `umpire-*` targets that read a Model are removed; the live and repeat targets are renamed; the model gate becomes the framework check.
- Docs: the tools README, the module map, the model README's pipeline section and the measurement note's obligation table say the series closed; the Direction in MILESTONES says one language reads Models.
- Boundary test: `tools/testpilot` imports nothing under `model` and nothing that reads a Model; `common/testing/testpilot` imports nothing from `tools`.

## API Contracts

- `go build ./...` and `go test -tags test_dep ./tools/testpilot/... ./common/testing/testpilot/...` pass with no IR, no JVM and no server.
- The live commands keep their flags and exit codes; `--model` on run and assess takes the descriptor-bearing Case, not an IR path.
- The campaign and replay bridge protocols are unchanged; the Scala bridge server is their only producer.

## Edge Cases & Constraints

- The SPEC and Vision require Go replay of backend counterexamples; the amendment names `Checker.replay` as the replacement and records the owner's authorization.
- Generated Go bindings for the IR proto are deleted with the proto's last Go reader.
- The canary case binding reads Cases only and is unaffected beyond paths.

## Acceptance Criteria

- **R1:** Every listed Go package and command is deleted and no gate references it; `go build ./...` passes. Errors: none beyond the build.
- **R2:** Surviving commands live under `tools/testpilot/cmd` with the new prefix, their flags and exit codes unchanged, and the Make targets renamed; the live checks pass against a deployment. No error surface.
- **R3:** Boundary tests hold `tools/testpilot` to no Model imports and `common/testing/testpilot` to no `tools` imports. Errors: a violating import fails the test naming the path.
- **R4:** The SPEC, Vision, tools README, module map, model README and measurement note are amended; MILESTONES' Direction says one language reads Models and fn-196.4 and fn-196 R5 are closed by reference. No error surface.

## Boundaries

- No change to Testpilot runtime behaviour.
- No Temporal Model migration; that is the owner's sweep.
- No Rust or other-language port.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

Deleting at the end rather than per spec keeps each earlier spec's gate green and lets the Go packages serve as the byte-equality reference until fn-203's recut. The rename makes the remaining tree's purpose its name. Depends on fn-200, fn-203, fn-204, fn-206, fn-207; fn-205 is deferred and not a dependency. Filed 2026-10-18 as the last of the one-language series (fn-199 through fn-208).

2026-10-18, owner decision (R4): the SPEC and Vision clause requiring Go replay of backend counterexamples is amended to: a backend counterexample replays through Checker.replay, and TLC's exhaustive reachable set (fn-207 R4) must agree with the native graph on the gate's Models. Independence moves from one replayed path to the whole graph read by a different engine.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.
