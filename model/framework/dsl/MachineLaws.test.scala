package framework.dsl

import java.util.Objects
import scala.util.Try

object MachineLaws:
  final case class Result(subject: String, law: String, disposition: String, reason: String)

  private def finiteStates[S](data: Data[S]): Finite[S] = data match
    case finite: Finite[S] => finite
    case _ => throw IllegalArgumentException("machine laws require explicit Finite state evidence")

  def bindingKeys[S, O, F](machine: Machine[S, O, F]): Vector[String] =
    machine.table.map: (decl, bound) =>
      val action = bound match
        case Bound.Ruled(rules) => rules.head.action
        case _                  =>
          val owners = Seq(decl.actor, machine)
          val names = owners.flatMap: owner =>
            owner.getClass.getDeclaredMethods.toVector
              .filter(m =>
                m.getParameterCount.equals(0) && classOf[Action[?]]
                  .isAssignableFrom(m.getReturnType)
              )
              .flatMap(m =>
                m.invoke(owner) match
                  case a: Action[?] if a.decl eq decl => Some(m.getName)
                  case _                              => None
              )
          names.sorted.headOption.getOrElse(decl.name)
      s"${decl.actor.name}.$action"

  def check[S, O, F](machine: Machine[S, O, F], order: Option[Vector[String]]): Vector[Result] =
    val id = IrFile.identity(machine)
    def result(law: String)(run: => Option[String]): Result =
      Try(run).fold(
        e => Result(id, law, "failed", s"${e.getClass.getSimpleName}: ${e.getMessage}"),
        reason =>
          Result(
            id,
            law,
            if reason.isEmpty then "passed" else "inapplicable",
            reason.getOrElse("all finite values checked")
          )
      )
    val totality = result("refinement totality"):
      machine.refinementEvidence match
        case None    => Some("no refinement declared by this subject or derivation")
        case Some(r) =>
          val images = finiteStates(r.of.fs).values.toSet
          for state <- finiteStates(machine.fs).values do
            val image = r.toProduct(state)
            require(images.contains(image), s"$state maps outside ${IrFile.identity(r.of)}: $image")
          None
    val closedness = result("closedness"):
      machine.refinementEvidence match
        case None    => Some("no refinement declared by this subject or derivation")
        case Some(r) =>
          (machine.phaseEvidence, r.of.phaseEvidence) match
            case (Some(phase), Some(productPhase)) =>
              val sourceHasRoles = finiteStates(machine.fs).values.exists(s => hasRole(phase(s)))
              val targetHasRoles =
                finiteStates(r.of.fs).values.exists(s => hasRole(productPhase(s)))
              if !sourceHasRoles || !targetHasRoles then
                Some("source or target phase carries no roles")
              else
                val differences = Refinement.unclosed(r)(phase, productPhase)(using
                  finiteStates(machine.fs)
                )
                require(
                  differences.isEmpty,
                  s"states map with a different Closed role: $differences"
                )
                None
            case _ => Some("source or target declares no phase projection")
    val relation = result("repeatable ordered relation"):
      val finite = finiteStates(machine.fs)
      val states = finite.values.toSet
      val outcomes = machine.fo.values.toSet
      val facts = machine.ff.values.toSet
      var unknownRows = 0
      def read(run: => List[Step[S, O, F]]): Either[Hole, List[Step[S, O, F]]] =
        try Right(run)
        catch case HoleReached(hole) => Left(hole)
      for binding <- machine.bindings do
        val run = effectOf[S, O, F](binding.decl, binding.function)
        for inputs <- classesOf(binding.decl); state <- finite.values do
          val first = read(run(state, inputs))
          val second = read(run(state, inputs))
          require(
            (first, second) match
              case (Left(a), Left(b))   => a eq b
              case (Right(a), Right(b)) => Objects.equals(a, b)
              case _                    => false,
            s"$state / $inputs repeats different ordered alternatives: $first / $second"
          )
          if first.isLeft then unknownRows += 1
          for steps <- first.toOption; step <- steps do
            require(
              states.contains(step.state),
              s"$state / $inputs leaves the finite state domain: ${step.state}"
            )
            require(
              outcomes.contains(step.outcome),
              s"$state / $inputs answers outside the outcome domain: ${step.outcome}"
            )
            require(
              step.facts.forall(facts.contains),
              s"$state / $inputs records outside the fact domain: ${step.facts}"
            )
      if unknownRows == 0 then None
      else Some(s"$unknownRows declared row holes remain unknown; all known rows checked")
    val bindings = result("binding order"):
      val declarations = machine.bindings.map(_.decl)
      require(
        declarations.distinct.size.equals(declarations.size),
        "one action is bound more than once"
      )
      order match
        case None =>
          throw IllegalStateException("missing independent binding-order inventory entry")
        case Some(expected) =>
          val actual = bindingKeys(machine)
          require(
            Objects.equals(actual, expected),
            s"declared actor-group/binding order differs: $actual; pinned: $expected"
          )
          None
    Vector(totality, closedness, relation, bindings)

  private def hasRole(phase: Any): Boolean = phase match
    case _: Closed | _: Live => true
    case _                   => false
