package fixture.phasecapabilities

import framework.*
import temporal.capabilities.*

package first:
  enum ClosePhase derives Finite:
    case open extends ClosePhase, Held
    case succeeded extends ClosePhase, Succeeded

  final case class CloseState(phase: ClosePhase) derives Finite

  object Same extends Machine[CloseState, Result, Nothing], Phased[CloseState, ClosePhase](_.phase):
    val init = CloseState(ClosePhase.open)
    object rules extends Rules
    object capabilities extends Capabilities:
      val closable: Capability = Closable(Result.accepted)
    object queries:
      capabilities.bound(bounds)

package second:
  enum ClosePhase derives Finite:
    case open extends ClosePhase, Held
    case failed extends ClosePhase, Failed

  final case class CloseState(phase: ClosePhase) derives Finite

  object Same extends Machine[CloseState, Result, Nothing], Phased[CloseState, ClosePhase](_.phase):
    val init = CloseState(ClosePhase.open)
    object rules extends Rules
    object capabilities extends Capabilities:
      val closable: Capability = Closable(Result.accepted)
    object queries:
      capabilities.bound(bounds)
