import Testpilot.Tests.ProtoJSON

def main : IO Unit := do
  match ← Testpilot.ProtoJSON.canonical Testpilot.Tests.ProtoJSON.representativeCase with
  | .ok text => IO.print text
  | .error error => throw (IO.userError (toString error))
