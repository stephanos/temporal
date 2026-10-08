package fixture.enumrefusals

import umpire.realize.*

val unknownBasis = Realization(
  "unknownBasis",
  serverSteps = Vector(ServerStep(TimeoutBasis.unknown))
)
val unknownMode = Realization(
  "unknownMode",
  scripts = Vector(
    Script(
      "controller",
      Activation.Controller,
      Vector(Item(Command("withhold", AttemptWithheld(WithholdingMode.unknown))))
    )
  )
)
