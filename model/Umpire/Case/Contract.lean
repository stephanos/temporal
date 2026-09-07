module

public import Umpire.Case.Run

public section

/-!
Compatibility name for the generated Testpilot Contract.

Remove this alias when downstream imports use `Testpilot.Protocol` directly.
-/

namespace Umpire.Case

abbrev Contract := temporal.server.api.testpilot.v1.Contract

end Umpire.Case
