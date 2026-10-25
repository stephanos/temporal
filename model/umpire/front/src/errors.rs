#[derive(Default)]
pub(crate) struct Errors(Option<syn::Error>);

impl Errors {
    pub(crate) fn push(&mut self, error: syn::Error) {
        match &mut self.0 {
            Some(previous) => previous.combine(error),
            None => self.0 = Some(error),
        }
    }

    pub(crate) fn finish<T>(self, value: T) -> syn::Result<T> {
        self.0.map_or(Ok(value), Err)
    }
}
