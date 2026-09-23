import Temporal.Tool.ReplayBridge

/-! Effect-thin executable boundary for the replay bridge: frames on stdin and stdout, progress and
diagnostics on stderr. -/

def main (arguments : List String) : IO UInt32 := do
  unless arguments.isEmpty do
    IO.eprintln s!"{Temporal.Tool.ReplayBridge.diagnosticPrefix} takes no arguments; frames arrive on stdin"
    return 2
  Temporal.Tool.ReplayBridge.serve (← Temporal.Tool.Bridge.processEffects)
    Temporal.Tool.ReplayBridge.boundSets
