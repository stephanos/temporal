import Umpire.Artifact.PortableEvaluationContract

namespace Umpire.Artifact.Tests.PortableEvaluationContract

def mutationChangesBytes
    (original mutated : Umpire.Artifact.PortableEvaluationContract.Contract) : Bool :=
  Umpire.Artifact.PortableEvaluationContract.canonicalProtoJSON original !=
    Umpire.Artifact.PortableEvaluationContract.canonicalProtoJSON mutated

end Umpire.Artifact.Tests.PortableEvaluationContract
