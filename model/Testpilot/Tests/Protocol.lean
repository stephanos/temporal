import Testpilot.Protocol

/-! Narrow-import checks for the generated Testpilot protocol boundary. -/

open temporal.server.api.testpilot.v1

namespace Testpilot.Tests.Protocol

private def instructionGuard : Expression :=
  { expression := some (.reference { reference := some (.slot_id "slot") }) }

private def observation : Expression :=
  { expression := some (.reference { reference := some (.observation_id "observation") }) }

private def transitionPredicate : Expression :=
  { expression := some (.not { operand := some observation }) }

/-! Program and Contract expressions are one type, so either can stand where the other appears;
Go preparation, not the type, checks each reference against its context. -/

private def guardReadingAnObservation : InstructionNode :=
  { instruction_id := "node", guard := some transitionPredicate }

private def predicateReadingASlot : ContractTransition :=
  { transition_id := "transition", predicate := some instructionGuard }

#guard instructionGuard.expression.isSome
#guard transitionPredicate.expression.isSome
#guard guardReadingAnObservation.guard.isSome
#guard predicateReadingASlot.predicate.isSome

end Testpilot.Tests.Protocol
