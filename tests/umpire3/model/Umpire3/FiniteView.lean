import Umpire3.ExecutableView

namespace Umpire3

universe u v

structure StateIdentity (State : Type u) where
  Code : Type u
  codeDecidableEq : DecidableEq Code
  encode : State → Code
  encode_injective : Function.Injective encode
  fingerprint : Code → Nat
  encodedSize : Code → Nat

def StateIdentity.decidableEq (identity : StateIdentity State) : DecidableEq State :=
  fun left right =>
    match identity.codeDecidableEq (identity.encode left) (identity.encode right) with
    | isTrue equality => isTrue (identity.encode_injective equality)
    | isFalse different => isFalse fun equality => different (congrArg identity.encode equality)

def StateIdentity.withFingerprint (identity : StateIdentity State)
    (fingerprint : identity.Code → Nat) : StateIdentity State where
  Code := identity.Code
  codeDecidableEq := identity.codeDecidableEq
  encode := identity.encode
  encode_injective := identity.encode_injective
  fingerprint := fingerprint
  encodedSize := identity.encodedSize

structure FiniteView {World : Type u} (model : Behavior World) (world : World) where
  executable : ExecutableView model
  identity : StateIdentity (model.State world)
  actionDecidableEq : DecidableEq (model.Action world)
  actionName : model.Action world → String
  actionName_injective : Function.Injective actionName

def FiniteView.withFingerprint {model : Behavior World} {world : World}
    (view : FiniteView model world) (fingerprint : view.identity.Code → Nat) :
    FiniteView model world where
  executable := view.executable
  identity := view.identity.withFingerprint fingerprint
  actionDecidableEq := view.actionDecidableEq
  actionName := view.actionName
  actionName_injective := view.actionName_injective

def FiniteView.initials {model : Behavior World} {world : World}
    (view : FiniteView model world) : List (model.State world) :=
  view.executable.initials world

def FiniteView.successors {model : Behavior World} {world : World}
    (view : FiniteView model world) (state : model.State world) :
    List (model.Action world × model.State world) :=
  view.executable.successors world state

end Umpire3
