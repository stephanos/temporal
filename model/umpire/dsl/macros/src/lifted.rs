//! The admitted Model, printed as data the compiled Model embeds for the lifted back ends.
use umpire_front::Embedded;
use umpire_ir::print::Print;

/// `impl Lifts for World` in the expanded module, off the wasm guest.
pub(crate) fn embed(declared: &umpire_front::declare::Machine, input: &syn::ItemMod) -> syn::Item {
    let span = input.ident.span();
    // rustc's path for the file; the runner resolves it, since the macro reads no files.
    let file = span.local_file().map_or_else(|| span.file(), |file| file.display().to_string());
    let embedded = Embedded {
        file,
        model: umpire_front::admit(declared, input),
        source: umpire_front::source(input),
    }
    .print();
    syn::parse_quote! {
        #[cfg(not(target_arch = "wasm32"))]
        #[doc(hidden)]
        impl ::umpire_dsl::__lifted::Lifts for World {
            fn __umpire_lifted() -> ::umpire_dsl::__lifted::Embedded { #embedded }
        }
    }
}
