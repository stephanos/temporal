import Umpire.Evidence.Evaluate.Types
import Shared.Reachability

/-!
The Evidence structure: the ordering facts, closures, and per-link support of one offline Evidence
bundle or accepted trace, normalized once and judged for one audience.

`EvidenceStructure.analyze` normalizes facts and closures a raw bundle supplies directly, or the ones
an accepted trace's Evidence Links carry. `orderingFault?` and `closureFault?` return the first fault
in the audience's precedence table, with the related identities that audience reports:

* `raw` judges a bundle. Identity and origin faults come first; fault receipts and causal parents are
  judged per record in canonical order, and closures per required kind (global sequences) or per
  closure and then per record in bundle order (source sequences).
* `accepted` judges an admitted envelope. Conflicting identities and uncovered evidence come first,
  then the first ordering fault and link order; closures are judged per link, per fact in canonical
  order, per closure, and then per required kind.

Both audiences judge ordering before closures. The findings, closure expectations, and origin-mode
branch behind these tables are private; callers only name their own failure kind for a fault.
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

namespace EvidenceStructure

/-- Who judges an Evidence structure: a raw Evidence bundle, or an accepted trace's envelope. -/
inductive Audience where
  | raw
  | accepted
  deriving BEq, DecidableEq, Repr

/-- A raw record's claim to receive the fault injected into `target`. -/
structure FaultReceipt where
  recordId : DefinitionId
  target : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- One Evidence Link's ordering and closure support, as an accepted trace carries it. -/
structure LinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

/-- Every Evidence Link of an accepted trace, with the evidence identities its envelope claims. -/
structure LinkedSupport where
  evidenceIdentities : List DefinitionId
  links : List LinkSupport
  deriving BEq, DecidableEq, Repr

/-- One Evidence Link's support after canonical ordering and closure normalization. -/
structure NormalizedLinkSupport where
  ruleId : DefinitionId
  evidenceIdentities : List DefinitionId
  facts : List EvidenceOrderingFact
  closures : List EvidenceClosureFact
  deriving BEq, DecidableEq, Repr

/--
The kind of an ordering fault. Fault receipts exist only in raw bundles, and uncovered evidence and
link order only in accepted envelopes, so the audience index rules them out for the other audience.
-/
inductive OrderingFaultKind : Audience → Type where
  | duplicateIdentity : OrderingFaultKind audience
  | incomparableOrder : OrderingFaultKind audience
  | sequenceGap : OrderingFaultKind audience
  | missingCausalParent : OrderingFaultKind audience
  | contradictoryOrder : OrderingFaultKind audience
  | misdirectedFaultReceipt : OrderingFaultKind .raw
  | uncoveredEvidence : OrderingFaultKind .accepted
  | inconsistentLinkOrder : OrderingFaultKind .accepted
  deriving BEq, DecidableEq, Repr

/-- The first ordering fault for one audience and the identities that audience reports with it. -/
structure OrderingFault (audience : Audience) where
  kind : OrderingFaultKind audience
  related : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- The first closure fault for one audience and the identities that audience reports with it. -/
structure ClosureFault where
  related : List DefinitionId
  deriving BEq, DecidableEq, Repr

private inductive OriginMode where
  | globalSequence
  | sourceSequence
  | mixed
  deriving BEq, DecidableEq, Repr

private inductive Finding where
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

private structure ClosureExpectation where
  source : Option DefinitionId
  kind : DefinitionId
  recordIds : List DefinitionId
  lastSequence : Nat
  recordCount : Nat
  deriving BEq, DecidableEq, Repr

end EvidenceStructure

open EvidenceStructure in
/--
An analyzed Evidence structure. Its constructor and fields are private: callers analyze it with
`EvidenceStructure.analyze`, judge it with `orderingFault?` and `closureFault?`, and read it only
through `factsInOrder` and `linkSupport`.
-/
structure EvidenceStructure where
  private mk ::
  private facts : List EvidenceOrderingFact
  private suppliedFacts : List EvidenceOrderingFact
  private closures : List EvidenceClosureFact
  private requiredKinds : List DefinitionId
  private evidenceIdentities : List DefinitionId
  private faultReceipts : List FaultReceipt
  private originMode : OriginMode
  private closureExpectations : List ClosureExpectation
  private links : List NormalizedLinkSupport
  private findings : List Finding

