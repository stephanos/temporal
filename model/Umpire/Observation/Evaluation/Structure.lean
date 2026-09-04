import Umpire.Observation.Evaluation.Types

/-!
Internal normalized structural analysis for Observation Evidence facts, closures, and per-link
support.
-/

namespace Umpire

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def closureLe (left right : EvidenceClosureFact) : Bool :=
  match left.source, right.source with
  | some leftSource, some rightSource =>
      decide (leftSource.value < rightSource.value) ||
        (leftSource == rightSource && idLe left.kind right.kind)
  | none, none => idLe left.kind right.kind
  | none, some _ => true
  | some _, none => false

namespace Observation.Internal

inductive StructuralOriginMode where
  | globalSequence
  | sourceSequence
  | mixed
  deriving BEq, DecidableEq, Repr

inductive StructuralFinding where
  | duplicateIdentity (recordId : DefinitionId) (conflicting : Bool)
  | mixedOrigins (recordIds : List DefinitionId)
  | duplicateSequence (firstId secondId : DefinitionId) (sequence : Nat)
  | sequenceGap
      (recordId : DefinitionId)
      (source : Option DefinitionId)
      (expected actual : Nat)
  | missingCausalParent (recordId : DefinitionId) (parentId : Option DefinitionId)
  | contradictoryOrder (recordId parentId : DefinitionId)
  | duplicateClosure
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (conflicting : Bool)
  | closureWithoutFacts (source : Option DefinitionId) (kind : DefinitionId)
  | missingClosure
      (recordIds : List DefinitionId)
      (source : Option DefinitionId)
      (kind : DefinitionId)
  | closureSequenceMismatch
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (expected actual : Nat)
  | closureCountMismatch
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (expected : Nat)
      (actual : Option Nat)
  | closureByteCountMissing (source : Option DefinitionId) (kind : DefinitionId)
  | missingRequiredKind (kind : DefinitionId)
  | inconsistentOrderingSupport
      (ruleId : DefinitionId)
      (expected actual : List EvidenceOrderingFact)
  | inconsistentClosureSupport
      (ruleId : DefinitionId)
      (expected actual : List EvidenceClosureFact)
  | duplicateClosureSupport
      (ruleId : DefinitionId)
      (linkIndex : Nat)
      (source : Option DefinitionId)
      (kind : DefinitionId)
      (conflicting : Bool)
  deriving BEq, DecidableEq, Repr

structure ClosureExpectation where
  source : Option DefinitionId
  kind : DefinitionId
  recordIds : List DefinitionId
  lastSequence : Nat
  recordCount : Nat
  deriving BEq, DecidableEq, Repr

structure StructuralLinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

structure NormalizedStructuralLinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  facts : List EvidenceOrderingFact
  closures : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

structure StructuralAnalysis where
  facts : List EvidenceOrderingFact
  closures : List EvidenceClosureFact
  originMode : StructuralOriginMode
  closureExpectations : List ClosureExpectation
  links : List NormalizedStructuralLinkSupport
  findings : List StructuralFinding
  deriving BEq, DecidableEq, Repr

private def factByRecordLe (left right : EvidenceOrderingFact) : Bool :=
  idLe left.recordId right.recordId

private def factBySequenceLe (left right : EvidenceOrderingFact) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin =>
      decide (leftOrigin.source.value < rightOrigin.source.value) ||
        (leftOrigin.source == rightOrigin.source &&
          (leftOrigin.ordinal < rightOrigin.ordinal ||
            (leftOrigin.ordinal == rightOrigin.ordinal && idLe left.recordId right.recordId)))
  | none, none => left.sequence < right.sequence ||
      (left.sequence == right.sequence && idLe left.recordId right.recordId)
  | none, some _ => true
  | some _, none => false

private def canonicalFacts :
    List EvidenceOrderingFact → List EvidenceOrderingFact × List StructuralFinding
  | [] => ([], [])
  | [fact] => ([fact], [])
  | first :: second :: rest =>
      if first.recordId == second.recordId then
        let (facts, findings) := canonicalFacts (first :: rest)
        (facts, .duplicateIdentity first.recordId (first != second) :: findings)
      else
        let (facts, findings) := canonicalFacts (second :: rest)
        (first :: facts, findings)

