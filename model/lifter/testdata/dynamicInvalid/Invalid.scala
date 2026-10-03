//> using scala 3.9.0
//> using jar ../../../gen/model-scala.jar
//> using jar ../../../gen/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
package fixture.dynamicInvalid

import temporal.server.api.testpilot.v1.InstructionOutcome
import umpire.realize.*

val wrongRun = Operand.Run.as[InstructionOutcome]
