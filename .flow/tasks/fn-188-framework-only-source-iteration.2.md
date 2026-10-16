---
satisfies: [R4, R5]
---
# fn-188-framework-only-source-iteration.2 Establish the framework-only build and verification loop

## Description
Touches: Makefile, model/framework/**, model/README.md, .flow task/spec lifecycle. Reuse framework.check Registry/Module/Tools and Boundary rather than creating a parallel harness. Add one documented framework-only entrypoint, package the fresh generic framework and compiler plugin without model/project.scala, Temporal source/API/Model jars, and classify every discovered framework suite honestly as generic or held integration. Run the actual generic suites; preserve all original checks. Existing three uncommitted framework edits from interrupted fn-174.2 are adopted within this task and must be verified/preserved. Do not read or edit model/temporal, its disabled sources or generated payloads; no schemas, Go, tools/umpire, backend or regeneration work. Keep one build lane. After this loop is green, author-facing DSL changes will use it; implementing expect/record inputs belongs to later tasks.

## Acceptance
A documented command builds/tests the framework with Temporal entirely excluded; no stale Temporal artifacts or API dependencies enter generic fixtures; every discovered framework suite is accounted for with explicit held integration reasons; boundary/command regression tests and the actual framework gate pass; current uncommitted framework repairs are safely included or preserved with evidence; no Temporal source is restored or consumed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
