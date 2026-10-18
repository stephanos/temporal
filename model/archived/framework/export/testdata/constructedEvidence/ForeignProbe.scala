package framework.evidenceforeignprobe

import framework.dsl.*
import fixture.constructedevidence.{Fact, Foreign}

object Probe:
  val function = Reify.function((fact: Fact) => Foreign.text(fact))
