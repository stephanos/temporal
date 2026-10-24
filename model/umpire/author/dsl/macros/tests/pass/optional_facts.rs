#![allow(dead_code, private_interfaces)]
#[umpire_dsl::machine]
mod sample0 {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model {
        active: bool,
    }
    #[derive(Action)]
    pub enum User {
        Change,
        Empty,
        Disabled,
        Quiet,
    }
    #[derive(Action)]
    pub enum Ordinary {
        Step,
    }
    #[derive(Debug)]
    pub enum Fact {
        A,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {
            match action {
                User::Change => {
                    self.active = !self.active;
                    Some(Fact::A)
                }
                User::Empty => {
                    self.active = !self.active;
                    None
                }
                User::Disabled => Some(Fact::A),
                User::Quiet => None,
            }
        }
        #[on(Ordinary)]
        fn ordinary(&mut self, _: Ordinary) {
            self.active = !self.active;
        }
    }
    pub fn check() {
        use umpire_dsl::Machine;
        let world = world();
        let state = &world.starts()[0];
        assert_eq!(world.step(state, User::Change.into())[0].1.len(), 1);
        assert!(world.step(state, User::Empty.into())[0].1.is_empty());
        assert!(world.step(state, User::Disabled.into()).is_empty());
        assert!(world.step(state, User::Quiet.into()).is_empty());
        assert!(world.step(state, Ordinary::Step.into())[0].1.is_empty());
    }
}
#[umpire_dsl::machine]
mod sample1 {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {
        active: bool,
    }
    #[derive(Action)]
    enum User {
        Change,
        Empty,
        Disabled,
        Quiet,
        Stay,
    }
    #[derive(Action)]
    enum Ordinary {
        Step,
    }
    #[derive(Debug)]
    enum Fact {
        A,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> Option<Fact> {
            match action {
                User::Change => {
                    self.active = !self.active;
                    Some(Fact::A)
                }
                User::Empty => {
                    self.active = !self.active;
                    None
                }
                User::Disabled => Some(Fact::A),
                User::Quiet => None,
                User::Stay => {
                    umpire_dsl::stay!();
                    Some(Fact::A)
                }
            }
        }
        #[on(Ordinary)]
        fn ordinary(&mut self, _: Ordinary) {
            self.active = !self.active;
        }
    }
    pub fn check() {
        use umpire_dsl::Machine;
        let world = world();
        let state = &world.starts()[0];
        assert_eq!(world.step(state, User::Change.into())[0].1.len(), 1);
        assert!(world.step(state, User::Empty.into())[0].1.is_empty());
        assert!(world.step(state, User::Disabled.into()).is_empty());
        assert!(world.step(state, User::Quiet.into()).is_empty());
        assert!(world.step(state, Ordinary::Step.into())[0].1.is_empty());
        assert_eq!(world.step(state, User::Stay.into())[0].1.len(), 1);
    }
}
#[umpire_dsl::machine]
mod sample2 {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model {
        active: bool,
    }
    #[derive(Action)]
    pub enum User {
        Change,
        Empty,
        Disabled,
        Quiet,
        Rejected,
        Invalid,
    }
    #[derive(Action)]
    pub enum Ordinary {
        Step,
    }
    #[derive(Debug)]
    pub enum Fact {
        A,
    }
    #[derive(umpire_dsl::World)]
    pub struct World {}
    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> ::std::option::Option<Fact> {
            match action {
                User::Change => {
                    self.active = !self.active;
                    Some(Fact::A)
                }
                User::Empty => {
                    self.active = !self.active;
                    None
                }
                User::Disabled => Some(Fact::A),
                User::Quiet => None,

                User::Rejected => {
                    self.deny();
                    Some(Fact::A)
                }
                User::Invalid => {
                    self.deny();
                    self.active = !self.active;
                    Some(Fact::A)
                }
            }
        }
        #[on(Ordinary)]
        fn ordinary(&mut self, _: Ordinary) {
            self.active = !self.active;
        }
        #[effect]
        fn deny(&mut self) {
            umpire_dsl::reject!(umpire_dsl::Rejection::InvalidArgument);
        }
    }
    pub fn check() {
        use umpire_dsl::Machine;
        let world = World {};
        let state = &world.starts()[0];
        assert_eq!(world.step(state, User::Change.into())[0].1.len(), 1);
        assert!(world.step(state, User::Empty.into())[0].1.is_empty());
        assert!(world.step(state, User::Disabled.into()).is_empty());
        assert!(world.step(state, User::Quiet.into()).is_empty());
        assert!(world.step(state, Ordinary::Step.into())[0].1.is_empty());

        assert_eq!(
            world
                .checked_step(state, User::Rejected.into())
                .unwrap_or_else(|error| panic!("{error:?}"))[0]
                .1
                .len(),
            1
        );
        assert!(matches!(
            world.checked_step(state, User::Invalid.into()),
            Err(umpire_dsl::StepError::RejectedAfterMutation)
        ));
    }
}
#[umpire_dsl::machine]
mod sample3 {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model {
        active: bool,
    }
    #[derive(Action)]
    enum User {
        Change,
        Empty,
        Disabled,
        Quiet,
        Stay,
        Rejected,
        Invalid,
    }
    #[derive(Action)]
    enum Ordinary {
        Step,
    }
    #[derive(Debug)]
    enum Fact {
        A,
    }
    #[derive(umpire_dsl::World)]
    struct World {}
    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) -> core::option::Option<Fact> {
            match action {
                User::Change => {
                    self.active = !self.active;
                    Some(Fact::A)
                }
                User::Empty => {
                    self.active = !self.active;
                    None
                }
                User::Disabled => Some(Fact::A),
                User::Quiet => None,
                User::Stay => {
                    umpire_dsl::stay!();
                    Some(Fact::A)
                }
                User::Rejected => {
                    self.deny();
                    Some(Fact::A)
                }
                User::Invalid => {
                    self.deny();
                    self.active = !self.active;
                    Some(Fact::A)
                }
            }
        }
        #[on(Ordinary)]
        fn ordinary(&mut self, _: Ordinary) {
            self.active = !self.active;
        }
        #[effect]
        fn deny(&mut self) {
            umpire_dsl::reject!(umpire_dsl::Rejection::InvalidArgument);
        }
    }
    pub fn check() {
        use umpire_dsl::Machine;
        let world = World {};
        let state = &world.starts()[0];
        assert_eq!(world.step(state, User::Change.into())[0].1.len(), 1);
        assert!(world.step(state, User::Empty.into())[0].1.is_empty());
        assert!(world.step(state, User::Disabled.into()).is_empty());
        assert!(world.step(state, User::Quiet.into()).is_empty());
        assert!(world.step(state, Ordinary::Step.into())[0].1.is_empty());
        assert_eq!(world.step(state, User::Stay.into())[0].1.len(), 1);
        assert_eq!(
            world
                .checked_step(state, User::Rejected.into())
                .unwrap_or_else(|error| panic!("{error:?}"))[0]
                .1
                .len(),
            1
        );
        assert!(matches!(
            world.checked_step(state, User::Invalid.into()),
            Err(umpire_dsl::StepError::RejectedAfterMutation)
        ));
    }
}
fn main() {
    sample0::check();
    sample1::check();
    sample2::check();
    sample3::check();
}
