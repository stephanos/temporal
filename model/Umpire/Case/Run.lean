module

public import Umpire.Case.Program

public section

/-!
Compatibility name for the generated Testpilot Run.

Remove this alias when downstream imports use `Testpilot.Protocol` directly.
-/

namespace Umpire.Case

abbrev Run := temporal.server.api.testpilot.v1.Run

end Umpire.Case
