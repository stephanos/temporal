use crate::shared_returns::gate;
use gate::World;

umpire_testing::fixture!(World = World::new(2), 64);

#[test]
fn sequential_returns_share_definitions_in_unit_and_payload_steps() -> umpire_testing::TestResult {
    let folder = std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("fixtures/shared_returns");
    let module = bound().lift(folder.join("model.rs"))?;
    for operator in ["Client_Step ==", "Client_Add_At("] {
        let step = module.tla.lines().find(|line| line.starts_with(operator)).ok_or(operator)?;
        assert!(step.contains("LET (* @type: Bool; *) __umpire_l1 =="), "{step}");
    }
    Ok(())
}
