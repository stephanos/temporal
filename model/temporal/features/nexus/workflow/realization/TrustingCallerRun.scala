package temporal
package features.nexus
package workflow
package realization

import framework.realize.*
import temporal.realize.*

import protocol.TrustingCaller
import CallerDeclarations.*

// The forged control's realization: WorkflowNexusRun, with the workflow inspected after the scheduled
// event.
object TrustingCallerRun
    extends DerivesFrom(WorkflowNexusRun, TrustingCaller, runs = Seq(TrustingCaller.queries)):

  object changes
      extends Changes(inserting(after = awaitScheduled)(perform(caller.inspect -> inspectWorkflow)))
