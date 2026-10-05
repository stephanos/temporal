//> using scala 3.9.0
//> using jar ../../../build/model-scala.jar
//> using jar ../../../build/api-scalapb.jar
//> using dep com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20
package fixture.dynamicInvalid

import temporal.server.api.testpilot.v1.InstructionOutcome
import umpire.realize.*

val wrongRun = Operand.Run.as[InstructionOutcome]
