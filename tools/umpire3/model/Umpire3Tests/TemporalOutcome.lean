import Temporal.Outcome

namespace Umpire3.Tests.TemporalOutcome

open Umpire3.Temporal.Outcome

example : classify .conforming (some .success) = .recovered := by decide
example : classify .conforming (some .failure) = .degraded := degradedIsConformingFailure
example : classify .violating none = .flagged := by decide
example : classify .inconclusive none = .unreached := by decide

end Umpire3.Tests.TemporalOutcome
