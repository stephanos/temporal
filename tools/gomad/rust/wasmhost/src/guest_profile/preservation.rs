use super::Config;

#[test]
fn profile_configuration_rejects_original_malformed_bounds_before_sampling() {
    let valid = Config {
        path: "/diagnostic/profile.json".into(),
        interval_micros: 1_000,
        max_samples: 100,
    };
    valid.validate().unwrap();
    for (name, configuration) in [
        (
            "relative output path",
            Config {
                path: "profile.json".into(),
                ..valid.clone()
            },
        ),
        (
            "zero interval",
            Config {
                interval_micros: 0,
                ..valid.clone()
            },
        ),
        (
            "sample count 100001",
            Config {
                max_samples: 100_001,
                ..valid.clone()
            },
        ),
    ] {
        assert!(
            configuration
                .validate()
                .unwrap_err()
                .to_string()
                .contains("invalid guest profile path or bounds"),
            "{name}"
        );
    }
}
