import Lake

open System Lake DSL

package «temporal-model» where
  lintDriver := "modelLint"
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

input_file testpilotOutcomeProto where
  path := "../proto/internal/temporal/server/api/testpilot/v1/outcome.proto"

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
    testpilotOutcomeProto,
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

lean_exe testpilotProtoJSONFixture where
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

@[default_target] lean_exe «temporal-model-inspect» where
  root := `Temporal.Tool.Inspect

lean_exe «temporal-model-semantic-inventory» where
  root := `Temporal.Tool.SemanticInventoryMain

lean_exe «temporal-model-semantic-inventory-tests» where
  root := `Temporal.Tool.SemanticInventoryMainTests

lean_exe «temporal-model-semantic-inventory-make-tests» where
  root := `Temporal.Tool.SemanticInventoryMakeTestsMain

@[default_target] lean_exe «temporal-testpilot» where
  root := `Temporal.Tool.Testpilot

lean_exe modelLint where
  root := `ModelLint
  supportInterpreter := true

lean_exe modelLintTests where
  root := `ModelLint.ImportGraphTests
  supportInterpreter := true
