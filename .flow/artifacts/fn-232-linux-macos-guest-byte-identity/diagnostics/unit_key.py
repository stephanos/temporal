import json
from pathlib import Path


def unit_key(argv, cwd):
    if "--target" not in argv or "--crate-name" not in argv:
        return None
    metadata = next((i + 1 for i, a in enumerate(argv[:-1])
                     if a == "-C" and argv[i + 1].startswith("metadata=")), None)
    if metadata is None:
        return None
    source = next(a for a in argv if a.endswith(".rs"))
    source = str((Path(cwd) / source).resolve())
    if "/lib/rustlib/src/rust/library/" in source:
        source = "/std/" + source.split("/lib/rustlib/src/rust/library/", 1)[1]
    elif "/registry/src/" in source:
        source = "/registry/" + source.split("/registry/src/", 1)[1].split("/", 1)[1]
    elif "/source-a/model/" in source:
        source = "/workspace/" + source.split("/source-a/model/", 1)[1]
    cfg = sorted(argv[i + 1] for i, a in enumerate(argv[:-1]) if a == "--cfg")
    profile = sorted(argv[i + 1] for i, a in enumerate(argv[:-1]) if a == "-C"
                     and not argv[i + 1].startswith(("metadata=", "extra-filename=")))
    return json.dumps([argv[argv.index("--crate-name") + 1], source,
                       argv[argv.index("--target") + 1], cfg, profile]), metadata
