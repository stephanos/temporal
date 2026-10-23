#!/usr/bin/env python3
"""Admit the official Apalache assembly into the workspace build root."""
import fcntl
import hashlib
import io
import os
from pathlib import Path
import stat
import subprocess
import tarfile
import tempfile

VERSION = "0.62.4"
ARCHIVE_SHA256 = "b781b53bba32fac14ac2e02b0332e1dc2edc64cb7b850c0608d8142cc7aa2ef4"
JAR_SHA256 = "e5223915507b65cc5f1638bd037ede60f8a4fc1644aff46355779f9b05df9262"


def install(root):
    root = root.resolve(strict=True)
    cache = root / "model/build/apalache"
    current = root
    for part in ["model", "build", "apalache"]:
        current = current / part
        if current.is_symlink():
            raise RuntimeError("Apalache build root contains a symlink")
        current.mkdir(exist_ok=True)
        if not current.is_dir() or current.resolve() != current:
            raise RuntimeError("Apalache build root escapes workspace")
    lock = cache / ".install.lock"
    if lock.is_symlink():
        raise RuntimeError("Apalache lock is a symlink")
    fd = os.open(lock, os.O_CREAT | os.O_WRONLY | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    with os.fdopen(fd, "w") as stream:
        if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
            raise RuntimeError("Apalache lock is not a regular file")
        fcntl.flock(stream, fcntl.LOCK_EX)
        destination = cache / f"apalache-{VERSION}"
        jar = destination / "lib/apalache.jar"
        if any(p.is_symlink() for p in [destination, destination / "lib", jar]):
            raise RuntimeError("Apalache installation is a symlink")
        if jar.is_file():
            if not jar.resolve().is_relative_to(cache.resolve()):
                raise RuntimeError("Apalache jar escapes cache")
            if hashlib.sha256(jar.read_bytes()).hexdigest() != JAR_SHA256:
                raise RuntimeError("Apalache jar integrity mismatch")
            return jar
        if destination.exists():
            raise RuntimeError("partial Apalache installation")
        with tempfile.TemporaryDirectory(prefix="install-", dir=cache) as scratch:
            staging = Path(scratch)
            archive = staging / "release.tgz"
            subprocess.run(["curl", "--proto", "=https", "--tlsv1.2", "--fail", "--location",
                            "--retry", "2", "--max-time", "180",
                            f"https://github.com/apalache-mc/apalache/releases/download/v{VERSION}/apalache-{VERSION}.tgz",
                            "-o", str(archive)], check=True)
            payload = archive.read_bytes()
            if hashlib.sha256(payload).hexdigest() != ARCHIVE_SHA256:
                raise RuntimeError("Apalache archive integrity mismatch")
            with tarfile.open(fileobj=io.BytesIO(payload), mode="r:gz") as source:
                member = source.getmember(f"apalache-{VERSION}/lib/apalache.jar")
                if not member.isfile():
                    raise RuntimeError("Apalache jar is not a regular archive member")
                contents = source.extractfile(member).read()
            if hashlib.sha256(contents).hexdigest() != JAR_SHA256:
                raise RuntimeError("Apalache extracted jar integrity mismatch")
            private = staging / "validated"
            (private / "lib").mkdir(parents=True)
            (private / "lib/apalache.jar").write_bytes(contents)
            if cache.resolve() != cache or destination.exists() or destination.is_symlink():
                raise RuntimeError("Apalache cache changed during admission")
            os.rename(private, destination)
        return jar


if __name__ == "__main__":
    print(install(Path(__file__).resolve().parents[2]))
