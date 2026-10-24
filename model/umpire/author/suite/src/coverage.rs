use std::{
    collections::{HashMap, HashSet},
    fmt,
    panic::{AssertUnwindSafe, catch_unwind},
};

use umpire_dsl::{Machine, Step, Verdict};

use crate::Refusal;

/// One declared catalog member, identified by its declaration ordinal.
#[derive(Debug, PartialEq, Eq)]
pub struct CatalogEntry {
    /// Zero-based ordinal in this catalog, independent of display spelling.
    pub index: usize,
    /// Display spelling only, never used to classify observations.
    pub name: String,
    /// Whether an actual witness observed this member.
    pub touched: bool,
}

/// The registration list containing a claim.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ClaimKind {
    /// A predicate on each witness state.
    Invariant,
    /// A predicate on applicable witness edges.
    Postcondition,
    /// Trigger and target predicates on each witness state.
    Progress,
}

/// Actual callback observations for one registered claim, without a proof verdict.
#[derive(Debug, PartialEq, Eq)]
pub struct ClaimCoverage {
    /// The claim's registration kind.
    pub kind: ClaimKind,
    /// Zero-based ordinal within that kind's registration list.
    pub index: usize,
    /// Authored display spelling; duplicate names remain separate entries.
    pub name: &'static str,
    /// Number of actual witness state or applicable edge evaluations.
    pub evaluated: usize,
    /// Applicable edge count or progress trigger successes; absent for invariants.
    pub triggered: Option<usize>,
}

/// Catalog members and claims observed on selected witnesses only.
#[derive(Debug, PartialEq, Eq)]
pub struct Coverage {
    /// Complete declared action catalog.
    pub actions: Vec<CatalogEntry>,
    /// Declared outcome classes, or unknown metadata when absent.
    pub outcomes: Option<Vec<CatalogEntry>>,
    /// Declared fault actions, or unknown metadata when absent.
    pub faults: Option<Vec<CatalogEntry>>,
    /// Claim callbacks evaluated on these witnesses.
    pub claims: Vec<ClaimCoverage>,
}

pub(crate) struct ObservedStep<'a, M: Machine> {
    pub before: &'a M::State,
    pub action: &'a M::Action,
    pub emitted: &'a Step<M::State, M::Outcome, M::Fact>,
}

pub(crate) struct WitnessObservation<'a, M: Machine> {
    pub states: Vec<&'a M::State>,
    pub rows: Vec<ObservedStep<'a, M>>,
}

