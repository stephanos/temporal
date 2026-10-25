//! Typed calls and response assertions for native guest tests.

use prost::Message;
use prost_types::Any;

use crate::{
    Dispatch, Guest,
    protocol::{self, response::Outcome},
};

/// Successful publication through a guest's native transport.
pub trait GuestExt {
    /// Encodes a typed request, publishes and decodes the complete response.
    fn call(&mut self, method: protocol::Method, request: &impl Message) -> protocol::Response;
    /// Judges one packed observation.
    fn judge(&mut self, observation: Any) -> protocol::Response {
        self.call(
            protocol::Method::Judge,
            &protocol::JudgeRequest { observation: Some(observation) },
        )
    }
    /// Resets the run with optional packed configuration.
    fn reset(&mut self, configuration: Option<Any>) -> protocol::Response {
        self.call(protocol::Method::Reset, &protocol::ResetRequest { configuration })
    }
    /// Requires a proposal for the selected seed.
    fn propose(&mut self, seed: u64) -> protocol::ProposeResponse {
        let response = self.call(protocol::Method::Propose, &protocol::ProposeRequest { seed });
        let Some(Outcome::Propose(proposal)) = response.outcome else {
            panic!("expected proposal: {response:?}")
        };
        proposal
    }
}

impl<D: Dispatch> GuestExt for Guest<D> {
    fn call(&mut self, method: protocol::Method, request: &impl Message) -> protocol::Response {
        let mut response = None;
        assert!(self.dispatch_to(method as u32, &request.encode_to_vec(), |bytes| {
            response = Some(
                protocol::Response::decode(bytes)
                    .unwrap_or_else(|error| panic!("invalid guest response: {error}")),
            );
            true
        }));
        response.unwrap_or_else(|| panic!("guest did not publish a response"))
    }
}

/// Assert the response kind while retaining all diagnostic and explanation data.
pub trait ResponseExt {
    /// Requires a refusal with the expected code, retaining its message.
    fn refusal(self, code: protocol::RefusalCode) -> protocol::Refusal;
    /// Requires an explanation, retaining both steps and outcomes.
    fn explanation(self) -> protocol::Explanation;
    /// Requires an unexplained judgment and returns its expected actions.
    fn unexplained(self) -> Vec<String>;
}

impl ResponseExt for protocol::Response {
    fn refusal(self, code: protocol::RefusalCode) -> protocol::Refusal {
        let Some(Outcome::Refusal(refusal)) = self.outcome else {
            panic!("expected refusal: {self:?}")
        };
        assert_eq!(refusal.code, code as i32);
        refusal
    }
    fn explanation(self) -> protocol::Explanation {
        let Some(Outcome::Judge(judgment)) = self.outcome else {
            panic!("expected Judge: {self:?}")
        };
        let Some(protocol::judge_response::Judgment::Explained(explanation)) = judgment.judgment
        else {
            panic!("expected explanation: {judgment:?}")
        };
        explanation
    }
    fn unexplained(self) -> Vec<String> {
        let Some(Outcome::Judge(judgment)) = self.outcome else {
            panic!("expected Judge: {self:?}")
        };
        let Some(protocol::judge_response::Judgment::Unexplained(unexplained)) = judgment.judgment
        else {
            panic!("expected unexplained judgment: {judgment:?}")
        };
        unexplained.expected_actions
    }
}
