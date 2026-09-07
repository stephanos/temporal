module

public import Umpire.Case.Value

public section

/-!
Compatibility name for the generated Testpilot Program.

Remove this alias when downstream imports use `Testpilot.Protocol` directly.
-/

namespace Umpire.Case

abbrev Program := temporal.server.api.testpilot.v1.Program

end Umpire.Case
