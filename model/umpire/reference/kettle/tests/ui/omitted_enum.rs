use umpire_reference_kettle::wire::umpire::reference::kettle::v1::HeatingAction;
umpire_catalog::annotate! {
    pub catalog controls;
    enum action: HeatingAction = "umpire.reference.kettle.v1.HeatingAction" {
        HeatingAction::Unspecified = "HEATING_ACTION_UNSPECIFIED",
        HeatingAction::Heat = "HEATING_ACTION_HEAT",
    }
}
fn main() {}
