package fixture.binding

import framework.*
// Its actions name no message: a realization binding derives what a class carries (fn-133.8).

enum Outcome derives Finite:
  case accepted
enum Fact derives Finite:
  case taken
final case class State(done: Boolean) derives Finite
enum Resolution derives Finite:
  case done
  case failed(retryable: Boolean)

val ready = input[Boolean]
val resolution = input[Resolution]

object worker extends Actor:
  val take = action(this)
    .input(ready)
    .input(resolution)
    .results("Delivery")
  val respond = action(this)
    .input(resolution)
    .example(Resolution.failed(false), "NonRetryable")
    .example(Resolution.failed(true), "Retryable")
object caller extends Actor:
  val create = action(this)

object workflowBinding:
  val job = Entity(name = "job", key = "scheduledEvent")
  val take = worker.take.on(job)
  val respond = worker.respond.on(job)
  val create = caller.create.creates(job)

object standaloneBinding:
  val job = Entity(name = "job", key = "jobId")
  val take = worker.take.on(job)
  val respond = worker.respond.on(job)
  val create = caller.create.creates(job)

object WorkflowForm extends Machine[State, Outcome, Fact]:
  val init = State(false)
  def end(s: State) = s.done
  object rules
      extends Bindings(
        workflowBinding.create ~> (s => List(Step(Outcome.accepted, s))),
        workflowBinding.respond ~> ((s, _) => List(Step(Outcome.accepted, s))),
        workflowBinding.take ~> ((s, ready, _) =>
          if ready then List(Step(Outcome.accepted, s.copy(done = true), List(Fact.taken))) else Nil
        )
      )

object StandaloneForm extends Machine[State, Outcome, Fact]:
  val init = State(false)
  def end(s: State) = s.done
  object rules
      extends Bindings(
        standaloneBinding.create ~> (s => List(Step(Outcome.accepted, s))),
        standaloneBinding.respond ~> ((s, _) => List(Step(Outcome.accepted, s))),
        standaloneBinding.take ~> ((s, ready, _) =>
          if ready then List(Step(Outcome.accepted, s.copy(done = true), List(Fact.taken))) else Nil
        )
      )

object wrongBinding:
  val alien = Entity()
  val take = worker.take.on(alien)

object WrongForm extends Machine[State, Outcome, Fact]:
  val init = State(false)
  def end(s: State) = s.done
  object rules
      extends Bindings(
        workflowBinding.create ~> (s => List(Step(Outcome.accepted, s))),
        wrongBinding.take ~> ((s, _, _) => List(Step(Outcome.accepted, s)))
      )
