use gomad_wasmhost::linked::Conditions;

/// # Errors
/// Keeps diagnostic replay limited to the cooperative profile and matching linked role.
pub fn select(
    profile: &str,
    stress_jitter: bool,
    determinism_fixture: bool,
) -> Result<Conditions, crate::cli::Error> {
    if profile != "gomad.wasi-cooperative/v1" && (stress_jitter || determinism_fixture) {
        return Err(crate::cli::Error {
            status: 2,
            message: if stress_jitter {
                "stress replay requires a cooperative WASM installation"
            } else {
                "determinism fixture replay requires a cooperative WASM installation"
            }
            .into(),
        });
    }
    let conditions = if stress_jitter {
        Conditions::stress_replay()
    } else if determinism_fixture {
        Conditions::determinism_fixture_replay()
    } else {
        Conditions::default()
    };
    conditions.validate().map_err(|error| crate::cli::Error {
        status: 3,
        message: format!(
            "WASM instrumentation divergence: inspection terminal \"invalid\": {error}"
        ),
    })?;
    Ok(conditions)
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Original `TestStressReplayRequiresExplicitCooperativeProvider`: a stock
    /// installation refuses stress replay, and a cooperative one selects the
    /// stress-jitter configuration revision only where the linked helper has
    /// that role; ordinary selection is unchanged.
    #[test]
    fn stress_replay_requires_cooperative_profile_and_the_jitter_revision() {
        let Err(error) = select("gomad.wasi-stock-exploratory/v1", true, false) else {
            panic!("stock stress replay admitted");
        };
        assert_eq!(error.status, 2);
        assert!(error.message.contains("cooperative"), "{}", error.message);
        assert!(matches!(
            select("gomad.wasi-cooperative/v1", false, false),
            Ok(conditions) if conditions == Conditions::default()
        ));
        assert_eq!(
            Conditions::stress_replay().configuration(),
            format!("{}-stress-jitter", gomad_wasmhost::linked::CONFIGURATION)
        );
        match select("gomad.wasi-cooperative/v1", true, false) {
            Ok(conditions) => assert!(conditions == Conditions::stress_replay()),
            Err(error) => {
                assert!(!cfg!(feature = "helper-stress-jitter"), "{}", error.message);
                assert_eq!(error.status, 3);
            }
        }
    }
}
