import Umpire.Property

/-! Narrow-import regression for the `Umpire.Property` public facade. -/

namespace Umpire.PropertyImportTests

#check (Umpire.PropertyDeclaration : Type)
#check (Umpire.checkProperty :
  Umpire.PropertyCheckContext → Umpire.PropertyAuthoring →
    Except Umpire.PropertyError Umpire.CheckedProperty)
#check (Umpire.CheckedProperty.traceView :
  Umpire.CheckedProperty →
    Umpire.ModelTrace Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue →
      Umpire.PropertyTraceView)
#check (Umpire.evaluateProperty :
  Umpire.CheckedProperty →
    Umpire.ModelTrace Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue Umpire.ModelValue →
      Umpire.PropertyEvaluation)

#guard_msgs (error, substring := true) in
#check Umpire.BehaviorDeclaration

#guard_msgs (error, substring := true) in
#check Umpire.QueryDeclaration

end Umpire.PropertyImportTests
