import Testpilot.Tests.ProtoJSON
import Umpire.Case.ProtoJSON

namespace Testpilot.Tests.Compatibility

private def generatedVersion (value : Umpire.Case.FormatVersion) :
    temporal.server.api.testpilot.v1.FormatVersion := value
private def generatedValue (value : Umpire.Case.Value) :
    temporal.server.api.testpilot.v1.Value := value
private def generatedValueType (value : Umpire.Case.ValueType) :
    temporal.server.api.testpilot.v1.ValueType := value
private def generatedProgram (value : Umpire.Case.Program) :
    temporal.server.api.testpilot.v1.Program := value
private def generatedRun (value : Umpire.Case.Run) : temporal.server.api.testpilot.v1.Run := value
private def generatedContract (value : Umpire.Case.Contract) :
    temporal.server.api.testpilot.v1.Contract := value
private def generatedCase (value : Umpire.Case) : temporal.server.api.testpilot.v1.Case := value

private def render
    (codec : temporal.server.api.testpilot.v1.Case → IO (Except Testpilot.ProtoJSON.Error String))
    (value : temporal.server.api.testpilot.v1.Case) :
    IO String := do
  match ← codec value with
  | .ok encoded => pure encoded
  | .error error => throw (IO.userError (toString error))

private def tests : IO Unit := do
  let _ := generatedVersion
  let _ := generatedValue
  let _ := generatedValueType
  let _ := generatedProgram
  let _ := generatedRun
  let _ := generatedContract
  let _ := generatedCase
  for value in [Testpilot.Tests.ProtoJSON.literalCase,
      Testpilot.Tests.ProtoJSON.representativeCase] do
    let canonical ← render Testpilot.ProtoJSON.canonical value
    let compatibility ← render Umpire.Case.ProtoJSON.canonical value
    unless canonical == compatibility do
      throw (IO.userError "Umpire compatibility codec diverged from Testpilot.ProtoJSON")

#eval tests

end Testpilot.Tests.Compatibility
