use super::*;
use crate::campaign::scenario;
use std::time::Duration;

#[test]
fn actual_cluster_cost_projection_detaches_counters_and_retains_global_backend_observation() {
    let corpus = Path::new(env!("CARGO_MANIFEST_DIR")).join("../../testdata/parity");
    let publication = publication(&corpus, 7).unwrap();
    let input: Inputs = serde_json::from_slice(&publication.payloads[cluster::FILE]).unwrap();
    let modules = publication.payloads;
    let prepared = prepare_configuration(&input.configuration, &modules).unwrap();
    let execution = helper::execute_prepared(prepared).unwrap();
    assert_eq!(execution.incarnations.len(), 3);
    let limits = input
        .configuration
        .incarnations
        .iter()
        .map(|item| {
            (
                format!("{}/{}", item.handle.node, item.handle.incarnation),
                (item.limits.fuel, item.limits.memory_bytes),
            )
        })
        .collect();
    let wall = Duration::from_nanos(31);
    let mut projected = scenario::store_metrics(&execution, &limits, wall).unwrap();
    let actual_backend = canonical::encode(execution.outcome.metrics.as_deref().unwrap()).unwrap();
    assert_eq!(
        projected.raw_backend.as_deref(),
        Some(actual_backend.as_slice())
    );
    assert_eq!(projected.wall_nanos, 31);
    assert!(projected.store_peak_rss_bytes.is_none());
    let expected_trace: u64 = execution
        .incarnations
        .iter()
        .flat_map(|item| &item.execution.evidence)
        .filter(|(name, _)| {
            crate::campaign::execution::TRACE_STREAMS.contains(&name.as_str())
                && name.as_str() != "network"
        })
        .map(|(_, bytes)| bytes.len() as u64)
        .sum::<u64>()
        + execution.network.len() as u64;
    assert_eq!(
        projected
            .stores
            .iter()
            .map(|item| item.trace_bytes)
            .sum::<u64>(),
        expected_trace
    );
    for (index, item) in execution.incarnations.iter().enumerate() {
        let key = format!("{}/{}", item.handle.node, item.handle.incarnation);
        let (fuel, memory) = limits[&key];
        let observed = &projected.stores[index];
        assert_eq!(observed.incarnation.as_deref(), Some(key.as_str()));
        assert_eq!(
            observed.fuel_used,
            fuel - item.execution.outcome.fuel_remaining
        );
        assert_eq!(observed.guest_memory_limit_bytes, memory);
        assert_eq!(
            observed.guest_peak_memory_bytes,
            item.execution.outcome.peak_memory_bytes as u64
        );
    }
    let expected = format!("{:?}", projected.stores);
    projected.raw_backend.as_mut().unwrap().fill(9);
    for item in &mut projected.stores {
        item.incarnation.as_mut().unwrap().clear();
        item.fuel_used = u64::MAX;
        item.trace_bytes = u64::MAX;
        item.guest_peak_memory_bytes = u64::MAX;
        item.guest_memory_limit_bytes = u64::MAX;
    }
    projected.stores.clear();
    let again = scenario::store_metrics(&execution, &limits, wall).unwrap();
    assert_eq!(again.raw_backend.unwrap(), actual_backend);
    assert_eq!(format!("{:?}", again.stores), expected);
    println!(
        "{}",
        json!({"schema":"gomad.native-cluster-cost-projection/v1", "incarnations":3,
        "shared_trace_bytes":execution.network.len(), "trace_bytes":expected_trace,
        "backend_bytes":actual_backend, "wall_nanos":31})
    );
}
