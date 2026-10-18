//> using scala 3.9.0
//> using jvm 27
//> using options -Werror -deprecation -feature -unchecked -Wunused:all -Wvalue-discard -Wnonunit-statement -Wsafe-init -Wimplausible-patterns -Yretain-trees -language:strictEquality
package framework.prototype

import framework.check.{Area, Baseline, Covers, Module, Registry}
import java.nio.file.Path

object Inventory:
  def main(arguments: Array[String]): Unit =
    require(arguments.isEmpty)
    val registry = new Registry:
      def source: String = "prototype discovery experiment"
      def current: Set[String] = Set.empty
      def baselines: Map[String, Baseline] = Map.empty
      def committedIrHeld: Boolean = true
      def rules: Seq[(String, Area)] = Seq.empty
      def selectsEveryFeature: Set[Area] = Set.empty
      def suites: Map[String, Covers] = Map.empty
      def modules: Seq[Module] = Seq(Module("prototype", Seq("model/design/modelsuite-prototype"), Seq.empty))
    val discovered = registry.declared(Path.of(".")).map(_.name)
    assert(discovered.sorted == Vector("framework.prototype.LifecycleTest", "framework.prototype.LifecycleOptionsTest").sorted, discovered)
    println(s"Registry inventory: ${discovered.mkString(", ")}")
    println("PRODUCTION BLOCKER: Registry does not recognize BankTest, QuintBankTest or KettleTest extending ModelSuite; run.sh selects them explicitly")
