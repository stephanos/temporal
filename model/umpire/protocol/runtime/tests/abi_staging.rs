//! Publication is the commit boundary for native guest transports.

use std::{cell::Cell, panic::AssertUnwindSafe, rc::Rc};

#[path = "../guests/stock/src/model.rs"]
mod stock;

use prost::Message;
use umpire_runtime::{
    Dispatch, Guest,
    protocol::{self, response::Outcome},
};

#[test]
fn unpublished_exploration_leaves_stock_guest_unexplored() {
    let mut guest = Guest::new(stock::model());
    let explore = protocol::ExploreRequest { budget: 1 << 20 }.encode_to_vec();
    assert!(!guest.dispatch_to(1, &explore, |bytes| {
        assert!(matches!(decode(bytes).outcome, Some(Outcome::Explore(_))));
        false
    }));
    let rows = protocol::RowsRequest { state_id: 0 }.encode_to_vec();
    assert!(guest.dispatch_to(2, &rows, |bytes| {
        let Some(Outcome::Refusal(refusal)) = decode(bytes).outcome else {
            panic!("unpublished exploration must not install rows")
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::NotExplored);
        true
    }));
}

struct Dispatcher {
    commits: Rc<Cell<usize>>,
    response_size: usize,
}

struct Transaction;

impl Dispatch for Dispatcher {
    type Transaction = Transaction;

    fn prepare(&self, _: u32, _: &[u8]) -> (protocol::Response, Option<Transaction>) {
        (
            protocol::Response {
                outcome: Some(Outcome::Rows(protocol::RowsResponse {
                    rows: vec!["x".repeat(self.response_size)],
                })),
            },
            Some(Transaction),
        )
    }

    fn commit(&mut self, _: Transaction) {
        self.commits.set(self.commits.get() + 1);
    }
}

#[test]
fn oversized_response_never_publishes_or_commits() {
    let commits = Rc::new(Cell::new(0));
    let mut guest = Guest::new(Dispatcher { commits: commits.clone(), response_size: 1 << 20 });
    assert!(!guest.dispatch_to(1, &[], |_| panic!("oversized response reached publisher")));
    assert_eq!(commits.get(), 0);
}

#[test]
fn panicking_publisher_leaves_transaction_uncommitted() {
    let commits = Rc::new(Cell::new(0));
    let mut guest = Guest::new(Dispatcher { commits: commits.clone(), response_size: 1 });
    assert!(
        std::panic::catch_unwind(AssertUnwindSafe(|| {
            guest.dispatch_to(1, &[], |_| panic!("publication failed"))
        }))
        .is_err()
    );
    assert_eq!(commits.get(), 0);
    assert!(guest.dispatch_to(1, &[], |_| true));
    assert_eq!(commits.get(), 1);
}

fn decode(bytes: &[u8]) -> protocol::Response {
    protocol::Response::decode(bytes)
        .unwrap_or_else(|error| panic!("response must decode: {error}"))
}