private def canonicalClosures :
    List EvidenceClosureFact → List EvidenceClosureFact × List StructuralFinding
  | [] => ([], [])
  | [closure] => ([closure], [])
  | first :: second :: rest =>
      if first.source == second.source && first.kind == second.kind then
        let (closures, findings) := canonicalClosures (first :: rest)
        (closures, .duplicateClosure first.source first.kind (first != second) :: findings)
      else
        let (closures, findings) := canonicalClosures (second :: rest)
        (first :: closures, findings)

private partial def factDependsOn
    (facts : List EvidenceOrderingFact)
    (recordId target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if recordId == target then true
  else if visited.contains recordId then false
  else
    match facts.find? fun fact => fact.recordId == recordId with
    | none => false
    | some fact => fact.causalParents.any fun parent =>
        factDependsOn facts parent target (recordId :: visited)

private def globalSequenceFindings
    (facts : List EvidenceOrderingFact) : List StructuralFinding := Id.run do
  let mut findings := []
  let mut expectedSequence := 1
  let mut previous : Option EvidenceOrderingFact := none
  for fact in facts do
    match previous with
    | some prior =>
        if fact.sequence == prior.sequence then
          findings := findings ++ [.duplicateSequence prior.recordId fact.recordId fact.sequence]
    | none => pure ()
    if fact.sequence != expectedSequence then
      findings := findings ++ [.sequenceGap fact.recordId none expectedSequence fact.sequence]
    if previous.isSome && fact.causalParents.isEmpty then
      findings := findings ++ [.missingCausalParent fact.recordId none]
    for parent in fact.causalParents do
      match facts.find? fun candidate => candidate.recordId == parent with
      | none => findings := findings ++ [.missingCausalParent fact.recordId (some parent)]
      | some parentFact =>
          if factDependsOn facts parent fact.recordId || parentFact.sequence >= fact.sequence then
            findings := findings ++ [.contradictoryOrder fact.recordId parent]
    previous := some fact
    expectedSequence := expectedSequence + 1
  pure findings

private def sourceSequenceFindings
    (facts : List EvidenceOrderingFact) : List StructuralFinding := Id.run do
  let mut findings := []
  let sources := DefinitionId.canonicalSet <|
    facts.filterMap fun fact => fact.origin.map EvidenceOrigin.source
  for source in sources do
    let sourceFacts := facts.filter fun fact =>
      fact.origin.any fun origin => origin.source == source
    let mut expectedOrdinal := 0
    for fact in sourceFacts do
      match fact.origin with
      | some origin =>
          if origin.ordinal != expectedOrdinal then
            findings := findings ++ [
              .sequenceGap fact.recordId (some source) expectedOrdinal origin.ordinal]
      | none => pure ()
      expectedOrdinal := expectedOrdinal + 1
  for fact in facts do
    for parent in fact.causalParents do
      match facts.find? fun candidate => candidate.recordId == parent with
      | none => findings := findings ++ [.missingCausalParent fact.recordId (some parent)]
      | some parentFact =>
          let reversesSourceOrder := match fact.origin, parentFact.origin with
            | some factOrigin, some parentOrigin =>
                factOrigin.source == parentOrigin.source &&
                  parentOrigin.ordinal >= factOrigin.ordinal
            | _, _ => false
          if factDependsOn facts parent fact.recordId || reversesSourceOrder then
            findings := findings ++ [.contradictoryOrder fact.recordId parent]
  pure findings

private structure ClosureKey where
  source : Option DefinitionId
  kind : DefinitionId
  deriving BEq, DecidableEq, Repr

private def closureKeyLe (left right : ClosureKey) : Bool :=
  match left.source, right.source with
  | some leftSource, some rightSource =>
      decide (leftSource.value < rightSource.value) ||
        (leftSource == rightSource && idLe left.kind right.kind)
  | none, none => idLe left.kind right.kind
  | none, some _ => true
  | some _, none => false

private def closureExpectations
    (originMode : StructuralOriginMode)
    (facts : List EvidenceOrderingFact) : List ClosureExpectation :=
  let keys := facts.map fun fact => {
    source := if originMode == .sourceSequence then fact.origin.map EvidenceOrigin.source else none
    kind := fact.kind
  }
  let keys := keys.mergeSort closureKeyLe |>.eraseDups
  keys.map fun key =>
    let matchingFacts := facts.filter fun fact =>
      fact.kind == key.kind &&
        (key.source.isNone || fact.origin.map EvidenceOrigin.source == key.source)
    let lastSequence := matchingFacts.foldl (fun current fact =>
      let sequence := match key.source, fact.origin with
        | some _, some origin => origin.ordinal + 1
        | _, _ => fact.sequence
      Nat.max current sequence) 0
    {
      source := key.source
      kind := key.kind
      recordIds := matchingFacts.map EvidenceOrderingFact.recordId
      lastSequence
      recordCount := matchingFacts.length
    }

private def globalClosureFindings
    (requiredKinds : List DefinitionId)
    (closures : List EvidenceClosureFact)
    (expectations : List ClosureExpectation) : List StructuralFinding := Id.run do
  let mut findings := []
  for closure in closures do
    match expectations.find? fun expectation => expectation.kind == closure.kind with
    | none => findings := findings ++ [.closureWithoutFacts closure.source closure.kind]
    | some expectation =>
        if closure.lastSequence != expectation.lastSequence then
          findings := findings ++ [
            .closureSequenceMismatch none closure.kind
              expectation.lastSequence closure.lastSequence]
  for expectation in expectations do
    if !(closures.any fun closure => closure.kind == expectation.kind) then
      findings := findings ++ [
        .missingClosure expectation.recordIds none expectation.kind]
  for kind in requiredKinds do
    if !(expectations.any fun expectation => expectation.kind == kind) then
      findings := findings ++ [.missingRequiredKind kind]
  pure findings

private def sourceClosureFindings
    (requiredKinds : List DefinitionId)
    (closures : List EvidenceClosureFact)
    (expectations : List ClosureExpectation) : List StructuralFinding := Id.run do
  let mut findings := []
  for closure in closures do
    match expectations.find? fun expectation =>
        expectation.source == closure.source && expectation.kind == closure.kind with
    | none => findings := findings ++ [.closureWithoutFacts closure.source closure.kind]
    | some expectation =>
        if closure.lastSequence != expectation.lastSequence then
          findings := findings ++ [
            .closureSequenceMismatch closure.source closure.kind
              expectation.lastSequence closure.lastSequence]
        if closure.recordCount != some expectation.recordCount then
          findings := findings ++ [
            .closureCountMismatch closure.source closure.kind
              expectation.recordCount closure.recordCount]
        if closure.byteCount.isNone then
          findings := findings ++ [.closureByteCountMissing closure.source closure.kind]
  for expectation in expectations do
    if !(closures.any fun closure =>
        closure.source == expectation.source && closure.kind == expectation.kind) then
      findings := findings ++ [
        .missingClosure expectation.recordIds expectation.source expectation.kind]
  for kind in requiredKinds do
    if !(expectations.any fun expectation => expectation.kind == kind) then
      findings := findings ++ [.missingRequiredKind kind]
  pure findings

private def normalizeLinkSupport
    (linkIndex : Nat)
    (originMode : StructuralOriginMode)
    (sharedFacts : List EvidenceOrderingFact)
    (sharedClosures : List EvidenceClosureFact)
    (support : StructuralLinkSupport) :
    NormalizedStructuralLinkSupport × List StructuralFinding :=
  let facts := match originMode with
    | .globalSequence => support.orderingSupport.mergeSort factByRecordLe
    | .sourceSequence | .mixed => support.orderingSupport.mergeSort factBySequenceLe
  let expectedFacts := match originMode with
    | .globalSequence =>
        (sharedFacts.filter fun fact => support.evidenceIdentities.contains fact.recordId).mergeSort
          factByRecordLe
    | .sourceSequence | .mixed => sharedFacts
  let orderingConsistent := match originMode with
    | .globalSequence =>
        facts.length == support.evidenceIdentities.length &&
          DefinitionId.canonicalSet (facts.map EvidenceOrderingFact.recordId) ==
            DefinitionId.canonicalSet support.evidenceIdentities &&
          facts.all fun fact => sharedFacts.contains fact
    | .sourceSequence | .mixed => facts == expectedFacts
  let sortedClosures := support.closureSupport.mergeSort closureLe
  let (closures, duplicateClosureFindings) := canonicalClosures sortedClosures
  let duplicateClosureFindings := duplicateClosureFindings.filterMap fun finding => match finding with
    | .duplicateClosure source kind conflicting =>
        some (.duplicateClosureSupport support.ruleId linkIndex source kind conflicting)
    | _ => none
  let findings :=
    (if orderingConsistent then [] else
      [.inconsistentOrderingSupport support.ruleId expectedFacts facts]) ++
    duplicateClosureFindings ++
    (if closures == sharedClosures then [] else
      [.inconsistentClosureSupport support.ruleId sharedClosures closures])
  ({
    ruleId := support.ruleId
    evidenceIdentities := support.evidenceIdentities
    facts
    closures
  }, findings)

def analyzeStructure
    (directFacts : List EvidenceOrderingFact)
    (directClosures : List EvidenceClosureFact)
    (requiredKinds : List DefinitionId := [])
    (linkSupport : List StructuralLinkSupport := []) : StructuralAnalysis :=
  let suppliedFacts := if linkSupport.isEmpty then directFacts
    else linkSupport.flatMap StructuralLinkSupport.orderingSupport
  let suppliedClosures := if linkSupport.isEmpty then directClosures
    else linkSupport.flatMap StructuralLinkSupport.closureSupport
  let factsById := suppliedFacts.mergeSort factByRecordLe
  let (facts, identityFindings) := canonicalFacts factsById
  let identityFindings := if linkSupport.isEmpty then identityFindings else
    identityFindings.filter fun finding => match finding with
      | .duplicateIdentity _ true => true
      | _ => false
  let facts := facts.mergeSort factBySequenceLe
  let originMode := if facts.isEmpty then .globalSequence
    else if facts.all fun fact => fact.origin.isSome then .sourceSequence
    else if facts.any fun fact => fact.origin.isSome then .mixed
    else .globalSequence
  let orderingFindings := match originMode with
    | .globalSequence => globalSequenceFindings facts
    | .sourceSequence => sourceSequenceFindings facts
    | .mixed => [.mixedOrigins (facts.map EvidenceOrderingFact.recordId)]
  let sortedClosures := suppliedClosures.mergeSort closureLe
  let (closures, duplicateClosureFindings) := canonicalClosures sortedClosures
  let duplicateClosureFindings := if linkSupport.isEmpty then duplicateClosureFindings else
    duplicateClosureFindings.filter fun finding => match finding with
      | .duplicateClosure _ _ true => true
      | _ => false
  let closureExpectations := closureExpectations originMode facts
  let closureFindings := match originMode with
    | .globalSequence => globalClosureFindings requiredKinds closures closureExpectations
    | .sourceSequence => sourceClosureFindings requiredKinds closures closureExpectations
    | .mixed => []
  let normalizedLinks := linkSupport.mapIdx fun linkIndex support =>
    normalizeLinkSupport linkIndex originMode facts closures support
  let links := normalizedLinks.map Prod.fst
  let linkFindings := normalizedLinks.flatMap Prod.snd
  {
    facts
    closures
    originMode
    closureExpectations
    links
    findings := identityFindings ++ orderingFindings ++ duplicateClosureFindings ++ closureFindings ++
      linkFindings
  }

end Observation.Internal

end Umpire
