import Umpire.Command.Authoring
import Umpire.Command.Records

/-!
# The abstraction claims a Model's actions make

A class with an `examples:` line is an abstraction claim: the author claims several realized values
behave alike in it, and a functional Case runs the example. Which claims a Case makes is decided by
its path -- the classes it performs -- so this module pairs each claim with the machine's action
member that realizes the class, and the Producer records the ones the path performs.
-/

namespace Umpire.Command

open Umpire

/-- The claims the Model's actions make at the members of one machine's Action domain: for each
member, the example of each of its field's classes, where the action declares one. A class with no
example is no claim, and an action the machine does not step on contributes nothing. -/
def classClaims {Setup State Act Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Act] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Act Outcome Fact)
    (actions : List Umpire.Command.Action) : List Umpire.Case.Producer.ClassClaim :=
  (model.actionClasses.zip model.actionIds).flatMap fun ((actionName, fields), member) =>
    match actions.find? (·.name == actionName) with
    | none => []
    | some action => fields.filterMap fun (field, className) =>
        (action.example? field className).map fun found => {
          member
          row := { action := action.id.value, field, className,
                   exampleValue := found.member.value } }

/-- The same claims as the Model's own record, for a reader that does not need the member. -/
def AbstractionClaim.ofRow (row : Umpire.Provenance.AbstractionClaimRow) : AbstractionClaim :=
  { action := .of row.action, field := row.field, className := row.className,
    exampleValue := row.exampleValue }

end Umpire.Command
