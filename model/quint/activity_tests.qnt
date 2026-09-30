module activity_tests {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./standalone_activity"
  import activityProtocolTable.* from "./standalone_activity"

  run terminalControlsReportNotFoundTest = assert(
    controls.forall(c => protocolControlStep({ ...unstarted, phase: Completed }, c) ==
      [{ outcome: NotFound, state: { ...unstarted, phase: Completed }, facts: [] }] ))

  run retryRecordsVisibleScheduledStatusTest = assert(
    protocolAttemptResultStep({ ...unstarted, phase: Started, attempts: 1 }, AttemptFailed(true)) ==
      [{ outcome: Accepted, state: { ...unstarted, phase: BackingOff, attempts: 1 },
         facts: [StatusScheduled, AttemptCount] }])
}
