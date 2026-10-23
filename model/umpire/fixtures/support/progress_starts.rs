use umpire_dsl::{Machine, Progress, ProgressEvaluation, Steps};

pub(crate) struct ProgressStarts<M: Machine> {
    pub(crate) world: M,
    pub(crate) starts: Vec<M::State>,
    pub(crate) claim: Progress<M>,
}

impl<M: Machine> Machine for ProgressStarts<M> {
    type State = M::State;
    type Action = M::Action;
    type Outcome = M::Outcome;
    type Fact = M::Fact;

    fn starts(&self) -> Vec<Self::State> {
        self.starts.clone()
    }

    fn actions(&self) -> Vec<Self::Action> {
        self.world.actions()
    }

    fn internal(&self, action: &Self::Action) -> bool {
        self.world.internal(action)
    }

    fn assumptions(&self) -> Vec<(&'static str, umpire_dsl::Assumption<Self::Action>)> {
        self.world.assumptions()
    }

    fn step(
        &self,
        state: &Self::State,
        action: Self::Action,
    ) -> Steps<Self::State, Self::Outcome, Self::Fact> {
        self.world.step(state, action)
    }

    fn progress(&self) -> Vec<Progress<Self>> {
        vec![Progress {
            name: self.claim.name,
            trigger: |adapter, state| (adapter.claim.trigger)(&adapter.world, state),
            target: |adapter, state| (adapter.claim.target)(&adapter.world, state),
            scope: self.claim.scope.clone(),
            bound: self.claim.bound,
            assumption_override: self.claim.assumption_override.clone(),
            bound_override: self.claim.bound_override,
            evaluator: Some(|adapter, state| {
                if let Some(evaluate) = adapter.claim.evaluator {
                    evaluate(&adapter.world, state)
                } else {
                    ProgressEvaluation {
                        trigger: (adapter.claim.trigger)(&adapter.world, state),
                        target: (adapter.claim.target)(&adapter.world, state),
                        assumption: adapter.claim.scope.clone(),
                        bound: adapter.claim.bound,
                    }
                }
            }),
        }]
    }
}
