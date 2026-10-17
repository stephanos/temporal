package expectauthor

// The vocabulary an author outside the framework's packages writes a Model in: the plain form
// (outside.Lamp and the samples) compiles, and every retired word is refused.
class ExpectExternalTest extends munit.FunSuite:
  test("legacy finishers and private core construction are unavailable to authors"):
    val refusals = List(
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.holds(_ => true)"""),
      compileErrors(
        """framework.dsl.WitnessFixture.Subject.property.holdsAcross((_, _) => true)"""
      ),
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.sameStep(_ => true)"""),
      compileErrors("""framework.dsl.WitnessFixture.Subject.property.transition((_, _) => true)"""),
      compileErrors(
        """framework.dsl.WitnessFixture.Subject.property.expect((s: Step[WitnessFixture.State, WitnessFixture.Outcome, WitnessFixture.Fact]) => s.state.equals(s.state))"""
      )
    )
    assertEquals(
      refusals.zipWithIndex.collect { case (errors, index) if errors.isEmpty => index },
      Nil
    )

  test("the retired vocabulary is unavailable to authors"):
    val retired = List(
      "RecordMachine" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends RecordMachine[Boolean, Said, Note]:
          val init = false
          def end(s: Boolean) = s
          object transitions extends Transitions
      """),
      "Rules" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends Machine[Said, Note]:
          var lit = false
          object transitions extends Rules
      """),
      "property" -> compileErrors("""import framework.dsl.*; outside.Lamp.Model.property"""),
      "property(name)" -> compileErrors(
        """import framework.dsl.*; outside.Lamp.Model.property("lit")"""
      ),
      "expect" -> compileErrors("""
        import framework.dsl.*
        val p: Step[Boolean, String, Nothing] => Boolean = expect:
          check(true)
      """),
      "expect.across" -> compileErrors("""
        import framework.dsl.*
        val p: (Boolean, Step[Boolean, String, Nothing]) => Boolean = expect.across:
          check(true)
      """),
      "invariant and transition kind words" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends Machine[Said, Note]:
          var lit = false
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
          object effects:
            def press() =
              lit = !lit
          object invariants:
            val any = invariant:
              check(lit || !lit)
          object postconditions:
            val toggles = transition:
              check(lit != before.lit)
      """),
      "properties and safety groups" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends Machine[Said, Note]:
          var lit = false
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
          object effects:
            def press() =
              lit = !lit
          object properties:
            object safety:
              val any = check(lit || !lit)
      """),
      "liveness group" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends Machine[Said, Note]:
          var lit = false
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
          object effects:
            def press() =
              lit = !lit
          object liveness:
            val lights = leadsTo(from = !lit, to = lit, within = 1)
      """),
      "after.state" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object M extends Machine[Said, Note]:
          var lit = false
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
          object effects:
            def press() =
              lit = !lit
          object postconditions:
            val toggles = check(after.state != before)
      """),
      "~>" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        Model.rebind(hand.press ~> Model.effects.press())
      """),
      "always" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        Model.rebind(
          on(hand.press):
            always ~> Model.effects.press()
        )
      """),
      "where" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        Model.rebind(
          on(hand.press):
            where(Model.lit) ~> Model.effects.press()
        )
      """),
      "rejects" -> compileErrors("""
        import framework.dsl.*
        import framework.dsl.outcomes.Outcome
        import outside.Lamp.hand
        object M extends Machine[Outcome, Nothing]:
          var lit = false
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
          object effects:
            def press() =
              lit = !lit
        M.rebind(
          on(hand.press):
            rejects(outcomes.Rejection.notFound)
        )
      """),
      "enter" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        enter[Boolean, Said, Note](true)
      """),
      "query" -> compileErrors("""import framework.dsl.*; query"""),
      "query(name)" -> compileErrors("""import framework.dsl.*; query("lighting")"""),
      "limits" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        verify(Model.invariants.bounded) limits Model.queries.bounding.limits
      """),
      "Limits" -> compileErrors(
        """import framework.dsl.*; Limits(steps = 3, actions = 3, search = 64)"""
      ),
      "frameworkReach" -> compileErrors("""import framework.dsl.*; frameworkReach"""),
      "Witness.limits" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        Model.witness(hand.press).records(Note.lit).limits(Model.queries.bounding.limits)
      """),
      "assume" -> compileErrors("""import framework.dsl.*; framework.dsl.assume("lossy")"""),
      "assume (named after its val)" -> compileErrors(
        """import framework.dsl.*; val lossy = framework.dsl.assume"""
      ),
      "assuming" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        Model.assuming()
      """),
      "leadsTo(name)(from, to, within, under*)" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        Model.leadsTo("wearsOut")(Model.presses == 0, Model.presses == 3, 3)
      """),
      "restrict" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        Model.restrict(hand.press)
      """),
      "internal" -> compileErrors("""import framework.dsl.*; framework.dsl.internal"""),
      "action(this)" -> compileErrors("""
        import framework.dsl.*
        object hand extends Actor:
          val press = framework.dsl.action(this)
      """)
    )
    assertEquals(retired.collect { case (word, errors) if errors.isEmpty => word }, Nil)
    val limits = retired.toMap.apply("limits")
    assert(limits.contains("a Model declares no Limits"), limits)

  test("the plain form's words compile for authors"):
    val kept = List(
      "hole, choose, disabled and crashes" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.{*, given}
        object fault extends Actor:
          val drop = action.crash
        object M extends Machine[Said, Note]:
          var lit = false
          durability(durable(lit))
          val unknown = hole
          object transitions extends Transitions:
            on(hand.press):
              effects.press()
            disabled(hand.set)
            crashes(fault.drop, Said.ok)
          object effects:
            val on = choice
            val off = choice
            def press() =
              choose(on -> { lit = true }, off -> { lit = false })
      """),
      "witness" -> compileErrors("""
        import framework.dsl.*
        import outside.Lamp.*
        Model.witness(hand.press).records(Note.lit)
      """)
    )
    assertEquals(kept.filter(_._2.nonEmpty), Nil)
