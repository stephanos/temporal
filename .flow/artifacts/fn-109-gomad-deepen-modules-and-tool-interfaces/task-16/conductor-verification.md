# Conductor verification — task 16 source candidate

Source was explicitly frozen by the worker with no live commands before these serial checks. HEAD remains `0dd05b313acd0986312da7fd3159520e6a21f1bf`; no commits were made. All ten source/test SHA-256 values printed by the conductor match task-16/evidence.json. simulation_model.go and the selected design remain unchanged.

These are developmental stock Go 1.27.1 checks on linux/arm64, not patched-runtime or supported-native acceptance.

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep ./runner/internal/execution -run 'SimulationTime|SimulationModel|SimulationCoordinator|ServeSimulation'

Exit: 0.

```text
ok  	go.temporal.io/server/tools/gomad3/runner/internal/execution	0.004s
```

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'SimulationTime|SimulationModel|SimulationCoordinator|ServeSimulation'

Exit: 0.

```text
ok  	go.temporal.io/server/tools/gomad3/runner/internal/execution	1.025s
```

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=100 -tags test_dep ./runner/internal/execution -run 'Simulation(Time|Model)(Progress|Lifecycle)'

Exit: 0.

```text
ok  	go.temporal.io/server/tools/gomad3/runner/internal/execution	0.067s
```

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go vet -tags test_dep ./runner/internal/execution

Exit: 0.

```text
(no output)
```

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$'

Exit: 0.

```text
ok  	go.temporal.io/server/tools/gomad3	0.289s
```

## env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off go run ../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-16/compare-test-bodies.go ../../.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-16/simulation_progress_test.before.txt runner/internal/execution/simulation_progress_test.go

Exit: 0.

```text
{
  "TestSimulationModelProgressCancellationKeepsCommitAndDiscardsLateArrival": {
    "before_sha256": "b5e304ecca24a0f9aa9302da7cd65c8aab829b7f2c218b7c789e068124645c2e",
    "after_sha256": "b5e304ecca24a0f9aa9302da7cd65c8aab829b7f2c218b7c789e068124645c2e",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationModelProgressConcurrentOperationsOnOneParticipant": {
    "before_sha256": "a5778a784d1ce41f0e27e77279dc9c3e08143da6d5b37710789c0eea63e02b91",
    "after_sha256": "a5778a784d1ce41f0e27e77279dc9c3e08143da6d5b37710789c0eea63e02b91",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationModelProgressRejectsUnknownAndDuplicateResponsesBeforeCallbacks": {
    "before_sha256": "d11481496f798f5adf106ffdfb9166f5ffc8393393f96b73ef0b28b57be7bef5",
    "after_sha256": "d11481496f798f5adf106ffdfb9166f5ffc8393393f96b73ef0b28b57be7bef5",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationModelProgressUnknownDiscardAcknowledgementKeepsDeliveryRunnable": {
    "before_sha256": "38a491302bebec70e1536bbb8dc9ee9ac5d2fdd0d987ca66649182c2c7c55dc7",
    "after_sha256": "38a491302bebec70e1536bbb8dc9ee9ac5d2fdd0d987ca66649182c2c7c55dc7",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationTimeProgressArrivalAtInstalledQuiescence": {
    "before_sha256": "2435aa0523fc9c6898843e5af2eb295d8545a10fc3256b4c4a97bbf2b999ab17",
    "after_sha256": "2435aa0523fc9c6898843e5af2eb295d8545a10fc3256b4c4a97bbf2b999ab17",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationTimeProgressDeathAndRestartRejectStaleIncarnation": {
    "before_sha256": "f87e1a8da53542606a7855320eedbde9944da4e2b87cbe021b672cd3ef52e8c7",
    "after_sha256": "f87e1a8da53542606a7855320eedbde9944da4e2b87cbe021b672cd3ef52e8c7",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationTimeProgressPartialAcknowledgementKeepsOtherDeliveryRunnable": {
    "before_sha256": "6924c8835427864fb0c486c2b55fa0bd763ec62c45fb74127197ecfe8e480a41",
    "after_sha256": "6924c8835427864fb0c486c2b55fa0bd763ec62c45fb74127197ecfe8e480a41",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationTimeProgressPreexistingDuplicateAdmissionConsumesArrival": {
    "before_sha256": "a431f3d659baef7e8227fa0d9dba29ee1618159d9e10b6a5a4cf489106243f5d",
    "after_sha256": "3b3c52b7596d7c0a3d3e2b742f3d1a35cb440c43358099edf853e664b1dea650",
    "body_byte_identical": false,
    "strengthened_negative": true
  },
  "TestSimulationTimeProgressPreexistingMalformedWaitWakesQuiescence": {
    "before_sha256": "fc0daa94023e8fed2fe28f96ca1436d6c0373802230a159b4441180af18ee18d",
    "after_sha256": "0b7a717011a758dbfe21ca6295b297446b8355076c3f6c51d501c9d6319bbbfa",
    "body_byte_identical": false,
    "strengthened_negative": true
  },
  "TestSimulationTimeProgressUnknownAcknowledgementBeforeAdmission": {
    "before_sha256": "e3b0c6b617d64ec7ccb0abbff143773fce71db050127638cac3bfddc19b8d745",
    "after_sha256": "e3b0c6b617d64ec7ccb0abbff143773fce71db050127638cac3bfddc19b8d745",
    "body_byte_identical": true,
    "strengthened_negative": false
  },
  "TestSimulationTimeProgressUnknownAcknowledgementBeforeForwardAndTransfer": {
    "before_sha256": "e6bdcab3df58a9adf1719cb672915a4438d206842d0bcb823a4cff67aeaec2e4",
    "after_sha256": "e6bdcab3df58a9adf1719cb672915a4438d206842d0bcb823a4cff67aeaec2e4",
    "body_byte_identical": true,
    "strengthened_negative": false
  }
}
```

