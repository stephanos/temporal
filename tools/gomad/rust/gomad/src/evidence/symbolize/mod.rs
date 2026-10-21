use serde::Serialize;
use std::collections::BTreeMap;

mod bytes;
mod tables;
mod wasm;

const _: () = {
    assert!(std::mem::size_of::<Frame>() <= 128);
    assert!(std::mem::size_of::<(Site, Symbol)>() <= 192);
};

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
#[serde(untagged)]
pub(crate) enum Symbol {
    Resolved {
        #[serde(flatten)]
        frame: Frame,
        inlined_from: Vec<Frame>,
    },
    Unavailable {
        unavailable: Reason,
    },
}

#[derive(Clone, Debug, PartialEq, Eq, Serialize)]
pub(crate) struct Frame {
    pub package: String,
    pub function: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub line: Option<i32>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub(crate) enum Reason {
    SiteMissing,
    NoSymbols,
    OffsetOutOfRange,
    SymbolMismatch,
    SymbolTableTooLarge,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub(crate) struct Site {
    pub value: u64,
    pub missing: bool,
}

#[derive(Debug)]
#[allow(
    dead_code,
    reason = "The inspect consumer is implemented in dependent fn-229.2."
)]
pub(crate) struct Symbols {
    entries: BTreeMap<Site, Symbol>,
    failure: Option<Symbol>,
}

impl Symbols {
    #[allow(
        dead_code,
        reason = "The inspect consumer is implemented in dependent fn-229.2."
    )]
    pub(crate) fn get(&self, site: Site) -> &Symbol {
        const MISSING: Symbol = Symbol::Unavailable {
            unavailable: Reason::SiteMissing,
        };
        const UNKNOWN: Symbol = Symbol::Unavailable {
            unavailable: Reason::NoSymbols,
        };
        if site.missing {
            &MISSING
        } else {
            self.failure
                .as_ref()
                .or_else(|| self.entries.get(&site))
                .unwrap_or(&UNKNOWN)
        }
    }

    fn failed(reason: Reason) -> Self {
        Self {
            entries: BTreeMap::new(),
            failure: Some(unavailable(reason)),
        }
    }
}

#[allow(
    dead_code,
    reason = "The inspect consumer is implemented in dependent fn-229.2."
)]
pub(crate) fn resolve(
    opened: &super::store::Opened,
    sites: impl IntoIterator<Item = Site>,
) -> Result<Symbols, String> {
    let entry = opened
        .manifest()
        .files
        .get("target/module.wasm")
        .ok_or("unlisted artifact payload")?;
    let bytes = opened.payload("target/module.wasm", entry.bytes)?;
    Ok(resolve_bytes(&bytes, sites))
}

fn unavailable(reason: Reason) -> Symbol {
    Symbol::Unavailable {
        unavailable: reason,
    }
}

fn resolve_bytes(bytes: &[u8], sites: impl IntoIterator<Item = Site>) -> Symbols {
    let mut budget = bytes::Budget::default();
    let mut requested = BTreeMap::new();
    for site in sites {
        if budget.steps(1).is_err() {
            return Symbols::failed(Reason::SymbolTableTooLarge);
        }
        if requested.contains_key(&site) {
            continue;
        }
        if requested.is_empty() && budget.bytes(4096).is_err() {
            return Symbols::failed(Reason::SymbolTableTooLarge);
        }
        if budget.entries(1).and_then(|()| budget.bytes(512)).is_err() {
            return Symbols::failed(Reason::SymbolTableTooLarge);
        }
        requested.insert(site, unavailable(Reason::NoSymbols));
    }
    let parsed = {
        wasm::Module::read(bytes, &mut budget).and_then(|module| {
            let tables = tables::Tables::read(&module, &mut budget)?;
            for (site, result) in &mut requested {
                let value = if site.missing {
                    unavailable(Reason::SiteMissing)
                } else {
                    match resolve_site(&module, &tables, site.value, &mut budget) {
                        Ok(value) => value,
                        Err(Reason::SymbolTableTooLarge) => {
                            return Err(Reason::SymbolTableTooLarge);
                        }
                        Err(reason) => unavailable(reason),
                    }
                };
                *result = value;
            }
            Ok(())
        })
    };
    match parsed {
        Ok(()) => Symbols {
            entries: requested,
            failure: None,
        },
        Err(reason) => Symbols::failed(reason),
    }
}

fn resolve_site(
    module: &wasm::Module<'_>,
    tables: &tables::Tables,
    site: u64,
    budget: &mut bytes::Budget,
) -> bytes::Result<Symbol> {
    budget.steps(1)?;
    let mut comparisons = 0;
    let index = tables.functions.partition_point(|function| {
        comparisons += 1;
        function.offset <= site
    });
    budget.steps(comparisons)?;
    let function = index
        .checked_sub(1)
        .and_then(|index| tables.functions.get(index))
        .filter(|function| site == function.offset)
        .ok_or(Reason::OffsetOutOfRange)?;
    budget.bytes(function.name.len() as u64 * 3 + bytes::FRAME_CHARGE)?;
    budget.steps(bytes::mul(function.name.len() as u64, 3)?)?;
    let physical = tables::split(&function.name);
    let fallback = || Symbol::Resolved {
        frame: physical.clone(),
        inlined_from: Vec::new(),
    };
    let Some(select) = tables.select else {
        return Ok(fallback());
    };
    let entry = bytes::add(tables.text, bytes::mul(function.offset, 65536)?)?;
    let Some(candidates) = wasm::returns(module.body(function.index)?, select, entry, budget)?
    else {
        return Ok(fallback());
    };
    let mut consensus: Option<Vec<Frame>> = None;
    for pc in candidates {
        if (pc - tables.text) >> 16 != site {
            return Err(Reason::SymbolMismatch);
        }
        let chain = tables.chain(&module.memory, function, pc - 1 - entry, budget)?;
        if let Some(previous) = &mut consensus {
            if chain.len() != previous.len()
                || chain.iter().zip(previous.iter()).any(|(left, right)| {
                    left.package != right.package || left.function != right.function
                })
            {
                return Ok(fallback());
            }
            for (left, right) in previous.iter_mut().zip(chain) {
                if left.file != right.file || left.line != right.line {
                    left.file = None;
                    left.line = None;
                }
            }
        } else {
            consensus = Some(chain);
        }
    }
    let chain = consensus.ok_or(Reason::SymbolMismatch)?;
    budget.bytes(bytes::mul(chain.len() as u64, bytes::FRAME_CHARGE)?)?;
    let mut chain = chain.into_iter();
    Ok(Symbol::Resolved {
        frame: chain.next().ok_or(Reason::SymbolMismatch)?,
        inlined_from: chain.collect(),
    })
}

#[cfg(test)]
mod tests;
