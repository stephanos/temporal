/* The checked-in IR file of the standalone Nexus operation Model (umpire.irFile). */
package temporal
package nexusoperation

import umpire.*

val nexusOperationFile =
  irFile("nexus-operation")(nexusOperation, operationCapabilities, OperationRealization.standalone)
