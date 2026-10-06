package umpire.irgen

import io.temporal.server.api.umpire.v1 as ir

/**
 * The whole-run check of the IDs a lift derives from names (fn-126 decision 23): a machine's
 * `<family>.target.<name>` and a Query's `<family>.query.<name>`, the family its package. Scala names
 * no two declarations alike, but two files of one package may each name a Query `completion`; one
 * declaration lifted into several IR files is one.
 */
class DerivedIds extends munit.FunSuite:
  private def at(file: String, line: Int) = Some(ir.Position(file = file, line = line))
  private val lamp =
    ir.Machine(family = "fixture.lamp", name = "lamp", position = at("Lamp.scala", 3))
  private def queryAt(file: String, line: Int) =
    ir.Query(name = "lit", position = at(file, line), scenario = Some(ir.ClaimRef("lamp", "s")))
  private def model(queries: ir.Query*) = ir.Model(machines = Seq(lamp), queries = queries)

  test("one declaration lifted into two IR files derives its IDs once"):
    val one = model(queryAt("Lamp.scala", 9))
    assertEquals(derivedIdTwins(Seq("a" -> one, "b" -> one)), Nil)

  test("two Queries of one name in one package are refused at the later, across IR files"):
    val refused = derivedIdTwins(
      Seq("a" -> model(queryAt("Lamp.scala", 9)), "b" -> model(queryAt("Other.scala", 4)))
    )
    assertEquals(
      refused.map(_.getMessage),
      Seq(
        "Other.scala:4: this declaration and the one at Lamp.scala:9 derive the ID " +
          "fixture.lamp.query.lit: name them apart, since an ID derived from a name is unique in " +
          "its package"
      )
    )

  test("two machines of one name in one package are refused, and in two packages are not"):
    val twin = lamp.withPosition(ir.Position("Twin.scala", 5))
    val apart = twin.withFamily("fixture.other")
    assertEquals(
      derivedIdTwins(
        Seq("a" -> ir.Model(machines = Seq(lamp)), "b" -> ir.Model(machines = Seq(twin)))
      )
        .map(_.position),
      Seq("Twin.scala:5")
    )
    assertEquals(
      derivedIdTwins(
        Seq("a" -> ir.Model(machines = Seq(lamp)), "b" -> ir.Model(machines = Seq(apart)))
      ),
      Nil
    )
