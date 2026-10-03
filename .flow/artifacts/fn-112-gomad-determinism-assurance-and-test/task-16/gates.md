# Gates (linux/arm64, development harness on, base bce040728 plus this change)

| Command (from tools/gomad3) | Result |
| --- | --- |
| `GOFLAGS='-tags=test_dep -count=1' make test-host` | exit 2 in 263 s; every failing test is in the harness baseline (adapter replacement digests, doctor unsupported-host, clock/process-termination set). No new failure. `artifact`, `runner`, `runner/internal/campaign`, `cmd/gomad` ok. Three baseline failures (TestCLIWatchdogRetainsIncompleteEvidence, TestRunIOTerminalAfterTermination, TestRunTimesOutAndRemovesTermIgnoringProcessGroup) passed this run. |
| `make validate` | exit 0 (5 s) |
| `.toolchain/bin/go test -tags test_dep -count=1 ./artifact/` | ok |
| `.toolchain/bin/go test -tags test_dep -count=1 -run 'TestRunKeepsTwoSuccesses\|TestRunCountsASharedTarget\|Retention' ./runner/` | ok |
| `.toolchain/bin/go test -tags test_dep -count=1 -run 'TestCLIExploreKeepsSuccesses\|TestCLIExploreReplay' ./cmd/gomad/` | ok |
| `.toolchain/bin/go test -tags test_dep -count=1 -run TestPackageArchitecture .` | ok |
| `GOOS=linux GOARCH=amd64 .toolchain/bin/go vet -tags test_dep ./artifact/ ./runner/ ./cmd/gomad/` | exit 0 |
| Red first: runner success store without `Key: artifact.StoreKeyExecution` | `TestRunKeepsTwoSuccessesOfOneOutcomeSignatureApart` FAIL, both `SuccessArtifacts` name one directory |

Not run: darwin/arm64 and linux/amd64 native gates (no host); CI.
