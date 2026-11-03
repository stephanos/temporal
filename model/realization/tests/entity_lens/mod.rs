//! Frozen pre-extraction facade against the shared lifecycle on the same actual inputs.

use prost::Message;
use umpire_runtime::{
    AnyMessage as _, Dispatch, Guest,
    prost_types::Any,
    protocol::{self, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal_api::temporal::api::workflowservice::v1::{
    DescribeActivityExecutionRequest, StartActivityExecutionResponse,
};

#[path = "baseline.rs"]
mod baseline;
#[path = "../common/mod.rs"]
mod common;

struct Pair {
    before: Guest<baseline::LiveRuntime>,
    after: Guest<super::LiveRuntime>,
}
impl Pair {
    fn new() -> Self {
        Self {
            before: Guest::new(baseline::LiveRuntime::new()),
            after: Guest::new(super::runtime()),
        }
    }
    fn call(&mut self, method: u32, request: &[u8], publish: bool) -> protocol::Response {
        let mut before = None;
        let mut after = None;
        let before_published = self.before.dispatch_to(method, request, |bytes| {
            before =
                Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
            publish
        });
        let after_published = self.after.dispatch_to(method, request, |bytes| {
            after =
                Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
            publish
        });
        assert_eq!(before_published, after_published);
        assert_eq!(before, after, "method {method}, published {publish}");
        after.unwrap_or_else(|| panic!("response must reach the publisher"))
    }
    fn judge(&mut self, observation: &Observation, publish: bool) -> protocol::Response {
        self.call(
            4,
            &protocol::JudgeRequest { observation: Some(Any::pack(observation)) }.encode_to_vec(),
            publish,
        )
    }
    fn rows(&mut self) {
        let response = self.call(1, &protocol::ExploreRequest { budget: 64 }.encode_to_vec(), true);
        if let Some(Outcome::Explore(exploration)) = response.outcome {
            for state_id in 0..exploration.reached {
                self.call(2, &protocol::RowsRequest { state_id }.encode_to_vec(), true);
            }
        }
    }
}

#[test]
fn entity_lens_equivalence_admission_freeze_replay_reset_and_failed_publication() {
    for attempts in [1, 2, 3, 4, 0] {
        for publication in [false, true] {
            let mut pair = Pair::new();
            pair.judge(&common::describe(1, 1, 1), true);
            pair.judge(&common::start(attempts), publication);
            pair.rows();
            pair.judge(&common::start(attempts), true);
            pair.judge(&common::describe(1, 1, 2), true);
            pair.judge(&common::start(attempts), true);
            let mut replay = common::start(attempts);
            replay.result =
                Some(ResultValue::Response(Any::pack(&StartActivityExecutionResponse {
                    run_id: common::RUN.into(),
                    started: false,
                    ..Default::default()
                })));
            pair.judge(&replay, true);
            pair.call(3, &[], publication);
            pair.rows();
            pair.judge(&common::start(attempts), true);
            pair.call(3, &[], true);
            pair.judge(&common::start(attempts), true);
        }
    }
}

#[test]
fn entity_lens_equivalence_configured_programs_pending_identity_reads_and_hold() {
    for seed in 0..6 {
        let mut pair = Pair::new();
        let reset =
            protocol::ResetRequest { configuration: common::start(3).request }.encode_to_vec();
        pair.call(3, &reset, false);
        pair.call(3, &reset, true);
        pair.call(5, &protocol::AnswerRequest::default().encode_to_vec(), true);
        let response = pair.call(6, &protocol::ProposeRequest { seed }.encode_to_vec(), true);
        let Some(Outcome::Propose(proposal)) = response.outcome else {
            panic!("bootstrap proposal")
        };
        pair.call(6, &protocol::ProposeRequest { seed }.encode_to_vec(), true);
        pair.judge(&common::start(3), false);
        let mut start = common::start(3);
        start.request = proposal.request;
        pair.judge(&start, false);
        pair.judge(&start, true);
        for next in 0..4 {
            pair.call(6, &protocol::ProposeRequest { seed: next }.encode_to_vec(), false);
            pair.call(6, &protocol::ProposeRequest { seed: next }.encode_to_vec(), true);
            let mut crossed = common::describe(1, 1, 1);
            let mut request: DescribeActivityExecutionRequest = crossed
                .request
                .as_ref()
                .unwrap_or_else(|| panic!("request"))
                .unpack()
                .unwrap_or_else(|error| panic!("{error:?}"));
            request.run_id = "33333333-3333-4333-8333-333333333333".into();
            crossed.request = Some(Any::pack(&request));
            pair.judge(&crossed, true);
            pair.judge(&common::describe(1, 1, 1), true);
        }
        pair.judge(&common::describe(1, 2, 0), true);
        pair.call(6, &protocol::ProposeRequest { seed: 0 }.encode_to_vec(), true);
        pair.call(3, &reset, false);
        pair.judge(&start, true);
        pair.call(3, &[], true);
        pair.call(6, &protocol::ProposeRequest { seed }.encode_to_vec(), true);
    }
}

#[test]
fn entity_lens_equivalence_malformed_and_crossed_actual_evidence_precedence() {
    let mut pair = Pair::new();
    for method in [3, 4, 6, u32::MAX] {
        pair.call(method, &[255], true);
    }
    pair.judge(&common::start(1), true);
    let mut crossed = common::start(1);
    crossed.request = Some(Any::default());
    pair.judge(&crossed, true);
    let mut crossed = common::start(1);
    crossed.result = Some(ResultValue::Response(Any::default()));
    pair.judge(&crossed, true);
    let mut crossed = common::start(1);
    crossed.result = Some(ResultValue::Response(Any::pack(&StartActivityExecutionResponse {
        run_id: "33333333-3333-4333-8333-333333333333".into(),
        started: true,
        ..Default::default()
    })));
    pair.judge(&crossed, true);
    pair.judge(&common::describe(1, 1, 1), true);
}

#[test]
fn entity_lens_equivalence_publisher_panic_does_not_commit() {
    fn panic_publish<D: Dispatch>(guest: &mut Guest<D>, request: &[u8]) {
        let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            guest.dispatch_to(4, request, |_| panic!("publisher refused"))
        }));
        assert!(result.is_err());
    }
    for attempts in [1, 4] {
        let mut pair = Pair::new();
        let bytes =
            protocol::JudgeRequest { observation: Some(Any::pack(&common::start(attempts))) }
                .encode_to_vec();
        panic_publish(&mut pair.before, &bytes);
        panic_publish(&mut pair.after, &bytes);
        pair.judge(&common::describe(1, 1, 1), true);
        pair.judge(&common::start(attempts), true);
    }
}

