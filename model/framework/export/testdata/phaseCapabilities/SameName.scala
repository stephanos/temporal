package fixture.phasecapabilities

import framework.dsl.*
import temporal.capabilities.product.*

package first:
  enum ClosePhase:
    case open extends ClosePhase, Held
    case succeeded extends ClosePhase, Succeeded

  final case class CloseState(phase: ClosePhase)

  object Same extends Machine[CloseState, Result, Nothing], Phased[CloseState, ClosePhase](_.phase):
    val init = CloseState(ClosePhase.open)
    object rules extends Rules
    object capabilities extends Capabilities:
      val closable: Capability = Closable(Result.accepted)
    object queries:
      capabilities.bound(bounds)

package second:
  enum ClosePhase:
    case open extends ClosePhase, Held
    case failed extends ClosePhase, Failed

  final case class CloseState(phase: ClosePhase)

  object Same extends Machine[CloseState, Result, Nothing], Phased[CloseState, ClosePhase](_.phase):
    val init = CloseState(ClosePhase.open)
    object rules extends Rules
    object capabilities extends Capabilities:
      val closable: Capability = Closable(Result.accepted)
    object queries:
      capabilities.bound(bounds)
