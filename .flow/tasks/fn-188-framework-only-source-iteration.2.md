---
satisfies: [R4, R5]
---
# fn-188-framework-only-source-iteration.2 Establish the framework-only build and verification loop

## Description
Current admission: this broad exporter/verification task depends on fn-188.3 and remains separately owner-held. Native DSL implementation is already committed and the next work is the native final public-client/batch-review gate, not this task. Completing that gate does not authorize resumption here. The original description below records the interrupted loop's intended scope and evidence obligations; its statement that DSL changes come later is superseded.

Touches: Makefile, model/framework/**, model/README.md, .flow task/spec lifecycle. Reuse framework.check Registry/Module/Tools and Boundary rather than creating a parallel harness. Add one documented framework-only entrypoint, package the fresh generic framework and compiler plugin without model/project.scala, Temporal source/API/Model jars, and classify every discovered framework suite honestly as generic or held integration. Run the actual generic suites; preserve all original checks. Existing three uncommitted framework edits from interrupted fn-174.2 are adopted within this task and must be verified/preserved. Do not read or edit model/temporal, its disabled sources or generated payloads; no schemas, Go, tools/umpire, backend or regeneration work. Keep one build lane. After this loop is green, author-facing DSL changes will use it; implementing expect/record inputs belongs to later tasks.

## Acceptance
A documented command builds/tests the framework with Temporal entirely excluded; no stale Temporal artifacts or API dependencies enter generic fixtures; every discovered framework suite is accounted for with explicit held integration reasons; boundary/command regression tests and the actual framework gate pass; current uncommitted framework repairs are safely included or preserved with evidence; no Temporal source is restored or consumed.

## Done summary
Blocked:
OWNER HOLD: native DSL specifications now take priority. The owner explicitly defers IR/exporter/Go work and ordered termination of the owned full framework gate before its final summary. The gate is not green: core251/check91 passed; export had interleaved JUnit error markers and was interrupted with exit143. No pass receipt or task completion is authorized.

Implementation is preserved in checkpoint437016b7d148b87cccee78c926af92119dec7fb4. Sources remain unchanged and the owned PGID670307 has no survivors. Full source/artifact, failed-run, focused-test and terminal-process evidence is in .flow/tmp/fn1882-final-summary.md and .flow/tmp/fn1882-final-evidence.json. Resume IR/export diagnosis and the required successful full gate only when the owner reauthorizes that lane; native DSL work is now the priority.
## Evidence
- Commits:
- Tests:
- PRs:
