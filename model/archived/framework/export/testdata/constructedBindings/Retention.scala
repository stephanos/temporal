package framework.bindingretention

import fixture.bindingdiscovery.*

object Retention:
  def main(args: Array[String]): Unit =
    for machine <- Seq(Base, Restricted, Unmonitored, Replaced, Added, Chained) do
      for binding <- machine.bindings do
        assert(binding.captured.nonEmpty, s"${machine.name} discarded captured metadata")
        assert(binding.captured.get.callable.eq(binding.function))
    println("retention:passed")
