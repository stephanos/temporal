/* The checked-in IR file of the standalone Nexus operation Model (umpire.irFile). */
package temporal
package features.nexusoperation

import umpire.*

val nexusOperationFile =
  irFile("nexus-operation")(nexusOperation, operationCapabilities, OperationRealization.standalone)
