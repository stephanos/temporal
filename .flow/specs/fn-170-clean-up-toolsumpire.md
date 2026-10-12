# Clean up tools/umpire

## Conversation Evidence

> user (2026-10-12): "tools/umpire/cmd why are there separate packages instead of a single CLI (using cobra)"
> user (2026-10-12): "do we have a spec/tasks to cleanup the tools/umpire package?" Conductor: no open spec; earlier cleanups fn-124, fn-84, fn-61 and fn-66 are done. Proposed a research pass with the six Testpilot questions, then one cleanup spec.
> user (2026-10-12): "yes, add CLI consolidation task"

## Goal & Context

The Go Umpire tooling (`tools/umpire`) has grown by spec since its last cleanup (fn-124). It has eight separate command binaries, feature-specific code inside generic packages, and Quint-specific code inside the generic `export` package [inferred]. This spec collects its cleanup into one place. A read-only research pass (conductor, 2026-10-12) asks the six questions used for Testpilot: clear package jobs, layering, hidden internals, ease of understanding, tidiness, and deep modules. Its findings become this spec's further tasks [paraphrase].

## Architecture & Data Models

Task 1, the CLI, is known now; the rest follows the research.

## Edge Cases & Constraints

- Go work is held until the Scala source batch ends (MILESTONES "Current goal"); this spec runs in the Go propagation phase.
- Make target names stay stable; their recipes call the new subcommands.
- fn-169.4 holds the Temporal-adapter separation, the `export/quint` split and the `framework.dsl` IR name follow-up. Once the research lands, the conductor moves the `tools/umpire` items of fn-169.4 here, leaving fn-169.4 the IR-name follow-up and Testpilot.

## Acceptance Criteria

- **R1:** One `umpire` binary with subcommands replaces the eight `tools/umpire/cmd/umpire-*` binaries. Flags, exit codes and outputs are unchanged per subcommand. Shared flags and usage are declared once. Make targets keep their names.
- **R2+:** from the research pass.

## Decision Context

- 2026-10-12, owner: `tools/umpire` mirrors `model/`: generic code under `tools/umpire/framework/`, the Temporal adapter under `tools/umpire/temporal/` (.2); fn-169.4's `tools/umpire` items moved here.
- 2026-10-12, owner: the generic CLI is Temporal-free; API descriptors arrive as data; the binary's `main` in the Temporal adapter wires the Temporal target (Driver, SDK, in-process cluster). A large single binary is fine.
- 2026-10-12, owner: consolidate the commands into a single CLI. Library: urfave/cli v2, already a dependency the server uses, unless the owner chooses cobra. AGENTS.md admits a new third-party library only when specifically requested.

## Requirement coverage

| Req | Task |
| --- | --- |
| R1 | .1 |
| R2 (layout mirroring model/) | .2 |
