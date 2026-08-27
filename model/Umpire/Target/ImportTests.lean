import Umpire.Target

/-! Focused import contract for target authoring and checked composition. -/

#check Umpire.TargetDeclaration
#check Umpire.TargetDefinition
#check Umpire.TargetComposition
#check Umpire.TargetComposition.empty
#check Umpire.TargetComposition.provide
#check Umpire.TargetComposition.connect
#check Umpire.AuthoredTarget
#check Umpire.AuthoredTarget.make
#check Umpire.CheckedTarget
#check Umpire.checkTarget
#check Umpire.checkedTarget
#check Umpire.composeTarget
#check Umpire.elaborateTarget

#guard_msgs (error, substring := true) in
#check Umpire.CheckedTarget.mk

#guard_msgs (error, substring := true) in
#check Umpire.AuthoredTarget.mk
