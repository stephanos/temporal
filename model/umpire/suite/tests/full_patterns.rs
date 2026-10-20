//! Test-space admission, exact retained witnesses and finite seeded experiments.

use umpire_dsl::{Action, State};
use umpire_suite::{Cases, Refusal, Run, Space, cover, find, path, sample};

#[umpire_dsl::machine]
#[allow(unreachable_pub)]
mod counter {
    use super::{Action, State};

    #[derive(State)]
    pub struct Model {
        pub count: u8,
    }

    #[derive(Action, Copy)]
    pub enum User {
        Loop,
        Advance,
        Disabled,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Loop => umpire_dsl::stay!(),
                User::Advance if self.count < 2 => self.count += 1,
                _ => {}
            }
        }
    }
}

use counter::{Model, User};

#[test]
fn cases_mismatch_and_alternative_budget_precede_model_search() {
    let space = Space::from([Model { count: 0 }]).budget(0);
    let error = cover(&space, Cases(1), path![(User::Loop | User::Advance)])
        .err()
        .unwrap_or_else(|| panic!("mismatched cases passed"));
    assert!(matches!(error, Refusal::Cases { actual: 2, expected: 1, .. }));
    let message = error.to_string();
    assert!(message.contains("User::Loop") && message.contains("User::Advance"));
    let space = space.alternative_budget(1);
    assert!(matches!(
        cover(&space, Cases(2), path![(User::Loop | User::Advance)]),
        Err(Refusal::Pattern(_))
    ));
    assert!(matches!(
        find(&space.alternative_budget(0), path![true, ..=0, true]),
        Err(Refusal::Pattern(_))
    ));
}

#[test]
fn model_state_budget_does_not_spend_alternative_capacity() -> Result<(), Refusal> {
    let space = Space::from([Model { count: 2 }]).budget(1).alternative_budget(2);
    let covered = cover(&space, Cases(2), path![(User::Loop | User::Loop)])?;
    assert_eq!(covered.len(), 2);
    assert!(covered.all(|run| run.state().count == 2));
    assert_eq!(covered.runs()[0].path().steps.len(), 1);
    Ok(())
}

#[test]
fn full_grammar_find_and_along_preserve_state_inference() -> Result<(), Refusal> {
    let space = Space::from([Model { count: 0 }]);
    let found = find(&space, path![(User::Disabled | User::Advance), |s| s.count == 1])?;
    assert_eq!(found.steps.len(), 1);
    let along = Run::along(path![Model { count: 0 }, [User::Advance; 2], |s| s.count == 2])?;
    assert_eq!(along.path().steps.len(), 2);
    let error = Run::along_with_budget(path![Model { count: 0 }, (User::Loop | User::Advance)], 0)
        .err()
        .unwrap_or_else(|| panic!("story alternatives accepted"));
    assert!(matches!(error, Refusal::StoryAlternatives { count: 2 }));
    Ok(())
}

#[test]
fn cover_refuses_empty_unwitnessed_and_cut_spaces() -> Result<(), Refusal> {
    let space = Space::from([Model { count: 0 }]);
    assert!(matches!(cover(&space, Cases(0), path![[User::Loop; 2..=1]]), Err(Refusal::Empty)));
    let empty_sequence = cover(&space, Cases(1), path![[User::Loop; 0]])?;
    assert_eq!(empty_sequence.runs()[0].path().steps.len(), 0);
    assert!(matches!(
        cover(&space, Cases(1), path![Model { count: 0 }, User::Disabled]),
        Err(Refusal::Alternative { index: 0, .. })
    ));
    assert!(matches!(
        cover(&space.clone().budget(1), Cases(1), path![Model { count: 0 }, User::Loop]),
        Err(Refusal::Cut { .. })
    ));
    Ok(())
}

#[test]
fn sampling_walks_admitted_loops_reproducibly_and_reports_its_bound() -> Result<(), Refusal> {
    let space = Space::from([Model { count: 2 }]).budget(1).walk_bound(8);
    let a = sample(&space, path![Model { count: 2 }, .., User::Loop], 20, 42)?;
    let b = sample(&space, path![Model { count: 2 }, .., User::Loop], 20, 42)?;
    assert_eq!(a.expansion(), b.expansion());
    assert!(a.runs().iter().any(|run| run.path().steps.len() > 1));
    assert!(a.runs().iter().all(|run| run.path().steps.len() <= 8));
    assert!(a.expansion().contains("seed 42") && a.expansion().contains("walk bound 8"));
    let short = Space::from([Model { count: 0 }]).walk_bound(1);
    assert!(matches!(
        sample(&short, path![[User::Advance; 2]], 1, 0),
        Err(Refusal::Alternative { .. })
    ));
    Ok(())
}
