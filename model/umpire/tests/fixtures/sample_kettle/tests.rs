use crate::sample_kettle::kettle;
use umpire_dsl::Claim;

umpire_testing::fixture!(kettle::world(), 1 << 20, Claim::new("never_beeps", |_, state: &kettle::Model| (!state.beeping).into()));
