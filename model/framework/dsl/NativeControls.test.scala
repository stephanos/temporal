package framework.dsl

// Negative controls as tests (fn-190.4): a claim a deliberately wrong design must break is asserted
// refuted here, by the native checker (NativeCheck), rather than declared as a Query of the Model.
// A suite imports them, `import NativeControls.assertRefuted`; each returns the result, whose
// counterexample a test may pin further.
object NativeControls:
  // Asserts that an invariant or postcondition is refuted on `machine` from `starts` (its initial
  // state unless named): `assertRefuted(Bank.Leaky, Bank.Leaky.invariants.leakyTotalSupplyInv)`.
  def assertRefuted[S, O, F](
      machine: RecordMachine[S, O, F],
      claim: Property,
      starts: Seq[S] = Seq.empty,
      budget: Int = NativeCheck.defaultBudget
  )(using munit.Location): NativeCheck.Result =
    val origins = if starts.isEmpty then Seq(machine.init) else starts
    val graph = NativeCheck.explore(machine, origins, budget = budget)
    val subject = s"${machine.name}.${NativeCheck.claimName(claim, machine)}"
    refuted(NativeCheck.verify(graph, subject, claim.decl))

  // Asserts that a monitor is broken on `machine`, such as one the machine keeps from the source it
  // is derived from: `assertRefuted(Naive, Model.monitors.agreement)`.
  def assertRefuted[S, O, F, M](
      machine: RecordMachine[S, O, F],
      watch: Monitor[S, O, F, M]
  )(using munit.Location): NativeCheck.Result =
    val subject = s"${machine.name}.${NativeCheck.claimName(watch, machine)}"
    refuted(NativeCheck.monitor(machine, subject, watch, Seq(machine.init)))

  private def refuted(result: NativeCheck.Result)(using munit.Location): NativeCheck.Result =
    munit.Assertions.assertEquals(result.verdict, NativeCheck.Verdict.refuted, result.toString)
    munit.Assertions.assert(result.detail.exists(_.startsWith("counterexample: ")), result.toString)
    result
