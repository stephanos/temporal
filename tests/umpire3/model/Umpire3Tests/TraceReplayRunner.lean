import Umpire3.TraceReplayRunner

namespace Umpire3.Tests.TraceReplayRunner

def mutatedRequest : TraceReplay.Request where
  traceDigest := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
  target := "nexus-cancellation"
  property := "nexus.cancellation.won-excludes-success"
  world := "smoke"
  variant := "stale-completion-guard-removed"
  semanticHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
  actions := [
    "dispatch-task",
    "acquire-ownership",
    "worker-returns-success",
    "persist-success",
  ]

example : TraceReplay.checkRequest mutatedRequest = true := by
  simp [TraceReplay.checkRequest, TraceReplay.Request.matchesMutatedNexus,
    TraceReplay.Request.validDigest, TraceReplay.Request.validSemanticHash,
    TraceReplay.lowerHexDigit, mutatedRequest]
  decide

example : TraceReplay.checkRequest { mutatedRequest with variant := "sound" } = false := by
  simp [TraceReplay.checkRequest, TraceReplay.Request.matchesMutatedNexus, mutatedRequest]

example : TraceReplay.checkRequest { mutatedRequest with actions := ["unknown-action"] } = false := by
  simp [TraceReplay.checkRequest, TraceReplay.Request.matchesMutatedNexus,
    TraceReplay.Request.validDigest, TraceReplay.Request.validSemanticHash,
    TraceReplay.lowerHexDigit, mutatedRequest]
  decide

example : TraceReplay.checkRequest { mutatedRequest with
    traceDigest := "sha256:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz" } = false := by
  simp [TraceReplay.checkRequest, TraceReplay.Request.matchesMutatedNexus,
    TraceReplay.Request.validDigest, TraceReplay.lowerHexDigit, mutatedRequest]

example : TraceReplay.checkedRequestAxioms = ["Classical.choice", "Quot.sound", "propext"] := by
  simp [TraceReplay.checkedRequestAxioms]

end Umpire3.Tests.TraceReplayRunner
