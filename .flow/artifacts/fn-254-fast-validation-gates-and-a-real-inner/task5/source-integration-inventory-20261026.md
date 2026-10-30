# fn254.5 source integration inventory

The conductor can recover .5 from five isolated commits after .6 closes and its closure reaches `umpire`. The source assembly contains unrelated tasks and an unresolved merge. Preserve that checkout and integrate only the ranges below into the current `umpire` branch. This inventory ran source/Git inspection only. It ran no builds, tests, formatting, lint, benchmarks, Git mutations or Flow commands.

The inspected root was `fa1ccb8af1`; assembly HEAD was `7cf05d1c53`. Read AGENTS.md, model/README.md, UMPIRE4 specification/vision/module map and the task .5 contract. Report prose follows `/home/agent/.codex/skills/flow-next-prose/SKILL.md` and its prose contract.

| .5 change | Assembly commit | Original commit | Scope |
| --- | --- | --- | --- |
| 24 independently scheduled framework fixture cases | ac7f099e2300cf152dfc0974ba4220fea5ebb005 | 51c7f8c3ef9d47325f1da16783a50571d5fad4e5 | dev/fixtures/{build.rs,fixtures.rs}; dev/testing/{src/lib.rs,src/runner.rs,tests/consumers.rs} |
| Separate compiler failure/pass batches | d6d8915370465a0121638bbf06158494e9084855 | fb664e116a1c3e257ccc290e5e4799e106749f0b | temporal/realization/tests/{registry.rs,supplemental_admission.rs} |
| 63 parser conversions; 16 retained compiler/wrapper cases | f3a9c6334cd3fad86d1e8e8b67157fce4083eefb | 383b16f8a1de7b28b1cef285eb9849c86596ee47 | author/dsl/macros/tests/fixtures.rs and nested parser_refusals; 126 byte-preserving input/stderr moves |
| TLC discovery default heap cap | d6f0b430a7aad6adb3595c74a6d1952e70839dbf | b4b9b05cdefa4874d54c9799b536a8dcb14a0845 | backends/tlc/src/runner.rs only |
| Remove two Scala Make targets | b482cbd9db7d2cf56075aed75e80075532830cf1 | same | Makefile deletion of model-compiler-inputs/model-check-changed block only |

Framework paths above are relative to `model/umpire`; Temporal paths are relative to `model`. Stable Git patch IDs match for all four original/assembly pairs. The fixture, compiler and parser paths have no subsequent committed edits in assembly. TLC runner has later .16 provisioning/shell-leaf edits, and Makefile has subsequent gate/Scala changes. Use the listed commits' patches, never those two files from assembly HEAD.

The minimal commit sequence is `ac7f099e23 d6d8915370 f3a9c6334c d6f0b430a7 b482cbd9db`. These commits contain no Flow lifecycle or milestone changes. A successful patch application would establish source integration only; acceptance evidence and remaining concurrency work still follow.

Two additional .5 configuration changes are embedded in mixed commit `a28c45400e7e77c294af742cf3c9d16a419a7797`. Extract only its complete diffs for `model/.config/nextest.toml` and `model/umpire/.config/nextest.toml`. Those change JVM max-threads from 1 to 2 and add `[profile.required] test-threads = 4`, preserving existing membership filters and CPU-based defaults. The same commit also introduces mise DAG/docs work owned elsewhere. Do not cherry-pick that whole commit.

Original fixture/parser branch tips add unrelated native wiring/scanner work. Exclude `088e98ecae` from the fixture branch and `77af035124` from the parser branch. All three original prep checkouts were clean. Their first .5 commits share historical base `eb881b03ac`; the original compiler branch's two commits are both .5.

Assembly had staged modifications in model/dev/xtask/src/gate.rs, backend process lib.rs, TLC runner.rs, framework gate.rs, gate/scheduler.rs, gate/timing.rs and test_run.rs; staged additions process/work.rs and gate/work_counts.rs; and an unresolved `model/umpire/backends/process/src/support.rs` conflict. These changes belong to shared/future lanes. No inventory operation altered them. Recheck this live state before integration.

