use super::*;
use std::{cell::Cell, rc::Rc};

struct Publication<D> {
    inner: D,
    oversized: Rc<Cell<bool>>,
}
impl<D: Dispatch> Dispatch for Publication<D> {
    type Transaction = D::Transaction;
    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>) {
        let (mut response, transaction) = self.inner.prepare(method, request);
        if self.oversized.get() {
            response.outcome =
                Some(Outcome::Rows(protocol::RowsResponse { rows: vec!["x".repeat(1 << 20)] }));
        }
        (response, transaction)
    }
    fn commit(&mut self, transaction: Self::Transaction) {
        self.inner.commit(transaction);
    }
}

#[test]
fn success_freeze_and_composed_reset_roll_back_false_panic_and_oversize() {
    for operation in ["success", "freeze", "reset live", "reset freeze"] {
        for publication in ["false", "panic", "oversize"] {
            let oversized = Rc::new(Cell::new(false));
            let mut guest = Guest::new(Publication {
                inner: annotation_runtime(),
                oversized: oversized.clone(),
            });
            let captured = start(if operation.ends_with("freeze") { 4 } else { 1 });
            if operation == "reset live" {
                explained(judge(&mut guest, &captured));
                advanced(judge(&mut guest, &describe(1, 1, 2)), &["Worker::Poll(1)"]);
            } else if operation == "reset freeze" {
                refused(judge(&mut guest, &captured), protocol::RefusalCode::Unsupported);
            }
            let Some(Outcome::Explore(exploration)) =
                call(&mut guest, 1, &protocol::ExploreRequest { budget: 64 }).outcome
            else {
                panic!("explore")
            };
            let old_rows = all_rows(&mut guest, exploration.reached);
            let (method, request) = if operation.starts_with("reset") {
                (3, Vec::new())
            } else {
                (
                    4,
                    protocol::JudgeRequest { observation: Some(Any::pack(&captured)) }
                        .encode_to_vec(),
                )
            };
            oversized.set(publication == "oversize");
            let failed = std::panic::catch_unwind(AssertUnwindSafe(|| {
                guest.dispatch_to(method, &request, |_| {
                    assert_ne!(publication, "oversize", "oversized response reached publisher");
                    assert_ne!(publication, "panic", "publication failed");
                    false
                })
            }));
            assert_eq!(failed.is_err(), publication == "panic");
            if let Ok(published) = failed {
                assert!(!published);
            }
            oversized.set(false);
            assert_eq!(all_rows(&mut guest, exploration.reached), old_rows);
            match operation {
                "reset live" => {
                    explained(judge(&mut guest, &captured));
                    explained(judge(&mut guest, &describe(1, 1, 2)));
                    call(&mut guest, 3, &protocol::ResetRequest::default());
                    explained(judge(&mut guest, &captured));
                    explained(judge(&mut guest, &describe(1, 1, 1)));
                    assert_eq!(all_rows(&mut guest, exploration.reached), old_rows);
                }
                "reset freeze" => {
                    refused(judge(&mut guest, &captured), protocol::RefusalCode::Unsupported);
                    call(&mut guest, 3, &protocol::ResetRequest::default());
                    refused(judge(&mut guest, &captured), protocol::RefusalCode::State);
                    refused(judge(&mut guest, &start(1)), protocol::RefusalCode::State);
                    refused(judge(&mut guest, &describe(1, 1, 1)), protocol::RefusalCode::State);
                    assert_eq!(all_rows(&mut guest, exploration.reached), old_rows);
                }
                "success" => {
                    refused(judge(&mut guest, &describe(1, 1, 1)), protocol::RefusalCode::State);
                    explained(judge(&mut guest, &captured));
                    explained(judge(&mut guest, &captured));
                    refused(
                        call(&mut guest, 2, &protocol::RowsRequest { state_id: 0 }),
                        protocol::RefusalCode::NotExplored,
                    );
                    let Some(Outcome::Explore(result)) =
                        call(&mut guest, 1, &protocol::ExploreRequest { budget: 64 }).outcome
                    else {
                        panic!("explore")
                    };
                    assert_eq!(result.reached, 5);
                }
                "freeze" => {
                    refused(judge(&mut guest, &describe(1, 1, 1)), protocol::RefusalCode::State);
                    let first =
                        refused(judge(&mut guest, &captured), protocol::RefusalCode::Unsupported);
                    assert_eq!(
                        first,
                        refused(judge(&mut guest, &captured), protocol::RefusalCode::Unsupported)
                    );
                    assert_eq!(all_rows(&mut guest, exploration.reached), old_rows);
                }
                _ => unreachable!(),
            }
        }
    }
}

fn all_rows(guest: &mut Guest<impl Dispatch>, reached: u32) -> Vec<protocol::Response> {
    (0..reached).map(|state_id| call(guest, 2, &protocol::RowsRequest { state_id })).collect()
}
