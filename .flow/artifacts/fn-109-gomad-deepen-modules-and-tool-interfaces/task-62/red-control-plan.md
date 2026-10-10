# Red controls before the repair

The root accepted baseline terminal release and requires red controls before any helper/fixture correction. runner_test.go remains exact BASE (`a4d57dbfa6fd4dbf0586f08b33edcc06f3740045731f50e2cabf8149e7faf3a7`). The additive control file contains the original receive-only helper and is frozen at `b4130a05ab0fdbeb8239018b0102e3303d1564711c44fa517357fecb274e1ea6`.

Execute `go test -tags test_dep -count=1 -timeout=10s -json -run ^TestWaitForProgressStart ./runner` after the root grants the short red-control lane. The fixed start case must pass. Nil-completion and deadline cases must fail with nil rather than their literal expected errors. The real errorPreparer case must fail with nil rather than the preparation sentinel. These are assertion failures, with ordinary termination and three failure causes, rather than syntax failure or timeout-as-coverage.

Each control invokes the helper in the test goroutine. A one-second AfterFunc calls the control's sync.OnceFunc start signal, so a receive-only helper unblocks and returns nil. Cleanup uses that same once-only signal. No separate blocked helper goroutine needs joining. The real exploration sends its preparation error into a buffered channel and ends normally without Execute. Final source retains these same expectations and watchdogs. This concretizes the no-leak sensitivity mechanism in plan.md without changing its expected outcomes.

All historical baseline runner/receipt bytes remain unchanged. No Go execution or formatting follows until root authorization.
