package temporal
package parity
// The canonical strings and Behavior Fingerprints against Lean's, byte for byte.

import nexuscaller.*
import testsupport.Lean
import testsupport.Lean.*
import umpire.*
import umpire.Canonical.*

class CanonicalParity extends munit.FunSuite:
  /** The semantic object sliced out of a canonical-metadata string without re-encoding it. */
  def semanticOf(metadata: String): String =
    val prefix = """{"semantic":"""
    assert(metadata.startsWith(prefix))
    val depths =
      metadata.indices.drop(prefix.length).iterator.scanLeft((-1, 0)) { case ((_, depth), i) =>
        metadata(i) match
          case '{' => (i, depth + 1)
          case '}' => (i, depth - 1)
          case _   => (i, depth)
      }
    val end = depths
      .drop(1)
      .collectFirst { case (i, 0) if metadata(i) == '}' => i }
      .getOrElse(fail("unterminated semantic object"))
    metadata.substring(prefix.length, end + 1)

  def sameString(what: String, want: String, got: String): Unit =
    if want != got then
      val i = want.zip(got).indexWhere(_ != _) match
        case -1 => want.length.min(got.length)
        case j  => j
      val lo = (i - 120).max(0)
      fail(
        s"$what differs at byte $i:\n lean: …${want.slice(lo, i + 120)}\nscala: …${got.slice(lo, i + 120)}"
      )

  val protocol: Table = nexusProtocol.table.fold(e => fail(e.toString), identity)

  test("target semantic and fingerprint") {
    sameString(
      "target semantic",
      semanticOf(Lean.text("canonical-target-nexusProtocol.txt").trim),
      protocol.targetSemantic
    )
    assertEquals(
      protocol.targetFingerprint,
      "sha256:b38647500819c04cd82e179972ed23c76609d69e99a424748596295ce067af14"
    )
  }

  for q <- functionalQueries do
    test(s"scenario, property and query of ${q.name}") {
      val want = Lean.json(s"canonical-${q.name}.json")
      val scenario = q.scenario.scenarioSemantic(protocol)
      sameString("scenario semantic", semanticOf(want("scenario").str), scenario)
      assertEquals(fingerprint(scenario), want("scenarioFingerprint").str)
      val groups = Lower(q.property).fold(e => fail(e.toString), identity)
      val property = protocol.propertySemantic(q.property.propertyID(protocol), groups)
      sameString("property semantic", semanticOf(want("property").str), property)
      assertEquals(fingerprint(property), want("propertyFingerprint").str)
      val canonical = q.queryCanonical(protocol, fingerprint(property))
      sameString("query canonical", want("queryCanonical").str, canonical)
      assertEquals(fingerprint(canonical), want("queryFingerprint").str)
    }
