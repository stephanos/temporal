#[path = "countdown.rs"]
mod countdown;
use countdown::{Remaining, Ticks};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
enum Expired { Global, Attempt, Heartbeat, Waiting }
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
struct Budgets {
    waiting: Option<Remaining<3>>,
    attempt: Option<Remaining<3>>,
    global: Option<Remaining<3>>,
    heartbeat: Option<Remaining<3>>,
}
impl Budgets {
    fn tick(&mut self) -> Option<Expired> {
        self.waiting = countdown::tick(self.waiting);
        self.attempt = countdown::tick(self.attempt);
        self.global = countdown::tick(self.global);
        self.heartbeat = countdown::tick(self.heartbeat);
        if countdown::expired(self.global) { Some(Expired::Global) }
        else if countdown::expired(self.attempt) { Some(Expired::Attempt) }
        else if countdown::expired(self.heartbeat) { Some(Expired::Heartbeat) }
        else if countdown::expired(self.waiting) { Some(Expired::Waiting) }
        else { None }
    }
    fn retry(&mut self, waiting: Option<Ticks<3>>) {
        self.waiting = countdown::arm(waiting);
        self.attempt = None;
        self.heartbeat = None;
    }
}
#[test]
fn global_wins_simultaneous_expiry_and_retry_does_not_replenish_it() {
    let mut budgets = Budgets { waiting: Some(Remaining(1)), attempt: Some(Remaining(1)),
        global: Some(Remaining(1)), heartbeat: Some(Remaining(1)) };
    assert_eq!(budgets.tick(), Some(Expired::Global));
    let global = budgets.global;
    budgets.retry(Some(Ticks(3)));
    assert_eq!(budgets.global, global);
    assert_eq!(budgets.waiting, Some(Remaining(3)));
    assert_eq!(budgets.attempt, None);
    assert_eq!(budgets.heartbeat, None);
}
