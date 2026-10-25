use std::{path::Path, sync::atomic::{AtomicBool, Ordering}};
type Refusal = std::io::Error;
static REMOVE_AFTER_METADATA: AtomicBool = AtomicBool::new(true);
mod fs {
    pub use std::fs::read_dir;
    pub fn symlink_metadata(path: &std::path::Path) -> std::io::Result<std::fs::Metadata> {
        let metadata = std::fs::symlink_metadata(path)?;
        if path.file_name().is_some_and(|name| name == "victim")
            && super::REMOVE_AFTER_METADATA.swap(false, super::Ordering::SeqCst) {
            println!("fixture scheduling hook: directory metadata succeeded; remove directory before read_dir");
            std::fs::remove_dir(path)?;
        }
        Ok(metadata)
    }
}
fn within_budget(path: &Path, limit: u64) -> Result<bool, Refusal> {
    fn size(
        path: &Path,
        remaining: &mut u64,
        entries: &mut usize,
        depth: usize,
    ) -> Result<bool, Refusal> {
        if depth > 64 || *entries == 0 {
            return Ok(false);
        }
        *entries -= 1;
        let meta = match fs::symlink_metadata(path) {
            Ok(meta) => meta,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(true),
            Err(error) => return Err(error.into()),
        };
        if meta.is_symlink() {
            return Ok(false);
        }
        if meta.is_dir() {
            for entry in fs::read_dir(path)? {
                if !size(&entry?.path(), remaining, entries, depth + 1)? {
                    return Ok(false);
                }
            }
        } else if meta.is_file() && meta.len() <= *remaining {
            *remaining -= meta.len();
        } else {
            return Ok(false);
        }
        Ok(true)
    }
    size(path, &mut { limit }, &mut 4096, 0)
}

fn main() -> std::io::Result<()> {
    let root = std::env::args_os().nth(1).map(std::path::PathBuf::from).unwrap();
    std::fs::create_dir(&root)?;
    std::fs::create_dir(root.join("victim"))?;
    let result = within_budget(&root, 134_217_728);
    println!("original within_budget body returned {result:?}");
    assert_eq!(result.as_ref().unwrap_err().kind(), std::io::ErrorKind::NotFound);
    let allowed = result.unwrap_or(false);
    println!("production caller unwrap_or(false) yields allowed={allowed}");
    assert!(!allowed);
    std::fs::remove_dir(&root)?;
    Ok(())
}
