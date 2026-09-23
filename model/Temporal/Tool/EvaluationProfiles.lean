import Temporal.Evaluation.Local

/-!
Writer for the rendered Evaluation Profiles Go embeds: every Profile this model declares, one
canonical JSON file per Profile, named by the Profile's exact name. A declaration that does not
check, or two Profiles with one name, writes nothing and fails.
-/

namespace Temporal.Tool.EvaluationProfiles

open Umpire.Evaluation

/-- Every declared Profile as its file name and canonical bytes, or the first reason there are
none. -/
def rendered : Except String (List (String × String)) := do
  let profiles ← Temporal.Evaluation.Local.declared.mapM fun declared =>
    declared.mapError ProfileError.render
  if let some name := firstDuplicate (profiles.map (·.name)) then
    throw s!"Profile '{name}' is declared twice"
  pure (profiles.map fun profile => (profile.name ++ ".json", profile.render))

/-- Write every rendered Profile under `outputDir`, which must exist. -/
def write (outputDir : System.FilePath) : IO Unit := do
  unless ← outputDir.isDir do
    throw (IO.userError s!"output directory {outputDir} does not exist")
  match rendered with
  | .error message => throw (IO.userError message)
  | .ok files =>
      for (name, contents) in files do
        IO.FS.writeFile (outputDir / name) contents

end Temporal.Tool.EvaluationProfiles

private def parseOutputDir : List String → IO System.FilePath
  | ["--output-dir", directory] => pure directory
  | arguments =>
      throw (IO.userError s!"umpire-evaluation-profiles accepts only --output-dir <dir>: {arguments}")

def main (arguments : List String) : IO UInt32 := do
  try
    Temporal.Tool.EvaluationProfiles.write (← parseOutputDir arguments)
    pure 0
  catch failure =>
    IO.eprintln s!"umpire-evaluation-profiles: {failure}"
    pure 1
