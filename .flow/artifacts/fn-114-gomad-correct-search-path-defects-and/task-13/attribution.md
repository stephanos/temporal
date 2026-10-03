# E4 attribution listings (linux/arm64, development harness)

Throwaway instrumented builds printed one stderr line per run-queue event from
`gomadChoiceRunqIndex`; neither instrumentation is committed. Lines are
counted with `sort | uniq -c`; `runtime.` prefixes are stripped. Each listing
is the first retained success of `gomad qualify --seed SEED --repeat 2
--choices --replay-successes ...` (see `../control-probe.md`).

## Before: the unmodified rule (every queue of two or more is a decision)

`decision alternatives=N: <start function>/system=<isSystemGoroutine(gp, false)> ...`
in queue order.

```
== artifacts-control-11
     10 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true main/system=false
      8 gomad-e4-probe decision alternatives=2: main/system=false gcenable.gowrap1/system=true
      2 gomad-e4-probe decision alternatives=3: gcenable.gowrap2/system=true gcenable.gowrap1/system=true main/system=false
      2 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=3: updateMaxProcsGoroutine/system=true runFinalizers/system=true gcBgMarkStartWorkers.gowrap1/system=true
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=2: runFinalizers/system=true main/system=false
      1 gomad-e4-probe decision alternatives=2: runFinalizers/system=true gcBgMarkStartWorkers.gowrap1/system=true
decisions with no system alternative: 0
== artifacts-control-17
      9 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true main/system=false
      7 gomad-e4-probe decision alternatives=2: main/system=false gcenable.gowrap1/system=true
      2 gomad-e4-probe decision alternatives=3: gcenable.gowrap2/system=true gcenable.gowrap1/system=true main/system=false
      1 gomad-e4-probe decision alternatives=4: updateMaxProcsGoroutine/system=true gcenable.gowrap2/system=true gcenable.gowrap1/system=true main/system=false
      1 gomad-e4-probe decision alternatives=4: forcegchelper/system=true updateMaxProcsGoroutine/system=true runFinalizers/system=true gcBgMarkStartWorkers.gowrap1/system=true
      1 gomad-e4-probe decision alternatives=3: updateMaxProcsGoroutine/system=true forcegchelper/system=true gcBgMarkStartWorkers.gowrap1/system=true
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true updateMaxProcsGoroutine/system=true main/system=false
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap2/system=true main/system=false
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=2: updateMaxProcsGoroutine/system=true main/system=false
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap2/system=true main/system=false
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap2/system=true forcegchelper/system=true
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=2: forcegchelper/system=true main/system=false
decisions with no system alternative: 0
== artifacts-twouser-11
      4 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=4: updateMaxProcsGoroutine/system=true runFinalizers/system=true sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=3: runFinalizers/system=true sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false runFinalizers/system=true
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true main/system=false
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap1/system=true gcenable.gowrap2/system=true
decisions with no system alternative: 4
== artifacts-twouser-17
      4 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=5: forcegchelper/system=true updateMaxProcsGoroutine/system=true runFinalizers/system=true sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=4: updateMaxProcsGoroutine/system=true runFinalizers/system=true forcegchelper/system=true sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe decision alternatives=3: runFinalizers/system=true forcegchelper/system=true updateMaxProcsGoroutine/system=true
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap2/system=true main/system=false
      1 gomad-e4-probe decision alternatives=3: forcegchelper/system=true gcenable.gowrap1/system=true gcenable.gowrap2/system=true
      1 gomad-e4-probe decision alternatives=2: gcenable.gowrap2/system=true forcegchelper/system=true
      1 gomad-e4-probe decision alternatives=2: forcegchelper/system=true updateMaxProcsGoroutine/system=true
      1 gomad-e4-probe decision alternatives=2: forcegchelper/system=true main/system=false
decisions with no system alternative: 4
```

## After: the task-13 rule

`runtime-first offset=K count=N F` is a pick of the first runtime-owned
goroutine F at queue offset K of N entries, without a draw or record (K > 0
means it ran ahead of K user goroutines). `decision` lines are recorded
decisions; every one is among user goroutines only. `fixture` is
`runq_user_choice` with the mode as its argument.

