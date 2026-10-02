module activity_tests {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"
  import activityProtocolTable.* from "./activity_protocol_table"
  import activityProtocol as activity from "./activity_protocol"

  run terminalControlsReportNotFoundTest = assert(
    controls.forall(c => protocolControlStep({ ...unstarted, phase: Completed }, c) ==
      [{ outcome: NotFound, state: { ...unstarted, phase: Completed }, facts: [] }] ))

  run retryRecordsVisibleScheduledStatusTest = assert(
    protocolAttemptResultStep({ ...unstarted, phase: Started, attempts: 1 }, AttemptFailed(true)) ==
      [{ outcome: Accepted, state: { ...unstarted, phase: BackingOff, attempts: 1 },
         facts: [StatusScheduled, AttemptCount] }])

  run lateControlsRespectSafetyTest =
    activity::init.then(activity::start(Unset, Unset, Unset))
      .then(activity::attemptStart).then(activity::attemptResult(AttemptCompleted))
      .then(activity::control(Terminate)).expect(activity::properties)
}
