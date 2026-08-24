import Umpire3.Catalog

namespace Umpire3

structure LifecycleDeclaration where
  entity : EntityDeclaration
  actions : List ActionDeclaration
  observations : List ObservationDeclaration
  properties : List PropertyDeclaration
  module : ModuleDeclaration
  target : TargetDeclaration

def LifecycleDeclaration.wellFormed (declaration : LifecycleDeclaration) : Bool :=
  declaration.entity.identifier != "" &&
    !declaration.actions.isEmpty && declaration.actions.all (·.identifier != "") &&
    !declaration.observations.isEmpty && declaration.observations.all (·.identifier != "") &&
    !declaration.properties.isEmpty && declaration.properties.all (·.identifier != "") &&
    declaration.module.identifier != "" && declaration.target.identifier != "" &&
    declaration.target.modules.contains declaration.module.identifier &&
    declaration.properties.all (fun property => declaration.target.properties.contains property.identifier)

def LifecycleDeclaration.WellFormed (declaration : LifecycleDeclaration) : Prop :=
  declaration.wellFormed = true

end Umpire3
