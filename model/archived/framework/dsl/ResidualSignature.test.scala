package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class ResidualSignatureSuite extends munit.FunSuite:
  test("an explicitly empty Action name is preserved by modifier copies"):
    val id = "fixture.empty"
    val original = action("", Actor("caller")).declaredAt(id).retainName(id, "empty")
    val modified = original.results("reply").input[Boolean](name = "flag")
    modified.retainInputSchema(
      0,
      "scala.Boolean",
      Seq(Base64.getEncoder.encodeToString(ir.TypeRef(ir.TypeRef.Ref.Bool(Empty())).toByteArray)),
      Nil,
      Nil
    ): Unit
    assertEquals(original.toIR.name, "")
    assertEquals(modified.toIR.name, "")

  test("Hole conversion preserves reached behavior and local collection deduplicates definitions"):
    val value = hole.retainName("fixture.unknown", "unknown")
    assertEquals(value.toIR.id, "fixture.unknown")
    assertEquals(value.toIR.name, "unknown")
    val reached = intercept[HoleReached](value.reached)
    assert(reached.hole eq value)
    val collected = Signature.collect(Nil, Nil, holes = List(value, value))
    assertEquals(collected.holes.size, 1)

  test("same-kind residual name conflicts refuse while identical contributions coalesce"):
    val first = new EnvironmentAssumption("same").retainName("fixture.first", "first")
    val second = new EnvironmentAssumption("same").retainName("fixture.second", "second")
    assertEquals(Signature.collect(Nil, Nil, assumptions = List(first, first)).assumptions.size, 1)
    val error = intercept[IllegalArgumentException](
      Signature.collect(Nil, Nil, assumptions = List(first, second))
    )
    assert(error.getMessage.contains("assumptions"))
    val one = hole.retainName("fixture.one", "same")
    val two = hole.retainName("fixture.two", "same")
    intercept[IllegalArgumentException](Signature.collect(Nil, Nil, holes = List(one, two))): Unit
    assertEquals(
      Signature.collect(Nil, Nil, assumptions = List(first), holes = List(one)).holes.size,
      1
    )

  test("private Query bounds retain validation names copies and collection conflicts"):
    val zero = QueryBounds("", 0, 0, 0).retainExplicitName(true)
    assertEquals(zero.toIR.name, "")
    assertEquals(zero.toIR.steps, 0)
    for (bound, value) <- List(
        "steps" -> QueryBounds("bounds", -1, 0, 0),
        "actions" -> QueryBounds("bounds", 0, -1, 0),
        "search" -> QueryBounds("bounds", 0, 0, -1)
      )
    do
      val error = intercept[IllegalArgumentException](value.toIR)
      assert(error.getMessage.contains("limits bounds"), error.getMessage)
      assert(error.getMessage.contains(bound), error.getMessage)
    Signature.collect(Nil, Nil, limits = List(zero, zero)): Unit
    intercept[IllegalArgumentException](
      Signature.collect(Nil, Nil, limits = List(zero, zero.copy(steps = 1)))
    ): Unit
    intercept[IllegalArgumentException](QueryBounds(steps = 0, actions = 0, search = 0).toIR): Unit
    val named = QueryBounds(steps = 0, actions = 0, search = 0)
      .retainExplicitName(false)
      .retainName("fixture.bounds", "bounds")
    assertEquals(named.toIR.name, "bounds")
    val copy = named.copy(steps = 1)
    assertEquals(copy.toIR.name, "bounds")
    assertEquals(copy.declarationName.identity("bounds copy"), "fixture.bounds")
    intercept[IllegalArgumentException](named.retainExplicitName(true)): Unit
    intercept[IllegalArgumentException](named.retainName("fixture.bounds", "other")): Unit

  test("missing, conflicting and anonymous retained metadata refuses contextually"):
    val missing =
      intercept[IllegalArgumentException](Signature.assumption(new EnvironmentAssumption("opaque")))
    assert(missing.getMessage.contains("assumption opaque"))
    intercept[IllegalArgumentException](hole.toIR): Unit
    val anonymous = hole
      .retainName("fixture.given_Hole", "given_Hole")
      .retainAnonymousName("fixture.given_Hole")
    intercept[IllegalArgumentException](anonymous.toIR): Unit
    val ordinary = hole.retainName("fixture.given_token", "given_token")
    assertEquals(ordinary.toIR.name, "given_token")

  test("missing Entity reference metadata retains the Action declaration context"):
    val value = action("missing", Actor("caller")).on(Entity()).declaredAt("fixture.missing")
    val error = intercept[IllegalArgumentException](value.toIR)
    assert(error.getMessage.contains("action fixture.missing"), error.getMessage)

  test("case-class copies preserve raw values, products and retained reference names"):
    val entity = Entity(key = "id").retainExplicitName(false).retainName("fixture.entity", "entity")
    val copy = entity.copy(key = "changed")
    val empty = entity.copy(name = "").retainCopyName(true)
    assertEquals(entity.name, "")
    assertEquals(copy.sourceName, "entity")
    assertEquals(empty.sourceName, "")
    assertEquals(entity.productElementNames.toList, List("name", "key", "refer"))
    val Entity(raw, key, refer) = entity
    assertEquals((raw, key, refer), ("", "id", Map.empty[String, Entity]))
    assertEquals(entity, Entity(key = "id"))
    val observation =
      Observation(on = entity).retainExplicitName(false).retainName("fixture.read", "read")
    assertEquals(observation.copy().sourceName, "read")
    assertEquals(observation.copy(name = "").retainCopyName(true).sourceName, "")
    assertEquals(observation.productArity, 2)
    val Observation(observationRaw, observedEntity) = observation
    assertEquals(observationRaw, "")
    assert(observedEntity eq entity)
    assertEquals(observation, Observation(on = entity))
