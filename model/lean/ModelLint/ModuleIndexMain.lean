import ModelLint.ModuleIndexExporter

/-! Executable boundary for the on-demand module impact index; the decisions live in
`ModelLint.ModuleIndexExporter`. -/

unsafe def main (args : List String) : IO UInt32 :=
  ModelLint.ModuleIndexExporter.main args
