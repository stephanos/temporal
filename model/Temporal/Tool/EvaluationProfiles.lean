import Temporal.Evaluation.Local
import Temporal.Evaluation.Canary

/-!
Writer for the rendered Evaluation Profiles Go embeds, one canonical JSON file per Profile named
by the Profile's exact name, in three groups, each into the directory its flag names: the local
Profiles `umpire-assess` embeds, the production canary's Profile, and the canary harness's
Profile, which only the harness build embeds. No Profile carries a path. A declaration that does
not check, or two Profiles with one name, writes nothing and fails.
-/

namespace Temporal.Tool.EvaluationProfiles

open Umpire.Evaluation

/-- The groups of declared Profiles, each named by the flag that places it. -/
def groups : List (String × List (Except ProfileError Profile)) := [
  ("--local-dir", Temporal.Evaluation.Local.declared),
  ("--canary-dir", [Temporal.Evaluation.Canary.productionCanary]),
  ("--harness-dir", [Temporal.Evaluation.Canary.canaryHarness])
]

/-- Every declared Profile as its group's flag, file name and canonical bytes, or the first reason
there are none. -/
def rendered : Except String (List (String × String × String)) := do
  let profiles ← groups.flatMap (fun (flag, declared) => declared.map (flag, ·)) |>.mapM
    fun (flag, declared) => do pure (flag, ← declared.mapError ProfileError.render)
  if let some name := firstDuplicate (profiles.map (·.2.name)) then
    throw s!"Profile '{name}' is declared twice"
  pure (profiles.map fun (flag, profile) => (flag, profile.name ++ ".json", profile.render))

/-- Write every rendered Profile into its group's directory, each of which must exist. -/
def write (directories : List (String × System.FilePath)) : IO Unit := do
  for (flag, _) in groups do
    let some directory := directories.lookup flag
      | throw (IO.userError s!"{flag} <dir> is required")
    unless ← directory.isDir do
      throw (IO.userError s!"{flag} {directory} does not exist")
  match rendered with
  | .error message => throw (IO.userError message)
  | .ok files =>
      for (flag, name, contents) in files do
        if let some directory := directories.lookup flag then
          IO.FS.writeFile (directory / name) contents

end Temporal.Tool.EvaluationProfiles

private def parseDirectories : List String → IO (List (String × System.FilePath))
  | [] => pure []
  | flag :: directory :: rest => do
      unless Temporal.Tool.EvaluationProfiles.groups.any (·.1 == flag) do
        throw (IO.userError s!"unknown flag {flag}")
      pure ((flag, (directory : System.FilePath)) :: (← parseDirectories rest))
  | arguments =>
      throw (IO.userError s!"umpire-evaluation-profiles accepts --local-dir, --canary-dir and --harness-dir <dir>: {arguments}")

def main (arguments : List String) : IO UInt32 := do
  try
    Temporal.Tool.EvaluationProfiles.write (← parseDirectories arguments)
    pure 0
  catch failure =>
    IO.eprintln s!"umpire-evaluation-profiles: {failure}"
    pure 1
