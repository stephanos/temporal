import Temporal.Tool.RunEvaluation

private partial def readBounded
    (stream : IO.FS.Stream)
    (limit : Nat) : IO ByteArray := do
  let rec loop (input : ByteArray) : IO ByteArray := do
    if input.size >= limit then
      return input
    let chunk ← stream.read (limit - input.size).toUSize
    if chunk.isEmpty then
      return input
    loop (input ++ chunk)
  loop .empty

def main (_args : List String) : IO UInt32 := do
  let input ← readBounded (← IO.getStdin)
    (Temporal.Tool.RunEvaluation.Protocol.maxBytes + 1)
  let result := Temporal.Tool.RunEvaluation.runBytes input
  (← IO.getStdout).write result.stdout
  IO.eprint result.stderr
  pure result.status
