# Runtime qualification gate

The receiving roots are `TestCooperativeRuntimeQualification` and
`TestCooperativeRuntimeQualificationAggregate` from the former
`wasi/integration_test.go`. The gate keeps 32 indices, seed `index * 1000003`,
25 fresh stores per index, and one fresh full-tape replay per index. The
original argv, four environment entries, read-step clock, entropy key,
namespace and WASI bounds are constructed before preparation freezes the build
key. Recordings keep diagnostics. The replay disables diagnostics through the
same authenticated replay preflight and compares exit, stdout, stderr, WASI
and choices. Public operator replay behavior is unchanged.

Build the repository harness with the fixed prefreeze build producer:

```sh
python3 tools/gomad/rust/driver/main_cli_build.py build \
  --tools-plan /absolute/static-plan.json --gnu-plan /absolute/gnu-toolchain.json \
  --root /absolute/fresh-runtime-build --role runtime-harness
```

The harness exposes three repository gate operations:

```text
runtime-qualification-replay identity
runtime-qualification-replay install ORIGINAL-PUBLIC-INSTALL-FLAGS
runtime-qualification-replay produce INSTALLATION SOURCE-DIRECTORY INDEX NEW-OUTPUT
runtime-qualification-replay ARTIFACT NEW-REPLAY-OUTPUT
```

The executable uses production preparation, linked execution, artifact
publication and replay admission. A new engine/store is instantiated for each
of the 25 recordings; separate immutable artifact roots retain every one.
The example refuses debug builds, fixture builds and helper diagnostic/stress
roles. Its status is 0 only when production execution, all raw recording
payload equalities and the forced replay succeed; application exits 0 and 17
are recorded in the artifacts. The adapter additionally requires all original
hook, collector and scheduling witnesses before publishing `runtime.json`.
The adapter first invokes the production installer through the harness, using
the admitted original guest policy and compiler/runtime inputs. This creates a
fresh installation bound to the harness executable, whose host/helper library
identities must equal the operator's. Existing sealed installations remain
unchanged. The installation command and producer share the 600 s shard bound.
The 25 authenticated inspections and final report publication also share that
same bound; a late inspection prevents the shard report from being published.

Each runtime stage binding needs `harness` and `harness_build_receipt`, both
sealed in the driver `files` inventory. The harness receipt uses
`gomad.driver-main-build/v1`, role `runtime-harness`, and the actual fixed Cargo
example recipe. Its independent reader authenticates the complete source,
GNU/compiler, build-script, Cargo, process and publication evidence, with empty
host/helper features and ordinary candidate identities. The freezer retains
every raw receipt input. A release label or executable identity response alone
cannot supply build evidence. The existing `installation`, original `source`
and `arguments` stage bindings remain required. The driver command timeout
limits the complete shard preparation, 25 recordings and replay to 600 s;
every guest execution retains the original 60 s ceiling.

The retirement adapter integration delegates its existing functions without
changing their callers:

```python
def runtime(config, root, label, entry):
    import runtime_gate
    return runtime_gate.runtime(config, root, label, entry, globals())

def runtime_aggregate(config, root, label):
    import runtime_gate
    return runtime_gate.runtime_aggregate(config, root, label, globals())
```

Passing the adapter globals preserves the independent receiving reader's
substitution of `command` and `write`: it authenticates retained raw commands
and recomputes reports without executing guests. Its runtime artifact callback
independently validates the build-key preimage, compiler and selected source
inventories, current compiler/runtime/collector, module metadata, exact
configuration and choice identity against raw retained bytes and admitted
inputs. It decodes choice wire records to recompute hook and branching counts;
edited inspection stdout cannot supply those counts. Fully resealed negative
fixtures verify refusal after ordinary artifact integrity checks succeed.
`receiving.Retained.runtime`
validates each stage's `producer.command.json` as a distinct producer PID,
the 600 s bound through the last inspection, its 25 unique
`recordings/instance-NN` artifacts, and each
artifact through the raw manifest reader. This internal producer has no public
campaign journal: its 25 separate stores reuse the production execution and
publication owners directly. The original gate required fresh stores within
one process, rather than 25 distinct producer processes.

The aggregate authenticates the common harness once and requires identical
harness bindings in all 32 shards. It reopens all 800 artifacts through authenticated public inspection,
rechecks diagnostics and the full terminal peak frame, compares every retained
stream and summary, authenticates all 32 diagnostics-off replay reports and
raw replay streams, and requires both canary exits and variation in choices,
stdout, map, run queue, timer callback and canary domains. Source/candidate,
compiler/collector, helper/host/runtime, profile, clock and limits are pinned.
The added unit failure matrix does not establish a live qualification pass.
