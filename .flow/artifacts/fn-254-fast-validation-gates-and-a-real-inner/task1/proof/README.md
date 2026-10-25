# fn254.1 baseline probes

The task owns this scratch directory and `model/design/gate-speed.md`. The conductor owns admission, the frozen main source and the only target lease. No source or gate behavior changes are made by these probes.

Owner amendment, 2026-10-25: fresh baseline measurements are waived; no measurement lease was granted and no production command below has run. The historical report is extracted with `bash .flow/tmp/fn254-baseline/run.sh historical`, which only reads/copies existing evidence. The remaining commands are optional, synthetically checked preparation for future/final measurement, with their real-tool integration unverified. They do not by themselves discharge R4's exact inventory or R6's final measurements.

Run from the main checkout with an explicit lease file containing `admitted: true`, `main`, `target` and `model_tree`. Main is `/Users/stephan/Workspace/skunkworks/umpire/temporal`; target is `/home/agent/.cache/codex-build/umpire-milestones-20261024/fn24612`. The collector refuses changed tracked/untracked Model/Make/tool-pin source. It never selects an inherited target or the worker checkout.

```
bash .flow/tmp/fn254-baseline/run.sh host --run baseline --lease .flow/tmp/fn254-baseline/lease.json
bash .flow/tmp/fn254-baseline/run.sh build --run baseline --lease .flow/tmp/fn254-baseline/lease.json
bash .flow/tmp/fn254-baseline/run.sh inventory --run baseline --lease .flow/tmp/fn254-baseline/lease.json
bash .flow/tmp/fn254-baseline/run.sh default --run baseline --lease .flow/tmp/fn254-baseline/lease.json
bash .flow/tmp/fn254-baseline/run.sh serial --run baseline --lease .flow/tmp/fn254-baseline/lease.json
```

Use a new run name for retries so original receipts and raw logs survive. `inventory` and `serial` consume the build's executable identities; the sequence is intentional. The baseline's required canonical gate is reused from the conductor's terminal receipt when equivalent. An absent terminal receipt is not a pass.

The build captures Cargo compiler-artifact JSON and stable Cargo timing HTML without cleaning the warm target. Each executable retains its package, target, features, profile, hash and fresh flag. Per-binary compile/link costs are unknown when reused or unavailable. The exact unit, ignored, doctest and registered trybuild case identities and diagnostic hashes form the migration inventory. Doctest listing runs under workspace feature unification, with package headings retained in one merged stream.

`default` uses the existing full stage timer and a scratch Cargo target runner to record executable wall times without changing harness concurrency. Its outer and stage times contain that observer overhead. `serial` runs the discovered binaries once with TLC required, Apalache skipped and one harness thread, timing the flushed partial-byte `test NAME ... ` prefix and completion. These serial per-test and per-binary times are instrumented observations; canonical parallel per-test times remain unknown. Ignored tests are listed but not forced to execute.

The serial run places a scratch `mise` dispatcher on PATH. It intercepts only `mise which java` and returns an absolute Java observer; other invocations delegate to the real mise. The Java observer records admission separately from actual `tlc2.TLC -tool` launches, saves each emitted module/config and preserves output/exit status. Missing end events remain incomplete. Module/config hashes plus exact binary/test contexts support fixture mapping; ambiguous mappings remain explicit. Mean JVM duration is not fixed startup cost.

Every measured command has monotonic elapsed/exit receipts, raw output and process snapshots before/after and every five seconds. Five-second samples cannot exclude all short competing work. Profile overrides, host and tool identities are in `host.json`. No unstable libtest or bootstrap override is used.

Synthetic verification: `python3 .flow/tmp/fn254-baseline/probe/controls.py`. No real Cargo/Go/Java or production test binary runs. Initial controls failed because resolving the Java symlink returned the common Python script path; returning the absolute un-resolved wrapper path corrected that before measurement. Both outcomes are retained in `synthetic-attempts.json`.
