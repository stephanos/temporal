package framework

import java.nio.file.{Files, Path}

object ExportedLaws:
  lazy val files: Vector[IrFile] = IrFile.discover("temporal")
  lazy val subjects: Vector[Model] = files
    .flatMap(_.subjects())
    .groupBy(IrFile.identity)
    .toVector
    .sortBy(_._1)
    .map: (identity, instances) =>
      require(
        instances.forall(_ eq instances.head),
        s"distinct instances of $identity across exports"
      )
      instances.head
  lazy val machines: Vector[Machine[?, ?, ?]] = subjects.collect { case m: Machine[?, ?, ?] => m }
  lazy val orders: Map[String, Vector[String]] =
    val path = Path.of("model/testdata/binding-order.tsv")
    scopeOrders(
      Files.readString(path),
      files.map(_.name).toSet,
      machines.map(m => IrFile.identity(m).split('.').dropRight(1).mkString(".")).toSet
    )

  private[framework] def scopeOrders(
      text: String,
      features: Set[String],
      packages: Set[String]
  ): Map[String, Vector[String]] =
    text
      .linesIterator
      .filter(_.nonEmpty)
      .map { line =>
        val fields = line.split("\t", -1)
        fields(0) -> fields(1).split(",").filter(_.nonEmpty).toVector
      }
      .filter: (identity, _) =>
        val pkg = identity.split('.').dropRight(1).mkString(".")
        if identity.startsWith("temporal.features.") then
          IrFile.featureOf(pkg).exists(name => features.exists(file => file == name || file.startsWith(name + "-")))
        else packages.contains(pkg)
      .toMap

  lazy val results: Vector[MachineLaws.Result] =
    machines.flatMap(m => MachineLaws.check(m, orders.get(IrFile.identity(m)))) ++ subjects
      .collect { case c: Composition[?] => c }
      .flatMap(c =>
        Vector("refinement totality", "closedness", "repeatable ordered relation", "binding order")
          .map(law =>
            MachineLaws.Result(
              IrFile.identity(c),
              law,
              "inapplicable",
              "composition has no native finite transition table; every member is checked independently; composed interpreter pins belong to R7"
            )
          )
      )

class ExportedMachineLawsTest extends munit.FunSuite:
  test("binding inventory scoping retains missing extra and permuted selected entries") {
    val selected = "fixture.actors.selected"
    val machine = selected + ".Machine"
    val expected = Map(machine -> Vector("caller.submit", "worker.take"))
    val baseline = machine + "\tcaller.submit,worker.take\n"
    val unselected = "fixture.actors.unselected.Machine\tworker.take\n"
    assertEquals(ExportedLaws.scopeOrders(baseline + unselected, Set.empty, Set(selected)), expected)
    for changed <- Seq(
        baseline + selected + ".Unknown\tworker.take\n",
        "",
        machine + "\tworker.take,caller.submit\n"
      )
    do
      intercept[munit.ComparisonFailException] {
        assertEquals(ExportedLaws.scopeOrders(changed + unselected, Set.empty, Set(selected)), expected)
      }
  }

  test("every exported declaration enters each applicable law with an explicit disposition") {
    println(
      s"exported laws: ${ExportedLaws.files.size} exports, ${ExportedLaws.subjects.size} declarations, ${ExportedLaws.machines.size} machines"
    )
    for result <- ExportedLaws.results do
      println(s"${result.subject}\t${result.law}\t${result.disposition}\t${result.reason}")
    assertEquals(ExportedLaws.results.size, ExportedLaws.subjects.size * 4)
    val failures = ExportedLaws.results.filter(_.disposition.equals("failed"))
    assert(failures.isEmpty, failures.mkString("\n"))
  }

  test("every exported machine has an independently pinned declared binding order") {
    for machine <- ExportedLaws.machines do
      println(
        s"BINDING_ORDER\t${IrFile.identity(machine)}\t${MachineLaws.bindingKeys(machine).mkString(",")}"
      )
    assertEquals(ExportedLaws.orders.keySet, ExportedLaws.machines.map(IrFile.identity).toSet)
  }