namespace EvidenceStructure

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
    List EvidenceOrderingFact → List EvidenceOrderingFact × List Finding
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
    List EvidenceClosureFact → List EvidenceClosureFact × List Finding
  | [] => ([], [])
  | [closure] => ([closure], [])
  | first :: second :: rest =>
      if first.source == second.source && first.kind == second.kind then
        let (closures, findings) := canonicalClosures (first :: rest)
        (closures, .duplicateClosure first.source first.kind (first != second) :: findings)
      else
        let (closures, findings) := canonicalClosures (second :: rest)
        (first :: closures, findings)

/-- `ancestor` is `recordId` or reaches it through causal parents; facts carry unique identities. -/
private def factDescendsFrom
    (facts : List EvidenceOrderingFact)
    (recordId ancestor : DefinitionId) : Bool :=
  Shared.Reachability.reaches facts EvidenceOrderingFact.recordId
    (fun current candidate => candidate.causalParents.contains current) ancestor recordId

private def globalSequenceFindings
    (facts : List EvidenceOrderingFact) : List Finding := Id.run do
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
          if factDescendsFrom facts parent fact.recordId || parentFact.sequence >= fact.sequence then
            findings := findings ++ [.contradictoryOrder fact.recordId parent]
    previous := some fact
    expectedSequence := expectedSequence + 1
  pure findings

private def sourceSequenceFindings
    (facts : List EvidenceOrderingFact) : List Finding := Id.run do
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
          if factDescendsFrom facts parent fact.recordId || reversesSourceOrder then
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

private def expectationsFor
    (originMode : OriginMode)
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
    (expectations : List ClosureExpectation) : List Finding := Id.run do
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
    (expectations : List ClosureExpectation) : List Finding := Id.run do
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
    (originMode : OriginMode)
    (sharedFacts : List EvidenceOrderingFact)
    (sharedClosures : List EvidenceClosureFact)
    (support : LinkSupport) :
    NormalizedLinkSupport × List Finding :=
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

/--
Normalize one Evidence structure. A raw bundle supplies `facts` in bundle order with each record's
causal parents as written, its `closures`, and its `faultReceipts`; an accepted trace supplies its
Evidence Links in `linked`, whose ordering and closure support replace `facts` and `closures`
whenever at least one link is present. `requiredKinds` are the closure kinds the checked Reading
declares, in declaration order.
-/
def analyze
    (facts : List EvidenceOrderingFact)
    (closures : List EvidenceClosureFact)
    (requiredKinds : List DefinitionId := [])
    (linked : Option LinkedSupport := none)
    (faultReceipts : List FaultReceipt := []) : EvidenceStructure :=
  let linkSupport := linked.map LinkedSupport.links |>.getD []
  let suppliedFacts := if linkSupport.isEmpty then facts
    else linkSupport.flatMap LinkSupport.orderingSupport
  let suppliedClosures := if linkSupport.isEmpty then closures
    else linkSupport.flatMap LinkSupport.closureSupport
  let factsById := suppliedFacts.mergeSort factByRecordLe
  let (canonical, identityFindings) := canonicalFacts factsById
  let identityFindings := if linkSupport.isEmpty then identityFindings else
    identityFindings.filter fun finding => match finding with
      | .duplicateIdentity _ true => true
      | _ => false
  let canonical := canonical.mergeSort factBySequenceLe
  let originMode := if canonical.isEmpty then .globalSequence
    else if canonical.all fun fact => fact.origin.isSome then .sourceSequence
    else if canonical.any fun fact => fact.origin.isSome then .mixed
    else .globalSequence
  let orderingFindings := match originMode with
    | .globalSequence => globalSequenceFindings canonical
    | .sourceSequence => sourceSequenceFindings canonical
    | .mixed => [.mixedOrigins (canonical.map EvidenceOrderingFact.recordId)]
  let sortedClosures := suppliedClosures.mergeSort closureLe
  let (normalizedClosures, duplicateClosureFindings) := canonicalClosures sortedClosures
  let duplicateClosureFindings := if linkSupport.isEmpty then duplicateClosureFindings else
    duplicateClosureFindings.filter fun finding => match finding with
      | .duplicateClosure _ _ true => true
      | _ => false
  let closureExpectations := expectationsFor originMode canonical
  let closureFindings := match originMode with
    | .globalSequence => globalClosureFindings requiredKinds normalizedClosures closureExpectations
    | .sourceSequence => sourceClosureFindings requiredKinds normalizedClosures closureExpectations
    | .mixed => []
  let normalizedLinks := linkSupport.mapIdx fun linkIndex support =>
    normalizeLinkSupport linkIndex originMode canonical normalizedClosures support
  let links := normalizedLinks.map Prod.fst
  let linkFindings := normalizedLinks.flatMap Prod.snd
  {
    facts := canonical
    suppliedFacts
    closures := normalizedClosures
    requiredKinds
    evidenceIdentities := linked.map LinkedSupport.evidenceIdentities |>.getD []
    faultReceipts
    originMode
    closureExpectations
    links
    findings := identityFindings ++ orderingFindings ++ duplicateClosureFindings ++ closureFindings ++
      linkFindings
  }

