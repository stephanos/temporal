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
