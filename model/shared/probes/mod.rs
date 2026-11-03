//! Typed baseline routing; catalog admission and gap reconciliation remain catalog-owned.

use crate::registry::{ActionKind, Registry};
use umpire_runtime::{capability::Request as _, prost_types::Any, protocol::Refusal};

/// One declared bootstrap, read or action baseline constructor.
pub struct Filler<C> {
    /// Descriptor method name used by the existing probe inventory.
    pub method: &'static str,
    /// Typed system adapter, packed only at the transport boundary.
    pub fill: fn(&C, &str) -> Result<Any, Refusal>,
}

/// An admitted, immutable baseline inventory independent of mutation or gap policy.
pub struct Baselines<'a, C> {
    fillers: &'a [Filler<C>],
}

impl<'a, C> Baselines<'a, C> {
    /// Refuse ambiguous method ownership before any filler can run.
    pub fn new(fillers: &'a [Filler<C>]) -> Result<Self, Refusal> {
        for (index, filler) in fillers.iter().enumerate() {
            if filler.method.is_empty()
                || fillers[..index].iter().any(|other| other.method == filler.method)
            {
                return Err(Refusal::malformed("invalid or duplicate probe baseline method"));
            }
        }
        Ok(Self { fillers })
    }

    /// Declared filler methods, in declaration order.
    pub fn methods(&self) -> impl Iterator<Item = &'static str> + '_ {
        self.fillers.iter().map(|filler| filler.method)
    }

    /// Construct the declared baseline without mutating shared inventory state.
    pub fn fill(&self, method: &str, context: &C, variant: &str) -> Result<Any, Refusal> {
        let filler = self
            .fillers
            .iter()
            .find(|filler| filler.method == method)
            .ok_or_else(|| Refusal::unsupported("unknown prerequisite method"))?;
        (filler.fill)(context, variant)
    }
}

/// Fill an action baseline through the same registry capability used by proposals and workers.
pub fn registered<B, A>(
    binding: &B,
    method: &str,
    action: &A,
    world: &B::World,
    inputs: &B::Inputs,
    identity: &B::Id,
) -> Result<Any, Refusal>
where
    B: Registry<A>,
{
    if !matches!(binding.row(action).kind, ActionKind::Rpc { method: actual, .. } if actual == method)
    {
        return Err(Refusal::unsupported("probe baseline does not match registered RPC action"));
    }
    Ok(binding.fill(action, world, inputs, identity)?.pack_request())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::cell::Cell;
    #[expect(clippy::unnecessary_wraps, reason = "Matches the declared filler signature")]
    fn filled(calls: &Cell<u32>, _: &str) -> Result<Any, Refusal> {
        calls.set(calls.get() + 1);
        Ok(Any { type_url: "type.googleapis.com/probe.Request".into(), value: vec![1] })
    }
    #[test]
    fn duplicate_and_unknown_methods_never_call_a_filler() -> Result<(), Refusal> {
        let calls = Cell::new(0);
        assert!(
            Baselines::new(&[
                Filler { method: "Read", fill: filled },
                Filler { method: "Read", fill: filled },
            ])
            .is_err()
        );
        let fillers = [Filler { method: "Read", fill: filled }];
        let baselines = Baselines::new(&fillers)?;
        assert!(baselines.fill("Write", &calls, "").is_err());
        assert_eq!(calls.get(), 0);
        assert_eq!(baselines.methods().collect::<Vec<_>>(), ["Read"]);
        assert_eq!(baselines.fill("Read", &calls, "")?.value, [1]);
        assert_eq!(calls.get(), 1);
        Ok(())
    }
}
