import Lake

open Lake DSL

package umpire3_veil

require veil from git
  "https://github.com/verse-lab/veil.git" @ "300c305e945750ab3fb62de4a79c23161b24da39"

lean_lib Umpire3Veil

lean_exe umpire3_veil_sound where
  root := `Umpire3Veil.RunSound

lean_exe umpire3_veil_mutated where
  root := `Umpire3Veil.RunMutated

lean_exe umpire3_veil_sound_proof where
  root := `Umpire3Veil.RunSoundProof

lean_exe umpire3_veil_sound_trusted_proof where
  root := `Umpire3Veil.RunSoundTrustedProof
