# Task15 standalone harness: prepared, proof unexecuted

This is an adapted copy of the retained task16 proof/observer and watchdog controls. `input-manifest.json` records the exact prepared inputs and original source hashes. The four safe Python controls use a fake child and stub Git responses; their receipt does not establish runtime proof acceptance.

Root owns admission and execution after committing a clean candidate. No candidate SHA or retained task16 binary hash is admitted here. Supply a full SHA explicitly:

```
python3 .flow/tmp/fn25415-validation/standalone-harness/watchdog.py --expected-head <40-character-candidate-SHA> --execute
```

Without `--execute`, this command only describes preparation. Before any proof process launch, the watchdog verifies clean main source and expected HEAD, input hashes, controls bound to this manifest, and 50 GiB free on both the source filesystem and actual `/tmp` filesystem. During execution it retains the 30 GiB free guard and waits for the proof child to reach a terminal state. The proof additionally retains a 30 GiB growth bound over its target/private tmp/Go cache/standalone-copy roots, output and timeout guards, source seals (including task15's Markdown), owned PID/start-time observation and cleanup, exact 17 child stages, parent/child count separation, native inventory and separate doctest accounting.

Task15 explicitly owns reuse of `.flow/tmp/fn25416-provider-validation/target`; this retained path is the sole task16-named runtime output exception. The harness builds current task15 native bootstrap binaries into that target and records their fresh hashes. This is not an empty-target/cold build claim. New proof output is beneath this directory's `fn25415-proof-output`; isolated production copies use `/tmp/umpire-fn25415-final-*`. Existing originals are untouched. Darwin and task8 timing obligations remain deferred. No build or standalone proof has been executed by this preparation.
