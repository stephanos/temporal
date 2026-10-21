#!/usr/bin/env python3
import json
import os
import sys

selected = {key: value for key, value in os.environ.items()
            if key in {"OUT_DIR", "HOST", "TARGET", "RUSTC_BOOTSTRAP"}
            or key.startswith("CARGO_CFG_TARGET_")}
entry = {"argv": sys.argv[1:], "cwd": os.getcwd(), "env": selected}
fd = os.open(os.environ["FN232_RUSTC_LOG"], os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
os.write(fd, (json.dumps(entry) + "\n").encode())
os.close(fd)
os.execv(sys.argv[1], sys.argv[1:])
