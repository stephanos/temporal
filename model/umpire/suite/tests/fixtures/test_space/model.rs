use std::fmt::Write;

use umpire_suite::{Cases, Run, Space, cover, find, path, sample};

#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, path, reject, Rejection};

    #[derive(State)]
    pub struct Model {
        pub count: u8,
        pub closed: bool,
    }

    #[derive(Action, Copy)]
    pub enum User {
        Add {
            #[inputs(0..=1)]
            amount: u8,
        },
        Close,
        Missing,
    }

    #[derive(Action, Copy)]
    #[fault]
    pub enum Fault {
        Crash,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Add { amount } if !self.closed && self.count + amount <= 2 => {
                    self.count += amount;
                    umpire_dsl::stay!();
                }
                User::Close if !self.closed => self.closed = true,
                User::Missing => reject!(Rejection::NotFound),
                _ => {}
            }
        }

        #[on(Fault)]
        fn fault(&mut self, action: Fault) {
            match action {
                Fault::Crash if self.count > 0 && !self.closed => self.closed = true,
                Fault::Crash => {}
            }
        }

        #[invariant]
        fn bounded(&self) {
            umpire_dsl::check!(self.count <= 2);
        }

        #[postcondition(after = User::Missing)]
        fn missing_preserves(&self, before: &Self) {
            umpire_dsl::check!(self == before);
        }

        #[progress]
        fn closes(&self) -> LeadsTo {
            path![self.count > 0, .., self.closed]
        }

        #[progress]
        fn never_triggered(&self) -> LeadsTo {
            path![self.count > 2, .., self.closed]
        }
    }
}

use model::{Fault, Model, User};

pub(crate) fn run() -> umpire_testing::RunResult {
    let space = Space::from([Model { count: 0, closed: false }]);
    let covered = cover(&space, Cases(4), path![Model { count: 0, closed: false }, [User::Add { amount: 1 }; 0..=1], .., (User::Close | Fault::Crash)])?;
    let sampled = sample(&space.clone().walk_bound(4), path![Model { count: 0, closed: false }, .., User::Close], 3, 42)?;
    let mut files = umpire_testing::Files::from([
        ("expansion/closings.txt".into(), covered.expansion().into()),
        ("expansion/sample.txt".into(), sampled.expansion().into()),
        ("coverage.txt".into(), format!("cover\n{}\nsample (finite experiment)\n{}", covered.coverage(), sampled.coverage())),
    ]);
    let mut messages = String::new();
    macro_rules! refused {
        ($name:literal, $answer:expr) => {
            writeln!(messages, "{}: {}", $name, $answer.err().ok_or("expected refusal")?)?;
        };
    }
    refused!("mismatch first", cover(&space.clone().budget(0), Cases(3), path![Model { count: 0, closed: false }, (User::Close | Fault::Crash)]));
    refused!("impossible alternative", cover(&space, Cases(1), path![Model { count: 0, closed: false }, Fault::Crash]));
    refused!("alternative admission", cover(&space.clone().alternative_budget(1), Cases(2), path![Model { count: 0, closed: false }, (User::Close | Fault::Crash)]));
    refused!("permutation admission", cover(&space, Cases(5040), path![Model { count: 0, closed: false }, unordered![User::Close, User::Close, User::Close, User::Close, User::Close, User::Close, User::Close]]));
    refused!("cut coverage", cover(&space.clone().budget(1), Cases(1), path![Model { count: 0, closed: false }, User::Close]));
    refused!("story alternatives", Run::along_with_budget(path![Model { count: 0, closed: false }, (User::Close | Fault::Crash)], 0));
    refused!("state predicate panic", find(&space, path![Model { count: 0, closed: false }, [|_: &Model| panic!("state defect"); 1]]));
    let outcome_panic = umpire_dsl::Path::<Model, model::Action, umpire_dsl::Outcome>::pattern(umpire_dsl::Pattern::Sequence(vec![
        umpire_dsl::Pattern::State(std::rc::Rc::new(umpire_dsl::Waypoint::Is(Model { count: 0, closed: false }))),
        umpire_dsl::Pattern::Action(umpire_dsl::ActionDomain::new(Ok(1), |_| model::Action::from(User::Close)), Some(std::rc::Rc::new(|_| panic!("outcome defect"))), None),
    ]));
    refused!("outcome predicate panic", cover(&space, Cases(1), outcome_panic));
    refused!("empty alternative set", cover(&space, Cases(0), path![Model { count: 0, closed: false }, [User::Close; 2..=1]]));
    refused!("sample walk bound", sample(&space.clone().walk_bound(0), path![Model { count: 0, closed: false }, User::Close], 1, 42));
    files.insert("messages.txt".into(), messages);
    Ok(files)
}