```
=== control 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: virtual_now_unix_nanos 946684800000000000
num_gc 4
last_gc_is_host_time true
memstats_pause_total_ns 0
     13 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      6 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== twouser 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=5 decisions=5 branching=5 runnable=5 select-poll=0 select-result=0 terminal=complete
-- stdout: ababbaab
      5 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
=== two-users 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=5 decisions=5 branching=5 runnable=5 select-poll=0 select-result=0 terminal=complete
-- stdout: two-users ababbaab
      5 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
=== main-only 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: main-only done
     13 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      6 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== one-user 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: one-user done
     13 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      6 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== busy-runtime 11
gomad: qualification qualified=true deterministic=true target-success=true seed=11 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=40 decisions=40 branching=40 runnable=40 select-poll=0 select-result=0 terminal=complete
-- stdout: busy-runtime a=32 b=32 a-saw-finalizers=true b-saw-finalizers=true
     40 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      9 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      7 gomad-e4-probe runtime-first offset=2 count=3 runFinalizers
      7 gomad-e4-probe runtime-first offset=1 count=2 runFinalizers
      5 gomad-e4-probe runtime-first offset=1 count=5 gcenable.gowrap2
      5 gomad-e4-probe runtime-first offset=1 count=4 gcenable.gowrap1
      5 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      3 gomad-e4-probe runtime-first offset=2 count=3 gcenable.gowrap1
      3 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=1 count=3 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=4 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=1 count=2 gcBgMarkStartWorkers.gowrap1
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== control 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: virtual_now_unix_nanos 946684800000000000
num_gc 4
last_gc_is_host_time true
memstats_pause_total_ns 0
     11 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      8 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== twouser 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=5 decisions=5 branching=5 runnable=5 select-poll=0 select-result=0 terminal=complete
-- stdout: baabbaba
      5 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
=== two-users 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=5 decisions=5 branching=5 runnable=5 select-poll=0 select-result=0 terminal=complete
-- stdout: two-users baabbaba
      5 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
=== main-only 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: main-only done
     11 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      8 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== one-user 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=0 decisions=0 branching=0 runnable=0 select-poll=0 select-result=0 terminal=complete
-- stdout: one-user done
     11 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      8 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap2
      1 gomad-e4-probe runtime-first offset=0 count=3 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
=== busy-runtime 17
gomad: qualification qualified=true deterministic=true target-success=true seed=17 repeat=2 
gomad: choices profile=gomad3-choice-trace/v3 records=39 decisions=39 branching=39 runnable=39 select-poll=0 select-result=0 terminal=complete
-- stdout: busy-runtime a=32 b=32 a-saw-finalizers=true b-saw-finalizers=true
     39 gomad-e4-probe decision alternatives=2: sync.(*WaitGroup).Go.func1/system=false sync.(*WaitGroup).Go.func1/system=false
     10 gomad-e4-probe runtime-first offset=1 count=2 gcenable.gowrap1
      9 gomad-e4-probe runtime-first offset=1 count=2 runFinalizers
      6 gomad-e4-probe runtime-first offset=2 count=3 runFinalizers
      6 gomad-e4-probe runtime-first offset=0 count=3 gcenable.gowrap1
      6 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap1
      4 gomad-e4-probe runtime-first offset=1 count=5 gcenable.gowrap2
      4 gomad-e4-probe runtime-first offset=1 count=4 gcenable.gowrap1
      4 gomad-e4-probe runtime-first offset=0 count=4 gcenable.gowrap2
      2 gomad-e4-probe runtime-first offset=2 count=3 gcenable.gowrap1
      2 gomad-e4-probe runtime-first offset=1 count=3 gcenable.gowrap1
      1 gomad-e4-probe runtime-first offset=1 count=2 gcBgMarkStartWorkers.gowrap1
      1 gomad-e4-probe runtime-first offset=0 count=4 updateMaxProcsGoroutine
      1 gomad-e4-probe runtime-first offset=0 count=3 runFinalizers
      1 gomad-e4-probe runtime-first offset=0 count=3 forcegchelper
      1 gomad-e4-probe runtime-first offset=0 count=2 gcenable.gowrap2
```
