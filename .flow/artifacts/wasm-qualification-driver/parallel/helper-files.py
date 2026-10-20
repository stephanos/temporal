#!/usr/bin/env python3
"""Admitted helpers must be independent executable single-link regular files."""
from pathlib import Path
import os
import stat
import sys

identities = []
for value in sys.argv[1:]:
    path = Path(value)
    info = path.lstat()
    assert stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and os.access(path, os.X_OK), value
    identities.append((info.st_dev, info.st_ino))
assert len(identities) == len(set(identities)), "helpers alias one file"
