//! Computed complete-state starts and typed actor inference use the selected owned World.

#![forbid(unsafe_code)]

use umpire_dsl::{Machine, path};
use umpire_suite::{Cases, Refusal, Run, Space, cover, find, sample, verify_all};

#[umpire_dsl::machine]
mod correlated {
    use umpire_dsl::{Action, State, World, check};

    #[derive(World)]
    pub(super) struct World {
        first: u8,
        second: u8,
    }
    #[derive(State)]
    pub(super) struct Model {
        pub(super) left: u8,
        pub(super) right: u8,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Swap,
    }

    impl Model {
        #[starts]
        fn starts(w: &World) -> Vec<Self> {
            if w.first == 0 {
                return Vec::new();
            }
            vec![
                Self { left: w.first, right: w.second },
                Self { left: w.second, right: w.first },
                Self { left: w.first, right: w.second },
            ]
        }
        #[on(User)]
        fn user(&mut self, _: User, w: &World) {
            if self.left == w.first {
                std::mem::swap(&mut self.left, &mut self.right);
            }
        }
        #[invariant]
        fn correlated(&self, w: &World) {
            check!(
                (self.left, self.right) == (w.first, w.second)
                    || (self.left, self.right) == (w.second, w.first)
            );
        }
    }
}

#[test]
fn complete_starts_keep_correlation_first_occurrence_and_deduplication() -> Result<(), Refusal> {
    use correlated::{Model, User, World};

    let world = World::new(3, 4);
    assert_eq!(
        world.starts().iter().map(|s| (s.left, s.right)).collect::<Vec<_>>(),
        [(3, 4), (4, 3), (3, 4)]
    );
    let space = Space::<Model>::new(world);
    let report = verify_all(&space)?;
    assert_eq!(report.counts.starts, 2);
    assert_eq!(report.counts.reached, 2);
    let first = find(&space, path![true, ..=0, true])?;
    assert_eq!((first.start.left, first.start.right), (3, 4));
    let covered = cover(&space, Cases(1), path![User::Swap, |state| state.left == 4])?;
    assert_eq!((covered.runs()[0].state().left, covered.runs()[0].state().right), (4, 3));
    let sampled = sample(&space, path![User::Swap, |state| state.left == 4], 1, 7)?;
    assert_eq!(sampled.runs()[0].state().left, 4);
    let retained = Run::along_in(World::new(3, 4), path![Model { left: 3, right: 4 }, User::Swap])?;
    assert_eq!(retained.state().left, 4);
    let empty = Space::<Model>::new(World::new(0, 4));
    assert_eq!(verify_all(&empty)?.counts.starts, 0);
    assert_eq!(verify_all(&empty)?.counts.reached, 0);
    assert!(find(&empty, path![User::Swap, true]).is_err());
    Ok(())
}