pub(crate) fn coverage<M: Machine>(
    world: &M,
    catalog_limit: usize,
    traces: &[WitnessObservation<'_, M>],
) -> Result<Coverage, Refusal> {
    catch_unwind(AssertUnwindSafe(|| observe(world, catalog_limit, traces))).unwrap_or_else(
        |panic| {
            let failure = panic
                .downcast_ref::<String>()
                .map(String::as_str)
                .or_else(|| panic.downcast_ref::<&str>().copied())
                .unwrap_or("non-string panic");
            Err(metadata(format!("callback panicked: {failure}")))
        },
    )
}

fn metadata(detail: impl Into<String>) -> Refusal {
    Refusal::Metadata { detail: detail.into() }
}

fn observe<M: Machine>(
    world: &M,
    catalog_limit: usize,
    traces: &[WitnessObservation<'_, M>],
) -> Result<Coverage, Refusal> {
    let actions =
        world.checked_catalog(catalog_limit).map_err(umpire_checker::Refusal::from)?.into_actions();
    let mut action_ordinals = HashMap::with_capacity(actions.len());
    for (index, action) in actions.iter().enumerate() {
        if action_ordinals.insert(action, index).is_some() {
            return Err(metadata("duplicate action identity in the action catalog"));
        }
    }
    let outcomes = world.outcome_catalog();
    if let Some(classes) = &outcomes {
        let mut names = HashSet::with_capacity(classes.len());
        for class in classes {
            if !names.insert(class.name) {
                return Err(metadata(format!("duplicate outcome class name: {}", class.name)));
            }
        }
    }
    let faults = world.fault_catalog();
    let mut fault_ordinals = Vec::new();
    if let Some(faults) = &faults {
        let mut seen = HashSet::with_capacity(faults.len());
        for (index, fault) in faults.iter().enumerate() {
            let Some(&ordinal) = action_ordinals.get(fault) else {
                return Err(metadata(format!("fault {index} is outside the action catalog")));
            };
            if !seen.insert(fault) {
                return Err(metadata(format!("duplicate fault action identity at {index}")));
            }
            fault_ordinals.push(ordinal);
        }
    }
    let mut result = Coverage {
        actions: entries(actions.iter().map(ToString::to_string)),
        outcomes: outcomes.as_ref().map(|classes| entries(classes.iter().map(|c| c.name.into()))),
        faults: faults.as_ref().map(|faults| entries(faults.iter().map(ToString::to_string))),
        claims: Vec::new(),
    };
    for row in traces.iter().flat_map(|trace| &trace.rows) {
        let Some(&index) = action_ordinals.get(row.action) else {
            return Err(metadata("observed action is outside the action catalog"));
        };
        result.actions[index].touched = true;
        if let (Some(classes), Some(entries)) = (&outcomes, &mut result.outcomes) {
            let mut matched = None;
            for (index, class) in classes.iter().enumerate() {
                if class.matches(&row.emitted.0) && matched.replace(index).is_some() {
                    return Err(metadata("observed outcome matches overlapping outcome classes"));
                }
            }
            let Some(index) = matched else {
                return Err(metadata("observed outcome matches no declared outcome class"));
            };
            entries[index].touched = true;
        }
    }
    if let Some(entries) = &mut result.faults {
        for (entry, ordinal) in entries.iter_mut().zip(fault_ordinals) {
            entry.touched = result.actions[ordinal].touched;
        }
    }
    result.claims = observe_claims(world, traces);
    Ok(result)
}

fn observe_claims<M: Machine>(
    world: &M,
    traces: &[WitnessObservation<'_, M>],
) -> Vec<ClaimCoverage> {
    let mut claims = Vec::new();
    for (index, claim) in world.invariants().iter().enumerate() {
        let mut entry = claim_entry(ClaimKind::Invariant, index, claim.name, None);
        for state in traces.iter().flat_map(|trace| &trace.states) {
            claim.evaluate(world, state);
            entry.evaluated += 1;
        }
        claims.push(entry);
    }
    for (index, claim) in world.postconditions().iter().enumerate() {
        let mut entry = claim_entry(ClaimKind::Postcondition, index, claim.name, Some(0));
        for row in traces.iter().flat_map(|trace| &trace.rows) {
            if claim.applies_to(row.action) {
                claim.evaluate(world, row.before, row.action, row.emitted);
                entry.evaluated += 1;
            }
        }
        entry.triggered = Some(entry.evaluated);
        claims.push(entry);
    }
    for (index, claim) in world.progress().iter().enumerate() {
        let mut entry = claim_entry(ClaimKind::Progress, index, claim.name, Some(0));
        let mut triggered = 0;
        for state in traces.iter().flat_map(|trace| &trace.states) {
            let trigger = if let Some(evaluator) = claim.evaluator {
                evaluator(world, state).trigger
            } else {
                let trigger = (claim.trigger)(world, state);
                (claim.target)(world, state);
                trigger
            };
            entry.evaluated += 1;
            triggered += usize::from(trigger == Verdict::Holds);
        }
        entry.triggered = Some(triggered);
        claims.push(entry);
    }
    claims
}

fn entries(names: impl Iterator<Item = String>) -> Vec<CatalogEntry> {
    names.enumerate().map(|(index, name)| CatalogEntry { index, name, touched: false }).collect()
}

fn claim_entry(
    kind: ClaimKind,
    index: usize,
    name: &'static str,
    triggered: Option<usize>,
) -> ClaimCoverage {
    ClaimCoverage { kind, index, name, evaluated: 0, triggered }
}

impl fmt::Display for ClaimKind {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.write_str(match self {
            Self::Invariant => "invariant",
            Self::Postcondition => "postcondition",
            Self::Progress => "progress",
        })
    }
}

impl fmt::Display for Coverage {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        catalog(out, "actions", Some(&self.actions))?;
        catalog(out, "outcomes", self.outcomes.as_deref())?;
        catalog(out, "faults", self.faults.as_deref())?;
        writeln!(out, "claims (witness evaluations only):")?;
        for claim in &self.claims {
            write!(
                out,
                "  {}[{}] {}: {} evaluated={}",
                claim.kind,
                claim.index,
                claim.name,
                if claim.evaluated > 0 { "touched" } else { "untouched" },
                claim.evaluated
            )?;
            if let Some(triggered) = claim.triggered {
                write!(out, " triggered={triggered}")?;
            }
            writeln!(out)?;
        }
        Ok(())
    }
}

fn catalog(
    out: &mut fmt::Formatter<'_>,
    name: &str,
    entries: Option<&[CatalogEntry]>,
) -> fmt::Result {
    match entries {
        None => writeln!(out, "{name}: unknown (undeclared catalog)"),
        Some([]) => writeln!(out, "{name}: complete (empty catalog)"),
        Some(entries) => {
            writeln!(out, "{name}:")?;
            for entry in entries {
                writeln!(
                    out,
                    "  [{}] {}: {}",
                    entry.index,
                    entry.name,
                    if entry.touched { "touched" } else { "untouched" }
                )?;
            }
            Ok(())
        }
    }
}

#[cfg(test)]
mod tests {
    use std::{
        cell::Cell,
        hash::{Hash, Hasher},
    };

    use umpire_dsl::{
        Assumption, Claim, OutcomeClass, Progress, ProgressEvaluation, StepClaim, Steps, Verdict,
    };

    use super::*;

    thread_local! {
        static EQUALITY_CALLS: Cell<usize> = const { Cell::new(0) };
    }

    #[derive(Clone, Eq)]
    struct CountedAction(usize);

    impl Hash for CountedAction {
        fn hash<H: Hasher>(&self, state: &mut H) {
            self.0.hash(state);
        }
    }

    impl PartialEq for CountedAction {
        fn eq(&self, other: &Self) -> bool {
            EQUALITY_CALLS.update(|count| count + 1);
            self.0 == other.0
        }
    }

    impl fmt::Display for CountedAction {
        fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
            self.0.fmt(out)
        }
    }

    struct LargeCatalog;

    impl Machine for LargeCatalog {
        type State = u8;
        type Action = CountedAction;
        type Outcome = ();
        type Fact = ();

        fn starts(&self) -> Vec<u8> {
            vec![0]
        }

        fn actions(&self) -> Vec<CountedAction> {
            (0..512).map(CountedAction).collect()
        }

        fn fault_catalog(&self) -> Option<Vec<CountedAction>> {
            Some((0..512).rev().map(CountedAction).collect())
        }

        fn internal(&self, _: &CountedAction) -> bool {
            false
        }

        fn step(&self, _: &u8, _: CountedAction) -> Steps<u8, (), ()> {
            unreachable!("coverage must use retained observations")
        }
    }

    #[test]
    fn catalog_validation_and_row_classification_avoid_quadratic_equality_scans()
    -> Result<(), Refusal> {
        let actions = LargeCatalog.actions();
        let emitted = ((), vec![], 0);
        let observations = [WitnessObservation {
            states: vec![&0],
            rows: actions
                .iter()
                .map(|action| ObservedStep { before: &0, action, emitted: &emitted })
                .collect(),
        }];
        EQUALITY_CALLS.set(0);
        let result =
            coverage(&LargeCatalog, umpire_checker::Checker::CATALOG_LIMIT, &observations)?;
        let comparisons = EQUALITY_CALLS.get();
        assert!(comparisons < 512 * 32, "catalog used {comparisons} equality comparisons");
        assert_eq!(result.actions.len(), 512);
        assert!(result.actions.iter().all(|entry| entry.touched));
        let faults = result.faults.ok_or_else(|| metadata("expected declared faults"))?;
        assert_eq!(faults[0].name, "511");
        assert_eq!(faults[511].name, "0");
        assert!(faults.iter().all(|entry| entry.touched));
        Ok(())
    }

    #[derive(Clone, Copy, PartialEq, Eq, Hash)]
    enum Action {
        A,
        B,
        C,
    }

    impl fmt::Display for Action {
        fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
            out.write_str("same")
        }
    }

    #[derive(Debug)]
    struct Outcome(bool);

    #[derive(Clone, Copy, Default)]
    enum Metadata {
        #[default]
        Valid,
        Undeclared,
        Empty,
        DuplicateAction,
        DuplicateClass,
        Overlap,
        Unmatched,
        OutsideFault,
        DuplicateFault,
        ClassifierPanic,
        ClaimPanic,
        TargetPanic,
        Combined,
    }

    struct World(Metadata);

    impl Machine for World {
        type State = u8;
        type Action = Action;
        type Outcome = Outcome;
        type Fact = ();

        fn starts(&self) -> Vec<u8> {
            vec![0]
        }

        fn actions(&self) -> Vec<Action> {
            if matches!(self.0, Metadata::DuplicateAction) {
                vec![Action::A, Action::A]
            } else {
                vec![Action::A, Action::B]
            }
        }

        fn internal(&self, _: &Action) -> bool {
            false
        }

        fn step(&self, _: &u8, _: Action) -> Steps<u8, Outcome, ()> {
            unreachable!("coverage must use retained observations")
        }

        fn outcome_catalog(&self) -> Option<Vec<OutcomeClass<Outcome>>> {
            match self.0 {
                Metadata::Undeclared => None,
                Metadata::Empty => Some(vec![]),
                Metadata::DuplicateClass => Some(vec![
                    OutcomeClass::new("same", |o| o.0),
                    OutcomeClass::new("same", |o| !o.0),
                ]),
                Metadata::Overlap => {
                    Some(vec![OutcomeClass::new("a", |_| true), OutcomeClass::new("b", |_| true)])
                }
                Metadata::Unmatched => Some(vec![OutcomeClass::new("none", |_| false)]),
                Metadata::ClassifierPanic => {
                    Some(vec![OutcomeClass::new("panic", |_| panic!("classifier"))])
                }
                _ => {
                    Some(vec![OutcomeClass::new("yes", |o| o.0), OutcomeClass::new("no", |o| !o.0)])
                }
            }
        }

        fn fault_catalog(&self) -> Option<Vec<Action>> {
            match self.0 {
                Metadata::Undeclared => None,
                Metadata::Empty => Some(vec![]),
                Metadata::OutsideFault => Some(vec![Action::C]),
                Metadata::DuplicateFault => Some(vec![Action::B, Action::B]),
                _ => Some(vec![Action::A, Action::B]),
            }
        }

        fn invariants(&self) -> Vec<Claim<Self>> {
            vec![Claim::new("same", |world, _| {
                assert!(!matches!(world.0, Metadata::ClaimPanic), "claim");
                Verdict::Holds
            })]
        }

        fn postconditions(&self) -> Vec<StepClaim<Self>> {
            vec![
                StepClaim::after("same", Action::A, |_, _, _, _, o: &Outcome, _| {
                    assert!(o.0);
                    Verdict::Holds
                }),
                StepClaim::after("same", Action::B, |_, _, _, _, _, _| panic!("not observed")),
            ]
        }

        fn progress(&self) -> Vec<Progress<Self>> {
            if matches!(self.0, Metadata::Combined) {
                return vec![Progress::evaluated("same", |_, state| ProgressEvaluation {
                    trigger: (*state == 1).into(),
                    target: Verdict::Fails,
                    assumption: Assumption::default(),
                    bound: None,
                })];
            }
            if matches!(self.0, Metadata::TargetPanic) {
                return vec![Progress::new("same").eventually(|_, _| panic!("target"))];
            }
            vec![Progress::new("same").whenever(|_, _| Verdict::Fails)]
        }
    }

    #[test]
    fn classifies_retained_outcomes_and_equal_display_actions_by_typed_identity()
    -> Result<(), Refusal> {
        let emitted = (Outcome(true), vec![], 1);
        let observations = [WitnessObservation {
            states: vec![&0, &emitted.2],
            rows: vec![ObservedStep { before: &0, action: &Action::A, emitted: &emitted }],
        }];
        let result = coverage(
            &World(Metadata::Valid),
            umpire_checker::Checker::CATALOG_LIMIT,
            &observations,
        )?;
        assert_eq!(
            result.actions.iter().map(|e| (e.index, e.touched)).collect::<Vec<_>>(),
            [(0, true), (1, false)]
        );
        assert_eq!(
            result
                .outcomes
                .as_ref()
                .map(|entries| entries.iter().map(|e| e.touched).collect::<Vec<_>>()),
            Some(vec![true, false])
        );
        assert_eq!(
            result
                .faults
                .as_ref()
                .map(|entries| entries.iter().map(|e| e.touched).collect::<Vec<_>>()),
            Some(vec![true, false])
        );
        assert_eq!(
            result
                .claims
                .iter()
                .map(|c| (c.kind, c.index, c.evaluated, c.triggered))
                .collect::<Vec<_>>(),
            [
                (ClaimKind::Invariant, 0, 2, None),
                (ClaimKind::Postcondition, 0, 1, Some(1)),
                (ClaimKind::Postcondition, 1, 0, Some(0)),
                (ClaimKind::Progress, 0, 2, Some(0)),
            ]
        );
        Ok(())
    }

    #[test]
    fn undeclared_and_complete_empty_catalogs_remain_distinct() -> Result<(), Refusal> {
        let unknown =
            coverage(&World(Metadata::Undeclared), umpire_checker::Checker::CATALOG_LIMIT, &[])?;
        let empty = coverage(&World(Metadata::Empty), umpire_checker::Checker::CATALOG_LIMIT, &[])?;
        assert!(unknown.outcomes.is_none());
        assert_eq!(empty.outcomes, Some(vec![]));
        assert!(unknown.to_string().contains("unknown"));
        assert!(empty.to_string().contains("empty"));
        assert!(empty.claims.iter().all(|claim| claim.evaluated == 0));
        Ok(())
    }

    #[test]
    fn progress_uses_the_combined_evaluator_on_witness_states() -> Result<(), Refusal> {
        let observations = [WitnessObservation { states: vec![&0, &1], rows: vec![] }];
        let result = coverage(
            &World(Metadata::Combined),
            umpire_checker::Checker::CATALOG_LIMIT,
            &observations,
        )?;
        assert_eq!(result.claims.last().map(|p| (p.evaluated, p.triggered)), Some((2, Some(1))));
        Ok(())
    }

    #[test]
    fn refuses_invalid_metadata_and_panicking_callbacks() {
        let emitted = (Outcome(true), vec![], 1);
        let observations = [WitnessObservation {
            states: vec![&0],
            rows: vec![ObservedStep { before: &0, action: &Action::A, emitted: &emitted }],
        }];
        for mode in [
            Metadata::DuplicateAction,
            Metadata::DuplicateClass,
            Metadata::Overlap,
            Metadata::Unmatched,
            Metadata::OutsideFault,
            Metadata::DuplicateFault,
            Metadata::ClassifierPanic,
            Metadata::ClaimPanic,
            Metadata::TargetPanic,
        ] {
            assert!(matches!(
                coverage(&World(mode), umpire_checker::Checker::CATALOG_LIMIT, &observations),
                Err(Refusal::Metadata { .. })
            ));
        }
    }
}
