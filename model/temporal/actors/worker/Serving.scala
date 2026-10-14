// The Serving facet of Polling (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package actors.worker

import framework.dsl.*
import framework.dsl.outcomes.Outcome

trait Serving extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object serving:

    object effects:
      // A polling worker serves and keeps polling.
      def serve(s: State) = stay(s)

    // A polling worker stops and serves; a stopped one resumes, and has nothing to stop or serve.
    object rules extends Rules:
      on(worker.stop) {
        when(Phase.polling) ~> enters(Phase.stopped)
      }
      on(worker.resume) {
        when(Phase.stopped) ~> enters(Phase.polling)
      }
      on(worker.serve) {
        when(Phase.polling) ~> effects.serve
      }
