package framework.realize

class ExpectationsTest extends munit.FunSuite:
  private def fields(value: RunExpectation) = (
    value.conformance,
    value.property,
    value.contract,
    value.disposition,
    value.cleanup,
    value.reason,
    value.conformanceReason,
    value.monitors,
    value.because
  )

  private def refuses(value: RunExpectation, field: String): Unit =
    val error = intercept[IllegalArgumentException](validateConformanceExpectation(value))
    assert(error.getMessage.contains(field), error.getMessage)

  test("path and sampled defaults retain every independent expectation field") {
    assertEquals(
      fields(satisfied),
      (
        Conformance.conformant,
        PropertyOutcome.satisfied,
        PropertyOutcome.satisfied,
        Disposition.completed,
        Cleanup.succeeded,
        None,
        None,
        Vector.empty,
        ""
      )
    )
    assertEquals(
      fields(conformant),
      (
        Conformance.conformant,
        PropertyOutcome.inconclusive,
        PropertyOutcome.satisfied,
        Disposition.completed,
        Cleanup.succeeded,
        Some(Reason.neverEvaluated),
        None,
        Vector.empty,
        "A sampled path selects no individual Property"
      )
    )
    validateConformanceExpectation(satisfied)
    validateConformanceExpectation(conformant)
  }

  test("Property-only shorthand admits exactly its three reasons and preserves authored text") {
    val allowed = Vector(Reason.explanationsDisagree, Reason.neverEvaluated, Reason.unreadable)
    allowed.foreach: reason =>
      val value = inconclusive(reason, because = "  authored explanation  ")
      assertEquals(
        fields(value),
        (
          Conformance.conformant,
          PropertyOutcome.inconclusive,
          PropertyOutcome.satisfied,
          Disposition.completed,
          Cleanup.succeeded,
          Some(reason),
          None,
          Vector.empty,
          "  authored explanation  "
        )
      )
      validateConformanceExpectation(value)
      Vector("", " \t\n ").foreach: because =>
        val error = intercept[IllegalArgumentException](inconclusive(reason, because))
        assert(error.getMessage.contains("because"), error.getMessage)
    Reason.values
      .filterNot(allowed.contains)
      .foreach: reason =>
        val error = intercept[IllegalArgumentException](inconclusive(reason, "explicit assessment"))
        assert(error.getMessage.contains("full RunExpectation"), error.getMessage)
        validateConformanceExpectation(
          RunExpectation(
            Conformance.inconclusive,
            PropertyOutcome.inconclusive,
            PropertyOutcome.violated,
            Disposition.incomplete,
            Cleanup.timedOut,
            Some(reason),
            Some(Reason.noEvidence),
            because = "explicit assessment"
          )
        )
  }

  test("all nine Property and Contract pairs validate their shared reason and explanation") {
    type Builder = (Option[Reason], String) => RunExpectation
    val pairs: Vector[(String, Boolean, Builder)] = Vector(
      (
        "satisfied/satisfied",
        true,
        (reason, _) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.satisfied,
            PropertyOutcome.satisfied,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason
          )
      ),
      (
        "satisfied/violated",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.satisfied,
            PropertyOutcome.violated,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "satisfied/inconclusive",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.satisfied,
            PropertyOutcome.inconclusive,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "violated/satisfied",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.violated,
            PropertyOutcome.satisfied,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "violated/violated",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.violated,
            PropertyOutcome.violated,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "violated/inconclusive",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.violated,
            PropertyOutcome.inconclusive,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "inconclusive/satisfied",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.inconclusive,
            PropertyOutcome.satisfied,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "inconclusive/violated",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.inconclusive,
            PropertyOutcome.violated,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      ),
      (
        "inconclusive/inconclusive",
        false,
        (reason, text) =>
          RunExpectation(
            Conformance.conformant,
            PropertyOutcome.inconclusive,
            PropertyOutcome.inconclusive,
            Disposition.completed,
            Cleanup.succeeded,
            reason = reason,
            because = text
          )
      )
    )
    pairs.foreach: (name, bothSatisfied, build) =>
      val reason = if bothSatisfied then None else Some(Reason.incomplete)
      val text = if bothSatisfied then "" else "  independently assessed  "
      val value = build(reason, text)
      validateConformanceExpectation(value)
      assertEquals(value.because, text, name)
      refuses(build(if bothSatisfied then Some(Reason.incomplete) else None, text), "reason")
      if !bothSatisfied then
        Vector("", " \t\n ").foreach: invalid =>
          val raw = build(reason, invalid)
          assertEquals(raw.because, invalid, name)
          refuses(raw, "because")
        val copied = value.copy(because = "")
        assertEquals(copied.because, "", name)
        refuses(copied, "because")
    refuses(
      RunExpectation(
        Conformance.conformant,
        PropertyOutcome.inconclusive,
        PropertyOutcome.satisfied,
        Disposition.completed,
        Cleanup.succeeded,
        reason = Some(Reason.neverEvaluated)
      ),
      "because"
    )
  }

  test("conformance reason follows conformance independently of both satisfied assessments") {
    Conformance.values.foreach: conformance =>
      Vector(None, Some(Reason.noEvidence)).foreach: reason =>
        val value = RunExpectation(
          conformance,
          PropertyOutcome.satisfied,
          PropertyOutcome.satisfied,
          Disposition.completed,
          Cleanup.succeeded,
          conformanceReason = reason
        )
        if conformance.equals(Conformance.conformant) == reason.isEmpty then
          validateConformanceExpectation(value)
        else refuses(value, "conformanceReason")
  }

  test("each monitor validates its own outcome, reason and explanation") {
    type Builder = (Option[Reason], String) => MonitorExpectation
    val monitors: Vector[(Boolean, Builder)] = Vector(
      (true, (reason, _) => MonitorExpectation("satisfied", PropertyOutcome.satisfied, reason)),
      (
        false,
        (reason, text) =>
          MonitorExpectation(
            "violated",
            PropertyOutcome.violated,
            reason,
            text
          )
      ),
      (
        false,
        (reason, text) =>
          MonitorExpectation(
            "inconclusive",
            PropertyOutcome.inconclusive,
            reason,
            text
          )
      )
    )
    def run(monitor: MonitorExpectation): RunExpectation = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded,
      monitors = Vector(monitor)
    )
    monitors.foreach: (isSatisfied, build) =>
      val reason = if isSatisfied then None else Some(Reason.unreadable)
      val text = if isSatisfied then "" else "  monitor explanation  "
      val value = build(reason, text)
      validateConformanceExpectation(run(value))
      assertEquals(value.because, text)
      refuses(run(build(if isSatisfied then Some(Reason.unreadable) else None, text)), "monitor")
      if !isSatisfied then
        Vector("", " \t\n ").foreach: invalid =>
          refuses(run(build(reason, invalid)), "monitor")
        val copied = value.copy(because = "")
        assertEquals(copied.because, "")
        refuses(run(copied), "monitor")
    refuses(
      run(MonitorExpectation("omitted", PropertyOutcome.inconclusive, Some(Reason.unreadable))),
      "monitor"
    )
  }

  test("validation preserves mixed fields and monitor references without binding identities") {
    val declaration = framework.dsl.monitor[Boolean, Boolean, Boolean, Boolean]("watch", false)(
      (state, _, _) => state
    )(_ => false)
    val reference = MonitorExpectation(declaration, PropertyOutcome.satisfied)
    val named = MonitorExpectation(
      "watch",
      PropertyOutcome.violated,
      Some(Reason.everyExplanationViolates),
      "monitor failed"
    )
    val monitors = Vector(reference, named, reference)
    Disposition.values.foreach: disposition =>
      Cleanup.values.foreach: cleanup =>
        val value = RunExpectation(
          Conformance.nonconformant,
          PropertyOutcome.inconclusive,
          PropertyOutcome.violated,
          disposition,
          cleanup,
          Some(Reason.incomplete),
          Some(Reason.unexplained),
          monitors,
          "independent assessments"
        )
        val before = fields(value)
        validateConformanceExpectation(value)
        assertEquals(fields(value), before)
        assert(value.monitors eq monitors)
        assert(value.monitors.head eq reference)
        assert(value.monitors(1) eq named)
        assert(value.monitors.last eq reference)
        assert(value.monitors.head.name.asInstanceOf[AnyRef] eq declaration)
        assertEquals(value.monitors(1).name, "watch")
  }
