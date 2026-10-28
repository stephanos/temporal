use serde::Serialize;
use std::{
    fs::File,
    io,
    path::{Path, PathBuf},
};

#[derive(Clone, Debug)]
pub(super) struct SourceRoot {
    pub label: String,
    pub root: PathBuf,
    pub excluded: Vec<PathBuf>,
}
#[derive(Debug, Serialize)]
pub(super) struct Authority {}
pub(super) struct Inputs;
impl Inputs {
    pub fn capture(
        _roots: &[SourceRoot],
        _tools: &[PathBuf],
        _semantic: &impl Serialize,
    ) -> io::Result<Self> {
        Err(io::Error::other("input custody not implemented"))
    }
    pub fn verify_unchanged(&self) -> io::Result<()> {
        Err(io::Error::other("input custody not implemented"))
    }
    pub fn digest(&self) -> &str {
        ""
    }
    pub fn authority(&self) -> &Authority {
        unimplemented!()
    }
}
pub(super) struct PinnedImage;
impl PinnedImage {
    fn from_file(_file: File) -> io::Result<Self> {
        Err(io::Error::other("image custody not implemented"))
    }
    pub fn digest(&self) -> &str {
        ""
    }
    pub fn copy_to(&self, _destination: &Path) -> io::Result<()> {
        Err(io::Error::other("image custody not implemented"))
    }
}
pub(super) fn pin_current_image() -> io::Result<PinnedImage> {
    Err(io::Error::other("image custody not implemented"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{
        ffi::OsString,
        fs,
        os::unix::{
            ffi::OsStringExt,
            fs::{PermissionsExt, symlink},
        },
    };
    fn root(path: &Path) -> SourceRoot {
        SourceRoot { label: "fixture".into(), root: path.canonicalize().unwrap(), excluded: vec![] }
    }
    fn capture(path: &Path) -> io::Result<Inputs> {
        Inputs::capture(
            &[root(path)],
            &[],
            &serde_json::json!({"budget": 10, "selection": ["one"]}),
        )
    }
    #[test]
    fn changes_after_admission_are_refused() -> io::Result<()> {
        for mutation in ["edit", "add", "delete", "mode", "replace", "symlink"] {
            let dir = tempfile::tempdir()?;
            let file = dir.path().join("source");
            fs::write(&file, "before admission")?;
            fs::write(&file, "admitted")?;
            let inputs = capture(dir.path())?;
            inputs.verify_unchanged()?;
            match mutation {
                "edit" => fs::write(&file, "after")?,
                "add" => fs::write(dir.path().join("added"), "new")?,
                "delete" => fs::remove_file(&file)?,
                "mode" => fs::set_permissions(&file, fs::Permissions::from_mode(0o755))?,
                "replace" => {
                    fs::remove_file(&file)?;
                    fs::write(&file, "admitted")?;
                }
                _ => {
                    fs::remove_file(&file)?;
                    symlink("missing", &file)?;
                }
            }
            assert!(inputs.verify_unchanged().is_err(), "accepted {mutation}");
        }
        Ok(())
    }
    #[test]
    fn copied_roots_and_raw_names_have_the_same_semantic_digest() -> io::Result<()> {
        let one = tempfile::tempdir()?;
        let two = tempfile::tempdir()?;
        let name = OsString::from_vec(vec![b'f', 0xff]);
        fs::write(one.path().join(&name), "same")?;
        fs::write(two.path().join(&name), "same")?;
        assert_eq!(capture(one.path())?.digest(), capture(two.path())?.digest());
        Ok(())
    }
    #[test]
    fn budget_and_selection_are_bound_to_the_digest() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let roots = [root(dir.path())];
        let a =
            Inputs::capture(&roots, &[], &serde_json::json!({"budget": 10, "selection": ["one"]}))?;
        for semantic in [
            serde_json::json!({"budget": 11, "selection": ["one"]}),
            serde_json::json!({"budget": 10, "selection": ["two"]}),
        ] {
            assert_ne!(a.digest(), Inputs::capture(&roots, &[], &semantic)?.digest());
        }
        Ok(())
    }
    #[test]
    fn explicit_exclusions_cannot_escape_or_follow_symlinks() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        symlink("missing", dir.path().join("alias"))?;
        for excluded in ["", ".", "..", "../outside", "/outside", "a/../b", "alias", "alias/child"]
        {
            let mut source = root(dir.path());
            source.excluded.push(PathBuf::from(excluded));
            assert!(Inputs::capture(&[source], &[], &()).is_err(), "accepted {excluded}");
        }
        fs::remove_file(dir.path().join("alias"))?;
        fs::create_dir(dir.path().join("outputs"))?;
        let mut source = root(dir.path());
        source.excluded.push("outputs".into());
        let inputs = Inputs::capture(&[source], &[], &())?;
        fs::write(dir.path().join("outputs/generated"), "output")?;
        inputs.verify_unchanged()?;
        Ok(())
    }
    #[test]
    fn root_aliases_and_duplicate_labels_are_refused() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let source = root(dir.path());
        assert!(Inputs::capture(&[source.clone(), source.clone()], &[], &()).is_err());
        let mut invalid = source.clone();
        invalid.label = "../bad".into();
        assert!(Inputs::capture(&[invalid], &[], &()).is_err());
        let mut invalid = source;
        invalid.root.push(".");
        assert!(Inputs::capture(&[invalid], &[], &()).is_err());
        Ok(())
    }
    #[test]
    fn external_tool_replacement_is_refused_even_with_identical_bytes() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let tool = dir.path().join("tool");
        fs::write(&tool, "image")?;
        let inputs = Inputs::capture(&[], &[tool.canonicalize()?], &())?;
        fs::rename(&tool, dir.path().join("old"))?;
        fs::write(&tool, "image")?;
        assert!(inputs.verify_unchanged().is_err());
        Ok(())
    }
    #[test]
    fn retained_image_copies_original_bytes_after_path_replacement() -> io::Result<()> {
        let dir = tempfile::tempdir()?;
        let path = dir.path().join("image");
        fs::write(&path, "running image")?;
        let image = PinnedImage::from_file(File::open(&path)?)?;
        let digest = image.digest().to_owned();
        fs::rename(&path, dir.path().join("old"))?;
        fs::write(&path, "replacement image")?;
        let destination = dir.path().join("private-image");
        image.copy_to(&destination)?;
        assert_eq!(fs::read(&destination)?, b"running image");
        assert_ne!(fs::metadata(&destination)?.permissions().mode() & 0o111, 0);
        assert_eq!(image.digest(), digest);
        assert!(image.copy_to(&destination).is_err());
        Ok(())
    }
}
