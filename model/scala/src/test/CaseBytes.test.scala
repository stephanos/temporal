package nexuscaller

// The seven functional Cases, produced from the Scala Model through the Scala realization, against
// the checked-in fixtures Lean renders, byte for byte.

import caseproducer.*
import com.google.gson.{JsonElement, JsonParser}
import java.nio.file.Files
import scala.jdk.CollectionConverters.*
import testsupport.Lean

class CaseBytes extends munit.FunSuite:
  val realization: caseproducer.Realization = NexusRealization.asyncNexus("umpire.case.service", "complete")

  /** The first path at which two JSON documents differ, for a readable failure. */
  def firstDifference(path: String, want: JsonElement, got: JsonElement): Option[String] =
    if want.isJsonObject && got.isJsonObject then
      val (w, g) = (want.getAsJsonObject, got.getAsJsonObject)
      (w.keySet.asScala ++ g.keySet.asScala).toVector.sorted.iterator
        .flatMap(k => firstDifference(s"$path.$k", w.get(k), g.get(k))).nextOption()
    else if want != null && got != null && want.isJsonArray && got.isJsonArray then
      val (w, g) = (want.getAsJsonArray.asScala.toVector, got.getAsJsonArray.asScala.toVector)
      w.zip(g).zipWithIndex.iterator.flatMap { case ((a, b), i) => firstDifference(s"$path[$i]", a, b) }.nextOption()
        .orElse(Option.when(w.size != g.size)(s"$path: lean has ${w.size} elements, scala ${g.size}"))
    else Option.when(want != got)(s"$path: lean $want, scala $got")

  for q <- functionalQueries do
    test(s"${q.name} is byte-identical to its fixture") {
      val produced = produce(q, Identity.forQuery("temporal.case", "nexusCallerTests", q.name), realization, NexusRealization.modelSource)
        .fold(e => fail(e.toString), identity)
      val got = Render.persisted(produced)
      val want = Files.readString(Lean.root.resolve(s"tests/testcore/testpilot/testdata/nexusCallerTests-${q.name}-case.json"))
      if got != want then
        firstDifference("$", JsonParser.parseString(want), JsonParser.parseString(got)) match
          case Some(d) => fail(s"first difference: $d")
          case None    => fail(s"same JSON value, different bytes; first byte ${want.zip(got).indexWhere(_ != _)}")
    }
