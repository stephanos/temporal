---
satisfies: [R1]
---
# fn-117-type-the-temporal-api-in-the-models.1 Prove the typed API toolchain and author surface before choosing generation

## Description
Implements R1 before any other implementation. Resolve the descriptor and selector feasibility gaps, measure the two generation alternatives, and leave a reviewable report for the user to commit.

**Size:** M (bounded spike; stop at the mechanism decision)
**Files:** .plans/UMPIRE_TYPED_API_SPIKE.md; .flow/tmp/fn117-1/**; cmd/tools/getproto/{files.go,main.go} only if the existing descriptor input must be completed.
**Touches:** [.plans/UMPIRE_TYPED_API_SPIKE.md, .flow/tmp/fn117-1/**, cmd/tools/getproto/**]

### Approach
- Reuse fn-113 task16 and task3 completion-fix passing evidence for unaffected inputs. Read the current gate's ScalaPB toolchain and the pinned protoc/Scala/JDK settings. Capture hashes of the six current IRs, positive fixture outputs and Cases before probing.
- Establish the descriptor closure before claiming generation works: api.binpb omits workflowservice/v1/service.proto although Go links that descriptor. Reuse getproto's descriptor assembly to include the service and its imports, or record an equally deterministic repository-owned input derived from the same linked descriptors. Include Testpilot and required well-known messages. A missing input fails with its existing make target named. Do not alter Go lowering or create a broad API drift/CI gate.
- Use scratch sources under the task directory, not edits to the user's Models, for the one-line-edit benchmark. Measure today's warm Model compile with the same command/environment as the candidate. Report generated-jar cold construction time, jar size, warm Model compile after the edit and whether generation/compilation was reused. Distinguish dependency-download time from cold jar construction. Do not clear shared caches or run heavy gates concurrently.
- Generate/compile with ScalaPB first. Prove a nested optional-message selection, generated enum and message type, AND a unary service method with correctly typed request/response selections can compile and lift to today's exact proto names/paths. Prove enough repeated/oneof and typed map-message shape to choose honest editor-completable spelling; the sketch's _.taskQueue.name is not presumed to work with ScalaPB Option fields. No Scala Model builds or sends a message.
- If the full API exceeds twice the measured warm baseline, try the referenced-message closure (including required service/method metadata and imported fields). If still too slow, or generation/name recovery fails, exercise the typed-catalog fallback and record why. Cold construction is reported, not a rejection threshold. Keep the same public typing contracts in either outcome.
- Record the chosen mechanism, exact author spelling constraints, complete descriptor inputs, measured commands/exits/wall times/size, fallback outcome, and names recovered from actual compiled trees. Preserve current generated artifacts; the report and existing spec are the decision boundary for task2.

### Investigation targets
**Required:**
- model/gate/Gate.scala:63-235
- cmd/tools/getproto/files.go:45-101 and cmd/tools/getproto/main.go:35
- Makefile:114-123,337-343,684-725
- model/umpire/Action.scala:72 and model/umpire/realize/Realize.scala:256-345
- model/lifter/Realizations.scala:1-36
- model/lifter/test/Fixtures.test.scala:56-64
- tools/umpire/lower/descriptor.go:53-135

### Quick commands
Inspect the existing descriptor assembly target before invoking it; mise exec -- protoc --version; mise exec -- scala-cli version; mise exec -- scala-cli compile model/lifter; scratch compile/lift probes and comparable timed Model compile commands recorded in the report. If getproto source changes, run its established focused tests with -tags test_dep and make lint-code-fast against origin/main.

### Execution constraints
No staging, commits, push, worktrees, recursive deletion or shared-cache clearing. The user owns the report commit. Preserve comments. Baseline/verification reuse follows MILESTONES.md; logs and evidence stay under .flow/tmp/fn117-1/. No backend comparison, owner-deferred. No production author API or Model migration before the report passes review.

## Acceptance
- [ ] The report proves ScalaPB compilation and compiled-tree recovery of nested field, enum, message and unary method/request/response names from a complete descriptor closure, or records and proves the required fallback.
- [ ] Cold jar time/size and equivalent warm baseline/candidate one-line-edit times are recorded; the selective-generation and twice-baseline fallback rules are applied explicitly.
- [ ] Current IR/Case/expected outputs remain unchanged, probe evidence is reproducible, and any descriptor-tool change has focused tests and lint.

## Done summary
ScalaPB generated and compiled the complete linked API, Testpilot and Umpire descriptor closure. A compiled TASTy probe recovered the existing proto message, field, enum and typed unary method names. The warm full-Model edit stayed below the two-times limit, so the report selects the full ScalaPB jar; all current IR, Case and positive fixture hashes remained unchanged.

The recorded independent implementation review reached SHIP after both findings were fixed. The task references `.flow/tmp/fn117-1-review/receipt.json`, which is absent from the current checkout; this reconciliation relies on the tracked summary and evidence and does not claim to have read that receipt.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline green via .flow/tmp/fn113-16-evidence.json and .flow/tmp/fn113-3-completion-fix/{lifter-test-result.json,check-model-result.json}, mise exec -- protoc --version (exit 0), mise exec -- scala-cli version (exit 0), mise exec -- scala-cli compile model/lifter (exit 0), mise exec -- go run .flow/tmp/fn117-1/descriptors.go proto/api.binpb .flow/tmp/fn117-1/api-complete.binpb (exit 0), mise exec -- protoc --plugin=protoc-gen-scala=model/gen/protoc-gen-scala --descriptor_set_in=.flow/tmp/fn117-1/api-complete.binpb --scala_out=flat_package,scala3_sources,grpc:.flow/tmp/fn117-1/generated-cold <descriptor-files.txt> (exit 0), mise exec -- scala-cli --power package --library .flow/tmp/fn117-1/generated-cold --scala 3.9.0 --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -f -o .flow/tmp/fn117-1/api-scalapb-cold.jar (exit 0), mise exec -- scala-cli compile model/project.scala model/umpire .flow/tmp/fn117-1/temporal-copy (primed then one-line Model edit; exit 0), mise exec -- scala-cli compile model/project.scala model/umpire .flow/tmp/fn117-1/temporal-copy --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 (primed then one-line Model edit; exit 0), mise exec -- scala-cli --power package --library .flow/tmp/fn117-1/Probe.scala --scala 3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -f -o .flow/tmp/fn117-1/probe-cold.jar (exit 0), mise exec -- scala-cli compile .flow/tmp/fn117-1/Invalid.scala --scala 3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-grpc.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 (expected exit 1), mise exec -- scala-cli run .flow/tmp/fn117-1/Inspect.scala --scala 3.9.0 --dep org.scala-lang::scala3-tasty-inspector:3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --jar .flow/tmp/fn117-1/probe-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -- .flow/tmp/fn117-1/probe-cold.jar (exit 0), mise exec -- scala-cli --power package --library .flow/tmp/fn117-1/ProbeChanged.scala --scala 3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -f -o .flow/tmp/fn117-1/probe-changed.jar (exit 0), mise exec -- scala-cli run .flow/tmp/fn117-1/Inspect.scala --scala 3.9.0 --dep org.scala-lang::scala3-tasty-inspector:3.9.0 --jar .flow/tmp/fn117-1/api-scalapb-cold.jar --jar .flow/tmp/fn117-1/probe-changed.jar --dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20 -- .flow/tmp/fn117-1/probe-changed.jar (expected exit 1), diff -u .flow/tmp/fn117-1/artifacts-before.sha256 .flow/tmp/fn117-1/artifacts-after.sha256 (exit 0), current48 source/evidence hashes verified against review snapshot (exit 0); current30 artifact hashes verified (exit 0), git diff --check (exit 0), Independent implementation review SHIP; both findings fixed
- PRs: