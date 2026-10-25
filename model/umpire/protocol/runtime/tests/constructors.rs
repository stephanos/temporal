//! Runtime authoring constructors retain the declared path and response.

use umpire_runtime::{AnswerPlan, Candidate, Projection, Transition, prost_types::Any};

#[test]
fn constructors_preserve_actions_choices_paths_and_reply() {
    let transition = Transition::from(7_u8);
    assert_eq!(transition.action, 7);
    assert_eq!(transition.choice, 0);
    let Projection::Candidates(candidates) = Projection::<u8>::step(7) else {
        panic!("expected one candidate")
    };
    assert_eq!(candidates.len(), 1);
    assert_eq!(candidates[0].path[0].action, 7);
    assert_eq!(candidates[0].path[0].choice, 0);
    assert_eq!(candidates[0].outcome, None);
    let Projection::Candidates(candidates) = Projection::<u8>::steps([
        vec![Transition { action: 8, choice: 2 }],
        vec![9.into(), 10.into()],
    ]) else {
        panic!("expected candidate paths")
    };
    assert_eq!(candidates.len(), 2);
    assert_eq!(candidates[0].path[0].choice, 2);
    assert_eq!(candidates[1].path[1].action, 10);
    assert!(candidates.iter().all(|candidate| candidate.outcome.is_none()));
    let candidate =
        Candidate { path: vec![7.into()], outcome: Some(umpire_dsl::Outcome::Accepted) };
    assert_eq!(candidate.outcome, Some(umpire_dsl::Outcome::Accepted));
    let reply = Any { type_url: "test/reply".into(), value: vec![42] };
    let plan = AnswerPlan::new(vec![transition], 8, reply.clone());
    assert_eq!(plan.delivery[0].action, 7);
    assert_eq!(plan.response_action, 8);
    assert_eq!(plan.reply, reply);
}
