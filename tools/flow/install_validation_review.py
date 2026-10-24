"""Install tested staged Flow files, preserving hash-checked recovery copies."""

import hashlib
import json
import os
from pathlib import Path
import shutil
import sys
import tempfile


TARGETS = ("flowctl.py", "make-pr-preflight.sh")


def replace_checked(target, staged, baseline, backup):
    if target.is_symlink() or not target.is_file():
        raise ValueError(f"Installed target must be a regular file: {target}")
    expected = hashlib.sha256(baseline.read_bytes()).digest()
    if hashlib.sha256(target.read_bytes()).digest() != expected:
        raise ValueError(f"Installation baseline changed: {target}; restage and retest current files")
    backup.parent.mkdir(parents=True, exist_ok=True)
    with backup.open("xb") as output:
        output.write(baseline.read_bytes())
        output.flush()
        os.fsync(output.fileno())
    shutil.copymode(target, backup)
    fd, temporary = tempfile.mkstemp(prefix=".flow-validation-install-", dir=target.parent)
    try:
        with os.fdopen(fd, "wb") as output:
            output.write(staged.read_bytes())
            output.flush()
            os.fsync(output.fileno())
        shutil.copymode(target, temporary)
        if target.is_symlink() or hashlib.sha256(target.read_bytes()).digest() != expected:
            raise ValueError(f"Installation baseline changed: {target}; no replacement performed")
        os.replace(temporary, target)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def install(stage, targets, backups):
    # Check the whole pair before the first replacement; each replacement rechecks.
    for name in TARGETS:
        if (targets / name).read_bytes() != (stage / (name + ".baseline")).read_bytes():
            raise ValueError(f"Installation baseline changed: {targets / name}; restage and retest")
    for name in TARGETS:
        replace_checked(targets / name, stage / "scripts" / name,
                        stage / (name + ".baseline"), backups / name)
    return {name: hashlib.sha256((targets / name).read_bytes()).hexdigest() for name in TARGETS}


if __name__ == "__main__":
    if len(sys.argv) != 4:
        sys.exit("Usage: install_validation_review.py STAGE_DIR INSTALLED_SCRIPTS BACKUP_DIR")
    print(json.dumps(install(*(Path(arg) for arg in sys.argv[1:])), indent=2))
