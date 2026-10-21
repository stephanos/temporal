#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys
from unit_key import unit_key

argv = sys.argv[1:]
original = argv.copy()
result = unit_key(argv, os.getcwd())
if result:
    key, index = result
    metadata = json.loads(Path(os.environ["FN232_METADATA_MAP"]).read_text())
    argv[index] = metadata[key]
entry = {"original": original, "argv": argv, "cwd": os.getcwd()}
fd = os.open(os.environ["FN232_RUSTC_LOG"], os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
os.write(fd, (json.dumps(entry) + "\n").encode())
os.close(fd)
os.execv(argv[0], argv)
