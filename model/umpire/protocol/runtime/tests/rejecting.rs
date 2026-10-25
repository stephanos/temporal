//! Invalid rejecting steps are typed refusals on every runtime evaluation path.

use prost::Message;
use umpire_runtime::{
    AnswerPlan, Projection, Realize, Runtime, Transition, prost_types::Any, protocol,
};

#[umpire_dsl::machine]
/// A deliberately invalid Model for transactional refusal controls.
pub mod invalid {
    use umpire_dsl::{Action, Rejection, State};

    #[derive(State)]
    /// A value changed only by accepted steps in a valid Model.
    pub struct Model {
        #[starts(0..=0)]
        /// The current value.
        pub value: u8,
    }

    #[derive(Action, Copy)]
    /// An accepted step and an invalid rejecting step.
    pub enum User {
        /// Advances normally.
        Good,
        /// Mutates and then rejects.
        Bad,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Good if self.value == 0 => self.value = 1,
                User::Bad => {
                    self.value = 2;
                    umpire_dsl::reject!(Rejection::NotFound);
                }
                User::Good => {}
            }
        }
    }
}

struct Realization;

fn transition(action: invalid::User) -> Transition<invalid::Action> {
    Transition { action: action.into(), choice: 0 }
}

impl umpire_runtime::capability::Identity for Realization {
    type Id = ();

    fn identify(&self, _: &Any) -> Result<(), protocol::Refusal> {
        Ok(())
    }
}

impl umpire_runtime::capability::Fill<<invalid::World as umpire_dsl::Machine>::Action>
    for Realization
{
    type World = invalid::World;
    type Inputs = ();
    type Request = umpire_runtime::capability::NoRequest;
}

impl umpire_runtime::capability::Snapshot<invalid::World> for Realization {
    type Target = umpire_runtime::NoTarget;

    fn project(
        &self,
        _world: &invalid::World,
        observation: &Any,
        state: &invalid::Model,
    ) -> Result<
        Projection<invalid::Action, umpire_dsl::Outcome, umpire_runtime::NoTarget>,
        protocol::Refusal,
    > {
        Ok(match observation.value.as_slice() {
            [0] if state.value == 0 => Projection::Current,
            [1] => Projection::steps([vec![transition(invalid::User::Bad)]]),
            _ => Projection::Candidates(Vec::new()),
        })
    }
}

impl umpire_runtime::capability::Evidence<umpire_runtime::evidence::Rpc> for Realization {
    type World = invalid::World;
    type View = umpire_runtime::capability::NoEvidence;
}

impl umpire_runtime::capability::ReplyPolicy<invalid::World> for Realization {
    fn answer(
        &self,
        _world: &invalid::World,
        callback: &Any,
        _: &invalid::Model,
    ) -> Result<AnswerPlan<invalid::Action>, protocol::Refusal> {
        let delivery = if callback.value == [1] { invalid::User::Bad } else { invalid::User::Good };
        Ok(AnswerPlan {
            delivery: vec![transition(delivery)],
            response_action: invalid::User::Bad.into(),
            reply: payload(42),
        })
    }
}

impl Realize<invalid::World> for Realization {}

fn payload(value: u8) -> Any {
    Any { type_url: "test/rejecting".into(), value: vec![value] }
}

#[test]
fn invalid_rejections_refuse_explore_replay_expected_actions_and_answer_without_advancing() {
    for (method, request, before) in [
        (1, protocol::ExploreRequest { budget: 3 }.encode_to_vec(), 0),
        (4, protocol::JudgeRequest { observation: Some(payload(1)) }.encode_to_vec(), 0),
        (4, protocol::JudgeRequest { observation: Some(payload(2)) }.encode_to_vec(), 0),
        (5, protocol::AnswerRequest { callback: Some(payload(1)) }.encode_to_vec(), 0),
        (5, protocol::AnswerRequest { callback: Some(payload(2)) }.encode_to_vec(), 1),
    ] {
        let mut runtime = Runtime::new(invalid::world(), Realization);
        let response = runtime.dispatch(method, &request);
        let encoded = response.encode_to_vec();
        let decoded = protocol::Response::decode(encoded.as_slice())
            .unwrap_or_else(|error| panic!("response failed to decode: {error}"));
        let Some(protocol::response::Outcome::Refusal(refusal)) = decoded.outcome else {
            panic!("expected invalid Model refusal for method {method}");
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::MalformedRequest);
        assert_eq!(
            refusal.message,
            format!(
                "invalid action User::Bad from Model {{ value: {before} }}: rejecting step changed state"
            )
        );
        let response = runtime
            .dispatch(4, &protocol::JudgeRequest { observation: Some(payload(0)) }.encode_to_vec());
        let Some(protocol::response::Outcome::Judge(judgment)) = response.outcome else {
            panic!("current observation failed after refusal");
        };
        assert!(matches!(
            judgment.judgment,
            Some(protocol::judge_response::Judgment::Explained(_))
        ));
    }
}