/-- The canonical facts: one per record identity, in source-local or global sequence order. -/
def factsInOrder (evidence : EvidenceStructure) : List EvidenceOrderingFact :=
  evidence.facts

/-- Each supplied Evidence Link's support after normalization, in link order. -/
def linkSupport (evidence : EvidenceStructure) : List NormalizedLinkSupport :=
  evidence.links

private def firstFault (check : Except α Unit) : Option α :=
  match check with
  | .ok () => none
  | .error fault => some fault

private def throwFirst (fault : Option α) : Except α Unit :=
  match fault with
  | some fault => throw fault
  | none => pure ()

private def rawFaultReceipt
    (evidence : EvidenceStructure)
    (fact : EvidenceOrderingFact) : Except (OrderingFault .raw) Unit := do
  let some receipt := evidence.faultReceipts.find? fun receipt => receipt.recordId == fact.recordId
    | return
  let misdirected : OrderingFault .raw := {
    kind := .misdirectedFaultReceipt
    related := [fact.recordId, receipt.target]
  }
  let targetFact ← match evidence.facts.find? fun candidate => candidate.recordId == receipt.target with
    | some candidate => pure candidate
    | none => throw misdirected
  match evidence.originMode with
  | .globalSequence =>
      if targetFact.sequence >= fact.sequence then
        throw misdirected
  | .sourceSequence =>
      let sameSourceBefore := match targetFact.origin, fact.origin with
        | some targetOrigin, some factOrigin =>
            targetOrigin.source == factOrigin.source &&
              targetOrigin.ordinal < factOrigin.ordinal
        | _, _ => false
      if !sameSourceBefore && !factDescendsFrom evidence.facts fact.recordId receipt.target then
        throw misdirected
  | .mixed => pure ()

private def rawParentFault
    (evidence : EvidenceStructure)
    (fact : EvidenceOrderingFact)
    (parentId : DefinitionId) : Except (OrderingFault .raw) Unit :=
  throwFirst <| evidence.findings.findSome? fun
    | .missingCausalParent candidate (some candidateParent) =>
        if candidate == fact.recordId && candidateParent == parentId then
          some { kind := .missingCausalParent, related := [candidate, candidateParent] }
        else none
    | .contradictoryOrder candidate candidateParent =>
        if candidate == fact.recordId && candidateParent == parentId then
          some { kind := .contradictoryOrder, related := [candidate, candidateParent] }
        else none
    | _ => none

