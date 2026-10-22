use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ModuleImport {
    pub module: String,
    pub name: String,
    pub params: Vec<String>,
    pub results: Vec<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ModuleMetadata {
    pub imports: Vec<ModuleImport>,
    pub initial_memory_pages: u64,
}

/// # Errors
/// Returns the imports and initial memory pages encoded in a core module.
pub fn module_metadata(bytes: &[u8]) -> wasmtime::Result<ModuleMetadata> {
    use crate::wasmparser::{Parser, Payload, TypeRef, ValType};
    use wasmtime::Error;

    if !bytes.starts_with(b"\0asm\x01\0\0\0") {
        return Err(Error::msg("WASM binary header is required"));
    }
    let mut types = Vec::new();
    let mut imports = Vec::new();
    let mut pages = None;
    for payload in Parser::new(0).parse_all(bytes) {
        match payload? {
            Payload::TypeSection(section) => {
                for signature in section.into_iter_err_on_gc_types() {
                    let signature = signature?;
                    if signature
                        .params()
                        .iter()
                        .chain(signature.results())
                        .any(|ty| {
                            !matches!(
                                ty,
                                ValType::I32 | ValType::I64 | ValType::F32 | ValType::F64
                            )
                        })
                    {
                        return Err(Error::msg("unsupported module value type"));
                    }
                    types.push(signature);
                }
            }
            Payload::ImportSection(section) => {
                for item in section.into_imports() {
                    let item = item?;
                    let TypeRef::Func(index) = item.ty else {
                        return Err(Error::msg("nonfunction module import is unsupported"));
                    };
                    let signature = types
                        .get(usize::try_from(index)?)
                        .ok_or_else(|| Error::msg("invalid import type"))?;
                    imports.push(ModuleImport {
                        module: item.module.into(),
                        name: item.name.into(),
                        params: signature.params().iter().map(ToString::to_string).collect(),
                        results: signature
                            .results()
                            .iter()
                            .map(ToString::to_string)
                            .collect(),
                    });
                }
            }
            Payload::MemorySection(section) => {
                if section.count() != 1 || pages.is_some() {
                    return Err(Error::msg("exactly one module memory is required"));
                }
                for memory in section {
                    let memory = memory?;
                    if memory.memory64 || memory.shared || memory.page_size_log2.is_some() {
                        return Err(Error::msg("shared or memory64 module is unsupported"));
                    }
                    if memory
                        .maximum
                        .is_some_and(|maximum| maximum < memory.initial)
                    {
                        return Err(Error::msg("invalid module memory maximum"));
                    }
                    pages = Some(memory.initial);
                }
            }
            _ => (),
        }
    }
    let initial_memory_pages = pages
        .filter(|pages| (1..=65536).contains(pages))
        .ok_or_else(|| Error::msg("invalid initial module memory"))?;
    Ok(ModuleMetadata {
        imports,
        initial_memory_pages,
    })
}
