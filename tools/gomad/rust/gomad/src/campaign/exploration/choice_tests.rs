use super::*;

fn config() -> ChoiceConfig {
    ChoiceConfig {
        execution: linked::ChoiceIdentity {
            target_sha256: [1; 32],
            toolchain_build_key: "ab".repeat(32),
            goos: "wasip1".into(),
            goarch: "wasm".into(),
            implementation_sha256: [2; 32],
        },
        controller_sha256: implementation(),
        base_seed: 7,
        parallel: 1,
        max_executions: 20,
        max_choice_depth: 2,
        start_ordinal: 0,
        max_exploration_bytes: 1 << 20,
        failure_policy: "all".into(),
        failure_budget: 1,
        site_scope: None,
    }
}

fn trace(first_selected: u32) -> Vec<u8> {
    let mut bytes = Vec::new();
    for (ordinal, alternatives, selected) in [(0_u64, 3_u32, first_selected), (1, 2, 0)] {
        let mut record = [0_u8; 96];
        record[..8].copy_from_slice(&ordinal.to_be_bytes());
        record[8] = 1;
        record[9] = 1;
        record[12..16].copy_from_slice(&alternatives.to_be_bytes());
        record[16..20].copy_from_slice(&selected.to_be_bytes());
        record[24..32].copy_from_slice(&(100 + ordinal).to_be_bytes());
        record[32..64].fill(u8::try_from(selected).unwrap() + 1);
        record[64..96].fill(9);
        bytes.extend_from_slice(&record);
    }
    bytes
}

fn observed(candidate: &ChoiceCandidate, raw: Vec<u8>) -> ChoiceResult {
    ChoiceResult {
        candidate_sha256: candidate.sha256.clone(),
        outcome_sha256: format!("sha256:{}", canonical::digest(&raw)),
        failed: false,
        failure_sha256: String::new(),
        divergence: None,
        source_trace: Some(raw),
    }
}

fn scoped_fixture(pattern: &str) -> (ChoiceState, Vec<u8>) {
    use sha2::Digest as _;
    let fixture =
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity/symbolize");
    let module = std::fs::read(fixture.join("guest.wasm")).unwrap();
    let raw = std::fs::read(fixture.join("choices.bin")).unwrap();
    let mut config = config();
    config.execution.target_sha256 = sha2::Sha256::digest(&module).into();
    config.max_choice_depth = 32;
    let patterns = vec![pattern.into()];
    let mut resolution = Resolution::default();
    resolution
        .observe(&module, &raw, &config.execution, &patterns)
        .unwrap();
    config.site_scope = Some(resolution.scope(&patterns));
    let mut state = ChoiceState::new(config).unwrap();
    state.bind_scope_module(module.into()).unwrap();
    (state, raw)
}

#[test]
fn site_scope_empty_base_commits_trace_and_reports_skipped_alternatives() {
    let (state, raw) = scoped_fixture("not.in.the.module");
    let round = state.next_round().unwrap();
    let (next, segment) = state
        .commit(&round, &[observed(&round.candidates[0], raw)])
        .unwrap();
    assert_eq!(next.stop_reason, "site_scope_empty");
    assert_eq!(next.queue, Vec::new());
    assert_eq!(next.logical_executions, 1);
    assert_eq!(segment.site_scope.as_ref().unwrap().resolved_count, 0);
    assert_eq!(segment.scope_counts.as_ref().unwrap().forced_in_scope, 0);
    assert!(segment.scope_counts.as_ref().unwrap().skipped_out_of_scope > 0);
    assert_eq!(
        state.replay_segment(&segment).unwrap().hash().unwrap(),
        next.hash().unwrap()
    );
}

