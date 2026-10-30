//! Native published responses for the production control guest's current-contract proof.
#![forbid(unsafe_code)]

#[path = "../../../../tests/support/control.rs"]
mod control;
#[path = "../../../../tests/support/mod.rs"]
#[allow(dead_code, reason = "Shared fixture has controls outside this transcript")]
mod support;

use prost::Message;
use std::{error::Error, fmt::Write, fs, path::PathBuf};
use umpire_runtime::{Guest, protocol};

struct Record {
    method: protocol::Method,
    request: Vec<u8>,
    response: Vec<u8>,
}

fn call(
    guest: &mut Guest<control::Control>,
    records: &mut Vec<Record>,
    method: protocol::Method,
    request: &impl Message,
) -> protocol::response::Outcome {
    let request = request.encode_to_vec();
    let mut response = Vec::new();
    assert!(guest.dispatch_to(method as u32, &request, |bytes| {
        response.extend_from_slice(bytes);
        true
    }));
    let decoded = protocol::Response::decode(response.as_slice())
        .unwrap_or_else(|error| panic!("{method:?}: {error}"));
    records.push(Record { method, request, response });
    decoded.outcome.unwrap_or_else(|| panic!("{method:?}: missing response"))
}

fn transcript(name: &str) -> Vec<Record> {
    use protocol::{Method, response::Outcome};

    if name == "frontier" {
        return frontier_transcript();
    }
    let mut guest = Guest::new(control::control());
    let mut records = Vec::new();
    let configured = name == "configured";
    let reset = call(
        &mut guest,
        &mut records,
        Method::Reset,
        &protocol::ResetRequest { configuration: configured.then(|| support::any(42)) },
    );
    let Outcome::Reset(reset) = reset else { panic!("Reset refused: {reset:?}") };
    assert_eq!(reset.proof_contract, protocol::ProofContract::WholePrefixV1 as i32);
    assert_eq!(reset.program, configured.then(|| support::any(42)));
    if configured {
        let proposal =
            call(&mut guest, &mut records, Method::Propose, &protocol::ProposeRequest { seed: 0 });
        let Outcome::Propose(proposal) = proposal else { panic!("Propose refused") };
        assert_eq!(proposal.kind, protocol::ProposalKind::Bootstrap as i32);
        assert_eq!(proposal.request, Some(support::any(0)));
        assert_eq!(proposal.action, "");
        judge(&mut guest, &mut records, 0, &[]);
    }
    if name == "callback" {
        let answer = call(
            &mut guest,
            &mut records,
            Method::Answer,
            &protocol::AnswerRequest { callback: Some(support::any(4)) },
        );
        let Outcome::Answer(answer) = answer else { panic!("Answer refused") };
        assert_eq!(answer.reply, Some(support::any(42)));
        judge(&mut guest, &mut records, 20, &["Deliver"]);
    }
    for (value, steps) in [
        (1, vec!["Deliver"]),
        (2, vec!["Deliver", "Respond"]),
        (3, vec!["Deliver", "Respond", "Finish"]),
    ] {
        judge(&mut guest, &mut records, value, &steps);
        judge(&mut guest, &mut records, 20, &steps);
    }
    records
}

