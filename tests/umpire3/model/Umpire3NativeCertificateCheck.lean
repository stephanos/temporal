import Umpire3.NativeCertificateRunner

def main (arguments : List String) : IO UInt32 := do
  match arguments with
  | certificateDigest :: viewDigest :: target :: property :: world :: variant :: semanticHash ::
      replicasText :: expandedStatesText :: representativeStatesText :: nodeCountText :: values =>
      let some replicas := replicasText.toNat?
        | IO.eprintln "native certificate replica count must be a natural number"; return 2
      let some expandedStates := expandedStatesText.toNat?
        | IO.eprintln "native certificate expanded state count must be a natural number"; return 2
      let some representativeStates := representativeStatesText.toNat?
        | IO.eprintln "native certificate representative count must be a natural number"; return 2
      let some nodeCount := nodeCountText.toNat?
        | IO.eprintln "native certificate node count must be a natural number"; return 2
      let some nodes := Umpire3.NativeCertificate.parseWireNodes nodeCount values
        | IO.eprintln "native certificate node encoding is invalid"; return 2
      let request : Umpire3.NativeCertificate.Request := {
        certificateDigest
        viewDigest
        target
        property
        world
        variant
        semanticHash
        replicas
        expandedStates
        representativeStates
        nodes
      }
      if accepted : Umpire3.NativeCertificate.checkRequest request then
        let _ := Umpire3.NativeCertificate.checkedRequest request accepted
        IO.println request.receipt.compress
        return 0
      else
        IO.eprintln "canonical Lean native certificate checker rejected the request"
        return 1
  | _ =>
      IO.eprintln "native certificate digest, identity, scale, and compact nodes are required"
      return 2