#[test]
fn site_scope_selects_match_symbolized_packages_and_resume_rechecks_digest() {
    let (state, raw) = scoped_fixture("main");
    let round = state.next_round().unwrap();
    let result = observed(&round.candidates[0], raw);
    let (next, segment) = state.commit(&round, std::slice::from_ref(&result)).unwrap();
    assert_ne!(next.queue, Vec::new());
    assert!(segment.site_scope.as_ref().unwrap().resolved_count > 0);
    assert_eq!(
        segment.scope_counts.as_ref().unwrap().skipped_out_of_scope,
        0
    );
    assert_eq!(
        segment.scope_counts.as_ref().unwrap().forced_in_scope,
        next.queue.len() as u64
    );
    let mut changed = state;
    changed.config.site_scope.as_mut().unwrap().resolved_sha256 =
        format!("sha256:{}", "0".repeat(64));
    assert_eq!(
        changed.commit(&round, &[result]).err().unwrap(),
        "exploration plan identity or bounds changed"
    );
}

#[test]
fn bounded_breadth_first_rounds_replay_and_report_execution_limit() {
    let mut config = config();
    config.parallel = 2;
    config.max_executions = 3;
    let initial = ChoiceState::new(config).unwrap();
    let root = initial.next_round().unwrap();
    let (expanded, first) = initial
        .commit(&root, &[observed(&root.candidates[0], trace(0))])
        .unwrap();
    assert_eq!(
        expanded
            .queue
            .iter()
            .map(|candidate| candidate.forced_depth)
            .collect::<Vec<_>>(),
        [1, 1]
    );
    assert!(
        expanded
            .queue
            .windows(2)
            .all(|pair| pair[0].sha256 < pair[1].sha256)
    );
    assert_eq!(expanded.omitted_by_execution_bound, 1);
    assert_eq!(
        initial.replay_segment(&first).unwrap().hash().unwrap(),
        expanded.hash().unwrap()
    );

    let round = expanded.next_round().unwrap();
    let results = round
        .candidates
        .iter()
        .map(|candidate| observed(candidate, trace(candidate.rank.unwrap())))
        .collect::<Vec<_>>();
    let (finished, second) = expanded.commit(&round, &results).unwrap();
    assert_eq!(
        (
            finished.logical_executions,
            finished.committed_rounds,
            finished.seen.len(),
            finished.omitted_by_execution_bound,
            finished.stop_reason.as_str()
        ),
        (3, 2, 3, 3, "max_executions")
    );
    assert!(finished.next_round().is_none());
    assert_eq!(finished.summary()["bounded_complete"], false);
    assert_eq!(
        expanded.replay_segment(&second).unwrap().hash().unwrap(),
        finished.hash().unwrap()
    );
}

#[test]
fn frontier_orders_shallow_prefixes_before_deeper_prefixes() {
    let initial = ChoiceState::new(config()).unwrap();
    let root = initial.next_round().unwrap();
    let (expanded, _) = initial
        .commit(&root, &[observed(&root.candidates[0], trace(0))])
        .unwrap();
    assert_eq!(
        expanded
            .queue
            .iter()
            .map(|candidate| candidate.forced_depth)
            .collect::<Vec<_>>(),
        [1, 1, 2]
    );
    let round = expanded.next_round().unwrap();
    let parent = &round.candidates[0];
    let (next, _) = expanded
        .commit(&round, &[observed(parent, trace(parent.rank.unwrap()))])
        .unwrap();
    assert_eq!(next.next_round().unwrap().candidates[0], expanded.queue[1]);
    assert_eq!(
        next.queue
            .iter()
            .map(|candidate| candidate.forced_depth)
            .collect::<Vec<_>>(),
        [1, 2, 2]
    );
}

#[test]
fn depth_and_capacity_bounds_report_omitted_prefixes() {
    let mut depth = config();
    depth.max_choice_depth = 1;
    let initial = ChoiceState::new(depth).unwrap();
    let root = initial.next_round().unwrap();
    let (expanded, _) = initial
        .commit(&root, &[observed(&root.candidates[0], trace(0))])
        .unwrap();
    assert_eq!((expanded.queue.len(), expanded.omitted_by_depth), (2, 1));

    let mut capacity = config();
    capacity.max_exploration_bytes = ChoiceState::new(capacity.clone()).unwrap().pending_bytes;
    let initial = ChoiceState::new(capacity).unwrap();
    let root = initial.next_round().unwrap();
    let (finished, _) = initial
        .commit(&root, &[observed(&root.candidates[0], trace(0))])
        .unwrap();
    assert_eq!(
        (
            finished.queue.len(),
            finished.omitted_by_capacity,
            finished.stop_reason.as_str()
        ),
        (0, 3, "exploration_capacity")
    );
    assert_eq!(finished.summary()["bounded_complete"], false);
}

