// An action that names the message it carries, which fn-133.8 retired: what a class carries is
// derived from the realization's binding of it.
package fixture.retiredschema

import umpire.*
import io.temporal.api.workflowservice.v1.StartActivityExecutionRequest

val start =
  action(Actor("caller")).schema[StartActivityExecutionRequest]