## git diff --check

Exit: 0.

```text
(no output)
```

## gofmt -l runner/internal/execution/simulation_progress.go runner/internal/execution/simulation_time.go runner/internal/execution/simulation_unix.go runner/internal/execution/process_unix.go runner/internal/execution/simulation_time_test.go runner/internal/execution/simulation_unix_test.go runner/internal/execution/simulation_progress_test.go runner/internal/execution/simulation_progress_fixture_test.go runner/internal/execution/simulation_progress_lifecycle_test.go

Exit: 0.

```text
(no output)
```

## Shared input validation

`env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off make validate` exited 0. It checked version/protocol/boundary generation, compiler fixtures, patch/overlay inventory, script ownership, compatibility packs/profile tests and the qualification manifest. No generated input was changed by task 16.

## Preservation checksum clarification

The conductor initially mistook the prefix of `e3b0c6b617d64ec7ccb0abbff143773fce71db050127638cac3bfddc19b8d745` for the SHA-256 of empty input. It is a different full digest. The integration reviewer independently extracted the nonempty brace-delimited body of UnknownAcknowledgementBeforeAdmission and reproduced this digest for both preimage and current source; direct whole-function extraction also produced matching `d709ef4b23e2a437cb87f34c12ff39b2fe2256788b60c5f7c3f4a3c0ee9eaf7d`. No comparator/source fix was needed. All nine valid test bodies remain identical; the two historical negatives are the only test-body changes.

## Acceptance still open

The five canonical patched-toolchain Quick commands remain unavailable (exit127, absent `.toolchain/bin/go`), and the exact linter is an incompatible Mach-O CpuArm64 binary (exit2 before analysis). Whole patched host/Runner/process tests, native timers, isolation and both supported native-platform qualification are not established. Formal implementation review stays deferred on these unavailable gates. Separate fresh read-only source audits do not constitute SHIP or R11 completion.