#[test]
fn replay_segment_refuses_corruption_even_after_rehashing() {
    let initial = ChoiceState::new(config()).unwrap();
    let root = initial.next_round().unwrap();
    let (_, segment) = initial
        .commit(&root, &[observed(&root.candidates[0], trace(0))])
        .unwrap();
    let mut corrupt = segment.clone();
    corrupt.sha256 = "sha256:invalid".into();
    assert!(initial.replay_segment(&corrupt).is_err());

    let mut corrupt = segment.clone();
    corrupt.after_sha256 = initial.hash().unwrap();
    corrupt.sha256 = corrupt.hash().unwrap();
    assert!(
        initial
            .replay_segment(&corrupt)
            .err()
            .unwrap()
            .contains("replay changed")
    );

    let mut corrupt = segment;
    corrupt.results[0].candidate_sha256 = format!("sha256:{}", "00".repeat(32));
    corrupt.sha256 = corrupt.hash().unwrap();
    assert!(
        initial
            .replay_segment(&corrupt)
            .err()
            .unwrap()
            .contains("candidate mismatch")
    );
}

#[test]
fn forced_prefix_divergence_authenticates_helper_decisions() {
    let initial = ChoiceState::new(config()).unwrap();
    let root = initial.next_round().unwrap();
    let raw = trace(0);
    let (expanded, _) = initial
        .commit(&root, &[observed(&root.candidates[0], raw.clone())])
        .unwrap();
    let round = expanded.next_round().unwrap();
    let parent = &round.candidates[0];
    let request = parent.replay(&expanded.config.execution).unwrap().unwrap();
    let plan = linked::project_choices(&raw, &expanded.config.execution).unwrap();
    let actual = plan.decisions[0].clone();
    let prefix = plan
        .prefix(request.prefix_control.as_ref().unwrap())
        .unwrap();
    let divergence = linked::Divergence {
        ordinal: 0,
        reason: 4,
        expected: Some(prefix.decisions[0].clone()),
        observed: Some(actual.clone()),
        tape_records: parent.forced_depth,
    };
    validate_divergence(parent, &expanded.config.execution, &divergence).unwrap();
    assert!(
        validate_divergence(&root.candidates[0], &expanded.config.execution, &divergence).is_err()
    );
    for field in ["expected", "observed", "ordinal", "tape_records", "reason"] {
        let mut corrupt = divergence.clone();
        match field {
            "expected" => corrupt.expected = Some(actual.clone()),
            "observed" => corrupt.observed = divergence.expected.clone(),
            "ordinal" => corrupt.ordinal = parent.forced_depth,
            "tape_records" => corrupt.tape_records += 1,
            "reason" => corrupt.reason = 6,
            _ => unreachable!(),
        }
        assert!(
            validate_divergence(parent, &expanded.config.execution, &corrupt).is_err(),
            "{field}"
        );
    }
    let result = ChoiceResult {
        candidate_sha256: parent.sha256.clone(),
        outcome_sha256: String::new(),
        failed: false,
        failure_sha256: String::new(),
        divergence: Some(divergence),
        source_trace: None,
    };
    let (next, segment) = expanded.commit(&round, &[result]).unwrap();
    assert_eq!(
        expanded.replay_segment(&segment).unwrap().hash().unwrap(),
        next.hash().unwrap()
    );
    assert_eq!(
        (
            next.logical_executions,
            next.outcomes.len(),
            next.failure_signatures.len()
        ),
        (2, 1, 0)
    );
}
