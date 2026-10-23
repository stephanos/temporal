"""Pinned Apalache installation is private, contained and atomically published."""
import hashlib
import importlib.util
import io
import multiprocessing
import os
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / "fetch-apalache.py"
spec = importlib.util.spec_from_file_location("fetch_apalache", SCRIPT)
apalache = importlib.util.module_from_spec(spec)
spec.loader.exec_module(apalache)
JAR = b"local installer test jar"


def archive(contents=JAR):
    output = io.BytesIO()
    with tarfile.open(fileobj=output, mode="w:gz") as target:
        member = tarfile.TarInfo(f"apalache-{apalache.VERSION}/lib/apalache.jar")
        member.size = len(contents)
        target.addfile(member, io.BytesIO(contents))
    return output.getvalue()


def install_fake(root, payload, calls):
    def fetch(args, **_):
        with calls.open("ab") as log:
            log.write(b"download\n")
        Path(args[-1]).write_bytes(payload)
    with patch.object(apalache, "ARCHIVE_SHA256", hashlib.sha256(payload).hexdigest()), \
         patch.object(apalache, "JAR_SHA256", hashlib.sha256(JAR).hexdigest()), \
         patch.object(apalache.subprocess, "run", fetch):
        return apalache.install(root)


class Installer(unittest.TestCase):
    def test_symlink_in_each_install_component_is_refused(self):
        for component in ["model", "model/build", "model/build/apalache", 
                          f"model/build/apalache/apalache-{apalache.VERSION}/lib"]:
            with self.subTest(component=component), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                outside = root / "other"
                outside.mkdir()
                (outside / "apalache.jar").write_bytes(JAR)
                link = root / component
                link.parent.mkdir(parents=True, exist_ok=True)
                link.symlink_to(outside, target_is_directory=True)
                with patch.object(apalache, "JAR_SHA256", hashlib.sha256(JAR).hexdigest()), \
                     patch.object(apalache.subprocess, "run", side_effect=AssertionError("download")):
                    with self.assertRaisesRegex(RuntimeError, "symlink|escapes"):
                        apalache.install(root)

    def test_partial_and_wrong_jar_never_download_or_publish(self):
        for contents in [None, b"wrong"]:
            with self.subTest(contents=contents), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                dest = root / f"model/build/apalache/apalache-{apalache.VERSION}"
                (dest / "lib").mkdir(parents=True)
                if contents is not None:
                    (dest / "lib/apalache.jar").write_bytes(contents)
                with patch.object(apalache.subprocess, "run", side_effect=AssertionError("download")):
                    with self.assertRaisesRegex(RuntimeError, "partial|integrity"):
                        apalache.install(root)

    def test_archive_and_extracted_jar_mismatches_leave_no_partial_install(self):
        for payload, digest in [(archive(), "bad"), (archive(b"wrong jar"), None)]:
            with self.subTest(digest=digest), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                def fetch(args, **_):
                    Path(args[-1]).write_bytes(payload)
                with patch.object(apalache, "ARCHIVE_SHA256", digest or hashlib.sha256(payload).hexdigest()), \
                     patch.object(apalache.subprocess, "run", fetch):
                    with self.assertRaisesRegex(RuntimeError, "integrity"):
                        apalache.install(root)
                cache = root / "model/build/apalache"
                self.assertEqual([p.name for p in cache.iterdir()], [".install.lock"])

    def test_dangling_components_and_unsafe_locks_refuse_without_download(self):
        for component in ["model", "model/build", "model/build/apalache"]:
            with self.subTest(component=component), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                link = root / component
                link.parent.mkdir(parents=True, exist_ok=True)
                missing = root / "missing"
                link.symlink_to(missing, target_is_directory=True)
                with patch.object(apalache.subprocess, "run", side_effect=AssertionError("download")):
                    with self.assertRaisesRegex(RuntimeError, "symlink"):
                        apalache.install(root)
                self.assertFalse(missing.exists())
        for kind in ["symlink", "fifo"]:
            with self.subTest(kind=kind), tempfile.TemporaryDirectory() as tmp:
                root = Path(tmp)
                cache = root / "model/build/apalache"
                cache.mkdir(parents=True)
                lock = cache / ".install.lock"
                if kind == "symlink":
                    lock.symlink_to(root / "missing")
                else:
                    os.mkfifo(lock)
                with self.assertRaises((RuntimeError, OSError)):
                    apalache.install(root)

    def test_failed_download_cleans_private_staging(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            def fail(args, **_):
                staging = Path(args[-1]).parent
                self.assertEqual(staging.stat().st_mode & 0o777, 0o700)
                self.assertFalse((staging.parent / f"apalache-{apalache.VERSION}").exists())
                Path(args[-1]).write_bytes(b"partial")
                raise OSError("download failed")
            with patch.object(apalache.subprocess, "run", fail):
                with self.assertRaisesRegex(OSError, "download failed"):
                    apalache.install(root)
            self.assertEqual([p.name for p in (root / "model/build/apalache").iterdir()], [".install.lock"])

    def test_concurrent_installers_publish_once_and_converge(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            calls = root / "calls"
            payload = archive()
            context = multiprocessing.get_context("fork")
            workers = [context.Process(target=install_fake, args=(root, payload, calls)) for _ in range(4)]
            for worker in workers:
                worker.start()
            for worker in workers:
                worker.join(10)
                alive = worker.is_alive()
                if alive:
                    worker.terminate()
                    worker.join()
                self.assertFalse(alive)
                self.assertEqual(worker.exitcode, 0)
            self.assertEqual(calls.read_bytes(), b"download\n")
            jar = root / f"model/build/apalache/apalache-{apalache.VERSION}/lib/apalache.jar"
            self.assertEqual(jar.read_bytes(), JAR)
            self.assertEqual(sorted(p.name for p in jar.parents[2].iterdir()), 
                             [".install.lock", f"apalache-{apalache.VERSION}"])


if __name__ == "__main__":
    unittest.main()
