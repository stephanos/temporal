Execution choice fixture repair after integrated host gate

Root cause: TestTargetHelper's choice-trace case exited immediately; choice-marker printed the marker and exited. Neither created concurrent user goroutines, so removing runtime-owned workers from user choice alternatives correctly made their traces empty. The integrated red evidence is integrated-test-host.log (the five failure reports at lines 25–35).

Only runner/internal/execution/process_test.go changed: both cases call runChoiceRunnableTarget, which launches four user goroutines that signal a buffered completion channel and joins all four before returning. The marker remains after these deliberate choices. No production code, assertions, or existing comments changed.

Patched candidate build key: 4a6e5b695ea538f0a56eb70874ff945693223b53d0fcee56cc555f89e1a9ac0e. Environment: GOROOT/GOMADSEED/GOMAD3_CHILD_SEED unset; PATH starts with stock Go 1.27.1; GOMAD3_STOCK_GO names its binary; GOWORK=off.

PASS: .toolchain/bin/go test -tags test_dep -count=1 -v -run '^TestRun(TransportsCompleteChoiceTrace|ReturnsValidatedOverflowChoiceTrace|RejectsExhaustedChoiceTapeBeforeTargetMarker|RejectsChoiceMetadataMismatchBeforeTargetMarker|RejectsUnconsumedChoiceTape|ReplaysCompleteChoiceTape)$' ./runner/internal/execution (execution-choice-fixtures-green.log).
PASS: .toolchain/bin/go test -tags test_dep -count=1 ./runner/internal/execution (execution-package-green.log, full execution package).
PASS: git diff --check.

Source frozen. No commits, staging, task completion, or full-host gate performed by this worker. Parent owns integrated gates and review.
