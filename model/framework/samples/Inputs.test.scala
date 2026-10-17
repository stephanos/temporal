package framework.dsl

import samples.Inputs

class InputsTest extends munit.FunSuite:
  private def run[S, O](
      machine: RecordMachine[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  test("Inputs accepts an amount within its ceiling once, by an input guard on its vars"):
    import Inputs.*
    def accepted(seen: Set[Int]): State = Accepted.init.copy(seen = seen)
    assertEquals(
      run(Accepted, Accepted.init, caller.submit(Request(0, 1))),
      List(Step(true, accepted(Set(0))))
    )
    assertEquals(run(Accepted, Accepted.init, caller.submit(Request(2, 1))), Nil)
    assertEquals(run(Accepted, accepted(Set(0)), caller.submit(Request(0, 2))), Nil)
    assertEquals(
      run(Accepted, accepted(Set(0)), caller.submit(Request(2, 2))),
      List(Step(true, accepted(Set(0, 2))))
    )
    assert(!Accepted.end(Accepted.init))
    assert(Accepted.end(accepted(Set(2))))
    def zero(step: Step[State, Boolean, Nothing]) =
      SampleClaims
        .judged(
          Accepted,
          Accepted,
          "postconditions",
          "zeroWasAccepted",
          step,
          Some(Accepted.init),
          Some(caller.submit(amount = 0))
        )
        .get
    assert(zero(Step(true, accepted(Set(0)))))
    assert(zero(Step(true, accepted(Set(0, 2)))))
    assert(!zero(Step(true, accepted(Set(2)))))

  test("Inputs' postcondition is checked by default over every input class"):
    import Inputs.*
    val report = NativeCheck.verifyAll(Accepted)
    assertEquals(report.verdicts, Map("zeroWasAccepted" -> NativeCheck.Verdict.verified))
    assertEquals(report.warnings, Vector.empty)
