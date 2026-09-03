import Umpire.CoreTests.Primitives
import Umpire.CoreTests.Trace

namespace Umpire.CoreTests

open Umpire

example : ModelValue.named (DefinitionId.of "switch.state.power") "on" = ({
    definitionId := DefinitionId.of "switch.state.power"
    value := "on"
  } : ModelValue) := by
  rfl

example : [
    ModelValue.named (DefinitionId.of "") "",
    ModelValue.named (DefinitionId.of "state") "unknown"
  ] = [
    { definitionId := DefinitionId.of "", value := "" },
    { definitionId := DefinitionId.of "state", value := "unknown" }
  ] := by
  rfl

end Umpire.CoreTests
