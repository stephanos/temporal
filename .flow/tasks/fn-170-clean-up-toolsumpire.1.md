# fn-170-clean-up-toolsumpire.1 One umpire CLI with subcommands replaces the eight cmd binaries

## Description
**Recut 2026-10-18 (one-language series):** Only run, assess, replay, repeat and fuzz remain as subcommands; gen-cases, lint and ir-bridge are gone (fn-203, fn-200, fn-208). fn-208's `tools/testpilot` rename owns the CLI layout; coordinate there.

Replace the eight binaries under `tools/umpire/cmd/` (`umpire-assess`, `-fuzz`, `-gen-cases`, `-ir-bridge`, `-lint`, `-repeat`, `-replay`, `-run`; about 6,800 lines, seven on the standard `flag` package) with one binary and a subcommand each (`umpire run`, `umpire lint`, `umpire gen-cases`, ...).

- Library: urfave/cli v2 (already in `go.mod`, used by `cmd/server`), unless the owner chooses cobra before this starts.
- Each subcommand keeps its own package under `tools/umpire/framework/cmd/` (after fn-170.2), mostly today's code moved; `framework/internal/cli` keeps the shared edge helpers (interrupts, one-line reports, error flattening).
- Flags, defaults, exit codes, stdout/stderr formats and receipts stay identical per subcommand; shared flags (timeouts, output paths, IR location) are declared once.
- Make targets keep their names and call the subcommands. Update every doc, script and CI reference to the old binary names.
- Build weight: keep heavy dependencies (Testpilot, Temporal SDK, server code) out of the light subcommands' packages, so only the main links everything; record cold and warm build times before and after.

**Temporal-free generic CLI (owner, 2026-10-12).** A large single binary is fine; generic code depending on Temporal is not. Today the commands depend on Temporal three ways:
1. Model-only commands (`umpire-lint`, `-gen-cases`, `-ir-bridge`) blank-import `go.temporal.io/api/workflowservice/v1` only so the protobuf registry resolves the Temporal message names the IR's realizations use.
2. Case-running commands (`umpire-run`, `-assess`, `-fuzz`, `-replay`, `-ir-bridge`) import Testpilot's Temporal Driver (`testpilot/temporal`, `binding`).
3. `umpire-run` and `umpire-repeat` link the server: `tests/testcore`, `tests`, `service/history/tasks`, `history/consts`, `matching/counter` (an in-process test cluster).

Target:
- The generic `umpire` CLI (subcommand packages and shared plumbing) imports no Temporal API, SDK, `service/...` or `tests/...`. It resolves API messages from a descriptor-set file named with the IR (or passed by flag), not linked-in generated code. It defines a small target interface (a Testpilot Driver plus provisioning, such as starting an in-process cluster) for the Case-running subcommands.
- The binary's `main` lives in the Temporal adapter at `tools/umpire/temporal/cmd/umpire`. It composes the generic commands with the Temporal target: the Testpilot Temporal Driver, the Go SDK and the `tests/testcore` cluster.
- An import rule (a dependency-boundary test like Testpilot's) enforces the separation.
## Acceptance
- One `umpire` binary; the eight old `cmd/umpire-*` packages are gone.
- Each subcommand's flags, exit codes and outputs match its old binary (a golden or table test per subcommand).
- Make targets unchanged in name and behaviour; docs updated; build times recorded.
- Generic CLI packages import no Temporal API, SDK, `service/...` or `tests/...` (enforced by a boundary test); API descriptors come from a descriptor-set file; the Temporal target is wired only in the adapter's `main`.
## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: