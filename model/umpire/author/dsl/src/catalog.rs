use crate::ActionDomain;

/// The normal maximum number of concrete actions a Model catalog may declare.
pub const CATALOG_LIMIT: usize = 4096;

/// One declared action variant and its exact concrete action count, possibly zero.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CatalogVariant {
    /// The variant's qualified spelling, such as `User::Set`.
    pub name: String,
    /// Concrete actions this variant contributes.
    pub count: usize,
}

/// An admitted concrete action catalog in declaration order, including disabled actions.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ActionCatalog<A> {
    actions: Vec<A>,
    variants: Option<Vec<CatalogVariant>>,
}

impl<A> ActionCatalog<A> {
    /// Admits an already materialized catalog by its length, without variant metadata.
    ///
    /// # Errors
    /// Refuses a catalog longer than `limit`.
    pub fn checked(actions: Vec<A>, limit: usize) -> Result<Self, CatalogError> {
        if actions.len() > limit {
            return Err(CatalogError::TooLarge { count: actions.len(), limit });
        }
        Ok(Self { actions, variants: None })
    }

    /// Counts every declared variant's lazy domain before materializing any action.
    ///
    /// # Errors
    /// Refuses an exact count above `limit`, or a count that overflows `usize`.
    pub fn admit(
        limit: usize,
        variants: Vec<(String, ActionDomain<A>)>,
    ) -> Result<Self, CatalogError>
    where
        A: 'static,
    {
        let mut counts = Vec::with_capacity(variants.len());
        let mut total = 0usize;
        for (_, domain) in &variants {
            let count = domain.count().map_err(|_| CatalogError::Overflow { limit })?;
            total = total.checked_add(count).ok_or(CatalogError::Overflow { limit })?;
            counts.push(count);
        }
        if total > limit {
            return Err(CatalogError::TooLarge { count: total, limit });
        }
        let mut actions = Vec::with_capacity(total);
        let mut declared = Vec::with_capacity(variants.len());
        for ((name, domain), count) in variants.into_iter().zip(counts) {
            actions.extend((0..count).map(|index| domain.get(index)));
            declared.push(CatalogVariant { name, count });
        }
        Ok(Self { actions, variants: Some(declared) })
    }

    /// Concrete actions in declaration order.
    pub fn actions(&self) -> &[A] {
        &self.actions
    }

    /// Releases the concrete actions.
    pub fn into_actions(self) -> Vec<A> {
        self.actions
    }

    /// Declared variant counts, or `None` when the catalog was not built from declarations.
    pub fn variants(&self) -> Option<&[CatalogVariant]> {
        self.variants.as_deref()
    }

    /// Converts actions while retaining variant metadata.
    #[must_use]
    pub fn map<B>(self, map: impl FnMut(A) -> B) -> ActionCatalog<B> {
        ActionCatalog {
            actions: self.actions.into_iter().map(map).collect(),
            variants: self.variants,
        }
    }
}

/// A catalog refused before any Model step, distinct from an exploration verdict.
#[derive(Clone, Copy, Debug, PartialEq, Eq, thiserror::Error)]
pub enum CatalogError {
    /// The exact representable count exceeds the admitted limit.
    #[error("action catalog has {count} actions, over limit {limit}")]
    TooLarge {
        /// Exact concrete action count.
        count: usize,
        /// Admitted maximum.
        limit: usize,
    },
    /// The concrete action count is not representable in `usize`.
    #[error("action catalog count overflows usize (limit {limit})")]
    Overflow {
        /// Admitted maximum.
        limit: usize,
    },
}
