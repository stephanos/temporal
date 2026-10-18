package framework.dsl

object ActorFixture:

  object user extends Actor:
    val press = action

  object heater extends Internal:
    val heat = step
    val expire = timer

class ActorTest extends munit.FunSuite:
  import ActorFixture.*

  private val placement =
    "an action is declared in an Actor object; a step the system takes on its own in an Internal object"

  private def assertPlaced(errors: String): Unit =
    assert(errors.contains(placement), errors)

  test("an empty in-place Actor retains its exact constructor name in IR"):
    val actor = Actor("")
    assertEquals(action("empty", actor).declaredAt("fixture.empty").toIR.actor, "")
    assertEquals(actor.name, "actor")
    assertEquals(actor.toString, "actor")

  test("a module's retained source name overrides its constructor only in IR"):
    object Named extends Actor("public")
    Named.retainActorName("named"): Unit
    assertEquals(action("named", Named).declaredAt("fixture.named").toIR.actor, "named")
    assertEquals(Named.name, "public")

  test("IR reads no authored getter and names missing or conflicting metadata with its Action"):
    var reads = 0
    val actor = new Actor("constructor"):
      override def name: String =
        reads += 1
        "getter"
    val value = action("read", actor).declaredAt("fixture.read")
    val missing = intercept[IllegalArgumentException](value.toIR)
    assert(missing.getMessage.contains("action fixture.read"))
    assert(missing.getMessage.contains("no retained Actor module name"))
    actor.retainActorName("source"): Unit
    assertEquals(value.toIR.actor, "source")
    assertEquals(reads, 0)
    assertEquals(actor.name, "getter")
    assertEquals(actor.toString, "getter")
    assertEquals(reads, 2)
    actor.retainActorName("other"): Unit
    val conflict = intercept[IllegalArgumentException](value.toIR)
    assert(conflict.getMessage.contains("action fixture.read"))
    assert(conflict.getMessage.contains("conflicting retained Actor module names"))

  test("native direct constructor strings include explicit and omitted empty values"):
    for (actor, expected) <- Seq(
        new Actor("") -> "",
        new Actor() -> "",
        new Actor(named = "nonempty") -> "nonempty"
      )
    do
      val value = action("constructor", actor).declaredAt("fixture.constructor")
      assertEquals(value.toIR.actor, expected)

  test("an Actor object declares its actions and an Internal object its steps and timers"):
    assertEquals(user.press.decl.actor, user)
    assert(!user.press.decl.internal && !user.press.decl.timer)
    assertEquals(heater.heat.decl.actor, Actor.system)
    assert(heater.heat.decl.internal && !heater.heat.decl.timer)
    assertEquals(heater.expire.decl.actor, Actor.system)
    assert(heater.expire.decl.timer && !heater.expire.decl.internal)

  test("an action outside an Actor object is refused with the placement text"):
    assertPlaced(compileErrors("object loose:\n  val press = action"))

  test("a step or timer outside an Internal object is refused with the placement text"):
    assertPlaced(compileErrors("object loose:\n  val heat = step"))
    assertPlaced(compileErrors("object loose:\n  val expire = timer"))

  test("an action in an Internal object is refused with the placement text"):
    assertPlaced(compileErrors("object heater extends Internal:\n  val press = action"))

  test("a step or timer in an Actor object is refused with the placement text"):
    assertPlaced(compileErrors("object user extends Actor:\n  val heat = step"))
    assertPlaced(compileErrors("object user extends Actor:\n  val expire = timer"))
