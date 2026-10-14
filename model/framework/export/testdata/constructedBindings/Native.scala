package framework.bindingnative

import framework.dsl.*
import fixture.bindingdiscovery.*
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

object Native:
  def main(args: Array[String]): Unit =
    val machines = Seq(
      Base,
      Named,
      Eta,
      Forwarding,
      Explicit,
      ActionAlias,
      Replaced,
      Added,
      Restricted,
      Unmonitored,
      Chained,
      DuplicateInitial,
      DuplicateAnonymous
    )
    val contributions = machines.map(m => m.name -> CoreBindings.collect(m))
    val functions =
      contributions.flatMap(_._2.functions).groupBy(_.name).toVector.sortBy(_._1).map {
        (_, values) =>
          assert(values.distinct.size == 1)
          values.head
      }
    val types =
      contributions.flatMap(_._2.types).groupBy(_.name).toVector.sortBy(_._1).map { (_, values) =>
        assert(values.distinct.size == 1)
        values.head
      }
    val model = ir.Model(
      machines = contributions
        .sortBy(_._1)
        .map((name, value) => ir.Machine(name = name, steps = value.steps)),
      functions = functions,
      types = types
    )
    println("bindings:" + Base64.getEncoder.encodeToString(model.toByteArray))
    for machine <- Seq(
        DuplicateReplace,
        DuplicateAdd,
        UnknownReplace,
        AlreadyBound,
        InvalidThenRestrict,
        UnchangedCopy,
        ChangedCallableCopy,
        ChangedDeclarationCopy
      )
    do
      val refused = try
        CoreBindings.collect(machine): Unit
        false
      catch
        case error: IllegalArgumentException =>
          assert(error.getMessage.contains("core bindings"))
          true
      assert(refused, s"${machine.name} accepted an old-refused binding shape")
    val oldRefusedAlias = CoreBindings.collect(ValAlias)
    assert(
      oldRefusedAlias.steps.head.function == "fixture.bindingdiscovery.Bindings$package$.namedStep"
    )
    val retained = Base.bindings.head
    assert(retained.productArity == 2)
    assert(retained.equals(StepBinding[State, Outcome, Nothing](retained.decl, retained.function)))
    val StepBinding(decl, callable) = retained
    assert(decl == retained.decl)
    assert(callable.eq(retained.function))
    assert(retained.copy().captured.isEmpty)
    assert(retained.copy(function = changedStep).captured.isEmpty)
    assert(retained.copy(decl = other.decl).captured.isEmpty)
    assert(Base.bindings.head.captured.get.root == retained.captured.get.root)
    assert(
      DuplicateAnonymous.bindings
        .map(
          _.function
            .asInstanceOf[State => List[Step[State, Outcome, Nothing]]](State.idle)
            .head
            .state
        )
        .equals(List(State.idle, State.done))
    )
    println("controls:passed; val-function-alias:native-only; copies:old-refused")
