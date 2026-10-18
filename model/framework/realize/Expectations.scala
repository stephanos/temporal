package framework.realize

def satisfied: RunExpectation =
  val value = RunExpectation(
    Conformance.conformant,
    PropertyOutcome.satisfied,
    PropertyOutcome.satisfied,
    Disposition.completed,
    Cleanup.succeeded
  )
  validateConformanceExpectation(value)
  value

def conformant: RunExpectation =
  inconclusive(Reason.neverEvaluated, because = "A sampled path selects no individual Property")

def inconclusive(reason: Reason, because: String): RunExpectation =
  require(
    reason.equals(Reason.explanationsDisagree) || reason.equals(Reason.neverEvaluated) ||
      reason.equals(Reason.unreadable),
    "Property-only inconclusive requires explanationsDisagree, neverEvaluated or unreadable; " +
      "use a full RunExpectation for other reasons"
  )
  val value = RunExpectation(
    Conformance.conformant,
    PropertyOutcome.inconclusive,
    PropertyOutcome.satisfied,
    Disposition.completed,
    Cleanup.succeeded,
    reason = Some(reason),
    because = because
  )
  validateConformanceExpectation(value)
  value

private[framework] def validateConformanceExpectation(value: RunExpectation): Unit =
  val bothSatisfied = value.property.equals(PropertyOutcome.satisfied) &&
    value.contract.equals(PropertyOutcome.satisfied)
  require(
    bothSatisfied == value.reason.isEmpty,
    "Property/Contract reason must be absent exactly when both assessments are satisfied"
  )
  require(
    if bothSatisfied then value.because.isEmpty else value.because.trim.nonEmpty,
    "Property/Contract because must be empty when both assessments are satisfied and nonblank otherwise"
  )
  require(
    value.conformance.equals(Conformance.conformant) == value.conformanceReason.isEmpty,
    "conformanceReason must be absent exactly when conformance is conformant"
  )
  value.monitors.zipWithIndex.foreach: (monitor, index) =>
    val isSatisfied = monitor.outcome.equals(PropertyOutcome.satisfied)
    require(
      isSatisfied == monitor.reason.isEmpty,
      s"monitor[$index] reason must be absent exactly when its outcome is satisfied"
    )
    require(
      if isSatisfied then monitor.because.isEmpty else monitor.because.trim.nonEmpty,
      s"monitor[$index] because must be empty for satisfied and nonblank otherwise"
    )
