//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:imports
//> using jar ../../../build/model-scala.jar
//> using jar ../../../build/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
// fn-118.2: a hint relates methods of the generated API and kinds of cause, by value, so a hint of
// a method the API does not have, of a write or a read that is no method, with a `when` that is no
// `Visible` or a bound that is no `WaitBound`, built by its constructor rather than `visibleTo`, or
// a server step of no class, fails to compile at its source. Each form is written once as it
// compiles, then wrong; each wrong form is the one error on its line.
package fixture.hintsInvalid

import io.temporal.api.workflowservice.v1.{DescribeActivityExecutionRequest, WorkflowServiceGrpc}
import temporal.realize.*
import temporal.features.standaloneactivity.worker

private val start = WorkflowServiceGrpc.METHOD_START_ACTIVITY_EXECUTION
private val describe = WorkflowServiceGrpc.METHOD_DESCRIBE_ACTIVITY_EXECUTION

val seen = start.visibleTo(describe, Visible.atOnce)
val noSuchMethod = WorkflowServiceGrpc.METHOD_DESCRIBE_NOTHING.visibleTo(describe, Visible.atOnce)
val writeNoMethod = "StartActivityExecution".visibleTo(describe, Visible.atOnce)
val readNoMethod = start.visibleTo(DescribeActivityExecutionRequest, Visible.atOnce)

val answered = CauseKind.activityAnswer.visibleTo(describe, Visible.atOnce)
val whenNoVisible = CauseKind.activityAnswer.visibleTo(describe, WaitBound(100, 1000))
val constructed = Visibility(CauseKind.activityAnswer, describe, Visible.atOnce)

val bounded = CauseKind.delivery.boundedBy(WaitBound(100, 1000))
val boundNoWait = CauseKind.delivery.boundedBy(Visible.atOnce)

val step = ServerStep(worker.poll, CauseKind.delivery)
val stepNoClass = ServerStep(CauseKind.delivery, CauseKind.delivery)
