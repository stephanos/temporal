Each future `driver.command` completion retains `resources` in both its original
`.command.json` and `.process.json`. The command receipt's existing SHA binding
therefore authenticates resource bytes along with argv, PID, exit, environment,
interval and completion. Missing measurements are not upgraded from old receipts.

The object is `gomad.driver-child-resources/v1`:

- `measurement: wait4`, `scope: whole-command-child`.
- `pid`, `waited_pid`: the exact launched/registered PID; the gated launcher execs
  the target in place, retaining that PID.
- `wait_status`: raw Linux wait status; `exit_code` uses the same negative signal
  convention as Popen and the existing command receipt.
- `user_seconds`, `system_seconds`: raw wait4 rusage CPU values.
- `user_nanos`, `system_nanos`: `round(seconds * 1_000_000_000)`;
  `cpu_conversion: round(seconds*1e9)` records that conversion. These do not claim
  finer kernel precision than the raw values.
- `ru_maxrss`: raw Linux wait4 peak RSS; `ru_maxrss_unit: KiB`.
- `peak_rss_bytes`: `ru_maxrss * 1024`;
  `rss_conversion: ru_maxrss*1024` records normalization.

`MeasuredPopen` preserves Popen's wait lock and return-code handling. Its readiness
probe is Linux waitid with WNOWAIT, which never reaps or samples resource usage.
Only a ready exact PID reaches one wait4 call. Status and usage are cached with
termination/interrupt signals blocked briefly, before a pending handler can run;
interrupted cleanup can finish Popen.wait from that cached status without another
wait4 or reap. No-timeout readiness polling sleeps between probes. Timeout,
nonzero/signal exit, stdout/stderr retention, descendant process-group refusal,
kill/reap cleanup and original exception precedence remain intact. The existing
scheduler still owns stage cleanup and admission. No RUSAGE_CHILDREN difference,
other-child observation, elapsed-time CPU approximation, or synthetic resource
fallback is used.

This is the kernel's whole command-child measurement, including normal launcher
and process lifecycle costs. It is distinct from the original mandatory-corpus
and SQLite-catalogue checker's scoped self observations. The independent receiving
reader must retain those original monotone self CPU/positive <=2GiB lifetime RSS
requirements, checked uint64 additions and counting rules. The generic hook does
not impose that checker bound on every build or guest command. It does not count
producer/terminal CPU again or qualify any workload.

Portable tests run bounded Python children for successful output/actual PID,
nonzero accepted/refused statuses, self-signal, timeout, concurrent differently
sized/CPU-active children plus an unrelated child, and an interrupt injected
immediately after the single reap. They check exactly one wait4 per command,
resource/status/unit conversions, both raw receipts and original cleanup/error
behavior. They run no Go, Cargo, guest, or qualification producer.

Command and process journals also retain `clock: CLOCK_MONOTONIC`, the actual
parent PID, the timestamp immediately before `Popen`, and the timestamp immediately
after that same `wait4`. These lifecycle boundaries authenticate nested native
host captures without converting wall time or inferring a start from elapsed time.
The accounting and lifecycle observations share the single existing reap.
