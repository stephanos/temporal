# Task9 fresh BASE evidence

The unchanged public helper and focused baseline controls pass at `d2e0e035519f1385b9acf630a70152655b113f61`. These results support the separately reviewed helper integration admission. They establish no completed R10, prepared-adapter pin reproduction or native qualification.

Execution used stock Go1.27.1 on Linux arm64 with Go module downloads and checksum lookups disabled through `GOPROXY=off` and `GOSUMDB=off`. These settings establish no physical network isolation. The Go executable SHA-256 is `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. `run-base.cjs` retains the exact serial commands and environment. Each command directory contains its receipt, raw stdout/stderr and before/after source/input hashes. All 987 tracked nested-module files remained unchanged. The source freeze covers that selected tree, not every repository or toolchain input. The conductor's later Flow admission edits appear in the final status check and do not change those sources.

| Command evidence | Exit | Elapsed | Observed result |
| --- | --- | --- | --- |
| process-probe-1 | 0 | 3.507 s | 29 cases, 18 started children reaped and 18 observed GOPATHs removed |
| process-probe-2 | 0 | 2.627 s | Same 29 cases and cleanup observations |
| source-selection | 0 | 1.920 s | Real stock-Go listings and public-helper digests for both source sets |
| focused-command-packages | 0 | 5.374 s | 57 test results passed, zero failed/skipped across hostexec, gocommand and capabilityreview |
| bounded-target-controls | 0 | 1.187 s | 43 test results passed, zero failed/skipped |

The retained historical `base-process-probe/probe.go` was run unchanged. Its SHA-256 is `54af6ab429167b059e0f951068154a34e0015186efbc880d92d137228be783a0`. `probe-comparison.json` and `verify-probes.cjs` record zero normalized differences between the new runs and the historical conductor run. Only explicit temporary paths, process IDs and the startup timestamp are normalized. Concrete error types, ordered unwrap chains, errno/operation, signal/exit status, context identities, stderr, quoting, digest and cleanup observations remain in the comparison. The initial comparison in `verification.json` omitted temporary GOPATH normalization and reported 18 path-only differences; the later comparison supersedes that analysis without changing any raw evidence.

After acknowledged startup and stderr, cancellation and deadline still return wrapped `*exec.ExitError`, SIGKILL 9 and ExitCode -1, with both context `errors.Is` results false. This is fresh BASE behavior, not evidence for the future transport integration.

## Nonempty source selection

`source-selection.go` creates literal source fixtures and invokes the real stock Go executable with module mode disabled, cgo disabled and `GOEXPERIMENT=nogreenteagc`. It invokes the unchanged public helper separately. `source-selection/stdout.txt` retains exact fixture bytes, source hashes, raw listing JSON, inventories and digests.

Both platforms select `common.go`, their own `selected_<os>_<arch>.go`, `common.h`, `common.s` and their own platform assembly file. The other platform, Windows and cgo Go files are excluded; the test file remains a test-only listing entry. Each real listing is 634 bytes, with zero stderr.

| Source set | Literal BASE digest |
| --- | --- |
| darwin/arm64 | `sha256:2eeca986c81436b15fed9df5b22be9aa11d1b3a6349917817fe26497fe6527f0` |
| linux/amd64 | `sha256:f07021259f831522a922eb7f5e36a72d58a7d5efcd96c5197945d79484579191` |

## Capacity observations and limits

The supplementary survey lists available exact-version cached original adapter packages. It does not prepare replacements or compare production pins. Ten of fifteen exact module directories are available. Pebble, validator, hashicorp/go-metrics, go-sockaddr and memberlist exact versions are absent; their paths and errors are retained without downloads.

The initial survey used module roots for Cactus and Temporal SDK, yielding four explicit no-Go-files helper failures, and measured the otel SDK root instead of its prepared package. `cached-subpackage-correction/` retains separate real listings and helper results for `statsd`, `internal` and `resource` respectively. Every corrected command and helper exits zero. The otel resource listings additionally exercise the exact accepted quoted import-comment suffix. Original scripts and raw captures remain unchanged.

The corrected observed maximum is **4,075 stdout bytes** for Temporal SDK `internal` on each platform. All measured listing stderr streams are empty. The proposed 4 MiB per-stream limit is over 1,029 times this measured maximum. That is useful observed headroom for an explicitly disclosed refusal policy; it is not a proven maximum for all prepared adapters, arbitrary package directories, future source versions or diagnostics. Five exact cached inputs and all complete prepared-pin reproduction remain unproved. Capacity and capacity-plus-one refusal controls belong to implementation verification. The existing 15-minute watchdog remains an expressly added R10 lifetime bound.

All execution sessions are terminal. Session `94562` ran the serial baseline runner and exited zero; session `17088` ran the supplementary mapping correction and exited zero. No production/test source, Flow state, Git index/history, pin, toolchain launcher or historical artifact was changed by this research. Only this new evidence subtree was authored or generated. No lint, generator, full-host, patched-runtime or native qualification command ran.