fn frontier_transcript() -> Vec<Record> {
    use protocol::{Method, response::Outcome};
    let mut guest = Guest::new(control::control());
    let mut records = Vec::new();
    for (terminal, steps, outcomes) in [
        (
            support::frontier::GLOBAL,
            vec!["Tick", "Renew", "Tick", "Tick"],
            vec!["Advanced", "Renewed", "Advanced", "GlobalClosed"],
        ),
        (
            support::frontier::LOCAL,
            vec!["Renew", "Tick", "Tick"],
            vec!["Renewed", "Advanced", "PendingClosed"],
        ),
    ] {
        let reset = call(
            &mut guest,
            &mut records,
            Method::Reset,
            &protocol::ResetRequest { configuration: Some(support::any(support::frontier::RENEW)) },
        );
        assert!(matches!(reset, Outcome::Reset(_)));
        let proposal =
            call(&mut guest, &mut records, Method::Propose, &protocol::ProposeRequest { seed: 0 });
        assert!(matches!(proposal, Outcome::Propose(_)));
        for (value, expected_steps, expected_outcomes) in [
            (support::frontier::RENEW, vec!["Renew"], vec!["Renewed"]),
            (terminal, steps, outcomes),
        ] {
            let response = call(
                &mut guest,
                &mut records,
                Method::Judge,
                &protocol::JudgeRequest { observation: Some(support::any(value)) },
            );
            let Outcome::Judge(response) = response else { panic!("frontier Judge refused") };
            let Some(protocol::judge_response::Judgment::Explained(explanation)) =
                response.judgment
            else {
                panic!("frontier Judge not explained")
            };
            assert_eq!(explanation.steps, expected_steps);
            assert_eq!(explanation.outcomes, expected_outcomes);
            assert_eq!(response.continue_polling, value == support::frontier::RENEW);
        }
    }
    quiescence_transcript(&mut guest, &mut records);
    records
}

fn quiescence_transcript(guest: &mut Guest<control::Control>, records: &mut Vec<Record>) {
    use protocol::{Method, response::Outcome};
    for configuration in [54, 55] {
        assert!(matches!(
            call(
                guest,
                records,
                Method::Reset,
                &protocol::ResetRequest { configuration: Some(support::any(configuration)) }
            ),
            Outcome::Reset(_)
        ));
        assert!(matches!(
            call(guest, records, Method::Propose, &protocol::ProposeRequest { seed: 0 }),
            Outcome::Propose(_)
        ));
        let response = call(
            guest,
            records,
            Method::Judge,
            &protocol::JudgeRequest { observation: Some(support::any(support::frontier::OPAQUE)) },
        );
        let Outcome::Judge(response) = response else { panic!("opaque Judge refused") };
        let Some(protocol::judge_response::Judgment::Explained(explanation)) = response.judgment
        else {
            panic!("opaque Judge not explained")
        };
        assert_eq!(explanation.steps, ["Tick", "Tick"]);
        assert_eq!(explanation.outcomes, ["Advanced", "PendingClosed"]);
        let response = call(guest, records, Method::Propose, &protocol::ProposeRequest { seed: 0 });
        let Outcome::Refusal(refusal) = response else {
            panic!("ambiguous quiescence was accepted")
        };
        assert_eq!(refusal.code(), protocol::RefusalCode::State);
        assert!(refusal.message.contains("disagree"));
    }
}

fn judge(
    guest: &mut Guest<control::Control>,
    records: &mut Vec<Record>,
    value: u8,
    steps: &[&str],
) {
    let response = call(
        guest,
        records,
        protocol::Method::Judge,
        &protocol::JudgeRequest { observation: Some(support::any(value)) },
    );
    let protocol::response::Outcome::Judge(response) = response else { panic!("Judge refused") };
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = response.judgment else {
        panic!("Judge was not explained")
    };
    assert_eq!(explanation.steps, steps);
    assert_eq!(
        explanation.outcomes,
        (1..=steps.len()).map(|index| format!("Outcome({index})")).collect::<Vec<_>>()
    );
    assert!(!response.continue_polling);
}

fn main() -> Result<(), Box<dyn Error>> {
    let output = PathBuf::from(std::env::args_os().nth(1).ok_or("missing output directory")?);
    fs::create_dir_all(&output)?;
    for name in ["ordinary", "callback", "configured", "frontier"] {
        let records = transcript(name);
        let mut methods = String::new();
        for (index, record) in records.into_iter().enumerate() {
            writeln!(methods, "{}", record.method as i32)?;
            fs::write(output.join(format!("{name}-{index}.request.pb")), record.request)?;
            fs::write(output.join(format!("{name}-{index}.response.pb")), record.response)?;
        }
        fs::write(output.join(format!("{name}.methods")), methods)?;
    }
    Ok(())
}

#[test]
fn native_current_contract_carries_whole_prefixes() {
    for name in ["ordinary", "callback", "configured", "frontier"] {
        assert!(!transcript(name).is_empty());
    }
}
