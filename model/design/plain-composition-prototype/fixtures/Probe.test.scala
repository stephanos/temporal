package framework.dsl
package compose

// The probe's entry points for its suite and report: a composition's default check, its witnesses,
// and a negative control written where it is checked.
object Probe:
  // Every invariant and postcondition of `sections` (the composition's own unless named) and every
  // witness of its queries, checked on the composition's reachable graph, with the vacuity
  // warnings: postconditions no step triggers, composed steps that never fire.
  def checkAll[F](
      composition: Composition[F],
      sections: Option[AnyRef] = None
  ): (ComposeRuntime.Search[F], Vector[NativeCheck.Result], Vector[String]) =
    val search = ComposeRuntime[F](composition).search()
    val owner = sections.getOrElse(composition)
    val (claims, vacuous) = ComposeRuntime.verifyAll(search, owner)
    (search, claims ++ ComposeRuntime.witnesses(search, owner), ComposeRuntime.idle(search) ++ vacuous)

  // `invariant(check(...))`: an invariant written in a test.
  inline def invariant(inline body: Unit): NativeClaims.Adhoc =
    NativeClaims.Adhoc(
      NativeLocation.declaredName.filterNot(_.startsWith("<")).getOrElse("invariant"),
      () => body
    )

  // A negative control: the claim is refuted on the composition, with a counterexample.
  def assertRefuted[F](composition: Composition[F], claim: NativeClaims.Adhoc)(using
      munit.Location
  ): NativeCheck.Result =
    val result = ComposeRuntime.verify(ComposeRuntime[F](composition).search(), claim)
    munit.Assertions.assertEquals(result.verdict, NativeCheck.Verdict.refuted, result.toString)
    munit.Assertions.assert(result.detail.exists(_.startsWith("counterexample: ")), result.toString)
    result

  // The message a refused composition is refused with, where the runtime reads it.
  def refusal(read: => Any): String =
    try
      val _ = read
      "not refused"
    catch case e: IllegalArgumentException => e.getMessage
