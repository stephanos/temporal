# Pre-edit Quick baseline

Source snapshot: `/tmp/gomad-task17.WFwzYI/baseline-overlay` and `baseline-version` contain the actual dirty task-14 candidate, including all new typed command files. `baseline-source.sha256` binds the overlay, execution, root simulation and descriptor files. HEAD is `0dd05b313acd0986312da7fd3159520e6a21f1bf`.

Before production editing, `make generate && make validate` ran from tools/gomad3 and returned 0. All version, protocol, boundary, compiler-test, patch, script, compatibility and qualification-manifest checks passed. The command was observed through PTY session 26703; returned waits total 3.225 seconds (tool wait measurements, not an exact process duration).

At 2026-10-04 02:03:04–02:03:05 UTC, these canonical commands ran serially from tools/gomad3:

| Command | Exit | Observation |
| --- | --- | --- |
| `.toolchain/bin/go test -count=1 -tags test_dep internal/gomadio` | 127 | `.toolchain/bin/go: No such file or directory` |
| `make toolchain && make overlay-test` | 2 | Validation passed; toolchain recipe selected PATH Go 1.26.0 with GOTOOLCHAIN=local, rejected module requirement Go 1.27.1 |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Network\|NetBind'` | 127 | pinned executable absent |
| `cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim -run 'Network\|Process\|Backend'` | 127 | pinned executable absent |

Observed tool wall time for the serial canonical batch was 0.74745825 seconds. The default root module-selected Go reports 1.27.0; the nested tools/gomad3 module selects stock Go 1.27.1 linux/arm64 at `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64`. The final verification uses that executable on PATH to expose the native builder's platform rejection rather than the earlier PATH version mismatch. Neither command qualifies a supported platform.

The ownership regression was added before production migration. `ownership-red.log` records the actual AST failures for both old optional-state handles (exit 1), not compilation errors. `behavior-baseline.log` records the first three new direct-handle characterizations on the actual pre-migration task-14 source (exit 0, developmental stock-GOROOT stand-ins).
