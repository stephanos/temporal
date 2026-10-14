# fn-166-mini-activity-a-small-complete-model-to.14 Lifter suites compile each fixture once per run and lift in a shared JVM

## Description
Conductor decision under the owner's standing instruction to follow its own recommendations, 2026-10-12. This is part 5 of the change-scoped loop proposal. It runs before fn-169.1 and fn-141, whose tasks all change the lifter.

Measured (fn-166.13): a lifter change runs all 36 lifter suites in 633–901 s (723 s typical), and a mini-only change spends 340 s in 11 mini lifter suites. Most of this is per-test fixture builds: each test copies a fixture, compiles it with its own `scala-cli --server=false` and lifts it in a fresh JVM (`model/irgen/test/Fixtures.test.scala`: "A test builds a fixture with scala-cli and lifts it in a JVM of its own"). The heaviest single tests build whole projects only to see them refused (50–77 s each).

Do:
1. Compile each fixture project once per test run, shared across every test that reads it (a run-scoped cache keyed by the fixture's source hash), instead of per test.
2. Lift in-process, or in one long-lived lifter JVM per run, wherever a test only needs the lift output (`InProcess.lift` already exists for some suites).
3. Batch the compile-refusal tests: compile many deliberately broken sources in one scala-cli invocation where the diagnostics can still be attributed per file and line, or check them through the compiler API in-process.
4. Keep every assertion and refusal message; any test whose meaning would change is left as is and listed.
5. Measure the full `model/irgen` suite and `make model-check-changed` on a lifter-only change and on a mini-only change, before and after. Target: a lifter change under 5 minutes.

## Acceptance
- Fixture projects compile once per run; lifts reuse one JVM or run in-process where possible; refusal tests batched where attribution is kept.
- Same tests and assertions pass (274+ pass, 0 fail, the recorded sweep-held skips).
- Before/after wall times recorded for the full irgen suite and for model-check-changed on a lifter change and a mini change.


## Done summary
Lifter fixtures build in-process, once per run (commit c3865a21a9). The 5-minute target was missed. The coordinator closed the task after one measurement round.

Requirement evidence
- 1 (compile once per run): FixtureBuild in model/irgen/test/ModelBuild.scala extends the fn-166.11 helpers. `tools.buildFixture(args)` stands in for `tools.scalaCli(args)`. It reads `--server=false` `package --library` / `compile` command lines (`--test`, `--classpath`, `--jar`, `--scala`, `--jvm`, one directory or one .scala file, and the directives scala/jvm/options/jar plus the Models' one dep). It compiles with the scala3 compiler on the lifter's test classpath, using scala-cli's workspace as `-sourceroot`. It caches per run by sources (relative path and content), options and classpath, so identical copies compile once. It prints the compiler's report as scala-cli's compiler JVM does (ASCII, with `?` for other characters). It falls back to scala-cli for any other command line.
- Faithfulness: a temporary compare suite built every testdata project both ways (46 builds, `.flow/tmp/fn166/task14/compare-equal.log` and `FixtureBuildCompare.test.scala.txt`). Exit codes, output and every jar entry (classes and TASTy) were identical. A per-call verify mode inside the real suites gave 116 identical results. One intermittent `.class` byte difference appeared in an enum anon class (`Phase$$anon$2.class`). The TASTy was identical, and a rerun of the same suite was identical, so this is compiler nondeterminism, not a meaning change.
- 2 (lifts): lifts were already in-process (InProcess). There are now three lift slots instead of two, still within -Xmx3g. Peak test-side java RSS was about 5 GB. The Bloop server was already running at about 3 GB.
- 3 (refusals): the build-refusal tests that read Bloop's `[error] path:line:col` report (Fixtures.refusals, retiredRuleCase) are left on scala-cli's Bloop server. They are already served by one warm compiler (0.5 to 7 s each), and Bloop's report and its zinc diagnostics differ from the compiler's own. Changing that would change their attribution format, so they are listed as left as is. The `--server=false` refusals (crossed, werror, typed*, ...) now run in-process with identical output.
- 4: no test or assertion was dropped. Six package call sites that only need a jar now pass `--server=false` (Fixtures.packaged, QualifiedNames, Conventions, DefaultEnds, PhaseCapabilities, StatusTableSeam). Each only orFail()s, and their builds are identical. Bloop's zinc "No class ... defined" check no longer applies to them. Other suites already ran with `--server=false`.
- Still on scala-cli: GapResolution/PropertyGroups Model packages (multi-input), testdata/gaps (`using file`/`exclude`), Facets `scala-cli run`, coreDiscovery, and the Bloop refusals.

Measurements (the machine is shared; another session's load was 4 to 12)
| run | before | after |
| full `scala-cli test model/irgen --server=false` | 672 s (274 pass, 6 sweep-held) | 425 s (274 pass, 0 fail, 6 held) |
| `make model-check-changed`, lifter change | 798 s (fn-166.13 record) | 506 s (lifter suites 401 s, record .flow/tmp/checks/20261012-175633) |
| `make model-check-changed`, mini-only change | 463 s (fn-166.13) | not re-measured (closed by the coordinator) |
Missed: under 5 minutes for a lifter change.

Remaining cost, top 8 lifter suites by summed test time (after, s), and what dominates each
- Fixtures 115: about 244 in-process lifts (about 225 s lift time, three slots), plus 15 Bloop refusal compiles (about 44 s)
- ConventionsFixtures 36: `java lintConventions` subprocesses (tools.run), not builds
- WitnessFixtures 31: 58 lifts (about 27 s)
- FacetsTest 29 (was 137): 33 lifts plus one `scala-cli run` (native rows)
- QualifiedNames 22 (was 59): three in-process builds of the large lifts fixture (about 17 s), 10 lifts
- HintSnapshotsSuite 21 (8 before; noisy): lifts
- GapResolutionSuite 20: scala-cli fallback packaging of model/framework and friends, plus `--print-class-path`
- ExpectationFixtures 14: 24 lifts
Across the run there were 403 lifts at about 0.8 s each (each reads the whole Models jar's TASTy). Also outside the suite times: about 50 s compiling the irgen test project, and about 90 s of check stages.
Follow-up ideas: run lifter suites side by side in one JVM (a bounded JUnit Computer, with the global build and lift slots keeping memory bounded), make each lift cheaper (shared TASTy reading), and run lintConventions in-process.

Verification
- Full irgen suite: exit 0, 274 pass, 0 fail, 6 sweep-held (irgen-after3.log; also after2 at 519 s).
- `make model-check-changed`: exit 0, lifter 275 tests, check 27 tests, held committed-IR item only.
- scalafmt --check: clean on the changed files that were clean before. Fixtures.test.scala and LayoutFiles.test.scala were already unformatted at HEAD.

Failures retained / open: none new. Coverage.scala is unchanged (no suite added or renamed).
Deferred gates: lint-model-irgen scalafix (inherited debt), full gates and review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c3865a21a9
- Tests: scala-cli test model/irgen --server=false: 274 pass, 0 fail, 6 sweep-held, 425s (before 672s), make model-check-changed (lifter change): exit 0, 506s (before 798s), record .flow/tmp/checks/20261012-175633, FixtureBuildCompare: 46/46 fixture builds identical to scala-cli
- PRs: