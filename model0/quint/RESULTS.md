# Quint experiment results

Verified 2026-09-30 on macOS/arm64 with Quint 0.33.0, Node 24.3.0, Python 3.14.2,
Go 1.27.0, Java 21.0.2, and Apalache distribution 0.62.1.

```sh
model/quint/run.sh --tla --verify
```

The command completed successfully: **766 tests**, four TLA+ exports, four complete
finite TLC safety checks, and six seeded simulation campaigns. Each simulation used
100 samples, a maximum of 12 steps, and seed `0x1`.

## Table agreement

| Model | Catalog states | Action classes | Enabled rows | Reachable states | Oracles |
| --- | ---: | ---: | ---: | ---: | --- |
| Nexus product | 6 | 12 | 25 | 6 | Live Go, committed Lean |
| Nexus protocol | 192 | 23 | 1,152 | 158 | Live Go, committed Lean |
| Activity product | 9 | 11 | 43 | 9 | Live Go, committed Lean |
| Activity protocol | 288 | 22 | 1,788 | 238 | Live Go |
| Worker | 2 | 3 | 3 | 2 | Live Go, committed Lean |

The checks cover 10,929 state/action pairs against Go and 4,593 against Lean, including
disabled pairs. Every enabled result matches its expected target state, outcome, and
ordered facts. Starts, ends, catalogs, and reachable states also agree. Changing one
reference successor in a temporary fixture produced the expected `QNT508` assertion
failure.

## TLC safety checks

| Model | Reachable checker states | Search depth | Result |
| --- | ---: | ---: | --- |
| Nexus product | 38 | 4 | No violation; queue exhausted |
| Nexus protocol | 1,771 | 10 | No violation; queue exhausted |
| Activity product | 87 | 6 | No violation; queue exhausted |
| Activity protocol | 2,781 | 10 | No violation; queue exhausted |

Checker states include `last` and `occurrence`, so they differ from base-table states.
TLC checks each machine's `properties` invariant. Worker composition is covered by
regression scenarios and sampled invariants, rather than an exhaustive composition check.
The exported files are in `.out/`, alongside compiled Quint JSON and ITF traces.

The Go parity reader also passed `go vet -tags test_dep` and the repository-configured
`golangci-lint` check with zero issues. Shell syntax, Python syntax, and local documentation
links were checked.

## What the experiment exposed

- The archived Activity sketch omitted the scheduled status on retry and returned no
  result for terminal pause/unpause. Regression tests failed before those corrections.
- Scenario completion and control claims cannot all serve as universal invariants:
  late requests return `notFound`. Explicit delivery properties cover late requests,
  while accepted requests retain their success assertions.
- A repeated real step needs an occurrence marker even when its state and facts repeat.
  A toggling Boolean preserves that distinction while keeping the checker finite.
- Quint 0.33.0 accepted and simulated module imports that failed flattening, and another
  flattened arrangement collided on private names during TLC execution. Separate files,
  explicit product imports, and unique internal names made all four exports checkable.
  Typechecking and compilation alone did not establish agreement.

This verifies the finite baseline and the selected safety properties. Definition IDs,
Behavior Fingerprints, Model IR export, Case production, runtime conformance, channels,
history-sensitive monitors, fairness, bounded-progress queries, and known-bug reporting
remain outside the experiment. See [README.md](README.md) for the supported surface.
