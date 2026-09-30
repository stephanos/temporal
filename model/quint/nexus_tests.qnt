module nexus_tests {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProtocol as operation from "./nexus_protocol"

  run lateCompletionRespectsSafetyTest =
    operation::init.then(operation::schedule(Unset, Unset, Unset))
      .then(operation::handlerReply(SyncSuccess))
      .then(operation::complete(ResolvedSucceeded)).expect(operation::properties)
}
