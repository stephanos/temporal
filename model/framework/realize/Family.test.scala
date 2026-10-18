// A feature package supplies its Family once; a kit function that forms a Definition ID takes it as

// a using parameter. The fixture packages are those of the retained export output
// model/framework/export/testdata/constructed/family-original.json, whose IDs they must reproduce.
// A feature writes `given Family = Family.here` at its package's top level; a test file's name has
// a dot, so here each package holds it in an object instead (`feature`).
package framework.realize {
  import framework.dsl.Family

  // A kit: the family is the caller's, never the kit's own.
  object familyKit:
    def familyVocabulary()(using f: Family): String = f.id + ".vocabulary"

  // The machine the fixtures' realizations are of.
  object FamilySubject extends framework.dsl.Machine[Boolean, Boolean]:
    var ready = false
    override def end: Boolean = ready

    object transitions
        extends framework.dsl.Transitions(using summon, framework.dsl.Ok(true), summon)

  class FamilyTest extends munit.FunSuite:
    import scala.compiletime.testing.typeCheckErrors

    test("a feature's given Family roots the Definition IDs it and its kit form, as retained"):
      val retained = java.nio.file.Files.readString(
        java.nio.file.Path.of("model/framework/export/testdata/constructed/family-original.json")
      )
      for root <- Vector(fixture.first.feature.root, fixture.second.feature.root) do
        val pkg = root.name match
          case "first" => "fixture.first"
          case _       => "fixture.second"
        val value = root.toIR(s"$pkg.root")
        val start = retained.indexOf(s"\"id\": \"$pkg.root\"")
        val entry = retained.slice(start, retained.indexOf("\n  }", start))
        val correlation = value.getCorrelation
        for (field, id) <- Vector(
            "id" -> value.id,
            "producer" -> value.producer,
            "producerVersion" -> value.producerVersion,
            "projection" -> correlation.projection,
            "run" -> correlation.run,
            "operation" -> correlation.operation,
            "observation" -> correlation.observation,
            "cleanup" -> value.cleanup
          )
        do assert(entry.contains(s"\"$field\": \"$id\""), s"$pkg $field $id not in $entry")
        assertEquals(value.producer, s"$pkg.direct")
        assertEquals(correlation.projection, "fixture.shared.owned")
        assertEquals(correlation.operation, s"$pkg.vocabulary")

    test("Family.here in a framework package refuses, naming the package"):
      assertEquals(
        typeCheckErrors("framework.dsl.Family.here").map(_.message),
        List(
          "Family.here in framework.realize: a family names a feature, not the framework or a realization kit"
        )
      )

    test("Family.here in a realization kit's package refuses, naming the package"):
      assertEquals(
        fixture.realize.kitProbe.errors,
        List(
          "Family.here in fixture.realize: a family names a feature, not the framework or a realization kit"
        )
      )

    test("a kit function with no Family in scope is the ordinary missing given"):
      val errors = typeCheckErrors("familyKit.familyVocabulary()").map(_.message)
      assertEquals(errors.size, 1, errors)
      assert(errors.head.contains("No given instance of type framework.dsl.Family"), errors)
}

package fixture.realize {

  object kitProbe:
    val errors =
      scala.compiletime.testing.typeCheckErrors("framework.dsl.Family.here").map(_.message)
}

package fixture.shared {
  import framework.dsl.Family

  object feature:
    given Family = Family.here
  import feature.given

  object Helpers:
    val shared = summon[Family].id + ".shared"
    def inner(prefix: String): String = prefix + ".inner"
    def outer(prefix: String): String = inner(prefix)
    def owned(): String = summon[Family].id + ".owned"
    def curried(prefix: String)(suffix: String): String = prefix + suffix
}

package fixture.first {
  import framework.dsl.Family
  import framework.realize.{Correlation, Realization}
  import framework.realize.familyKit.familyVocabulary
  import fixture.shared.Helpers

  object feature:
    given Family = Family.here
    val root = Realization(
      name = "first",
      machine = framework.realize.FamilySubject,
      producer = summon[Family].id + ".direct",
      producerVersion = Helpers.curried(Helpers.outer(summon[Family].id))(""),
      roles = Vector.empty,
      correlation = Correlation(
        Helpers.owned(),
        Helpers.shared,
        familyVocabulary(),
        ".literal",
        8,
        8,
        2,
        16,
        1000,
        512
      ),
      scripts = Vector.empty,
      cleanup = ".payload"
    )
}

package fixture.second {
  import framework.dsl.Family
  import framework.realize.{Correlation, Realization}
  import framework.realize.familyKit.familyVocabulary
  import fixture.shared.Helpers

  object feature:
    given Family = Family.here
    val root = Realization(
      name = "second",
      machine = framework.realize.FamilySubject,
      producer = summon[Family].id + ".direct",
      producerVersion = Helpers.curried(Helpers.outer(summon[Family].id))(""),
      roles = Vector.empty,
      correlation = Correlation(
        Helpers.owned(),
        Helpers.shared,
        familyVocabulary(),
        ".literal",
        8,
        8,
        2,
        16,
        1000,
        512
      ),
      scripts = Vector.empty,
      cleanup = ".payload"
    )
}
