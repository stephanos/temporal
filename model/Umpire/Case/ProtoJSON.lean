import Testpilot.ProtoJSON
import Umpire.Case

namespace Umpire.Case.ProtoJSON

/-- Compatibility forwarder to the sole Testpilot ProtoJSON codec.

Remove this name when downstream callers import `Testpilot.ProtoJSON` directly.
-/
def canonical (item : Umpire.Case) : IO (Except Testpilot.ProtoJSON.Error String) :=
  Testpilot.ProtoJSON.canonical item

end Umpire.Case.ProtoJSON
