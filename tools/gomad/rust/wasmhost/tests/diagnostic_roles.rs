use gomad_wasmhost::linked::Conditions;

#[test]
fn typed_native_role_admission_requires_the_exact_compiled_identity() {
    let mut configurations = std::collections::BTreeSet::new();
    for (name, role) in [
        ("release", Conditions::Ordinary),
        ("diagnostic-profile", Conditions::DiagnosticProfile),
        ("stress-jitter", Conditions::StressReplay),
        ("determinism-fixture", Conditions::DeterminismFixtureReplay),
    ] {
        assert!(configurations.insert(role.configuration()), "{name}");
        if role == Conditions::compiled() {
            role.validate().unwrap();
        } else {
            assert!(
                role.validate()
                    .unwrap_err()
                    .to_string()
                    .contains("identity mismatch before authorization"),
                "{name}"
            );
        }
    }
}

#[test]
fn release_binary_cannot_authorize_a_diagnostic_or_stress_identity() {
    if Conditions::compiled() == Conditions::Ordinary {
        for role in [
            Conditions::DiagnosticProfile,
            Conditions::StressReplay,
            Conditions::DeterminismFixtureReplay,
        ] {
            assert!(role.validate().is_err());
        }
    } else {
        assert!(Conditions::Ordinary.validate().is_err());
    }
}