private def rawOrdering (evidence : EvidenceStructure) : Except (OrderingFault .raw) Unit := do
  throwFirst <| evidence.findings.findSome? fun
    | .duplicateIdentity recordId _ => some { kind := .duplicateIdentity, related := [recordId] }
    | _ => none
  throwFirst <| evidence.findings.findSome? fun
    | .mixedOrigins recordIds => some { kind := .incomparableOrder, related := recordIds }
    | _ => none
  match evidence.originMode with
  | .globalSequence =>
      for fact in evidence.facts do
        rawFaultReceipt evidence fact
      for fact in evidence.facts do
        throwFirst <| evidence.findings.findSome? fun
          | .duplicateSequence firstId secondId _ =>
              if secondId == fact.recordId then
                some { kind := .incomparableOrder, related := [firstId, secondId] }
              else none
          | .sequenceGap candidate source _ _ =>
              if candidate == fact.recordId then
                some { kind := .sequenceGap, related := candidate :: source.toList }
              else none
          | .missingCausalParent candidate none =>
              if candidate == fact.recordId then
                some { kind := .missingCausalParent, related := [candidate] }
              else none
          | _ => none
        for parent in fact.causalParents do
          rawParentFault evidence fact parent
  | .sourceSequence =>
      throwFirst <| evidence.findings.findSome? fun
        | .duplicateSequence firstId secondId _ =>
            some { kind := .incomparableOrder, related := [firstId, secondId] }
        | .sequenceGap recordId source _ _ =>
            some { kind := .sequenceGap, related := recordId :: source.toList }
        | _ => none
      for fact in evidence.facts do
        for parent in fact.causalParents do
          rawParentFault evidence fact parent
        rawFaultReceipt evidence fact
  | .mixed => pure ()

private def rawClosures (evidence : EvidenceStructure) : Except ClosureFault Unit := do
  let firstDuplicateClosure : Option ClosureFault := evidence.findings.findSome? fun
    | .duplicateClosure _ kind _ => some { related := [kind] }
    | _ => none
  match evidence.originMode with
  | .globalSequence =>
      for required in evidence.requiredKinds do
        throwFirst <| evidence.findings.findSome? fun
          | .duplicateClosure _ candidate _ =>
              if candidate == required then some { related := [required] } else none
          | _ => none
        let closure ← match evidence.closures.find? fun closure => closure.kind == required with
          | some closure => pure closure
          | none => throw { related := [required] }
        let lastSequence := evidence.closureExpectations.find?
          (fun expectation => expectation.kind == required)
          |>.map ClosureExpectation.lastSequence
          |>.getD 0
        if closure.lastSequence != lastSequence then
          throw { related := [required] }
      throwFirst firstDuplicateClosure
  | .sourceSequence =>
      throwFirst firstDuplicateClosure
      for closure in evidence.closures do
        if closure.source.isNone ||
            !(evidence.requiredKinds.any fun required => required == closure.kind) then
          throw { related := [closure.kind] }
        throwFirst <| evidence.findings.findSome? fun
          | .closureWithoutFacts source kind | .closureSequenceMismatch source kind _ _ |
              .closureCountMismatch source kind _ _ | .closureByteCountMissing source kind =>
              if source == closure.source && kind == closure.kind then
                some { related := source.toList ++ [kind] }
              else none
          | _ => none
      -- Bundle order decides which record of a missing closure is reported.
      for fact in evidence.suppliedFacts do
        throwFirst <| evidence.findings.findSome? fun
          | .missingClosure recordIds source kind =>
              if recordIds.contains fact.recordId &&
                  source == fact.origin.map EvidenceOrigin.source && kind == fact.kind then
                some { related := fact.recordId :: source.toList ++ [kind] }
              else none
          | _ => none
      throwFirst <| evidence.findings.findSome? fun
        | .missingRequiredKind kind => some { related := [kind] }
        | _ => none
  | .mixed => pure ()