#[test]
fn entity_lens_equivalence_pending_cancellation_reconciliation_uses_actual_pair() {
    use umpire_temporal_api::{
        temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse,
        umpire::temporal::v1::{ReconciledObservation, TemporalInstruction},
    };
    let mut pair = Pair::new();
    pair.call(
        3,
        &protocol::ResetRequest { configuration: common::start(3).request }.encode_to_vec(),
        true,
    );
    let response = pair.call(6, &protocol::ProposeRequest { seed: 0 }.encode_to_vec(), true);
    let Some(Outcome::Propose(bootstrap)) = response.outcome else { panic!("bootstrap") };
    let mut start = common::start(3);
    start.request = bootstrap.request;
    pair.judge(&start, true);
    let response = pair.call(6, &protocol::ProposeRequest { seed: 2 }.encode_to_vec(), true);
    let Some(Outcome::Propose(proposal)) = response.outcome else { panic!("hold proposal") };
    let instruction: TemporalInstruction = proposal
        .request
        .unwrap_or_else(|| panic!("instruction"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert!(instruction.hold_worker_polls);
    let acknowledged = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: instruction.request,
        result: Some(ResultValue::Response(Any::pack(&RequestCancelActivityExecutionResponse {}))),
    };
    let actual = Any::pack(&ReconciledObservation {
        acknowledged: Some(Any::pack(&acknowledged)),
        current: Some(Any::pack(&common::describe(1, 1, 1))),
    });
    let bytes = protocol::JudgeRequest { observation: Some(actual) }.encode_to_vec();
    pair.call(4, &bytes, false);
    pair.call(4, &bytes, true);
    pair.call(3, &[], true);
    pair.call(4, &bytes, true);
}

#[test]
fn entity_lens_equivalence_oversized_publication_retains_admission_and_freeze() {
    use std::{cell::Cell, rc::Rc};
    struct Oversized<D> {
        inner: D,
        refuse: Rc<Cell<bool>>,
    }
    impl<D: Dispatch> Dispatch for Oversized<D> {
        type Transaction = D::Transaction;
        fn prepare(
            &self,
            method: u32,
            bytes: &[u8],
        ) -> (protocol::Response, Option<Self::Transaction>) {
            let (mut response, transaction) = self.inner.prepare(method, bytes);
            if self.refuse.get() {
                response.outcome =
                    Some(Outcome::Rows(protocol::RowsResponse { rows: vec!["x".repeat(1 << 20)] }));
            }
            (response, transaction)
        }
        fn commit(&mut self, transaction: Self::Transaction) {
            self.inner.commit(transaction);
        }
    }
    fn publish<D: Dispatch>(guest: &mut Guest<D>, observation: &Observation) -> protocol::Response {
        let bytes =
            protocol::JudgeRequest { observation: Some(Any::pack(observation)) }.encode_to_vec();
        let mut response = None;
        assert!(guest.dispatch_to(4, &bytes, |bytes| {
            response =
                Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
            true
        }));
        response.unwrap_or_else(|| panic!("response"))
    }
    for attempts in [1, 4] {
        let refuse = Rc::new(Cell::new(true));
        let mut before =
            Guest::new(Oversized { inner: baseline::LiveRuntime::new(), refuse: refuse.clone() });
        let mut after = Guest::new(Oversized { inner: super::runtime(), refuse: refuse.clone() });
        let bytes =
            protocol::JudgeRequest { observation: Some(Any::pack(&common::start(attempts))) }
                .encode_to_vec();
        assert!(!before.dispatch_to(4, &bytes, |_| panic!("oversized response reached publisher")));
        assert!(!after.dispatch_to(4, &bytes, |_| panic!("oversized response reached publisher")));
        refuse.set(false);
        assert_eq!(
            publish(&mut before, &common::describe(1, 1, 1)),
            publish(&mut after, &common::describe(1, 1, 1))
        );
        assert_eq!(
            publish(&mut before, &common::start(attempts)),
            publish(&mut after, &common::start(attempts))
        );
        assert_eq!(
            publish(&mut before, &common::start(attempts)),
            publish(&mut after, &common::start(attempts))
        );
    }
}
