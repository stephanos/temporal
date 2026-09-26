import Temporal.Tool.ExplorationBridge

/-! Effect-thin executable boundary for the exploration bridge: frames on stdin and stdout, progress
and diagnostics on stderr. -/

def main (arguments : List String) : IO UInt32 := do
  unless arguments.isEmpty do
    IO.eprintln s!"{Temporal.Tool.ExplorationBridge.diagnosticPrefix} takes no arguments; frames arrive on stdin"
    return 2
  Temporal.Tool.ExplorationBridge.serve (← Temporal.Tool.ExplorationBridge.processEffects)
    Temporal.Tool.ExplorationBridge.boundSets
