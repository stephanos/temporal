module nexusCompositionTests {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProtocolTable as table from "./nexus_protocol_table"
  import nexusCaller as caller from "./nexus_caller"

  run workerGatesRepliesTest =
    caller::init
      .then(caller::fire(table::Schedule({ scheduleToClose: Unset,
        scheduleToStart: Expires, startToClose: Unset })))
      .then(caller::workerStop)
      .then(caller::handlerReply(SyncSuccess)).fail()

  run stoppedWorkerRepliesNothingTest = caller::stoppedWorkerRepliesNothing
}

module activityCompositionTests {
  import umpire.* from "./umpire"
  import activityProtocolTable as table from "./activity_protocol_table"
  import standaloneActivity as activity from "./standalone_activity"

  run workerGatesDispatchTest =
    activity::init
      .then(activity::fire(table::Start({ scheduleToClose: Unset,
        scheduleToStart: Expires, startToClose: Unset })))
      .then(activity::workerStop)
      .then(activity::attemptStart).fail()

  run stoppedWorkerStartsNothingTest = activity::stoppedWorkerStartsNothing
}
