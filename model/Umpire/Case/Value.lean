module

public import Testpilot.Protocol

public section

/-!
Compatibility names for the generated Testpilot value protocol.

New code imports `Testpilot.Protocol` and uses `temporal.server.api.testpilot.v1` directly. Remove
these aliases when downstream Umpire imports have completed that migration.
-/

namespace Umpire.Case

abbrev FormatVersion := temporal.server.api.testpilot.v1.FormatVersion
abbrev Value := temporal.server.api.testpilot.v1.Value
abbrev ValueType := temporal.server.api.testpilot.v1.ValueType

end Umpire.Case
