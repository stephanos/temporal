import Lake

open System Lake DSL

package «temporal-model» where
  lintDriver := "umpire-lint"
  builtinLint? := true

require "leanprover-community" / batteries @ git "v4.33.0"

require protobuf from git
  "https://github.com/Lean-zh/protobuf.git"@"406da521c0ebb47207be28e3d9ef738de95a4dd3"

input_file testpilotCaseProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/case.proto"

input_file testpilotContractProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/contract.proto"

input_file testpilotExpressionProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/expression.proto"

input_file testpilotInstructionProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/instruction.proto"

input_file testpilotProgramProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/program.proto"

input_file testpilotRunProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/run.proto"

input_file testpilotValueProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/value.proto"

target testpilotProtocolSchemas (pkg : NPackage __name__) : FilePath := do
  let mut inputJobs : Array (Job FilePath) := #[]
  for input in #[
    testpilotCaseProto,
    testpilotContractProto,
    testpilotExpressionProto,
    testpilotInstructionProto,
    testpilotProgramProto,
    testpilotRunProto,
    testpilotValueProto
  ] do
    let inputTarget ← input.get
    inputJobs := inputJobs.push (← fetch inputTarget.default)
  let inputs := Job.collectArray (traceCaption := "Testpilot protocol schemas") inputJobs
  let stamp := pkg.buildDir / "testpilot-protocol-schemas"
  buildFileAfterDep stamp inputs fun _ => do
    removeFileIfExists <| pkg.leanLibDir / "Testpilot/Protocol.olean"
    createParentDirs stamp
    IO.FS.writeFile stamp ""

@[default_target] lean_lib Shared

@[default_target] lean_lib Testpilot where
  extraDepTargets := #[`testpilotProtocolSchemas]

@[default_target] lean_lib TestpilotTests where
  roots := #[`Testpilot.Tests]

lean_exe «umpire-protojson-fixture» where
  root := `Testpilot.Tests.ProtoJSONMain

@[default_target] lean_lib Temporal

@[default_target] lean_lib Umpire

@[default_target] lean_lib UmpireTests

@[default_target] lean_lib TemporalModelTests

@[default_target] lean_lib TemporalExperimentalTests

lean_lib ModelLintSupport where
  roots := #[
    `Tools.LeanImportGraph,
    `Tools.LeanImportGraphTests,
    `Tools.LeanSourceInventory,
    `Tools.LeanSourceInventoryTests,
    `ModelLint.ImportGraph
  ]

@[default_target] lean_exe «umpire-inspect» where
  root := `Temporal.Tool.Inspect

lean_exe «umpire-inventory» where
  root := `Temporal.Tool.InventoryMain

lean_exe «umpire-inventory-tests» where
  root := `Temporal.Tool.InventoryMainTests

lean_exe «umpire-inventory-make-tests» where
  root := `Temporal.Tool.InventoryMakeTestsMain

@[default_target] lean_exe «umpire-case» where
  root := `Temporal.Tool.Testpilot

lean_exe «umpire-goldens» where
  root := `Temporal.Tool.Goldens

lean_exe «umpire-correlated-fixtures» where
  root := `Umpire.Case.Tests.CorrelatedFixtureMain

lean_exe «umpire-lint» where
  root := `ModelLint
  supportInterpreter := true

lean_exe «umpire-lint-tests» where
  root := `ModelLint.ImportGraphTests
  supportInterpreter := true
