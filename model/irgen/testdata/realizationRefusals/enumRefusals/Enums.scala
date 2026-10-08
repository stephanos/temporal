package umpire.realize

enum TimeoutBasis:
  case unknown
enum WithholdingMode:
  case unknown
enum Activation:
  case Controller
final case class ServerStep(timeoutBasis: TimeoutBasis)
final case class AttemptWithheld(mode: WithholdingMode)
final case class Command(id: String, attemptWithheld: AttemptWithheld)
final case class Item(command: Command)
final case class Script(id: String, activation: Activation, items: Vector[Item])
final case class Realization(
    name: String,
    serverSteps: Vector[ServerStep] = Vector.empty,
    scripts: Vector[Script] = Vector.empty
)
