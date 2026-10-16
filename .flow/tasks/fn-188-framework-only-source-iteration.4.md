---
satisfies: [R4, R5]
---
# fn-188-framework-only-source-iteration.4 Include native sample suites and explicitly hold Go-dependent cases

## Description
Implement the R5 discovery/execution-boundary prerequisite and verify the reached R4 artifact contract. This is partial R4/R5 work, not the successful full gate owned by .2. Native fn-188.3 and fn-147.5 closures are committed; no new native feature or exporter semantics are admitted.

**Size:** M
**Files:** model/framework/check/FrameworkLoop.scala, model/framework/check/Samples.test.scala, model/framework/check/FrameworkLoop.test.scala, model/README.md
**Touches:** [model/framework/check/FrameworkLoop.scala, model/framework/check/Samples.test.scala, model/framework/check/FrameworkLoop.test.scala, model/README.md]

### Approach

1. In FrameworkLoop.scala:19, add model/framework/samples to the existing core Module's discovery roots and executable test inputs, not ModelSources.frameworkCore's reusable-library inputs. Reuse Registry.declared and exact JUnit selection. Discover framework.dsl.SampleExpectTest, NativeMapsTest, NativeVotingTest and NativeHistoryTest once each as core suites; their package is not framework.samples.
2. In Samples.test.scala:542/691, hold only the two real-Go registrations in framework mode (presence of UMPIRE_FRAMEWORK_BUILD), before either body can run. The exact names are `the generated Go checker overlaps at most two checks, joins failures and orders diagnostics` and `the generated parallel checker preserves complete real sample reports`. Follow export/ModelBuild.test.scala:165's HELD-message/munit.Ignore pattern with a private check-local helper; do not introduce a check-to-export dependency or a configurable hold registry. Preserve both complete bodies/assertions and the real-report body's existing assume. Ordinary non-framework registrations remain runnable.
3. Extend FrameworkLoop.test.scala's existing scratch discovery, stubbed command and record tests. Assert all four names/input paths, no duplicate selectors, and that reusable core compilation excludes samples. Preserve poisoned-classpath, zero-executed-tests and failed-command refusals. Feed both HELD lines (including duplicate output) through the existing record path and prove explicit reasons are retained without pass credit.
4. Add registration-only controls inspecting the exact two SamplesSuite munitTests options. Execute the control in separate framework/non-framework environments: Tools.withEnvironment does not change the current JVM's sys.env. Under framework mode, poison SAMPLE_REPORT_INPUT/BASE and prove both cases still ignore before bodies execute; no other SamplesSuite case is silently held. In ordinary mode inspect registration only, never execute either Go body. Use an existing focused suite, not a new permanent verifier.
5. Document sample coverage and explicit Go holds in model/README.md's framework-only loop section. Keep .2's complete successful supported-generic gate outstanding; historical94/20/13 counts are not current counts. No assertion, golden or fixture is weakened to make this prerequisite green.

### Existing artifact baseline

At source HEAD2f5aca4157, root freshly compiled framework DSL/realize to framework.jar and packaged the current compiler plugin with the existing IR jar; both exited0. Normal strict export/test compilation also exited0. Logs: .flow/tmp/framework-followup-{core,plugin,export-normal}-probe.log. Scratch path is recorded by .flow/tmp/framework-followup-probe-path.txt. Preserve these observations; they are compile/build evidence, not semantic suite success.

The attempted export compile using a relocated --workspace exited1 before compilation because exclude directives no longer belonged to its project root; retain framework-followup-export-probe.log as a probe-configuration failure, not a product defect. Do not change exporter directives or add a compatibility feature to fix that invalid probe. Separate-library macro recipes still require their documented fresh workspace isolation.

### Verification

- Run the existing focused FrameworkLoopSuite and BoundarySuite via the check project's JUnitCore entrypoint in framework mode; run the registration controls in a separate ordinary-mode invocation selecting only FrameworkLoopSuite, not SamplesSuite. Report actual counts/exits for each mode.
- Run the four explicitly selected native sample suites from dsl/realize/samples through the existing JUnitCore entrypoint with --server=false --test. Prove nonzero executed tests; do not infer success from selector strings alone.
- Under an explicit fresh UMPIRE_FRAMEWORK_BUILD, run existing framework.export.FrameworkFixtureSettingsSuite to exercise fresh-library substitution, held mini access and poisoned eligible-package refusal. Reuse the actual current-source artifact commands if source changes invalidate the baseline; never use cached system/API/Model jars or unset production fallback.
- Run reached boundary/lint/format checks and git diff --check. Keep one compiler lane. Broad generic/export and semantic Class/default proofs remain in .2/.174.24, not here.

### Boundaries

No model/temporal reads, restoration or disabled-body consumption. No model/project.scala, Temporal/API/Model jars, Go execution or changes, schema generation, regeneration, full exporter/IR semantic activation, new command/proof harness/hash manifest, parent-spec completion or automatic release of .2/.174.24. The original failed/interrupted .2 evidence remains intact.

## Acceptance
- [ ] All four native sample suites are discovered exactly once as core and are present in executable inputs/selectors, while reusable-library inputs remain DSL/realize only.
- [ ] Only the exact two Go/report cases receive explicit HELD reasons and munit.Ignore before body execution in framework mode, including poisoned report variables; non-framework registration behavior and both bodies/assertions remain unchanged.
- [ ] Existing scratch/stub regressions prove selectors, artifact/boundary checks, HELD recording/deduplication and refusal of zero-test/nonzero-exit success. No check-to-export dependency or new harness is introduced.
- [ ] Actual selected native samples, focused runner/boundary controls in both environment modes and existing fresh-library settings controls pass with positive counts; current-source compile/build evidence and reached lint/format checks are recorded honestly.
- [ ] README documents coverage/holds and the receipt leaves full R4/R5, .2's terminal successful generic gate, .174.24 semantics and all Temporal/Go/backend obligations open. Held/skipped cases earn no pass credit.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
