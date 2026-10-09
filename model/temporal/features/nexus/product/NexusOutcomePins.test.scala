package umpire
// The Nexus and shared-worker rejection cells, recorded before they adopted the shared Outcome.

import temporal.features.nexus.handler as nexusHandler
import temporal.features.nexus.product
import temporal.features.nexus.standalone
import temporal.features.nexus.workflow
import temporal.features.nexus.workflow.system.given
import temporal.actors.worker as sharedWorker
import umpire.outcomes.Outcome

class NexusOutcomePins extends munit.FunSuite:
  final case class ExpectedStep[S, F](
      outcome: String,
      state: S,
      facts: List[F],
      because: String
  )

  final case class ExpectedRejection[S](
      decl: ActionDecl,
      inputs: Option[List[Any]],
      guard: S => Boolean,
      outcome: String,
      because: String = ""
  )

  private def run[S, F](
      binding: StepBinding[S, Outcome, F],
      state: S,
      inputs: List[Any]
  ): List[Step[S, Outcome, F]] =
    // scalafix:off DisableSyntax.asInstanceOf
    inputs match
      case Nil       => binding.function.asInstanceOf[S => List[Step[S, Outcome, F]]](state)
      case List(one) =>
        binding.function.asInstanceOf[(S, Any) => List[Step[S, Outcome, F]]](state, one)
      case List(one, two) =>
        binding.function.asInstanceOf[(S, Any, Any) => List[Step[S, Outcome, F]]](state, one, two)
      case List(one, two, three) =>
        binding.function
          .asInstanceOf[(S, Any, Any, Any) => List[Step[S, Outcome, F]]](state, one, two, three)
      case other => fail(s"the pinned Nexus action has ${other.size} inputs")
    // scalafix:on DisableSyntax.asInstanceOf

  private def observed[S, F](steps: List[Step[S, Outcome, F]]): List[ExpectedStep[S, F]] =
    steps.collect { case step @ Step(Outcome.rejected(_), _, _, _) =>
      ExpectedStep(step.outcome.toString, step.state, step.facts, step.because)
    }

  private def assertRejectingCells[S, F](
      machine: Machine[S, Outcome, F],
      rules: List[ExpectedRejection[S]],
      expectedCount: Int
  )(using states: Finite[S]): Unit =
    val rejectingCells =
      (for
        binding <- machine.bindings
        inputs <- classesOf(binding.decl)
        state <- states.values
      yield
        val matches = rules.filter(r =>
          r.decl == binding.decl && r.inputs.forall(_ == inputs) && r.guard(state)
        )
        assert(
          matches.sizeIs <= 1,
          s"several pinned rejections match ${binding.decl.name}$inputs in $state"
        )
        val expected = matches.map(r => ExpectedStep(r.outcome, state, List.empty[F], r.because))
        val actual = observed(run(binding, state, inputs))
        assertEquals(actual, expected, s"${machine.name}: ${binding.decl.name}$inputs in $state")
        actual.nonEmpty
      ).count(identity)
    assertEquals(rejectingCells, expectedCount, s"${machine.name} rejecting cells")

  private val productRejections =
    import product.Phase.*
    List(
      ExpectedRejection[product.State](
        nexusHandler.complete.decl,
        None,
        s =>
          s.phase == succeeded || s.phase == failed || s.phase == canceled ||
            s.phase == timedOut || s.phase == terminated,
        "rejected(notFound)"
      )
    )

  private val standaloneRejections =
    import standalone.system.Phase.*
    List(
      ExpectedRejection[standalone.system.State](
        standalone.client.requestCancel.decl,
        None,
        s =>
          !s.cancelRequested &&
            (s.phase == succeeded || s.phase == failed || s.phase == canceled ||
              s.phase == terminated),
        "rejected(failedPrecondition)",
        "operation already completed"
      ),
      ExpectedRejection[standalone.system.State](
        standalone.client.terminate.decl,
        None,
        s => s.phase == succeeded || s.phase == failed || s.phase == canceled,
        "rejected(failedPrecondition)",
        "operation already completed"
      )
    )

  private val workflowRejections =
    import workflow.system.Phase.*
    List(
      ExpectedRejection[workflow.system.State](
        workflow.handler.complete.decl,
        None,
        s =>
          s.phase == succeeded || s.phase == failed || s.phase == canceled ||
            s.phase == timedOut,
        "rejected(notFound)"
      )
    )

  test("the Nexus product keeps its 15 rejecting action-class-state cells"):
    assertRejectingCells(product.NexusProduct, productRejections, expectedCount = 15)

  test("the standalone Nexus System keeps its 10 rejecting action-class-state cells"):
    assertRejectingCells(
      standalone.system.NexusSystem,
      standaloneRejections,
      expectedCount = 10
    )

  test("the workflow Nexus System keeps its 288 rejecting action-class-state cells"):
    assertRejectingCells(workflow.system.NexusSystem, workflowRejections, expectedCount = 288)

  test("the shared worker keeps its empty rejection set"):
    assertRejectingCells(sharedWorker.Polling, Nil, expectedCount = 0)
