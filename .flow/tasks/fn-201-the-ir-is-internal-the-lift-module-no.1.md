---
satisfies: [R1]
---
# fn-201-the-ir-is-internal-the-lift-module-no.1 framework/export becomes framework/lift; no file written by default

## Description
Rename the exporter package and directory to `framework.lift`, make it package-private to the framework, return an in-memory `Lifted` from `Lift.of(instance)`, and keep a `Lift.debugWrite` for inspection only. Keep every capture diagnostic.

**Files:** model/framework/export/ -> model/framework/lift/, model/framework/reify/plugin (output path option), Makefile MODEL_TOOL_DIRS and scalafmt paths, fn-198 case-folder runners.

**Quick commands:** `mise exec -- scala-cli compile --server=false model/framework/lift`; `make check-framework`

## Acceptance
- No gate writes IR by default; the debug write works on request.
- Capture diagnostics unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
