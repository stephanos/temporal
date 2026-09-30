module trace_tests {
  import umpire.* from "./umpire"
  import nexusProtocol as operation from "./nexus_caller"

  run repeatedFactsHaveDistinctOccurrencesTest =
    operation::init.expect(not(operation::occurrence))
      .then(operation::workerStop).expect(operation::occurrence)
      .then(operation::workerStop).expect(not(operation::occurrence))

  run disabledActionHasNoTraceTest =
    operation::init.then(operation::backoff).fail()
}
