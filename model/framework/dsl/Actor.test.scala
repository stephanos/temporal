package framework.dsl

class ActorTest extends munit.FunSuite:
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
