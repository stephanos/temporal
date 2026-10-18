package proto

import plain.*

// What run.sh prints: the native evidence of the probe. There is no IR here (the lifting moved to
// fn-190.9); the arms are the `on` macro's reading of each transition, first match as nested ifs.
@main def report(): Unit =
  import Kettle.*
  val native = Runtime(Model)
  println("== Kettle")
  println(s"State record: ${native.schema.mkString("State(", ", ", ")")}")
  println(s"initial state: ${native.state()}")
  val (starts, states, edges) = native.explore()
  println(s"starts: ${starts.size}, reachable states: ${states.size}, steps: ${edges.size}")
  println("transitions:")
  native.bindings.foreach(b => println(s"  on(${b.action}): ${b.arms.mkString("; ")}"))
  println("claims, checked by default from every start:")
  native.verifyAll().toList.sortBy(_._1).foreach((k, v) => println(s"  $k: $v"))
  val boils = Model.progress.boils
  println("progress:")
  println(s"  boils as written (strict environment): ${native.progress(boils)}")
  println(
    s"  boils, from states over the whole graph: ${native.progress(boils, fromWholeGraph = true)}"
  )
  val room = boils in (only(heater.heat, heater.failover) &
    scenario.starts(temperature = 20, heatingOn = true))
  println(
    s"  boils in only(heat, failover) & starts(temperature = 20, heatingOn = true): ${native.progress(room)}"
  )
  val any = boils in (only(heater.heat, heater.failover) & scenario.starts(heatingOn = true))
  println(s"  boils in only(heat, failover) & starts(heatingOn = true): ${native.progress(any)}")
  val began = System.nanoTime
  val rounds = 5
  (1 to rounds).foreach(_ => native.verifyAll())
  val evaluations = starts.size + 2 * edges.size
  val micros = (System.nanoTime - began) / 1000 / rounds / evaluations
  println(s"claim evaluation by class reload: ~$micros us each ($evaluations per verifyAll)")

  println("== Lamps (the Facets fixture)")
  import Lamps.*
  val whole = Runtime(Whole)
  println(s"State record: ${whole.schema.mkString("State(", ", ", ")")}")
  println(s"bounds: ${whole.schema.map(v => s"${v.name} -> ${v.bound}").mkString(", ")}")
  println("transitions:")
  whole.bindings.foreach(b => println(s"  on(${b.action}):\n    ${b.arms.mkString("\n    ")}"))
  println(
    s"Faceted facet order: ${Runtime(Faceted).bindings.map(b => s"${b.facet}:${b.action}").mkString(", ")}"
  )
  println(
    s"Reordered facet order: ${Runtime(Reordered).bindings.map(b => s"${b.facet}:${b.action}").mkString(", ")}"
  )
