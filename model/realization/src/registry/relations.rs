//! Preserve generated relation identity and typed failure classes at the protocol boundary.

use umpire_catalog::{RelationCheckError, RelationFailureClass};
use umpire_runtime::protocol::{Refusal, RefusalCode};

/// Value-free proof that one actual bound invocation passed its generated checks.
pub struct CheckedRelations {
    rpc: String,
    declarations: Vec<String>,
}

impl CheckedRelations {
    /// Descriptor-admitted RPC identity.
    pub fn rpc(&self) -> &str {
        &self.rpc
    }

    /// Exact declarations checked by this invocation.
    pub fn declarations(&self) -> &[String] {
        &self.declarations
    }
}

/// Execute the production evaluator before minting an invocation audit.
pub fn check<Q: 'static, S: 'static, R: 'static, P: 'static>(
    binding: &umpire_catalog::BoundRpc<Q, S, R, P>,
    context: &umpire_catalog::RpcContext<'_, Q, S, R, P>,
) -> Result<CheckedRelations, Refusal> {
    binding.check(context).map_err(|error| refusal(&error))?;
    Ok(CheckedRelations {
        rpc: binding.schema().method().method.clone(),
        declarations: binding
            .schema()
            .relations()
            .iter()
            .map(|relation| relation.identity().to_owned())
            .collect(),
    })
}

/// Combine a generated relation check with the hand-written correlation it accompanies.
///
/// When both refuse with different classes the established correlation refusal wins, so
/// generated checks never reclassify existing evidence; matching classes keep the generated
/// diagnostic, and a refusal only one side reaches is retained.
pub fn reconcile<T>(
    generated: Result<CheckedRelations, Refusal>,
    legacy: Result<T, Refusal>,
) -> Result<T, Refusal> {
    match (generated, legacy) {
        (Ok(_), legacy) => legacy,
        (Err(generated), Ok(_)) => Err(generated),
        (Err(generated), Err(legacy)) => {
            Err(if generated.code == legacy.code { generated } else { legacy })
        }
    }
}

/// Map an already sanitized relation diagnostic without changing its meaning.
pub fn refusal(error: &RelationCheckError) -> Refusal {
    let code = match error.class {
        RelationFailureClass::Malformed => RefusalCode::MalformedRequest,
        RelationFailureClass::Unsupported => RefusalCode::Unsupported,
        RelationFailureClass::State => RefusalCode::State,
    };
    Refusal::new(code, error.to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn checked() -> CheckedRelations {
        CheckedRelations { rpc: "rpc".into(), declarations: vec!["declaration".into()] }
    }

    fn state(message: &str) -> Refusal {
        Refusal::new(RefusalCode::State, message)
    }

    #[test]
    fn reconcile_keeps_established_classes_and_generated_only_refusals() {
        assert_eq!(reconcile(Ok(checked()), Ok(1)), Ok(1));
        assert_eq!(reconcile(Ok(checked()), Err::<(), _>(state("legacy"))), Err(state("legacy")));
        assert_eq!(reconcile(Err(state("generated")), Ok(())), Err(state("generated")));
        assert_eq!(
            reconcile(Err(state("generated")), Err::<(), _>(state("legacy"))),
            Err(state("generated")),
            "a matching class surfaces the generated diagnostic",
        );
        for legacy in [Refusal::malformed("legacy"), Refusal::unsupported("legacy")] {
            assert_eq!(
                reconcile(Err(state("generated")), Err::<(), _>(legacy.clone())),
                Err(legacy),
                "a differing established class wins",
            );
        }
        assert_eq!(
            reconcile(Err(Refusal::malformed("generated")), Err::<(), _>(state("legacy"))),
            Err(state("legacy")),
        );
    }
}
