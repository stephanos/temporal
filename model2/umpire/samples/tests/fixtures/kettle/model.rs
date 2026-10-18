umpire_testing::fixture!(samples::kettle::KettleWorld, samples::kettle::BUDGET, umpire_dsl::Claim::<samples::kettle::KettleWorld>::new("never_beeps", |_, state| (!state.beeping).into()));