private def acceptedOrdering
    (evidence : EvidenceStructure) : Except (OrderingFault .accepted) Unit := do
  throwFirst <| evidence.findings.findSome? fun
    | .duplicateIdentity recordId true => some { kind := .duplicateIdentity, related := [recordId] }
    | _ => none
  if evidence.facts.map EvidenceOrderingFact.recordId != evidence.evidenceIdentities then
    throw { kind := .uncoveredEvidence, related := [] }
  throwFirst <| evidence.findings.findSome? fun
    | .mixedOrigins _ => some { kind := .incomparableOrder, related := [] }
    | .duplicateSequence _ secondId _ => some { kind := .incomparableOrder, related := [secondId] }
    | .sequenceGap recordId source _ _ =>
        some { kind := .sequenceGap, related := recordId :: source.toList }
    | .missingCausalParent recordId parentId =>
        some { kind := .missingCausalParent, related := recordId :: parentId.toList }
    | .contradictoryOrder recordId parentId =>
        some { kind := .contradictoryOrder, related := [recordId, parentId] }
    | _ => none
  throwFirst <| evidence.findings.findSome? fun
    | .inconsistentOrderingSupport ruleId _ _ =>
        some { kind := .inconsistentLinkOrder, related := [ruleId] }
    | _ => none

private def acceptedClosures (evidence : EvidenceStructure) : Except ClosureFault Unit := do
  let firstClosures := evidence.links.head?.map NormalizedLinkSupport.closures |>.getD []
  if firstClosures.isEmpty then
    throw { related := [] }
  throwFirst <| evidence.findings.findSome? fun
    | .duplicateClosureSupport ruleId linkIndex _ kind _ =>
        some { related := if linkIndex == 0 then [kind] else [ruleId] }
    | .inconsistentClosureSupport ruleId _ _ => some { related := [ruleId] }
    | _ => none
  let sourced := evidence.originMode == .sourceSequence
  for fact in evidence.facts do
    throwFirst <| evidence.findings.findSome? fun
      | .missingClosure recordIds source kind =>
          if recordIds.contains fact.recordId && kind == fact.kind &&
              source == if sourced then fact.origin.map EvidenceOrigin.source else none then
            some { related := match evidence.originMode, fact.origin with
              | .sourceSequence, some origin => [fact.recordId, origin.source, fact.kind]
              | _, _ => [fact.recordId, fact.kind] }
          else none
      | _ => none
  let closureRelated (source : Option DefinitionId) (kind : DefinitionId) : ClosureFault := {
    related := match evidence.originMode, source with
      | .sourceSequence, some sourceId => [sourceId, kind]
      | _, _ => [kind]
  }
  for closure in evidence.closures do
    throwFirst <| evidence.findings.findSome? fun
      | .closureWithoutFacts source kind =>
          if source == closure.source && kind == closure.kind then
            -- An explicit zero-record global closure is satisfied without facts.
            if evidence.originMode == .globalSequence && closure.lastSequence == 0 then none
            else some (closureRelated source kind)
          else none
      | .closureSequenceMismatch source kind _ _ | .closureCountMismatch source kind _ _ |
          .closureByteCountMissing source kind =>
          if source == closure.source && kind == closure.kind then
            some (closureRelated source kind)
          else none
      | _ => none
  match evidence.originMode with
  | .globalSequence =>
      for required in evidence.requiredKinds do
        match evidence.closures.find? fun closure => closure.kind == required with
        | some closure =>
            if !(evidence.closureExpectations.any fun expectation =>
                expectation.kind == required) && closure.lastSequence != 0 then
              throw { related := [required] }
        | none => throw { related := [required] }
  | .sourceSequence | .mixed =>
      throwFirst <| evidence.findings.findSome? fun
        | .missingRequiredKind kind => some { related := [kind] }
        | _ => none

/--
The first ordering fault in the audience's precedence table, or `none` when the facts order.
Ordering faults precede every closure fault for both audiences.
-/
def orderingFault? (evidence : EvidenceStructure) : (audience : Audience) → Option (OrderingFault audience)
  | .raw => firstFault (rawOrdering evidence)
  | .accepted => firstFault (acceptedOrdering evidence)

/-- The first closure fault in the audience's precedence table, or `none` when closures cover the facts. -/
def closureFault? (evidence : EvidenceStructure) : Audience → Option ClosureFault
  | .raw => firstFault (rawClosures evidence)
  | .accepted => firstFault (acceptedClosures evidence)

end EvidenceStructure

end Umpire