Task dependencies are .1, .10 and .6. The .6 closure must be committed into `umpire` before admitting .5. After .6, preserve its test-run/quick selection and receipt behavior. Do not replace those files with assembly versions.

The prepared concurrency work is incomplete. Assembly's `gate/standalone.rs` still sets `NEXTEST_TEST_THREADS=2` at line 221 and writes `nextest_profile=default` and `test_threads=2` at line 495. Implement the .5 cap4 and truthful receipt/control changes against the integrated .6 tip. Propagate required profile selection through existing framework and Temporal gate/test leaves for TLC or Apalache required mode, retaining backend-skipped CPU defaults. Assembly gate files mix this behavior with .17 scheduling, so port only the bounded profile behavior and focused environment controls. Task .17 retains exclusive ownership of cross-invocation JVM leases, DAG execution, simultaneous failures and sibling cleanup. Until that lease exists, the conductor must serialize required-backend invocations and other heavy builds under the existing allocation.

Preparation handoffs under `.flow/tmp/prep254/{fixtures,compiler,parser}.md` explicitly report zero validation. The parser MAPPING.md repeats that limitation. There is no task5 completion/proof directory in committed artifacts before this report. The .5 candidate therefore has no demonstrated compilation, exact spans, fixture execution, compiler batching timings or simultaneous owner resource proof.

Once .6 releases capacity, run the following checks sequentially using the conductor's admitted tool/cache environment and CARGO_BUILD_JOBS=2. Commands below identify actual packages and binaries; no command was executed by this inventory.

```sh
cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire
mise exec -- cargo fmt --all --check
mise exec -- cargo nextest list --locked -p umpire-fixtures --test fixtures
mise exec -- cargo nextest run --locked -p umpire-testing --test consumers
mise exec -- cargo nextest run --locked -p umpire-dsl-macros --test fixtures
UMPIRE_QUICK_CACHE=0 UMPIRE_CACHE_DISABLE=1 UMPIRE_BACKENDS=require mise exec -- cargo nextest run --locked --profile required -p umpire-fixtures --test fixtures
cd /Users/stephan/Workspace/skunkworks/umpire/temporal/model
mise exec -- cargo fmt --all --check
mise exec -- cargo nextest run --locked -p umpire-temporal --test registry --test supplemental_admission
```

The inspected production gate uses `UMPIRE_QUICK_CACHE=0` and `UMPIRE_CACHE_DISABLE=1`; retain that bypass at the integrated .6 tip. Capture nextest JSON lists and compare against .3 identity evidence, explicitly accounting for the 24 replacements, four testing controls, two parser controls and two positive compiler identities. Preserve all other fixture-local/interpreted tests, ignored identities, doctests and the separate 119 dependency tuples. Run all 24 framework cases with TLC required and cache disabled. The full fixtures binary above also preserves its other cases.

Record 79 old macro UI inputs mapping exactly to 63 table rows plus 16 retained trybuild inputs. Verify all preserved error vectors and authored spans; retain any case whose equivalence fails. Record the actual selected/executed compiler counts and timings for 18 registry negatives, the registry positive, supplemental negative and generated API positive. Include affected clippy targets and required repo lint checks after source changes.

Use existing gate mode/stage-presence controls at the integrated tip, including Apalache explicit opt-in behavior and fail-fast/environment propagation. Real TLC discovery must demonstrate its retained exit/version admission and default heap cap. Follow `.flow/tmp/fn254-jvm-parallel-shape/decision.md` for exact JVM membership, simultaneous real framework owners, a Temporal representative, process counts and available-memory sampling. Optional Apalache requires its representative proof or a retained owner cap of one. Synthetic scheduler controls alone cannot establish real model admission or memory bounds.

Routine gates already skip Apalache, so report its historical opt-in cost separately. Do not claim a new default saving. Whole-gate timing and standalone final proof remain .8; scheduler concurrency proofs remain .17. Keep those deferred obligations open when committing any .5 source checkpoint.

The next safe action is to wait for .6's integrated closure, inspect disk and active cache leases, start .5 on `umpire`, apply the five isolated commit patches and two config diffs, then implement the missing bounded profile/standalone receipt controls. Preserve the assembly checkout and unrelated prep branches until their own tasks resolve.
