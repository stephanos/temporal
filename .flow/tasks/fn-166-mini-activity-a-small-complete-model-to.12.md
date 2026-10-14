# fn-166-mini-activity-a-small-complete-model-to.12 Model Make targets run without held API generation (cached API jar)

## Description
Owner decision, 2026-10-12 (part 2 of the change-scoped model loop; runs before fn-141; authorizes this Makefile change outside `model/`).

Problem: the scoped Make targets (`make model FEATURES=...`, `make model-mini`) first regenerate the API jar (`build/api-scalapb.jar`), which needs owner-held linked protobuf generation (`make protoc`). So every worker falls back to hand-written scala-cli commands, loses `FEATURES` scoping and re-derives its proof steps.

Do: make the model-side Make targets run without the held generation step whenever a usable API jar already exists (cached under `model/build/`, or produced once by an explicit opt-in target), and fail with one clear message naming the opt-in target when none exists. Never run the held protobuf/linked-API generation implicitly, never change Go or proto sources, and keep the canonical full route (with regeneration) available under its existing name for the sweep. Record the exact commands workers now use.

## Acceptance
- `make model FEATURES=activity_mini` (and `make model-mini`) completes its Scala stages without invoking held protobuf generation, using the cached API jar; Go-check stages that remain held are skipped with a named message, not reported as passing.
- With no API jar present, the target fails fast naming the opt-in target.
- Stage wall times recorded (compile, lift, equivalence) with warm caches.


## Done summary
Model Make targets run on cached IR/API jars; held linked-API generation never runs implicitly (commit 66d9ee8642).

Requirement evidence
- Cached jars: `$(MODEL_PROTO_JARS)` no longer have a regenerating rule; when missing they fail with "<jar> is missing; run make gen-model-api once (the held linked-API generation)". New opt-in `make gen-model-api` (gate `--generate-api --if-stale`). The loop (`modelLoop`) uses a new step "use the cached IR and API classes" (refuses missing jars or an IR jar whose stamp is older than the schema, naming gen-model-api). Canonical full route unchanged: `umpire-check-model`/`umpire-gen-model` still regenerate.
- `make model-mini MODEL_GATE_ARGS=--skip-lint` (loop-final.log): cached jars, compile 8s, package 18s, lift 4s, Go reader check skipped with "held: api/umpire/v1/claim.pb.go is older than the IR schema ...claim.proto; run make protoc"; then stops at "compare selected IR with checked-in outputs": checked-in activity_mini/generated/ir.json is behind (regeneration held), exit 2. Wall 37.8s. No protoc/getproto ran.
- Held Go stages (Go reader, IR lint, Case lowering) are skipped with `== skipped <stage> (held: <reason>)` and a closing `Scala stages passed; NOT run (held Go checks): ...` line; never reported passing.
- No-jar fail-fast (no-jar.log): model-mini, compile-model, lift-model, umpire-check-equivalence with an empty MODEL_BUILD each fail in 1-2s naming gen-model-api.
- Composable targets honoring FEATURES: compile-model (existing, now jar-gated), new lift-model (packages $(MODEL_JAR)+classpath, lifts into LIFT_OUT or fresh model/build/history/lift.*, reports per feature current/differs), umpire-check-equivalence (existing, jar-gated only). lift-model output byte-identical to the gate loop's lift.

Stage wall times, warm caches (stage-times.txt)
- compile-model FEATURES=activity_mini: 9.2s (9.8/11.3 earlier)
- lift-model FEATURES=activity_mini: 13.3s lift only (31.8s when it repackages the jar)
- umpire-check-equivalence FEATURES=activity_mini STEPS_ONLY=1 vs mini-selected-baseline-00d22597d2, fn160/task3/steps.map: 12.9-14.4s, equivalent (16 tables)
- make model-mini --skip-lint loop up to the compare: 37.8s. Loop + equivalence ~52s, under 90s.

Verification
- scala-cli test model/check: 121 pass, 1 fail (SyntaxRuleSuite repository sugar, inherited framework findings FacetSource/InputSource/RuleSource + activity_mini Entering.test block form). New tests: 3 GateSuite loop tests (cached jars/no held generation; missing/stale jar names gen-model-api; stale Go code skips named), 1 MakeFeaturesSuite (Make targets fail fast without jars, never regenerate with cached jars).
- go test ./tools/umpire/ir -run TestModelNamesNoRetiredFrontEnd: ok. scalafmt applied to new code only; git diff --check clean.

Failures retained (inherited)
- Loop lint stage fails on the same SyntaxRule/BlockForm findings (model-mini-1.log).
- Loop compare fails: checked-in mini IR is behind (held regeneration).
- lint-model-check: DisableSyntax.var at Gate.scala `var completed` and StepLedger.scala:346, both pre-existing.

Open obligations / deferred
- Go-check stages stay skipped until owner runs make protoc; gen-model-api itself not run (held).
- Full gates and review deferred to batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 66d9ee8642
- Tests: mise exec -- scala-cli test model/check --server=false (121 pass, 1 inherited SyntaxRuleSuite fail), make model-mini MODEL_GATE_ARGS=--skip-lint (cached jars, Go skipped held, stops at behind checked-in IR; 37.8s), make compile-model FEATURES=activity_mini (9.2s), make lift-model FEATURES=activity_mini (13.3s warm), make umpire-check-equivalence FEATURES=activity_mini STEPS_ONLY=1 (equivalent, 12.9-14.4s), no-jar fail-fast for model-mini/compile-model/lift-model/umpire-check-equivalence, go test ./tools/umpire/ir -run TestModelNamesNoRetiredFrontEnd
- PRs: