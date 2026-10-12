# fn-170-clean-up-toolsumpire.1 One umpire CLI with subcommands replaces the eight cmd binaries

## Description
Replace the eight binaries under `tools/umpire/cmd/` (`umpire-assess`, `-fuzz`, `-gen-cases`, `-ir-bridge`, `-lint`, `-repeat`, `-replay`, `-run`; about 6,800 lines, seven on the standard `flag` package) with one `tools/umpire/cmd/umpire` main and a subcommand each (`umpire run`, `umpire lint`, `umpire gen-cases`, ...).

- Library: urfave/cli v2 (already in `go.mod`, used by `cmd/server`), unless the owner chooses cobra before this starts.
- Each subcommand keeps its own package under `cmd/umpire/`, mostly today's code moved; `tools/umpire/internal/cli` keeps the shared edge helpers (interrupts, one-line reports, error flattening).
- Flags, defaults, exit codes, stdout/stderr formats and receipts stay identical per subcommand; shared flags (timeouts, output paths, IR location) are declared once.
- Make targets keep their names and call the subcommands. Update every doc, script and CI reference to the old binary names.
- Build weight: keep heavy dependencies (Testpilot, Temporal SDK, server code) out of the light subcommands' packages, so only the main links everything; record cold and warm build times before and after.

## Acceptance
- One `umpire` binary; the eight old `cmd/umpire-*` packages are gone.
- Each subcommand's flags, exit codes and outputs match its old binary (a golden or table test per subcommand).
- Make targets unchanged in name and behaviour; docs updated; build times recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
